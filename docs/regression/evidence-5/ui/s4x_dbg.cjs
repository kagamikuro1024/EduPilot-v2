const L = require('./ui_lib.cjs'); const { chromium, login, B } = L; const fs = require('fs');
const FX = JSON.parse(fs.readFileSync('/tmp/qc2L/uix2.json'));
(async () => { const br = await chromium.launch({ headless: true }); const { ctx, p } = await login(br, 'r5s31', { width: 1440, height: 900 });
  await p.goto(`${B}/exams/${FX.xcode}/take?course=${FX.c1}`); await p.waitForTimeout(2500); await L.dismiss(p);
  const n = p.locator('[data-part=integrity-notice]'); console.log('notice', await n.count()); if (await n.count()) await n.scrollIntoViewIfNeeded();
  const b = p.getByRole('button', { name: /Bắt đầu làm bài/ }); console.log('startbtn', await b.count()); if (await b.count()) { await b.first().click(); await p.waitForTimeout(3500); }
  console.log('URL', p.url()); console.log('btns', (await p.getByRole('button').allInnerTexts()).map((s) => s.replace(/\s+/g, ' ')).slice(0, 30).join(' | '));
  console.log('ta', await p.locator('textarea').count(), 'cm', await p.locator('.cm-content, .monaco-editor, [contenteditable=true]').count());
  console.log((await p.locator('main').innerText()).replace(/\s+/g, ' ').slice(0, 500));
  await p.screenshot({ path: __dirname + '/shots/s4x-dbg-take2.png' }); await ctx.close(); await br.close(); })();
