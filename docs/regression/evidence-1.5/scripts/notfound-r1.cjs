// TC-00-57/59: đã đăng nhập mở đường dẫn không có thật → màn «Không tìm thấy trang» + «Về Hôm nay» (không về /login)
const R = '/Users/kuro/Documents/TA_Agent_qc2/frontend/node_modules/';
const { chromium } = require(R + '@playwright/test');
const B = 'https://localhost:1143', PW = 'Edupilot#Seed-2026';
(async () => {
  const br = await chromium.launch({ headless: true }); let bad = 0;
  for (const u of ['teacher', 'sv.kha', 'admin']) {
    const ctx = await br.newContext({ ignoreHTTPSErrors: true, viewport: { width: 1440, height: 900 }, locale: 'vi-VN' }); const p = await ctx.newPage();
    await p.goto(B + '/login'); await p.getByLabel(/email/i).fill(u + '@edupilot.local'); await p.getByLabel(/mật khẩu/i).first().fill(PW);
    await p.getByRole('button', { name: /đăng nhập/i }).click(); await p.waitForTimeout(3000);
    await p.goto(B + '/duong-dan-khong-co-that'); await p.waitForTimeout(3500);
    const url = p.url(), txt = await p.locator('body').innerText(); const back = p.getByRole('link', { name: /Về Hôm nay/ });
    const ok = !/\/login/.test(url) && /không tìm thấy/i.test(txt) && (await back.count()) > 0;
    console.log(`${ok ? 'PASS' : 'FAIL'} ${u}: url=${url.replace(B, '')} 404=${/không tìm thấy/i.test(txt)} nút «Về Hôm nay»=${await back.count()}`); if (!ok) bad++;
    if (ok) { await back.first().click(); await p.waitForTimeout(2000); console.log('   → sau bấm:', p.url().replace(B, '')); }
    await ctx.close();
  }
  await br.close(); console.log(bad ? 'RESULT: FAIL' : 'RESULT: PASS');
})();
