// QC US-P4-03 giao diện /inbox (TC-P4-03-08, 16, 31, 32, 33, 34, 35, 36, 38). Chạy từ thư mục frontend: C1=<uuid> node ../docs/sprints/7/qc/scripts/p4-03-ui.mjs [id]
import { createRequire } from "node:module";
const { chromium } = createRequire(process.cwd() + "/")("@playwright/test");
const GW = process.env.GW ?? "https://localhost:543", C1 = process.env.C1, C2 = process.env.C2, PASSWORD = process.env.PASSWORD ?? "Edupilot#Seed-2026";
const which = process.argv[2] ?? "all"; const b = await chromium.launch();
const out = (id, ok, note) => console.log(`${ok === null ? "INFO" : ok ? "PASS" : "FAIL"} ${id} ${note ?? ""}`);
async function login(email, vp = { width: 1280, height: 900 }) {
  const ctx = await b.newContext({ ignoreHTTPSErrors: true, viewport: vp }); const p = await ctx.newPage();
  await p.goto(`${GW}/login`); await p.fill("input:not([type=password])", email); await p.fill("input[type=password]", PASSWORD);
  await p.click('button:has-text("Đăng nhập")'); await p.waitForURL((u) => !u.pathname.startsWith("/login")); await p.waitForTimeout(1500); return { ctx, p };
}
const rows = (p) => p.locator("main button").filter({ hasText: /phút|giờ|ngày/ });
const mine = (p) => rows(p).filter({ hasText: "TS. Lê Thu Hà" }); // phiếu do giảng viên đã nhận (ô soạn chỉ có ở phiếu đã nhận)
async function openMine(p) { await openInbox(p); await p.locator("main button", { hasText: "Đã nhận" }).first().click(); await p.waitForTimeout(1500); }
async function openInbox(p, course = C1, q = "") { await p.goto(`${GW}/inbox?course=${course}${q}`); await p.waitForTimeout(2500); }
const run = (id) => which === "all" || which === id;

