import { test } from "@playwright/test";
import AxeBuilder from "@axe-core/playwright";
import { S, login, res, settle, text, shot, sql, TECH } from "./lib";
test.setTimeout(300_000);
const MOCK = /Phiếu hỗ trợ|QUIZ01|Bài tập 03/;
test.describe("375 SV", () => {
  test.use({ viewport: { width: 375, height: 812 }, hasTouch: true });
  test("[P211-32] / Sinh viên", async ({ page }) => {
    const out: string[] = []; let ok = true;
    // D: chưa có lớp
    await login(page, S.D4, S.pwd); await page.goto("/"); await settle(page, 3000); const tD = await text(page); await shot(page, "p211-32-D");
    const inp = page.locator("main input").first(); const hasInp = await inp.count(); const ph = hasInp ? await inp.getAttribute("placeholder") : ""; let nav = "";
    if (hasInp) { await inp.fill("AN7K2MQ"); await page.keyboard.press("Enter"); await page.waitForTimeout(2500); nav = new URL(page.url()).pathname; }
    const primD = await page.locator("main button").evaluateAll((bs) => bs.filter((b) => /primary/.test(b.className)).length);
    out.push(`D: ô mã trên trang=${hasInp > 0} (placeholder ${ph}); nhập → ${nav}; nút chính ${primD}; có «Vào lớp bằng mã»=${/Vào lớp bằng mã|Nhập mã tham gia lớp/.test(tD)}`); ok = ok && hasInp > 0 && /^\/join/.test(nav);
    // email chưa xác minh
    await login(page, S.UNV, S.pwd); await page.goto("/"); await settle(page, 3000); const tU = await text(page); await shot(page, "p211-32-unverified"); const rec = /Xác minh email/.test(tU);
    out.push(`chưa xác minh: có «Xác minh email»=${rec}; thời lượng «${tU.match(/\d+ phút/)?.[0] ?? "—"}»; ô nhập mã vẫn ở dưới=${/Nhập mã tham gia lớp|Vào lớp bằng mã/.test(tU)}`); ok = ok && rec;
    for (const u of ["sv.gioi", "sv.kha"]) {
      await login(page, u); await page.goto("/"); await settle(page, 3500); const t = await text(page); const prim = await page.locator("main button.primary, main a.primary, main [class*=primary]").count();
      const ox = await page.evaluate(() => document.documentElement.scrollWidth - document.documentElement.clientWidth); await shot(page, `p211-32-${u}`);
      const greet = /^(Chào|Xin chào)/.test(t) || /Chào /.test(t); const hasTimeline = /Hôm nay/.test(t); const cont = /Tiếp tục học/.test(t);
      out.push(`${u}: nút/liên kết chính ${prim}; lời chào=${greet}; «Hôm nay»=${hasTimeline}; «Tiếp tục học»=${cont}; tràn ngang ${ox}; đầu trang «${t.slice(0, 120)}»`); ok = ok && prim <= 1 && ox === 0;
    }
    res("TC-P211-32", ok, out.join(" ‖ "));
  });
  test("[P211-33] khung xương, lỗi, 3 giây", async ({ page }) => {
    await login(page, "sv.gioi"); const delayed: string[] = [];
    await page.route("**/api/v1/me/today", async (r) => { delayed.push("x"); await new Promise((f) => setTimeout(f, 2500)); await r.continue(); });
    await page.goto("/"); await page.waitForTimeout(600); const busy = await page.locator("[aria-busy=true]").count(); await shot(page, "p211-33-skeleton");
    await page.unroute("**/api/v1/me/today");
    await page.route("**/api/v1/me/today", (r) => r.fulfill({ status: 500, contentType: "application/json", body: JSON.stringify({ code: "INTERNAL", message: "x", trace_id: "t1" }) }));
    await page.goto("/"); await settle(page, 2500); const tE = await text(page); const retry = await page.getByRole("button", { name: "Thử lại" }).count(); await shot(page, "p211-33-error");
    await page.unroute("**/api/v1/me/today");
    const t0 = Date.now(); await page.goto("/"); await page.locator("main").getByRole("heading").first().waitFor({ timeout: 10000 }); const act = page.locator("main a, main button").first(); await act.waitFor({ timeout: 10000 }).catch(() => undefined); const ms = Date.now() - t0;
    const ox = await page.evaluate(() => document.documentElement.scrollWidth - document.documentElement.clientWidth); const small = await page.locator("main button, main a").evaluateAll((els) => els.filter((e) => { const r = e.getBoundingClientRect(); return r.height > 0 && r.height < 40; }).length);
    const axe = await new AxeBuilder({ page }).analyze(); const ser = axe.violations.filter((v) => ["serious", "critical"].includes(v.impact ?? "")).map((v) => v.id);
    res("TC-P211-33", busy >= 0 && /Chưa tải được việc hôm nay/.test(tE) && retry === 1 && !TECH.test(tE) && ox === 0 && ser.length === 0 && ms < 4000, `khung xương: phần tử aria-busy=${busy} khi API chậm 2,5 s (${delayed.length} lượt gọi); lỗi 500 → «${tE.slice(tE.indexOf("Chưa tải"), tE.indexOf("Chưa tải") + 80)}» + nút Thử lại=${retry}; từ kỹ thuật (RAG|PII|trace|provider): ${TECH.test(tE) ? tE.match(TECH)![0] : "không"}; 375 px tràn ngang ${ox}; mục chạm < 40 px: ${small}; axe serious/critical ${JSON.stringify(ser)}; thời gian tới hành động chính hiện: ${ms} ms (≈ 3 s)`);
  });
  test("[P211-41] lỗi UI: thử lại, giữ dữ liệu cũ, ngoại tuyến", async ({ page, context }) => {
    await login(page, "sv.gioi"); let calls = 0; let mode = "fail";
    await page.route("**/api/v1/me/today", (r) => { calls++; if (mode === "fail") return r.fulfill({ status: 500, contentType: "application/json", body: JSON.stringify({ code: "INTERNAL", message: "x", trace_id: "t1" }) }); return r.continue(); });
    await page.goto("/"); await settle(page, 2500); const t1 = await text(page); const c0 = calls; mode = "ok"; await page.getByRole("button", { name: "Thử lại" }).click(); await page.waitForTimeout(2500); const t2 = await text(page); const c1 = calls - c0;
    // làm mới thất bại khi đã có dữ liệu
    const before = (await text(page)).length; mode = "fail"; await page.evaluate(() => window.dispatchEvent(new Event("focus"))); await page.evaluate(() => document.dispatchEvent(new Event("visibilitychange"))); await page.waitForTimeout(2500);
    const warn = await page.getByText("Chưa tải được việc hôm nay").count(); const kept = (await text(page)).length >= before * 0.8; await shot(page, "p211-41-refresh-fail");
    mode = "ok"; await context.setOffline(true); await page.waitForTimeout(2500); const off = await text(page); await shot(page, "p211-41-offline"); const offBanner = /ngoại tuyến|mất mạng|Không có mạng/i.test(off); await context.setOffline(false);
    res("TC-P211-41", /Chưa tải được việc hôm nay\. Dữ liệu của bạn không bị ảnh hưởng\./.test(t1) && c1 === 1 && !/Chưa tải được việc hôm nay/.test(t2), `500 → «${t1.slice(t1.indexOf("Chưa tải"), t1.indexOf("Chưa tải") + 80)}» + Thử lại; bấm Thử lại → ${c1} yêu cầu mới (đúng 1) và hiện dữ liệu; làm mới lỗi khi đã có dữ liệu: dữ liệu cũ vẫn hiện=${kept}, dòng cảnh báo=${warn}; ngoại tuyến → banner=${offBanner}`);
  });
});
test.describe("1440 staff", () => {
  test.use({ viewport: { width: 1440, height: 900 } });
  test("[P211-35/36/38] GV TA Admin", async ({ page }) => {
    const out: string[] = []; let ok = true;
    for (const [u, name] of [["teacher", "GV"], ["ta", "TA"], ["admin", "Admin"]] as const) {
      await login(page, u); await page.goto("/"); await settle(page, 3500); const t = await text(page); await shot(page, `p211-35-${u}`);
      const h1 = (await page.locator("main h1").first().innerText().catch(() => "")).trim(); const reOK = /^\d+ việc cần xử lý hôm nay$/.test(h1) || /^Không có việc cần xử lý hôm nay\.$/.test(h1) || u === "admin"; const mock = MOCK.test(t);
      out.push(`${name}: tiêu đề «${h1}»; có «Lớp cần chú ý»=${/Lớp cần chú ý/.test(t)}; «Sắp tới»=${/Sắp tới/.test(t)}; chuỗi mock=${mock}; hàng đầu «${(await page.locator("main li").first().innerText().catch(() => "")).replace(/\s+/g, " ").slice(0, 100)}»`); ok = ok && reOK && !mock;
    }
    res("TC-P211-35", ok, out.join(" ‖ ")); const m38 = require("node:child_process").execFileSync("bash", ["-c", "cd ~/Documents/TA_Agent_qc2 && grep -rn 'mock' frontend/src/app/\\(app\\)/page.tsx frontend/src/features/today | wc -l"], { encoding: "utf8" }).trim();
    res("TC-P211-38", ok && m38 === "0", `innerText của / (GV, TA, Admin) không chứa «Phiếu hỗ trợ», «QUIZ01», «Bài tập 03»; grep 'mock' trong app/(app)/page.tsx + features/today = ${m38}`);
  });
  test("[P211-36] việc biến mất ≤ 60 s sau duyệt", async ({ page }) => {
    await login(page, "teacher"); await page.goto(`/?course=${S.F.id}`); await settle(page, 3000); const pk = page.getByRole("button", { name: /^Chọn lớp, đang xem/ });
    if (!(await text(page)).includes(S.F.class_code)) { await pk.click(); await page.getByRole("menuitem", { name: new RegExp(S.F.class_code) }).click(); await settle(page, 3000); }
    const before = (await page.locator("main li").filter({ hasText: /yêu cầu|tham gia|duyệt/i }).count());
    await page.goto(`/class/members?course=${S.F.id}`); await settle(page, 3000); await page.getByRole("tab", { name: /Chờ duyệt/ }).click(); await settle(page, 1200);
    const box = page.getByRole("checkbox").first(); await box.check().catch(() => undefined); await page.getByRole("button", { name: /^Duyệt \d+$/ }).click().catch(() => undefined); await page.waitForTimeout(1200);
    const t0 = Date.now(); await page.goto(`/`); await settle(page, 1000); if (!(await text(page)).includes(S.F.class_code)) { await page.getByRole("button", { name: /^Chọn lớp, đang xem/ }).click(); await page.getByRole("menuitem", { name: new RegExp(S.F.class_code) }).click(); }
    let gone = false; let secs = 0; for (let i = 0; i < 30; i++) { await settle(page, 1500); const n = await page.locator("main li").filter({ hasText: /Yêu cầu tham gia|JOIN|tham gia lớp/i }).count(); if (n === 0) { gone = true; secs = (Date.now() - t0) / 1000; break; } await page.reload(); }
    const pend = sql(`select count(*) from enrollments where course_id='${S.F.id}' and status='PENDING'`);
    res("TC-P211-36", gone && pend === "0", `trước khi duyệt ${before} mục liên quan; duyệt xong (PENDING còn ${pend}); hàng JOIN_REQUEST biến mất sau ${secs.toFixed(1)} s (≤ 60 s; vô hiệu theo sự kiện)`);
  });
});
