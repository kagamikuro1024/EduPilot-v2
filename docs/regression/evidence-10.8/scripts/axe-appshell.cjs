// Chạy lại DEF-R5-01 / DEF-R10.8-01 (axe ở khung ứng dụng) trên bản dựng ≥ ec1657c2.  node axe-appshell.cjs
const R = '/Users/kuro/Documents/TA_Agent_qc2/frontend/node_modules/';
const { chromium } = require(R + '@playwright/test');
const AxeBuilder = require(R + '@axe-core/playwright').default || require(R + '@axe-core/playwright');
const B = 'https://localhost:1143', PW = 'Edupilot#Seed-2026';
(async () => {
  const br = await chromium.launch({ headless: true });
  let bad = 0;
  for (const u of ['admin', 'teacher', 'sv.kha']) for (const vp of [{ width: 1440, height: 900 }, { width: 375, height: 812 }]) {
    const ctx = await br.newContext({ ignoreHTTPSErrors: true, viewport: vp, locale: 'vi-VN' }); const p = await ctx.newPage();
    await p.goto(B + '/login'); await p.getByLabel(/email/i).fill(u + '@edupilot.local'); await p.getByLabel(/mật khẩu/i).first().fill(PW);
    await p.getByRole('button', { name: /đăng nhập/i }).click(); await p.waitForTimeout(3500);
    for (const tags of [['wcag2a', 'wcag2aa', 'wcag21aa', 'wcag22aa'], ['wcag2a', 'wcag2aa', 'wcag21a', 'wcag21aa', 'wcag22aa']]) {
      const r = await new AxeBuilder({ page: p }).withTags(tags).analyze();
      const v = r.violations.filter((x) => ['serious', 'critical'].includes(x.impact)).map((x) => x.id + '×' + x.nodes.length);
      const lcn = r.violations.filter((x) => x.id === 'label-content-name-mismatch').length;
      console.log(`${lcn ? 'FAIL' : 'PASS'} ${u} ${vp.width} tags=${tags.includes('wcag21a') ? '+21a' : 'TC'} serious/critical=${JSON.stringify(v)} label-content-name-mismatch=${lcn}`); if (lcn) bad++;
    }
    await ctx.close();
  }
  await br.close(); console.log(bad ? 'RESULT: FAIL' : 'RESULT: PASS');
})();
