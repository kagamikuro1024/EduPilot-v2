// Soát lại lô 3 (M1/M2/M3/M6/M9/M11/M12/M20): BASE=<url> node recheck3.mjs <out>
import { chromium } from 'playwright'; import fs from 'fs';
const BASE=process.env.BASE, PW=fs.readFileSync('/srv/work/techlead-ui/.pw','utf8').trim(), O=process.argv[2];
const C87='01a212f1-9182-7a79-a1aa-0abf98264f5e', C88='01a212f1-9196-7145-a5a1-6fbc7e10e3d0';
fs.mkdirSync(O,{recursive:true}); const log=(...a)=>fs.appendFileSync(O+'/log.txt',a.join(' ')+'\n');
const b=await chromium.launch();
const ctxOf=(w,h,mob)=>b.newContext({viewport:{width:w,height:h},deviceScaleFactor:2,isMobile:mob,hasTouch:mob});
async function login(p,email){await p.goto(BASE+'/login'); await p.getByLabel('Email').fill(email); await p.getByLabel('Mật khẩu',{exact:true}).fill(PW); await p.keyboard.press('Enter'); await p.waitForURL(u=>!u.pathname.startsWith('/login'),{timeout:20000}).catch(()=>log('login fail',email));}
const go=async(p,r)=>{await p.goto(BASE+r); await p.waitForLoadState('networkidle',{timeout:15000}).catch(()=>{}); await p.waitForTimeout(1000); await p.getByRole('button',{name:'Đã hiểu'}).click({timeout:800}).catch(()=>{});};
const shot=async(p,n)=>{await p.screenshot({path:`${O}/${n}.png`}); log(n,p.url());};
const metrics=p=>p.evaluate(()=>{const s=[...document.querySelectorAll('button,a[href],input,select,[role=radio]')].filter(e=>{const r=e.getBoundingClientRect();return r.width>0&&r.height>0&&r.left>=0&&(r.height<44||r.width<44)}).map(e=>(e.innerText||e.getAttribute('aria-label')||'').trim().slice(0,16)+':'+Math.round(e.getBoundingClientRect().height)); const nav=[...document.querySelectorAll('nav')].map(n=>{const r=n.getBoundingClientRect();return Math.round(r.left)+','+Math.round(r.top)+' '+Math.round(r.width)+'x'+Math.round(r.height)}); return {small:s.slice(0,12),nav,header:document.querySelector('header')?.innerText.replace(/\n/g,' | ').slice(0,80)};});
// khách: M3, M6
for (const [w,h,mob] of [[844,390,true],[375,812,true]]) { const c=await ctxOf(w,h,mob); const p=await c.newPage(); for (const r of ['/login','/register']) { await go(p,r); await shot(p,`M3M6${r.replace('/','-')}-${w}x${h}`);} await c.close(); }
// SV
for (const [w,h,mob] of [[375,812,true],[844,390,true]]) { const c=await ctxOf(w,h,mob); const p=await c.newPage(); await login(p,'sv.gioi@edupilot.local');
  await go(p,'/?course='+C87); await shot(p,`M1M11-today-sv-${w}x${h}`); log('m',w,JSON.stringify(await metrics(p)));
  await go(p,'/checkin'); await shot(p,`M6-checkin-${w}x${h}`);
  await go(p,'/chat'); await shot(p,`M9-chat-${w}x${h}`);
  await go(p,'/calendar'); await shot(p,`M2-calendar-sv-${w}x${h}`); log('m',w,JSON.stringify(await metrics(p)));
  if (w===375) { await p.route('**/api/v1/**', async r=>{await new Promise(s=>setTimeout(s,6000)); r.continue().catch(()=>{});}); await p.goto(BASE+'/practice'); await p.waitForTimeout(1500); await shot(p,'M12-loading-practice-375'); await p.goto(BASE+'/threads'); await p.waitForTimeout(1500); await shot(p,'M12-loading-threads-375'); await p.unroute('**/api/v1/**'); }
  await c.close(); }
// GV
for (const [w,h,mob] of [[375,812,true],[844,390,true]]) { const c=await ctxOf(w,h,mob); const p=await c.newPage(); await login(p,'teacher@edupilot.local');
  await go(p,'/?course='+C88); await shot(p,`M20-today-gv-${w}x${h}`); log('m',w,JSON.stringify(await metrics(p)));
  await go(p,'/attendance'); await shot(p,`M2-attendance-gv-${w}x${h}`); log('m',w,JSON.stringify(await metrics(p)));
  await c.close(); }
await b.close(); log('DONE');