if (run("16")) { // hai ngữ cảnh cùng Nhận
  const A = await login("teacher@edupilot.local"), B = await login("ta@edupilot.local");
  await openInbox(A.p); await openInbox(B.p);
  await rows(A.p).first().click(); await rows(B.p).first().click(); await A.p.waitForTimeout(600);
  await Promise.all([A.p.getByRole("button", { name: /^Nhận$/ }).first().click(), B.p.getByRole("button", { name: /^Nhận$/ }).first().click()]);
  await A.p.waitForTimeout(2000);
  const ta = await A.p.locator("main").innerText(), tb = await B.p.locator("main").innerText();
  const notice = [ta, tb].map((t) => t.match(/[^\n]*đã nhận câu này lúc[^\n]*/)?.[0]).filter(Boolean);
  out("TC-P4-03-16", notice.length === 1, `InlineNotice: ${JSON.stringify(notice)} (kỳ vọng đúng một bên thấy "… đã nhận câu này lúc HH:MM")`);
  await A.ctx.close(); await B.ctx.close();
}
if (run("31") || run("32")) {
  const { ctx, p } = await login("teacher@edupilot.local"); const leaked = []; let posts = [];
  p.on("request", (r) => { const d = r.postData() ?? ""; if (d.includes("NHÁP-T1") && !/\/answer$/.test(r.url())) leaked.push(r.url()); if (r.method() === "POST" && /\/answer$/.test(r.url())) posts.push(r.url()); });
  await openMine(p);
  await mine(p).nth(0).click(); await p.locator("textarea").first().fill("NHÁP-T1 đang gõ dở"); await p.waitForTimeout(2600);
  await mine(p).nth(1).click(); await p.waitForTimeout(800); const t2 = await p.locator("textarea").first().inputValue();
  await p.locator("textarea").first().fill("NHÁP-T2 khác"); await p.waitForTimeout(2600);
  await mine(p).nth(0).click(); await p.waitForTimeout(800); const back1 = await p.locator("textarea").first().inputValue();
  await p.reload(); await p.waitForTimeout(2500); await p.locator("main button", { hasText: "Đã nhận" }).first().click(); await p.waitForTimeout(1200); await mine(p).nth(0).click(); await p.waitForTimeout(800); const afterReload = await p.locator("textarea").first().inputValue();
  out("TC-P4-03-31", back1.includes("NHÁP-T1") && !t2.includes("NHÁP-T1") && afterReload.includes("NHÁP-T1"), `chuyển T1→T2: T2="${t2}"; về T1="${back1}"; sau tải lại="${afterReload}"; request chứa nháp trước khi gửi: ${leaked.length}`);
  // TC-32: lỗi mạng/5xx giữ nháp; gửi thành công xoá nháp
  await p.route("**/answer", (r) => r.fulfill({ status: 500, contentType: "application/json", body: JSON.stringify({ code: "INTERNAL", message: "x" }) }));
  await p.getByRole("button", { name: /Gửi trả lời/ }).first().click(); await p.waitForTimeout(1500);
  const keep = await p.locator("textarea").first().inputValue(); const txt = await p.locator("main").innerText();
  out("TC-P4-03-32 lỗi 5xx", keep.includes("NHÁP-T1") && /Gửi lại|Thử lại/.test(txt), `nháp còn="${keep}"; nút: ${/Gửi lại/.test(txt) ? "Gửi lại" : /Thử lại/.test(txt) ? "Thử lại" : "(không thấy)"}`);
  await p.unroute("**/answer"); await p.locator("textarea").first().fill("Trả lời QC đúng chuẩn."); await p.getByRole("button", { name: /Gửi trả lời|Gửi lại/ }).first().click(); await p.waitForTimeout(2500);
  const after = await p.locator("textarea").count() ? await p.locator("textarea").first().inputValue() : "(đóng)";
  out("TC-P4-03-32 gửi thành công xoá nháp", !after.includes("Trả lời QC"), `ô soạn sau gửi="${after}"; POST answer=${posts.length}`);
  await ctx.close();
}
if (run("33") || run("34")) {
  const { ctx, p } = await login("teacher@edupilot.local", { width: 375, height: 812 }); await openMine(p);
  const sw0 = await p.evaluate(() => document.documentElement.scrollWidth <= window.innerWidth);
  const detailVisibleBefore = await p.locator("textarea").count();
  await p.evaluate(() => window.scrollTo(0, 120)); const y0 = await p.evaluate(() => window.scrollY);
  const r = mine(p).nth(2); const bb = await r.boundingBox(); await r.click(); await p.waitForTimeout(1500);
  const url = p.url(); const hasDetail = (await p.locator("textarea").count()) > 0;
  const ta = hasDetail ? await p.locator("textarea").first().boundingBox() : null; const vh = 812;
  out("TC-P4-03-33", /ticket=/.test(url) && hasDetail && detailVisibleBefore === 0, `danh sách toàn màn hình (textarea trước khi chọn=${detailVisibleBefore}); URL=${url.replace(GW, "")}; chi tiết toàn màn hình=${hasDetail}; ô trả lời bottom=${ta ? Math.round(ta.y + ta.height) : "-"}/${vh}`);
  const back = p.getByRole("button", { name: /Quay lại|Danh sách|Trở lại/ }).first(); const hasBack = await back.count();
  if (hasBack) { await back.click(); await p.waitForTimeout(1200); } else await p.goBack().catch(() => {});
  await p.waitForTimeout(800); const filterKept = await p.locator("main button", { hasText: "Đã nhận" }).first().getAttribute("aria-pressed").catch(() => null);
  out("TC-P4-03-33 Quay lại giữ bộ lọc/cuộn", null, `nút Quay lại=${hasBack}; aria-pressed("Đã nhận")=${filterKept}; scrollY trước=${y0} sau=${await p.evaluate(() => window.scrollY)}`);
  const small = await p.evaluate(() => [...document.querySelectorAll("main button, main a, main textarea")].map((e) => { const r = e.getBoundingClientRect(); return r.width > 0 ? [e.innerText.slice(0, 20), Math.round(r.width), Math.round(r.height)] : null; }).filter((x) => x && x[2] < 44).slice(0, 8));
  out("TC-P4-03-34 vùng chạm ≥ 44 px", small.length === 0 && sw0, `không tràn ngang=${sw0}; phần tử cao < 44 px: ${JSON.stringify(small)}`);
  await ctx.close();
}
if (run("35")) {
  const { ctx, p } = await login("teacher@edupilot.local", { width: 1280, height: 900 }); await openMine(p); await mine(p).first().click(); await p.waitForTimeout(800);
  const w = await p.locator("section").first().evaluate((e) => Math.round(e.getBoundingClientRect().width)); // cột danh sách (grid 380px | 1fr)
  const both = (await p.locator("textarea").count()) > 0 && (await rows(p).count()) > 1;
  out("TC-P4-03-35", both && Math.abs(w - 380) <= 20, `chia đôi=${both}; bề rộng cột danh sách≈${w}px (kỳ vọng 380)`); await ctx.close();
}
if (run("36")) {
  const { ctx, p } = await login("teacher@edupilot.local"); await openMine(p); await mine(p).first().click(); await p.waitForTimeout(800);
  const sel = async () => (await p.locator("main").innerText()).match(/\n([^\n]+)\n\n761987\n\n(Tuần[^\n]*)/)?.[2] ?? "?";
  await p.locator("main").click({ position: { x: 5, y: 5 } }).catch(() => {}); const s0 = await sel();
  await p.keyboard.press("j"); await p.waitForTimeout(500); const s1 = await sel(); await p.keyboard.press("k"); await p.waitForTimeout(500); const s2 = await sel();
  out("TC-P4-03-36 J/K", s0 !== s1 && s0 === s2, `${s0} → J → ${s1} → K → ${s2}`);
  await p.keyboard.press("r"); await p.waitForTimeout(400); const focused = await p.evaluate(() => document.activeElement?.tagName);
  out("TC-P4-03-36 R", focused === "TEXTAREA", `focus=${focused}`);
  await p.keyboard.type("jjkk"); const v = await p.locator("textarea").first().inputValue(); const sAfter = await sel();
  out("TC-P4-03-36 phím không kích hoạt khi đang gõ", v === "jjkk" && sAfter === s2, `ô="${v}" ticket=${sAfter}`);
  await p.locator("textarea").first().fill("Trả lời bằng phím tắt QC"); const before = await sel(); await p.keyboard.press("Control+Enter"); await p.waitForTimeout(2500); const next = await sel();
  out("TC-P4-03-36 Ctrl+Enter gửi rồi sang ticket kế", before !== next, `${before} → ${next}`);
  const btn = p.getByRole("button", { name: /Gửi trả lời/ }).first(); await btn.hover(); await p.waitForTimeout(700);
  const tip = (await btn.getAttribute("title")) ?? (await btn.getAttribute("aria-keyshortcuts")) ?? (await p.locator("[role=tooltip]").allInnerTexts()).join("|");
  out("TC-P4-03-36 tooltip phím trên nút Gửi trả lời", !!tip, `tooltip="${tip}"`); await ctx.close();
}
if (run("38")) {
  const { ctx, p } = await login("teacher@edupilot.local");
  await p.route("**/api/v1/courses/*/tickets*", async (r) => { await new Promise((s) => setTimeout(s, 3500)); await r.continue().catch(() => {}); });
  await p.goto(`${GW}/inbox?course=${C1}`); await p.waitForTimeout(1200); const sk = await p.locator('[data-part=skeleton],[class*=skeleton],[aria-busy=true]').count();
  out("TC-P4-03-38 tải", sk > 0, `khung xương/aria-busy=${sk}`); await p.unroute("**/api/v1/courses/*/tickets*"); await p.waitForTimeout(3500);
  await p.route("**/api/v1/courses/*/tickets*", (r) => r.abort()); await p.goto(`${GW}/inbox?course=${C1}`); await p.waitForTimeout(2500);
  const et = await p.locator("main").innerText(); out("TC-P4-03-38 lỗi", /Thử lại/.test(et), `thấy "Thử lại"=${/Thử lại/.test(et)}; chữ: ${et.replace(/\n+/g, " | ").slice(0, 160)}`); await p.unroute("**/api/v1/courses/*/tickets*");
  await ctx.close();
}
await b.close();
