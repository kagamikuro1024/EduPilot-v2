import { chromium, newCtx, text, BASE } from "./ulib.mjs";
import { readFileSync } from "node:fs";
const C = JSON.parse(readFileSync("/tmp/qc3e/p8c.json", "utf8"));
const b = await chromium.launch({ headless: true });
const { ctx, page } = await newCtx(b, "gv.s8f"); await ctx.addInitScript((c) => localStorage.setItem("ep:ui:course", c), C.U);
await page.goto(`${BASE}/documents?course=${C.U}`); await page.waitForTimeout(3500);
let chooser = null; page.on("filechooser", (c) => { chooser = c; });
let found = false; for (let i = 0; i < 80 && !found; i++) { await page.keyboard.press("Tab"); const t = await page.evaluate(() => { const e = document.activeElement; return (e.innerText || "").slice(0, 40) + "|" + e.tagName + "|" + (e.getAttribute("role") || "") + "|" + e.className.slice(0, 40); }); if (/Kéo thả/.test(t)) { found = true; console.log("focus", i, t); } }
await page.keyboard.press("Enter"); await page.waitForTimeout(1500); console.log("chooser", !!chooser);
await page.keyboard.press("Space"); await page.waitForTimeout(1500); console.log("chooser after space", !!chooser);
if (chooser) { await chooser.setFiles("/tmp/qc3e/ui/files/u0.docx"); await page.waitForTimeout(3000); console.log((await page.locator("main").innerText()).split("\n").filter((l) => /u0|Đang|Chờ/.test(l)).slice(0, 6)); }
await b.close();
