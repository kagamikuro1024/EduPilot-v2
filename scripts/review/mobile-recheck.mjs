// Soát lại lô sửa: BASE=<url> node recheck.mjs <out> — 375 + 1440, lớp 761987 (và thread 761988 có nội dung docling).
import { chromium } from 'playwright'; import fs from 'fs';
const BASE=process.env.BASE, PW=fs.readFileSync('/srv/work/techlead-ui/.pw','utf8').trim(), O=process.argv[2];
fs.mkdirSync(O,{recursive:true}); const log=(...a)=>fs.appendFileSync(O+'/log.txt',a.join(' ')+'\n');
const b=await chromium.launch();
for (const [w,h,mob] of [[375,812,true],[1440,900,false]]) {
  const ctx=await b.newContext({viewport:{width:w,height:h},deviceScaleFactor:2,isMobile:mob,hasTouch:mob}); const p=await ctx.newPage();
  await p.goto(BASE+'/login'); await p.getByLabel('Email').fill('sv.gioi@edupilot.local'); await p.getByLabel('Mật khẩu',{exact:true}).fill(PW); await p.keyboard.press('Enter');
  await p.waitForURL(u=>!u.pathname.startsWith('/login'),{timeout:20000}).catch(()=>log('login fail',w));
  const go=async r=>{await p.goto(BASE+r); await p.waitForLoadState('networkidle',{timeout:15000}).catch(()=>{}); await p.waitForTimeout(1000); await p.getByRole('button',{name:'Đã hiểu'}).click({timeout:800}).catch(()=>{});};
  const shot=async n=>{await p.screenshot({path:`${O}/${n}-${w}.png`}); const H=await p.evaluate(()=>document.documentElement.scrollHeight); if(H>h+4) await p.screenshot({path:`${O}/${n}-${w}-full.png`,fullPage:true}); log(n,w,p.url(),H);};
  await go('/?course=01a212f1-9182-7a79-a1aa-0abf98264f5e');
  await go('/library'); log('library focus',w,await p.evaluate(()=>document.activeElement?.tagName)); await shot('M4-library');
  await p.getByRole('button',{name:/Bộ lọc/}).first().click({timeout:2000}).then(()=>p.waitForTimeout(700)).then(()=>shot('M4-library-boloc')).catch(()=>log('no Bộ lọc',w));
  await go('/threads'); await shot('M16-threads');
  await go('/me'); log('me can cu',w,JSON.stringify(await p.evaluate(()=>[...document.querySelectorAll('button,a')].filter(e=>/Căn cứ|Chi tiết/.test(e.innerText)).map(e=>{const r=e.getBoundingClientRect();return Math.round(r.width)+'x'+Math.round(r.height)})))); await shot('M14-me');
  await go('/threads/01a212f5-edf1-7b1a-9a43-b0b0b38ddd39?course=01a212f1-9196-7145-a5a1-6fbc7e10e3d0'); await shot('M5-thread');
  await ctx.close();
}
await b.close(); log('DONE');
