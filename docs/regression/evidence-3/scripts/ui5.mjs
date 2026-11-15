import { chromium } from '/tmp/qc2-b-ws/frontend/node_modules/@playwright/test/index.mjs';
import AxeBuilder from '/tmp/qc2-b-ws/frontend/node_modules/@axe-core/playwright/dist/index.mjs';
import fs from 'fs';
const EXE=process.env.HOME+'/Library/Caches/ms-playwright/chromium_headless_shell-1247/chrome-headless-shell-mac-arm64/chrome-headless-shell';
const SH='/Users/kuro/Documents/TA_Agent_qc2/docs/regression/evidence-3/shots/';
const b=await chromium.launch({executablePath:EXE}); const R={};
async function sess(user,vp){ const ctx=await b.newContext({ignoreHTTPSErrors:true,viewport:vp,locale:'vi-VN'}); const {fixRefresh}=await import('/tmp/qc2-b/uih.mjs'); await fixRefresh(ctx); const p=await ctx.newPage();
  await p.goto('https://localhost:1143/login'); await p.locator('input[type=email]').fill(user+'@edupilot.local'); await p.locator('input[type=password]').fill('Edupilot#Seed-2026'); await p.keyboard.press('Enter'); await p.waitForURL(u=>!u.pathname.startsWith('/login')); return {ctx,p}; }
// admin 1440: nâng cao, 40-42, axe
{ const {ctx,p}=await sess('admin',{width:1440,height:900}); const puts=[]; p.on('request',r=>{if(r.method()!=='GET'&&r.url().includes('/admin/llm'))puts.push(r.method()+' '+r.url().split('/api/v1')[1]);});
  await p.goto('https://localhost:1143/settings/llm'); await p.waitForTimeout(2500);
  const adv=p.getByRole('button',{name:'Cài đặt nâng cao'}).nth(1); await adv.click(); await p.waitForTimeout(400);
  const temp=p.getByLabel('Nhiệt độ').first(); R.advLabels=await p.evaluate(()=>[...document.querySelectorAll('label')].map(l=>l.innerText.trim()).filter(t=>/Nhiệt độ|Số token|Thời gian chờ|Số lần thử|Giá|Giới hạn/.test(t)).slice(0,10));
  const res=[];
  for (const v of ['3','-1']) { await temp.fill(v); await temp.blur(); await p.waitForTimeout(300); res.push([v,(await p.locator('main').innerText()).match(/[^\n]*(từ 0|không hợp lệ|trong khoảng|Nhập)[^\n]*/)?.[0]||'']); }
  R.advInvalid=res; R.advPuts=puts.slice(); await p.screenshot({path:SH+'p105-advanced-1440.png'});
  const ax=await new AxeBuilder({page:p}).withTags(['wcag2a','wcag2aa','wcag21aa','wcag22aa']).analyze(); R.axeAdmin1440=ax.violations.filter(v=>['serious','critical'].includes(v.impact)).map(v=>v.id+':'+v.nodes.length);
  R.adminControls=await p.evaluate(()=>({buttons:[...document.querySelectorAll('main button')].filter(b=>b.offsetParent).map(b=>b.innerText.trim()).filter(Boolean).length}));
  await ctx.close(); }
// teacher 1440: chỉ xem
{ const {ctx,p}=await sess('teacher',{width:1440,height:900}); const reqs=[]; p.on('request',r=>{if(r.url().includes('/admin/llm'))reqs.push(r.method())});
  await p.goto('https://localhost:1143/settings/llm'); await p.waitForTimeout(2500);
  R.teacher={btn:await p.evaluate(()=>[...document.querySelectorAll('main button')].filter(b=>b.offsetParent).map(b=>b.innerText.trim()).filter(Boolean)),inputs:await p.locator('main input:visible, main select:visible').count(),reqs:[...new Set(reqs)]};
  const ax=await new AxeBuilder({page:p}).withTags(['wcag2a','wcag2aa','wcag21aa','wcag22aa']).analyze(); R.axeTeacher1440=ax.violations.filter(v=>['serious','critical'].includes(v.impact)).map(v=>v.id+':'+v.nodes.length);
  await ctx.close(); }
