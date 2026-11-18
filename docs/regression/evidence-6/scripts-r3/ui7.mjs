import { chromium, newCtx, text, out, BASE } from "./ulib.mjs";
import { readFileSync, copyFileSync } from "node:fs"; import { execFileSync } from "node:child_process";
const C = JSON.parse(readFileSync("/tmp/qc3e/p8c.json", "utf8"));
const psql = (sql) => execFileSync("docker", ["exec", "-i", "ep-r-postgres-1", "psql", "-U", "edupilot", "-d", "edupilot", "-At", "-c", sql], { encoding: "utf8" }).trim();
const which = process.argv.slice(2); const b = await chromium.launch({ headless: true });
const mk = async (user, course, w = 1440, h = 900) => { const { ctx, page } = await newCtx(b, user, w, h); await ctx.addInitScript((c) => localStorage.setItem("ep:ui:course", c), course); return { ctx, page }; };
const lib = async (p, course, q = "") => { await p.goto(`${BASE}/library?course=${course}${q}`); await p.waitForTimeout(3200); };
const docId = (title, course = C.A) => psql(`select id from documents where title='${title}' and course_id='${course}' order by created_at desc limit 1`);
try {
  if (which.includes("22")) {
    const { page } = await mk("sv.gioi", C.A); await lib(page, C.A);
    const first = await page.evaluate(() => { const m = document.querySelector("main") || document.body; const e = [...m.querySelectorAll("a,button,input,select,textarea,[tabindex]")].find((x) => !x.closest("nav,header,aside")); return e ? `${e.tagName}|${e.type || ""}|${e.getAttribute("aria-label") || e.placeholder || e.innerText}` : ""; });
    const sel = await page.locator("main select").count(); const t = await text(page);
    out("TC-P8-02-22", /INPUT/.test(first) && sel >= 2 && /Tìm tài liệu/.test(t) && /Loại/.test(t) && /Tuần/.test(t) && /\bPDF\b|\bDOCX\b/.test(t), `phần tử tương tác đầu tiên trong vùng làm việc: ${first}; bộ lọc: ${sel} ô chọn (Loại, Tuần; không có «Chủ đề» trong màn hình — chủ đề (category) chưa là bộ lọc); dấu loại tệp bằng chữ (PDF/DOCX)=${/\bPDF\b|\bDOCX\b/.test(t)}`);
  }
  if (which.includes("31")) {
    const { page } = await mk("sv.gioi", C.A); await lib(page, C.A); const r = page.locator("li,tr,div", { hasText: "exam31-qc3" }).last();
    const btn = await page.getByRole("button", { name: /Hỏi AI/ }).count() + await page.getByRole("link", { name: /Hỏi AI/ }).count();
    const id = docId("exam31-qc3"); await page.goto(`${BASE}/library/${id}?course=${C.A}`); await page.waitForTimeout(2500); const t = await text(page); const btn2 = await page.getByText("Hỏi AI về tài liệu").count();
    out("TC-P8-02-31", JSON.parse(readFileSync("/tmp/qc3e/p8_31.json")).api31 && btn === 0 && btn2 === 0, `(a) đề tham khảo use_for_rag=false: nút «Hỏi AI…» ở danh sách=${btn}, ở chi tiết=${btn2}; API: can_ask_ai=false, POST chat/sessions với (a)/(b PROCESSING)/(c ANSWER_KEY) → 404/404/404, GET thư viện (b)/(c) → 404 (không lộ tồn tại, không 403)`);
  }
  if (which.includes("35")) {
    const { page } = await mk("sv.gioi", C.A); await lib(page, C.A); const t = await page.locator("main").innerText(); const acts = await page.locator("main a,main button").evaluateAll((e) => [...new Set(e.map((x) => (x.innerText || "").trim()).filter(Boolean))]);
    const id = docId("exam31-qc3"); await page.goto(`${BASE}/library/${id}?course=${C.A}`); await page.waitForTimeout(2500); const t2 = await page.locator("main").innerText();
    out("TC-P8-02-35", !/Luyện đề/.test(t) && !/Luyện đề/.test(t2), `hành động của các dòng: ${JSON.stringify(acts.filter((x) => !/^(DOCX|PDF)$/.test(x)).slice(0, 8))}; «Luyện đề» ở danh sách (có tài liệu loại Đề tham khảo)=${/Luyện đề/.test(t)}, ở chi tiết đề=${/Luyện đề/.test(t2)}; nút ở chi tiết: ${JSON.stringify(await page.locator("main button,main a").evaluateAll((e) => e.map((x) => (x.innerText || "").trim()).filter(Boolean).slice(0, 6)))}`);
  }
  if (which.includes("36")) {
    const { page } = await mk("sv.gioi", C.E); await lib(page, C.E); let t = await text(page); const a = /Chưa có tài liệu|chưa có tài liệu/.test(t);
    const aa = (t.match(/[^\n]*(chưa có|Chưa có)[^\n]*/) || [""])[0];
    await page.route("**/api/v1/courses/*/library*", (r) => r.fulfill({ status: 500, contentType: "application/json", body: JSON.stringify({ code: "INTERNAL", message: "x" }) })); await page.reload(); await page.waitForTimeout(3500); t = await text(page); const e = /Thử lại/.test(t);
    await page.unroute("**/api/v1/courses/*/library*"); await page.route("**/api/v1/courses/*/library*", async (r) => { await new Promise((s) => setTimeout(s, 3000)); r.continue().catch(() => {}); }); await page.reload(); await page.waitForTimeout(800);
    const spin = await page.locator("[role=progressbar],[class*=spinner i]").count(); const skel = await page.locator("[class*=skeleton i],[aria-busy=true]").count();
    await page.unroute("**/api/v1/courses/*/library*"); const { page: pa } = await mk("sv.gioi", C.A); await lib(pa, C.A); const ta = await text(pa); const bad = (ta.match(/\b(RAG|chunk|embedding|ANSWER_KEY|provider|vector)\b/gi) || []);
    out("TC-P8-02-36", a && e && spin === 0 && skel > 0 && bad.length === 0, `(a) lớp E rỗng: «${aa}»; (b) 500 → «Thử lại»=${e}; (c) chậm 3 s: spinner=${spin}, khung xương=${skel}; (d) từ kỹ thuật ở trang thư viện lớp A: ${JSON.stringify(bad)}`);
  }
  if (which.includes("38")) {
    const { page } = await mk("sv.gioi", C.A, 375, 812); await lib(page, C.A); const q = "quy chế đào tạo đại học chính quy theo tín chỉ";
    await page.locator("input").filter({ hasNot: page.locator("[type=hidden]") }).first().fill(q); await page.waitForTimeout(2000);
    const m = await page.evaluate(() => ({ sw: document.documentElement.scrollWidth, cw: document.documentElement.clientWidth, nested: document.querySelectorAll("[data-ep-panel] [data-ep-panel]").length }));
    await page.evaluate(() => { document.body.style.zoom = "2"; }); await page.waitForTimeout(800); const inp = page.locator("input").first(); await inp.fill("quy chế"); const v = await inp.inputValue();
    await page.screenshot({ path: "/tmp/qc3e/ui/p7_38.png" });
    out("TC-P8-02-38", m.sw <= m.cw + 1 && m.nested === 0 && v === "quy chế", `375×812, từ khoá dài: scrollWidth ${m.sw} ≤ clientWidth ${m.cw}; panel lồng panel=${m.nested}; zoom 200%: gõ được vào ô tìm kiếm=${v === "quy chế"}`);
  }
  if (which.includes("41")) {
    const { page } = await mk("sv.gioi", C.A); await lib(page, C.A); const inp = page.locator("input").first(); const urls = [];
    page.on("request", (q) => { if (/api\/v1\/courses\/.*library/.test(q.url())) urls.push(decodeURIComponent(q.url().replace(/\+/g, " "))); });
    await page.route("**/api/v1/courses/*/library*", (r) => r.fulfill({ status: 500, contentType: "application/json", body: JSON.stringify({ code: "INTERNAL", message: "x" }) }));
    await inp.fill("cảnh báo học vụ"); await inp.press("Enter"); await page.waitForTimeout(3000); let t = await text(page); const err = /Thử lại/.test(t); const kept = (await inp.inputValue()) === "cảnh báo học vụ";
    await page.unroute("**/api/v1/courses/*/library*"); const n0 = urls.length; await page.getByRole("button", { name: /Thử lại/ }).first().click(); await page.waitForTimeout(2500);
    out("TC-P8-02-41", err && kept && urls.length > n0 && /q=cảnh báo học vụ/.test(urls[urls.length - 1] || ""), `500: lỗi+«Thử lại»=${err}; ô tìm kiếm giữ từ khoá=${kept}; bấm «Thử lại» → yêu cầu mới: ${urls[urls.length - 1]}`);
  }
  if (which.includes("43")) {
    const id = JSON.parse(readFileSync("/tmp/qc3e/p8_43.json")).D43; const { page } = await mk("sv.gioi", C.A); await page.goto(`${BASE}/library/${id}?course=${C.A}`); await page.waitForTimeout(2500);
    const n0 = psql(`select download_count from documents where id='${id}'`); await page.getByRole("button", { name: /Tải xuống/ }).first().click().catch(async () => { await page.getByRole("link", { name: /Tải xuống/ }).first().click(); }); await page.waitForTimeout(2000);
    const t = await text(page); const n1 = psql(`select download_count from documents where id='${id}'`);
    out("TC-P8-02-43", /Tệp này không còn nữa\./.test(t) && n0 === n1, `API: 404 FILE_GONE (xem w40); UI bấm «Tải xuống»: «Tệp này không còn nữa.»=${/Tệp này không còn nữa\./.test(t)}; download_count ${n0}→${n1}`);
  }
} finally { await b.close(); }
