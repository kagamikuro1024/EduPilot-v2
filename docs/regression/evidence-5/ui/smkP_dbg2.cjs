const L = require('./ui_lib.cjs'); const { chromium, login, B } = L; const C1='01a1df30-523a-7d7a-a240-c120dbd57ac3';
(async()=>{ const b=await chromium.launch({headless:true}); const {ctx,p}=await login(b,'r5t',{width:390,height:900});
await p.goto(`${B}/exams/01a1e136-5a4e-7b5d-a18d-a7ab3720c726/results?course=${C1}`); await p.waitForTimeout(4000); await p.locator('tbody tr').first().click(); await p.waitForTimeout(2000);
console.log('DRAWER', (await p.locator('dialog[open]').allInnerTexts()).join(' ').replace(/\n/g,' ¦ ').slice(0,700));
await p.keyboard.press('Escape');
await p.goto(`${B}/exams/01a1e4c0-f71a-78a3-bffd-5a7172384974/results?course=${C1}`); await p.waitForTimeout(4000);
console.log('EMPTY', (await p.locator('main').innerText()).replace(/\n/g,' ¦ ').slice(0,900));
await b.close(); })();
