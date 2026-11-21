const L = require('./ui_lib.cjs'); const { chromium, login, B } = L; const fs = require('fs');
const FX = JSON.parse(fs.readFileSync('/tmp/qc2L/uix2.json'));
(async () => { const br = await chromium.launch({ headless: true }); const { ctx, p } = await login(br, 'r5s31', { width: 1440, height: 900 });
  await p.goto(`${B}/exams/${FX.xcode}/take?course=${FX.c1}`); await p.waitForTimeout(4000);
  console.log('URL', p.url()); console.log((await p.locator('main').innerText()).replace(/\s+/g, ' ').slice(0, 700));
  console.log('dialogs', await p.locator('dialog[open]').count(), (await p.locator('dialog[open]').allInnerTexts()).join('|').slice(0, 300));
  console.log('btns', (await p.getByRole('button').allInnerTexts()).slice(0, 25).join(' | '));
  await p.screenshot({ path: __dirname + '/shots/s4x-dbg-take.png' }); await ctx.close(); await br.close(); })();
