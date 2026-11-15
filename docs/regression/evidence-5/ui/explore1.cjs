const L = require('./ui_lib.cjs'); const { chromium, login, B, pickClass } = L;
(async () => { const browser = await chromium.launch({ headless: true });
  const { ctx, p } = await login(browser, 'r5t', { width: 1440, height: 900 });
  await p.goto(B + '/questions'); await p.waitForTimeout(2000); await pickClass(p, 'R5C1');
  await p.getByText('Đặc điểm chế độ mã hóa ECB', { exact: true }).first().click(); await p.waitForTimeout(1500);
  await p.locator('dialog[open]').getByRole('button', { name: /Chỉnh sửa/ }).first().click(); await p.waitForTimeout(800);
  const ta = p.locator('dialog[open] textarea').first(); await ta.click(); await p.keyboard.type(' QCMARK2'); await p.waitForTimeout(3500);
  await p.close(); const p2 = await ctx.newPage(); await p2.goto(B + '/questions'); await p2.waitForTimeout(2500); await pickClass(p2, 'R5C1');
  await p2.getByText('Đặc điểm chế độ mã hóa ECB', { exact: true }).first().click(); await p2.waitForTimeout(1800);
  console.log('LS2', await p2.evaluate(() => Object.keys(localStorage).filter(k=>k.startsWith('ep:draft')).map(k => k + '=' + localStorage[k].slice(-150))));
  console.log('DRAWER-1:', (await p2.locator('dialog[open]').innerText()).slice(0, 600));
  const e2 = p2.locator('dialog[open]').getByRole('button', { name: /Chỉnh sửa/ }); if (await e2.count()) { await e2.first().click(); await p2.waitForTimeout(1200); }
  console.log('DRAWER-2:', (await p2.locator('dialog[open]').innerText()).slice(0, 400));
  console.log('VALUES', await p2.locator('dialog[open] textarea').evaluateAll(els => els.map(e => e.value.slice(-40))));
  await ctx.close(); await browser.close(); })();
