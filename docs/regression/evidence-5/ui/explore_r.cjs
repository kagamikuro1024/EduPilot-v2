const L = require('./ui_lib.cjs'); const { chromium, login, B, sql } = L; const fs = require('fs'); const E30 = JSON.parse(fs.readFileSync('/tmp/qc2L/e30.json')); const C1 = JSON.parse(fs.readFileSync('/tmp/qc2L/uix2.json')).c1;
(async () => { const br = await chromium.launch({ headless: true });
  for (const [u, w, path] of [['r5s01', 1440, `/exams/${E30.eid}/take?course=${C1}`], ['r5t', 1440, `/exams/${E30.eid}/results?course=${C1}`]]) {
    const { ctx, p } = await login(br, u, { width: w, height: 900 }); await p.goto(B + path); await p.waitForTimeout(4000); await L.dismiss(p);
    console.log('=====', u, w, p.url()); console.log((await p.locator('main').innerText()).slice(0, 2500)); console.log('BTN', JSON.stringify(await p.locator('main button:visible').allInnerTexts()).slice(0, 600)); await p.screenshot({ path: L.SH + `/m-res-${u}-${w}.png` }); await ctx.close(); }
  await br.close(); })();
