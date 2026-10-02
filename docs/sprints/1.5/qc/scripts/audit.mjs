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
// NGUYÊN VĂN `TOUCH` của US.md (v5.1): vùng bấm < 44 px, chạy ở 375 / 390. Đạt khi `[]`.
export const TOUCH_SRC = `[...document.querySelectorAll('main a, main button, main [role=tab], main [role=radio], main label, main input, header a, header button, nav a')]
  .filter(e => e.offsetParent && !e.closest('[data-inline]') && !(e.matches('input') && e.closest('label')) && !(e.matches('label') && !e.querySelector('input[type=checkbox],input[type=radio]') && !(e.control && ['checkbox','radio'].includes(e.control.type))))
  .map(e => { const b = e.getBoundingClientRect(); return { t: (e.getAttribute('aria-label') || e.textContent).trim().slice(0, 24), w: Math.round(b.width), h: Math.round(b.height) }; })
  .filter(x => x.w < 44 || x.h < 44)`;
// `LEFT` của US.md, bọc để route thiếu `[data-part=page-title]` trả null (= FAIL) thay vì ném lỗi.
export const LEFT_SRC = `(() => { const e = document.querySelector('[data-part=page-title]'); return e ? Math.round(e.getBoundingClientRect().left) : null; })()`;
// phạm vi TOUCH của 00-AC15: 12 route SV + /attendance, /inbox của GV
const TOUCH_SV = ['/', '/chat', '/threads', '/threads/t-cbc', '/practice', '/practice/at-symmetric', '/practice/history', '/library', '/calendar', '/me', '/assignments/bt03', '/join'];
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
    'form thread + hộp thoại hai lối': async (p) => { await click(p, /^Đặt câu hỏi$/, 'button', 500); const i = await p.$$('[data-part=thread-form] input[type=text],[data-part=thread-form] input:not([type])'); if (i[0]) { try { await i[0].click(); } catch { await p.evaluate((e) => e.focus(), i[0]); } await i[0].type('Hỏi về điểm'); } await type(p, '[data-part=thread-form] textarea', 'MSSV của em là 20229002 ạ'); await click(p, /^Đăng câu hỏi$/, 'button', 700); },
    'menu hồ sơ mở': async (p) => { await click(p, /Tài khoản/, '[aria-label^="Tài khoản"],button'); },
    'bộ chọn lớp mở': async (p) => { await click(p, /761987/, 'header button'); },
    'bộ chọn lớp mở (SV A, 2 lớp)': async (p) => { await click(p, /761987|761988/, 'header button'); },
    'sidebar thu gọn': async (p) => { await click(p, /Thu gọn/); },
    'ticket mở (chi tiết)': async (p) => { await click(p, /Nguyễn Minh Trung/, 'li,button,a,[role=button],[role=option]', 600); },
    'drawer Thêm': async (p) => { await click(p, /^Thêm$/, 'button,a'); },
  };
}

