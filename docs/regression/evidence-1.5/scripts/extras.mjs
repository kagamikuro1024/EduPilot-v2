import { chromium } from '/Users/kuro/Documents/TA_Agent_qc2/frontend/node_modules/@playwright/test/index.mjs';
const F='https://localhost:1143', PW='Edupilot#Seed-2026';
const b=await chromium.launch({executablePath:process.env.E}); const out=[];
async function as(who, fn){ const c=await b.newContext({ignoreHTTPSErrors:true,viewport:{width:1440,height:900}}); const p=await c.newPage();
  await p.goto(F+'/login'); await p.getByLabel(/Email/i).first().fill(who+'@edupilot.local'); await p.getByLabel(/Mật khẩu/i).first().fill(PW); await p.getByRole('button',{name:/Đăng nhập/i}).first().click(); await p.waitForURL(u=>!u.pathname.startsWith('/login'),{timeout:25000});
  const txt=async r=>{ await p.goto(F+r,{waitUntil:'domcontentloaded'}).catch(()=>{}); await p.waitForTimeout(3500); if(p.url().includes('/login')){ await p.getByLabel(/Email/i).first().fill(who+'@edupilot.local'); await p.getByLabel(/Mật khẩu/i).first().fill(PW); await p.getByRole('button',{name:/Đăng nhập/i}).first().click(); await p.waitForURL(u=>!u.pathname.startsWith('/login'),{timeout:25000}); await p.goto(F+r,{waitUntil:'domcontentloaded'}); await p.waitForTimeout(3500);} return (await p.innerText('body').catch(()=>'')).replace(/\s+/g,' '); };
  await fn(txt); await c.close(); }
await as('ta', async t=>{ const x=await t('/grading'); out.push({tc:'tc_03_06 TA /grading',ok:/Chỉ giảng viên công bố điểm/.test(x),snip:x.slice(0,160)}); });
await as('teacher', async t=>{
  let x=await t('/settings/llm'); out.push({tc:'tc_04_04 GV /settings/llm Test kết nối=0',ok:(x.match(/Test kết nối/g)||[]).length===0,n:(x.match(/Test kết nối/g)||[]).length});
  x=await t('/observability'); out.push({tc:'tc_04_03 GV /observability không [[SV_',ok:!/\[\[SV_/.test(x),snip:x.slice(0,100)});
  x=await t('/insights'); out.push({tc:'tc_04_02 GV /insights không MSSV',ok:!/\b2022\d{4}\b/.test(x)&&!/\b\d{8}\b/.test(x),snip:x.slice(0,140)});
  x=await t('/'); out.push({tc:'GV / mở (h1)',ok:x.length>200,snip:x.slice(0,100)});
});
await as('sv.kha', async t=>{ const x=await t('/me'); out.push({tc:'SV /me mở',ok:/Kết quả của tôi/.test(x)}); });
await b.close(); console.log(JSON.stringify(out,null,1));
