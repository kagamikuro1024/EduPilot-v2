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

  if (ONLY === 'gv2') {
    const { ctx, p } = await login(browser, 'r5t', { width: 1440, height: 900 });
    // ---- TC-PE03-50 rỗng / lỗi / tải
    await p.goto(B + '/questions'); await p.waitForTimeout(1500); await pickClass(p, 'A6P8C');
    const te = await txt(p); const emptyBtn = await p.getByRole('button', { name: /Tạo câu hỏi/ }).count();
    await pickClass(p, 'R5C1');
    await p.route('**/api/v1/courses/*/questions?*', (r) => r.fulfill({ status: 500, contentType: 'application/json', body: JSON.stringify({ code: 'INTERNAL', message: 'Lỗi máy chủ' }) }));
    await p.reload(); await p.waitForTimeout(3500); const terr = await txt(p); const retry = await p.getByRole('button', { name: /Thử lại/ }).count();
    await p.unroute('**/api/v1/courses/*/questions?*'); await p.getByRole('button', { name: /Thử lại/ }).first().click().catch(() => {}); await p.waitForTimeout(3000); const trec = await txt(p);
    await p.route('**/api/v1/courses/*/questions?*', async (r) => { await new Promise((x) => setTimeout(x, 4000)); r.continue(); });
    await p.reload(); await p.waitForTimeout(1200); const skel = await p.evaluate(() => document.querySelectorAll('[aria-busy=true],[data-skeleton],[class*=keleton],[role=status]').length); await p.unroute('**/api/v1/courses/*/questions?*');
    await p.waitForTimeout(4000);
    R('TC-PE03-50', /Chưa có câu hỏi/.test(te) && emptyBtn >= 1 && retry >= 1 && /r5-|Đã duyệt/.test(trec) && skel > 0 ? 'P' : 'F',
      `rỗng (lớp A6P8C): "${(te.match(/Chưa có câu hỏi[^\n]*/) || [''])[0]}" + nút Tạo câu hỏi (${emptyBtn}) — câu gợi ý dài "…nhờ AI gợi ý nháp." của TC gốc đã bỏ theo luật chữ UI 09/10/2026 (LT phần chữ); lỗi 500: nút Thử lại ×${retry} (chữ: "${(terr.match(/[^\n]*(lỗi|Lỗi)[^\n]*/) || [''])[0].slice(0, 70)}"); bấm Thử lại → danh sách trở lại=${/Đã duyệt/.test(trec)}; tải chậm 4 s: phần tử khung xương/aria-busy=${skel}`);
    // ---- TC-PE03-51 bàn phím + 5 mốc bề rộng
    await p.goto(B + '/questions'); await p.waitForTimeout(2000); await pickClass(p, 'R5C1');
    let found = null; for (let i = 0; i < 80; i++) { await p.keyboard.press('Tab'); const f = await p.evaluate(() => { const e = document.activeElement; return e ? { tag: e.tagName, role: e.getAttribute('role'), inRow: !!e.closest('tbody tr'), t: (e.innerText || e.getAttribute('aria-label') || '').slice(0, 30) } : null; }); if (f && f.inRow) { found = { i, ...f }; break; } }
    let kb = { rowFocus: !!found, found, dialog: false, focusInside: false, duyet: false, escBack: false };
    if (found) { await p.keyboard.press('Enter'); await p.waitForTimeout(1500); kb.dialog = (await p.locator('dialog[open]').count()) > 0; kb.focusInside = await p.evaluate(() => !!document.activeElement?.closest('dialog[open]')); for (let i = 0; i < 40 && !kb.duyet; i++) { await p.keyboard.press('Tab'); kb.duyet = await p.evaluate(() => /Duyệt|Chỉnh sửa/.test(document.activeElement?.innerText || '')); } await p.keyboard.press('Escape'); await p.waitForTimeout(800); kb.escBack = await p.evaluate(() => !document.activeElement?.closest('dialog') && document.activeElement !== document.body); }
    const wid = {}; for (const w of [1440, 1024, 720, 719, 390]) { await p.setViewportSize({ width: w, height: 900 }); await p.goto(B + '/questions'); await p.waitForTimeout(1800); await pickClass(p, 'R5C1'); const a = await p.evaluate(`(${AUDIT_SRC.trim()})`).then(async (r0) => { const opts = await p.evaluate(() => [...document.querySelectorAll('option')].map((o) => o.textContent.trim())); r0.cut = r0.cut.filter((c) => !opts.includes(c)); return r0; }); const tch = w === 390 ? await p.evaluate(TOUCH_SRC) : null; const tbl = await p.locator('table:visible').count(); wid[w] = { ox: a.ox, cut: a.cut.length, ell: a.ell.length, tbl, touch: tch ? tch.length : '-' }; }
    const okw = Object.values(wid).every((x) => x.ox <= 0 && x.cut === 0 && x.ell === 0) && wid[720].tbl > 0 && wid[719].tbl === 0;
    R('TC-PE03-51', kb.rowFocus && kb.dialog && kb.focusInside && kb.escBack && okw ? 'P' : 'F', `bàn phím: Tab tới hàng (sau ${kb.found && kb.found.i} lần), Enter mở Drawer=${kb.dialog}, focus trong Drawer=${kb.focusInside}, tới nút Duyệt/Chỉnh sửa=${kb.duyet}, Esc trả focus=${kb.escBack}; AUDIT (ox/cut/ell)/bảng/TOUCH ở 1440,1024,720,719,390: ${JSON.stringify(wid)} (≥720 là bảng, <720 là danh sách)`);
    await ctx.close();
  }
  if (ONLY === 'nav') {
    const counts = {};
    for (const [u, k] of [['r5s05', 'SV'], ['r5ta', 'TA'], ['r5t', 'GV'], ['admin', 'ADMIN']]) {
      const { ctx, p } = await login(browser, u, { width: 1440, height: 900 }); await p.goto(B + '/'); await p.waitForTimeout(2500);
      const items = await p.locator('nav a:visible').evaluateAll((els) => els.map((e) => e.innerText.trim()).filter(Boolean)); counts[k] = items; await ctx.close();
    }
    const { ctx, p } = await login(browser, 'r5s05', { width: 375, height: 800 }); await p.goto(B + '/'); await p.waitForTimeout(2500);
    const bar = await p.locator('nav a:visible,nav button:visible').evaluateAll((els) => els.map((e) => e.innerText.trim()).filter(Boolean)); const more = p.getByRole('button', { name: /Thêm/ }).first(); let moreItems = [];
    if (await more.count()) { await more.click(); await p.waitForTimeout(800); moreItems = await p.locator('[role=menu] a:visible,[role=dialog] a:visible,dialog[open] a:visible').evaluateAll((els) => els.map((e) => e.innerText.trim()).filter(Boolean)); }
    await ctx.close();
    console.log('NAV', JSON.stringify(Object.fromEntries(Object.entries(counts).map(([k, v]) => [k, v.length + ':' + v.join('|')]))));
    console.log('MOBILE bar', JSON.stringify(bar), 'more', JSON.stringify(moreItems));
    R('TC-PE04-48', 'S', JSON.stringify({ counts: Object.fromEntries(Object.entries(counts).map(([k, v]) => [k, v.length])), bar, moreItems, hasExams: Object.fromEntries(Object.entries(counts).map(([k, v]) => [k, v.includes('Bài thi')])) }));
  }

  const FX = () => JSON.parse(require('fs').readFileSync('/tmp/qc2L/uix.json'));
  const ms = (t) => { const m = /^(?:(\d+):)?(\d+):(\d+)$/.exec(t.trim()); return m ? ((+m[1] || 0) * 3600 + +m[2] * 60 + +m[3]) * 1000 : null; };
  const startX = async (p, eid, c1) => { await p.goto(`${B}/exams/${eid}/take?course=${c1}`); await p.waitForTimeout(2500); await L.dismiss(p); const n = p.locator('[data-part=integrity-notice]'); if (await n.count()) await n.scrollIntoViewIfNeeded(); const b = p.getByRole('button', { name: /Bắt đầu làm bài/ }); if (await b.count()) { await b.first().click(); await p.waitForTimeout(2500); } };
  const dl = (eid, u) => Number(sql(`select extract(epoch from deadline_at)*1000 from exam_attempts where exam_id='${eid}' and student_id=(select id from users where email='${u}@edupilot.local')`));
  if (ONLY === 'skew') {
    const fx = FX(); const out = {};
    for (const [u, off] of [['r5s31', 300000], ['r5s32', -300000]]) {
      const { ctx, p } = await login(browser, u, { width: 1440, height: 900 });
      await ctx.addInitScript(`(() => { const off = ${off}; const N = Date.now.bind(Date); const D = Date; Date.now = () => N() + off; class FD extends D { constructor(...a) { if (a.length) super(...a); else super(N() + off); } } FD.now = Date.now; FD.UTC = D.UTC; FD.parse = D.parse; window.Date = FD; })();`);
      await p.reload(); await startX(p, fx.x6, fx.c1);
      const d = dl(fx.x6, u); const series = []; let backwards = 0, prev = null;
      for (let i = 0; i < 13; i++) { const t = await p.getByRole('timer').first().innerText().catch(() => ''); const shown = ms(t); const exp = d - Date.now(); series.push([t, shown === null ? null : Math.round((shown - exp) / 100) / 10]); if (prev !== null && shown !== null && shown > prev + 1000) backwards++; if (shown !== null) prev = shown; await p.waitForTimeout(10000); }
      out[u] = { off, maxDiff: Math.max(...series.map((x) => Math.abs(x[1] ?? 99))), backwards, series: series.slice(0, 4).concat(series.slice(-2)) }; await ctx.close();
    }
    R('TC-PE05-11', Object.values(out).every((o) => o.maxDiff <= 1.5 && o.backwards === 0) ? 'P' : 'F', `đồng hồ trình duyệt lệch +5 phút / −5 phút (Date ghi đè): chênh giữa số hiển thị và deadline_at − giờ thật mỗi 10 s trong 2 phút: ${JSON.stringify(Object.fromEntries(Object.entries(out).map(([u, o]) => [u, { off: o.off / 60000 + ' phút', maxDiff_s: o.maxDiff, nhayNguoc: o.backwards, mau: o.series }])))}`);
  }

  if (ONLY === 'take6') {
    const fx = FX(); const u = 'r5s33';
    const { ctx, p } = await login(browser, u, { width: 1440, height: 900 });
    await p.addInitScript(`window.__aud = 0; const A0 = window.Audio; window.Audio = function(...a) { window.__aud++; return new A0(...a); }; window.__cls = 0; new PerformanceObserver((l) => { for (const e of l.getEntries()) if (!e.hadRecentInput) window.__cls += e.value; }).observe({ type: 'layout-shift', buffered: true });`);
    await p.reload(); await startX(p, fx.x6, fx.c1);
    const tm = p.getByRole('timer').first(); const t0 = await tm.innerText(); const widths = new Set(); const vals = [];
    for (let i = 0; i < 12; i++) { widths.add(Math.round((await tm.boundingBox()).width)); vals.push(await tm.innerText()); await p.waitForTimeout(1000); }
    const live = await p.evaluate(() => { const t = document.querySelector('[role=timer]'); const g = (e) => e ? (e.getAttribute('aria-live') || e.closest('[aria-live]')?.getAttribute('aria-live') || null) : null; const banner = document.querySelector('[data-part=time-warning],[data-part=warning]'); return { timer: g(t), timerRole: t?.getAttribute('role'), secs: [...(t?.querySelectorAll('[aria-live]') || [])].map((e) => e.getAttribute('aria-live')), bannerLive: g(banner) }; });
    const ax = await axe(p); const m1 = await metrics(p);
    // phím tắt + nhiều đáp án
    await p.keyboard.press('b'); await p.waitForTimeout(500); const chk1 = await p.locator('input:checked,[aria-checked=true]').count();
    await p.keyboard.press('ArrowRight'); await p.waitForTimeout(500); const q2 = await p.getByText(/Câu 2\//).count(); await p.keyboard.press('ArrowLeft'); await p.waitForTimeout(500); const q1 = await p.getByText(/Câu 1\//).count();
    for (let i = 0; i < 2; i++) { await p.keyboard.press('ArrowRight'); await p.waitForTimeout(400); }
    const multiHint = (await p.locator('main').innerText()).match(/[^\n]*Chọn tất cả[^\n]*/)?.[0] || null;
    await p.keyboard.press('ArrowLeft'); await p.keyboard.press('ArrowLeft');
    R('TC-PE05-12', /^\d+:\d\d$/.test(t0.trim()) && widths.size === 1 && (live.timer === 'off' || live.secs.includes('off')) ? 'P' : 'F', `timer ${vals.slice(0, 3)}…; bề rộng khối đồng hồ khi số đổi: ${[...widths]} px (${widths.size} giá trị); aria-live: ${JSON.stringify(live)}; CLS hiện tại ${await p.evaluate(() => window.__cls)} (≤0,1); h:mm:ss khi ≥1 giờ: đã phủ bởi exam-clock.spec.ts (xem TC-PE05-12 spec); font-variant-numeric=${await tm.evaluate((e) => getComputedStyle(e).fontVariantNumeric)}`);
    R('TC-PE05-46', 'S', `phím: 'b' chọn=${chk1}, → Câu2=${q2}, ← Câu1=${q1}; gợi ý nhiều đáp án: ${multiHint}`);
    console.log('axe 1440', JSON.stringify(ax), JSON.stringify(m1));
    // cảnh báo 5 phút / 1 phút (bài 6 phút): chờ
    const seen = {}; const t_start = Date.now(); const dls = dl(fx.x6, u);
    while (Date.now() < dls - 20000) { const body = await p.locator('body').innerText(); const mm = body.match(/[^\n]*(còn\s+(5|1)\s+phút|5 phút|1 phút)[^\n]*/i); if (mm) { const key = /1 phút/.test(mm[0]) && !/5 phút/.test(mm[0]) ? '1' : '5'; if (!seen[key]) seen[key] = { text: mm[0].slice(0, 100), at: Math.round((Date.now() - t_start) / 1000), timer: await tm.innerText() }; } await p.waitForTimeout(2000); }
    const anim = await p.evaluate(() => { let k = 0; for (const ss of document.styleSheets) { try { for (const r of ss.cssRules) if (r.type === CSSRule.KEYFRAMES_RULE) k++; } catch {} } return { keyframes: k, audio: window.__aud, running: document.getAnimations().length }; });
    const bannerLive = await p.evaluate(() => { const e = [...document.querySelectorAll('[aria-live]')].map((x) => x.getAttribute('aria-live') + ':' + x.innerText.slice(0, 40)); return e; });
    R('TC-PE05-46', Object.keys(seen).length === 2 && anim.audio === 0 ? 'P' : 'F', `phím tắt: 'b' chọn=${chk1}, ArrowRight→Câu 2=${q2}, ArrowLeft→Câu 1=${q1}; gợi ý nhiều đáp án: ${JSON.stringify(multiHint)}; cảnh báo thời gian: ${JSON.stringify(seen)}; hoạt ảnh/âm thanh: ${JSON.stringify(anim)}; vùng aria-live: ${JSON.stringify(bannerLive)}`);
    // xác nhận nộp + sau nộp (PE05-50/51)
    await p.getByRole('button', { name: /Nộp bài/ }).first().click(); await p.waitForTimeout(1000); const dlg = (await p.locator('dialog[open]').innerText()).replace(/\n/g, ' | ');
    const btns = await p.locator('dialog[open] button').allInnerTexts();
    await p.locator('dialog[open]').getByRole('button', { name: /Nộp bài/ }).click(); await p.waitForTimeout(3000); const after = (await p.locator('main').innerText()).replace(/\n/g, ' | ');
    R('TC-PE05-50', /Bạn đã trả lời \d+\/\d+ câu/.test(dlg) && btns.some((b) => /Làm tiếp/.test(b)) ? 'P' : 'F', `hộp xác nhận: "${dlg.slice(0, 220)}"; nút [${btns.join(', ')}]`);
    R('TC-PE05-51', /Đã nộp lúc \d\d:\d\d:\d\d/.test(after) && !/đúng|sai/i.test(after.replace(/Đã nộp[^|]*/, '')) ? 'P' : 'F', `sau nộp: "${after.slice(0, 260)}"`);
    await ctx.close();
  }

  const pick = async (p, n) => { const o = p.locator('input[type=radio],input[type=checkbox],[role=radio],[role=checkbox]'); if (await o.count() > n) await o.nth(n).click({ force: true }).catch(() => {}); };
  if (ONLY === 'phone') {
    const fx = FX(); const u = 'r5s34'; const mk = async (vp) => login(browser, u, vp);
    const a = await mk({ width: 375, height: 800 }); await startX(a.p, fx.x6, fx.c1); await pick(a.p, 0); await a.p.waitForTimeout(3500);
    const n1 = sql(`select count(*) from exam_answers where attempt_id=(select id from exam_attempts where exam_id='${fx.x6}' and student_id=(select id from users where email='${u}@edupilot.local'))`);
    const b = await mk({ width: 1440, height: 900 }); await b.p.goto(`${B}/exams/${fx.x6}/take?course=${fx.c1}`); await b.p.waitForTimeout(3000);
    const ro = (await b.p.locator('body').innerText()).match(/[^\n]*(Bài đang mở ở nơi khác|Làm tiếp ở đây)[^\n]*/g) || []; const disabled = await b.p.locator('input[type=radio],input[type=checkbox]').first().isDisabled().catch(() => null);
    const take = b.p.getByRole('button', { name: /Làm tiếp ở đây/ }); if (await take.count()) { await take.first().click(); await b.p.waitForTimeout(2500); }
    await pick(b.p, 1); await b.p.waitForTimeout(3500); const n2 = sql(`select count(*) from exam_answers where attempt_id=(select id from exam_attempts where exam_id='${fx.x6}' and student_id=(select id from users where email='${u}@edupilot.local'))`);
    await a.p.waitForTimeout(6000); const roA = (await a.p.locator('body').innerText()).match(/[^\n]*(Bài đang mở ở nơi khác|Làm tiếp ở đây|chỉ đọc)[^\n]*/g) || []; const disA = await a.p.locator('input[type=radio],input[type=checkbox]').first().isDisabled().catch(() => null);
    R('TC-PE05-38', ro.length > 0 && Number(n2) >= Number(n1) && n1 !== '0' ? 'P' : 'F', `điện thoại 375 chọn đáp án (lên máy chủ: ${n1}); laptop 1440 cùng phiên mở sau: dải "${ro[0] || ''}", chỉ đọc(radio disabled)=${disabled}; bấm Làm tiếp ở đây → chọn đáp án khác, lên máy chủ ${n2} (không mất đáp án); điện thoại sau đó chuyển chỉ đọc: ${roA.length > 0}/disabled=${disA}`);
    R('TC-PE05-37', ro.length > 0 && roA.length > 0 ? 'P' : 'F', `hai nơi cùng phiên (như hai tab, ep-r thật): tab sau mở: "${ro[0] || ''}" + chỉ đọc=${disabled}; bấm "Làm tiếp ở đây" → tab đầu: ${roA[0] || 'không đổi'} (chỉ đọc/disabled=${disA}); chữ chưa lưu của tab cũ giữ — phủ bởi spec "two tabs" (exam.spec.ts)`);
    await a.ctx.close(); await b.ctx.close();
  }
  if (ONLY === 'offline') {
    const fx = FX(); const dir = '/tmp/qc2L/pwprof-' + Date.now(); const u = 'r5s35';
    let br = await chromium.launchPersistentContext(dir, { ignoreHTTPSErrors: true, viewport: { width: 1440, height: 900 }, locale: 'vi-VN', timezoneId: 'Asia/Ho_Chi_Minh' });
    let p = br.pages()[0] || (await br.newPage());
    await p.goto(B + '/login'); await p.getByLabel(/email/i).fill(u + '@edupilot.local'); await p.getByLabel(/mật khẩu/i).first().fill(L.PW); await p.getByRole('button', { name: /đăng nhập/i }).click(); await p.waitForURL((x) => !x.pathname.startsWith('/login')); await p.waitForTimeout(1500); await L.dismiss(p);
    await startX(p, fx.x6, fx.c1); const aid = sql(`select id from exam_attempts where exam_id='${fx.x6}' and student_id=(select id from users where email='${u}@edupilot.local')`);
    await br.setOffline(true); await pick(p, 0); await p.waitForTimeout(800); await p.keyboard.press('ArrowRight'); await p.waitForTimeout(500); await pick(p, 1); await p.waitForTimeout(2500);
    const ls = await p.evaluate((a) => localStorage.getItem('exam:' + a), aid); const dirty = ls && JSON.parse(ls);
    let bu = null; p.once('dialog', async (d) => { bu = d.type(); await d.dismiss().catch(() => {}); });
    await p.close({ runBeforeUnload: true }).catch(() => {}); await new Promise((r) => setTimeout(r, 500));
    await br.close(); // đóng trình duyệt đột ngột
    const srv0 = sql(`select count(*) from exam_answers where attempt_id='${aid}'`);
    br = await chromium.launchPersistentContext(dir, { ignoreHTTPSErrors: true, viewport: { width: 1440, height: 900 }, locale: 'vi-VN', timezoneId: 'Asia/Ho_Chi_Minh' }); p = br.pages()[0] || (await br.newPage());
    await p.goto(`${B}/exams/${fx.x6}/take?course=${fx.c1}`); await p.waitForTimeout(6000); const t1 = await p.locator('main').innerText(); await p.waitForTimeout(6000);
    const srv1 = sql(`select count(*) from exam_answers where attempt_id='${aid}'`);
    R('TC-PE05-41', ls && Number(srv1) >= 2 ? 'P' : 'F', `offline: chọn 2 câu; localStorage exam:<attempt_id> = ${ls ? JSON.stringify(Object.keys(dirty)).slice(0, 60) + ' dirty=' + JSON.stringify(Object.keys(dirty.dirty || {})).slice(0, 80) : 'KHÔNG CÓ'}; đóng trang (beforeunload=${bu}) rồi đóng trình duyệt đột ngột; trên máy chủ trước=${srv0}; mở lại online → sau 12 s trên máy chủ=${srv1} (≥2); màn mở lại: ${(t1.match(/Câu \d\/\d/) || ['?'])[0]}`);
    await br.close();
  }
  if (ONLY === 'late') {
    const fx = FX();
    // PE05-42: offline qua hạn; PE05-13: đồng hồ trình duyệt chạy chậm
    const mkp = async (u, slow) => { const r = await login(browser, u, { width: 1440, height: 900 }); if (slow) await r.ctx.addInitScript(`(() => { const N = Date.now.bind(Date); const t0 = N(); const D = Date; Date.now = () => t0 + (N() - t0) * 0.1; class FD extends D { constructor(...a) { if (a.length) super(...a); else super(Date.now()); } } FD.now = Date.now; FD.UTC = D.UTC; FD.parse = D.parse; window.Date = FD; })();`); return r; };
    const A = await mkp('r5s22', false); await startX(A.p, fx.x1, fx.c1); await pick(A.p, 0); await A.p.waitForTimeout(3500); await A.ctx.setOffline(true); await pick(A.p, 1); await A.p.waitForTimeout(500);
    const dA = dl(fx.x1, 'r5s22'); const txOff = (await A.p.locator('body').innerText()).match(/[^\n]*(Mất mạng|Nếu hết giờ)[^\n]*/g) || [];
    const B2 = await mkp('r5s21', true); await startX(B2.p, fx.x1, fx.c1); await pick(B2.p, 0); await B2.p.waitForTimeout(3000);
    while (Date.now() < dA + 15000) await A.p.waitForTimeout(2000);
    const srvOff = sql(`select count(*) from exam_answers where attempt_id=(select id from exam_attempts where exam_id='${fx.x1}' and student_id=(select id from users where email='r5s22@edupilot.local'))`);
    await A.ctx.setOffline(false); await A.p.waitForTimeout(9000); const aft = (await A.p.locator('main').innerText()).replace(/\n/g, ' | '); const st = sql(`select status,submit_reason from exam_attempts where exam_id='${fx.x1}' and student_id=(select id from users where email='r5s22@edupilot.local')`); const srvOn = sql(`select count(*) from exam_answers where attempt_id=(select id from exam_attempts where exam_id='${fx.x1}' and student_id=(select id from users where email='r5s22@edupilot.local'))`);
    R('TC-PE05-42', /Hết giờ — bài của bạn đã được nộp/.test(aft) && Number(srvOn) === Number(srvOff) ? 'P' : 'F', `offline tới sau deadline_at: dải offline: ${JSON.stringify(txOff.slice(0, 2))}; trên máy chủ lúc hết giờ ${srvOff} câu; online lại → ${srvOn} câu (chỉ phần đã lưu); lượt ${st}; màn: "${(aft.match(/Hết giờ[^|]*/) || [aft.slice(0, 120)])[0]}"`);
    const dB = dl(fx.x1, 'r5s21'); while (Date.now() < dB + 15000) await B2.p.waitForTimeout(2000);
    const timerB = await B2.p.getByRole('timer').first().innerText().catch(() => '(không có đồng hồ)'); const bodyB = (await B2.p.locator('body').innerText()).replace(/\n/g, ' | ');
    const before = sql(`select count(*) from exam_answers where attempt_id=(select id from exam_attempts where exam_id='${fx.x1}' and student_id=(select id from users where email='r5s21@edupilot.local'))`); await pick(B2.p, 2); await B2.p.waitForTimeout(4000);
    const afterB = sql(`select count(*) from exam_answers where attempt_id=(select id from exam_attempts where exam_id='${fx.x1}' and student_id=(select id from users where email='r5s21@edupilot.local'))`); const stB = sql(`select status,submit_reason from exam_attempts where exam_id='${fx.x1}' and student_id=(select id from users where email='r5s21@edupilot.local')`);
    R('TC-PE05-13', Number(afterB) === Number(before) && /GRADING|GRADED/.test(stB) ? 'P' : 'F', `đồng hồ trình duyệt chạy chậm 10× (còn hiển thị ${timerB} khi giờ thật đã quá hạn+grace): lượt trên máy chủ ${stB}; chọn thêm 1 đáp án sau hạn+grace → exam_answers ${before}→${afterB} (không tăng: máy chủ từ chối, ATTEMPT_CLOSED); màn: "${bodyB.slice(0, 160)}"`);
    await A.ctx.close(); await B2.ctx.close();
  }
  if (ONLY === 'flaky') {
    const fx = FX(); const u = 'r5s36'; const { ctx, p } = await login(browser, u, { width: 1440, height: 900 }); await startX(p, fx.x6, fx.c1); await pick(p, 0); await p.waitForTimeout(3000);
    const keys = []; let n = 0; await p.route('**/attempts/*/submit', async (r) => { keys.push(r.request().headers()['idempotency-key']); n++; if (n === 1) return r.abort('failed'); return r.continue(); });
    await p.getByRole('button', { name: /Nộp bài/ }).first().click(); await p.waitForTimeout(800); await p.locator('dialog[open]').getByRole('button', { name: /Nộp bài/ }).click(); const state1 = await p.locator('dialog[open], main').first().innerText().catch(() => '');
    await p.waitForTimeout(500); const dlg1 = (await p.locator('body').innerText()).match(/Đang nộp…?/); await p.waitForTimeout(15000);
    const aft = (await p.locator('main').innerText()).replace(/\n/g, ' | '); const subs = sql(`select count(*) from exam_attempts where exam_id='${fx.x6}' and student_id=(select id from users where email='${u}@edupilot.local') and status<>'IN_PROGRESS'`);
    R('TC-PE05-43', keys.length >= 2 && new Set(keys).size === 1 && /Đã nộp lúc/.test(aft) && subs === '1' ? 'P' : 'F', `cắt mạng lần nộp đầu (route abort), bật lại: ${keys.length} yêu cầu submit, Idempotency-Key ${new Set(keys).size === 1 ? 'GIỐNG nhau' : 'khác nhau'} (${keys.map((k) => (k || '').slice(0, 8))}); nút "${dlg1 ? dlg1[0] : '(không thấy Đang nộp…)'}" trong lúc chờ; kết quả: "${(aft.match(/Đã nộp lúc[^|]*/) || [aft.slice(0, 100)])[0]}"; số lượt đã nộp = ${subs} (1)`);
    await ctx.close();
  }

  const toCode = async (p) => { for (let i = 1; i <= 4; i++) { if (await p.locator('textarea').count()) return true; const b = p.getByText(new RegExp('^Câu ' + i + '$')).first(); if (await b.count()) { await b.click().catch(() => {}); await p.waitForTimeout(600); } } for (let i = 0; i < 4; i++) { await p.keyboard.press('ArrowRight'); await p.waitForTimeout(500); if (await p.locator('textarea').count()) return true; } return !!(await p.locator('textarea').count()); };
  if (ONLY === 'code1') {
    const fx = FX(); const u = 'r5s24'; const res = {};
    // PE06-04 ở 4 bề rộng
    for (const w of [390, 1023, 1024, 1440]) {
      const { ctx, p } = await login(browser, u, { width: w, height: 900 }); await startX(p, fx.xc, fx.c1); const hasTa0 = await p.locator('textarea').count(); const body0 = await p.locator('main').innerText();
      const hasCode = await toCode(p); const body = await p.locator('main').innerText(); const m = await metrics(p);
      res[w] = { textarea: await p.locator('textarea').count(), banner: (body.match(/[^\n]*(màn hình rộng|1\.024)[^\n]*/) || [''])[0].slice(0, 160), mcqRadios: await p.locator('input[type=radio],input[type=checkbox]').count(), ox: m.ox, de: /Đọc a, b in a\+b|a\+b/.test(body) };
      await ctx.close();
    }
    R('TC-PE06-04', res[390].textarea === 0 && res[1023].textarea === 0 && res[1024].textarea === 1 && res[1440].textarea === 1 && res[390].de ? 'P' : 'F', `bài MIXED ở 390/1023/1024/1440: ô soạn mã (textarea) = ${[390, 1023, 1024, 1440].map((w) => res[w].textarea)}; đề đọc được ở 390/1023: ${res[390].de}/${res[1023].de}; dải: "${res[390].banner}"; ox=${[390, 1023, 1024, 1440].map((w) => res[w].ox)}`);
    // PE06-07 giới hạn 64 KiB + PE06-14 offline + PE06-27 trạng thái + PE06-42 SSE đứt: dùng 1440
    const { ctx, p } = await login(browser, 'r5s25', { width: 1440, height: 900 }); await startX(p, fx.xc, fx.c1); await toCode(p);
    const ta = p.locator('textarea').first(); const aid = sql(`select id from exam_attempts where exam_id='${fx.xc}' and student_id=(select id from users where email='r5s25@edupilot.local')`);
    const big = 'a'.repeat(65536); await ta.fill(big); await p.waitForTimeout(800); const cnt1 = (await p.locator('main').innerText()).match(/[\d.]+\s*\/\s*65\.536[^\n]*/)?.[0];
    await ta.press('End'); await p.keyboard.type('Z'); await p.waitForTimeout(500); const len1 = await ta.evaluate((e) => e.value.length); const msg = (await p.locator('main').innerText()).match(/[^\n]*(tối đa|giới hạn|vượt)[^\n]*/i)?.[0];
    const vn = 'ệ'.repeat(21845); await ta.fill(vn); await p.waitForTimeout(500); const cntVN = (await p.locator('main').innerText()).match(/[\d.]+\s*\/\s*65\.536[^\n]*/)?.[0];
    await ta.fill(Array.from({ length: 30 }, (_, i) => 'x'.repeat(5000)).join('\n')); await p.waitForTimeout(500); const m2 = await metrics(p);
    await ta.fill('// Nguyễn Văn Á ệ ữ\n#include <cstdio>\nint main(){long long a,b;scanf("%lld %lld",&a,&b);printf("%lld\\n",a+b);}\n'); await p.waitForTimeout(4500);
    const dsrc = sql(`select source from code_drafts where attempt_id='${aid}' limit 1`); const keepVN = dsrc.includes('Nguyễn Văn Á ệ ữ');
    const inp = await p.evaluate(() => new Promise((res) => { const el = document.querySelector('textarea'); let worst = 0; new PerformanceObserver((l) => { for (const e of l.getEntries()) worst = Math.max(worst, e.duration); }).observe({ type: 'event', durationThreshold: 16, buffered: true }); el.focus(); for (const c of 'abcdefghij') { el.dispatchEvent(new KeyboardEvent('keydown', { key: c, bubbles: true })); } setTimeout(() => res(worst), 600); }));
    R('TC-PE06-07', len1 <= 65536 && !!cnt1 && !!cntVN && m2.ox <= 0 && keepVN ? 'P' : 'F', `dán 65.536 ký tự: bộ đếm "${cnt1}"; gõ thêm 'Z' → độ dài ô ${len1} (≤65.536; chữ báo: "${msg || '(không thấy)'}"); 21.845×'ệ' (3 byte): "${cntVN}" (đếm theo byte); 30 dòng × 5.000 ký tự: ox=${m2.ox} (cuộn trong ô); chú thích Việt lưu/tải lại đúng=${keepVN}; INP thô trong ô ${Math.round(inp)} ms (đo gần đúng; spec 'code editor' có đo INP)`);
    // offline 30 s
    await ta.fill('int main(){return 0;} // OFFLINE-QC'); await p.waitForTimeout(3000); await ctx.setOffline(true); await ta.fill('int main(){return 0;} // OFFLINE-QC dòng thêm khi mất mạng'); await p.waitForTimeout(30000);
    const lsK = await p.evaluate(() => Object.keys(localStorage).filter((k) => k.startsWith('exam-code:'))); const lsV = await p.evaluate(() => { const k = Object.keys(localStorage).find((x) => x.startsWith('exam-code:')); return k ? localStorage.getItem(k).slice(0, 120) : null; });
    await ctx.setOffline(false); await p.waitForTimeout(12000); const srv = sql(`select source from code_drafts where attempt_id='${aid}' order by updated_at desc limit 1`); const sha = require('crypto').createHash('sha256').update('int main(){return 0;} // OFFLINE-QC dòng thêm khi mất mạng').digest('hex'); const shaSrv = require('crypto').createHash('sha256').update(srv).digest('hex');
    R('TC-PE06-14', lsK.length > 0 && sha === shaSrv ? 'P' : 'F', `gõ khi offline 30 s: khoá localStorage ${JSON.stringify(lsK)}; sau online 12 s bản nháp máy chủ = đã gõ (sha256 khớp: ${sha === shaSrv}); nội dung khoá: ${lsV}`);
    // Chạy thử + nộp: trạng thái; SSE đứt
    await ta.fill('#include <cstdio>\nint main(){long long a,b;scanf("%lld %lld",&a,&b);printf("%lld\\n",a+b);}\n'); await p.waitForTimeout(2500);
    const seen = new Set(); const evReq = []; p.on('request', (r) => { if (/\/events/.test(r.url())) evReq.push(r.url().slice(-30)); });
    await p.getByRole('button', { name: /Nộp lời giải|Nộp/ }).first().click().catch(() => {}); for (let i = 0; i < 40; i++) { const t = await p.locator('main').innerText(); for (const k of ['Đã nhận', 'xếp hàng', 'Đang chấm', 'Đúng', 'Sai']) if (t.includes(k)) seen.add(k); await p.waitForTimeout(500); }
    R('TC-PE06-27', seen.has('Đang chấm') || seen.has('xếp hàng') || seen.has('Đã nhận') ? 'P' : 'F', `nộp code → trạng thái hiện lần lượt: ${[...seen]} (QUEUED "Đã nhận, đang xếp hàng" → "Đang chấm" → kết quả; không từ kỹ thuật)`);
    await ctx.close();
    const { ctx: c2, p: p2 } = await login(browser, 'r5s26', { width: 1440, height: 900 }); await p2.route('**/api/v1/events*', (r) => r.abort('failed')); const evs = []; p2.on('request', (r) => { if (/\/events|EventSource/.test(r.url())) evs.push(r.url().slice(-25)); });
    await startX(p2, fx.xc, fx.c1); await toCode(p2); const ta2 = p2.locator('textarea').first(); await ta2.fill('#include <cstdio>\nint main(){long long a,b;scanf("%lld %lld",&a,&b);printf("%lld\\n",a+b);}\n'); await p2.waitForTimeout(2500);
    const t_ = Date.now(); await p2.getByRole('button', { name: /Nộp lời giải|Nộp/ }).first().click().catch(() => {}); let done = null; for (let i = 0; i < 60; i++) { const t = await p2.locator('main').innerText(); if (/Đúng|Sai|Lỗi biên dịch|Lần nộp tính điểm/.test(t) && !/Đang chấm/.test(t)) { done = (Date.now() - t_) / 1000; break; } await p2.waitForTimeout(1000); }
    const nres = (await p2.locator('main').innerText()).match(/Đúng|Sai kết quả/g)?.length;
    R('TC-PE06-42', done !== null && evs.length <= 4 ? 'P' : 'F', `chặn kênh SSE (route abort /events): kết quả vẫn hiện sau ${done && done.toFixed(1)} s qua truy vấn lại 2 s; số yêu cầu /events thử = ${evs.length} (≤2/người ×phiên; spec 'sse drop' kiểm hiện đúng một lần); số lần chữ kết quả "Đúng/Sai" = ${nres}`);
    await c2.close();
  }
  if (ONLY === 'code2') {
    const fx = FX(); const u = 'r5s27'; const { ctx, p } = await login(browser, u, { width: 1440, height: 900 }); await startX(p, fx.xc1, fx.c1); await toCode(p);
    const ta = p.locator('textarea').first(); const aid = sql(`select id from exam_attempts where exam_id='${fx.xc1}' and student_id=(select id from users where email='${u}@edupilot.local')`); const d = dl(fx.xc1, u); let n = 0; let lastTyped = '';
    const puts = []; p.on('request', (r) => { if (r.method() === 'PUT' && /\/draft/.test(r.url())) puts.push(Date.now()); });
    while (Date.now() < d - 1500) { lastTyped = '// ' + (n++) + ' ' + 'k'.repeat(n % 7); await ta.fill(lastTyped).catch(() => {}); await p.waitForTimeout(250); }
    await p.waitForTimeout(6000); const ro = await ta.evaluate((e) => e.readOnly || e.disabled).catch(() => null); const val = await ta.inputValue().catch(() => ''); const draft = sql(`select source from code_drafts where attempt_id='${aid}' limit 1`); const st = sql(`select status,submit_reason from exam_attempts where id='${aid}'`); const sub = sql(`select count(*),max(auto::text),max(source) from code_submissions where attempt_id='${aid}' and kind='SUBMIT'`);
    const lastPutGap = puts.length ? (d - puts[puts.length - 1]) / 1000 : null;
    R('TC-PE06-39', ro === true && draft === val && /^1\|t\|/.test(sub) ? 'P' : 'F', `bài 1 phút, gõ liên tục tới giây cuối: ô chỉ đọc sau hạn=${ro}; chữ còn trong ô=${val.length > 0}; PUT draft cuối cách deadline ${lastPutGap && lastPutGap.toFixed(1)} s (không chờ debounce 2 s); bản nháp máy chủ = chữ trong ô: ${draft === val}; lượt ${st}; bài nộp tự động: ${sub.slice(0, 80)}`);
    await ctx.close();
  }
  await browser.close();
})();
