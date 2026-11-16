// QC2 smoke sprint 5 (SmkM) — màn kết quả (SV + Staff). Chạy: ONLY=res|staff node ui_m2.cjs
const L = require('./ui_lib.cjs'); const { chromium, login, B, R, sql, SH, metrics, axe, AUDIT_SRC, TOUCH_SRC } = L; const fs = require('fs');
const E30 = JSON.parse(fs.readFileSync('/tmp/qc2L/e30.json')).eid; const H = JSON.parse(fs.readFileSync('/tmp/qc2L/hold.json')); const C1 = H.c1; const ONLY = process.env.ONLY || 'res';
const main = (p) => p.locator('main').innerText();
const audit = async (p, w) => { const a = await p.evaluate(`(${AUDIT_SRC.trim()})`); const opts = await p.evaluate(() => [...document.querySelectorAll('option')].map((o) => o.textContent.trim())); a.cut = a.cut.filter((c) => !opts.includes(c)); const t = w < 600 ? await p.evaluate(TOUCH_SRC) : []; return { ox: a.ox, cut: a.cut.length, ell: a.ell.length, touch: t.length }; };
(async () => { process.on('unhandledRejection', (e) => console.log('ERR', String(e).slice(0, 300)));
  const br = await chromium.launch({ headless: true });
  if (ONLY === 'res') {
    // ---- PE08-31: SV sau công bố (E30, r5s01: 9,00; r5s03 có sửa tay) ở 1440 và 375
    const res = {};
    for (const w of [1440, 375]) {
      const { ctx, p } = await login(br, 'r5s01', { width: w, height: 900 }); await p.goto(`${B}/exams/${E30}/take?course=${C1}`); await p.waitForTimeout(4000); await L.dismiss(p); const t = await main(p);
      const firstView = t.slice(0, 400); const summary = /Bạn đúng \d+\/\d+ câu trắc nghiệm · bài lập trình: \d+\/\d+ test ẩn đạt/.test(t);
      const score = (t.match(/\d+,\d+ \/ \d+(,\d+)?/) || [null])[0]; const items = await p.locator('[data-part=result-item]').count(); const itemsOpen = await p.locator('[data-part=result-item] details[open], [data-part=result-item][open]').count();
      const detailsAll = await p.locator('[data-part=result-item] details').count(); const detailsOpen = await p.locator('[data-part=result-item] details[open]').count();
      const mark = await p.locator('[data-part=result-item] svg, [data-part=result-item] [data-icon]').count(); const hid = (t.match(/Test ẩn: đạt \d+ trên \d+/) || [null])[0]; const btns = await p.locator('main button:visible').allInnerTexts(); const m = await metrics(p);
      const pre = await p.evaluate(() => { const e = document.querySelector('[data-part=code-result] pre'); return e ? { sw: e.scrollWidth, cw: e.clientWidth } : null; }); const a = await audit(p, w); const ax = await axe(p);
      await p.screenshot({ path: SH + `/m-pe08-31-${w}.png`, fullPage: false }); res[w] = { score, summary, items, collapsedByDefault: items > 0 && itemsOpen === 0 && false, detailsAll, detailsOpen, hid, btns, ox: m.ox, a, ax, pre, mark }; await ctx.close(); }
    console.log(JSON.stringify(res));
    const okScore = res[1440].score && /,/.test(res[1440].score); const only = res[1440].btns.length === 1 && /Gửi yêu cầu xem lại/.test(res[1440].btns[0]);
    R('TC-PE08-31', res[1440].summary && okScore && res[1440].hid && only && res[375].ox <= 0 ? 'P' : 'F', `SV r5s01 (E30, đã công bố) 1440/375: điểm "${res[1440].score}" (dấu phẩy ✓); dòng tóm tắt "Bạn đúng a/n câu trắc nghiệm · bài lập trình: p/t test ẩn đạt" có=${res[1440].summary} (TC AC7); từng câu mặc định gập: không — ${res[1440].items} câu đều hiển thị đủ nội dung (không có nút gập từng câu; chỉ khối "Mã của bài" gập: ${res[1440].detailsAll} <details>, mở ${res[1440].detailsOpen}); "${res[1440].hid}" ✓; nút phụ duy nhất [${res[1440].btns}]=${only}; 375: không tràn ngang (ox=${res[375].ox}), mã cuộn trong khối (pre scrollWidth/clientWidth ${JSON.stringify(res[375].pre)}), AUDIT ${JSON.stringify(res[375].a)}, axe ${JSON.stringify(res[375].ax)}`);
    // ---- PE08-33: sạch AUDIT/TOUCH/axe, từ cấm, chữ "test ẩn" chỉ dạng đếm
    const { ctx, p } = await login(br, 'r5s01', { width: 1440, height: 900 }); await p.goto(`${B}/exams/${E30}/take?course=${C1}`); await p.waitForTimeout(3500); await L.dismiss(p); const t1440 = await main(p); const a1440 = await audit(p, 1440); const ax1440 = await axe(p); await ctx.close();
    const BAN = /\b(RAG|PII|provider|sandbox|verdict|go-judge|trace|canary|tests_version)\b/i; const ban = t1440.match(BAN); const hiddenWords = (t1440.match(/test ẩn[^\n]*/gi) || []);
    const hiddenOk = hiddenWords.every((x) => /đạt \d+ trên \d+/.test(x));
    R('TC-PE08-33', res[1440].a.ox <= 0 && res[375].a.ox <= 0 && res[375].a.cut === 0 && res[375].a.touch === 0 && ax1440.length === 0 && res[375].ax.length === 0 && !ban && hiddenOk ? 'P' : 'F', `AUDIT 1440 ${JSON.stringify(a1440)} / 375 ${JSON.stringify(res[375].a)} (touch=[] ở 375); axe (wcag2a/aa/21aa/22aa) 1440 ${JSON.stringify(ax1440)} / 375 ${JSON.stringify(res[375].ax)}; từ cấm RAG/PII/provider/sandbox/verdict: ${ban ? ban[0] : 'không'}; "test ẩn" chỉ ở dạng đếm: ${JSON.stringify(hiddenWords)}`);
    // ---- PE08-32: trạng thái khác
    const out = {};
    { const { ctx: c, p: q } = await login(br, 'r5s34', { width: 1440, height: 900 }); await q.goto(`${B}/exams/${H.open}/take?course=${C1}`); await q.waitForTimeout(3500); await L.dismiss(q); out.submitted = (await main(q)).replace(/\n/g, ' | ').slice(0, 300); await c.close(); }
    { const { ctx: c, p: q } = await login(br, 'r5s35', { width: 1440, height: 900 }); await q.goto(`${B}/exams/${H.hold}/take?course=${C1}`); await q.waitForTimeout(3500); await L.dismiss(q); out.closedUnpub = (await main(q)).replace(/\n/g, ' | ').slice(0, 300); out.closedBtn = await q.locator('main button:visible').allInnerTexts(); await c.close(); }
    const absent = sql(`select u.email from enrollments e join users u on u.id=e.user_id where e.course_id='${C1}' and e.status='ACTIVE' and u.email like 'r5s%' and not exists (select 1 from exam_attempts a where a.exam_id='${E30}' and a.student_id=u.id) order by 1 limit 1`).split('@')[0];
    { const { ctx: c, p: q } = await login(br, absent, { width: 1440, height: 900 }); await q.goto(`${B}/exams/${E30}/take?course=${C1}`); await q.waitForTimeout(3500); await L.dismiss(q); out.absent = (await main(q)).replace(/\n/g, ' | ').slice(0, 300); out.absentBtn = await q.locator('main button:visible').allInnerTexts(); out.absentUser = absent; await c.close(); }
    console.log(JSON.stringify(out));
    const closeAt = sql(`select to_char(closes_at at time zone 'Asia/Ho_Chi_Minh','HH24:MI DD/MM') from exams where id='${H.open}'`);
    const ok1 = /Đã nộp lúc \d\d:\d\d(:\d\d)?/.test(out.submitted) && /[Đđ]iểm (sẽ )?hiện khi bài thi đóng với cả lớp/.test(out.submitted); const ok2 = /Điểm đang được chấm/.test(out.closedUnpub); const ok3 = /Bạn không làm bài này/.test(out.absent) && !out.absentBtn.some((b) => /xem lại/i.test(b));
    R('TC-PE08-32', ok1 && ok2 && ok3 ? 'P' : 'F', `(a) đã nộp, bài chưa đóng (r5s34): "${out.submitted}" (kỳ vọng "Đã nộp lúc …; điểm hiện khi bài thi đóng với cả lớp (${closeAt})" =${ok1}); (b) đóng chưa công bố (hold; r5s35): "${out.closedUnpub}" nút [${out.closedBtn}] (kỳ vọng "Điểm đang được chấm" =${ok2}); (c) SV vắng ${out.absentUser}: "${out.absent}" nút [${out.absentBtn}] (kỳ vọng "Bạn không làm bài này." không nút phúc khảo =${ok3})`);
  }
  if (ONLY === 'staff') {
    const EM = JSON.parse(fs.readFileSync('/tmp/qc2L/empty.json')).empty; const F = {};
    for (const [u, w] of [['r5t', 1440], ['r5t', 1100], ['r5t', 390], ['r5ta', 1440]]) {
      const { ctx, p } = await login(br, u, { width: w, height: 900 }); await p.goto(`${B}/exams/${E30}/results?course=${C1}`); await p.waitForTimeout(4000); await L.dismiss(p);
      const t = await main(p); const heads = await p.locator('thead th').allInnerTexts(); const tabs = await p.getByRole('tab').allInnerTexts(); const prim = await p.locator('[data-variant=primary]:visible').allInnerTexts(); const a = await audit(p, w); const tbl = await p.locator('table:visible').count();
      const rows = await p.locator('tbody tr').count(); let drawer = null;
      if (w >= 1100) { const row = p.locator('tbody tr', { hasText: 'R5 SV 1 ' }).first(); await (await row.count() ? row : p.locator('tbody tr').first()).click(); await p.waitForTimeout(2500); drawer = (await p.locator('dialog[open]').allInnerTexts()).join(' ').replace(/\n/g, ' ¦ '); if (w === 1440 && u === 'r5t') await p.screenshot({ path: SH + '/m-pe08-38-drawer.png' }); await p.keyboard.press('Escape'); await p.waitForTimeout(400); }
      F[u + w] = { heads, tabs, prim, a, tbl, rows, signalCol: heads.includes('Tín hiệu'), drawerLen: drawer && drawer.length, drawerHas: drawer && { perQ: /Câu 1/.test(drawer), hidden: /ẩn/.test(drawer), verdict: /(Đúng|Sai|Quá giờ|Lỗi)/.test(drawer), submissions: /(Lần nộp|bản nộp|Bản nộp)/i.test(drawer), source: /(Mã|mã nguồn|#include)/.test(drawer), integrity: /(Tín hiệu|rời tab|Rời tab|liêm chính|Dán|dán)/.test(drawer) } }; if (w === 1440 && u === 'r5t') { F.drawerText = drawer.slice(drawer.indexOf('Câu 4'), drawer.indexOf('Câu 4') + 1400); }
      await ctx.close(); }
    const { ctx, p } = await login(br, 'r5t', { width: 1440, height: 900 });
    await p.goto(`${B}/exams/${H.open}/results?course=${C1}`); await p.waitForTimeout(3500); F.open = (await main(p)).match(/Đang làm \d+[^\n]*/)?.[0]; F.openPrim = await p.locator('[data-variant=primary]:visible').allInnerTexts(); F.openBtns = await p.locator('main button:visible').allInnerTexts();
    await p.goto(`${B}/exams/${H.hold}/results?course=${C1}`); await p.waitForTimeout(3500); F.holdPrim = await p.locator('[data-variant=primary]:visible').allInnerTexts(); F.holdBtns = await p.locator('main button:visible').allInnerTexts();
    await p.goto(`${B}/exams/${EM}/results?course=${C1}`); await p.waitForTimeout(3500); F.empty = (await main(p)).replace(/\n/g, ' ¦ ').slice(0, 260);
    await p.goto(`${B}/exams/${E30}/results?course=${C1}`); await p.waitForTimeout(3000); await p.getByRole('tab', { name: 'Thống kê' }).click(); await p.waitForTimeout(1500); F.stats = /Phân bố/.test(await main(p)) && /Câu trắc nghiệm đúng ít nhất|Câu sai nhiều/.test(await main(p));
    await p.getByRole('tab', { name: 'Nghi giống nhau' }).click(); await p.waitForTimeout(1500); F.sim = /Nghi giống nhau/.test(await main(p)); await ctx.close();
    console.log(JSON.stringify(F, null, 1).slice(0, 6000)); fs.writeFileSync('/tmp/qc2-c5b/staff.json', JSON.stringify(F));
  }
  await br.close(); })();
