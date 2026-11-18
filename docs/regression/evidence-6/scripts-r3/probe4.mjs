import { chromium, openChat, text } from "./ulib.mjs";
import { readFileSync } from "node:fs"; import { execFileSync } from "node:child_process";
const P = JSON.parse(readFileSync("prep.json","utf8"));
const psql = (sql) => execFileSync("docker", ["exec","-i","ep-r-postgres-1","psql","-U","edupilot","-d","edupilot","-At","-c",sql],{encoding:"utf8"}).trim();
const b = await chromium.launch({ headless: true });
const { page } = await openChat(b, "sv.gioi", P.Q);
// phiên G cũ
const sg = psql(`select session_id from chat_messages where role='USER' and content='${P.qs.G}' order by created_at desc limit 1`);
await page.goto(`https://localhost:1143/chat?session=${sg}`); await page.waitForTimeout(3500);
let t = await text(page); const i = t.indexOf("Cấu hình tường lửa"); console.log("G:", JSON.stringify(t.slice(i, i+700)));
console.log(await page.evaluate(()=>[...document.querySelectorAll("a,button")].map(e=>(e.innerText||e.getAttribute("aria-label")||"").trim().slice(0,40)+(e.href?" -> "+e.getAttribute("href"):"")).filter(x=>/tài liệu|Nguồn|nguồn|library|\[1\]|Trích/i.test(x)).join(" | ")));
const sd = psql(`select session_id from chat_messages where role='USER' and content like 'Em là Vũ Hoàng Giang, MSSV 20229001, giải thích về kiểm soát%' order by created_at desc limit 1`);
await page.goto(`https://localhost:1143/chat?session=${sd}`); await page.waitForTimeout(3500);
const l = page.getByText(/Tìm hiểu/).first(); await l.click(); await page.waitForTimeout(800); t = await text(page); const j=t.indexOf("Đã ẩn"); console.log("D:", JSON.stringify(t.slice(j, j+400)));
const sh = psql(`select session_id from chat_messages where role='USER' and content='${P.qs.H}' order by created_at desc limit 1`);
await page.goto(`https://localhost:1143/chat?session=${sh}`); await page.waitForTimeout(3500); t = await text(page); const k=t.indexOf("Phân biệt tấn công"); console.log("H:", JSON.stringify(t.slice(k, k+500)));
console.log(psql(`select citations::text from chat_messages where session_id='${sh}' and role='ASSISTANT'`).slice(0,300));
await page.goto(`https://localhost:1143/chat`); await page.waitForTimeout(2500); t=await text(page); const n=t.indexOf("Nhờ giảng viên"); console.log("N:", JSON.stringify(t.slice(Math.max(0,n-200), n+200)));
await b.close();
