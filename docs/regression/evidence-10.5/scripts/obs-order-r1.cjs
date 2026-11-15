// TC-D07-02: /observability (Admin) — bảng «Quyết định nhanh» nằm DƯỚI bảng «7 ngày qua»
const R = '/Users/kuro/Documents/TA_Agent_qc2/frontend/node_modules/';
const { chromium } = require(R + '@playwright/test');
const B = 'https://localhost:1143', PW = 'Edupilot#Seed-2026';
(async () => {
  const br = await chromium.launch({ headless: true });
  const ctx = await br.newContext({ ignoreHTTPSErrors: true, viewport: { width: 1440, height: 900 }, locale: 'vi-VN' }); const p = await ctx.newPage();
  await p.goto(B + '/login'); await p.getByLabel(/email/i).fill('admin@edupilot.local'); await p.getByLabel(/mật khẩu/i).first().fill(PW);
  await p.getByRole('button', { name: /đăng nhập/i }).click(); await p.waitForTimeout(3000);
  await p.goto(B + '/observability'); await p.waitForTimeout(4000);
  const y = async (re) => { const l = p.getByText(re).first(); return (await l.count()) ? (await l.boundingBox())?.y : null; };
  const y7 = await y(/7 ngày qua/i), yq = await y(/Quyết định nhanh/i);
  const ok = y7 != null && yq != null && yq > y7;
  console.log(`${ok ? 'PASS' : 'FAIL'} y(7 ngày qua)=${y7} y(Quyết định nhanh)=${yq}`);
  await p.screenshot({ path: '/Users/kuro/Documents/TA_Agent_qc2/docs/regression/evidence-10.5/shots/observability-order-r1.png', fullPage: true });
  await br.close(); console.log(ok ? 'RESULT: PASS' : 'RESULT: FAIL');
})();
