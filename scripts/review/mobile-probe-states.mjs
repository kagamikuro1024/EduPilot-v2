import { chromium } from 'playwright'; import fs from 'fs';
const BASE='https://v2202609419695521098.happysrv.de', PW=fs.readFileSync('/srv/work/techlead-ui/.pw','utf8').trim(), O='/srv/work/techlead-ui/out/p2';
fs.mkdirSync(O,{recursive:true});
const b=await chromium.launch(); const p=await b.newPage({viewport:{width:375,height:812},deviceScaleFactor:2,isMobile:true,hasTouch:true});
const log=(...a)=>fs.appendFileSync(O+'/log.txt',a.join(' ')+'\n');
await p.goto(BASE+'/login'); await p.getByLabel('Email').fill('sv.gioi@edupilot.local'); await p.getByLabel('Mật khẩu',{exact:true}).fill(PW); await p.keyboard.press('Enter'); await p.waitForURL(u=>!u.pathname.startsWith('/login'));
const go=async r=>{await p.goto(BASE+r); await p.waitForLoadState('networkidle',{timeout:15000}).catch(()=>{}); await p.waitForTimeout(900); await p.getByRole('button',{name:'Đã hiểu'}).click({timeout:800}).catch(()=>{});};
const shot=async n=>{await p.screenshot({path:`${O}/${n}.png`}); const h=await p.evaluate(()=>document.documentElement.scrollHeight); if(h>820) await p.screenshot({path:`${O}/${n}-full.png`,fullPage:true}); log(n,p.url(),h);};
await go('/?course=01a212f1-9182-7a79-a1aa-0abf98264f5e');
await go('/assignments'); log('hrefs', await p.evaluate(()=>[...document.querySelectorAll('main a')].map(a=>a.getAttribute('href')).join(' ')));
await p.getByRole('link',{name:/Nộp bài|Xem/}).first().click().catch(e=>log('noclick',e.message)); await p.waitForLoadState('networkidle').catch(()=>{}); await p.waitForTimeout(1200); await shot('assignment-detail');
await go('/assignments'); await p.getByRole('link',{name:'Xem'}).first().click().catch(()=>{}); await p.waitForLoadState('networkidle').catch(()=>{}); await p.waitForTimeout(1200); await shot('assignment-detail-2');
await go('/practice'); await p.getByRole('button',{name:'Luyện 10 câu'}).first().click().catch(async()=>{await p.getByRole('link',{name:'Luyện 10 câu'}).first().click().catch(e=>log('nopractice',e.message))}); await p.waitForURL(/\/practice\/[^h]/,{timeout:20000}).catch(()=>{}); await p.waitForLoadState('networkidle').catch(()=>{}); await p.waitForTimeout(1500); await shot('practice-attempt');
await go('/threads'); await p.getByRole('button',{name:'Đặt câu hỏi'}).first().click().catch(()=>p.getByRole('link',{name:'Đặt câu hỏi'}).first().click().catch(()=>{})); await p.waitForTimeout(1500); await shot('threads-compose');
await go('/me'); await p.getByText('Mở').first().click().catch(()=>{}); await p.waitForTimeout(800); await shot('me-open');
// lỗi mạng: chặn API
await p.route('**/api/v1/**', r=>r.abort()); await go('/assignments'); await shot('err-assignments'); await go('/me'); await shot('err-me'); await p.unroute('**/api/v1/**');
// tải chậm
await p.route('**/api/v1/**', async r=>{await new Promise(s=>setTimeout(s,6000)); r.continue().catch(()=>{});}); await p.goto(BASE+'/practice'); await p.waitForTimeout(1500); await shot('loading-practice'); await p.goto(BASE+'/threads'); await p.waitForTimeout(1500); await shot('loading-threads');
await b.close(); log('DONE');
