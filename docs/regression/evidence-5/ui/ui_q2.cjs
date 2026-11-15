// QC2 smoke sprint 5 (SmkL) — giao diện thật trên ep-r. Chạy: ONLY=<stage> node ui_q2.cjs  (stage: gv | gv2 | nav | axe)
const L = require('./ui_lib.cjs'); const { chromium, login, B, pickClass, R, metrics, axe, AUDIT_SRC, TOUCH_SRC, SH, sql } = L;
const ONLY = process.env.ONLY || 'gv';
const txt = (p) => p.locator('main').innerText();
(async () => { process.on('unhandledRejection', (e) => console.log('ERR', String(e).slice(0, 300)));
  const browser = await chromium.launch({ headless: true });
  if (ONLY === 'gv') {
    const { ctx, p } = await login(browser, 'r5t', { width: 1440, height: 900 });
    await p.goto(B + '/questions'); await p.waitForTimeout(2000); await pickClass(p, 'R5C1');
    // ---- TC-PE03-48
    const m = await metrics(p); const t = await txt(p);
    const cols = ['Câu hỏi', 'Chủ đề', 'Loại', 'Độ khó', 'Trạng thái', 'Nguồn', 'Dùng trong'].filter((c) => t.includes(c));
    const filt = ['Tìm câu hỏi', 'Trạng thái', 'Chủ đề', 'Độ khó', 'Loại', 'Nguồn'].filter((c) => t.includes(c));
    const statusCells = await p.locator('tbody tr').first().locator('td').nth(4).evaluate((e) => ({ text: e.innerText.trim(), icon: !!e.querySelector('svg,[data-icon]') })).catch(() => null);
    const labels = { ai: /\bAI\b/.test(t), manual: /Soạn tay/.test(t) };
    await p.getByRole('button', { name: /Tạo câu hỏi/ }).click(); await p.waitForTimeout(700); const menu = (await p.locator('[role=menu]:visible').allInnerTexts()).join('|').split('\n').filter(Boolean); await p.keyboard.press('Escape');
    R('TC-PE03-48', m.ox <= 0 && m.primary.length === 1 && /Tạo câu hỏi/.test(m.primary[0]) && cols.length === 7 && filt.length === 6 && labels.ai && labels.manual && !!statusCells && statusCells.text.length > 0 ? 'P' : 'F',
      `GV /questions R5C1 1440: cột ${cols.length}/7; bộ lọc ${filt.length}/6; primary=${JSON.stringify(m.primary)} (1); ô Trạng thái = "${statusCells && statusCells.text}" (chữ; icon=${statusCells && statusCells.icon}); nhãn Nguồn AI=${labels.ai} Soạn tay=${labels.manual}; menu "Tạo câu hỏi" = ${menu.length} mục [${menu.join(', ')}] — nay 4 mục (thêm "Trích từ đề", "Sinh từ tài liệu" của US-P10-*; con số 3 của TC gốc lỗi thời); điều khiển AI chỉ nằm trong menu; ox=${m.ox}`);
    // ---- TC-PE03-49 Drawer
    const open = async (name) => { await p.getByText(name, { exact: true }).first().click(); await p.waitForTimeout(1800); };
    const dlg = async () => (await p.locator('dialog[open]').allInnerTexts()).join(' ');
    const btns = async () => p.locator('dialog[open] button:visible').evaluateAll((els) => els.map((e) => (e.getAttribute('aria-label') || e.innerText).trim()));
    await open('r5l-noref'); const dr1 = await dlg(); const btn1 = await btns(); await p.keyboard.press('Escape'); await p.waitForTimeout(500);
    await open('r5code-91'); const dr2 = await dlg(); const btn2 = await btns(); await p.screenshot({ path: SH + '/pe03-49-code-drawer.png' }); await p.keyboard.press('Escape'); await p.waitForTimeout(500);
    await open('Đặc điểm chế độ mã hóa ECB'); const dr3 = await dlg(); const btn3 = await btns();
    const marker = 'NHAPDOQC2' + Date.now(); let draftKept = null;
    const edit = p.locator('dialog[open]').getByRole('button', { name: /Chỉnh sửa/ });
    if (await edit.count()) {
      await edit.first().click(); await p.waitForTimeout(800);
      const ta = p.locator('dialog[open] textarea').first(); await ta.fill((await ta.inputValue()) + ' ' + marker); await p.waitForTimeout(2500);
      await p.close();
      const p2 = await ctx.newPage(); await p2.goto(B + '/questions'); await p2.waitForTimeout(2500); await pickClass(p2, 'R5C1');
      await p2.getByText('Đặc điểm chế độ mã hóa ECB', { exact: true }).first().click().catch(() => {}); await p2.waitForTimeout(1800);
      const e2 = p2.locator('dialog[open]').getByRole('button', { name: /Chỉnh sửa/ }); if (await e2.count()) { await e2.first().click(); await p2.waitForTimeout(1200); }
      const body2 = (await p2.locator('dialog[open]').allInnerTexts()).join(' ') + (await p2.locator('dialog[open] textarea,dialog[open] input').evaluateAll((els) => els.map((e) => e.value).join(' ')).catch(() => ''));
      draftKept = body2.includes(marker); await p2.screenshot({ path: SH + '/pe03-49-draft.png' });
    }
    const has = (s, ...w) => w.filter((x) => s.toLowerCase().includes(x.toLowerCase()));
    R('TC-PE03-49', draftKept === true && /nhân bản để sửa/.test(dr2) ? 'P' : 'F',
      `Drawer: câu nháp nút [${btn1.join('|')}]; câu code đang dùng trong bài: ${has(dr2, 'Lập trình', 'test', 'mẫu', 'ẩn', 'lời giải mẫu', 'ngôn ngữ', 'giới hạn').join(',')}; nút [${btn2.join('|')}]; khoá: ${/nhân bản để sửa/.test(dr2) ? '"Đang dùng trong bài thi … — nhân bản để sửa"' : 'KHÔNG thấy lời khoá'}; câu AI chờ duyệt: nút [${btn3.join('|')}] (Duyệt=${/Duyệt/.test(btn3.join())}); gõ dở "${marker}", đóng tab, mở lại: nháp còn=${draftKept}`);
    await ctx.close();
  }
  await browser.close();
})();
