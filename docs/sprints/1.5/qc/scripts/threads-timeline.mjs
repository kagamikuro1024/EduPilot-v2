// QC — đo TỪNG BƯỚC THỜI GIAN của Threads như thật (SRS 4.3.1 E, F, G, H; US 01-AC4, AC12, AC13, AC14; 02-AC11).
// Hộp đen: chỉ dùng chữ hiển thị và URL; không biết cấu trúc DOM. Dùng Puppeteer `page` của global `browser` (như sweep.mjs).
//   const m = await import('/abs/docs/sprints/1.5/qc/scripts/threads-timeline.mjs');
//   const rows = await m.default(browser, { base: 'http://localhost:3000', out: '/abs/.../shots/threads' });
//   console.log(m.table(rows));
// Chạy trên bản đã build/đang chạy khi dev báo xong Threads; KHÔNG chạy khi dev đang thi công. Mỗi kịch bản bắt đầu bằng xoá `ep_demo_state`.
// Dung sai: bước 1 = 1,2 s ±0,4; bước 2 = 2,5–3,5 s ±0,4; bước 3 = 0,3 s (0,0–0,8); phản hồi trễ: dòng "đang trả lời" ở 2 s (1,5–2,8), phản hồi ở 6 s (5,4–7,0). Lấy mẫu mỗi ~40 ms nên sai số đo ≈ ±0,1 s.
// Ngưỡng ở các hằng TOL dưới đây — nếu spec đổi số thì sửa ở đó, không sửa chỗ khác.
export const TOL = {
  compose: [0.8, 1.6], stream: [2.1, 3.9], source: [0, 0.8], label: [0, 0.6], navMax: 0.8, instantMax: 0.8,
  typingAt: [1.5, 2.8], lateAt: [5.4, 7.0],
};
const AI_NOMATCH = 'Mình chưa đủ chắc chắn để gợi ý câu này từ tài liệu của lớp. Mình đã báo giảng viên và trợ giảng; câu trả lời sẽ hiện ngay trong thread này.';
const H1_TA = 'Em xem bảng so sánh SHA-256 với bcrypt ở trang 9 rồi trả lời câu AI hỏi nhé.';

