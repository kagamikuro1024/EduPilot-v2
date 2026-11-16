const L = require('./ui_lib.cjs'); const { chromium, login, B } = L; const fx = JSON.parse(require('fs').readFileSync('/tmp/qc2L/uix2.json'));
(async () => { const br = await chromium.launch({ headless: true });
  const { ctx, p } = await login(br, 'r5s17', { width: 1440, height: 900 });
  await p.goto(`${B}/exams/${fx.xcode}/take?course=${fx.c1}`); await p.waitForTimeout(3000); await L.dismiss(p);
  console.log('URL', p.url()); console.log((await p.locator('main').innerText()).slice(0, 700));
  console.log('BTN', JSON.stringify(await p.locator('button:visible').allInnerTexts()));
  const n = p.locator('[data-part=integrity-notice]'); console.log('notice', await n.count());
  const b = p.getByRole('button', { name: /Bắt đầu làm bài/ }); console.log('start', await b.count(), await b.first().isEnabled().catch(() => null));
  if (await b.count()) { if (await n.count()) await n.scrollIntoViewIfNeeded(); await b.first().click(); await p.waitForTimeout(3000); }
  console.log('AFTER', (await p.locator('main').innerText()).slice(0, 500).replace(/\n/g, ' | '));
  console.log('BTN2', JSON.stringify(await p.locator('button:visible').allInnerTexts()));
  for (let i = 0; i < 4; i++) { if (await p.locator('textarea').count()) break; await p.keyboard.press('ArrowRight'); await p.waitForTimeout(500); }
  console.log('textarea', await p.locator('textarea').count());
  console.log('BTN3', JSON.stringify(await p.locator('button:visible').evaluateAll((els) => els.map((e) => (e.getAttribute('aria-label') || '') + '|' + e.innerText.trim() + '|' + e.disabled))));
  await p.screenshot({ path: L.SH + '/m-code-take.png' }); await br.close(); })();
