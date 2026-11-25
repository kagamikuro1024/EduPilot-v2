const puppeteer = require('puppeteer');
(async () => {
  const b = await puppeteer.launch({ headless: true, args: ['--no-sandbox'] });
  const p = await b.newPage(); await p.setViewport({ width: 375, height: 780 });
  const errs = [];
  p.on('console', m => { if (['error','warning'].includes(m.type())) errs.push(m.type()+': '+m.text().slice(0,240)); });
  p.on('pageerror', e => errs.push('pageerror: '+String(e).slice(0,240)));
  p.on('requestfailed', r => errs.push('reqfail: '+r.url().slice(0,140)+' '+(r.failure()&&r.failure().errorText)));
  await p.goto('https://v2202609419695521098.happysrv.de/checkin?c=01a21372-6d00-77b3-9c58-d3b3ea86311c', { waitUntil: 'domcontentloaded' });
  await new Promise(r => setTimeout(r, 12000));
  console.log('URL', p.url());
  console.log('BODY', (await p.evaluate(() => document.body.innerText)).slice(0,500).replace(/\n+/g,' | '));
  console.log('ERRS\n' + errs.slice(0,10).join('\n'));
  await p.screenshot({ path: '/tmp/checkin402.png' });
  await b.close();
})();
