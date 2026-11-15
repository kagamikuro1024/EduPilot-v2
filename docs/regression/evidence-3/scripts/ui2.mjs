import { chromium } from '/tmp/qc2-b-ws/frontend/node_modules/@playwright/test/index.mjs';
const EXE=process.env.HOME+'/Library/Caches/ms-playwright/chromium_headless_shell-1247/chrome-headless-shell-mac-arm64/chrome-headless-shell';
const b=await chromium.launch({executablePath:EXE}); const ctx=await b.newContext({ignoreHTTPSErrors:true,viewport:{width:1440,height:900}}); const {fixRefresh}=await import('/tmp/qc2-b/uih.mjs'); await fixRefresh(ctx); const p=await ctx.newPage();
await p.goto('https://localhost:1143/login'); await p.locator('input[type=email]').fill('admin@edupilot.local'); await p.locator('input[type=password]').fill('Edupilot#Seed-2026'); await p.keyboard.press('Enter'); await p.waitForURL(u=>!u.pathname.startsWith('/login'));
await p.goto('https://localhost:1143/settings/llm'); await p.waitForTimeout(3000);
console.log(await p.locator('[data-part=usage-section]').innerText()); console.log(await p.evaluate(()=>({
 prim:[...document.querySelectorAll('button.primary,[data-variant=primary]')].map(b=>({t:b.innerText.trim(),vis:b.offsetParent!==null,dis:b.disabled})),
 emb:[...document.querySelectorAll('body *')].filter(e=>e.children.length===0&&/\b(RAG|PII|trace|fallback|embedding|prompt)\b/i.test(e.textContent||'')).map(e=>e.tagName+':'+e.textContent.trim().slice(0,60)+' @'+(e.closest('[data-part]')?.getAttribute('data-part')||'')),
 sections:[...document.querySelectorAll('[data-part=settings-section]')].map(s=>s.querySelector('h2')?.innerText),
 nestedSections:document.querySelectorAll('[data-part=settings-section] [data-part=settings-section]').length,
 left:document.querySelector('[data-part=page-title]')?.getBoundingClientRect().left})));
await b.close();
