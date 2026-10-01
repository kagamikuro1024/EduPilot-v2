// QC — chạy AUDIT (US.md, "Đo bố cục / FR-X12") trên mọi route × vai × bề rộng + các phép đo riêng của spec v5 (SRS 4.7).
// Xuất bảng markdown PASS/FAIL. Không thêm phụ thuộc: dùng Puppeteer `page` của global `browser` (như sweep.mjs); frontend/ không có Playwright.
// Dùng trong Eval (JS):
//   const m = await import('/abs/docs/sprints/1.5/qc/scripts/audit.mjs');
//   const rows = await m.default(browser, { base: 'http://localhost:3000', out: '/abs/docs/sprints/1.5/qc/shots/audit' });
//   console.log(m.table(rows)); console.log(m.summary(rows));
// Tuỳ chọn: only:'student|ta|teacher|admin', skipSpec:true (chỉ ma trận AUDIT), skipMatrix:true (chỉ phép đo riêng), states:true (thêm ?state=empty|error).
// Không đọc mã dev: mọi bộ chọn là móc `data-part` ghi trong US.md ("Móc đo dev thêm") hoặc chữ hiển thị.
import { ROUTES, VIEWPORTS } from './sweep.mjs';

// NGUYÊN VĂN đoạn AUDIT của docs/sprints/1.5/spec/US.md — không sửa để kết quả so được với lệnh Console của spec.
export const AUDIT_SRC = `(() => { const ox = document.documentElement.scrollWidth - innerWidth, cut = [], ell = [];
  for (const e of document.querySelectorAll('main *, header *')) {
    if (e.children.length || !e.textContent.trim() || e.closest('[data-scroll-x]')) continue;
    if (getComputedStyle(e).textOverflow === 'ellipsis' && e.scrollWidth > e.clientWidth && !e.title) { ell.push(e.textContent.trim()); continue; }
    let p = e.parentElement; while (p && getComputedStyle(p).overflowX === 'visible') p = p.parentElement;
    if (!p) continue; const a = e.getBoundingClientRect(), b = p.getBoundingClientRect();
    if (a.right > b.right + 1 || a.left < b.left - 1) cut.push(e.textContent.trim().slice(0, 30));
  } return { ox, cut, ell }; })()`;

const PERSON = { A: 'sv-1', B: 'sv-2', C: 'sv-3', D: 'sv-4' };
// màn chặn quyền cũng phải qua AUDIT (một ví dụ mỗi vai)
const BLOCKED = { student: ['/inbox'], ta: ['/observability'], teacher: ['/chat'], admin: ['/threads'] };
// route SV + /attendance + /inbox thêm 375 (AC10); mọi route 1440 + 390
const needs375 = (role, r) => role === 'student' || r === '/attendance' || r === '/inbox';
// bề rộng quanh ngưỡng của SRS 4.7 (b: 720/1100; h2/h3: 720), chỉ cho ba route shell + route dùng DataTable
const EDGE_W = [{ w: 1100, tag: '1100' }, { w: 1024, tag: '1024' }, { w: 720, tag: '720' }, { w: 719, tag: '719' }];
const EDGE_ROUTES = { teacher: ['/', '/inbox', '/gradebook', '/students', '/grading', '/documents', '/attendance'], admin: ['/admin/courses', '/admin/users', '/settings/llm'], student: ['/', '/library'] };

// ---- Kịch bản có tương tác (AUDIT sau khi trạng thái thay đổi). Metadata ở đây; hàm chạy ở SCN_RUNNERS (chạy TRONG trang). ----
export const SCENARIOS = [
  { id: 'chat sau D1', role: 'student', route: '/chat', vps: ['1440', '390', '375'] },
  { id: 'thread sau Hỏi trợ lý AI', role: 'student', route: '/threads/t-cbc', vps: ['1440', '390', '375'] },
  { id: 'form thread + hộp thoại hai lối', role: 'student', route: '/threads', vps: ['1440', '390', '375'] },
  { id: 'menu hồ sơ mở', role: 'teacher', route: '/', vps: ['1440', '390'] },
  { id: 'bộ chọn lớp mở', role: 'teacher', route: '/', vps: ['1440', '390'] },
  { id: 'bộ chọn lớp mở (SV A, 2 lớp)', role: 'student', person: 'sv-1', route: '/', vps: ['390', '375'] },
  { id: 'sidebar thu gọn', role: 'teacher', route: '/', vps: ['1440'] },
  { id: 'ticket mở (chi tiết)', role: 'teacher', route: '/inbox', vps: ['1440', '390', '375'] },
  { id: 'drawer Thêm', role: 'student', route: '/', vps: ['390', '375'] },
];
export function SCN_RUNNERS({ click, type, sleep }) {
  return {
    'chat sau D1': async (p) => { await type(p, 'textarea', 'Em là Trần Thu Uyên, MSSV 20229002, điểm quá trình của em là bao nhiêu và vắng mấy buổi?'); await p.keyboard.press('Enter'); await sleep(5000); },
    'thread sau Hỏi trợ lý AI': async (p) => { await click(p, /^Hỏi trợ lý AI$/); await sleep(6000); },
    'form thread + hộp thoại hai lối': async (p) => { const i = await p.$$('input[type=text],input:not([type])'); if (i[0]) { await i[0].click(); await i[0].type('Hỏi về điểm'); } await type(p, 'textarea', 'MSSV của em là 20229002 ạ'); await click(p, /^Đăng câu hỏi$/, 'button', 700); },
    'menu hồ sơ mở': async (p) => { await click(p, /Tài khoản/, '[aria-label^="Tài khoản"],button'); },
    'bộ chọn lớp mở': async (p) => { await click(p, /761987/, 'header button'); },
    'bộ chọn lớp mở (SV A, 2 lớp)': async (p) => { await click(p, /761987|761988/, 'header button'); },
    'sidebar thu gọn': async (p) => { await click(p, /Thu gọn/); },
    'ticket mở (chi tiết)': async (p) => { await click(p, /Nguyễn Minh Trung/, 'li,button,a,[role=button],[role=option]', 600); },
    'drawer Thêm': async (p) => { await click(p, /^Thêm$/, 'button,a'); },
  };
}

