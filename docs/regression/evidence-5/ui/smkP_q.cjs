// smkP: chạy lại TC-PE03-49 và TC-PE03-51 sau R5-03/R5-04 (ep-r HEAD 634d59c5). node smkP_q.cjs
const L = require('./ui_lib.cjs'); const { chromium, login, B, pickClass, R, AUDIT_SRC, TOUCH_SRC, SH } = L;
(async () => { process.on('unhandledRejection', (e) => console.log('ERR', String(e).slice(0, 300)));
  const browser = await chromium.launch({ headless: true });
  const { ctx, p } = await login(browser, 'r5t', { width: 1440, height: 900 });
  await p.goto(B + '/questions'); await p.waitForTimeout(2000); await pickClass(p, 'R5C1');
  const search = async (pg, q) => { const i = pg.locator('input').first(); await i.fill(q); await pg.waitForTimeout(1800); await pg.locator('tbody tr').first().click(); await pg.waitForTimeout(1800); };
  const marker = 'NHAPDOQC2' + Date.now();
  await search(p, 'Đặc điểm chế độ mã hóa ECB');
  const edit = p.locator('dialog[open]').getByRole('button', { name: /Chỉnh sửa/ });
  const hasEdit = await edit.count(); let before = '', draftKept = null, banner = false;
  if (hasEdit) {
    await edit.first().click(); await p.waitForTimeout(800);
    const ta = p.locator('dialog[open] textarea').first(); before = await ta.inputValue(); await ta.fill(before + ' ' + marker); await p.waitForTimeout(3000);
    await p.close();
    const p2 = await ctx.newPage(); await p2.goto(B + '/questions'); await p2.waitForTimeout(2500); await pickClass(p2, 'R5C1');
    await search(p2, 'Đặc điểm chế độ mã hóa ECB');
    const e2 = p2.locator('dialog[open]').getByRole('button', { name: /Chỉnh sửa/ }); if (await e2.count()) { await e2.first().click(); await p2.waitForTimeout(1500); }
    const body2 = (await p2.locator('dialog[open]').allInnerTexts()).join(' ') + ' ' + (await p2.locator('dialog[open] textarea,dialog[open] input').evaluateAll((els) => els.map((e) => e.value).join(' ')).catch(() => ''));
    draftKept = body2.includes(marker); banner = /khôi phục/i.test(body2); await p2.screenshot({ path: SH + '/smkP-pe03-49-draft.png' });
    R('TC-PE03-49', draftKept && banner ? 'P' : 'F', `smkP @634d59c5: GV r5t, Chỉnh sửa câu AI chờ duyệt, gõ dở "${marker}", đóng tab, mở tab mới, vào lại Chỉnh sửa: chữ còn=${draftKept}, banner "khôi phục"=${banner}; các phần Drawer (nút Chỉnh sửa/Duyệt, khoá "nhân bản để sửa") đạt ở lượt cũ, không đổi`);
    await p2.close();
  } else R('TC-PE03-49', 'F', 'smkP: không có nút Chỉnh sửa ở Drawer câu AI chờ duyệt');
  // 51
  const p3 = await ctx.newPage(); await p3.goto(B + '/questions'); await p3.waitForTimeout(2000); await pickClass(p3, 'R5C1');
  let found = null; for (let i = 0; i < 80; i++) { await p3.keyboard.press('Tab'); const f = await p3.evaluate(() => { const e = document.activeElement; return e ? { inRow: !!e.closest('tbody tr') } : null; }); if (f && f.inRow) { found = { i }; break; } }
  let kb = { rowFocus: !!found, dialog: false, focusInside: false, duyet: false, escBack: false };
  if (found) { await p3.keyboard.press('Enter'); await p3.waitForTimeout(1500); kb.dialog = (await p3.locator('dialog[open]').count()) > 0; kb.focusInside = await p3.evaluate(() => !!document.activeElement?.closest('dialog[open]')); for (let i = 0; i < 40 && !kb.duyet; i++) { await p3.keyboard.press('Tab'); kb.duyet = await p3.evaluate(() => /Duyệt|Chỉnh sửa/.test(document.activeElement?.innerText || '')); } await p3.keyboard.press('Escape'); await p3.waitForTimeout(800); kb.escBack = await p3.evaluate(() => !document.activeElement?.closest('dialog') && document.activeElement !== document.body); }
  const wid = {}; for (const w of [1440, 1024, 720, 719, 390]) { await p3.setViewportSize({ width: w, height: 900 }); await p3.goto(B + '/questions'); await p3.waitForTimeout(1800); await pickClass(p3, 'R5C1'); const a = await p3.evaluate(`(${AUDIT_SRC.trim()})`).then(async (r0) => { const opts = await p3.evaluate(() => [...document.querySelectorAll('option')].map((o) => o.textContent.trim())); r0.cut = r0.cut.filter((c) => !opts.includes(c)); return r0; }); const tch = w === 390 ? await p3.evaluate(TOUCH_SRC) : null; const tbl = await p3.locator('table:visible').count(); wid[w] = { ox: a.ox, cut: a.cut.length, ell: a.ell.length, tbl, touch: tch ? tch.length : '-' }; }
  const okw = Object.values(wid).every((x) => x.ox <= 0 && x.cut === 0 && x.ell === 0) && wid[720].tbl > 0 && wid[719].tbl === 0;
  R('TC-PE03-51', kb.rowFocus && kb.dialog && kb.focusInside && kb.escBack && okw ? 'P' : 'F', `smkP @634d59c5: Tab tới hàng (sau ${found && found.i} lần), Enter mở Drawer=${kb.dialog}, focus trong Drawer=${kb.focusInside}, tới nút Duyệt/Chỉnh sửa=${kb.duyet}, Esc trả focus=${kb.escBack}; AUDIT ox/cut/ell/bảng/touch ở 1440,1024,720,719,390: ${JSON.stringify(wid)}`);
  await browser.close();
})();
