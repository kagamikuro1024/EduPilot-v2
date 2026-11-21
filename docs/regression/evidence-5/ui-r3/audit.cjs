// smkP: cổng AUDIT bốn vai bằng đăng nhập thật trên ep-r (cổng Playwright của docs/sprints/4/qc/scripts/audit-login.mjs: cùng AUDIT_SRC / TOUCH_SRC / chuỗi cấm / "Không tìm thấy trang").
// Tuyến lấy từ sweep.mjs (bỏ tuyến có tham số demo cũ như t-cbc, sv-3) + tuyến PE (/exams, /questions). Chạy: ROLES=student,ta,teacher,admin node smkP_audit.cjs
const L = require('./ui_lib.cjs'); const { chromium, B, PW, AUDIT_SRC, TOUCH_SRC, fs } = L;
const ACC = { student: 'sv.gioi', ta: 'ta', teacher: 'teacher', admin: 'admin' };
const R0 = { student: ['/', '/chat', '/threads', '/practice', '/practice/history', '/library', '/calendar', '/me', '/join', '/exams', '/assignments'],
  ta: ['/', '/threads', '/calendar', '/inbox', '/students', '/attendance', '/class/members', '/gradebook', '/gradebook/scheme', '/grading', '/questions', '/documents', '/insights', '/analytics', '/exams'],
  teacher: ['/', '/threads', '/calendar', '/inbox', '/students', '/attendance', '/class/members', '/gradebook', '/gradebook/scheme', '/grading', '/questions', '/documents', '/insights', '/analytics', '/exams', '/settings/llm', '/observability'],
  admin: ['/', '/observability', '/settings/llm', '/settings/integrations', '/admin/courses', '/admin/users', '/admin/audit'] };
const VP = [{ w: 1440, h: 900, tag: '1440' }, { w: 390, h: 844, tag: '390' }, { w: 375, h: 812, tag: '375', only: (role, r) => role === 'student' || r === '/attendance' || r === '/inbox' }];
const FORBIDDEN = /\b(RAG|PII|fallback|trace|provider|confidence|redaction|embedding|prompt|LLM|placeholder)\b|độ tin cậy|cần chú ý|rủi ro|\[\[SV_/i;
(async () => { const br = await chromium.launch({ headless: true }); const rows = []; const roles = (process.env.ROLES || 'student,ta,teacher,admin').split(',');
  for (const role of roles) {
    const ctx = await br.newContext({ ignoreHTTPSErrors: true, viewport: { width: 1440, height: 900 }, locale: 'vi-VN' }); const p = await ctx.newPage();
    await p.goto(B + '/login'); await p.getByLabel(/email/i).fill(ACC[role] + '@edupilot.local'); await p.getByLabel(/mật khẩu/i).first().fill(PW); await p.getByRole('button', { name: /đăng nhập/i }).click(); await p.waitForURL((u) => !u.pathname.startsWith('/login'), { timeout: 20000 }); await p.waitForTimeout(1500); await L.dismiss(p);
    for (const vp of VP) { await p.setViewportSize({ width: vp.w, height: vp.h });
      for (const r of R0[role]) { if (vp.only && !vp.only(role, r)) continue;
        try { await p.goto(B + r, { waitUntil: 'domcontentloaded' }); await p.waitForTimeout(2200); await L.dismiss(p);
          const a = await p.evaluate(`(${AUDIT_SRC.trim()})`); const opts = await p.evaluate(() => [...document.querySelectorAll('option')].map((o) => o.textContent.trim())); a.cut = a.cut.filter((c) => !opts.some((o) => o.slice(0, 30) === c)); // <option> không có hộp vẽ ra (rect 0): bỏ (kịch bản cũ chỉ so khi chữ ≤ 30 ký tự)
          const touch = vp.w <= 390 && (role === 'student' || r === '/attendance' || r === '/inbox') ? await p.evaluate(TOUCH_SRC) : [];
          const txt = await p.evaluate('document.body.innerText'); const notFound = /Không tìm thấy trang/.test(txt); const forb = role === 'student' ? (txt.match(new RegExp(FORBIDDEN.source, 'gi')) || []).length : 0;
          rows.push({ role, route: r, w: vp.tag, pass: a.ox <= 0 && !a.cut.length && !a.ell.length && forb === 0 && !notFound && !touch.length, ox: a.ox, cut: a.cut.slice(0, 3), ell: a.ell.slice(0, 3), touch: touch.slice(0, 3), forb, notFound });
        } catch (e) { rows.push({ role, route: r, w: vp.tag, pass: false, err: String(e).slice(0, 80) }); }
      } }
    await ctx.close(); }
  const fail = rows.filter((x) => !x.pass); fs.writeFileSync(__dirname + '/../logs-r3/audit-' + (process.env.ROLES || 'all').replace(/,/g, '_') + '.json', JSON.stringify(rows, null, 1));
  console.log('AUDIT rows', rows.length, 'FAIL', fail.length); for (const f of fail) console.log(JSON.stringify(f).slice(0, 300));
  await br.close(); })();