export default async function timeline(browser, { base = 'http://localhost:3000', out, only } = {}) {
  const tab = await browser.open({ name: 'qc-threads', url: base + '/login', viewport: { width: 1440, height: 900 } });
  const rows = await tab.run(async ({ page }, a) => {
    const { base, out, only, TOL, AI_NOMATCH, H1_TA, FALLBACK_TA_SRC } = a;
    const FALLBACK_TA = new Function('w', 'return `' + FALLBACK_TA_SRC + '`');
    const fs = await import('node:fs'); if (out) fs.mkdirSync(out, { recursive: true });
    const rows = [];
    const sleep = (ms) => new Promise((s) => setTimeout(s, ms));
    const push = (id, ok, detail, file) => rows.push({ id, ok: !!ok, detail: detail || '', file: file || '' });
    const shot = async (n) => { if (!out) return ''; const f = `${out}/${n}.png`; await page.screenshot({ path: f }); return f; };
    const setRole = async (role, person = 'sv-2') => { await page.deleteCookie(...(await page.cookies())); for (const [n, v] of [['ep_demo_role', role], ['ep_demo_person', role === 'student' ? person : ''], ['ep_demo_course', 'int1006-1']]) await page.setCookie({ name: n, value: v, url: base }); };
    const go = async (u) => { await page.goto(base + u, { waitUntil: 'networkidle0', timeout: 30000 }); await sleep(300); };
    const fresh = async (role = 'student', person = 'sv-2', url = '/threads') => { await setRole(role, person); await go('/login'); await page.evaluate(() => { try { localStorage.removeItem('ep_demo_state'); } catch {} }); await go(url); };
    const text = () => page.evaluate(() => document.body.innerText);
    const click = async (re, sel = 'button,a,[role=button],[role=menuitem],summary,label,li', wait = 200) => { const ok = await page.evaluate((src, fl, s) => { const r = new RegExp(src, fl); const el = [...document.querySelectorAll(s)].find((e) => r.test((e.innerText || e.getAttribute('aria-label') || '').trim()) && e.getBoundingClientRect().width > 0); if (el) { el.click(); return true; } return false; }, re.source, re.flags, sel); await sleep(wait); return ok; };
    const fill = async (labelRe, value) => {
      const ok = await page.evaluate((src, fl, v) => {
        const r = new RegExp(src, fl); let c = null;
        for (const l of document.querySelectorAll('label')) if (r.test(l.innerText)) { c = (l.htmlFor && document.getElementById(l.htmlFor)) || l.querySelector('input,textarea,select') || l.parentElement.querySelector('input,textarea,select'); if (c) break; }
        if (!c) c = [...document.querySelectorAll('input,textarea,select')].find((e) => r.test(e.getAttribute('aria-label') || '') || r.test(e.placeholder || ''));
        if (!c) return false;
        const proto = c.tagName === 'SELECT' ? HTMLSelectElement.prototype : c.tagName === 'TEXTAREA' ? HTMLTextAreaElement.prototype : HTMLInputElement.prototype;
        if (c.tagName === 'SELECT') { const o = [...c.options].find((x) => x.text.trim() === v); if (!o) return false; Object.getOwnPropertyDescriptor(proto, 'value').set.call(c, o.value); } else Object.getOwnPropertyDescriptor(proto, 'value').set.call(c, v);
        c.dispatchEvent(new Event('input', { bubbles: true })); c.dispatchEvent(new Event('change', { bubbles: true })); return true;
      }, labelRe.source, labelRe.flags, value);
      return ok;
    };
    const typeReply = async (v) => { const tas = await page.$$('textarea'); const ta = tas[tas.length - 1]; await ta.click({ clickCount: 3 }); await page.keyboard.press('Backspace'); await ta.type(v); };
    // lấy mẫu mỗi ~40 ms trong maxMs; trả về mốc đầu tiên mỗi sự kiện (giây từ t0) và chuỗi mẫu
    const sample = async (maxMs, t0, stopWhen) => {
      const ev = {}; const trace = []; let lastDung = false;
      const end = Date.now() + maxMs;
      while (Date.now() < end) {
        const s = await page.evaluate(() => {
          const t = document.body.innerText; const btns = [...document.querySelectorAll('button')].filter((b) => b.getBoundingClientRect().width > 0);
          const dung = btns.some((b) => /^Dừng$/.test(b.innerText.trim()));
          const lock = btns.filter((b) => /Hỏi trợ lý AI|Gửi phản hồi/.test(b.innerText)).map((b) => b.disabled || b.getAttribute('aria-disabled') === 'true');
          return { url: location.pathname, composing: /Trợ lý AI đang soạn/.test(t), dung, stopped: /Đã dừng/.test(t), askAgain: btns.some((b) => /^Hỏi lại$/.test(b.innerText.trim())), src: /Nguồn tham khảo \(\d+\)/.test(t), pending: /Chờ xác nhận/.test(t), waitTeacher: /Đang chờ giảng viên/.test(t), typing: /Phạm Quốc Bảo đang trả lời/.test(t), late: /Em xem bảng so sánh SHA-256 với bcrypt|Cảm ơn em, anh đã ghi nhận/.test(t), len: t.length, lock };
        });
        const now = (Date.now() - t0) / 1000; const mark = (k, c) => { if (c && ev[k] === undefined) ev[k] = now; };
        mark('url', /^\/threads\/[^/]+$/.test(s.url)); mark('compose', s.composing); mark('dung', s.dung); if (s.dung) lastDung = true; mark('dungEnd', lastDung && !s.dung);
        mark('src', s.src); mark('pending', s.pending); mark('waitTeacher', s.waitTeacher); mark('typing', s.typing); mark('late', s.late); mark('stopped', s.stopped); mark('askAgain', s.askAgain);
        if (s.composing || s.dung) { ev.lockedWhileBusy = ev.lockedWhileBusy === undefined ? s.lock.length > 0 && s.lock.every(Boolean) : ev.lockedWhileBusy && s.lock.length > 0 && s.lock.every(Boolean); }
        trace.push([+now.toFixed(2), s.composing ? 'C' : '', s.dung ? 'S' : '', s.len]);
        if (stopWhen && stopWhen(s, ev)) break;
        await sleep(30);
      }
      return { ev, trace };
    };
    const between = (v, [lo, hi]) => v !== undefined && v >= lo && v <= hi;
    const f2 = (v) => (v === undefined ? '—' : v.toFixed(2));
    const newThread = async (title, body, topic, { ai = true } = {}) => {
      await fill(/Tiêu đề/i, title); await fill(/Nội dung/i, body); if (topic) await fill(/Chủ đề/i, topic);
      const cb = await page.evaluate((want) => { const c = document.querySelector('input[type=checkbox]'); if (!c) return null; if (c.checked !== want) c.click(); return c.checked; }, ai);
      await sleep(150); const t0 = Date.now(); await click(/^Đăng câu hỏi$/, 'button', 0); return { t0, cb };
    };
    const want = (id) => !only || only.includes(id);

    // ---- T1: thread mới khớp S2 — đủ 4 bước E, theo thời gian ----
    if (want('T1')) {
      await fresh(); const { t0, cb } = await newThread('Dùng lại IV trong CTR có sao không?', 'Em thấy CTR dùng nonce, nếu dùng lại nonce với cùng khoá thì có sao không ạ?', 'Mật mã đối xứng');
      const { ev } = await sample(12000, t0, (s, e) => e.pending !== undefined && e.src !== undefined);
      const t = await text(); await shot('T1-final');
      push('T1a checkbox "Nhờ AI" mặc định bật', cb === true, `checked=${cb}`);
      push('T1b chuyển ngay sang /threads/<id>', ev.url !== undefined && ev.url <= TOL.navMax, `url@${f2(ev.url)}s`);
      push('T1c bước 1 "Trợ lý AI đang soạn…" xuất hiện ngay', ev.compose !== undefined && ev.compose <= TOL.navMax, `compose@${f2(ev.compose)}s`);
      push('T1d bước 1 kéo dài ~1,2 s (nút Dừng xuất hiện)', between(ev.dung - ev.compose, TOL.compose), `dung−compose=${f2(ev.dung - ev.compose)}s`);
      push('T1e bước 2 chữ chảy 2,5–3,5 s (Dừng biến mất)', between(ev.dungEnd - ev.dung, TOL.stream), `stream=${f2(ev.dungEnd - ev.dung)}s`);
      push('T1f bước 3 Nguồn tham khảo (n) sau bước 2 ≤ 0,8 s', between(ev.src - ev.dungEnd, TOL.source), `src−dungEnd=${f2(ev.src - ev.dungEnd)}s`);
      push('T1g bước 4 nhãn Chờ xác nhận ngay sau nguồn', ev.pending !== undefined && ev.pending - ev.src >= -0.05 && ev.pending - ev.src <= TOL.label[1], `pending−src=${f2(ev.pending - ev.src)}s`);
      push('T1h trong bước 1–2 hai nút Hỏi trợ lý AI / Gửi phản hồi khoá', ev.lockedWhileBusy === true, `locked=${ev.lockedWhileBusy}`);
      push('T1i nội dung S2: nonce + XOR hai bản mã, nguồn Chương 3 tr. 18–20, kết bằng câu hỏi', /XOR hai bản mã/.test(t) && /Chương 3/.test(t) && /(tr\.|trang)\s*18/.test(t) && /\?/.test(t.slice(t.indexOf('XOR hai bản mã'))), '');
      // phía GV (02-AC11 / H)
      await setRole('teacher'); await go('/');
      const tt = await text();
      push('T1j GV Hôm nay: "7 việc cần xử lý hôm nay"', /7 việc cần xử lý hôm nay/.test(tt), (tt.match(/\d+ việc cần xử lý hôm nay/) || [''])[0]);
      push('T1k GV Hôm nay: "Câu hỏi mới: «Dùng lại IV trong CTR có sao không?» · Mật mã đối xứng · vừa xong" + Chờ xác nhận', /Câu hỏi mới:\s*«Dùng lại IV trong CTR có sao không\?»\s*·\s*Mật mã đối xứng\s*·\s*vừa xong/.test(tt) && /Chờ xác nhận/.test(tt), '', await shot('T1-gv-home'));
      await click(/Thông báo/, 'button', 400); const bell = await text();
      push('T1l GV chuông: "Câu hỏi mới trong Threads: «…»"', /Câu hỏi mới trong Threads:\s*«Dùng lại IV trong CTR có sao không\?»/.test(bell), '');
      await page.keyboard.press('Escape');
      await click(/Câu hỏi mới:/, 'a,button,li,[role=button]', 600);
      push('T1m bấm việc → /threads/<id>', /^\/threads\/[^/]+$/.test(await page.evaluate(() => location.pathname)), await page.evaluate(() => location.pathname));
      await click(/^Xác nhận$/, 'button', 500); await go('/');
      const t2 = await text();
      push('T1n Xác nhận → việc rời, về "6 việc"', /6 việc cần xử lý hôm nay/.test(t2) && !/Câu hỏi mới:\s*«Dùng lại IV/.test(t2), (t2.match(/\d+ việc cần xử lý hôm nay/) || [''])[0]);
      await setRole('student'); await go('/');
      push('T1o SV không thấy việc "Câu hỏi mới" của GV', !/Câu hỏi mới:/.test(await text()), '');
    }

    // ---- T2: hai câu hỏi khác nhau → hai câu trả lời khác nhau (S1 vs S2) ----
    if (want('T2')) {
      await fresh(); let t0 = (await newThread('Vì sao ECB làm lộ ảnh?', 'Em chưa hiểu vì sao ảnh mã hoá bằng ECB vẫn nhìn ra hình ạ.', 'Mật mã đối xứng')).t0; await sample(10000, t0, (s, e) => e.pending !== undefined); const a1 = await text();
      await go('/threads'); await fill(/Tiêu đề/i, 'Dùng lại IV trong CTR có sao không?'); await fill(/Nội dung/i, 'Nếu dùng lại nonce thì sao ạ?'); await fill(/Chủ đề/i, 'Mật mã đối xứng'); t0 = Date.now(); await click(/^Đăng câu hỏi$/, 'button', 0); await sample(10000, t0, (s, e) => e.pending !== undefined); const a2 = await text();
      const pick = (t) => (t.match(/Trợ lý AI của lớp[\s\S]*?(?=Nguồn tham khảo)/) || [''])[0].replace(/\s+/g, ' ');
      push('T2a "Vì sao ECB làm lộ ảnh?" → S1 (so sánh ECB/CBC, nguồn tr. 14–17)', /ECB/.test(pick(a1)) && /(tr\.|trang)\s*14/.test(a1), pick(a1).slice(0, 90));
      push('T2b hai thread → hai câu trả lời khác nhau', pick(a1) !== pick(a2) && pick(a1).length > 40 && pick(a2).length > 40, '');
    }

    // ---- T3: Dừng + Hỏi lại ----
    if (want('T3')) {
      await fresh(); const { t0 } = await newThread('Vì sao ECB làm lộ ảnh?', 'Em chưa hiểu ECB lộ ảnh ạ.', 'Mật mã đối xứng');
      await sample(6000, t0, (s) => s.dung); await sleep(900); const clicked = await click(/^Dừng$/, 'button', 250);
      const l1 = await page.evaluate(() => document.body.innerText.length); await sleep(700); const l2 = await page.evaluate(() => document.body.innerText.length);
      const t = await text(); await shot('T3-stopped');
      push('T3a bấm Dừng: có "Đã dừng" + nút Hỏi lại', clicked && /Đã dừng/.test(t) && /Hỏi lại/.test(t), `clicked=${clicked}`);
      push('T3b chữ dừng lại (giữ phần đã hiện, không chảy tiếp, không nhãn Chờ xác nhận giả)', l1 === l2, `len ${l1}→${l2}`);
      await click(/^Hỏi lại$/, 'button', 150); const { ev } = await sample(9000, Date.now(), (s, e) => e.pending !== undefined);
      push('T3c Hỏi lại → soạn lại và chảy tới nhãn Chờ xác nhận', ev.compose !== undefined && ev.pending !== undefined, `compose@${f2(ev.compose)} pending@${f2(ev.pending)}`);
    }

    // ---- T4: nhánh không khớp (WPA3) — toàn văn ngay, không nguồn ----
    if (want('T4')) {
      await fresh(); const { t0 } = await newThread('WPA3 chặn được KRACK không?', 'Em đọc nói WPA3 dùng SAE, vậy có chặn được KRACK không ạ?', 'An toàn mạng không dây');
      const { ev } = await sample(5000, t0, (s, e) => e.waitTeacher !== undefined && Date.now() - t0 > 2500); const t = await text(); await shot('T4-nomatch');
      push('T4a toàn văn hiện ngay (≤ 0,8 s sau khi sang thread), không "đang soạn", không chữ chảy', ev.waitTeacher !== undefined && ev.waitTeacher <= TOL.instantMax + 0.4 && ev.compose === undefined && ev.dung === undefined, `waitTeacher@${f2(ev.waitTeacher)} compose=${ev.compose}`);
      push('T4b đúng toàn văn không khớp', t.replace(/\s+/g, ' ').includes(AI_NOMATCH), '');
      push('T4c không nguồn; nhãn Đang chờ giảng viên; không Chờ xác nhận', !/Nguồn tham khảo/.test(t) && /Đang chờ giảng viên/.test(t) && !/Chờ xác nhận/.test(t), '');
      await setRole('teacher'); await go('/inbox'); const inb = await text();
      push('T4d /inbox GV không thêm ticket (vẫn "Tất cả 6")', /Tất cả\s*6\b/.test(inb), (inb.match(/Tất cả\s*\d+/) || [''])[0]);
      await go('/'); const home = await text();
      push('T4e GV Hôm nay: "7 việc", nhãn "Cần giảng viên trả lời"', /7 việc cần xử lý hôm nay/.test(home) && /Cần giảng viên trả lời/.test(home), '', await shot('T4-gv-home'));
      await click(/Câu hỏi mới:/, 'a,button,li,[role=button]', 600); await typeReply('Em chào, WPA3 dùng SAE nên chặn được KRACK trên bắt tay 4 bước, thầy sẽ giải thích thêm trên lớp.'); await click(/^Gửi phản hồi$/, 'button', 500); await go('/');
      const home2 = await text(); push('T4f GV gửi phản hồi trong thread → việc rời, về "6 việc"', /6 việc cần xử lý hôm nay/.test(home2), (home2.match(/\d+ việc cần xử lý hôm nay/) || [''])[0]);
      await setRole('ta'); await go('/'); const ta = await text(); push('T4g TA thấy giống GV ở Hôm nay (đếm khớp thao tác của GV: 6 việc)', /6 việc cần xử lý hôm nay|6 việc/.test(ta), (ta.match(/\d+ việc[^\n]*/) || [''])[0]);
    }

    // ---- T5–T9: phản hồi trễ + chuông (G) ----
    const sendReply = async (v) => { await typeReply(v); const t0 = Date.now(); await click(/^Gửi phản hồi$/, 'button', 0); return t0; };
    const bellText = async () => { await click(/Thông báo/, 'button,[aria-label*="Thông báo"]', 450); const t = await page.evaluate(() => { const p = [...document.querySelectorAll('[role=dialog],[role=menu],[popover]')].find((e) => /Thông báo/.test(e.innerText)); return p ? p.innerText : ''; }); return t; };
    const taBlocks = () => page.evaluate(() => (document.body.innerText.match(/Phạm Quốc Bảo/g) || []).length);
    if (want('T5')) {
      await fresh('student', 'sv-2', '/threads/t-salt'); const before = await taBlocks();
      const t0 = await sendReply('Em hiểu rồi ạ');
      const imm = await page.evaluate(() => { const t = document.body.innerText; const el = [...document.querySelectorAll('main *')].filter((e) => !e.children.length && /Em hiểu rồi ạ/.test(e.textContent)).pop(); const r = el && el.getBoundingClientRect(); return { shown: /Em hiểu rồi ạ/.test(t), inView: !!r && r.top >= 0 && r.bottom <= innerHeight }; });
      const { ev } = await sample(8500, t0, () => false);
      const t = await text(); await shot('T5-late');
      push('T5a phản hồi hiện ngay cuối vùng thảo luận và cuộn tới', imm.shown && imm.inView, JSON.stringify(imm));
      push('T5b ~2 s sau: "Phạm Quốc Bảo đang trả lời…"', between(ev.typing, TOL.typingAt), `typing@${f2(ev.typing)}s`);
      push('T5c ~6 s sau: phản hồi TA đúng nội dung H1 (5,4–7,0 s), có khối trích phản hồi của B', between(ev.late, TOL.lateAt) && t.includes(H1_TA) && (t.match(/Em hiểu rồi ạ/g) || []).length >= 2, `late@${f2(ev.late)}s`);
      const b = await bellText(); push('T5d chuông: "Phạm Quốc Bảo đã trả lời trong «Vì sao cần muối (salt) khi băm mật khẩu?»"', /Phạm Quốc Bảo đã trả lời trong «Vì sao cần muối \(salt\) khi băm mật khẩu\?»/.test(b), b.replace(/\s+/g, ' ').slice(0, 140), await shot('T5-bell'));
      await click(/Phạm Quốc Bảo đã trả lời/, '[role=dialog] *,[role=menu] *,[popover] *,a,button,li', 700);
      const nav = await page.evaluate((h) => { const el = [...document.querySelectorAll('main *')].filter((e) => !e.children.length && e.textContent.includes(h.slice(0, 30))).pop(); const r = el && el.getBoundingClientRect(); return { url: location.pathname, inView: !!r && r.top >= 0 && r.bottom <= innerHeight }; }, H1_TA);
      push('T5e bấm thông báo → /threads/t-salt, cuộn tới đúng phản hồi', nav.url === '/threads/t-salt' && nav.inView, JSON.stringify(nav));
      await go('/threads'); const row = await page.evaluate(() => [...document.querySelectorAll('a[href="/threads/t-salt"]')].map((a) => a.innerText.replace(/\s+/g, ' ')).join(' | '));
      push('T5f /threads: t-salt "5 phản hồi"', /5 phản hồi/.test(row), row.slice(0, 120));
      // T8: phản hồi thứ hai → không còn phản hồi trễ
      await go('/threads/t-salt'); const n1 = await taBlocks(); await sendReply('Cảm ơn anh Bảo ạ'); await sleep(8500); const n2 = await taBlocks();
      push('T8 phản hồi thứ hai cùng thread → không có phản hồi trễ nữa', n2 === n1, `"Phạm Quốc Bảo" ${n1}→${n2}`);
    }
    if (want('T6')) { // rời trang trước 6 s
      await fresh('student', 'sv-2', '/threads/t-salt'); const t0 = await sendReply('Em hiểu rồi ạ'); await sleep(900);
      await click(/^Lịch$/, 'nav a,a', 500); const at = page.url().replace(base, '');
      await sleep(Math.max(0, 7600 - (Date.now() - t0))); const b = await bellText();
      push('T6a gửi xong sang /calendar trước 6 s → tới hạn chuông vẫn có thông báo', /calendar/.test(at) && /Phạm Quốc Bảo đã trả lời trong/.test(b), `at=${at}`);
      await page.keyboard.press('Escape'); await go('/threads/t-salt'); push('T6b quay lại thread thấy phản hồi TA', (await text()).includes(H1_TA), '');
    }
    if (want('T7')) { // tải lại ở giây 3
      await fresh('student', 'sv-2', '/threads/t-salt'); const t0 = await sendReply('Em hiểu rồi ạ'); await sleep(3000); await page.reload({ waitUntil: 'networkidle0' });
      const afterReload = (await text()).includes('Em hiểu rồi ạ');
      const { ev } = await sample(6000, t0, () => false); const t = await text();
      push('T7a tải lại ở giây 3: phản hồi của B còn nguyên', afterReload, '');
      push('T7b chờ tiếp: phản hồi TA vẫn hiện (≤ ~7,5 s kể từ lúc gửi), không phụ thuộc bộ hẹn giờ của trang', t.includes(H1_TA), '');
      const b = await bellText(); push('T7c chuông có thông báo sau tải lại', /Phạm Quốc Bảo đã trả lời trong/.test(b), '');
    }
    if (want('T9')) { // chủ đề không có mẫu
      await fresh('student', 'sv-2', '/threads/t-firewall'); const week = ((await text()).match(/Tuần\s*(\d+)/) || [])[1] || '10';
      await sendReply('Em nghĩ nó còn nhớ cả số thứ tự gói ạ'); await sleep(7600); const t = await text();
      push('T9 thread không có mẫu → TA trả lời câu chung có "tuần <tuần của thread>"', t.replace(/\s+/g, ' ').includes(FALLBACK_TA(week)), `tuần=${week}`, await shot('T9'));
    }

    // ---- T10–T12: Hỏi trợ lý AI (F) ----
    if (want('T10')) {
      await fresh('student', 'sv-2', '/threads/t-cbc'); const t0 = Date.now(); await click(/^Hỏi trợ lý AI$/, 'button', 0);
      const { ev } = await sample(9000, t0, (s, e) => e.dungEnd !== undefined && Date.now() - t0 > 3000); const t = await text();
      push('T10a ô trống + đã có câu AI → Gợi ý thêm của S1 ("…bao nhiêu khối bản rõ bị ảnh hưởng?")', /bao nhiêu khối bản rõ bị ảnh hưởng\?/.test(t), '', await shot('T10'));
      push('T10b theo trình tự E (soạn → chảy → Dừng)', ev.compose !== undefined && ev.dung !== undefined && ev.dungEnd !== undefined, `compose@${f2(ev.compose)} dung@${f2(ev.dung)} end@${f2(ev.dungEnd)}`);
      push('T10c hai nút khoá trong lúc AI soạn', ev.lockedWhileBusy === true, '');
      await sleep(600); const n = await text(); await click(/^Hỏi trợ lý AI$/, 'button', 800); const m = await text();
      push('T10d bấm lần nữa không bao giờ "không có gì" (có phản hồi hoặc lời nhắn hiển thị)', m.length !== n.length, `len ${n.length}→${m.length}`);
    }
    if (want('T11')) {
      await fresh('student', 'sv-2', '/threads/t-cbc'); const q = 'Còn CTR thì sao, có cần IV ngẫu nhiên không?'; await typeReply(q); const t0 = Date.now(); await click(/^Hỏi trợ lý AI$/, 'button', 0);
      await sample(9000, t0, (s, e) => e.dungEnd !== undefined && Date.now() - t0 > 3200); const t = await text();
      const iq = t.indexOf(q), ia = t.indexOf('↳ trả lời Trần Thu Uyên'), is2 = t.search(/nonce/);
      push('T11 có chữ: phản hồi của B hiện trước, AI trả lời ngay dưới theo S2 với "↳ trả lời Trần Thu Uyên"', iq >= 0 && ia > iq && is2 > ia, `q@${iq} ↳@${ia} nonce@${is2}`, await shot('T11'));
    }
    if (want('T12')) {
      await fresh('student', 'sv-2', '/threads/t-cbc'); const q = '@AI dùng lại nonce có sao không'; await typeReply(q); const t0 = Date.now(); await click(/^Gửi phản hồi$/, 'button', 0);
      await sample(9000, t0, (s, e) => e.dungEnd !== undefined && Date.now() - t0 > 3200); const t = await text();
      push('T12 "@AI …" qua Gửi phản hồi xử lý như Hỏi trợ lý AI (có "↳ trả lời Trần Thu Uyên", S2)', /↳ trả lời Trần Thu Uyên/.test(t) && /nonce/.test(t.slice(t.indexOf(q))), '');
    }

    // ---- T13: nháp theo thread ----
    if (want('T13')) {
      await fresh('student', 'sv-2', '/threads/t-cbc'); await typeReply('Em đang gõ dở'); await go('/threads'); await go('/threads/t-cbc');
      const v1 = await page.evaluate(() => [...document.querySelectorAll('textarea')].pop()?.value);
      await go('/threads/t-salt'); const v2 = await page.evaluate(() => [...document.querySelectorAll('textarea')].pop()?.value);
      await go('/threads/t-cbc'); await page.reload({ waitUntil: 'networkidle0' }); const v3 = await page.evaluate(() => [...document.querySelectorAll('textarea')].pop()?.value);
      push('T13 gõ dở → rời trang → quay lại chữ còn nguyên; thread khác không lẫn; tải lại vẫn còn', v1 === 'Em đang gõ dở' && !v2 && v3 === 'Em đang gõ dở', JSON.stringify({ v1, v2, v3 }));
      const dis = await page.evaluate(() => { const ta = [...document.querySelectorAll('textarea')].pop(); const set = Object.getOwnPropertyDescriptor(HTMLTextAreaElement.prototype, 'value').set; set.call(ta, ''); ta.dispatchEvent(new Event('input', { bubbles: true })); const b = [...document.querySelectorAll('button')].find((x) => /^Gửi phản hồi$/.test(x.innerText.trim())); const e = b.disabled; set.call(ta, '   '); ta.dispatchEvent(new Event('input', { bubbles: true })); return { empty: e, spaces: b.disabled }; });
      push('T13b ô trống / chỉ khoảng trắng → Gửi phản hồi khoá', dis.empty && dis.spaces, JSON.stringify(dis));
    }

    // ---- T14: prefers-reduced-motion ----
    if (want('T14')) {
      await page.emulateMediaFeatures([{ name: 'prefers-reduced-motion', value: 'reduce' }]); await fresh();
      const { t0 } = await newThread('Dùng lại IV trong CTR có sao không?', 'Nếu dùng lại nonce thì sao ạ?', 'Mật mã đối xứng'); const { ev } = await sample(3000, t0, (s, e) => e.pending !== undefined);
      push('T14 reduced-motion: không chấm/không chảy, hiện ngay kết quả bước 4 (Chờ xác nhận)', ev.pending !== undefined && ev.pending <= TOL.instantMax + 0.4 && ev.dung === undefined, `pending@${f2(ev.pending)} dung=${ev.dung}`);
      await page.emulateMediaFeatures([]);
    }
    return rows;
  }, { args: [{ base, out, only, TOL, AI_NOMATCH, H1_TA, FALLBACK_TA_SRC: 'Cảm ơn em, anh đã ghi nhận. Thầy cô sẽ trả lời chi tiết trong buổi học tới; em xem trước tài liệu tuần ${w} nhé.' }] });
  await tab.close();
  return rows;
}
export function table(rows) {
  return ['| bước | kết quả | chi tiết | ảnh |', '| --- | --- | --- | --- |'].concat(rows.map((r) => `| ${r.id} | ${r.ok ? 'PASS' : '**FAIL**'} | ${String(r.detail).replace(/\|/g, '\\|').slice(0, 200)} | ${r.file ? r.file.split('/').pop() : ''} |`)).join('\n');
}
export function summary(rows) { const bad = rows.filter((r) => !r.ok); return `Threads timeline: ${rows.length - bad.length}/${rows.length} PASS\n` + bad.map((r) => `  FAIL ${r.id} · ${r.detail}`).join('\n'); }
