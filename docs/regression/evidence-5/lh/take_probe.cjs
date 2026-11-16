process.env.NODE_TLS_REJECT_UNAUTHORIZED = '0';
const L = require('../ui/ui_lib.cjs'); const E = JSON.parse(L.fs.readFileSync('/tmp/qc2-c5/lhexam.json'));
(async () => { const b = await L.chromium.launch({ args: ['--no-sandbox'] });
  const { ctx, p } = await L.login(b, 'r5s02', { width: 390, height: 844 });
  await p.goto(L.B + '/exams/' + E.eid + '/take'); await p.waitForTimeout(3500);
  console.log(await p.evaluate(() => document.body.innerText.slice(0, 300).replace(/\n/g, ' | ')));
  await p.screenshot({ path: 'take-fresh-390.png' });
  const btn = p.getByRole('button', { name: /bắt đầu|làm bài/i }); console.log('btn', await btn.count(), await btn.allInnerTexts());
  await b.close(); })();
