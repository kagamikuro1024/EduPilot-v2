// QC US-P4-01 AC7/AC8 (TC-P4-01-18…24). Chạy từ thư mục frontend: node ../docs/sprints/7/qc/scripts/p4-01-ui.mjs
// Biến: GW (https://localhost:543), C1, PASSWORD; phiên chuẩn bị sẵn qua API (xem report-US-P4-01.md).
import { createRequire } from "node:module";
const { chromium } = createRequire(process.cwd() + "/")("@playwright/test");
const GW = process.env.GW ?? "https://localhost:543", C1 = process.env.C1, PASSWORD = process.env.PASSWORD ?? "Edupilot#Seed-2026";
const BTN = "Nhờ giảng viên hỗ trợ";
const b = await chromium.launch();
async function login(email, viewport = { width: 1280, height: 900 }) {
  const ctx = await b.newContext({ ignoreHTTPSErrors: true, viewport }); const p = await ctx.newPage();
  await p.goto(`${GW}/login`); await p.fill("input:not([type=password])", email); await p.fill("input[type=password]", PASSWORD);
  await p.click('button:has-text("Đăng nhập")'); await p.waitForURL((u) => !u.pathname.startsWith("/login")); return { ctx, p };
}
async function openSession(p, label) { await p.goto(`${GW}/chat?course=${C1}`); await p.waitForTimeout(1500); await p.locator("main button").filter({ hasText: label }).first().click(); await p.waitForTimeout(1500); }
const out = (id, ok, note) => console.log(`${ok ? "PASS" : "FAIL"} ${id} ${note ?? ""}`);
const which = process.argv[2] ?? "all";

if (which === "all" || which === "18") { // nút hiện / không hiện
  const { ctx, p } = await login("sv07@edupilot.local");
  await openSession(p, "QC07 một");
  const n = await p.getByRole("button", { name: BTN }).count();
  const txt = await p.locator("main").innerText();
  out("TC-P4-01-18", n === 2 && !/CRISIS|SMALLTALK/.test(txt), `nút=${n} (kỳ vọng 2: chỉ hai câu lịch thường; không có ở khủng hoảng / từ chối / chào)`);
  await ctx.close();
  const c = await login("sv.nguyco@edupilot.local"); // phiên có tin FAILED (trigger tạm làm outbox lỗi)
  await openSession(c.p, "Lịch bảo vệ đồ án thế nào?");
  const nf = await c.p.getByRole("button", { name: BTN }).count(); out("TC-P4-01-18 (FAILED)", nf === 0, `nút=${nf}`);
  await c.ctx.close();
}
if (which === "all" || which === "19" || which === "20") {
  const { ctx, p } = await login("sv.kha@edupilot.local");
  await openSession(p, "Quy định nộp muộn");
  const body = await p.locator("main").innerText();
  out("TC-P4-01-19", /Đang chờ giảng viên · (vừa gửi|đã gửi)/.test(body) && (await p.getByRole("button", { name: BTN }).count()) === 0, body.match(/Đang chờ giảng viên[^\n]*/)?.[0]);
  await ctx.close();
}
if (which === "all" || which === "20") {
  const { ctx, p } = await login("sv07@edupilot.local");
  await openSession(p, "QC07 một");
  const btn = p.getByRole("button", { name: BTN }).first(); await btn.hover(); await p.waitForTimeout(800);
  const tip = (await btn.getAttribute("title")) ?? (await p.locator('[role=tooltip]').allInnerTexts()).join("|");
  out("TC-P4-01-20", /Giảng viên sẽ đọc câu hỏi này và câu trả lời của AI/.test(tip ?? ""), `tooltip="${tip}"`);
  await ctx.close();
}
if (which === "all" || which === "21") {
  const { ctx, p } = await login("sv.gioi@edupilot.local");
  await openSession(p, "Tuần này em có lịch gì?");
  const btns = p.getByRole("button", { name: BTN }); const c = await btns.count();
  const dis = c ? await btns.last().isDisabled() : null; const tip = c ? await btns.last().getAttribute("title") : null;
  out("TC-P4-01-21", c > 0 && dis === true, `nút=${c} disabled=${dis} tooltip=${tip}`);
  await ctx.close();
}
if (which === "all" || which === "22") { // bấm đúp
  const { ctx, p } = await login("sv24@edupilot.local"); const reqs = [];
  p.on("request", (r) => { if (r.method() === "POST" && /\/escalate$/.test(r.url())) reqs.push(Date.now()); });
  await openSession(p, "QCU");
  const btn = p.getByRole("button", { name: BTN }).first();
  const t0 = await p.evaluate(() => performance.now());
  await btn.dblclick(); const t1 = Date.now();
  await p.getByText(/Đang chờ giảng viên/).first().waitFor({ timeout: 3000 }); const dt = Date.now() - t1; await p.waitForTimeout(1000);
  out("TC-P4-01-22", reqs.length === 1, `POST escalate=${reqs.length}; trạng thái hiện sau ${dt} ms (đo qua Playwright, gồm độ trễ công cụ)`);
  await ctx.close();
}
if (which === "all" || which === "23") { // offline
  const { ctx, p } = await login("sv25@edupilot.local"); const reqs = [];
  p.on("request", (r) => { if (r.method() === "POST" && /\/escalate$/.test(r.url())) reqs.push(r.url()); });
  await openSession(p, "QCU");
  await ctx.setOffline(true); await p.getByRole("button", { name: BTN }).first().click(); await p.waitForTimeout(1500);
  const mid = (await p.locator("main").innerText()).match(/Đang gửi…|Đang chờ giảng viên[^\n]*/)?.[0];
  await ctx.setOffline(false); await p.waitForTimeout(6000);
  const after = (await p.locator("main").innerText()).match(/Đang chờ giảng viên[^\n]*/)?.[0];
  out("TC-P4-01-23", !!after, `offline="${mid}" → online="${after}" requests=${reqs.length}`);
  await ctx.close();
}
if (which === "all" || which === "24") { // 5xx
  const { ctx, p } = await login("sv26@edupilot.local");
  await openSession(p, "QCU");
  await p.route("**/escalate", (r) => r.fulfill({ status: 500, contentType: "application/json", body: JSON.stringify({ code: "INTERNAL", message: "x" }) }));
  await p.getByRole("button", { name: BTN }).first().click(); await p.waitForTimeout(1500);
  const txt = await p.locator("main").innerText();
  out("TC-P4-01-24", /Chưa gửi được\. Thử lại/.test(txt) && (await p.getByRole("button", { name: BTN }).count()) >= 1, txt.match(/Chưa gửi được[^\n]*/)?.[0] ?? "(không thấy thông báo)");
  await ctx.close();
}
await b.close();
