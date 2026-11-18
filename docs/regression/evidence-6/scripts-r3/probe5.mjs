import { chromium, openChat, text } from "./ulib.mjs";
import { readFileSync } from "node:fs"; import { execFileSync } from "node:child_process";
const P = JSON.parse(readFileSync("prep.json","utf8"));
const psql = (sql) => execFileSync("docker", ["exec","-i","ep-r-postgres-1","psql","-U","edupilot","-d","edupilot","-At","-c",sql],{encoding:"utf8"}).trim();
const b = await chromium.launch({ headless: true }); const { page } = await openChat(b, "sv.gioi", P.Q);
for (const k of ["G","H"]) {
 const sg = psql(`select session_id from chat_messages where role='USER' and content='${P.qs[k]}' order by created_at desc limit 1`);
 await page.goto(`https://localhost:1143/chat?session=${sg}`); await page.waitForTimeout(3000);
 await page.getByText(/Nguồn tham khảo/).first().click(); await page.waitForTimeout(500); await page.getByText(k=="G"?"Bài giảng An ninh mạng":"QC3 tài liệu xoá").first().click(); await page.waitForTimeout(700);
 console.log(await page.evaluate(()=>{const e=[...document.querySelectorAll("section")].find(x=>/Nguồn tham khảo/.test(x.getAttribute("aria-label")||""));return e?e.outerHTML.replace(/class="[^"]*"/g,"").slice(0,900):"none"}));const t = await text(page); const i = t.indexOf("Nguồn tham khảo"); console.log(k, JSON.stringify(t.slice(i, i+500)));
}
await b.close();