export default async function audit(browser, { base = 'http://localhost:3000', out, only, skipSpec = false, skipMatrix = false, states = false } = {}) {
  const tab = await browser.open({ name: 'qc-audit', url: base + '/login', viewport: { width: 1440, height: 900 } });
  const rows = await tab.run(async ({ page }, a) => {
    const { base, out, only, skipSpec, skipMatrix, states, ROUTES, VIEWPORTS, BLOCKED, EDGE_W, EDGE_ROUTES, SCENARIOS_META, AUDIT_SRC, specSrc, scnSrc } = a;
    const rows = [];
    const click = async (p, re, sel = 'button,a,[role=button],[role=menuitem],[role=tab],summary,label', wait = 300) => {
      const ok = await p.evaluate((src, flags, s) => { const r = new RegExp(src, flags); const el = [...document.querySelectorAll(s)].find((e) => r.test((e.innerText || e.getAttribute('aria-label') || '').trim()) && e.getBoundingClientRect().width > 0); if (el) { el.click(); return true; } return false; }, re.source, re.flags, sel);
      await new Promise((s) => setTimeout(s, wait)); return ok;
    };
    const type = async (p, sel, text) => { const el = await p.$(sel); if (!el) return false; await el.click(); await el.type(text); return true; };
    const scn = new Function('h', 'return (' + scnSrc + ')(h)')({ click, type, sleep: (ms) => new Promise((s) => setTimeout(s, ms)) });
    const fs = await import('node:fs'); if (out) fs.mkdirSync(out, { recursive: true });
    const errs = [];
    page.on('console', (m) => { if (m.type() === 'error') errs.push(m.text().slice(0, 120)); });
    page.on('pageerror', (e) => errs.push('pageerror: ' + e.message.slice(0, 120)));
    const sleep = (ms) => new Promise((s) => setTimeout(s, ms));
    const setRole = async (role, person = 'sv-2', course = 'int1006-1') => {
      await page.deleteCookie(...(await page.cookies()));
      for (const [n, v] of [['ep_demo_role', role], ['ep_demo_person', role === 'student' ? person : ''], ['ep_demo_course', course]]) await page.setCookie({ name: n, value: v, url: base });
    };
    const setVp = (w, h = w >= 1000 ? 900 : 844) => page.setViewport({ width: w, height: h, deviceScaleFactor: 1, isMobile: w < 500, hasTouch: w < 500 });
    const go = async (url) => { await page.goto(base + url, { waitUntil: 'networkidle0', timeout: 30000 }); await sleep(450); };
    const reset = async () => { await page.evaluate(() => { try { localStorage.removeItem('ep_demo_state'); } catch {} }); };
    const push = (role, route, w, what, ok, detail, file) => rows.push({ role, route, w, what, ok, detail: detail || '', file: file || '' });
    const shot = async (name) => { if (!out) return ''; const f = `${out}/${name.replace(/[\/\[\]?=\s·]/g, '_')}.png`; await page.screenshot({ path: f }); return f; };
    const fmt = (r) => `ox=${r.ox} cut=${JSON.stringify(r.cut.slice(0, 3))} ell=${JSON.stringify(r.ell.slice(0, 3))} sx=${r.sx}`;
    const okOf = (r) => r.ox <= 0 && r.cut.length === 0 && r.ell.length === 0;
    const run = async (role, route, tag, w, label, person, pre) => {
      errs.length = 0; await go(route); if (pre) await pre(page); await sleep(250);
      const r = { ...(await page.evaluate(AUDIT_SRC)), sx: await page.evaluate(() => document.querySelectorAll('[data-scroll-x]').length) };
      const blocked = await page.evaluate(() => /Bạn không có quyền mở trang này/.test(document.body.innerText));
      const f = !okOf(r) || errs.length ? await shot(`${role}${route}-${tag}-${label || 'audit'}`) : '';
      push(role, route, tag, label || 'AUDIT', okOf(r) && !errs.length, fmt(r) + (blocked ? ' [màn chặn]' : '') + (errs.length ? ` console=${errs[0]}` : ''), f);
    };

    // ---------- 1. Ma trận AUDIT: mọi route × vai × bề rộng ----------
    if (!skipMatrix) for (const role of Object.keys(ROUTES)) {
      if (only && only !== role) continue;
      await setRole(role); await setVp(1440); await go(ROUTES[role][0]); await reset();
      const list = [...ROUTES[role], ...BLOCKED[role]];
      for (const r of list) for (const vp of VIEWPORTS) {
        if (vp.tag === '375' && !(role === 'student' || r === '/attendance' || r === '/inbox')) continue;
        await setVp(vp.w, vp.h);
        for (const q of states && !BLOCKED[role].includes(r) ? ['', '?state=empty', '?state=error'] : ['']) await run(role, r + q, vp.tag, vp.w);
      }
      for (const r of EDGE_ROUTES[role] || []) for (const e of EDGE_W) { await setVp(e.w, 900); await run(role, r, e.tag, e.w, 'AUDIT-biên'); }
    }
    // /login (không phiên): 1440, 390, 375
    if (!skipMatrix && !only) { await page.deleteCookie(...(await page.cookies())); for (const w of [1440, 390, 375]) { await setVp(w); await run('(không vai)', '/login', String(w), w); } }

    // ---------- 2. AUDIT sau tương tác ----------
    if (!skipMatrix) for (const s of SCENARIOS_META) {
      if (only && only !== s.role) continue;
      await setRole(s.role, s.person || 'sv-2');
      for (const vp of VIEWPORTS.filter((v) => s.vps.includes(v.tag))) {
        await setVp(vp.w, vp.h); await go(s.route); await reset(); await go(s.route);
        await run(s.role, s.route, vp.tag, vp.w, s.id, s.person, async (p) => { await scn[s.id](p); });
      }
    }

    // ---------- 3. Phép đo riêng của spec v5 (SRS 4.7) ----------
    if (!skipSpec && !only) await new Function('ctx', 'return (' + specSrc + ')(ctx)')({ page, base, setRole, setVp, go, reset, push, sleep, shot });
    return rows;
  }, { args: [{ base, out, only, skipSpec, skipMatrix, states, ROUTES, VIEWPORTS: VIEWPORTS.map(({ w, h, tag }) => ({ w, h, tag })), BLOCKED, EDGE_W, EDGE_ROUTES, SCENARIOS_META: SCENARIOS, AUDIT_SRC, specSrc: SPEC_CHECKS.toString(), scnSrc: SCN_RUNNERS.toString() }] });
  await tab.close();
  return rows;
}

