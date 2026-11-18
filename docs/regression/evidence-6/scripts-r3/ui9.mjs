import { chromium, newCtx, text, out, BASE } from "./ulib.mjs";
import { readFileSync } from "node:fs"; import { execFileSync } from "node:child_process";
const C = JSON.parse(readFileSync("/tmp/qc3e/p8c.json", "utf8"));
const psql = (sql) => execFileSync("docker", ["exec", "-i", "ep-r-postgres-1", "psql", "-U", "edupilot", "-d", "edupilot", "-At", "-c", sql], { encoding: "utf8" }).trim();
const py = (code) => execFileSync("python3", ["-c", `import sys;sys.path.insert(0,'/tmp/qc3e');from p8lib import *\n${code}`], { encoding: "utf8" }).trim();
const which = process.argv.slice(2); const b = await chromium.launch({ headless: true });
const mk = async (user, course) => { const { ctx, page } = await newCtx(b, user); await ctx.addInitScript((c) => localStorage.setItem("ep:ui:course", c), course); return { ctx, page }; };
const code = psql(`select class_code from courses where id='${C.U}'`);
const docs = async (p) => { for (let i = 0; i < 4; i++) { await p.goto(`${BASE}/documents?course=${C.U}`); await p.waitForTimeout(3000); if ((await text(p)).slice(0, 120).includes(code)) return; await p.evaluate((c) => localStorage.setItem("ep:ui:course", c), C.U); } throw new Error("không chọn được lớp"); };
const mkdoc = (name) => { py(`import time;open('/tmp/qc3e/ui/files/${name}.docx','wb').write(mkdocx(['${name} '+str(time.time())]+['Nội dung kiểm tra giao diện tài liệu của giảng viên. '*12]))`); return `/tmp/qc3e/ui/files/${name}.docx`; };
const row = (p, t) => p.locator("tr", { hasText: t }).first();
const ring = (p) => p.evaluate(() => { const e = document.activeElement; const s = getComputedStyle(e); return { tag: e.tagName, label: (e.getAttribute("aria-label") || e.innerText || "").slice(0, 40), ring: (s.outlineStyle !== "none" && parseFloat(s.outlineWidth) > 0) || (s.boxShadow && s.boxShadow !== "none") }; });
try {
  if (which.includes("08")) {
    const { page } = await mk("gv.s8f", C.U); await docs(page); const f = mkdoc("ui08k"); const name = "ui08k"; const log = []; let ok = true, chooser = null;
    page.on("filechooser", (c) => { chooser = c; });
    let found = false; await page.locator("body").click({ position: { x: 5, y: 5 } }).catch(() => {});
    for (let i = 0; i < 80 && !found; i++) { await page.keyboard.press("Tab"); const inf = await page.evaluate(() => { const e = document.activeElement; const z = e && e.closest("[class*=drop i],[data-part*=drop i],label"); return { t: (z ? z.innerText : e.innerText || "").slice(0, 60), tag: e.tagName }; }); if (/Kéo thả|chọn tệp/.test(inf.t)) { found = true; log.push(await ring(page)); } }
    await page.keyboard.press("Enter"); await page.waitForTimeout(800); if (chooser) await chooser.setFiles(f); else await page.locator("input[type=file]").first().setInputFiles(f);
    for (let i = 0; i < 40; i++) { if (await row(page, name).count()) break; await page.waitForTimeout(500); }
    for (let i = 0; i < 60; i++) { if (psql(`select count(*) from documents where filename='${name}.docx' and course_id='${C.U}'`) === "1") break; await page.waitForTimeout(1000); }
    await page.waitForTimeout(3000); let hit = 0; await page.evaluate(() => { document.activeElement && document.activeElement.blur(); window.scrollTo(0, 0); }); await page.locator("body").click({ position: { x: 5, y: 5 } }).catch(() => {});
    for (let i = 0; i < 220; i++) { await page.keyboard.press("Tab"); const lab = await page.evaluate(() => document.activeElement.getAttribute("aria-label") || ""); if (lab === `Loại của ${name}`) { hit = 1; log.push(await ring(page)); break; } }
    const t0 = psql(`select type||'|rag='||use_for_rag||'|vis='||visible_to_students from documents where filename='${name}.docx' and course_id='${C.U}'`);
    if (hit) { await page.keyboard.press("ArrowDown"); await page.keyboard.press("ArrowDown"); const tabs = []; await page.waitForTimeout(1500); await page.keyboard.press("Tab"); log.push(await ring(page)); await page.keyboard.press("Space"); await page.waitForTimeout(1200); await page.keyboard.press("Tab"); log.push(await ring(page)); await page.keyboard.press("Space"); await page.waitForTimeout(2000); }
    const t1 = psql(`select type||'|rag='||use_for_rag||'|vis='||visible_to_students from documents where filename='${name}.docx' and course_id='${C.U}'`);
    out("TC-P8-02-08", found && hit && t0 !== t1 && log.every((x) => x.ring), `chỉ bàn phím: Tab tới vùng thả=${found}, Enter ${chooser ? "mở hộp chọn tệp" : "(không bắt được hộp chọn tệp)"}; tới ô loại bằng Tab=${hit}; DB trước ${t0} → sau ${t1}; các phần tử nhận focus có viền thấy được: ${JSON.stringify(log)}`);
  }
  if (which.includes("15")) {
    const { page } = await mk("gv.s8f", C.U); await docs(page); const notice = async () => ({ n: (await text(page)).split("Lớp chưa có quy chế môn học.").length - 1, btn: await page.getByRole("button", { name: "Tải quy chế môn học" }).count() });
    const n0 = await notice(); let chooser = null; page.on("filechooser", (c) => { chooser = c; }); await page.getByRole("button", { name: "Tải quy chế môn học" }).first().click(); await page.waitForTimeout(1000);
    const f = mkdoc("ui15pol"); if (chooser) await chooser.setFiles(f); else await page.locator("input[type=file]").first().setInputFiles(f);
    for (let i = 0; i < 60; i++) { const st = psql(`select status||'|'||type from documents where filename='ui15pol.docx' and course_id='${C.U}'`); if (/^READY/.test(st)) break; await page.waitForTimeout(2000); }
    const dbs = psql(`select status||'|'||type from documents where filename='ui15pol.docx' and course_id='${C.U}'`); await docs(page); const n1 = await notice();
    const { page: sv } = await mk("sv.gioi", C.U); await sv.goto(`${BASE}/library?course=${C.U}`); await sv.waitForTimeout(3000); const tsv = await text(sv);
    out("TC-P8-02-15", n0.n === 1 && n0.btn >= 1 && /COURSE_POLICY/.test(dbs) && n1.n === 0 && !/Lớp chưa có quy chế/.test(tsv), `trước: ${n0.n} thông báo «Lớp chưa có quy chế môn học.» + nút «Tải quy chế môn học»=${n0.btn}; bấm nút → hộp chọn tệp ${chooser ? "mở" : "(không bắt được)"}, tệp tải với loại đã chọn: ${dbs}; sau READY + tải lại: thông báo=${n1.n}; màn sinh viên (thư viện) không có thông báo thiếu quy chế=${!/Lớp chưa có quy chế/.test(tsv)}`);
  }
  if (which.includes("19")) {
    const { page } = await mk("gv.s8f", C.U); await docs(page); const t = "ui06j2"; const id = psql(`select id from documents where filename='${t}.docx' and course_id='${C.U}'`);
    const n = psql(`select count(*) from content_chunks where document_id='${id}'`); const m = Number(psql(`select count(*) from document_courses where document_id='${id}'`)) + 1; const key = psql(`select blob_key from documents where id='${id}'`); const v0 = psql(`select 1`);
    const rag = () => execFileSync("docker", ["exec", "ep-r-redis-1", "redis-cli", "get", `ep:rag:ver:${C.U}`], { encoding: "utf8" }).trim(); const r0 = rag();
    await page.getByRole("button", { name: `Thao tác với ${t}` }).click(); await page.getByRole("menuitem", { name: "Xoá" }).click().catch(async () => { await page.getByText("Xoá", { exact: true }).first().click(); }); await page.waitForTimeout(1000);
    const dlg = await page.locator("[role=dialog]:visible,[role=alertdialog]:visible").first().innerText().catch(() => ""); const want = `Xoá ${t}? ${n} đoạn và ${m} lớp đang dùng tài liệu này sẽ mất nó.`;
    await page.getByRole("button", { name: /^Xoá/ }).last().click(); await page.waitForTimeout(6000);
    const c1 = [psql(`select count(*) from documents where id='${id}'`), psql(`select count(*) from content_chunks where document_id='${id}'`), psql(`select count(*) from document_courses where document_id='${id}'`)]; const r1 = rag();
    await page.waitForTimeout(30000); const gone = execFileSync("docker", ["exec", "ep-r-minio-1", "sh", "-c", `ls /data/edupilot/${key} 2>&1 | head -1`], { encoding: "utf8" });
    out("TC-P8-02-19", dlg.replace(/\s+/g, " ").includes(want) && c1.every((x) => x === "0") && Number(r1) > Number(r0) && /No such|cannot access/.test(gone), `hộp xác nhận: «${dlg.replace(/\s+/g, " ").slice(0, 120)}» (mong đợi «${want}»); sau xác nhận documents/content_chunks/document_courses = ${c1}; ep:rag:ver ${r0}→${r1}; object trong kho sau 36 s: ${gone.trim().slice(0, 60)}`);
  }
} finally { await b.close(); }
