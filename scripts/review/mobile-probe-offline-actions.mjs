import { chromium } from 'playwright'; import fs from 'fs';
const BASE='https://v2202609419695521098.happysrv.de', PW=fs.readFileSync('/srv/work/techlead-ui/.pw','utf8').trim(), O='/srv/work/techlead-ui/out/p4';
fs.mkdirSync(O,{recursive:true}); const log=(...a)=>fs.appendFileSync(O+'/log.txt',a.join(' ')+'\n');
const b=await chromium.launch(); const ctx=await b.newContext({viewport:{width:375,height:812},deviceScaleFactor:2,isMobile:true,hasTouch:true}); const p=await ctx.newPage();
const shot=async n=>{await p.screenshot({path:`${O}/${n}.png`}); log(n,p.url(), JSON.stringify(await p.evaluate(()=>[...document.querySelectorAll('[role=status],[role=alert]')].map(e=>e.innerText.trim()).filter(Boolean))));};
await p.goto(BASE+'/login'); await p.getByLabel('Email').fill('sv.gioi@edupilot.local'); await p.getByLabel('Mật khẩu',{exact:true}).fill(PW); await p.keyboard.press('Enter'); await p.waitForURL(u=>!u.pathname.startsWith('/login'));
await p.goto(BASE+'/?course=01a212f1-9182-7a79-a1aa-0abf98264f5e'); await p.waitForLoadState('networkidle').catch(()=>{}); await p.getByRole('button',{name:'Đã hiểu'}).click({timeout:1500}).catch(()=>{});
// 1) đứng yên trên trang, mất mạng
await ctx.setOffline(true); await p.waitForTimeout(3000); await shot('a-today-offline');
await ctx.setOffline(false); await p.waitForTimeout(2000);
// 2) chat: gửi khi mất mạng
await p.goto(BASE+'/chat'); await p.waitForLoadState('networkidle').catch(()=>{});
await ctx.setOffline(true); await p.waitForTimeout(1500);
await p.locator('textarea').first().fill('Tuần này tôi có buổi học nào?'); await p.getByRole('button',{name:'Gửi'}).click({timeout:3000}).catch(e=>log('send',e.message)); await p.waitForTimeout(4000); await shot('b-chat-send-offline');
await ctx.setOffline(false); await p.waitForTimeout(4000); await shot('c-chat-online-again');
// 3) luyện đề: trả lời khi mất mạng
await p.goto(BASE+'/practice'); await p.waitForLoadState('networkidle').catch(()=>{});
await p.getByRole('button',{name:'Luyện 10 câu'}).first().click().catch(()=>p.getByRole('link',{name:'Luyện 10 câu'}).first().click().catch(()=>{}));
await p.waitForURL(/\/practice\/[0-9a-f]/,{timeout:20000}).catch(()=>{}); await p.waitForLoadState('networkidle').catch(()=>{}); await p.waitForTimeout(1000);
await ctx.setOffline(true); await p.waitForTimeout(1500);
const ta=p.locator('main textarea').first(); if (await ta.count()) await ta.fill('ECB mã hoá từng khối độc lập nên lộ mẫu lặp.'); else await p.locator('main input[type=radio]').first().check().catch(()=>{});
await p.getByRole('button',{name:/Trả lời|Kiểm tra|Nộp/}).first().click({timeout:3000}).catch(e=>log('answer',e.message)); await p.waitForTimeout(4000); await shot('d-practice-answer-offline');
await ctx.setOffline(false); await p.waitForTimeout(5000); await shot('e-practice-online-again');
await b.close(); log('DONE');
