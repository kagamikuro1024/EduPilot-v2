import { chromium, newCtx, BASE, text } from "./ulib.mjs";
const b = await chromium.launch({ headless: true });
const { ctx, page } = await newCtx(b, "sv.gioi");
await page.goto(`${BASE}/chat?course=01a1df27-62b5-7775-a346-5c602a43e2f9`); await page.waitForTimeout(4000);
console.log(page.url()); console.log((await text(page)).slice(0, 800));
console.log(await page.evaluate(() => [...document.querySelectorAll("button,textarea,input,a")].slice(0,40).map(e=>e.tagName+":"+(e.getAttribute("aria-label")||e.placeholder||e.innerText||"").slice(0,40)).join(" | ")));
await b.close();
