const L = require('./ui_lib.cjs'); const { chromium, login, B, pickClass } = L;
(async()=>{ const b=await chromium.launch({headless:true}); const {ctx,p}=await login(b,'r5t',{width:1440,height:900});
await p.goto(B+'/questions'); await p.waitForTimeout(2500); await pickClass(p,'R5C1'); await p.waitForTimeout(1500);
console.log((await p.locator('main').innerText()).slice(0,1200));
console.log(await p.locator('input').evaluateAll(es=>es.map(e=>e.placeholder+'|'+e.getAttribute('aria-label'))));
console.log('rows',await p.locator('tbody tr').count());
await b.close(); })();