// admin 390
{ const {ctx,p}=await sess('admin',{width:390,height:844}); await p.goto('https://localhost:1143/settings/llm'); await p.waitForTimeout(2500);
  R.scrollX390=await p.evaluate(()=>document.documentElement.scrollWidth>document.documentElement.clientWidth);
  R.small390=await p.evaluate(()=>[...document.querySelectorAll('main button, main a, main input, main select')].filter(e=>e.offsetParent).map(e=>e.getBoundingClientRect()).filter(r=>r.height<40).length);
  const ax=await new AxeBuilder({page:p}).withTags(['wcag2a','wcag2aa','wcag21aa','wcag22aa']).analyze(); R.axeAdmin390=ax.violations.filter(v=>['serious','critical'].includes(v.impact)).map(v=>v.id+':'+v.nodes.length);
  await p.screenshot({path:SH+'p105-390.png',fullPage:true}); await ctx.close(); }
// 503 khi tải + Thử lại; 401 TOKEN_EXPIRED
{ const {ctx,p}=await sess('admin',{width:1440,height:900}); let n=0; await p.route('**/api/v1/admin/llm/providers',r=>{ if(r.request().method()==='GET'&&n++<1) return r.fulfill({status:503,contentType:'application/json',body:JSON.stringify({code:'NOT_READY',message:'Hệ thống chưa sẵn sàng.',trace_id:'x'})}); return r.continue(); });
  await p.goto('https://localhost:1143/settings/llm'); await p.waitForTimeout(2500); const txt=await p.locator('main').innerText(); R.err503=txt.slice(0,200).replace(/\n/g,' | ');
  const retry=p.getByRole('button',{name:/Thử lại/}); R.retryBtn=await retry.count(); if(R.retryBtn){ await retry.first().click(); await p.waitForTimeout(2000); R.afterRetry=(await p.locator('main h2').allInnerTexts()).slice(0,2);} await ctx.close(); }
// mất mạng khi Lưu → Gửi lại cùng Idempotency-Key
{ const {ctx,p}=await sess('admin',{width:1440,height:900}); const keys=[]; p.on('request',r=>{ if(r.method()==='POST'&&r.url().endsWith('/admin/llm/providers')) keys.push(r.headers()['idempotency-key']); });
  await p.goto('https://localhost:1143/settings/llm'); await p.waitForTimeout(2000); await p.getByRole('button',{name:/Thêm nhà cung cấp/}).first().click(); const f=p.locator('form').filter({has:p.getByLabel(/Khoá API/)}).first();
  await f.getByLabel(/Loại nhà cung cấp/).selectOption('openai_compatible'); await f.getByLabel(/Tên hiển thị/).fill('SmkB-off'); await f.getByLabel(/Khoá API/).fill('good-key'); await f.getByLabel(/Địa chỉ máy chủ/).fill('http://host.docker.internal:9741/v1'); await f.getByLabel(/Tên mô hình/).first().fill('m-ok');
  await ctx.setOffline(true); await f.getByRole('button',{name:/^Lưu/}).click(); await p.waitForTimeout(1500); R.offText=(await p.locator('main').innerText()).match(/[^\n]*(Gửi lại|mạng)[^\n]*/g)?.slice(0,2); R.keepVal=await f.getByLabel(/Tên hiển thị/).inputValue();
  await ctx.setOffline(false); const rb=p.getByRole('button',{name:/Gửi lại/}); R.resendBtn=await rb.count(); if(R.resendBtn){ await rb.first().click(); await p.waitForTimeout(3000);} R.idemKeys=keys; await ctx.close(); }
fs.writeFileSync('/tmp/qc2-b/ui5.json',JSON.stringify(R,null,1)); console.log(JSON.stringify(R,null,1)); await b.close();
