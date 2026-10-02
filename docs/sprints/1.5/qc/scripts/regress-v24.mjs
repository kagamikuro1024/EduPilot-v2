// QC — hồi quy 4 lỗi UI PM thấy ở nghiệm thu v5 (góp ý #24, docs/sprints/1.5/proposals.md). Hộp đen: chỉ chữ hiển thị, `data-part`/`data-scroll-x` ghi trong US.md, đo hộp.
//   const m = await import('/abs/docs/sprints/1.5/qc/scripts/regress-v24.mjs');
//   const rows = await m.default(browser, { base: 'http://localhost:3400', out: '/abs/.../shots/qc-v5/v24' });
//   console.log(m.table(rows)); console.log(m.summary(rows));
// TC-02-90 (a) /inbox: bốn tab "Đang chờ / Đã nhận / Đã trả lời / Tất cả" hiện đủ chữ, cao ≥ 32, ở MỌI bề rộng (02-AC9)
// TC-00-69 (b) nút hồ sơ mọi vai ghi TÊN NGƯỜI (00-AC9), vai chỉ ở dòng phụ trong menu
// TC-01-153 (c) /threads hàng chip chủ đề: không chip nào bị cắt ở mép phải mà không có dấu hiệu cuộn (mép mờ / nút cuộn / data-scroll-x) hoặc xuống dòng
// TC-02-91 (d) /attendance 390 & 375: ô chọn buổi hiện nhãn ngắn "Buổi 10 · 29/10" không bị cắt; trạng thái ở dòng phụ
export default async function regress(browser, { base = 'http://localhost:3400', out } = {}) {
  const tab = await browser.open({ name: 'qc-v24', url: base + '/login', viewport: { width: 1440, height: 900 } });
  try {
    return await tab.run(async ({ page }, a) => {
      const { base, out } = a; const rows = []; const sleep = (ms) => new Promise((s) => setTimeout(s, ms));
      const fs = await import('node:fs'); if (out) fs.mkdirSync(out, { recursive: true });
      const push = (id, w, what, ok, detail, file) => rows.push({ id, w: String(w), what, ok: !!ok, detail: detail || '', file: file || '' });
      const setRole = async (role, person = 'sv-2') => { await page.deleteCookie(...(await page.cookies())); for (const [n, v] of [['ep_demo_role', role], ['ep_demo_person', role === 'student' ? person : ''], ['ep_demo_course', 'int1006-1']]) await page.setCookie({ name: n, value: v, url: base }); };
      const setVp = (w, h = w >= 1000 ? 900 : 844) => page.setViewport({ width: w, height: h, deviceScaleFactor: 1, isMobile: w < 500, hasTouch: w < 500 });
      const go = async (u) => { await page.goto(base + u, { waitUntil: 'networkidle0', timeout: 30000 }); await sleep(450); };
      const shot = async (n) => { if (!out) return ''; const f = `${out}/${n}.png`; await page.screenshot({ path: f }); return f; };
      const q = (fn, ...x) => page.evaluate(fn, ...x);

      // ---- (a) TC-02-90: tab hộp thư ----
      const LABELS = ['Đang chờ', 'Đã nhận', 'Đã trả lời', 'Tất cả'];
      await setRole('teacher'); await setVp(1440); await go('/inbox'); await q(() => { try { localStorage.removeItem('ep_demo_state'); } catch {} }); await go('/inbox');
      for (const w of [1440, 1280, 1100, 1099, 900, 720, 390, 375]) {
        await setVp(w, w >= 1000 ? 900 : 812); await go('/inbox');
        const m = await q((LABELS) => {
          const list = document.querySelector('[data-part=inbox-list]'); if (!list) return { err: 'không có [data-part=inbox-list]' };
          const vis = (e) => { const r = e.getBoundingClientRect(); let l = r.left, t = r.top, rr = r.right, b = r.bottom; for (let p = e.parentElement; p && p !== document.documentElement; p = p.parentElement) { const cs = getComputedStyle(p); if (cs.overflowX !== 'visible' || cs.overflowY !== 'visible') { const pb = p.getBoundingClientRect(); l = Math.max(l, pb.left); t = Math.max(t, pb.top); rr = Math.min(rr, pb.right); b = Math.min(b, pb.bottom); } } return { w: Math.max(0, rr - l), h: Math.max(0, b - t) }; };
          const els = [...list.querySelectorAll('button,[role=tab],[role=radio],label,a')];
          const out = LABELS.map((L) => { const e = els.find((x) => new RegExp('^' + L + '(\\s*\\d+)?$').test(x.innerText.replace(/\s+/g, ' ').trim())); if (!e) return { L, found: false }; const b = e.getBoundingClientRect(), v = vis(e); const sc = !!e.closest('[data-scroll-x]'); const clipped = e.scrollWidth > e.clientWidth + 1; return { L, found: true, w: Math.round(b.width), h: Math.round(b.height), visH: Math.round(v.h), visW: Math.round(v.w), scrollRow: sc, clipped }; });
          const lb = list.getBoundingClientRect(); return { out, listW: Math.round(lb.width), listH: Math.round(lb.height) };
        }, LABELS);
        const ok = !m.err && m.out.every((t) => t.found && t.visH >= 28 && t.h >= 28 && t.w >= 40 && !t.clipped && (t.scrollRow || t.visW >= t.w * 0.9));
        push('TC-02-90', w, '02-AC9 (#24a) bốn tab hộp thư hiện đủ chữ, phần nhìn thấy cao ≥ 28, không co thành vạch', ok, JSON.stringify(m).slice(0, 700), ok ? '' : await shot(`inbox-tabs-${w}`));
      }

      // ---- (b) TC-00-69: nút hồ sơ ghi tên người ----
      const NAME = { student: ['sv-2', 'Trần Thu Uyên'], ta: ['', 'Phạm Quốc Bảo'], teacher: ['', 'TS. Lê Thu Hà'], admin: ['', 'Đỗ Hoàng Nam'] };
      const ROLE_LABELS = /^(Giảng viên|Trợ giảng|Admin|Sinh viên)$/;
      for (const w of [1440, 1100, 390]) for (const [role, [person, name]] of Object.entries(NAME)) {
        await setRole(role, person || 'sv-2'); await setVp(w, w >= 1000 ? 900 : 844); await go('/');
        const m = await q(() => { const b = document.querySelector('[aria-label^="Tài khoản"]'); if (!b) return null; const txt = (b.innerText || '').replace(/\s+/g, ' ').trim(); const r = b.getBoundingClientRect(); return { aria: b.getAttribute('aria-label'), txt, w: Math.round(r.width), clipped: [...b.querySelectorAll('*')].some((e) => !e.children.length && e.scrollWidth > e.clientWidth + 1) }; });
        // ≥ 1100 px nút hiện tên đủ; 390 px (nút thu gọn) nhãn truy cập phải là tên và chữ hiển thị (nếu có) KHÔNG là tên vai
        const wide = w >= 1100;
        const ok = !!m && m.aria === 'Tài khoản: ' + name && !ROLE_LABELS.test(m.txt) && (wide ? m.txt.includes(name) && !m.clipped : true);
        push('TC-00-69', `${role}@${w}`, '00-AC9 (#24b) nút hồ sơ ghi tên người, không ghi tên vai', ok, JSON.stringify(m), ok ? '' : await shot(`profile-btn-${role}-${w}`));
      }

      // ---- (c) TC-01-153: chip chủ đề Threads ----
      for (const [role, person] of [['student', 'sv-2'], ['teacher', '']]) for (const w of [1440, 1100, 900, 390]) {
        await setRole(role, person || 'sv-2'); await setVp(w, w >= 1000 ? 900 : 844); await go('/threads');
        const m = await q(() => {
          const seed = [...document.querySelectorAll('main button,main [role=tab],main [role=radio],main label,main a')].find((e) => /^Mật mã đối xứng$/.test(e.innerText.trim()) && e.getBoundingClientRect().width > 0);
          if (!seed) return { err: 'không thấy chip "Mật mã đối xứng"' };
          const row = seed.closest('[data-scroll-x]') || seed.parentElement;
          const chips = [...row.querySelectorAll('button,[role=tab],[role=radio],label,a')].filter((e) => e.getBoundingClientRect().width > 0);
          const scb = row.getBoundingClientRect(), cs = getComputedStyle(row);
          const scrollable = row.scrollWidth > row.clientWidth + 1;
          const wraps = (cs.flexWrap === 'wrap' || cs.flexWrap === 'wrap-reverse') && !scrollable;
          const cut = chips.filter((c) => { const b = c.getBoundingClientRect(); return (b.right > scb.right + 1 && b.left < scb.right - 1) || (b.left < scb.left - 1 && b.right > scb.left + 1); }).map((c) => c.innerText.trim().slice(0, 28));
          const anc = [row, row.parentElement, row.parentElement && row.parentElement.parentElement].filter(Boolean);
          const mask = anc.some((e) => { const s = getComputedStyle(e); return (s.maskImage && s.maskImage !== 'none') || (s.webkitMaskImage && s.webkitMaskImage !== 'none'); });
          const grad = anc.some((e) => ['::before', '::after'].some((p) => /gradient/.test(getComputedStyle(e, p).backgroundImage || '')));
          const btn = !![...(row.parentElement || document.body).querySelectorAll('button')].find((b) => /cuộn|scroll|trước|sau|tiếp|chủ đề khác|xem thêm/i.test((b.getAttribute('aria-label') || '') + ' ' + b.innerText) && b.getBoundingClientRect().width > 0 && !chips.includes(b));
          return { n: chips.length, wraps, scrollable, cut, visibleHint: { mask, grad, btn }, dataScrollX: !!row.closest('[data-scroll-x]'), rowW: Math.round(scb.width) };
        });
        const ok = !m.err && (m.wraps || m.cut.length === 0 || m.visibleHint.mask || m.visibleHint.grad || m.visibleHint.btn);
        push('TC-01-153', `${role}@${w}`, '01-AC19/00-AC10 (#24c) chip chủ đề: không chip nào cắt ở mép khi KHÔNG có dấu hiệu cuộn nhìn thấy (mép mờ / nút cuộn) hoặc xuống dòng', ok, JSON.stringify(m).slice(0, 420), ok ? '' : await shot(`threads-chips-${role}-${w}`));
      }

      // ---- (d) TC-02-91: ô chọn buổi điểm danh ----
      await setRole('teacher');
      for (const w of [390, 375]) {
        await setVp(w, 812); await go('/attendance');
        const m = await q(() => {
          const s = document.querySelector('[data-part=att-session-select]'); if (!s) return null; const r = s.getBoundingClientRect();
          const txt = s.options ? s.options[s.selectedIndex].text.trim() : s.innerText.trim();
          const cs = getComputedStyle(s); const c = document.createElement('canvas').getContext('2d'); c.font = `${cs.fontWeight} ${cs.fontSize} ${cs.fontFamily}`;
          const tw = c.measureText(txt).width; const room = r.width - parseFloat(cs.paddingLeft) - parseFloat(cs.paddingRight) - (s.tagName === 'SELECT' ? 24 : 0);
          const sub = [...document.querySelectorAll('main *')].filter((e) => !e.children.length && /đang diễn|đã kết thúc|chưa bắt đầu|đang mở/i.test(e.textContent) && e.getBoundingClientRect().width > 0 && !s.contains(e)).map((e) => e.textContent.trim().slice(0, 40));
          return { txt, w: Math.round(r.width), textW: Math.round(tw), room: Math.round(room), sub, ox: document.documentElement.scrollWidth - innerWidth };
        });
        const ok = !!m && m.txt === 'Buổi 10 · 29/10' && m.textW <= m.room && m.sub.length >= 1 && m.ox <= 0;
        push('TC-02-91', w, '02-AC15 (#24d) ô Buổi = "Buổi 10 · 29/10" không cắt; trạng thái ở dòng phụ ngoài ô', ok, JSON.stringify(m), ok ? '' : await shot(`att-select-${w}`));
      }
      return rows;
    }, { timeout: 280000, args: [{ base, out }] });
  } finally { try { await tab.close(); } catch (e) {} }
}
export const table = (rows) => ['| TC | bề rộng | phép đo | kết quả | chi tiết | ảnh |', '| --- | --- | --- | --- | --- | --- |'].concat(rows.map((r) => `| ${r.id} | ${r.w} | ${r.what} | ${r.ok ? 'PASS' : 'FAIL'} | ${r.detail.replace(/\|/g, '/').slice(0, 220)} | ${r.file ? r.file.replace(/^.*shots\//, 'shots/') : ''} |`)).join('\n');
export const summary = (rows) => { const bad = rows.filter((r) => !r.ok); return `v24: ${rows.length - bad.length}/${rows.length} PASS` + bad.map((r) => `\n  FAIL ${r.id} @${r.w} · ${r.what} · ${r.detail.slice(0, 160)}`).join(''); };