export default async function audit(browser, { base = 'http://localhost:3000', out, only, skipSpec = false, skipMatrix = false, states = false, routesOnly = null, skipScenarios = false, skipEdge = false, specWhich = "" } = {}) {
  const tab = await browser.open({ name: 'qc-audit', url: base + '/login', viewport: { width: 1440, height: 900 } });
  const rows = await tab.run(async ({ page }, a) => {
    const { base, out, only, skipSpec, skipMatrix, states, routesOnly, skipScenarios, skipEdge, specWhich, ROUTES, VIEWPORTS, BLOCKED, EDGE_W, EDGE_ROUTES, SCENARIOS_META, AUDIT_SRC, TOUCH_SRC, LEFT_SRC, TOUCH_SV, specSrc, spec51Src, scnSrc } = a;
    const rows = [];
    const click = async (p, re, sel = 'button,a,[role=button],[role=menuitem],[role=tab],summary,label', wait = 300) => {
      const ok = await p.evaluate((src, flags, s) => { const r = new RegExp(src, flags); const el = [...document.querySelectorAll(s)].find((e) => (r.test((e.innerText || '').trim()) || r.test((e.getAttribute('aria-label') || '').trim())) && e.getBoundingClientRect().width > 0); if (el) { const inner = [...el.querySelectorAll('button,a,[role=button],[role=option]')].find((x) => r.test((x.innerText || '').trim()) || r.test((x.getAttribute('aria-label') || '').trim())); (inner || el).click(); return true; } return false; }, re.source, re.flags, sel);
      await new Promise((s) => setTimeout(s, wait)); return ok;
    };
    const type = async (p, sel, text) => { const el = await p.$(sel); if (!el) return false; try { await el.click(); } catch { await p.evaluate((e) => e.focus(), el); } await el.type(text); return true; };
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
    const fmt = (r) => `ox=${r.ox} wide=${r.wide} cut=${JSON.stringify(r.cut.slice(0, 3))} ell=${JSON.stringify(r.ell.slice(0, 3))} sx=${r.sx}`;
    const okOf = (r) => r.ox <= 0 && r.cut.length === 0 && r.ell.length === 0 && r.wide === 0;
    const run = async (role, route, tag, w, label, person, pre) => {
      errs.length = 0; await go(route); if (pre) await pre(page); await sleep(250);
      const r = { ...(await page.evaluate(AUDIT_SRC)), sx: await page.evaluate(() => document.querySelectorAll('[data-scroll-x]').length), wide: Math.max(0, (await page.evaluate(() => Math.max(innerWidth, document.documentElement.scrollWidth))) - w) };
      const blocked = await page.evaluate(() => /Bạn không có quyền mở trang này/.test(document.body.innerText));
      const f = !okOf(r) || errs.length ? await shot(`${role}${route}-${tag}-${label || 'audit'}`) : '';
      push(role, route, tag, label || 'AUDIT', okOf(r) && !errs.length, fmt(r) + (blocked ? ' [màn chặn]' : '') + (errs.length ? ` console=${errs[0]}` : ''), f);
      if (label) return; // phép đo dưới đây chỉ cho ma trận chính (không cho kịch bản / biên)
      // 00-AC15 TOUCH: 375/390, 12 route SV + /attendance, /inbox (GV)
      if ((tag === '375' || tag === '390') && ((role === 'student' && TOUCH_SV.includes(route)) || (role === 'teacher' && (route === '/attendance' || route === '/inbox')))) {
        const t = await page.evaluate(TOUCH_SRC); push(role, route, tag, 'TOUCH (00-AC15)', t.length === 0, JSON.stringify(t.slice(0, 4)) + (t.length > 4 ? ` +${t.length - 4}` : ''), t.length ? await shot(`${role}${route}-${tag}-touch`) : '');
      }
      // 00-AC11 LEFT: 240 ở 1440, 16 ở 390 (mọi route trừ /login)
      if ((tag === '1440' || tag === '390') && role !== '(không vai)') { const l = await page.evaluate(LEFT_SRC); const want = tag === '1440' ? 240 : 16; push(role, route, tag, `LEFT (00-AC11) = ${want}`, l === want, `left=${l}`); }
      // 00-AC12 thanh trên đặc, kể cả sau khi cuộn 200 px
      { await page.evaluate(() => window.scrollTo(0, 200)); await sleep(120); const h = await page.evaluate(() => { const e = document.querySelector('header'); if (!e) return null; const c = getComputedStyle(e); return { bg: c.backgroundColor, bf: c.backdropFilter || c.webkitBackdropFilter || 'none' }; });
        const ok = h && /^(rgb|lab|oklab|lch|oklch|color)\(/.test(h.bg) && !/\/\s*0\.|rgba\(/.test(h.bg) && (h.bf === 'none' || h.bf === ''); push(role, route, tag, 'HEADER đặc (00-AC12)', ok, JSON.stringify(h)); await page.evaluate(() => window.scrollTo(0, 0)); }
    };

    // ---------- 1. Ma trận AUDIT: mọi route × vai × bề rộng ----------
    if (!skipMatrix) for (const role of Object.keys(ROUTES)) {
      if (only && only !== role) continue;
      await setRole(role); await setVp(1440); await go(ROUTES[role][0]); await reset();
      const list = [...ROUTES[role], ...BLOCKED[role]].filter((r) => !routesOnly || routesOnly.includes(r));
      for (const r of list) for (const vp of VIEWPORTS) {
        if (vp.tag === '375' && !(role === 'student' || r === '/attendance' || r === '/inbox')) continue;
        await setVp(vp.w, vp.h);
        for (const q of states && !BLOCKED[role].includes(r) ? ['', '?state=empty', '?state=error'] : ['']) await run(role, r + q, vp.tag, vp.w);
      }
      for (const r of skipEdge ? [] : (EDGE_ROUTES[role] || []).filter((r) => !routesOnly || routesOnly.includes(r))) for (const e of EDGE_W) { await setVp(e.w, 900); await run(role, r, e.tag, e.w, 'AUDIT-biên'); }
    }
    // /login (không phiên): 1440, 390, 375
    if (!skipMatrix && !only) { await page.deleteCookie(...(await page.cookies())); for (const w of [1440, 390, 375]) { await setVp(w); await run('(không vai)', '/login', String(w), w); } }

    // ---------- 2. AUDIT sau tương tác ----------
    if (!skipMatrix && !skipScenarios) for (const s of SCENARIOS_META) {
      if (only && only !== s.role) continue;
      await setRole(s.role, s.person || 'sv-2');
      for (const vp of VIEWPORTS.filter((v) => s.vps.includes(v.tag))) {
        await setVp(vp.w, vp.h); await go(s.route); await reset(); await go(s.route);
        await run(s.role, s.route, vp.tag, vp.w, s.id, s.person, async (p) => { try { await scn[s.id](p); } catch (e) { push(s.role, s.route, vp.tag, s.id + ' · kịch bản lỗi', false, String(e.message).slice(0, 140)); } });
      }
    }

    // ---------- 3. Phép đo riêng của spec v5 (SRS 4.7) ----------
    if (!skipSpec && !only) { const ctx = { page, base, setRole, setVp, go, reset, push, sleep, shot, ROUTES }; if (specWhich !== "v51") await new Function('ctx', 'return (' + specSrc + ')(ctx)')(ctx); if (specWhich !== "v5") await new Function('ctx', 'return (' + spec51Src + ')(ctx)')(ctx); }
    return rows;
  }, { timeout: 280000, args: [{ base, out, only, skipSpec, skipMatrix, states, routesOnly, skipScenarios, skipEdge, specWhich, ROUTES, VIEWPORTS: VIEWPORTS.map(({ w, h, tag }) => ({ w, h, tag })), BLOCKED, EDGE_W, EDGE_ROUTES, SCENARIOS_META: SCENARIOS, AUDIT_SRC, TOUCH_SRC, LEFT_SRC, TOUCH_SV, specSrc: SPEC_CHECKS.toString(), spec51Src: SPEC_V51.toString(), scnSrc: SCN_RUNNERS.toString() }] });
  await tab.close();
  return rows;
}

// ---- Phép đo riêng: chạy TRONG trang (Puppeteer page), nhận ctx ----
export async function SPEC_CHECKS({ page, setRole, setVp, go, reset, push, sleep, shot }) {
  const NAMES = { teacher: ['TS. Lê Thu Hà', 'Giảng viên'], ta: ['Phạm Quốc Bảo', 'Trợ giảng'], admin: ['Đỗ Hoàng Nam', 'Admin'] };
  const SV = { 'sv-1': 'Nguyễn Minh Trung', 'sv-2': 'Trần Thu Uyên', 'sv-3': 'Lê Quang Huy', 'sv-4': 'Phạm Ngọc Linh' };
  const q = (fn, ...args) => page.evaluate(fn, ...args);
  const clickTxt = async (re, sel = 'button,a,[role=menuitem],[role=button],[role=tab],label,summary', wait = 300) => {
    const ok = await q((src, flags, s) => { const r = new RegExp(src, flags); const el = [...document.querySelectorAll(s)].find((e) => (r.test((e.innerText || '').trim()) || r.test((e.getAttribute('aria-label') || '').trim())) && e.getBoundingClientRect().width > 0); if (el) { const inner = [...el.querySelectorAll('button,a,[role=button],[role=option]')].find((x) => r.test((x.innerText || '').trim()) || r.test((x.getAttribute('aria-label') || '').trim())); (inner || el).click(); return true; } return false; }, re.source, re.flags, sel);
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
  const col = await q(() => { const i = document.querySelector('[data-part=brand] img'); const b = i && i.getBoundingClientRect(); const s = document.querySelector('[data-part=brand]').getBoundingClientRect(); return b && { w: Math.round(b.width), h: Math.round(b.height), cx: Math.round(b.left + b.width / 2), colW: Math.round((document.querySelector('aside') || s).getBoundingClientRect().width), colCx: Math.round(s.left + s.width / 2) }; });
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

// ---- Spec v5.1 (#19): phép đo riêng, chạy TRONG trang. Mỗi phép đo ghi rõ AC. ----
export async function SPEC_V51({ page, setRole, setVp, go, reset, push, sleep, shot, ROUTES }) {
  const q = (fn, ...a) => page.evaluate(fn, ...a);
  const chk = (role, route, w, what, ok, detail, f) => push(role, route, String(w), what, !!ok, detail, f);
  const clickTxt = async (re, sel = 'button,a,[role=menuitem],[role=button],[role=tab],label,summary,li', wait = 300) => {
    const ok = await q((src, fl, s) => { const r = new RegExp(src, fl); const el = [...document.querySelectorAll(s)].find((e) => (r.test((e.innerText || '').trim()) || r.test((e.getAttribute('aria-label') || '').trim())) && e.getBoundingClientRect().width > 0); if (el) { const inner = [...el.querySelectorAll('button,a,[role=button],[role=option]')].find((x) => r.test((x.innerText || '').trim()) || r.test((x.getAttribute('aria-label') || '').trim())); (inner || el).click(); return true; } return false; }, re.source, re.flags, sel);
    await sleep(wait); return ok;
  };
  const ENG = /This page (could not be found|couldn.t load)|Application error|Internal Server Error/;
  const txt = () => q(() => document.body.innerText);
  const parts = (p) => q((s) => [...document.querySelectorAll('[data-part=' + s + ']')].filter((e) => e.getBoundingClientRect().width > 0).length, p);

  // 00-AC13 / FR-X16: 404 + lỗi chạy bằng tiếng Việt, không lộ trang mặc định của Next.js
  for (const [role, person] of [['teacher', ''], ['student', 'sv-2'], ['ta', ''], ['admin', '']]) {
    await setRole(role, person || 'sv-2'); await setVp(1440, 900);
    await go('/khong-co-trang'); const t = await txt();
    chk(role, '/khong-co-trang', 1440, '00-AC13 404 tiếng Việt: "Không tìm thấy trang" + Về Hôm nay', /Không tìm thấy trang/.test(t) && /Về Hôm nay/.test(t) && !ENG.test(t), t.replace(/\s+/g, ' ').slice(0, 120), await shot(`404-${role}`));
    for (const p of ['/threads/khong-co', '/assignments/khong-co', '/practice/khong-co', '/students/sv-999']) { await go(p); const n = await txt(); chk(role, p, 1440, '00-AC13 id lạ: không trang tiếng Anh, không trắng', !ENG.test(n) && n.trim().length > 30, n.replace(/\s+/g, ' ').slice(0, 100)); }
  }
  await setRole('student', 'sv-2'); await go('/');
  for (const [bad, strict] of [['{', true], ['null', true], ['[]', true], ['{"x":1}', true], ['{"practice":"x","tickets":5,"bell":7}', false]]) {
    await q((v) => localStorage.setItem('ep_demo_state', v), bad);
    for (const r of ['/', '/practice', '/chat', '/threads', '/me', '/calendar']) { await go(r); const t = await txt();
      const sick = /Trang này gặp sự cố/.test(t); chk('student', r, 1440, `00-AC13 ep_demo_state hỏng ${bad.slice(0, 14)}`, !ENG.test(t) && t.trim().length > 30 && (strict ? !sick : !sick || (/Thử lại/.test(t) && /Về Hôm nay/.test(t))), sick ? 'màn lỗi app' : 'dùng dữ liệu gốc'); }
  }
  await q(() => localStorage.removeItem('ep_demo_state'));

  // 01-AC17 / FR-X18: SV D chưa vào lớp
  await setRole('student', 'sv-4'); await setVp(1440, 900); await go('/'); await reset(); await go('/');
  for (const r of ['/chat', '/threads', '/threads/t-cbc', '/practice', '/practice/at-symmetric', '/practice/history', '/library', '/calendar', '/me', '/assignments/bt03']) {
    await go(r); const m = await q(() => { const t = document.body.innerText; return { url: location.pathname, title: /Bạn chưa vào lớp nào/.test(t), sentence: /Nhập mã tham gia do giảng viên cung cấp để dùng Chat riêng, Threads, Luyện đề, Thư viện và Lịch\./.test(t), blocked: /Bạn không có quyền mở trang này/.test(t), leak: /phiên trước|Cách chọn độ dài khoá RSA|Nộp muộn Bài tập 03|CBC khác ECB|Chương 3 — Mật mã/i.test(t), input: !!document.querySelector('input'), go: [...document.querySelectorAll('button,a')].some((b) => /^Tiếp tục$/.test(b.innerText.trim())) }; });
    chk('student:sv-4', r, 1440, '01-AC17 D: một màn "Bạn chưa vào lớp nào", URL giữ nguyên, không rò dữ liệu', m.title && m.sentence && !m.blocked && !m.leak && m.input && m.go && m.url === r, JSON.stringify(m));
  }
  await go('/'); { const n = await q(() => [...document.querySelectorAll('nav a')].filter((a) => a.offsetParent).map((a) => a.innerText.trim()).filter(Boolean)); chk('student:sv-4', '/', 1440, '01-AC17 sidebar chỉ có Hôm nay', n.length === 1 && /Hôm nay/.test(n[0]), JSON.stringify(n), await shot('D-home-1440')); }
  await setVp(390, 844); await go('/'); { const n = await q(() => [...document.querySelectorAll('nav a')].filter((a) => a.offsetParent).map((a) => a.innerText.trim()).filter(Boolean)); chk('student:sv-4', '/', 390, '01-AC17 thanh dưới chỉ có Hôm nay', n.length === 1, JSON.stringify(n), await shot('D-home-390')); }

  // 01-AC19 form tạo thread
  await setRole('student', 'sv-2'); await setVp(1440, 900); await go('/threads'); await reset(); await go('/threads');
  { const top = await q(() => { const r = document.querySelector('[data-part=thread-row]'); return r ? Math.round(r.getBoundingClientRect().top) : null; });
    chk('student', '/threads', '1440×900', '01-AC21 hàng thread đầu cách đỉnh ≤ 420 khi form chưa mở', top !== null && top <= 420, `top=${top}`, await shot('threads-1440')); }
  await clickTxt(/^Đặt câu hỏi$/, 'button', 400);
  { const o = await q(() => { const s = document.querySelector('[data-part=thread-form] select'); return s ? [...s.options].map((x) => x.textContent.trim()) : null; });
    chk('student', '/threads', 1440, '01-AC19 Chủ đề: đầu "Chọn chủ đề", không có "Thông báo"', o && o[0] === 'Chọn chủ đề' && !o.includes('Thông báo'), JSON.stringify(o));
    const hint = () => q(() => { const b = [...document.querySelectorAll('[data-part=thread-form] button')].find((x) => /^Đăng câu hỏi$/.test(x.innerText.trim())); if (!b) return null; const id = b.getAttribute('aria-describedby'); const d = id && document.getElementById(id); return { disabled: b.disabled, hint: d ? d.innerText.trim() : null }; });
    const h0 = await hint(); chk('student', '/threads', 1440, '01-AC21 "Đăng câu hỏi" khoá + "Còn thiếu: tiêu đề, chủ đề, nội dung"', h0 && h0.disabled && h0.hint === 'Còn thiếu: tiêu đề, chủ đề, nội dung', JSON.stringify(h0));
    const inp = await page.$('[data-part=thread-form] input[type=text], [data-part=thread-form] input:not([type]):not([type=checkbox])'); if (inp) { await inp.click(); await inp.type('Hỏi về IV'); }
    const h1 = await hint(); chk('student', '/threads', 1440, '01-AC21 dòng nhắc thu hẹp: "Còn thiếu: chủ đề, nội dung"', h1 && h1.disabled && h1.hint === 'Còn thiếu: chủ đề, nội dung', JSON.stringify(h1));
    const all = await q(() => [...document.querySelectorAll('button:disabled')].every((b) => b.getAttribute('aria-describedby') && document.getElementById(b.getAttribute('aria-describedby'))?.innerText.trim()));
    chk('student', '/threads', 1440, '01-AC21 mọi nút đang khoá có aria-describedby trỏ tới dòng chữ', all, '');
  }
  await setRole('teacher'); await go('/threads'); await clickTxt(/^Đặt câu hỏi$/, 'button', 400);
  { const o = await q(() => { const s = document.querySelector('[data-part=thread-form] select'); return s ? [...s.options].map((x) => x.textContent.trim()) : null; }); chk('teacher', '/threads', 1440, '01-AC19 GV có "Thông báo" trong Chủ đề', o && o.includes('Thông báo'), JSON.stringify(o)); }

  // 01-AC22 chat 390: Phiên trước (4)
  await setRole('student', 'sv-2'); await setVp(390, 844); await go('/chat'); await reset(); await go('/chat');
  { const b = await q(() => { const e = document.querySelector('[data-part=chat-sessions-button]'); if (!e) return null; const r = e.getBoundingClientRect(); return { h: Math.round(r.height), t: e.innerText.trim() }; });
    chk('student', '/chat', 390, '01-AC22 nút "Phiên trước (4)" cao ≥ 44', b && b.h >= 44 && /Phiên trước \(4\)/.test(b.t), JSON.stringify(b), await shot('chat-sessions-btn-390'));
    await clickTxt(/Phiên trước \(4\)/, 'button', 500);
    const sheet = await q(() => { const d = [...document.querySelectorAll('[role=dialog],[popover],[class*=sheet i],[class*=drawer i]')].find((e) => /Cách chọn độ dài khoá RSA/.test(e.innerText)); return d ? { n: ['Cách chọn độ dài khoá RSA', 'Nộp muộn Bài tập 03', 'Hàm băm SHA-256', 'Phân biệt'].filter((s) => d.innerText.includes(s)).length, newBtn: /Phiên mới/.test(d.innerText) } : null; });
    chk('student', '/chat', 390, '01-AC22 bảng liệt kê 4 phiên + Phiên mới', sheet && sheet.n === 4 && sheet.newBtn, JSON.stringify(sheet), await shot('chat-sessions-sheet-390'));
    await clickTxt(/Cách chọn độ dài khoá RSA/, '[role=dialog] *,[popover] *,button,li,a', 600); const t = await txt();
    chk('student', '/chat', 390, '01-AC22 chọn phiên: xem chỉ đọc + nút "Hỏi tiếp"', /Hỏi tiếp/.test(t), '');
    await setVp(1100, 900); await go('/chat'); const hp = await parts('chat-history'); const bp = await parts('chat-sessions-button'); chk('student', '/chat', 1100, '01-AC22 ≥ 1100 px: panel chat-history, không nút Phiên trước', hp >= 1 && bp === 0, `chat-history=${hp} nút=${bp}`);
    // PM #20(b): 720–1099 px như 390 px — nút `Phiên trước (n)` ≥ 44, không panel, hội thoại + composer cùng trục
    for (const w of [1099, 900, 720]) { await setVp(w, 900); await go('/chat'); const m = await q(() => { const R = (p) => { const e = document.querySelector('[data-part=' + p + ']'); return e && e.getBoundingClientRect(); }; const t = R('chat-thread'), c = R('chat-composer'), h = R('chat-history'), b = R('chat-sessions-button'); return { btn: !!b && b.width > 0, btnH: b ? Math.round(b.height) : null, hist: !!h && h.width > 0, sameLeft: !!t && !!c && t.left === c.left, sameW: !!t && !!c && t.width === c.width, ox: document.documentElement.scrollWidth - innerWidth }; });
      chk('student', '/chat', w, '01-AC22 (#20b) 720–1099: nút Phiên trước ≥ 44, không panel, cùng trục', m.btn && m.btnH >= 44 && !m.hist && m.sameLeft && m.sameW && m.ox <= 0, JSON.stringify(m), await shot(`chat-${w}`)); } }

  // 01-AC27 thẻ "Câu hỏi gốc"
  for (const [role, person] of [['student', 'sv-2'], ['ta', ''], ['teacher', ''], ['student', 'sv-1']]) { await setRole(role, person || 'sv-2'); await setVp(1440, 900);
    for (const th of ['t-cbc', 't-salt', 't-wifi']) { await go('/threads/' + th); const s = await q(() => { const c = document.querySelector('[data-part=thread-question]'); return c && c.lastElementChild ? Math.round(c.getBoundingClientRect().bottom - c.lastElementChild.getBoundingClientRect().bottom) : null; });
      chk(role + (person ? ':' + person : ''), '/threads/' + th, 1440, '01-AC27 đáy thẻ câu hỏi gốc ≤ 21 px', s !== null && s <= 21, `slack=${s}`); } }

  // 02-AC15 /attendance 375 / 390
  await setRole('teacher');
  for (const w of [375, 390]) { await setVp(w, 812); await go('/attendance');
    const m = await q(() => { const R = (e) => { const b = e.getBoundingClientRect(); return { l: b.left, r: b.right, t: b.top, b: b.bottom, w: b.width, h: b.height }; };
      const sel = document.querySelector('[data-part=att-session-select]'); const tog = [...document.querySelectorAll('main label,main button,main [role=switch]')].find((e) => /Giả lập mất mạng/.test(e.innerText)); const rows = [...document.querySelectorAll('[data-part=att-row]')].slice(0, 3);
      const inter = (a, b) => !(a.r <= b.l || b.r <= a.l || a.b <= b.t || b.b <= a.t);
      const out = { sel: sel ? Math.round(R(sel).w) : null, selText: sel ? (sel.options ? sel.options[sel.selectedIndex].text : sel.innerText.trim()) : null, overlap: sel && tog ? inter(R(sel), R(tog)) : null, ox: document.documentElement.scrollWidth - innerWidth, rows: [] };
      for (const row of rows) { const btns = [...row.querySelectorAll('button,[role=radio],label')].filter((b) => /Có mặt|Muộn|Vắng phép|Vắng|Phát biểu/.test(b.innerText || b.getAttribute('aria-label') || '') && b.getBoundingClientRect().width > 0); const small = btns.filter((b) => R(b).w < 44 || R(b).h < 44).map((b) => (b.innerText || b.getAttribute('aria-label')).trim().slice(0, 12)); const name = row.querySelector('[data-part=att-name],strong,b,span'); out.rows.push({ n: btns.length, small, inW: btns.every((b) => R(b).r <= innerWidth + 1), nameW: name ? Math.round(R(name).w) : null, h: Math.round(R(row).h) }); }
      const sum = [...document.querySelectorAll('main *')].find((e) => /\d+ có mặt/.test(e.textContent) && !e.children.length); out.sumHeight = sum ? Math.round(R(sum).h) : null; return out; });
    chk('teacher', '/attendance', w, '02-AC15 ô Buổi ≥ 160 hiện "Buổi 10 · 29/10"; không đè công tắc; ox 0', m.sel >= 160 && /Buổi 10 · 29\/10/.test(m.selText || '') && m.overlap === false && m.ox <= 0, JSON.stringify({ sel: m.sel, t: m.selText, overlap: m.overlap, ox: m.ox }), await shot(`att-${w}`));
    chk('teacher', '/attendance', w, '02-AC15 mỗi SV: 5 nút ≥ 44 × 44, trong bề rộng màn, tên ≥ 140, đủ trong một màn hình', m.rows.length === 3 && m.rows.every((r) => r.n >= 5 && r.small.length === 0 && r.inW && (r.nameW === null || r.nameW >= 140) && r.h < 300), JSON.stringify(m.rows)); }

  // 02-AC18..AC19 (+N3, N4) danh sách sinh viên
  await setRole('teacher'); await setVp(1440, 900); await go('/students');
  { const m = await q(() => { const chip = [...document.querySelectorAll('main button')].find((b) => /^Cần chú ý\s*\d+/.test(b.innerText.trim())); const n = chip ? +chip.innerText.match(/\d+/)[0] : null; return { n, text: document.body.innerText }; });
    const hasOld = /Theo dõi/.test(m.text);
    await clickTxt(/^Cần chú ý\s*\d+/, 'main button', 400);
    const rows = await q(() => { const r = [...document.querySelectorAll('[data-part=student-row]')]; return { n: r.length, flagged: r.filter((x) => x.innerText.includes('Cần chú ý')).length }; });
    chk('teacher', '/students', 1440, '02-AC18 chip Cần chú ý = số hàng sau lọc = số hàng ghi "Cần chú ý" (= 8); không còn "Theo dõi"', m.n === 8 && rows.n === 8 && rows.flagged === 8 && !hasOld, JSON.stringify({ chip: m.n, rows, theoDoi: hasOld }), await shot('students-watch-1440')); }
  const ids = async (route) => { await setVp(1440, 900); await go(route); await reset(); await go(route); return q(() => [...document.querySelectorAll('[data-part=student-row]')].map((r) => r.dataset.studentId)); };
  { const [a, b, c, d] = [await ids('/students'), await ids('/attendance'), await ids('/gradebook'), await ids('/class/members')]; const asc = (x) => x.every((v, i) => i === 0 || +v.split('-')[1] > +x[i - 1].split('-')[1]);
    chk('teacher', '/students,/attendance,/gradebook,/class/members', 1440, '02-AC19 cùng thứ tự sv-n tăng dần (30 hàng, A–C đầu)', [a, b, c, d].every((x) => x.length >= 30 && x.join() === a.join() && asc(x)) && a.slice(0, 3).join() === 'sv-1,sv-2,sv-3', JSON.stringify({ n: [a.length, b.length, c.length, d.length], head: a.slice(0, 4) })); }
  await go('/'); { const t = await txt(); chk('teacher', '/', 1440, '02-AC18 Hôm nay: "Lớp cần chú ý · 8 sinh viên" + Xem cả 8', /Lớp cần chú ý · 8 sinh viên/.test(t) && /Xem cả 8/.test(t), ''); }
  await go('/'); { const ids2 = await q(() => [...document.querySelectorAll('[data-part=today-task]')].map((e) => e.dataset.taskId)); chk('teacher', '/', 1440, '02-AC21 thứ tự thẻ Hôm nay (6 việc seed): ticket, …, ai-pending, setup, members', ids2.length === 6 && /ticket|inbox/.test(ids2[0]) && ids2[3] === 'ai-pending' && ids2[4] === 'setup' && ids2[5] === 'members', JSON.stringify(ids2)); }

  // 02-AC14 (N9) badge điều hướng
  for (const w of [1440, 390]) { await setVp(w, 844); for (const r of ['/', '/inbox', '/gradebook', '/grading']) { await go(r); const b = await q(() => [...document.querySelectorAll('[data-part^=nav-badge]')].filter((e) => e.offsetParent).map((e) => e.dataset.part + ':' + e.textContent.trim()));
      chk('teacher', r, w, '02-AC14 badge Hộp thư 5 và Chấm bài 4', b.includes('nav-badge-inbox:5') && b.includes('nav-badge-grading:4') && new Set(b).size === 2, JSON.stringify(b)); } }

  // 04-AC8 (N1, N2) analytics ↔ inbox
  await go('/inbox'); const over = await q(() => (document.body.innerText.match(/Quá 24 giờ/g) || []).length);
  await go('/analytics'); { const t = await txt(); const n24 = +(t.match(/Câu chờ quá 24 giờ[\s\S]{0,30}?(\d+)/) || [])[1];
    chk('teacher', '/inbox ↔ /analytics', 1440, '04-AC8 "Câu chờ quá 24 giờ" = 3 = số hàng "Quá 24 giờ"', n24 === 3 && over === 3, JSON.stringify({ n24, over }));
    chk('teacher', '/analytics', 1440, '04-AC8 "AI tự trả lời 98%" trong 392 câu, 6 chuyển', /AI tự trả lời 98%/.test(t) && /392 câu/.test(t) && /6 câu chuyển/.test(t), (t.match(/AI tự trả lời[^\n]*/) || [''])[0]);
    await clickTxt(/^30 ngày$/, 'button,[role=radio],label', 400); const t30 = await txt();
    chk('teacher', '/analytics', 1440, '04-AC8 30 ngày: 99% trong 1.424 câu, 19 chuyển; "chờ quá 24 giờ" vẫn 3', /99%/.test(t30) && /1\.424/.test(t30) && /19/.test(t30) && +(t30.match(/Câu chờ quá 24 giờ[\s\S]{0,30}?(\d+)/) || [])[1] === 3, '');
    // 04-AC12 biểu đồ
    const ch30 = await q(() => [document.querySelectorAll('[data-part=chart-point]').length, document.querySelectorAll('[data-part=chart-axis-label]').length]);
    await clickTxt(/^7 ngày$/, 'button,[role=radio],label', 400); const ch7 = await q(() => [document.querySelectorAll('[data-part=chart-point]').length, document.querySelectorAll('[data-part=chart-axis-label]').length, [...document.querySelectorAll('[data-part=chart-point]')].every((p) => p.getAttribute('aria-label'))]);
    chk('teacher', '/analytics', 1440, '04-AC12 7 ngày: 7 điểm, ≥ 3 nhãn trục, mỗi điểm có aria-label; 30 ngày: 10 điểm', ch7[0] === 7 && ch7[1] >= 3 && ch7[2] && ch30[0] === 10 && ch30[1] >= 3, JSON.stringify({ ch7, ch30 }), await shot('analytics-chart')); }

  // 04-AC11 khoảng cách các phần của /settings/llm; 03-AC7 /gradebook 390
  await setRole('admin'); await setVp(1440, 900); await go('/settings/llm');
  { const g = await q(() => [...document.querySelectorAll('[data-part=settings-section]')].map((s, i, a) => i ? Math.round(s.getBoundingClientRect().top - a[i - 1].getBoundingClientRect().bottom) : null).slice(1)); chk('admin', '/settings/llm', 1440, '04-AC11 khoảng cách các phần bằng nhau (lệch 0)', g.length >= 3 && new Set(g).size === 1, JSON.stringify(g), await shot('llm-sections')); }
  await setRole('teacher'); await setVp(390, 844); await go('/gradebook');
  { const m = await q(() => ({ inView: ['col-qt', 'col-status'].map((p) => { const e = document.querySelector('[data-part=' + p + ']'); return e ? e.getBoundingClientRect().right <= innerWidth : null; }), hint: /Kéo ngang để xem BT01–BT03 và Cuối kỳ/.test(document.body.innerText) }));
    chk('teacher', '/gradebook', 390, '03-AC7 col-qt, col-status trong khung nhìn đầu + dòng gợi ý kéo ngang', m.inView.every((x) => x === true) && m.hint, JSON.stringify(m), await shot('gradebook-v51-390')); }
}

export function table(rows) {
  const esc = (s) => String(s).replace(/\|/g, '\\|').replace(/\n/g, ' ');
  return ['| vai | route | rộng | phép đo | kết quả | chi tiết | ảnh |', '| --- | --- | --- | --- | --- | --- | --- |'].concat(rows.map((r) => `| ${r.role} | \`${r.route}\` | ${r.w} | ${esc(r.what)} | ${r.ok ? 'PASS' : '**FAIL**'} | ${esc(r.detail).slice(0, 220)} | ${r.file ? r.file.split('/').pop() : ''} |`)).join('\n');
}
export function summary(rows) {
  const bad = rows.filter((r) => !r.ok);
  return `AUDIT: ${rows.length - bad.length}/${rows.length} PASS; FAIL ${bad.length}\n` + bad.map((r) => `  FAIL ${r.role} ${r.route} @${r.w} · ${r.what} · ${r.detail.slice(0, 120)}`).join('\n');
}
