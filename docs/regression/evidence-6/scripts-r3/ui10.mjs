import { chromium, newCtx, text, out, BASE } from "./ulib.mjs";
import { readFileSync } from "node:fs"; import { execFileSync } from "node:child_process";
const C = JSON.parse(readFileSync("/tmp/qc3e/p8c.json", "utf8"));
const psql = (sql) => execFileSync("docker", ["exec", "-i", "ep-r-postgres-1", "psql", "-U", "edupilot", "-d", "edupilot", "-At", "-c", sql], { encoding: "utf8" }).trim();
const name = "ui08m"; execFileSync("python3", ["-c", `import sys;sys.path.insert(0,'/tmp/qc3e');from p8lib import *\nimport time;open('/tmp/qc3e/ui/files/${name}.docx','wb').write(mkdocx(['${name} '+str(time.time())]+['Nội dung bàn phím. '*30]))`]);
const b = await chromium.launch({ headless: true });
const { ctx, page } = await newCtx(b, "gv.s8f"); await ctx.addInitScript((c) => localStorage.setItem("ep:ui:course", c), C.U);
await page.goto(`${BASE}/documents?course=${C.U}`); await page.waitForTimeout(3500);
const ring = () => page.evaluate(() => { const e = document.activeElement; const s = getComputedStyle(e); return { tag: e.tagName, label: (e.getAttribute("aria-label") || e.innerText || "").slice(0, 36), ring: (s.outlineStyle !== "none" && parseFloat(s.outlineWidth) > 0) || (s.boxShadow && s.boxShadow !== "none") }; });
let chooser = null; page.on("filechooser", (c) => { chooser = c; }); const log = []; let found = false;
for (let i = 0; i < 80 && !found; i++) { await page.keyboard.press("Tab"); const t = await page.evaluate(() => document.activeElement.innerText || ""); if (/Kéo thả/.test(t)) { found = true; log.push(await ring()); } }
await page.keyboard.press("Enter"); await page.waitForTimeout(1500); const opened = !!chooser; if (chooser) await chooser.setFiles(`/tmp/qc3e/ui/files/${name}.docx`);
for (let i = 0; i < 90; i++) { if (psql(`select count(*) from documents where filename='${name}.docx' and course_id='${C.U}'`) === "1") break; await page.waitForTimeout(1000); }
await page.waitForTimeout(2500); const t0 = psql(`select type||'|rag='||use_for_rag||'|vis='||visible_to_students from documents where filename='${name}.docx' and course_id='${C.U}'`);
let hit = false; for (let i = 0; i < 80 && !hit; i++) { await page.keyboard.press("Tab"); const lab = await page.evaluate(() => document.activeElement.getAttribute("aria-label") || ""); if (lab === `Loại của ${name}`) { hit = true; log.push(await ring()); } }
if (hit) { await page.keyboard.press("ArrowDown"); await page.keyboard.press("ArrowDown"); await page.waitForTimeout(1500); await page.keyboard.press("Tab"); log.push(await ring()); await page.keyboard.press("Space"); await page.waitForTimeout(1200); await page.keyboard.press("Tab"); log.push(await ring()); await page.keyboard.press("Space"); await page.waitForTimeout(2000); }
const t1 = psql(`select type||'|rag='||use_for_rag||'|vis='||visible_to_students from documents where filename='${name}.docx' and course_id='${C.U}'`);
out("TC-P8-02-08", found && opened && hit && t0 !== t1 && log.length >= 3 && log.every((x) => x.ring), `chỉ bàn phím: Tab tới vùng thả=${found}, Enter mở hộp chọn tệp=${opened}; Tab tới ô «Loại của ${name}»=${hit}; ↓↓ đổi loại, Tab+Space bật/tắt «Dùng cho AI», Tab+Space «Hiện cho sinh viên»; DB trước ${t0} → sau ${t1}; phần tử nhận focus (có viền thấy được): ${JSON.stringify(log)}`);
await b.close();
