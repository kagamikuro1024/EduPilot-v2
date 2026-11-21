// smkP: chạy lại TC-PE08-31 (R5-08/P-11-60) và TC-PE08-38 (R5-09/P-11-59) trên ep-r HEAD 88ab83e4. ONLY=res|staff node smkP_m2.cjs
const L = require('./ui_lib.cjs'); const { chromium, login, B, R, SH, metrics, axe, AUDIT_SRC, TOUCH_SRC } = L; const fs = require('fs');
const E30 = '01a1e136-5a4e-7b5d-a18d-a7ab3720c726', EMPTY = process.env.EMPTY || '01a1ef4d-c267-7355-982a-49fc185e67be', C1 = '01a1df30-523a-7d7a-a240-c120dbd57ac3'; const ONLY = process.env.ONLY || 'res';
const main = (p) => p.locator('main').innerText();
const audit = async (p, w) => { const a = await p.evaluate(`(${AUDIT_SRC.trim()})`); const opts = await p.evaluate(() => [...document.querySelectorAll('option')].map((o) => o.textContent.trim())); a.cut = a.cut.filter((c) => !opts.includes(c)); const t = w < 600 ? await p.evaluate(TOUCH_SRC) : []; return { ox: a.ox, cut: a.cut.length, ell: a.ell.length, touch: t.length }; };
(async () => { process.on('unhandledRejection', (e) => console.log('ERR', String(e).slice(0, 300)));
  const br = await chromium.launch({ headless: true });
  if (ONLY === 'res') {
    const res = {};
    for (const w of [1440, 375]) {
      const { ctx, p } = await login(br, 'r5s01', { width: w, height: 900 }); await p.goto(`${B}/exams/${E30}/take?course=${C1}`); await p.waitForTimeout(4000); await L.dismiss(p); const t0 = await main(p);
      const summary = (t0.match(/Bạn đúng \d+\/\d+ câu trắc nghiệm · bài lập trình: \d+\/\d+ test ẩn đạt/) || [null])[0]; const score = (t0.match(/\d+,\d+ \/ \d+(,\d+)?/) || [null])[0];
      const items = await p.locator('[data-part=result-item]').count(); const fold = await p.locator('[data-part=result-item] > details').count(); const open0 = await p.locator('[data-part=result-item] > details[open]').count();
      const hid0 = /Test ẩn: đạt \d+ trên \d+/.test(t0); const btns = await p.locator('main button:visible').allInnerTexts();
      const m = await metrics(p); const a0 = await audit(p, w);
      const sums = p.locator('[data-part=result-item] > details > summary'); for (let i = 0; i < await sums.count(); i++) await sums.nth(i).click(); await p.waitForTimeout(500);
      const t1 = await main(p); const hid1 = (t1.match(/Test ẩn: đạt \d+ trên \d+/) || [null])[0]; const open1 = await p.locator('[data-part=result-item] > details[open]').count(); const a1 = await audit(p, w); const ax = await axe(p);
      await p.screenshot({ path: SH + `/smkP-pe08-31-${w}.png` }); res[w] = { score, summary, items, fold, open0, hid0, hid1, open1, btns, ox: m.ox, a0, a1, ax }; await ctx.close(); }
    console.log(JSON.stringify(res));
    const r = res[1440], q = res[375];
    const ok = r.score && /,/.test(r.score) && r.summary && r.items > 0 && r.fold === r.items && r.open0 === 0 && !r.hid0 && r.open1 === r.items && r.hid1 && r.btns.length === 1 && /Gửi yêu cầu xem lại/.test(r.btns[0]) && q.summary && q.fold === q.items && q.open0 === 0 && q.a1.ox <= 0 && q.a0.cut === 0 && q.a1.cut === 0 && q.a1.touch === 0 && !r.ax.length && !q.ax.length;
    R('TC-PE08-31', ok ? 'P' : 'F', `smkP @88ab83e4 (473472ec, P-11-60): SV r5s01 (E30 đã công bố) 1440/375: điểm "${r.score}"; dòng tóm tắt "${r.summary}"; ${r.items} câu, gập mặc định ${r.fold}/${r.items} (mở sẵn ${r.open0}); trước khi mở câu không có "Test ẩn" (${!r.hid0}), sau khi mở cả ${r.open1} câu: "${r.hid1}"; nút phụ [${r.btns}]; 375: tóm tắt=${!!q.summary}, gập ${q.fold}/${q.items}, ox=${q.a1.ox} cut=${q.a1.cut} touch=${q.a1.touch}; axe 1440 ${JSON.stringify(r.ax)} / 375 ${JSON.stringify(q.ax)}`);
  }
  if (ONLY === 'staff') {
    const F = {};
    for (const [u, w] of [['r5t', 1440], ['r5t', 1100], ['r5t', 720], ['r5t', 719], ['r5t', 390], ['r5ta', 1440], ['r5ta', 390]]) {
      const { ctx, p } = await login(br, u, { width: w, height: 900 }); await p.goto(`${B}/exams/${E30}/results?course=${C1}`); await p.waitForTimeout(4500); await L.dismiss(p);
      const heads = await p.locator('thead th').allInnerTexts(); const tabs = await p.getByRole('tab').allInnerTexts(); const prim = await p.locator('[data-variant=primary]:visible').allInnerTexts(); const a = await audit(p, w);
      const row = p.locator('tbody tr').first(); await row.click(); await p.waitForTimeout(2500); const drawer = (await p.locator('dialog[open]').allInnerTexts()).join(' ').replace(/\n/g, ' ¦ ');
      const hidTxt = (drawer.match(/Test ẩn: đạt \d+ trên \d+/) || [null])[0]; const hidVerdict = /(?:test ẩn|h\d)[^¦]*\b(Đúng|Sai|Quá giờ)\b/i.test(drawer.replace(/Test ẩn: đạt \d+ trên \d+/g, ''));
      F[u + w] = { heads: heads.map((h) => h.trim()), tabs, prim, a, rows: await p.locator('tbody tr').count(), drawerHidden: hidTxt, hidVerdict, drawerMeta: { mssv: /R5S\d{4}/.test(drawer), nop: /Nộp lúc/.test(drawer) }, signalCol: heads.some((h) => /Tín hiệu/.test(h)) };
      await p.keyboard.press('Escape'); await ctx.close(); }
    { const { ctx, p } = await login(br, 'r5t', { width: 1440, height: 900 }); await p.goto(`${B}/exams/${EMPTY}/results?course=${C1}`); await p.waitForTimeout(4000); const t = await main(p); F.empty = { chuaBatDau: (t.match(/Chưa bắt đầu/g) || []).length, head: t.replace(/\n/g, ' ¦ ').slice(0, 200), rows: await p.locator('tbody tr').count(), prog: (t.match(/Đang làm \d+[^\n]*/) || [''])[0] }; await p.screenshot({ path: SH + '/smkP-pe08-38-empty.png' });
      await p.goto(`${B}/exams/${E30}/results?course=${C1}`); await p.waitForTimeout(3000); await p.getByRole('tab', { name: 'Thống kê' }).click(); await p.waitForTimeout(1500); F.stats = /Phân bố/.test(await main(p)); await p.getByRole('tab', { name: 'Nghi giống nhau' }).click(); await p.waitForTimeout(1500); F.sim = /Nghi giống nhau/.test(await main(p)); await ctx.close(); }
    console.log(JSON.stringify(F, null, 1));
    const wide = ['r5t1440', 'r5t1100', 'r5t720'].every((k) => ['Họ tên', 'MSSV', 'Trạng thái', 'Điểm', 'Nộp lúc'].every((h) => F[k].heads.includes(h)) && F[k].signalCol && F[k].a.ox <= 0 && F[k].a.cut === 0);
    const narrow = ['r5t719', 'r5t390', 'r5ta390'].every((k) => JSON.stringify(F[k].heads) === JSON.stringify(['Họ tên', 'Trạng thái', 'Điểm']) && F[k].a.ox <= 0 && F[k].a.cut === 0 && F[k].a.ell === 0 && F[k].drawerMeta.mssv);
    const ta = !F.r5ta1440.signalCol && !F.r5ta1440.tabs.some((t) => /Nghi giống/.test(t)); const hid = ['r5t1440', 'r5t1100'].every((k) => F[k].drawerHidden && !F[k].hidVerdict);
    const emp = F.empty.chuaBatDau >= 1 && F.empty.rows > 1; const ok = wide && narrow && ta && hid && emp && F.stats && F.sim;
    R('TC-PE08-38', ok ? 'P' : 'F', `smkP @88ab83e4 (247bf4e4, P-11-59): ≥720 cột ${JSON.stringify(F.r5t1440.heads)} (GV có Tín hiệu=${F.r5t1440.signalCol}) ok=${wide}; <720 (719/390 GV, 390 TA) chỉ ${JSON.stringify(F.r5t390.heads)}, AUDIT 390 ${JSON.stringify(F.r5t390.a)}, 719 ${JSON.stringify(F.r5t719.a)}, MSSV (mã SV cạnh tên)/Nộp lúc ở Drawer=${F.r5t390.drawerMeta.mssv}/${F.r5t390.drawerMeta.nop} ok=${narrow}; TA không Tín hiệu/Nghi giống nhau ok=${ta}; Drawer test ẩn chỉ "${F.r5t1440.drawerHidden}", verdict từng test ẩn=${F.r5t1440.hidVerdict} ok=${hid}; bài chưa ai làm: ${F.empty.rows} hàng, ${F.empty.chuaBatDau}× «Chưa bắt đầu» ok=${emp}; tab Thống kê=${F.stats}, Nghi giống nhau=${F.sim}`);
  }
  await br.close(); })();
