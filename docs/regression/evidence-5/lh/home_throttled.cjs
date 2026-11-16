// TC-PU06-08 bổ sung: '/' với phiên thật (ep-r) dưới mạng 1,6 Mbps/150 ms + CPU 4×: thời điểm chữ h1 hiện.
process.env.NODE_TLS_REJECT_UNAUTHORIZED = '0';
const L = require('../ui/ui_lib.cjs'); const { execSync } = require('child_process');
const clear = () => { try { execSync("docker exec ep-r-redis-1 sh -c \"redis-cli --scan --pattern 'ep:rl:auth*' | xargs -r redis-cli del\"", { stdio: 'ignore', env: { ...process.env, DOCKER_HOST: 'unix://' + process.env.HOME + '/.colima/default/docker.sock' } }); } catch (e) {} };
(async () => { const b = await L.chromium.launch({ args: ['--no-sandbox'] });
  for (const [u, w] of [['r5s01', 412], ['r5t', 412]]) { clear(); const { ctx, p } = await L.login(b, u, { width: w, height: 823 }); const cdp = await ctx.newCDPSession(p);
    await p.addInitScript(() => { window.__e = []; new PerformanceObserver((l) => l.getEntries().forEach((e) => window.__e.push([e.entryType === 'paint' ? e.name : 'lcp', Math.round(e.startTime), e.element ? e.element.tagName + ':' + (e.element.textContent || '').trim().slice(0, 26) : '']))).observe({ type: 'largest-contentful-paint', buffered: true }); new PerformanceObserver((l) => l.getEntries().forEach((e) => window.__e.push([e.name, Math.round(e.startTime), '']))).observe({ type: 'paint', buffered: true }); });
    await cdp.send('Network.enable'); await cdp.send('Network.emulateNetworkConditions', { offline: false, latency: 150, downloadThroughput: 1.6 * 1024 * 1024 / 8, uploadThroughput: 750 * 1024 / 8 }); await cdp.send('Emulation.setCPUThrottlingRate', { rate: 4 });
    await p.goto(L.B + '/', { waitUntil: 'load' }); await p.waitForTimeout(5000); console.log(u, JSON.stringify(await p.evaluate(() => ({ e: window.__e, h1: [...document.querySelectorAll('h1')].map((h) => h.innerText.trim()) })))); await ctx.close(); }
  await b.close(); })();
