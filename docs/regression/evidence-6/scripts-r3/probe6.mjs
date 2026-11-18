import { chromium, openChat, text } from "./ulib.mjs";
const b = await chromium.launch({ headless: true });
const { page } = await openChat(b, "sv.gioi", "01a1df27-62b5-7775-a346-5c602a43e2f9");
await page.goto("https://localhost:1143/threads"); await page.waitForTimeout(3500);
console.log((await text(page)).slice(0,900).replace(/\n+/g," | "));
console.log(await page.evaluate(()=>[...document.querySelectorAll("button,textarea,input")].map(e=>e.tagName[0]+":"+(e.getAttribute("aria-label")||e.placeholder||e.innerText||"").trim().slice(0,30)).filter(x=>x.length>2).slice(0,30).join("|")));
await page.getByRole("button",{name:/Đặt câu hỏi/}).first().click().catch(e=>console.log("noask")); await page.waitForTimeout(1200);
console.log(await page.evaluate(()=>[...document.querySelectorAll("button,textarea,input")].map(e=>e.tagName[0]+":"+(e.getAttribute("aria-label")||e.placeholder||e.innerText||"").trim().slice(0,40)).filter(x=>x.length>2).slice(-14).join("|")));
console.log(await page.evaluate(()=>[...document.querySelectorAll("textarea,input")].map(e=>e.tagName+"|"+e.type+"|"+(e.placeholder||"")+"|"+(e.getAttribute("aria-label")||"")+"|"+(e.closest("label")?.innerText||"").slice(0,30)).join("\n")));await b.close();
