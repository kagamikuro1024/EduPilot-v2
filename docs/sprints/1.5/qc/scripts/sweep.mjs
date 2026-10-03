// QC — duyệt mọi route × vai × khung nhìn, chụp ảnh, đo lỗi cơ học. Chạy trong Eval (JS) có global `browser`:
//   const run = (await import('/abs/path/docs/sprints/1.5/qc/scripts/sweep.mjs')).default;
//   const res = await run(browser, { base:'http://localhost:3000', only:'student', out:'<abs>/qc/shots/sweep' });
// Không thêm phụ thuộc vào frontend/; dùng Puppeteer `page` có sẵn của `browser`.
export const ROUTES = {
  student: ['/', '/chat', '/threads', '/threads/t-cbc', '/practice', '/practice/at-symmetric', '/practice/at-quiz01', '/practice/history', '/library', '/calendar', '/me', '/assignments/bt03', '/join', '/join/BX4P9TW'],
  ta: ['/', '/threads', '/calendar', '/inbox', '/students', '/students/sv-3', '/attendance', '/class/members', '/gradebook', '/gradebook/scheme', '/grading', '/grading/sub-bt03-sv-2', '/questions', '/documents', '/insights', '/analytics'],
  teacher: ['/', '/threads', '/threads/t-cbc', '/calendar', '/inbox', '/students', '/students/sv-3', '/attendance', '/class/members', '/gradebook', '/gradebook/scheme', '/grading', '/grading/sub-bt03-sv-2', '/questions', '/documents', '/insights', '/analytics', '/observability', '/settings/llm', '/settings/integrations'],
  admin: ['/', '/observability', '/settings/llm', '/settings/integrations', '/admin/courses', '/admin/users'],
};
// khung nhìn: desktop 1440; điện thoại 390 (mọi route); 375 chỉ cho màn bắt buộc "dùng tốt" (SRS 7)
export const VIEWPORTS = [{ w: 1440, h: 900, tag: '1440' }, { w: 390, h: 844, tag: '390' }, { w: 375, h: 812, tag: '375', only: (role, r) => role === 'student' || r === '/attendance' || r === '/inbox' }];
const FORBIDDEN = /\b(RAG|PII|fallback|trace|provider|confidence|redaction|embedding|prompt|LLM|placeholder)\b|độ tin cậy|cần chú ý|rủi ro|\[\[SV_/i;

export default async function sweep(browser, { base = 'http://localhost:3000', only, out, person = 'sv-2', course = 'int1006-1', states = false } = {}) {
  const results = [];
  const tab = await browser.open({ name: 'qc-sweep', url: base + '/login', viewport: { width: 1440, height: 900 } });
  const got = await tab.run(async ({ page }, a) => {
    const { base, only, out, person, course, states, ROUTES, VIEWPORTS, FORBIDDEN } = a;
    const results = [];
    const re = new RegExp(FORBIDDEN.source, FORBIDDEN.flags);
    const fs = await import('node:fs');
    if (out) fs.mkdirSync(out, { recursive: true });
    const errs = [];
    page.on('console', (m) => { if (m.type() === 'error') errs.push(m.text()); });
    page.on('pageerror', (e) => errs.push('pageerror: ' + e.message));
    for (const role of Object.keys(ROUTES)) {
      if (only && only !== role) continue;
      await page.deleteCookie(...(await page.cookies()));
      for (const [name, value] of [['ep_demo_role', role], ['ep_demo_person', role === 'student' ? person : ''], ['ep_demo_course', course]]) await page.setCookie({ name, value, url: base });
      for (const r of ROUTES[role]) for (const q of states ? ['', '?state=loading', '?state=empty', '?state=error'] : ['']) {
        for (const vp of VIEWPORTS) {
          if (vp.only && !(role === 'student' || r === '/attendance' || r === '/inbox')) continue;
          errs.length = 0;
          await page.setViewport({ width: vp.w, height: vp.h, deviceScaleFactor: 1, isMobile: vp.w < 500, hasTouch: vp.w < 500 });
          const resp = await page.goto(base + r + q, { waitUntil: 'networkidle0', timeout: 30000 }).catch((e) => ({ status: () => 'ERR ' + e.message }));
          await new Promise((s) => setTimeout(s, 400));
          const m = await page.evaluate((isStudent) => {
            const sw = document.documentElement.scrollWidth, iw = window.innerWidth;
            const text = document.body.innerText || '';
            const small = [];
            if (isStudent) for (const el of document.querySelectorAll('a[href],button,input,select,textarea,[role=button],[role=tab]')) {
              const b = el.getBoundingClientRect(); const cs = getComputedStyle(el);
              if (b.width === 0 || b.height === 0 || cs.visibility === 'hidden') continue;
              if (b.height < 44 - 0.5 || b.width < 44 - 0.5) small.push(`${el.tagName.toLowerCase()}:${(el.innerText || el.getAttribute('aria-label') || '').trim().slice(0, 24)}:${Math.round(b.width)}x${Math.round(b.height)}`);
            }
            const clipped = []; for (const el of document.querySelectorAll('h1,h2,h3,td,th,button,a,p,li,label')) { if (el.scrollWidth > el.clientWidth + 2 && getComputedStyle(el).overflowX === 'visible' && el.clientWidth > 0 && el.getBoundingClientRect().right > iw + 1) clipped.push((el.innerText || '').trim().slice(0, 30)); }
            return { overflowX: sw > iw + 1, sw, iw, h1: document.querySelectorAll('h1').length, hasBanner: /Bản mô phỏng/.test(text), blocked: /Bạn không có quyền mở trang này/.test(text), spinner: !!document.querySelector('[class*=spinner i],[class*=Spinner],[role=progressbar]'), text, small, clipped };
          }, role === 'student');
          const file = out ? `${out}/${role}${r.replace(/[\/\[\]?=]/g, '_')}${q.replace(/[?=]/g, '_')}-${vp.tag}.png` : null;
          if (file) await page.screenshot({ path: file, fullPage: false });
          const forbid = role === 'student' ? (m.text.match(re) || [])[0] || null : null;
          results.push({ role, route: r + q, vp: vp.tag, status: typeof resp.status === 'function' ? resp.status() : resp.status, h1: m.h1, overflowX: m.overflowX, blocked: m.blocked, banner: m.hasBanner, spinner: m.spinner, forbidden: forbid, smallTargets: m.small.slice(0, 6), nSmall: m.small.length, clipped: m.clipped.slice(0, 3), consoleErrors: errs.slice(0, 3), file });
        }
      }
    }
    return results;
  }, { timeout: 280000, args: [{ base, only, out, person, course, states, ROUTES, VIEWPORTS: VIEWPORTS.map(({ w, h, tag, only }) => ({ w, h, tag, only: !!only })), FORBIDDEN: { source: FORBIDDEN.source, flags: FORBIDDEN.flags } }] });
  await tab.close();
  return got; // QC s3: tab.run trả giá trị (đối số không được sao ngược lại)
}
export function summarize(results) {
  const bad = results.filter((r) => r.status >= 400 || (typeof r.status === 'string') || r.overflowX || r.forbidden || r.consoleErrors.length || r.h1 !== 1 || r.nSmall > 0 || r.clipped.length);
  return { total: results.length, bad: bad.map((r) => `${r.role} ${r.route} @${r.vp}: ` + [r.status >= 400 && `HTTP ${r.status}`, r.overflowX && 'cuộn ngang', r.forbidden && `từ cấm "${r.forbidden}"`, r.consoleErrors.length && `console: ${r.consoleErrors[0].slice(0, 80)}`, r.h1 !== 1 && `h1=${r.h1}`, r.nSmall > 0 && `${r.nSmall} vùng chạm <44px (${r.smallTargets.slice(0, 2).join('; ')})`, r.clipped.length && `chữ tràn: ${r.clipped[0]}`].filter(Boolean).join(', ')) };
}
