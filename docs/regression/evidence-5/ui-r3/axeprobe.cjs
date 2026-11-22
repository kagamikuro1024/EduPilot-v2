const L = require('./ui_lib.cjs'); const { chromium, B, AxeBuilder } = L; const fs = require('fs');
const eid = JSON.parse(fs.readFileSync('/tmp/qc3e/f19b.json')).eid;
(async () => { const br = await chromium.launch(); const { ctx, p } = await L.login(br, 'r5s19', { width: 375, height: 800 });
  await p.goto(B + `/exams/${eid}/take`, { waitUntil: 'load' }); await p.waitForTimeout(4000);
  const sums = p.locator('summary'); const n = await sums.count(); for (let i = 0; i < n; i++) await sums.nth(i).click().catch(() => {}); await p.waitForTimeout(800);
  const r = await new AxeBuilder({ page: p }).analyze(); for (const v of r.violations.filter((x) => x.id === 'scrollable-region-focusable')) for (const nd of v.nodes) console.log(v.impact, nd.target.join(' '), nd.html.slice(0, 220));
  await br.close(); })();
