// QC2 smoke sprint 5 — kiểm giao diện thật (ep-r, https://localhost:1143). Chạy: node ui_pe.cjs  (NODE_TLS_REJECT_UNAUTHORIZED=0)
const { chromium } = require('/Users/kuro/Documents/TA_Agent_qc2/frontend/node_modules/@playwright/test');
const AxeBuilder = require('/Users/kuro/Documents/TA_Agent_qc2/frontend/node_modules/@axe-core/playwright').default || require('/Users/kuro/Documents/TA_Agent_qc2/frontend/node_modules/@axe-core/playwright');
const fs = require('fs'); const { execFileSync } = require('child_process');
const B = 'https://localhost:1143'; const PW = 'Edupilot#Seed-2026'; const ST = JSON.parse(fs.readFileSync('/tmp/qc2-c5/ui.json')); const SH = '/Users/kuro/Documents/TA_Agent_qc2/docs/regression/evidence-5/ui/shots'; fs.mkdirSync(SH, { recursive: true });
const out = []; const rec = (id, ok, note) => { out.push({ id, ok, note }); console.log((ok ? 'PASS ' : 'FAIL ') + id + ' ' + note); };
const sql = (q) => execFileSync('docker', ['exec', 'ep-r-postgres-1', 'psql', '-U', 'edupilot', '-d', 'edupilot', '-At', '-c', q], { encoding: 'utf8' }).trim();
async function login(browser, user, vp) {
  const ctx = await browser.newContext({ ignoreHTTPSErrors: true, viewport: vp, locale: 'vi-VN', timezoneId: 'Asia/Ho_Chi_Minh', hasTouch: vp.width < 600 }); const p = await ctx.newPage(); const errs = []; p.on('pageerror', (e) => errs.push(String(e).slice(0, 120))); p.on('console', (m) => { if (m.type() === 'error') errs.push(m.text().slice(0, 120)); });
  await p.goto(B + '/login'); await p.getByLabel(/email/i).fill(user + '@edupilot.local'); await p.getByLabel(/mật khẩu/i).first().fill(PW); await p.getByRole('button', { name: /đăng nhập/i }).click(); await p.waitForURL((u) => !u.pathname.startsWith('/login'), { timeout: 15000 }); await p.waitForTimeout(1500); await dismiss(p); return { ctx, p, errs };
}
async function sw(p) { const cur = p.getByText(/^R5C2$/).first(); if (await cur.count()) { await cur.click(); await p.waitForTimeout(800); await p.getByText(/R5C1/).first().click(); await p.waitForTimeout(2500); } }
async function dismiss(p) { for (let i = 0; i < 3; i++) { const d = p.locator('dialog[open]'); if (!(await d.count())) return; console.log('dialog:', (await d.first().innerText()).slice(0, 80).replace(/\n/g, ' | ')); const b = d.first().locator('button').last(); await b.click({ timeout: 3000 }).catch(() => {}); await p.waitForTimeout(800); } }
async function goCode(p) { for (let i = 1; i <= 6; i++) { if (await p.locator('textarea').count()) return true; const t = (await p.locator('body').innerText()); if (/màn hình/.test(t) && /lập trình|mã nguồn/i.test(t) && i > 1) return false; const b = p.getByText(new RegExp('^Câu ' + i + '$')).first(); if (await b.count()) { await b.click().catch(() => {}); await p.waitForTimeout(700); } } return !!(await p.locator('textarea').count()); }
async function metrics(p) {
  return await p.evaluate(() => ({ ox: document.documentElement.scrollWidth - window.innerWidth, h1: [...document.querySelectorAll('h1')].map((h) => h.innerText.trim()).slice(0, 2), primary: [...document.querySelectorAll('[data-variant=primary]')].filter((e) => e.offsetParent).map((e) => e.innerText.trim()), small: [...document.querySelectorAll('button,a,[role=button],input,select,textarea')].filter((e) => e.offsetParent && e.getBoundingClientRect().width > 0 && (e.getBoundingClientRect().height < 40 || e.getBoundingClientRect().width < 40)).length }));
}
async function axe(p) { const r = await new AxeBuilder({ page: p }).withTags(['wcag2a', 'wcag2aa', 'wcag21aa', 'wcag22aa']).analyze(); for (const v of r.violations.filter((v) => ['serious', 'critical'].includes(v.impact))) console.log('  axe', v.id, v.nodes.map((n) => n.html.slice(0, 140)).join(' || ')); return r.violations.filter((v) => ['serious', 'critical'].includes(v.impact)).map((v) => v.id + ':' + v.nodes.length); }
(async () => { process.on('unhandledRejection', (e) => { console.log('ERR', String(e).slice(0, 300)); });
  const browser = await chromium.launch({ headless: true });
  const bg = process.env.ONLY || 'all';
  // ===== GV 1440 =====
  if (bg === 'all') { const { ctx, p, errs } = await login(browser, 'r5t', { width: 1440, height: 900 });
    await p.goto(B + '/questions'); await p.waitForTimeout(2500); await sw(p); let m = await metrics(p); let ax = await axe(p); await p.screenshot({ path: SH + '/gv-questions-1440.png' });
    rec('TC-PE03-48', m.ox <= 0 && m.primary.length === 1 && /Tạo câu hỏi/.test(m.primary[0]), `/questions 1440: ox=${m.ox} primary=${JSON.stringify(m.primary)} h1=${m.h1} lỗi console=${errs.length}`);
    rec('TC-PE03-52', ax.length === 0, `axe 1440 /questions serious/critical: ${JSON.stringify(ax)}`);
    await p.goto(B + '/exams'); await p.waitForTimeout(2500); await sw(p); m = await metrics(p); ax = await axe(p); const txt = await p.locator('body').innerText(); await p.screenshot({ path: SH + '/gv-exams-1440.png' });
    rec('TC-PE04-42', m.ox <= 0 && m.primary.length === 1 && /Tạo bài thi/.test(m.primary[0]) && /Đang mở|Sắp tới|Nháp|Đã đóng/.test(txt), `/exams 1440 GV: primary=${JSON.stringify(m.primary)} nhóm: ${['Đang mở', 'Sắp tới', 'Đã đóng', 'Nháp'].filter((g) => txt.includes(g))}; axe ${JSON.stringify(ax)}`);
    await p.getByText('R5 UI bài thi').first().click(); await p.waitForTimeout(3500); m = await metrics(p); const t2 = await p.locator('body').innerText(); ax = await axe(p); await p.screenshot({ path: SH + '/gv-exam-detail-1440.png' });
    rec('TC-PE04-43', ['Thông tin', 'Câu hỏi', 'Xem trước'].every((s) => t2.includes(s)) && m.ox <= 0, `/exams/[id] GV: ${['Thông tin', 'Câu hỏi', 'Xem trước', 'Gia hạn', 'Lên lịch', 'Xem kết quả'].filter((s) => t2.includes(s))}; axe ${JSON.stringify(ax)}`);
    await ctx.close(); }
  // ===== TA =====
  if (bg === 'all') { const { ctx, p } = await login(browser, 'r5ta', { width: 1440, height: 900 });
    const draft = sql(`select id from exams where course_id='${ST.c1}' and status='DRAFT' order by created_at desc limit 1`);
    await p.goto(B + '/exams'); await p.waitForTimeout(2500); await sw(p); await p.getByText(/R5 04|R5 05|R5 06|R5 UI|bản sao/).first().click(); await p.waitForTimeout(3500); const t = await p.locator('body').innerText(); await p.screenshot({ path: SH + '/ta-exam-draft-1440.png' });
    rec('TC-PE04-44', !(await p.getByRole('button', { name: /^Lên lịch/ }).count()) && /Chỉ giảng viên/.test(t), `TA /exams/[nháp]: nút "Lên lịch" ẩn=${!(await p.getByRole('button', { name: /^Lên lịch/ }).count())}; chữ "Chỉ giảng viên lên lịch được": ${/Chỉ giảng viên/.test(t)}`);
    await ctx.close(); }
  // ===== SV 390: danh sách + làm bài =====
  if (bg !== 'sv1440') try { const { ctx, p, errs } = await login(browser, 'r5s11', { width: 390, height: 844 });
    await p.goto(B + '/exams'); await p.waitForTimeout(2500); let m = await metrics(p); let ax = await axe(p); const t = await p.locator('body').innerText(); await p.screenshot({ path: SH + '/sv-exams-390.png' });
    rec('TC-PE04-47', m.ox <= 0 && /Đang mở|Sắp tới|Đã có điểm/.test(t), `/exams 390 SV: ox=${m.ox}; vùng chạm <40px: ${m.small}; nhóm: ${['Đang mở', 'Sắp tới', 'Đã có điểm'].filter((g) => t.includes(g))}; axe ${JSON.stringify(ax)}`);
    await p.locator('li', { hasText: 'R5 UI bài thi' }).locator('a').first().click(); await p.waitForTimeout(3500); const t0 = await p.locator('body').innerText(); await p.screenshot({ path: SH + '/sv-take-start-390.png' });
    rec('TC-PE06-05', /1024/.test(t0), `màn trước khi bắt đầu (bài MIXED) 390: có dòng "cần màn hình ≥ 1024 px": ${/1024/.test(t0)}; chữ: ${(t0.match(/[^\n]*1024[^\n]*/) || [''])[0].slice(0, 120)}`);
    const start = p.getByRole('button', { name: /bắt đầu/i }).first(); if (await start.count()) { await start.click(); await p.waitForTimeout(3000); }
    m = await metrics(p); ax = await axe(p); await p.screenshot({ path: SH + '/sv-take-390.png' }); await goCode(p); const t1 = await p.locator('body').innerText(); await p.screenshot({ path: SH + '/sv-take-code-390.png' });
    const radios = await p.locator('input[type=radio],input[type=checkbox],[role=radio],[role=checkbox]').count();
    rec('TC-PE05-45', m.ox <= 0 && radios > 0, `/take 390: ox=${m.ox}; radio/checkbox=${radios}; vùng chạm<40px=${m.small}; axe ${JSON.stringify(ax)}; lỗi console=${errs.length}`);
    rec('TC-PE05-47', !/RAG|PII|provider|sandbox|verdict|answer_key/i.test(t1) && !/QCR5CANARY/.test(t1), `chữ làm bài không có từ kỹ thuật/canary: ${!/RAG|PII|provider|sandbox|verdict/i.test(t1)}`);
    rec('TC-PE06-04', /lập trình/i.test(t1) && !(await p.locator('textarea').count()) , `390: đề code đọc được, không textarea (dải "cần màn hình"): textarea=${await p.locator('textarea').count()}; dải: ${/màn hình/.test(t1)}`);
    // offline 30s: chọn đáp án, tắt mạng 30s, chọn thêm, bật mạng
    const aid = sql(`select id from exam_attempts where exam_id='${ST.eid}' and student_id=(select id from users where email='r5s11@edupilot.local')`) || '00000000-0000-0000-0000-000000000000';
    const opts = p.locator('input[type=radio],input[type=checkbox],[role=radio],[role=checkbox]'); if ((await opts.count()) > 0) await opts.nth(0).click({ force: true }).catch(() => {}); await p.waitForTimeout(3500);
    const n1 = sql(`select count(*) from exam_answers where attempt_id='${aid}'`);
    await ctx.setOffline(true); if ((await opts.count()) > 2) await opts.nth(2).click({ force: true }).catch(() => {}); await p.waitForTimeout(30000); const toff = await p.locator('body').innerText(); await p.screenshot({ path: SH + '/sv-take-offline-390.png' });
    await ctx.setOffline(false); await p.waitForTimeout(9000); const n2 = sql(`select count(*) from exam_answers where attempt_id='${aid}'`); await p.screenshot({ path: SH + '/sv-take-online-390.png' });
    rec('TC-PE05-40', /Mất mạng|offline|Chưa lưu/i.test(toff) && Number(n2) >= Number(n1), `offline 30 s: dải "${(toff.match(/[^\n]*(Mất mạng|Chưa lưu)[^\n]*/) || ['(không thấy)'])[0].slice(0, 100)}"; đáp án trên máy chủ trước ${n1} → sau online ${n2}`);
    await ctx.close(); } catch (e) { console.log('ERR SV390', String(e).slice(0, 200)); }
  // ===== SV 1440: làm bài code =====
  try { const { ctx, p } = await login(browser, 'r5s12', { width: 1440, height: 900 });
    await p.goto(B + '/exams'); await p.waitForTimeout(2500); await p.locator('li', { hasText: 'R5 UI bài thi' }).locator('a').first().click(); await p.waitForTimeout(3500); const s = p.getByRole('button', { name: /bắt đầu/i }).first(); if (await s.count()) { await s.click(); await p.waitForTimeout(3000); }
    { const tk = p.getByRole('button', { name: /Làm tiếp ở đây/ }); if (await tk.count()) { await tk.first().click(); await p.waitForTimeout(1500); } } await goCode(p); const ta = p.locator('textarea').first(); const has = await ta.count();
    if (has) { const at = await ta.evaluate((e) => ({ sp: e.getAttribute('spellcheck'), ac: e.getAttribute('autocapitalize'), al: e.getAttribute('aria-label') })); rec('TC-PE06-06', at.sp === 'false' && at.ac === 'off' && /Mã nguồn/.test(at.al || ''), `textarea: ${JSON.stringify(at)}`); await ta.click(); await ta.press('Tab'); const v = await ta.inputValue(); rec('TC-PE06-06b', v.includes('    '), `Tab chèn 4 dấu cách: ${JSON.stringify(v.slice(0, 12))}`); }
    else rec('TC-PE06-06', false, 'không thấy textarea mã nguồn ở 1440');
    const aid = sql(`select id from exam_attempts where exam_id='${ST.eid}' and student_id=(select id from users where email='r5s12@edupilot.local')`) || '00000000-0000-0000-0000-000000000000';
    if (has) { await ta.fill('#include <cstdio>\nint main(){long long a,b;scanf("%lld %lld",&a,&b);printf("%lld\\n",a+b);}\n'); await p.waitForTimeout(4000); }
    const dr = sql(`select count(*),max(length(source)) from code_drafts where attempt_id='${aid}'`); const run = p.getByRole('button', { name: /chạy thử/i }).first();
    if (await run.count()) { await run.click(); await p.waitForTimeout(9000); } const tr = await p.locator('body').innerText(); await p.screenshot({ path: SH + '/sv-code-1440.png' });
    const m = await metrics(p); const ax = await axe(p);
    rec('TC-PE06-21', /Đúng|Sai kết quả|Lỗi biên dịch/.test(tr), `Chạy thử: nhãn kết quả hiện: ${(tr.match(/(Đúng|Sai kết quả|Quá thời gian|Lỗi biên dịch|Lỗi khi chạy)/g) || []).slice(0, 4)}; nháp trên máy chủ ${dr}`);
    rec('TC-PE06-08', ax.length === 0 && m.ox <= 0, `axe 1440 làm bài code: ${JSON.stringify(ax)}; ox=${m.ox}`);
    await ctx.close(); } catch (e) { console.log('ERR SV1440', String(e).slice(0, 200)); }
  await browser.close(); fs.writeFileSync('/tmp/qc2-c5/ui-results.json', JSON.stringify(out, null, 1));
})();
