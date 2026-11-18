import { chromium, newCtx, text, out, BASE } from "./ulib.mjs";
import { readFileSync, copyFileSync } from "node:fs"; import { execFileSync } from "node:child_process";
const C = JSON.parse(readFileSync("/tmp/qc3e/p8c.json", "utf8"));
const psql = (sql) => execFileSync("docker", ["exec", "-i", "ep-r-postgres-1", "psql", "-U", "edupilot", "-d", "edupilot", "-At", "-c", sql], { encoding: "utf8" }).trim();
const which = process.argv.slice(2); const b = await chromium.launch({ headless: true });
const mk = async (user, course, w = 1440, h = 900) => { const { ctx, page } = await newCtx(b, user, w, h); await ctx.addInitScript((c) => localStorage.setItem("ep:ui:course", c), course); return { ctx, page }; };
const docs = async (p, course) => { const code = psql(`select class_code from courses where id='${course}'`); for (let i = 0; i < 4; i++) { await p.goto(`${BASE}/documents?course=${course}`); await p.waitForTimeout(3000); const head = (await text(p)).slice(0, 120); if (head.includes(code)) return; await p.evaluate((c) => localStorage.setItem("ep:ui:course", c), course); } throw new Error("không chọn được lớp " + code); };
const row = (p, t) => p.locator("tr", { hasText: t }).first();
const fresh = (n) => { const f = `/tmp/qc3e/ui/files/${n}-${Date.now()}.docx`; copyFileSync(`/tmp/qc3e/ui/files/u${n.slice(-1)}.docx`, f); return f; };
const mkdoc = (name) => { const ex = execFileSync("python3", ["-c", `import sys;sys.path.insert(0,'/tmp/qc3e');from p8lib import *;import time;open('/tmp/qc3e/ui/files/${name}.docx','wb').write(mkdocx(['${name} '+str(time.time())]+['Nội dung kiểm tra giao diện tài liệu của giảng viên. '*12]))`]); return `/tmp/qc3e/ui/files/${name}.docx`; };
const sw = (r, name) => r.getByLabel(new RegExp(name)).first();
const state = async (el) => (await el.isChecked().catch(async () => await el.getAttribute("aria-checked")));
try {
  if (which.includes("11")) {
    const { page } = await mk("teacher", C.A); await docs(page, C.A); const t = "mo-qc3"; const r = row(page, t); const el = sw(r, "Dùng cho AI");
    const q = () => psql(`select use_for_rag from documents where title='${t}' and course_id='${C.A}' order by created_at desc limit 1`); const v0 = q(); const b0 = await state(el);
    await el.click(); const t0 = Date.now(); const b1 = await state(el); const dt = Date.now() - t0; await page.waitForTimeout(700); const has = (await text(page)).includes("Hoàn tác");
    await page.waitForTimeout(6000); const gone = !(await text(page)).includes("Hoàn tác"); const v1 = q();
    await el.click(); await page.waitForTimeout(700); await page.getByRole("button", { name: "Hoàn tác" }).first().click().catch(() => {}); await page.waitForTimeout(1500); const v2 = q(); const b2 = await state(el);
    out("TC-P8-02-11", String(b0) !== String(b1) && has && gone && v1 !== v0 && v2 === v1, `công tắc «Dùng cho AI» ${b0}→${b1} ngay sau click (${dt} ms, trước khi API xong); «Đã đổi · Hoàn tác» hiện=${has}; biến mất sau ~6 s=${gone}; DB ${v0}→${v1}; đổi lần hai rồi bấm Hoàn tác ≤5 s → DB ${v2}, công tắc ${b2}`);
    if (q() !== v0) { await el.click(); await page.waitForTimeout(1500); }
  }
  if (which.includes("40")) {
    const { page } = await mk("teacher", C.A); await docs(page, C.A); const t = "mo-qc3"; const r = row(page, t); const el = sw(r, "Dùng cho AI");
    const q = () => psql(`select use_for_rag from documents where title='${t}' and course_id='${C.A}' order by created_at desc limit 1`); const v0 = q();
    await page.route("**/api/v1/courses/*/documents/*", (rt) => rt.request().method() === "PATCH" ? rt.fulfill({ status: 500, contentType: "application/json", body: JSON.stringify({ code: "INTERNAL", message: "x" }) }) : rt.continue());
    const before = new Set((await text(page)).split("\n")); const a0 = await state(el); await el.click(); await page.waitForTimeout(2000); const a1 = await state(el); const tx = await text(page); const v1 = q(); const msg = tx.split("\n").filter((l) => l.trim() && !before.has(l)).join(" / ");
    out("TC-P8-02-40", a0 === a1 && v0 === v1 && !/Đã đổi · Hoàn tác/.test(tx) && msg !== "", `PATCH 500: công tắc ${a0}→${a1} (hoàn lại); DB ${v0}→${v1}; «Đã đổi · Hoàn tác» hiện=${/Đã đổi · Hoàn tác/.test(tx)}; thông báo: ${JSON.stringify(msg)}`);
  }
  if (which.includes("05")) {
    const { page } = await mk("teacher", C.A); await docs(page, C.A); const f = mkdoc("ui05f"); const name = "ui05f"; let dialogs = 0; page.on("dialog", () => dialogs++);
    const zb = await page.getByText("Kéo thả hoặc chọn tệp").first().boundingBox(); const tb = await page.locator("table").first().boundingBox(); const dz = { zy: zb.y, ty: tb.y };
    await page.route("**/edupilot/**", async (rt) => { if (rt.request().method() !== "PUT") return rt.continue(); await new Promise((s2) => setTimeout(s2, 2500)); await rt.continue().catch(() => {}); });
    await page.locator("input[type=file]").first().setInputFiles(f); let dlgMax = 0; const seen = []; const t0 = Date.now();
    while (Date.now() - t0 < 120000) { if (!dlgMax) dlgMax = 0; dlgMax = Math.max(dlgMax, await page.locator("[role=dialog]:visible").count()); const lines = (await page.locator("main").innerText()).split("\n").map((l) => l.trim()).filter(Boolean); const k = lines.findIndex((l) => l.startsWith(name)); const near = k >= 0 ? lines.slice(k, k + 12).join(" | ") : ""; const st = (near.match(/Đang tải[^|]*|Đang xử lý[^|]*|Sẵn sàng|Lỗi[^|]*/) || [""])[0].trim(); if (st && seen[seen.length - 1] !== st.replace(/\d+ %/, "n %")) seen.push(st.replace(/\d+ %/, "n %")); if (/Sẵn sàng/.test(st)) break; await page.waitForTimeout(250); }
    const dlg = await page.locator("[role=dialog]:visible").count();
    await page.goto(`${BASE}/calendar?course=${C.A}`); await page.waitForTimeout(31000); await docs(page, C.A); const still = await row(page, name).count();
    out("TC-P8-02-05", dz && dz.zy < dz.ty && dlgMax === 0 && dialogs === 0 && seen[0] === "Đang tải" && seen.includes("Sẵn sàng") && still > 0, `vùng thả (y=${dz && Math.round(dz.zy)}) nằm trên bảng (y=${dz && Math.round(dz.ty)}); hộp thoại hiện thấy trong lúc tải=${dlgMax}; trạng thái thấy trên hàng mới: ${JSON.stringify(seen)}; sang /calendar 31 s rồi quay lại: hàng vẫn có=${still}`);
  }
  if (which.includes("06")) {
    const { page } = await mk("gv.s8f", C.U); await docs(page, C.U); const fs = [0, 1, 2, 3].map((i) => mkdoc("ui06j" + i)); let inflight = 0, max = 0;
    await page.route("**/edupilot/**", async (rt) => { if (rt.request().method() !== "PUT") return rt.continue(); inflight++; max = Math.max(max, inflight); await new Promise((s) => setTimeout(s, 3500)); await rt.continue().catch(() => {}); inflight--; });
    await page.locator("input[type=file]").first().setInputFiles(fs); await page.waitForTimeout(1500);
    const q4 = await page.locator("li[data-part=upload-row]", { hasText: "ui06j3" }).innerText().catch(() => ""); let r0 = null; for (let i = 0; i < 40; i++) { r0 = row(page, "ui06j0"); if (await r0.count()) break; await page.waitForTimeout(500); }
    const sel = await r0.locator("select").count(); const sws = await r0.locator("input[type=checkbox],[role=switch]").count(); const enabled = await r0.locator("select,input[type=checkbox],[role=switch]").evaluateAll((e) => e.every((x) => !x.disabled)); const st0 = (await r0.innerText()).replace(/\s+/g, " ");
    await r0.locator("select").first().selectOption("EXAM_PAPER"); await sw(r0, "Hiện cho sinh viên").click().catch(() => {}); await sw(r0, "Dùng cho AI").click().catch(() => {});
    await page.waitForTimeout(30000); const ty = psql(`select type||'|rag='||use_for_rag||'|vis='||visible_to_students from documents where filename like 'ui06j0%' and course_id='${C.U}' order by created_at desc limit 1`);
    out("TC-P8-02-06", max <= 3 && max >= 2 && /Chờ tải/.test(q4) && sel > 0 && sws >= 2 && enabled && /EXAM_PAPER/.test(ty), `4 tệp: số PUT chạy song song tối đa=${max} (≤3, tệp thứ 4 xếp hàng); tệp thứ 4 hiện «${q4.replace(/\s+/g, ' ')}» khi 3 tệp kia đang tải; hàng bảng của tệp đầu ngay khi tạo xong («${st0.slice(0, 70)}»): ${sel} ô chọn loại + ${sws} công tắc, đều bật được=${enabled}; sau khi đổi loại → Đề tham khảo và bật/tắt hai cờ, DB: ${ty}`);
  }
  if (which.includes("39")) {
    const { ctx, page } = await mk("gv.s8f", C.U); await docs(page, C.U); const f = mkdoc("ui39j"); let offlined = false;
    await page.route("**/edupilot/**", async (rt) => { if (rt.request().method() !== "PUT") return rt.continue(); await new Promise((s) => setTimeout(s, 4000)); await rt.continue().catch(() => {}); });
    await page.locator("input[type=file]").first().setInputFiles(f); await page.waitForTimeout(1500); await ctx.setOffline(true); await page.waitForTimeout(6000); const ul = page.locator("li[data-part=upload-row]", { hasText: "ui39j" }); const t1 = await ul.innerText().catch(() => "");
    await ctx.setOffline(false); await page.unroute("**/edupilot/**"); const btn = ul.getByRole("button", { name: "Tải lại" }); const has = await btn.count(); await btn.click().catch(() => {}); await page.waitForTimeout(35000);
    const n = psql(`select count(*) from documents where filename like 'ui39j%' and course_id='${C.U}'`); const t2 = await row(page, "ui39j").innerText().catch(() => "");
    out("TC-P8-02-39", /Lỗi/.test(t1) && /ui39j/.test(t1) && has > 0 && n === "1", `PUT bị ngắt mạng: hàng «${t1.replace(/\s+/g, " ").slice(0, 80)}» (giữ tên tệp, trạng thái Lỗi); nút «Tải lại»=${has}; sau Tải lại: documents cùng tên=${n}; hàng: «${t2.replace(/\s+/g, " ").slice(0, 60)}»`);
  }
} finally { await b.close(); }
