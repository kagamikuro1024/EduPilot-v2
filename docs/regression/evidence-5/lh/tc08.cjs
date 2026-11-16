// TC-PU06-08/11/14 QC2: đo trên bản dựng cổng :3330 + gateway giả :3332 (cookie lh_role). Mạng 4× CPU + 1,6 Mbps / 150 ms (như lcp.spec.ts).
const { chromium } = require('/Users/kuro/Documents/TA_Agent_qc2/frontend/node_modules/@playwright/test');
const ROUTES = [['/', 'student'], ['/chat', 'student'], ['/threads', 'student'], ['/me', 'student'], ['/inbox', 'teacher'], ['/gradebook', 'teacher'], ['/settings/llm', 'admin'], ['/dev/ui', null]];
(async () => {
  const b = await chromium.launch({ args: ['--no-sandbox'] });
  for (const mode of ['throttled']) for (const [path, role] of ROUTES) {
    const ctx = await b.newContext({ viewport: { width: 412, height: 823 }, locale: 'vi-VN' });
    if (role) await ctx.addCookies([{ name: 'lh_role', value: role, url: 'http://localhost:3330' }]);
    const p = await ctx.newPage(); const cdp = await ctx.newCDPSession(p);
    await cdp.send('Network.enable'); await cdp.send('Network.emulateNetworkConditions', { offline: false, latency: 150, downloadThroughput: 1.6 * 1024 * 1024 / 8, uploadThroughput: 750 * 1024 / 8 }); await cdp.send('Emulation.setCPUThrottlingRate', { rate: 4 });
    const fonts = []; p.on('response', (r) => { if (/font|woff/.test(r.headers()['content-type'] || '') || /\.woff2?/.test(r.url())) fonts.push(r.url().replace('http://localhost:3330', '')); });
    await p.addInitScript(() => { const w = window; w.__e = []; w.__cls = 0; const pc = (type, f) => new PerformanceObserver((l) => l.getEntries().forEach(f)).observe({ type, buffered: true });
      pc('paint', (e) => w.__e.push(['paint:' + e.name, Math.round(e.startTime)])); pc('largest-contentful-paint', (e) => w.__e.push(['lcp', Math.round(e.startTime), e.element ? e.element.tagName + ':' + (e.element.textContent || '').trim().slice(0, 30) : '', e.element && e.element.closest('[data-part=pre-shell]') ? 'pre' : 'app']));
      pc('layout-shift', (e) => { if (!e.hadRecentInput) w.__cls += e.value; }); });
    await p.goto('http://localhost:3330' + path, { waitUntil: 'load' }); await p.waitForTimeout(4500);
    const r = await p.evaluate(() => ({ e: window.__e, cls: window.__cls, fonts: document.fonts.status, loaded: [...document.fonts].filter((f) => f.status === 'loaded').map((f) => f.family + ' ' + f.weight), h1: [...document.querySelectorAll('h1')].map((h) => h.innerText.trim()), h1pre: !!document.querySelector('[data-part=pre-shell] h1'), preloads: [...document.querySelectorAll('link[rel=preload][as=font]')].length }));
    const html = await (await ctx.request.get('http://localhost:3330' + path)).text(); const sh1 = (html.match(/<h1[^>]*>([^<]*)<\/h1>/) || [])[1];
    console.log(JSON.stringify({ path, role, serverH1: sh1, clientH1: r.h1, cls: +r.cls.toFixed(4), fonts: r.fonts, loaded: r.loaded, fontReqs: fonts.length, preloadTags: (html.match(/rel="preload"[^>]*as="font"|as="font"[^>]*rel="preload"/g) || []).length, events: r.e }));
    await ctx.close();
  }
  await b.close();
})();
