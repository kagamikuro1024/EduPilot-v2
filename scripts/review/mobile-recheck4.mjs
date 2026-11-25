// Soát P-11-137: nav SV 10 mục, «Thêm», «Tìm nhanh», hàng buổi «Điểm danh» (checkin_link giả lập nếu gateway VPS chưa có), /checkin lớp không bật.
import { chromium } from 'playwright'; import fs from 'fs';
const BASE=process.env.BASE, PW=fs.readFileSync('/srv/work/techlead-ui/.pw','utf8').trim(), O=process.argv[2];
const C87='01a212f1-9182-7a79-a1aa-0abf98264f5e', C88='01a212f1-9196-7145-a5a1-6fbc7e10e3d0';
fs.mkdirSync(O,{recursive:true}); const log=(...a)=>fs.appendFileSync(O+'/log.txt',a.join(' ')+'\n');
const b=await chromium.launch();
for (const [w,h,mob] of [[375,812,true],[844,390,true],[1440,900,false]]) {
  const c=await b.newContext({viewport:{width:w,height:h},deviceScaleFactor:2,isMobile:mob,hasTouch:mob}); const p=await c.newPage(); const tag=`${w}x${h}`;
  let real=null;
  await p.route('**/api/v1/courses/*/today', async r=>{ const res=await r.fetch(); const j=await res.json(); real=JSON.stringify(j).includes('checkin_link'); const tl=j.timeline||j.sessions||[]; if(!real && tl[0]) tl[0].checkin_link=true; await r.fulfill({response:res, json:j}); });
  await p.goto(BASE+'/login'); await p.getByLabel('Email').fill('sv.gioi@edupilot.local'); await p.getByLabel('Mật khẩu',{exact:true}).fill(PW); await p.keyboard.press('Enter'); await p.waitForURL(u=>!u.pathname.startsWith('/login'),{timeout:20000}).catch(()=>log('login fail'));
  const go=async r=>{await p.goto(BASE+r); await p.waitForLoadState('networkidle',{timeout:15000}).catch(()=>{}); await p.waitForTimeout(1000); await p.getByRole('button',{name:'Đã hiểu'}).click({timeout:800}).catch(()=>{});};
  await go('/?course='+C87); log(tag,'today checkin_link từ API thật:',real); await p.screenshot({path:`${O}/today-${tag}.png`});
  const btn=p.getByRole('link',{name:'Điểm danh'}); log(tag,'links Điểm danh', await btn.count(), JSON.stringify(await btn.evaluateAll(es=>es.map(e=>{const r=e.getBoundingClientRect();return Math.round(r.width)+'x'+Math.round(r.height)+'@'+Math.round(r.top)}))));
  log(tag,'nav', JSON.stringify(await p.evaluate(()=>[...document.querySelectorAll('nav')].map(n=>[...n.querySelectorAll('a,button')].filter(a=>a.getBoundingClientRect().width>0).map(a=>a.innerText.trim().replace(/\n/g,' ')).join('|')))));
  if (mob) { await p.getByRole('button',{name:/Thêm/}).last().click({timeout:3000}).catch(e=>log('no Thêm',e.message)); await p.waitForTimeout(800); await p.screenshot({path:`${O}/them-${tag}.png`}); await p.keyboard.press('Escape'); }
  await p.getByRole('button',{name:/Tìm nhanh/}).first().click({timeout:3000}).catch(()=>{}); await p.waitForTimeout(500); await p.keyboard.type('điểm'); await p.waitForTimeout(800); await p.screenshot({path:`${O}/timnhanh-${tag}.png`}); await p.keyboard.press('Escape');
  for (const [n,cid] of [['c87',C87],['c88',C88]]) { await go('/checkin?course='+cid); await p.screenshot({path:`${O}/checkin-${n}-${tag}.png`}); log(tag,'checkin',n,(await p.locator('main').innerText()).replace(/\n/g,' | ').slice(0,160)); }
  await c.close();
}
await b.close(); log('DONE');
