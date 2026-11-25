// Soát M9 /chat: 375 dọc + 844×390 ngang; giả lập bàn phím bằng thu chiều cao khung nhìn sau khi focus ô soạn.
import { chromium } from 'playwright'; import fs from 'fs';
const BASE=process.env.BASE, PW=fs.readFileSync('/srv/work/techlead-ui/.pw','utf8').trim(), O=process.argv[2];
fs.mkdirSync(O,{recursive:true}); const log=(...a)=>fs.appendFileSync(O+'/log.txt',a.join(' ')+'\n');
const b=await chromium.launch();
const geo=p=>p.evaluate(()=>{const g=s=>{const e=document.querySelector(s);if(!e)return null;const r=e.getBoundingClientRect();return [Math.round(r.top),Math.round(r.bottom),Math.round(r.left),Math.round(r.right)]};const ta=document.querySelector('main textarea');const nav=[...document.querySelectorAll('nav')].pop();const panel=ta?.closest('section,[class*=anel]');const cs=panel?getComputedStyle(panel):null;return {vh:innerHeight,ta:ta&&g('main textarea'),nav:nav&&[Math.round(nav.getBoundingClientRect().top),Math.round(nav.getBoundingClientRect().bottom)],panelBorder:cs&&cs.borderTopWidth+' '+cs.borderLeftWidth,scrollY:Math.round(scrollY),docH:document.documentElement.scrollHeight,sw:document.documentElement.scrollWidth};});
for (const [w,h,kh] of [[375,812,480],[844,390,200]]) {
  for (const course of ['01a212f1-9182-7a79-a1aa-0abf98264f5e']) {
    const c=await b.newContext({viewport:{width:w,height:h},deviceScaleFactor:2,isMobile:true,hasTouch:true}); const p=await c.newPage(); const tag=`${w}x${h}`;
    await p.goto(BASE+'/login'); await p.getByLabel('Email').fill('sv.gioi@edupilot.local'); await p.getByLabel('Mật khẩu',{exact:true}).fill(PW); await p.keyboard.press('Enter'); await p.waitForURL(u=>!u.pathname.startsWith('/login'),{timeout:20000}).catch(()=>log('login fail'));
    await p.goto(BASE+'/chat?course='+course); await p.waitForLoadState('networkidle').catch(()=>{}); await p.waitForTimeout(1500); await p.getByRole('button',{name:'Đã hiểu'}).click({timeout:800}).catch(()=>{});
    // phiên trống
    await p.getByRole('button',{name:/Phiên mới/}).click({timeout:2000}).catch(()=>{}); await p.waitForTimeout(800);
    log(tag,'empty',JSON.stringify(await geo(p))); await p.screenshot({path:`${O}/chat-empty-${tag}.png`});
    // phiên có hội thoại
    await p.getByRole('button',{name:/Phiên trước/}).click({timeout:2000}).catch(()=>{}); await p.waitForTimeout(800);
    const first=p.locator('[role=dialog] button, [role=dialog] a, [role=menu] [role=menuitem]').filter({hasText:/\S/}).nth(1); await first.click({timeout:2000}).catch(e=>log('no session',e.message.slice(0,80))); await p.waitForTimeout(2000);
    log(tag,'convo',JSON.stringify(await geo(p))); await p.screenshot({path:`${O}/chat-convo-${tag}.png`});
    // bàn phím: focus rồi thu chiều cao
    await p.locator('main textarea').click().catch(()=>{}); await p.keyboard.type('Thử'); const before=await geo(p);
    await p.setViewportSize({width:w,height:kh}); await p.waitForTimeout(800); const after=await geo(p);
    log(tag,'kb before',JSON.stringify(before)); log(tag,'kb after',JSON.stringify(after)); await p.screenshot({path:`${O}/chat-kb-${tag}.png`});
    await c.close();
  }
}
await b.close(); log('DONE');
