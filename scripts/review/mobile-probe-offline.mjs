import { chromium } from 'playwright'; import fs from 'fs';
const BASE='https://v2202609419695521098.happysrv.de', PW=fs.readFileSync('/srv/work/techlead-ui/.pw','utf8').trim(), O='/srv/work/techlead-ui/out/p3';
fs.mkdirSync(O,{recursive:true}); const log=(...a)=>fs.appendFileSync(O+'/log.txt',a.join(' ')+'\n');
const b=await chromium.launch(); const ctx=await b.newContext({viewport:{width:375,height:812},deviceScaleFactor:2,isMobile:true,hasTouch:true}); const p=await ctx.newPage();
await p.goto(BASE+'/login'); await p.getByLabel('Email').fill('sv.gioi@edupilot.local'); await p.getByLabel('Mật khẩu',{exact:true}).fill(PW); await p.keyboard.press('Enter'); await p.waitForURL(u=>!u.pathname.startsWith('/login'));
await p.goto(BASE+'/?course=01a212f1-9182-7a79-a1aa-0abf98264f5e'); await p.waitForLoadState('networkidle').catch(()=>{}); await p.getByRole('button',{name:'Đã hiểu'}).click({timeout:1500}).catch(()=>{});
await ctx.setOffline(true);
for (const name of ['Luyện đề','Hỏi đáp','Chat riêng']) { await p.getByRole('link',{name}).first().click().catch(e=>log('click',name,e.message)); await p.waitForTimeout(4000); await p.screenshot({path:`${O}/offline-${name.replace(/ /g,'-')}.png`}); log(name,p.url()); }
await ctx.setOffline(false); await p.waitForTimeout(5000); await p.screenshot({path:`${O}/back-online.png`}); log('online',p.url());
await b.close(); log('DONE');
