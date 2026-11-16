process.env.NODE_TLS_REJECT_UNAUTHORIZED = '0';
const L = require('../ui/ui_lib.cjs'); const E = JSON.parse(L.fs.readFileSync('/tmp/qc2-c5/lhexam.json'));
(async () => { const b = await L.chromium.launch({ args: ['--no-sandbox'] });
  const { ctx, p } = await L.login(b, 'r5s07', { width: 412, height: 823 });
  await p.goto(L.B + '/exams/' + E.eid + '/take'); await p.waitForTimeout(2500); await L.dismiss(p);
  await p.getByRole('button', { name: /^Bắt đầu làm bài$/ }).click(); await p.waitForTimeout(2500);
  console.log(await p.evaluate(() => [...document.querySelectorAll('button,[role=radio],[role=checkbox],label,input')].filter((e) => e.offsetParent).map((e) => e.tagName + ':' + ((e.innerText || e.getAttribute('aria-label') || e.type || '')).trim().slice(0, 40)).join(' | ')));
  await p.screenshot({ path: 'take-q1-412.png' }); await b.close(); })();
