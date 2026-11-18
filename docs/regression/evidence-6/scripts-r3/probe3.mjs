import { chromium, openChat, text, composer } from "./ulib.mjs";
const b = await chromium.launch({ headless: true });
const { page } = await openChat(b, "sv.gioi", "01a1ef4f-0195-70a8-bc04-3e0b37610973");
console.log(await page.evaluate(()=>[...document.querySelectorAll("button")].map(e=>(e.getAttribute("aria-label")||e.innerText||"").trim().slice(0,30)).filter(Boolean).slice(-12).join("|")));
await composer(page).fill("Xin chào"); console.log(await page.evaluate(()=>[...document.querySelectorAll("button")].map(e=>(e.getAttribute("aria-label")||e.innerText||"").trim().slice(0,30)).filter(Boolean).slice(-6).join("|")));
console.log((await text(page)).slice(-500));
await b.close();
