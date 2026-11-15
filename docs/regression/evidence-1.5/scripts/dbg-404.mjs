import { chromium } from '/Users/kuro/Documents/TA_Agent_qc2/frontend/node_modules/@playwright/test/index.mjs';
const b=await chromium.launch({executablePath:process.env.E}); const c=await b.newContext({ignoreHTTPSErrors:true}); const p=await c.newPage(); const F='https://localhost:1143';
p.on('response',r=>{ if(r.url().includes('/auth/')) console.log('  RESP',r.request().method(),r.status(),r.url().replace(F,''),'origin-hdr=',r.request().headers()['origin']) });
await p.goto(F+'/login'); await p.getByLabel(/Email/i).first().fill('sv.kha@edupilot.local'); await p.getByLabel(/Mật khẩu/i).first().fill('Edupilot#Seed-2026'); await p.getByRole('button',{name:/Đăng nhập/i}).first().click(); await p.waitForURL(u=>!u.pathname.startsWith('/login'),{timeout:25000});
console.log('cookies',(await c.cookies()).map(x=>x.name+'@'+x.path+(x.httpOnly?'H':'')+(x.secure?'S':'')+x.sameSite).join(' '));
for (const r of ['/me','/khong-co-trang','/me']) { console.log('goto',r); await p.goto(F+r,{waitUntil:'domcontentloaded'}); await p.waitForTimeout(3500); console.log(' ->',p.url().replace(F,'')); }
await b.close();