// ---- Phép đo riêng: chạy TRONG trang (Puppeteer page), nhận ctx ----
export async function SPEC_CHECKS({ page, setRole, setVp, go, reset, push, sleep, shot }) {
  const NAMES = { teacher: ['TS. Lê Thu Hà', 'Giảng viên'], ta: ['Phạm Quốc Bảo', 'Trợ giảng'], admin: ['Đỗ Hoàng Nam', 'Admin'] };
  const SV = { 'sv-1': 'Nguyễn Minh Trung', 'sv-2': 'Trần Thu Uyên', 'sv-3': 'Lê Quang Huy', 'sv-4': 'Phạm Ngọc Linh' };
  const q = (fn, ...args) => page.evaluate(fn, ...args);
  const clickTxt = async (re, sel = 'button,a,[role=menuitem],[role=button],[role=tab],label,summary', wait = 300) => {
    const ok = await q((src, flags, s) => { const r = new RegExp(src, flags); const el = [...document.querySelectorAll(s)].find((e) => r.test((e.innerText || e.getAttribute('aria-label') || '').trim()) && e.getBoundingClientRect().width > 0); if (el) { el.click(); return true; } return false; }, re.source, re.flags, sel);
    await sleep(wait); return ok;
  };
  const chk = (role, route, w, what, ok, detail, f) => push(role, route, String(w), what, !!ok, detail, f);
  const rect = (sel) => q((s) => { const e = document.querySelector(s); if (!e) return null; const b = e.getBoundingClientRect(); return { l: b.left, r: b.right, t: b.top, b: b.bottom, w: b.width, h: b.height }; }, sel);

  // 00-AC7 brand
  for (const role of ['student', 'ta', 'teacher', 'admin']) {
    await setRole(role); await setVp(1440, 900); await go('/');
    const m = await q(() => { const b = document.querySelector('[data-part=brand]'), i = document.querySelector('[data-part=brand] img'), h = document.querySelector('header'); if (!b || !i || !h) return null; const bb = b.getBoundingClientRect(), ib = i.getBoundingClientRect(), hb = h.getBoundingClientRect(); return { img: Math.round(ib.height), brandH: Math.round(bb.height), bBottom: Math.round(bb.bottom), hBottom: Math.round(hb.bottom), border: getComputedStyle(b).borderBottomWidth, ratio: +(ib.width / ib.height).toFixed(2) }; });
    chk(role, '/', 1440, '00-AC7 brand 32/56/viền liền', m && m.img === 32 && m.brandH === 56 && m.bBottom === 56 && m.hBottom === 56 && m.border === '1px' && Math.abs(m.ratio - 188 / 40) < 0.15, JSON.stringify(m), await shot(`brand-${role}-1440`));
  }
  await setRole('teacher'); await setVp(1440, 900); await go('/'); await clickTxt(/Thu gọn/);
  const col = await q(() => { const i = document.querySelector('[data-part=brand] img'); const b = i && i.getBoundingClientRect(); const s = document.querySelector('[data-part=brand]').getBoundingClientRect(); return b && { w: Math.round(b.width), h: Math.round(b.height), cx: Math.round(b.left + b.width / 2), colW: Math.round(s.width), colCx: Math.round(s.left + s.width / 2) }; });
  chk('teacher', '/', 1440, '00-AC7 thu gọn: mark 32×32, giữa cột 72', col && col.w === 32 && col.h === 32 && col.colW === 72 && Math.abs(col.cx - col.colCx) <= 1, JSON.stringify(col), await shot('brand-collapsed-1440'));
  for (const role of ['student', 'teacher']) {
    await setRole(role); await setVp(390, 844); await go('/');
    const m = await q(() => { const a = document.querySelector('header a[href="/"]'), i = a && a.querySelector('img'); if (!a || !i) return null; const ab = a.getBoundingClientRect(), ib = i.getBoundingClientRect(); const chooser = [...document.querySelectorAll('header button')].find((b) => /\d{6}/.test(b.innerText)); return { img: [Math.round(ib.width), Math.round(ib.height)], tap: [Math.round(ab.width), Math.round(ab.height)], before: chooser ? ib.left < chooser.getBoundingClientRect().left : null }; });
    chk(role, '/', 390, '00-AC7 topbar mark 28, vùng chạm 44, đứng trước bộ chọn lớp', m && m.img[0] === 28 && m.img[1] === 28 && m.tap[0] >= 44 && m.tap[1] >= 44 && m.before !== false, JSON.stringify(m), await shot(`brand-mark-${role}-390`));
  }
  await page.deleteCookie(...(await page.cookies()));
  for (const [w, h, want] of [[1440, 900, 40], [390, 844, 32], [375, 812, 32]]) {
    await setVp(w, h); await go('/login');
    const m = await q(() => { const i = document.querySelector('img[alt="EduPilot"]'); return i && { h: Math.round(i.getBoundingClientRect().height), ox: document.documentElement.scrollWidth - innerWidth }; });
    chk('(không vai)', '/login', w, `00-AC7 logo /login = ${want}`, m && m.h === want && m.ox <= 0, JSON.stringify(m), await shot(`login-${w}`));
  }

  // 00-AC8 bộ chọn lớp
  await setRole('teacher');
  const chooserInfo = () => q(() => { const b = [...document.querySelectorAll('header button')].find((x) => /\d{6}/.test(x.innerText)); if (!b) return null; const t = b.querySelector('[class*=ell],span') || b; const r = b.getBoundingClientRect(); return { text: b.innerText.replace(/\s+/g, ' ').trim(), w: Math.round(r.width), title: b.title || b.querySelector('[title]')?.title || '', aria: b.getAttribute('aria-label') || '', trunc: [...b.querySelectorAll('*')].some((e) => e.scrollWidth > e.clientWidth + 1) || b.scrollWidth > b.clientWidth + 1 }; });
  await setVp(1440, 900); for (const r of ['/', '/inbox', '/gradebook']) { await go(r); const c = await chooserInfo(); const search = await q(() => { const i = document.querySelector('header input,header [placeholder]'); const s = [...document.querySelectorAll('header button,header input')].find((e) => /Tìm nhanh/.test(e.placeholder || e.innerText)); return s ? { full: /Tìm nhanh hoặc đi đến…/.test(s.placeholder || s.innerText), clipped: s.scrollWidth > s.clientWidth + 1 } : null; });
    chk('teacher', r, 1440, '00-AC8 chooser đủ "761987 · An ninh mạng", ≤ 360, ô tìm đủ chữ', c && /761987\s*·\s*An ninh mạng/.test(c.text) && !c.trunc && c.w <= 360 && search && search.full && !search.clipped, JSON.stringify({ c, search }), await shot(`chooser-1440${r.replace(/\//g, '_')}`)); }
  for (const [w, lo, hi] of [[1024, 0, 240], [1099, 0, 240], [720, 0, 240]]) { await setVp(w, 900); await go('/'); const c = await chooserInfo(); chk('teacher', '/', w, '00-AC8 chooser ≤ 240; nếu cắt thì title+aria = tên đủ', c && c.w <= hi && (!c.trunc || (c.title.includes('761987') && c.aria.includes('761987') && c.title.length >= c.text.length - 3)), JSON.stringify(c), await shot(`chooser-${w}`)); }
  await setVp(1100, 900); await go('/'); { const c = await chooserInfo(); chk('teacher', '/', 1100, '00-AC8 biên 1100: đủ tên, ≤ 360', c && /761987\s*·\s*An ninh mạng/.test(c.text) && !c.trunc && c.w <= 360, JSON.stringify(c)); }
  await setVp(719, 900); await go('/'); { const c = await chooserInfo(); chk('teacher', '/', 719, '00-AC8 biên 719: chỉ mã lớp', c && c.text.replace(/[^\d]/g, '') === '761987' && !/An ninh/.test(c.text), JSON.stringify(c)); }
  await setVp(390, 844); await go('/'); { const c = await chooserInfo(); chk('teacher', '/', 390, '00-AC8 390: chỉ "761987"', c && c.text.replace(/[^\d]/g, '') === '761987' && !/An ninh/.test(c.text), JSON.stringify(c)); await clickTxt(/761987/, 'header button'); const pop = await q(() => { const p = [...document.querySelectorAll('[role=menu],[role=listbox],[role=dialog],[popover]')].find((e) => /An ninh mạng/.test(e.innerText)); if (!p) return null; const cut = [...p.querySelectorAll('*')].filter((e) => !e.children.length && e.scrollWidth > e.clientWidth + 1).map((e) => e.textContent.trim().slice(0, 30)); return { text: p.innerText.split('\n')[0], cut, ox: p.scrollWidth - p.clientWidth }; }); chk('teacher', '/', 390, '00-AC8 popover có tên đầy đủ, không cắt', pop && /An ninh mạng/.test(pop.text) && pop.cut.length === 0 && pop.ox <= 1, JSON.stringify(pop), await shot('chooser-popover-390')); }

  // 00-AC9 menu hồ sơ
  const profile = async (role, person, name, roleLabel, w) => {
    await setRole(role, person); await setVp(w, w > 500 ? 900 : 844); await go('/');
    const aria = await q(() => document.querySelector('[aria-label^="Tài khoản"]')?.getAttribute('aria-label'));
    await clickTxt(/Tài khoản/, '[aria-label^="Tài khoản"]');
    const menu = await q(() => { const m = [...document.querySelectorAll('[role=menu],[role=dialog],[popover]')].find((e) => /Đổi vai/.test(e.innerText)); if (!m) return null; const ls = m.innerText.split('\n').map((s) => s.trim()).filter(Boolean); return { l1: ls[0], l2: ls[1], cut: [...m.querySelectorAll('*')].some((e) => !e.children.length && e.scrollWidth > e.clientWidth + 1) }; });
    chk(role + (person ? ':' + person : ''), '/', w, '00-AC9 aria "Tài khoản: <tên>", dòng 1 = tên, dòng 2 = vai', aria === 'Tài khoản: ' + name && menu && menu.l1 === name && menu.l2 === roleLabel && !menu.cut, JSON.stringify({ aria, menu }), await shot(`profile-${role}-${person || ''}-${w}`));
  };
  for (const [role, [n, l]] of Object.entries(NAMES)) { await profile(role, '', n, l, 1440); await profile(role, '', n, l, 390); }
  for (const [p, n] of Object.entries(SV)) { await profile('student', p, n, 'Sinh viên', 1440); }
  await profile('student', 'sv-1', SV['sv-1'], 'Sinh viên', 375);

  // 01-AC11 chat 1440 / 375
  const chat = async (w, h, label) => { const m = await q(() => { const r = (p) => { const e = document.querySelector('[data-part=' + p + ']'); return e && e.getBoundingClientRect(); }; const t = r('chat-thread'), c = r('chat-composer'), hi = r('chat-history'); if (!t || !c) return null; return { sameLeft: t.left === c.left, sameW: t.width === c.width, bottom: innerHeight - c.bottom, histW: hi ? Math.round(hi.width) : null, histVisible: hi ? hi.width > 0 : false, pageScroll: document.documentElement.scrollHeight - innerHeight }; });
    return m; };
  await setRole('student', 'sv-2'); await setVp(1440, 900); await go('/chat'); await reset(); await go('/chat');
  { const m = await chat(); chk('student', '/chat', 1440, '01-AC11 phiên trống: [left, width, đáy ≤ 24], lịch sử 240', m && m.sameLeft && m.sameW && m.bottom <= 24 && m.histW === 240, JSON.stringify(m), await shot('chat-empty-1440')); }
  await clickTxt(/Cách chọn độ dài khoá RSA/, 'button,a,li,[role=button]', 600);
  { const m = await chat(); chk('student', '/chat', 1440, '01-AC11 phiên ≥ 6 tin (mở phiên cũ 6 tin)', m && m.sameLeft && m.sameW && m.bottom <= 24 && m.pageScroll <= 1, JSON.stringify(m), await shot('chat-6msg-1440')); }
  await setVp(1440, 600); await go('/chat'); { const m = await chat(); chk('student', '/chat', '1440×600', '01-AC11 biên: màn thấp vẫn ghim đáy', m && m.sameLeft && m.sameW && m.bottom <= 24, JSON.stringify(m)); }
  await setVp(375, 812); await go('/chat'); { const m = await chat(); chk('student', '/chat', 375, '01-AC11 375: lịch sử ẩn', !m || !m.histVisible, JSON.stringify(m), await shot('chat-375')); }

  // 02-AC9/AC10 inbox
  await setRole('teacher'); await setVp(1440, 900); await go('/inbox'); await reset(); await go('/inbox');
  chk('teacher', '/inbox', 1440, '02-AC9 chip "Tất cả 6" đủ chữ + hàng lọc có data-scroll-x', await q(() => /Tất cả\s*6/.test(document.body.innerText) && !!document.querySelector('[data-part=inbox-list] [data-scroll-x],[data-scroll-x]')), '', await shot('inbox-1440'));
  const listW = await rect('[data-part=inbox-list]'); chk('teacher', '/inbox', 1440, '02-AC9 cột danh sách ≤ 380 px', listW && listW.w <= 381, JSON.stringify(listW));
  const names = ['Nguyễn Minh Trung', 'Đặng Gia An', 'Lê Quang Huy', 'Đỗ Thanh Long', 'Lý Gia Thảo'];
  for (const n of names) {
    await go('/inbox'); await clickTxt(new RegExp(n), 'li,button,a,[role=button],[role=option]', 500);
    const g = await q(() => { const r = document.querySelector('[data-part=inbox-reply]'); if (!r) return null; const prev = r.previousElementSibling; const ta = r.querySelector('textarea'), btn = [...r.querySelectorAll('button')].find((b) => /Gửi trả lời/.test(b.innerText)); const d = document.querySelector('[data-part=inbox-detail]'); return { gap: Math.round(r.getBoundingClientRect().top - (prev ? prev.getBoundingClientRect().bottom : 0)), btnGap: ta && btn ? Math.round(btn.getBoundingClientRect().top - ta.getBoundingClientRect().bottom) : null, slack: d ? Math.round(d.getBoundingClientRect().bottom - (d.lastElementChild ? d.lastElementChild.getBoundingClientRect().bottom : 0)) : null }; });
    chk('teacher', '/inbox', 1440, `02-AC10 ticket ${n}: khoảng trống ≤ 24, Gửi trả lời sát ô soạn`, g && g.gap <= 24 && (g.btnGap === null || g.btnGap <= 24), JSON.stringify(g), n === 'Lý Gia Thảo' ? await shot('inbox-detail-1440') : '');
  }
  // vị trí cuộn: ưu tiên cuộn trong danh sách nếu danh sách tự cuộn, không thì cuộn trang
  const setY = (y) => q((v) => { const l = document.querySelector('[data-part=inbox-list]'); if (l) { l.scrollTop = v; if (l.scrollTop > 0) return; } window.scrollTo(0, v); }, y);
  const getY = () => q(() => { const l = document.querySelector('[data-part=inbox-list]'); return l && l.scrollTop > 0 ? l.scrollTop : window.scrollY; });
  const inMain = 'main button,main a,[data-part=inbox-detail] button,[data-part=inbox-detail] a';
  for (const w of [375, 390]) {
    await setVp(w, 812); await go('/inbox'); await setY(250); await sleep(200); const y0 = await getY();
    await clickTxt(/Lý Gia Thảo/, 'li,button,a,[role=button],[role=option]', 600);
    const u = page.url();
    const back = await q(() => { const b = [...document.querySelectorAll('main button,main a,[data-part=inbox-detail] button,[data-part=inbox-detail] a')].find((e) => /Hộp thư/.test(e.innerText) || /Hộp thư/.test(e.getAttribute('aria-label') || '')); if (!b) return null; const r = b.getBoundingClientRect(); const l = document.querySelector('[data-part=inbox-list]'); return { w: Math.round(r.width), h: Math.round(r.height), listHidden: !l || l.getBoundingClientRect().width === 0 }; });
    chk('teacher', '/inbox', w, '02-AC9 mobile: URL ?ticket=, ← Hộp thư ≥ 44×44, danh sách ẩn', /[?&]ticket=/.test(u) && back && back.w >= 44 && back.h >= 44 && back.listHidden, JSON.stringify({ u: u.replace(/^.*\/inbox/, '/inbox'), back, y0 }), await shot(`inbox-ticket-${w}`));
    await clickTxt(/Hộp thư/, inMain, 700); const y1 = await getY();
    chk('teacher', '/inbox', w, '02-AC9 "← Hộp thư" về danh sách đúng vị trí cuộn (±2 px)', Math.abs(y0 - y1) <= 2 && !/[?&]ticket=/.test(page.url()), JSON.stringify({ y0, y1 }));
    await clickTxt(/Lý Gia Thảo/, 'li,button,a,[role=button],[role=option]', 500); await page.goBack(); await sleep(700); const y2 = await getY();
    chk('teacher', '/inbox', w, '02-AC9 Back của trình duyệt về danh sách đúng vị trí cuộn (±2 px)', Math.abs(y0 - y2) <= 2 && !/[?&]ticket=/.test(page.url()), JSON.stringify({ y0, y2 }));
  }
  for (const [w, wantSplit] of [[1100, true], [1099, false]]) { await setVp(w, 900); await go('/inbox'); const m = await q(() => { const l = document.querySelector('[data-part=inbox-list]'), d = document.querySelector('[data-part=inbox-detail]'); return { list: !!l && l.getBoundingClientRect().width > 0, detail: !!d && d.getBoundingClientRect().width > 0 }; }); chk('teacher', '/inbox', w, `02-AC9 biên ${w}: ${wantSplit ? 'hai panel' : 'chỉ danh sách'}`, wantSplit ? m.list && m.detail : m.list && !m.detail, JSON.stringify(m)); }

  // 04-AC7 provider
  await setRole('admin'); await setVp(1440, 900); await go('/settings/llm');
  const cols = await q(() => ['provider-status', 'provider-action'].map((p) => [...document.querySelectorAll('[data-part=' + p + ']')].map((e) => Math.round(e.getBoundingClientRect().left))));
  const wd = await q(() => ['provider-status', 'provider-action'].map((p) => [...new Set([...document.querySelectorAll('[data-part=' + p + ']')].map((e) => Math.round(e.getBoundingClientRect().width)))]));
  chk('admin', '/settings/llm', 1440, '04-AC7 cột trạng thái/nút thẳng hàng [1,1], ≥ 3 hàng', cols[0].length >= 3 && new Set(cols[0]).size === 1 && new Set(cols[1]).size === 1, JSON.stringify({ cols, wd }), await shot('llm-1440'));
  chk('admin', '/settings/llm', 1440, '04-AC7 lưới cố định: trạng thái 200, nút 140', wd[0].length === 1 && Math.abs(wd[0][0] - 200) <= 1 && wd[1].length === 1 && Math.abs(wd[1][0] - 140) <= 1, JSON.stringify(wd));
  for (const w of [390, 375]) { await setVp(w, 844); await go('/settings/llm'); const m = await q(() => [...document.querySelectorAll('[data-part=provider-status]')].map((s) => { const row = s.parentElement, a = row.querySelector('[data-part=provider-action]'), info = row.firstElementChild; const sb = s.getBoundingClientRect(), ab = a && a.getBoundingClientRect(), ib = info.getBoundingClientRect(), rb = row.getBoundingClientRect(); const btn = a && (a.querySelector('button') || a); const bb = btn && btn.getBoundingClientRect(); return { order: ib.top < sb.top && ab && sb.top < ab.top, full: bb && bb.width >= rb.width * 0.95 - 32, h: bb && Math.round(bb.height) }; }));
    chk('admin', '/settings/llm', w, '04-AC7 hàng xếp dọc: thông tin → trạng thái → nút rộng 100 %, ≥ 44', m.length >= 3 && m.every((x) => x.order && x.full && x.h >= 44), JSON.stringify(m), await shot(`llm-${w}`)); }
  await setRole('teacher'); await setVp(1440, 900); await go('/settings/llm'); { const c = await q(() => [...document.querySelectorAll('[data-part=provider-status]')].map((e) => Math.round(e.getBoundingClientRect().left))); chk('teacher', '/settings/llm', 1440, '04-AC7 GV (chỉ xem): cột trạng thái vẫn thẳng hàng', c.length >= 3 && new Set(c).size === 1, JSON.stringify(c)); }

  // 00-AC10 h1–h4
  const tableMode = async (role, route, w) => { await setRole(role); await setVp(w, 844); await go(route); return q(() => { const t = [...document.querySelectorAll('main table')].filter((x) => x.getBoundingClientRect().width > 0); return { tables: t.length, inScrollX: t.filter((x) => x.closest('[data-scroll-x]')).length, ox: document.documentElement.scrollWidth - innerWidth }; }); };
  for (const [role, route] of [['teacher', '/students'], ['teacher', '/grading'], ['teacher', '/documents'], ['admin', '/admin/courses'], ['admin', '/admin/users']]) {
    for (const [w, list] of [[390, true], [719, true], [720, false]]) { const m = await tableMode(role, route, w); chk(role, route, w, `00-AC10 h3 ${list ? 'dạng danh sách' : 'bảng'} ở ${w}`, list ? m.tables === 0 && m.ox <= 0 : m.tables >= 1 && m.ox <= 0, JSON.stringify(m), w === 390 ? await shot(`list-${role}${route.replace(/\//g, '_')}-390`) : ''); }
  }
  { await setRole('teacher'); await setVp(390, 844); await go('/gradebook');
    const m = await q(async () => { const sc = document.querySelector('[data-scroll-x]'); const c0 = () => { const td = sc && sc.querySelector('tbody td'); return td ? td.getBoundingClientRect().left : null; }; if (!sc) return null; const a = c0(); sc.scrollLeft = 220; await new Promise((r) => setTimeout(r, 200)); const b = c0(); const hd = [...sc.querySelectorAll('thead th')].map((t) => t.innerText.trim()); return { scrollable: sc.scrollWidth > sc.clientWidth, stickyDelta: Math.round(Math.abs(a - b)), hd, ox: document.documentElement.scrollWidth - innerWidth }; });
    chk('teacher', '/gradebook', 390, '00-AC10 h3 gradebook: data-scroll-x, cột tên dính trái, ox 0', m && m.scrollable && m.stickyDelta <= 1 && m.ox <= 0, JSON.stringify(m), await shot('gradebook-390')); }
  { await setRole('teacher'); await setVp(375, 812); await go('/attendance');
    const m = await q(() => { const hint = [...document.querySelectorAll('main *')].filter((e) => !e.children.length && /chọn hàng|đánh dấu|ghi phát biểu/.test(e.textContent) && e.getBoundingClientRect().width > 0); return { hintVisible: hint.length, vangPhep: /Vắng phép/.test(document.body.innerText), ox: document.documentElement.scrollWidth - innerWidth }; });
    chk('teacher', '/attendance', 375, '00-AC10 h2: ẩn chú thích phím, nhãn "Vắng phép" đủ chữ, ox 0', m.hintVisible === 0 && m.vangPhep && m.ox <= 0, JSON.stringify(m), await shot('attendance-375')); }
  { await setRole('teacher'); await setVp(390, 844); await go('/students/sv-3');
    const m = await q(async () => { const tabs = [...document.querySelectorAll('[role=tab]')]; if (!tabs.length) return null; const list = tabs[0].parentElement; const hd = tabs.find((t) => /Hoạt động học/.test(t.innerText)); const scrollable = list.scrollWidth > list.clientWidth || !!list.closest('[data-scroll-x]') || list.hasAttribute('data-scroll-x'); const note = tabs.find((t) => /Ghi chú/.test(t.innerText)); note.click(); await new Promise((r) => setTimeout(r, 400)); const lb = list.getBoundingClientRect(), nb = note.getBoundingClientRect(); return { scrollable, noteInView: nb.left >= lb.left - 1 && nb.right <= lb.right + 1, ox: document.documentElement.scrollWidth - innerWidth, hdFull: hd ? hd.scrollWidth <= hd.clientWidth + 1 : false }; });
    chk('teacher', '/students/sv-3', 390, '00-AC10 h4: tab cuộn ngang, tab chọn tự cuộn vào khung, "Hoạt động học" không cắt', m && m.scrollable && m.noteInView && m.hdFull && m.ox <= 0, JSON.stringify(m), await shot('tabs-sv3-390')); }
}

export function table(rows) {
  const esc = (s) => String(s).replace(/\|/g, '\\|').replace(/\n/g, ' ');
  return ['| vai | route | rộng | phép đo | kết quả | chi tiết | ảnh |', '| --- | --- | --- | --- | --- | --- | --- |'].concat(rows.map((r) => `| ${r.role} | \`${r.route}\` | ${r.w} | ${esc(r.what)} | ${r.ok ? 'PASS' : '**FAIL**'} | ${esc(r.detail).slice(0, 220)} | ${r.file ? r.file.split('/').pop() : ''} |`)).join('\n');
}
export function summary(rows) {
  const bad = rows.filter((r) => !r.ok);
  return `AUDIT: ${rows.length - bad.length}/${rows.length} PASS; FAIL ${bad.length}\n` + bad.map((r) => `  FAIL ${r.role} ${r.route} @${r.w} · ${r.what} · ${r.detail.slice(0, 120)}`).join('\n');
}
