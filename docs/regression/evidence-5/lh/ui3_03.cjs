// TC-UI3-03 QC2: LEFT_SRC trên 6 route + AUDIT_SRC (1440/1024/375), phiên đăng nhập THẬT trên ep-r.
process.env.NODE_TLS_REJECT_UNAUTHORIZED = '0';
const L = require('../ui/ui_lib.cjs'); const { execSync } = require('child_process');
const clear = () => { try { execSync("docker exec ep-r-redis-1 sh -c \"redis-cli --scan --pattern 'ep:rl:auth*' | xargs -r redis-cli del\"", { stdio: 'ignore', env: { ...process.env, DOCKER_HOST: 'unix://' + process.env.HOME + '/.colima/default/docker.sock' } }); } catch (e) {} };
const LEFT = `(() => { const e = document.querySelector('[data-part=page-title]'); return e ? Math.round(e.getBoundingClientRect().left) : null; })()`;
const PLAN = [['r5s01', ['/', '/chat', '/threads']], ['r5t', ['/inbox', '/gradebook']], ['admin', ['/settings/llm']]];
(async () => { const b = await L.chromium.launch({ args: ['--no-sandbox'] }); const rows = [];
  for (const [user, routes] of PLAN) for (const w of [1440, 1024, 375]) {
    clear(); const { ctx, p } = await L.login(b, user, { width: w, height: w === 375 ? 812 : 900 });
    for (const r of routes) { await p.goto(L.B + r); await p.waitForTimeout(3000); await L.dismiss(p);
      const left = await p.evaluate(LEFT); const au = await p.evaluate(L.AUDIT_SRC); const ox = await p.evaluate(() => document.documentElement.scrollWidth - innerWidth);
      const row = { user, route: r, w, left, ox, audit: au }; rows.push(row); console.log(JSON.stringify(row).slice(0, 400)); }
    await ctx.close(); }
  await b.close(); L.fs.writeFileSync(__dirname + '/ui3_03.json', JSON.stringify(rows, null, 1)); })();
