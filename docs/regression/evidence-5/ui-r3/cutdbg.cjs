const L = require('./ui_lib.cjs'); const { chromium, B, PW } = L;
const CASES = [['admin', '/settings/llm', 1440], ['admin', '/settings/integrations', 1440], ['teacher', '/students', 390], ['teacher', '/gradebook', 390]];
(async () => { const br = await chromium.launch(); const out = [];
  for (const [role, route, w] of CASES) {
    const ctx = await br.newContext({ ignoreHTTPSErrors: true, viewport: { width: w, height: 900 }, locale: 'vi-VN' }); const p = await ctx.newPage();
    await p.goto(B + '/login'); await p.getByLabel(/email/i).fill(role + '@edupilot.local'); await p.getByLabel(/mật khẩu/i).first().fill(PW); await p.getByRole('button', { name: /đăng nhập/i }).click(); await p.waitForURL((u) => !u.pathname.startsWith('/login'), { timeout: 20000 }); await p.waitForTimeout(1500); await L.dismiss(p);
    await p.goto(B + route, { waitUntil: 'domcontentloaded' }); await p.waitForTimeout(2500); await L.dismiss(p);
    const r = await p.evaluate(() => { const res = [];
      for (const e of document.querySelectorAll('main *, header *')) {
        if (e.children.length || !e.textContent.trim() || e.closest('[data-scroll-x]') || e.closest('.ep-sr-only')) continue;
        if (getComputedStyle(e).textOverflow === 'ellipsis' && e.scrollWidth > e.clientWidth && !e.title) continue;
        let p = e.parentElement; while (p && getComputedStyle(p).overflowX === 'visible') p = p.parentElement; if (!p) continue;
        const a = e.getBoundingClientRect(), b = p.getBoundingClientRect();
        if (a.right > b.right + 1 || a.left < b.left - 1) res.push({ tag: e.tagName, text: e.textContent.trim().slice(0, 40), cls: (e.className || '').toString().slice(0, 60), a: [Math.round(a.left), Math.round(a.right)], parent: p.tagName + '.' + (p.className || '').toString().slice(0, 50), pOver: getComputedStyle(p).overflowX, b: [Math.round(b.left), Math.round(b.right)], tableP: !!e.closest('table'), part: e.closest('[data-part]')?.getAttribute('data-part') });
      } return res; });
    out.push({ role, route, w, n: r.length, sample: r.slice(0, 6) }); await ctx.close(); }
  console.log(JSON.stringify(out, null, 1)); await br.close(); })();
