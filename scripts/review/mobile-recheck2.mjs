// Soát lại lô 2: BASE=<url> node recheck2.mjs <out> — M7/M8/M13, Phát biểu + ô số, chỉnh nhỏ M4/M5/M16; 375 + 1440.
import { chromium } from 'playwright'; import fs from 'fs';
const BASE=process.env.BASE, PW=fs.readFileSync('/srv/work/techlead-ui/.pw','utf8').trim(), O=process.argv[2];
const C87='01a212f1-9182-7a79-a1aa-0abf98264f5e', C88='01a212f1-9196-7145-a5a1-6fbc7e10e3d0';
fs.mkdirSync(O,{recursive:true}); const log=(...a)=>fs.appendFileSync(O+'/log.txt',a.join(' ')+'\n');
const b=await chromium.launch();
async function as(email, w, h, mob, steps) {
  const ctx=await b.newContext({viewport:{width:w,height:h},deviceScaleFactor:2,isMobile:mob,hasTouch:mob}); const p=await ctx.newPage();
  await p.goto(BASE+'/login'); await p.getByLabel('Email').fill(email); await p.getByLabel('Mật khẩu',{exact:true}).fill(PW); await p.keyboard.press('Enter');
  await p.waitForURL(u=>!u.pathname.startsWith('/login'),{timeout:20000}).catch(()=>log('login fail',email,w));
  const go=async r=>{await p.goto(BASE+r); await p.waitForLoadState('networkidle',{timeout:15000}).catch(()=>{}); await p.waitForTimeout(1000); await p.getByRole('button',{name:'Đã hiểu'}).click({timeout:800}).catch(()=>{});};
  const shot=async n=>{await p.screenshot({path:`${O}/${n}-${w}.png`}); const H=await p.evaluate(()=>document.documentElement.scrollHeight); if(H>h+4) await p.screenshot({path:`${O}/${n}-${w}-full.png`,fullPage:true}); log(n,w,p.url(),H);};
  await steps(p, go, shot); await ctx.close();
}
for (const [w,h,mob] of [[375,812,true],[1440,900,false]]) {
  await as('sv.gioi@edupilot.local', w, h, mob, async (p, go, shot) => {
    await go('/?course='+C88); await go('/practice/history'); await shot('M7-practice-history');
    await go('/?course='+C87); await go('/practice'); await shot('M13-practice');
    await go('/calendar'); await p.getByRole('radio',{name:'Danh sách'}).click({timeout:2000}).catch(()=>{}); await p.waitForTimeout(800); await shot('M8-calendar');
    await go('/library'); await shot('M4-library');
    await go('/threads'); await shot('M16-threads');
    await go('/threads/01a212f5-edf1-7b1a-9a43-b0b0b38ddd39?course='+C88); await shot('M5-thread');
  });
  await as('teacher@edupilot.local', w, h, mob, async (p, go, shot) => {
    await go('/attendance'); log('phat bieu',w,JSON.stringify(await p.evaluate(()=>[...document.querySelectorAll('button')].filter(e=>/phát biểu/i.test(e.getAttribute('aria-label')||e.innerText)).slice(0,3).map(e=>{const r=e.getBoundingClientRect();return [Math.round(r.width),Math.round(r.height),getComputedStyle(e).opacity,getComputedStyle(e).visibility].join('/')})))); await shot('PB-attendance');
    await go('/observability'); await shot('NW-observability');
  });
}
await b.close(); log('DONE');
