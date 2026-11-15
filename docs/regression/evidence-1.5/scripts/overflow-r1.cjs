// TC-00-36: TA tại «/» không tràn ngang ở 390/375
const R = '/Users/kuro/Documents/TA_Agent_qc2/frontend/node_modules/';
const { chromium } = require(R + '@playwright/test');
const B = 'https://localhost:1143', PW = 'Edupilot#Seed-2026';
(async () => {
  const br = await chromium.launch({ headless: true }); let bad = 0;
  for (const u of ['ta', 'teacher', 'sv.kha', 'admin']) for (const w of [375, 390]) {
    const ctx = await br.newContext({ ignoreHTTPSErrors: true, viewport: { width: w, height: 812 }, locale: 'vi-VN', hasTouch: true }); const p = await ctx.newPage();
    await p.goto(B + '/login'); await p.getByLabel(/email/i).fill(u + '@edupilot.local'); await p.getByLabel(/mật khẩu/i).first().fill(PW);
    await p.getByRole('button', { name: /đăng nhập/i }).click(); await p.waitForTimeout(3500);
    const ox = await p.evaluate(() => document.documentElement.scrollWidth - window.innerWidth);
    console.log(`${ox <= 0 ? 'PASS' : 'FAIL'} ${u} ${w}px ox=${ox}`); if (ox > 0) bad++; await ctx.close();
  }
  await br.close(); console.log(bad ? 'RESULT: FAIL' : 'RESULT: PASS');
})();
