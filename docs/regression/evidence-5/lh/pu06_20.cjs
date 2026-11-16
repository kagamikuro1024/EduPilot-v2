// TC-PU06-20 QC2: thăm dò 6 route × 375/1440 bằng phiên đăng nhập THẬT trên ep-r (https://localhost:1143); LCP của '/' với GV / TA / Admin.
process.env.NODE_TLS_REJECT_UNAUTHORIZED = '0';
const L = require('../ui/ui_lib.cjs'); const { execSync } = require('child_process');
const OUT = __dirname + '/pu06-20'; L.fs.mkdirSync(OUT, { recursive: true });
const clear = () => { try { execSync("docker exec ep-r-redis-1 sh -c \"redis-cli --scan --pattern 'ep:rl:auth*' | xargs -r redis-cli del\"", { stdio: 'ignore', env: { ...process.env, DOCKER_HOST: 'unix://' + process.env.HOME + '/.colima/default/docker.sock' } }); } catch (e) {} };
const PLAN = [['r5s01', ['/', '/chat', '/threads']], ['r5t', ['/', '/inbox', '/gradebook']], ['admin', ['/', '/settings/llm']], ['r5ta', ['/']]];
(async () => {
  const b = await L.chromium.launch({ args: ['--no-sandbox'] }); const res = [];
  for (const [user, routes] of PLAN) for (const w of [1440, 375]) {
    clear(); const { ctx, p, errs } = await L.login(b, user, { width: w, height: w === 1440 ? 900 : 812 });
    for (const r of routes) {
      const cdp = await ctx.newCDPSession(p); await cdp.send('Emulation.setCPUThrottlingRate', { rate: 4 });
      await p.addInitScript(() => { window.__lcp = []; new PerformanceObserver((l) => l.getEntries().forEach((e) => window.__lcp.push([Math.round(e.startTime), (e.element ? e.element.tagName + ':' + (e.element.textContent || '').trim().slice(0, 28) : '')]))).observe({ type: 'largest-contentful-paint', buffered: true }); });
      await p.goto(L.B + r, { waitUntil: 'load' }); await p.waitForTimeout(4000);
      const m = await L.metrics(p); const lcp = await p.evaluate(() => window.__lcp);
      const f = `${user}-${r === '/' ? 'home' : r.slice(1).replace(/\//g, '_')}-${w}.png`; await p.screenshot({ path: OUT + '/' + f });
      await cdp.send('Emulation.setCPUThrottlingRate', { rate: 1 });
      const row = { user, route: r, w, ox: m.ox, h1: m.h1, small: m.small, primary: m.primary.length, lcpLast: lcp[lcp.length - 1], errs: errs.slice(0, 2) }; res.push(row); console.log(JSON.stringify(row));
    }
    await ctx.close();
  }
  await b.close(); L.fs.writeFileSync(OUT + '/results.json', JSON.stringify(res, null, 1));
})();
