// qc3 s5: PU06-11 (CLS ≤ 0,1, không «Không có quyền»), PU06-16 (tay: F5, xoá ep_rt, đăng xuất giả) trên ep-r.
const L = require('./ui_lib.cjs'); const { chromium, B, rec } = L; const fs = require('fs');
const ROUTES = { 'sv.gioi': ['/', '/chat', '/threads', '/me', '/exams', '/library'], teacher: ['/', '/inbox', '/gradebook', '/threads', '/exams', '/settings/llm'], admin: ['/', '/settings/llm', '/admin/users'] };
(async () => { const br = await chromium.launch(); const rows = [];
  for (const [u, rs] of Object.entries(ROUTES)) { const { ctx, p } = await L.login(br, u, { width: 1440, height: 900 });
    await p.addInitScript(() => { window.__cls = 0; window.__flash = false; new PerformanceObserver((l) => { for (const e of l.getEntries()) if (!e.hadRecentInput) window.__cls += e.value; }).observe({ type: 'layout-shift', buffered: true }); new MutationObserver(() => { if (/Không có quyền/.test(document.body.innerText)) window.__flash = true; }).observe(document, { childList: true, subtree: true, characterData: true }); });
    for (const r of rs) { await p.goto(B + r, { waitUntil: 'load' }); await p.waitForTimeout(3500); await L.dismiss(p); const m = await p.evaluate(() => ({ cls: window.__cls, flash: window.__flash, h1: [...document.querySelectorAll('h1')].map((h) => h.innerText.trim()).slice(0, 1) })); rows.push({ u, r, ...m }); console.log(u, r, JSON.stringify(m)); }
    await ctx.close(); }
  const worst = rows.reduce((a, b) => (b.cls > a.cls ? b : a)); const flashes = rows.filter((x) => x.flash);
  rec('s5-PU06-11-tay', worst.cls <= 0.1 && !flashes.length, `CLS lớn nhất ${worst.cls.toFixed(4)} (${worst.u} ${worst.r}); ${rows.length} (vai, route) 1440; «Không có quyền» thoáng hiện: ${flashes.length}; h1 mỗi màn có = ${rows.filter((x) => x.h1.length).length}/${rows.length}`);
  // PU06-16 tay
  const { ctx, p } = await L.login(br, 'sv.gioi', { width: 1440, height: 900 }); await p.goto(B + '/me', { waitUntil: 'load' }); await p.waitForTimeout(2500);
  const reqs = []; p.on('request', (q) => { if (q.url().includes('/auth/refresh')) reqs.push(q.url()); });
  await p.reload({ waitUntil: 'load' }); await p.waitForTimeout(3000); const f5ok = !p.url().includes('/login') && /Tài khoản/.test(await p.locator('body').innerText()); const nref = reqs.length;
  const ck = await ctx.cookies(); const rt = ck.filter((c) => /ep_rt|refresh/i.test(c.name)); await ctx.clearCookies(); await p.goto(B + '/me', { waitUntil: 'load' }); await p.waitForTimeout(3500); const afterClear = p.url();
  const { ctx: c2, p: p2 } = await L.login(br, 'sv.gioi', { width: 1440, height: 900 }); await p2.route('**/auth/refresh', (r) => r.fulfill({ status: 401, contentType: 'application/json', body: JSON.stringify({ code: 'UNAUTHENTICATED', message: 'x', trace_id: 'a'.repeat(32) }) })); await p2.reload({ waitUntil: 'load' }); await p2.waitForTimeout(3500); const after401 = p2.url();
  rec('s5-PU06-16-tay', f5ok && /\/login/.test(afterClear) && /\/login/.test(after401), `F5 khi đăng nhập: vẫn ở /me=${f5ok} (${nref} lời gọi refresh); xoá cookie phiên (${rt.map((c) => c.name).join(',') || 'cookie'}) rồi mở /me → ${afterClear.replace(B, '')}; refresh giả 401 + tải lại → ${after401.replace(B, '')}`);
  await br.close(); })();
