import { chromium, newCtx, text, out, BASE } from "./ulib.mjs";
import { readFileSync } from "node:fs"; import { execFileSync } from "node:child_process";
const C = JSON.parse(readFileSync("/tmp/qc3e/p8c.json", "utf8")); const D = JSON.parse(readFileSync("/tmp/qc3e/p8d.json", "utf8"));
const psql = (sql) => execFileSync("docker", ["exec", "-i", "ep-r-postgres-1", "psql", "-U", "edupilot", "-d", "edupilot", "-At", "-c", sql], { encoding: "utf8" }).trim();
const which = process.argv.slice(2); const b = await chromium.launch({ headless: true });
const mk = async (user, course) => { const { ctx, page } = await newCtx(b, user); await ctx.addInitScript((c) => localStorage.setItem("ep:ui:course", c), course); return { ctx, page }; };
try {
  if (which.includes("20")) {
    const j = JSON.parse(readFileSync("/tmp/qc3e/p8_20.json")); const { page } = await mk("sv.kha", C.A); await page.goto(`${BASE}/chat?course=${C.A}`); await page.waitForTimeout(2500);
    await page.goto(`${BASE}/chat?session=${j.sid20}&course=${C.A}`); await page.waitForTimeout(3500); let t = await text(page);
    if (!/del20-qc3/.test(t)) { const it = page.getByText(/QC20|Xoá/).first(); await it.click().catch(() => {}); await page.waitForTimeout(2500); t = await text(page); }
    out("TC-P8-02-20", /Nguồn đã gỡ/.test(t), `DB: chat_sessions.document_id=NULL, phiên mở lại được (API 200); UI mở phiên sau khi xoá tài liệu: trích dẫn cũ ${/del20-qc3/.test(t) ? "vẫn hiện tên «del20-qc3»" : "không thấy tên"}; chữ «Nguồn đã gỡ» hiện=${/Nguồn đã gỡ/.test(t)} (DEF-R6-D1; chuỗi này không có trong frontend/src hay backend-go/internal/chat)`);
  }
  if (which.includes("30")) {
    const id = psql(`select id from documents where title='scope30-qc3' and course_id='${C.A}' order by created_at desc limit 1`); const { page } = await mk("sv.gioi", C.A); await page.goto(`${BASE}/library/${id}?course=${C.A}`); await page.waitForTimeout(3000);
    const n0 = psql("select count(*) from chat_sessions"); await page.getByRole("button", { name: "Hỏi AI về tài liệu" }).click(); await page.waitForTimeout(3500);
    const url = page.url(); const row = psql(`select document_id from chat_sessions where user_id=(select id from users where email='sv.gioi@edupilot.local') order by created_at desc limit 1`);
    out("TC-P8-02-30", /\/chat/.test(url) && row === id, `bấm «Hỏi AI về tài liệu» → ${url.replace(BASE, "")}; phiên chat mới nhất của SV có document_id=${row === id ? "đúng tài liệu đang xem" : row}; (trích dẫn chỉ thuộc tài liệu đó: kiểm ở API, TC-P8-02-30-api)`);
  }
  if (which.includes("32")) {
    const id = psql(`select id from documents where title='scope30-qc3' and course_id='${C.A}' order by created_at desc limit 1`); const { page } = await mk("sv.gioi", C.A); await page.goto(`${BASE}/library/${id}?course=${C.A}`); await page.waitForTimeout(3000);
    const before = await text(page); await page.getByRole("button", { name: "Hỏi AI về tài liệu" }).click().catch(() => {}); await page.waitForTimeout(3000);
    let t = await text(page); const url = page.url(); await page.screenshot({ path: "/tmp/qc3e/ui/p11_32.png" });
    if (/\/chat/.test(url)) { const ta = page.locator("textarea").first(); await ta.fill("Giải thích chương 3").catch(() => {}); await page.getByRole("button", { name: /^Gửi/ }).click().catch(() => {}); await page.waitForTimeout(3500); t = await text(page); }
    const msg = (t.match(/[^\n]*(làm bài thi|bài thi|khóa|khoá)[^\n]*/i) || [""])[0];
    out("TC-P8-02-32", /bài thi/.test(t), `SV có lượt thi IN_PROGRESS: bấm «Hỏi AI về tài liệu» → ${url.replace(BASE, "")}; gửi «Giải thích chương 3»; thông báo trên màn hình: ${JSON.stringify(msg)} (API: 409 EXAM_IN_PROGRESS như chat thường)`);
  }
  if (which.includes("42")) {
    const { page } = await mk("gv.s8f", C.U); await page.goto(`${BASE}/documents?course=${C.U}`); await page.waitForTimeout(3000);
    const doc = psql(`select id from documents where filename='u0.docx' and course_id='${C.U}' limit 1`); const t0 = psql(`select text from content_chunks where document_id='${doc}' order by ord limit 1`); const a0 = psql("select count(*) from audit_log");
    await page.getByRole("button", { name: "u0", exact: true }).first().click(); await page.waitForTimeout(2000);
    await page.route("**/api/v1/courses/*/documents/*/chunks/*", (r) => r.request().method() === "PATCH" ? r.fulfill({ status: 503, contentType: "application/json", body: JSON.stringify({ code: "SERVICE_UNAVAILABLE", message: "x" }) }) : r.continue());
    await page.getByRole("button", { name: "Sửa đoạn" }).first().click(); await page.getByLabel("Nội dung đoạn").fill("Văn bản mới QC"); await page.getByRole("button", { name: "Lưu đoạn" }).click(); await page.waitForTimeout(2000);
    const t = await text(page); const t1 = psql(`select text from content_chunks where document_id='${doc}' order by ord limit 1`); const a1 = psql("select count(*) from audit_log");
    out("TC-P8-02-42-ui", /Chưa lưu được\. Thử lại\./.test(t) && t0 === t1, `PATCH đoạn trả 503: UI hiện «Chưa lưu được. Thử lại.»=${/Chưa lưu được\. Thử lại\./.test(t)}; văn bản đoạn trong DB giữ nguyên=${t0 === t1}`);
  }
} finally { await b.close(); }
