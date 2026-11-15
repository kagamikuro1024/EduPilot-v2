import { chromium } from '/tmp/qc2-b-ws/frontend/node_modules/@playwright/test/index.mjs';
const EXE=process.env.HOME+'/Library/Caches/ms-playwright/chromium_headless_shell-1247/chrome-headless-shell-mac-arm64/chrome-headless-shell';
const b=await chromium.launch({executablePath:EXE});
const out={};
for (const [name,vp] of [['1440',{width:1440,height:900}],['390',{width:390,height:844}]]) {
  const ctx=await b.newContext({viewport:vp,ignoreHTTPSErrors:true,locale:'vi-VN'}); const {fixRefresh}=await import('/tmp/qc2-b/uih.mjs'); await fixRefresh(ctx); const p=await ctx.newPage();
  await p.goto("https://localhost:1143/login"); await p.locator("input[type=email]").fill("admin@edupilot.local"); await p.locator("input[type=password]").fill("Edupilot#Seed-2026"); await p.keyboard.press("Enter"); await p.waitForURL(u=>!u.pathname.startsWith("/login"));
  await p.waitForTimeout(2500);
  const reqs=[]; p.on('request',r=>{ if(r.url().includes('/admin/llm')) reqs.push(r.method()+' '+r.url().split('/api/v1')[1]); });
  await p.goto('https://localhost:1143/settings/llm'); await p.waitForTimeout(3500);
  await p.screenshot({path:`/Users/kuro/Documents/TA_Agent_qc2/docs/regression/evidence-3/shots/settings-llm-${name}.png`,fullPage:true});
  out[name]={url:p.url(),h1:await p.locator('h1').allInnerTexts(),h2:await p.locator('h2').allInnerTexts(),sections:await p.locator('[data-part=settings-section]').count(),primary:await p.locator('button.primary,[data-variant=primary]').count(),
   testBtns:await p.getByRole('button',{name:/Test kết nối|Kiểm tra kết nối/i}).count(),reqs,text:(await p.locator('main').innerText()).slice(0,1500),
   scrollX:await p.evaluate(()=>document.documentElement.scrollWidth>document.documentElement.clientWidth),
   jargon:(await p.locator('body').innerText()).match(/\b(RAG|PII|trace|fallback|embedding|prompt)\b/ig)};
  await ctx.close();
}
console.log(JSON.stringify(out,null,1)); await b.close();
