// smkP: PU01-20/22/25/27/36 + PU05-41/GATEPU-14 (Chrome thật trên ep-r). node smkP_pu01.cjs
const L = require('../../evidence-5/ui/ui_lib.cjs'); const { chromium, login, B, PW, fs } = L; const out = {};
(async () => { const br = await chromium.launch({ headless: true });
  // --- PU01-20 + 22 + 27 trên /login (khách)
  { const ctx = await br.newContext({ ignoreHTTPSErrors: true, viewport: { width: 1440, height: 900 }, locale: 'vi-VN' }); const p = await ctx.newPage(); const reqs = []; p.on('request', (r) => reqs.push(r.url()));
    await p.goto(B + '/login', { waitUntil: 'load' }); await p.waitForTimeout(1500);
    const ext = reqs.filter((u) => /fonts\.(googleapis|gstatic)\.com/.test(u)); const hosts = [...new Set(reqs.map((u) => new URL(u).host))];
    out.pu20 = { total: reqs.length, google: ext.length, hosts };
    out.pu22 = await p.evaluate(() => getComputedStyle(document.documentElement).getPropertyValue('--ep-font').trim());
    out.pu27 = await p.evaluate(async () => { const de = document.documentElement; const cs = getComputedStyle(de); const sel = document.createElement('p'); sel.textContent = 'chon'; document.body.appendChild(sel); const selbg = getComputedStyle(sel, '::selection').backgroundColor; const t = document.createElement('i'); t.style.color = 'var(--ep-red-soft)'; document.body.appendChild(t); const soft = getComputedStyle(t).color; t.remove(); sel.remove();
      const css = []; for (const l of document.querySelectorAll('link[rel=stylesheet]')) { try { css.push(await (await fetch(l.href)).text()); } catch {} } const all = css.join('\n');
      return { colorScheme: cs.colorScheme, selectionBg: selbg, redSoft: soft, scrollbarWidth: cs.scrollbarWidth, webkitScrollbarRule: /::-webkit-scrollbar/.test(all), prefersDark: /prefers-color-scheme\s*:\s*dark/.test(all), cssBytes: all.length }; });
    await ctx.close(); }
  // --- PU01-25 cỡ chữ
  const sizes = {}; let tokenSet = null;
  for (const [u, routes] of [['sv.gioi', ['/', '/chat', '/threads']], ['teacher', ['/gradebook']]]) {
    const { ctx, p } = await login(br, u, { width: 1440, height: 900 });
    for (const r of routes) { await p.goto(B + r); await p.waitForTimeout(3500);
      const res = await p.evaluate(() => { const toks = {}; for (const ss of document.styleSheets) { let rules; try { rules = ss.cssRules; } catch { continue; } for (const ru of rules) { if (ru.style) for (const n of ru.style) if (/^--ep-text-/.test(n)) toks[n] = 1; } }
        const probe = document.createElement('span'); document.body.appendChild(probe); const set = {}; for (const n of Object.keys(toks)) { probe.style.fontSize = `var(${n})`; set[n] = getComputedStyle(probe).fontSize; } probe.remove();
        const used = {}; const w = document.createTreeWalker(document.body, NodeFilter.SHOW_TEXT); let n; while ((n = w.nextNode())) { if (!n.textContent.trim()) continue; const e = n.parentElement; if (!e || !e.offsetParent && getComputedStyle(e).position !== 'fixed') continue; const fs = getComputedStyle(e).fontSize; used[fs] = (used[fs] || 0) + 1; }
        return { set, used }; });
      tokenSet = res.set; sizes[u + r] = res.used; }
    await ctx.close(); }
  const allowed = new Set(Object.values(tokenSet)); out.pu25 = { tokens: tokenSet, perRoute: sizes, strange: Object.fromEntries(Object.entries(sizes).map(([k, v]) => [k, Object.keys(v).filter((s) => !allowed.has(s))])) };
  // --- PU01-36 storage sau thao tác
  { const { ctx, p } = await login(br, 'sv.gioi', { width: 1440, height: 900 }); for (const r of ['/', '/chat', '/me']) { await p.goto(B + r); await p.waitForTimeout(1500); }
    out.pu36 = await p.evaluate(() => ({ ls: Object.keys(localStorage), ss: Object.keys(sessionStorage), cookie: document.cookie.split(';').map((c) => c.split('=')[0].trim()).filter(Boolean) })); await ctx.close();
    const c2 = await br.newContext({ ignoreHTTPSErrors: true }); const q = await c2.newPage(); await q.goto(B + '/login'); await q.waitForTimeout(1500); out.pu36guest = await q.evaluate(() => ({ ls: Object.keys(localStorage), cookie: document.cookie })); await c2.close(); }
  // --- PU05-41 /chat 375x330
  { const ctx = await br.newContext({ ignoreHTTPSErrors: true, viewport: { width: 375, height: 330 }, isMobile: true, hasTouch: true, locale: 'vi-VN' }); const p = await ctx.newPage();
    await p.goto(B + '/login'); await p.getByLabel(/email/i).fill('sv.gioi@edupilot.local'); await p.getByLabel(/mật khẩu/i).first().fill(PW); await p.getByRole('button', { name: /đăng nhập/i }).click(); await p.waitForURL((u) => !u.pathname.startsWith('/login'), { timeout: 20000 }); await L.dismiss(p); await p.goto(B + '/chat'); await p.waitForTimeout(3000);
    const ta = p.locator('textarea').first(); await ta.focus(); await ta.fill('xin chào'); await p.waitForTimeout(500);
    out.pu41 = await p.evaluate(() => { const t = document.querySelector('textarea'); const b = t.getBoundingClientRect(); const cx = b.left + b.width / 2, cy = b.top + b.height / 2; const top = document.elementFromPoint(cx, cy); const nav = [...document.querySelectorAll('nav,[data-part=bottom-nav],footer')].filter((e) => getComputedStyle(e).position === 'fixed' || getComputedStyle(e).position === 'sticky').map((e) => { const r = e.getBoundingClientRect(); return { tag: e.tagName, top: Math.round(r.top), h: Math.round(r.height) }; });
      return { rect: { top: Math.round(b.top), bottom: Math.round(b.bottom), left: Math.round(b.left), right: Math.round(b.right) }, vh: innerHeight, inViewport: b.top >= 0 && b.bottom <= innerHeight && b.width > 0, covered: !(top === t || t.contains(top)), topEl: top && (top.tagName + '.' + (top.className || '').toString().slice(0, 30)), fixedBars: nav, scrollW: document.documentElement.scrollWidth - innerWidth }; });
    const send = p.getByRole('button', { name: /gửi/i }).first(); out.pu41.sendVisible = (await send.count()) ? await send.isVisible() : null; await p.screenshot({ path: __dirname + '/../shots/smkP-chat-375x330.png' });
    await send.click().catch(() => {}); await p.waitForTimeout(3000); out.pu41.afterSend = (await p.locator('main').innerText()).slice(0, 120).replace(/\n/g, ' | ');
    await ctx.close(); }
  console.log(JSON.stringify(out, null, 1)); fs.writeFileSync(__dirname + '/../logs/smkP-pu01.json', JSON.stringify(out, null, 1)); await br.close(); })();
