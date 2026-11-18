import { chromium, newCtx, text, out, BASE } from "./ulib.mjs";
import { readFileSync } from "node:fs"; import { execFileSync } from "node:child_process";
const C = JSON.parse(readFileSync("/tmp/qc3e/p8c.json", "utf8")); const D = JSON.parse(readFileSync("/tmp/qc3e/p8d.json", "utf8"));
const psql = (sql) => execFileSync("docker", ["exec", "-i", "ep-r-postgres-1", "psql", "-U", "edupilot", "-d", "edupilot", "-At", "-c", sql], { encoding: "utf8" }).trim();
const F = (n) => `/tmp/qc3e/ui/files/${n}`;
const which = process.argv.slice(2);
const b = await chromium.launch({ headless: true });
const teacher = async (course, w = 1440, h = 900) => { const { ctx, page } = await newCtx(b, "teacher", w, h); await ctx.addInitScript((c) => localStorage.setItem("ep:ui:course", c), course); return { ctx, page }; };
const student = async (user, course, w = 1440, h = 900) => { const { ctx, page } = await newCtx(b, user, w, h); await ctx.addInitScript((c) => localStorage.setItem("ep:ui:course", c), course); return { ctx, page }; };
const docs = async (p, course) => { await p.goto(`${BASE}/documents?course=${course}`); await p.waitForTimeout(3000); };
const row = (p, title) => p.locator("tr", { hasText: title }).first();
try {
  if (which.includes("02")) { // chỉ đọc cho tài liệu chia sẻ
    const { page } = await teacher(C.B); await docs(page, C.B);
    const t = await text(page); const r = page.locator("tr", { hasText: "Chia sẻ từ" }).first();
    const n = await page.locator("tr", { hasText: "Chia sẻ từ" }).count();
    const ctrl = await r.locator("button:not([class*=name]),input,select,[role=switch],[role=checkbox]").evaluateAll((els) => els.map((e) => `${e.tagName}|${e.getAttribute("aria-label") || e.innerText || ""}|${e.disabled || e.getAttribute("aria-disabled") === "true"}`));
    const did = psql(`select dc.document_id from document_courses dc where dc.course_id='${C.B}' limit 1`);
    const tk = execFileSync("python3", ["-c", "import sys;sys.path.insert(0,'/tmp/qc3e');from lib import *;print(login('teacher'))"], { encoding: "utf8" }).trim();
    const r2 = await page.request.patch(`${BASE}/api/v1/courses/${C.B}/documents/${did}`, { headers: { Authorization: `Bearer ${tk}` }, data: { use_for_rag: false, version: 1 }, ignoreHTTPSErrors: true });
    const j = await r2.json().catch(() => ({}));
    out("TC-P8-02-02", n > 0 && ctrl.length > 0 && ctrl.every((x) => x.endsWith("true")) && /Chia sẻ từ QC3P8A/.test(t) && [403, 409].includes(r2.status()), `lớp B: ${n} hàng «Chia sẻ từ QC3P8A»; điều khiển trong hàng đầu: ${ctrl.length} (${JSON.stringify(ctrl)}); PATCH trực tiếp → ${r2.status()} ${j.code}`);
  }
  if (which.includes("04")) {
    const { page } = await teacher(C.E); await docs(page, C.E); let t = await text(page);
    const a = /Lớp chưa có tài liệu/.test(t) && (await page.getByRole("button", { name: "Tải tài liệu lên" }).count()) > 0;
    await page.route("**/api/v1/courses/*/documents*", (r) => r.fulfill({ status: 500, contentType: "application/json", body: JSON.stringify({ code: "INTERNAL", message: "x" }) }));
    await page.reload(); await page.waitForTimeout(3500); t = await text(page); let calls = 0; page.on("request", (q) => { if (/\/documents/.test(q.url()) && q.method() === "GET") calls++; });
    const errState = /Thử lại/.test(t); await page.getByRole("button", { name: /Thử lại/ }).first().click().catch(() => {}); await page.waitForTimeout(1500);
    await page.unroute("**/api/v1/courses/*/documents*");
    await page.route("**/api/v1/courses/*/documents*", async (r) => { await new Promise((s) => setTimeout(s, 3000)); r.continue(); }); await page.reload(); await page.waitForTimeout(800);
    const spinner = await page.locator("[role=progressbar],[class*=spinner i]").count(); const skel = await page.locator("[class*=skeleton i],[aria-busy=true]").count();
    out("TC-P8-02-04", a && errState && calls >= 1 && spinner === 0 && skel > 0, `(a) rỗng: «Lớp chưa có tài liệu»+nút «Tải tài liệu lên»=${a}; (b) 500 → lỗi+«Thử lại»=${errState}, bấm gọi lại API=${calls} yêu cầu; (c) chậm 3 s: spinner=${spinner}, khung xương=${skel}`);
  }
  if (which.includes("07")) {
    const { page } = await teacher(C.A); await docs(page, C.A); const reqs = []; page.on("request", (q) => { if (/uploads\/presign|\/edupilot\//.test(q.url())) reqs.push(q.method() + " " + q.url().slice(0, 80)); });
    const inp = page.locator("input[type=file]").first();
    await inp.setInputFiles(F("virus.exe")); await page.waitForTimeout(1500); const t1 = await text(page);
    await inp.setInputFiles(F("big60.pdf")); await page.waitForTimeout(1500); const t2 = await text(page);
    const msg = "Dùng PDF, DOCX hoặc PPTX, tối đa 50 MB."; const base = JSON.stringify([t1.slice(0,0)]); await page.screenshot({ path: "/tmp/qc3e/ui/p6_07.png" });
    out("TC-P8-02-07", reqs.length === 0 && t1.includes(msg) && t2.includes(msg), `.exe và PDF 60 MiB (đoạn thông báo trên trang: ${JSON.stringify((t2.match(/[^\n]*(PDF|DOCX|50 MB|không)[^\n]*/g)||[]).slice(0,6))}): yêu cầu presign/PUT=${reqs.length} ${JSON.stringify(reqs)}; thông báo đúng chữ sau .exe=${t1.includes(msg)}, sau 60 MiB=${t2.includes(msg)}`);
  }
  if (which.includes("09")) {
    const { page } = await teacher(C.A); await docs(page, C.A); const r = row(page, "dapan-qc3"); const t = await r.innerText();
    const lines = t.split("\n").map((x) => x.trim()).filter(Boolean);
    const sw = await r.locator("[role=switch],[role=checkbox],input[type=checkbox]").count();
    out("TC-P8-02-09", /Không hiển thị cho sinh viên/.test(t) && /Không dùng cho AI của sinh viên/.test(t), `hàng ANSWER_KEY: ${JSON.stringify(lines)}; công tắc trong hàng=${sw}`);
  }
  if (which.includes("11")) {
    const { page } = await teacher(C.A); await docs(page, C.A); const title = "mo-qc3"; const r = row(page, title);
    const sw = r.locator(`[aria-label^="Dùng cho AI"]`).first(); const before = await sw.getAttribute("aria-checked").catch(() => null) ?? await sw.isChecked().catch(() => null);
    const dbq = () => psql(`select use_for_rag from documents where title='${title}' and course_id='${C.A}'`); const v0 = dbq();
    await sw.click(); const t0 = Date.now(); const after = await sw.getAttribute("aria-checked").catch(() => null) ?? await sw.isChecked().catch(() => null); const quick = Date.now() - t0;
    await page.waitForTimeout(600); const undo = await page.getByText(/Đã đổi/).count(); const hasUndo = (await text(page)).includes("Hoàn tác");
    await page.waitForTimeout(6000); const gone = !(await text(page)).includes("Hoàn tác");
    const v1 = dbq(); // đổi lần hai rồi Hoàn tác
    await sw.click(); await page.waitForTimeout(700); await page.getByRole("button", { name: "Hoàn tác" }).first().click().catch(() => {}); await page.waitForTimeout(1500); const v2 = dbq();
    // khôi phục
    if (v2 !== v0) await sw.click();
    out("TC-P8-02-11", String(before) !== String(after) && hasUndo && gone && v1 !== v0 && v2 === v1, `công tắc «Dùng cho AI» đổi ngay (${before}→${after}, ${quick} ms sau click); «Đã đổi · Hoàn tác» hiện=${hasUndo}, biến mất sau ~6 s=${gone}; DB ${v0}→${v1}; đổi lần hai rồi Hoàn tác trong 5 s → DB ${v2}`);
  }
  if (which.includes("40")) {
    const { page } = await teacher(C.A); await docs(page, C.A); const title = "mo-qc3"; const r = row(page, title); const v0 = psql(`select use_for_rag from documents where title='${title}' and course_id='${C.A}'`);
    await page.route("**/api/v1/courses/*/documents/*", (rt) => rt.request().method() === "PATCH" ? rt.fulfill({ status: 500, contentType: "application/json", body: JSON.stringify({ code: "INTERNAL", message: "x" }) }) : rt.continue());
    const sw = r.locator(`[aria-label^="Dùng cho AI"]`).first(); const a0 = await sw.getAttribute("aria-checked"); await sw.click(); await page.waitForTimeout(1500);
    const a1 = await sw.getAttribute("aria-checked"); const t = await text(page); const v1 = psql(`select use_for_rag from documents where title='${title}' and course_id='${C.A}'`);
    out("TC-P8-02-40", a0 === a1 && v0 === v1 && !/Đã đổi · Hoàn tác/.test(t) && /Chưa|không|lỗi/i.test(t), `PATCH 500: công tắc ${a0}→${a1} (hoàn lại); DB ${v0}→${v1}; «Đã đổi · Hoàn tác» hiện=${/Đã đổi · Hoàn tác/.test(t)}; thông báo lỗi: ${JSON.stringify((t.match(/[^\n]*(Chưa|lỗi|Không thể)[^\n]*/i) || [""])[0])}`);
  }
} finally { await b.close(); }
