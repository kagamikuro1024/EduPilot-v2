// QC — AUDIT bố cục bằng ĐĂNG NHẬP THẬT (US-P2-12 AC10, TC-P212-21). Thay audit.mjs / sweep.mjs ở phần cookie `ep_demo_*` (đã bỏ).
// Dùng trong Eval (JS) với tab `t` (SHIM2.open / browser.open tới Chrome), seed đã chạy, frontend `next start`:
//   const m = await import('/abs/docs/sprints/4/qc/scripts/audit-login.mjs');
//   const rows = await m.run(t, { base: 'http://localhost:3400', role: 'student', email: 'sv.gioi@edupilot.local', password: PW });
//   console.log(m.summary(rows));
// Chỉ phần MA TRẬN AUDIT (ox / cut / ell) + TOUCH (≤ 390) + chuỗi cấm + màn "Không tìm thấy"; KHÔNG gồm phép đo SPEC riêng của audit.mjs.
import { AUDIT_SRC, TOUCH_SRC } from '../../../1.5/qc/scripts/audit.mjs';
import { ROUTES, VIEWPORTS } from '../../../1.5/qc/scripts/sweep.mjs';

const FORBIDDEN = /\b(RAG|PII|fallback|trace|provider|confidence|redaction|embedding|prompt|LLM|placeholder)\b|độ tin cậy|cần chú ý|rủi ro|\[\[SV_/i;
const sleep = (ms) => new Promise((r) => setTimeout(r, ms));

export async function login(t, base, email, password) {
  await t.run(async ({ page }) => { const c = await page.createCDPSession(); await c.send('Network.clearBrowserCookies'); });
  await t.goto(base + '/login');
  await t.evaluate('localStorage.clear()');
  for (let i = 0; i < 40; i++) { if (await t.evaluate("!!document.querySelector('input[autocomplete=username]')")) break; await sleep(250); }
  await t.evaluate(`(()=>{const s=(q,v)=>{const i=document.querySelector(q);Object.getOwnPropertyDescriptor(HTMLInputElement.prototype,'value').set.call(i,v);i.dispatchEvent(new Event('input',{bubbles:true}));};s('input[autocomplete=username]',${JSON.stringify(email)});s('input[autocomplete=current-password]',${JSON.stringify(password)});[...document.querySelectorAll('button')].find(b=>/^Đăng nhập$/.test(b.innerText.trim())).click();})()`);
  await sleep(3000);
}

export async function run(t, { base, role, email, password, routes }) {
  await login(t, base, email, password);
  const rows = [];
  for (const vp of VIEWPORTS) {
    await t.run(new Function('ctx', `return (async ({ page }) => { await page.setViewport({ width: ${vp.w}, height: ${vp.h}, isMobile: ${vp.w < 500}, hasTouch: ${vp.w < 500} }); })(ctx)`));
    for (const r of routes || ROUTES[role]) {
      if (vp.only && !vp.only(role, r)) continue;
      try {
        await t.goto(base + r);
        await sleep(1700);
        const a = await t.evaluate(AUDIT_SRC);
        const touch = vp.w <= 390 && (role === 'student' || r === '/attendance' || r === '/inbox') ? await t.evaluate(TOUCH_SRC) : []; // phạm vi TOUCH của 00-AC15 (SV + /attendance, /inbox)
        const txt = await t.evaluate('document.body.innerText');
        const notFound = /Không tìm thấy trang/.test(txt);
        const forb = role === 'student' ? (txt.match(new RegExp(FORBIDDEN.source, 'gi')) || []).length : 0; // chuỗi cấm chỉ áp cho Sinh viên (như sweep.mjs only:student)
        const pass = a.ox <= 0 && a.cut.length === 0 && a.ell.length === 0 && forb === 0 && !notFound && touch.length === 0;
        rows.push({ role, route: r, w: vp.tag, pass, ox: a.ox, cut: a.cut.length, ell: a.ell.length, touch: touch.length, forb, notFound });
      } catch (e) {
        rows.push({ role, route: r, w: vp.tag, pass: false, err: String(e).slice(0, 80) });
      }
    }
  }
  await t.run(async ({ page }) => { await page.setViewport({ width: 1440, height: 900 }); });
  return rows;
}

export const summary = (rows) => `AUDIT: ${rows.filter((x) => x.pass).length}/${rows.length} PASS; FAIL ${rows.filter((x) => !x.pass).length}; FORBIDDEN=${rows.reduce((s, x) => s + (x.forb || 0), 0)}\n` +
  rows.filter((x) => !x.pass).map((x) => `  FAIL ${x.role} ${x.route} @${x.w}: ${x.err || JSON.stringify({ ox: x.ox, cut: x.cut, ell: x.ell, touch: x.touch, forb: x.forb, nf: x.notFound })}`).join('\n');

// Ma trận vai × route (tương đương `tc_00_matrix` của proto-curl.sh, bản đăng nhập thật; SRS mục 2). Trả về hàng {role, route, want, got, ok}.
const G = {
  ALL: ['/'],
  SV: ['/chat', '/practice', '/practice/at-symmetric', '/practice/at-quiz01', '/practice/history', '/library', '/me', '/assignments/bt03', '/join', '/join/BX4P9TW'],
  SV_TA_GV: ['/threads', '/threads/t-cbc', '/calendar'],
  TA_GV: ['/inbox', '/students', '/students/sv-3', '/attendance', '/class/members', '/gradebook', '/gradebook/scheme', '/grading', '/grading/sub-bt03-sv-2', '/questions', '/documents', '/insights', '/analytics'],
  GV_AD: ['/observability', '/settings/llm', '/settings/integrations'],
  ADMIN: ['/admin/courses', '/admin/users'],
};
const ALLOW = { student: ['ALL', 'SV', 'SV_TA_GV'], ta: ['ALL', 'SV_TA_GV', 'TA_GV'], teacher: ['ALL', 'SV_TA_GV', 'TA_GV', 'GV_AD'], admin: ['ALL', 'GV_AD', 'ADMIN'] };
export async function matrix(t, { base, role, email, password }) {
  await login(t, base, email, password);
  const rows = [];
  for (const [g, routes] of Object.entries(G)) for (const r of routes) {
    const want = ALLOW[role].includes(g) ? 'MO' : 'CHAN';
    await t.goto(base + r);
    await sleep(1500);
    const txt = await t.evaluate('document.body.innerText');
    const got = /Bạn không có quyền (mở trang|xem màn) này/.test(txt) ? 'CHAN' : 'MO';
    rows.push({ role, route: r, want, got, ok: want === got });
  }
  return rows;
}
