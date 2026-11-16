const L = require('./ui_lib.cjs'); const { chromium, login, B, sql } = L; const fs = require('fs'); const E30 = JSON.parse(fs.readFileSync('/tmp/qc2L/e30.json')).eid; const C1 = JSON.parse(fs.readFileSync('/tmp/qc2L/hold.json')).c1; const EM = JSON.parse(fs.readFileSync('/tmp/qc2L/empty.json')).empty;
(async () => { const br = await chromium.launch({ headless: true });
  const { ctx, p } = await login(br, 'r5t', { width: 1440, height: 900 }); await p.goto(`${B}/exams/${E30}/results?course=${C1}`); await p.waitForTimeout(4000); await L.dismiss(p);
  console.log('TABS', JSON.stringify(await p.getByRole('tab').allInnerTexts())); console.log('PRIMARY', JSON.stringify(await p.locator('[data-variant=primary]:visible').allInnerTexts()));
  console.log('BTNS', JSON.stringify(await p.locator('main button:visible').allInnerTexts()).slice(0, 500));
  await p.locator('tbody tr').first().click(); await p.waitForTimeout(2500); const d = p.locator('dialog[open]'); console.log('DRAWER', (await d.allInnerTexts()).join('|').replace(/\n/g, ' ¦ ').slice(0, 2500));
  await p.screenshot({ path: L.SH + '/m-res-drawer.png' }); await p.keyboard.press('Escape'); await p.waitForTimeout(500);
  for (const t of ['Thống kê', 'Nghi giống nhau', 'Xem lại điểm']) { await p.getByRole('tab', { name: t }).click().catch(() => {}); await p.waitForTimeout(1500); console.log('TAB', t, (await p.locator('main').innerText()).replace(/\n/g, ' ¦ ').slice(0, 500)); }
  await p.goto(`${B}/exams/${EM}/results?course=${C1}`); await p.waitForTimeout(3000); console.log('EMPTY', (await p.locator('main').innerText()).replace(/\n/g, ' ¦ ').slice(0, 400));
  await ctx.close(); await br.close(); })();
