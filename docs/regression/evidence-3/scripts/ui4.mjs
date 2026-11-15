import { chromium } from '/tmp/qc2-b-ws/frontend/node_modules/@playwright/test/index.mjs';
import fs from 'fs';
const EXE=process.env.HOME+'/Library/Caches/ms-playwright/chromium_headless_shell-1247/chrome-headless-shell-mac-arm64/chrome-headless-shell';
const SH='/Users/kuro/Documents/TA_Agent_qc2/docs/regression/evidence-3/shots/';
const b=await chromium.launch({executablePath:EXE}); const ctx=await b.newContext({ignoreHTTPSErrors:true,viewport:{width:1440,height:900},locale:'vi-VN'}); const {fixRefresh}=await import('/tmp/qc2-b/uih.mjs'); await fixRefresh(ctx); const p=await ctx.newPage();
const R={}; const reqs=[]; p.on('request',r=>{ if(r.url().includes('/admin/llm')&&r.method()!=='GET') reqs.push(r.method()+' '+r.url().split('/api/v1')[1]+' key='+(r.headers()['idempotency-key']||'-').slice(0,8)); });
const store=async()=>JSON.stringify(await p.evaluate(()=>({l:{...localStorage},s:{...sessionStorage},c:document.cookie})));
await p.goto('https://localhost:1143/login'); await p.locator('input[type=email]').fill('admin@edupilot.local'); await p.locator('input[type=password]').fill('Edupilot#Seed-2026'); await p.keyboard.press('Enter'); await p.waitForURL(u=>!u.pathname.startsWith('/login'));
await p.goto('https://localhost:1143/settings/llm'); await p.waitForTimeout(2500);
const rows=async()=>await p.locator('[data-part=provider-row],[data-part=provider-status]').count();
const n0=await p.getByRole('button',{name:'Test kết nối'}).count();
R.testBtns=n0;
// P105-13: khoá sai khi thêm
await p.getByRole('button',{name:/Thêm nhà cung cấp/}).first().click(); await p.waitForTimeout(400);
const f=p.locator('form').filter({has:p.getByLabel(/Khoá API/)}).first();
await f.getByLabel(/Loại nhà cung cấp/).selectOption('openai_compatible');
await f.getByLabel(/Tên hiển thị/).fill('SmkB-ui'); const keyIn=f.getByLabel(/Khoá API/);
R.keyAttrs=await keyIn.evaluate(e=>({type:e.type,ac:e.autocomplete,spell:e.spellcheck,val:e.value}));
await keyIn.fill('bad-key'); await f.getByLabel(/Địa chỉ máy chủ/).fill('http://host.docker.internal:9741/v1');
R.formBtns=await f.getByRole('button').allInnerTexts(); const mnv=f.getByLabel(/Tên mô hình/).first(); if(!(await mnv.isVisible())){ const ex=f.getByRole('button',{name:/Mô hình|nâng cao|Thêm mô hình/}).first(); if(await ex.count()) await ex.click(); await p.waitForTimeout(300);} const vis=f.locator('input:visible'); R.visibleInputs=await vis.evaluateAll(es=>es.map(e=>e.labels?.[0]?.innerText||e.type)); await f.getByLabel(/Tên mô hình/).first().fill('m-ok').catch(e=>{R.modelFillErr=String(e).slice(0,100)});
await p.screenshot({path:SH+'p105-add-form-1440.png'});
await f.getByRole('button',{name:/^Lưu/}).click(); await p.waitForTimeout(2500);
R.afterBad=(await p.locator('main').innerText()).match(/Khoá API không được[^\n]*/)?.[0]||'(không thấy thông báo)';
R.badRowAdded=(await p.getByText('SmkB-ui').count());
await p.screenshot({path:SH+'p105-wrong-key-1440.png'});
R.storeAfterBad=(await store()).includes('bad-key');
// khoá đúng
await keyIn.fill('good-key'); 
await f.getByRole('button',{name:/^Lưu/}).click(); await p.waitForTimeout(2500);
R.goodSaved=await p.getByText('SmkB-ui').count(); R.afterGood=(await p.locator('main').innerText()).slice(0,500); R.formOpen=await f.isVisible();
R.leak=(await p.content()).includes('good-key')||(await store()).includes('good-key');
// Đổi khoá dialog/ô: nhấn "Đổi khoá" nếu có
const ck=p.getByRole('button',{name:/Đổi khoá/}).first(); 
if(await ck.count()){ await ck.click(); await p.waitForTimeout(300); const ki=p.locator('input[type=password]').first(); R.changeKey=await ki.evaluate(e=>({type:e.type,val:e.value,ac:e.autocomplete})); }
// Test kết nối ở hàng mới
R.reqs=reqs.slice();
fs.writeFileSync('/tmp/qc2-b/ui4.json',JSON.stringify(R,null,1)); console.log(JSON.stringify(R,null,1));
await b.close();
