import { chromium } from '/Users/kuro/Documents/TA_Agent_qc2/frontend/node_modules/@playwright/test/index.mjs';
const b=await chromium.launch({executablePath:process.env.E}); const F='https://localhost:1143';
for (const [who,route] of [['ta@edupilot.local','/'],['teacher@edupilot.local','/'],['sv.kha@edupilot.local','/inbox']]) {
 const c=await b.newContext({ignoreHTTPSErrors:true,viewport:{width:375,height:812}}); const p=await c.newPage();
 await p.goto(F+'/login'); await p.getByLabel(/Email/i).first().fill(who); await p.getByLabel(/Mật khẩu/i).first().fill('Edupilot#Seed-2026'); await p.getByRole('button',{name:/Đăng nhập/i}).first().click(); await p.waitForURL(u=>!u.pathname.startsWith('/login'),{timeout:25000});
 for (const w of [375,390,414]) { await p.setViewportSize({width:w,height:812}); await p.goto(F+route,{waitUntil:'domcontentloaded'}); await p.waitForTimeout(3000);
  const r=await p.evaluate(()=>{const W=innerWidth; const bad=[...document.querySelectorAll('body *')].filter(e=>{const r=e.getBoundingClientRect(); return r.right>W+1 && r.width>0}).slice(0,3).map(e=>e.tagName+'.'+(e.className||'').toString().slice(0,50)+' right='+Math.round(e.getBoundingClientRect().right)); return {sw:document.documentElement.scrollWidth-W,bad}});
  console.log(who.split('@')[0],route,w,JSON.stringify(r)); }
 await c.close(); }
const c=await b.newContext({ignoreHTTPSErrors:true}); const p=await c.newPage(); await p.goto(F+'/login'); await p.getByLabel(/Email/i).first().fill('sv.kha@edupilot.local'); await p.getByLabel(/Mật khẩu/i).first().fill('Edupilot#Seed-2026'); await p.getByRole('button',{name:/Đăng nhập/i}).first().click(); await p.waitForURL(u=>!u.pathname.startsWith('/login'),{timeout:25000}); await p.goto(F+'/inbox'); await p.waitForTimeout(3000); console.log('BLOCK TEXT:',(await p.innerText('main').catch(()=>p.innerText('body'))).replace(/\s+/g,' ').slice(0,300));
await b.close();
