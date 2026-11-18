import { chromium, newCtx, text, BASE } from "./ulib.mjs";
import { readFileSync, copyFileSync } from "node:fs"; import { execFileSync } from "node:child_process";
const C = JSON.parse(readFileSync("/tmp/qc3e/p8c.json", "utf8"));
execFileSync("python3", ["-c", `import sys;sys.path.insert(0,'/tmp/qc3e');from p8lib import *;import time;open('/tmp/qc3e/ui/files/pr9.docx','wb').write(mkdocx(['pr9 '+str(time.time())]+['Nội dung '*60]))`]);
const b = await chromium.launch({ headless: true });
const { ctx, page } = await newCtx(b, "teacher"); await ctx.addInitScript((c) => localStorage.setItem("ep:ui:course", c), C.A);
await page.goto(`${BASE}/documents?course=${C.A}`); await page.waitForTimeout(3000);
const reqs = []; page.on("request", (q) => { if (q.method() !== "GET") reqs.push(q.method() + " " + q.url().slice(0, 90)); });
await page.route("**/edupilot/**", async (rt) => { await new Promise((s) => setTimeout(s, 2500)); await rt.continue().catch(() => {}); });
await page.locator("input[type=file]").first().setInputFiles("/tmp/qc3e/ui/files/pr9.docx"); const seen = new Set();
for (let i = 0; i < 30; i++) { const t = await page.locator("main").innerText(); t.split("\n").filter((l) => /pr9|Đang|Sẵn|tải/i.test(l)).forEach((l) => seen.add(i + ":" + l.slice(0, 80))); await page.waitForTimeout(300); }
console.log([...seen].join("\n")); console.log(reqs.join("\n"));
await b.close();
