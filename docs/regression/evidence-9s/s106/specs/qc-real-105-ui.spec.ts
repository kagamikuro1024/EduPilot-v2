import { appendFileSync, readFileSync } from "node:fs";
import { test, type Browser } from "@playwright/test";
import { sql, PW, OUT } from "./qc-real-lib";
const RES = "/Users/kuro/Documents/TA_Agent_qc9s/docs/regression/evidence-9s/s106/results_106.jsonl";
const D = JSON.parse(readFileSync("/Users/kuro/.config/edupilot-team/qc3-work/ux60.json", "utf8")) as { a: string };
const verdict = (tc: string, cond: boolean, why: string) => { appendFileSync(RES, JSON.stringify({ tc, kq: cond ? "P" : "F", note: why }) + "\n"); console.log("VERDICT", cond ? "P" : "F", tc, why.slice(0, 380)); };
test.skip(!process.env.QC_REAL, "cần stack thật");
test.use({ actionTimeout: 20_000 });
const CID = () => sql(`select id from courses where class_code='QD105A'`)[0][0];
async function pageAs(browser: Browser, user: string, w = 1440) {
  const ctx = await browser.newContext({ baseURL: "https://localhost:943", ignoreHTTPSErrors: true, viewport: { width: w, height: w < 500 ? 812 : 900 }, locale: "vi-VN", hasTouch: w < 500, isMobile: w < 500 });
  await ctx.addInitScript((c) => localStorage.setItem("ep:ui:course", c), CID());
  const page = await ctx.newPage(); await page.addLocatorHandler(page.getByRole("button", { name: "Đã hiểu" }), async (b) => { await b.click(); });
  await page.goto("/login"); await page.locator("input[type=email]").fill(`${user}@edupilot.local`); await page.locator("input[type=password]").fill(PW);
  await page.keyboard.press("Enter"); await page.waitForURL((u) => !u.pathname.startsWith("/login")); await page.locator("[data-part=topbar]").waitFor(); return { ctx, page };
}
test("D06", async ({ browser }) => {
  const { ctx, page } = await pageAs(browser, "admin"); const out: string[] = []; const reqs: string[] = [];
  page.on("response", (r) => { if (/admin\/llm\/(decision|providers)/.test(r.url()) && r.request().method() !== "GET") reqs.push(`${r.request().method()} ${r.url().replace(/^.*\/admin\/llm\//, "")} ${r.status()}`); });
  sql(`UPDATE decision_gates SET enabled=false`); await page.goto("/settings/llm"); await page.waitForTimeout(6000);
  const sec = page.locator("section, div").filter({ has: page.getByText("Quyết định nhanh", { exact: true }) }).last(); const blk = page.locator("main");
  const gateRow = (name: string) => blk.getByText(name, { exact: true }).locator("xpath=ancestor::*[.//button[normalize-space()='Theo lớp']][1]");
  const g1 = gateRow("Kiểm duyệt Hỏi đáp công khai"); const sw = g1.getByRole("switch").first();
  // D06-09 / 11
  await g1.getByRole("button", { name: "Theo lớp" }).click(); await page.waitForTimeout(2000); const tbl = await page.getByRole("table").count();
  const rowSw = page.locator("table").last().getByRole("switch"); const nsw = await rowSw.count(); const dis = nsw ? await rowSw.first().isDisabled().catch(() => null) : null; const ariaDis = nsw ? await rowSw.first().getAttribute("aria-disabled") : null;
  if (nsw) await rowSw.first().hover({ force: true }).catch(() => {}); await page.waitForTimeout(700); const tip1 = await page.getByRole("tooltip").allInnerTexts().catch(() => []);
  out.push(`09: bảng theo lớp ${tbl}, công tắc lớp ${nsw}, disabled ${dis}/${ariaDis}, tooltip ${JSON.stringify([...new Set(tip1)])}`);
  await g1.getByRole("button", { name: "Theo lớp" }).click().catch(() => {}); await page.waitForTimeout(500);
  await sw.hover(); await page.waitForTimeout(700); const tip2 = await page.getByRole("tooltip").allInnerTexts().catch(() => []); out.push(`11: tooltip trên công tắc cổng (bản giả): ${JSON.stringify([...new Set(tip2)])}; title ${await sw.getAttribute("title")}`);
  // D06-13
  const v0 = sql(`select version from decision_gates where gate='CHAT_ROUTING' or true order by gate limit 3`).map((r: string[]) => r[0]).join(","); const a0 = sql(`select count(*) from audit_log where entity='decision_gate'`)[0][0];
  await sw.click(); await page.waitForTimeout(1200); const undo = await page.getByRole("button", { name: /Hoàn tác/ }).count(); const dlg = await page.getByRole("dialog").or(page.getByRole("alertdialog")).count(); const toast = await page.getByText(/Thành công/).count();
  const on1 = await sw.getAttribute("aria-checked"); await page.waitForTimeout(800); const a1 = sql(`select count(*) from audit_log where entity='decision_gate'`)[0][0];
  if (undo) await page.getByRole("button", { name: /Hoàn tác/ }).first().click(); await page.waitForTimeout(1500); const on2 = await sw.getAttribute("aria-checked"); const a2 = sql(`select count(*) from audit_log where entity='decision_gate'`)[0][0];
  out.push(`13: sau bật aria-checked ${on1}; «Hoàn tác» ${undo}; hộp thoại ${dlg}; toast «Thành công» ${toast}; sau hoàn tác ${on2}; audit_log decision_gate ${a0} → ${a1} → ${a2}`);
  // D06-04
  const chip = blk.getByText("gemini-decision-1", { exact: true }).first(); const m0 = sql(`select model from llm_models where provider_id='01a1deef-8e87-74a9-b3d2-aad764ed3dc8'`).map((r: string[]) => r[0]).join(",");
  await chip.click().catch(() => {}); await page.waitForTimeout(1500); const msg = (await blk.innerText()).match(/Chưa được phép[^¦\n]*/)?.[0] ?? "(không thấy)";
  const sel = await page.locator("main").getByRole("radio", { checked: true }).allInnerTexts().catch(() => []);
  out.push(`04: bấm «gemini-decision-1» → thông báo «${msg}»; mô hình trong DB ${m0}; đang chọn ${JSON.stringify(sel)}`);
  // D06-03
  const dec = blk.getByText("Quyết định", { exact: true }).first().locator("xpath=ancestor::*[.//button[normalize-space()='Test kết nối']][1]"); await dec.getByRole("button", { name: "Test kết nối" }).first().click(); await page.waitForTimeout(2500);
  out.push(`03: yêu cầu ${JSON.stringify(reqs.slice(-3))}; chữ sau test: ${((await dec.innerText()).match(/(Kết nối|Thành công|OK|lỗi)[^¦\n]*/i) ?? [""])[0]}`);
  console.log("VERDICT D06", JSON.stringify(out)); await page.screenshot({ path: `${OUT}/s105-d06b.png`, fullPage: true }); await ctx.close();
});

test("D06-08", async ({ browser }) => {
  const { ctx, page } = await pageAs(browser, "admin"); const codes: string[] = []; page.on("response", (r) => { if (/decision\/gates/.test(r.url()) && r.request().method() === "PUT") codes.push(String(r.status())); });
  await page.goto("/settings/llm"); await page.waitForTimeout(6000); const blk = page.locator("main"); const out: string[] = [];
  const g = blk.getByText("Định tuyến câu hỏi chat", { exact: true }).locator("xpath=ancestor::*[.//button[normalize-space()='Cài đặt nâng cao']][1]");
  await g.getByRole("button", { name: "Cài đặt nâng cao" }).click(); await page.waitForTimeout(1000);
  const th = page.locator("input[placeholder='0,70–0,99']"); const to = page.locator("input[placeholder='200–5000']");
  const tryv = async (inp: any, v: string, label: string) => { codes.length = 0; await inp.fill(v); await page.getByRole("button", { name: "Lưu", exact: true }).last().click(); await page.waitForTimeout(1500); const err = (await page.locator("[role=alert], [aria-invalid=true], [class*=error]").allInnerTexts().catch(() => [])).filter((x: string) => x.trim()).slice(0, 2); const toast = await page.locator("[class*=toast], [role=status]").allInnerTexts().catch(() => []); out.push(`${label}=${v}: PUT ${JSON.stringify(codes)}; lỗi tại ô ${JSON.stringify(err)}; toast ${JSON.stringify(toast.filter((x: string) => x.trim()).slice(0, 2))}`); };
  await tryv(th, "0,001", "ngưỡng"); await tryv(to, "5001", "thời hạn"); await tryv(th, "0,9", "ngưỡng hợp lệ"); await tryv(th, "0,95", "ngưỡng trả lại"); await tryv(to, "800", "thời hạn trả lại");
  console.log("VERDICT D06-08", JSON.stringify(out)); await page.screenshot({ path: `${OUT}/s105-d0608.png` }); await ctx.close();
});

test("D06-375", async ({ browser }) => {
  const { ctx, page } = await pageAs(browser, "admin", 375); await page.goto("/settings/llm"); await page.waitForTimeout(6000);
  const ov = await page.evaluate(() => document.documentElement.scrollWidth > window.innerWidth + 1); const sec = page.locator("main").getByRole("heading", { name: "Quyết định nhanh" }).first(); await sec.scrollIntoViewIfNeeded();
  const sw = await page.locator("main").getByRole("switch").evaluateAll((els) => els.map((e) => { const r = (e as HTMLElement).getBoundingClientRect(); return [Math.round(r.width), Math.round(r.height), Math.round(r.top + window.scrollY)]; }));
  const y0 = await sec.evaluate((e) => e.getBoundingClientRect().top + window.scrollY); const gateSw = sw.filter((x) => x[2] > y0 && x[2] < y0 + 400);
  const tgt = await page.locator("main").getByRole("button", { name: /Theo lớp|Cài đặt nâng cao/ }).evaluateAll((els) => els.slice(-6).map((e) => { const r = (e as HTMLElement).getBoundingClientRect(); return [Math.round(r.width), Math.round(r.height)]; }));
  await page.locator("main").getByRole("button", { name: "Theo lớp" }).first().click(); await page.waitForTimeout(1500); const ov2 = await page.evaluate(() => document.documentElement.scrollWidth > window.innerWidth + 1);
  console.log("VERDICT D06-375", JSON.stringify({ ov, ov2, gateSwitches: gateSw, buttons: tgt })); await page.screenshot({ path: `${OUT}/s105-d06-375.png` }); await ctx.close();
});

test("D07", async ({ browser }) => {
  const out: string[] = [];
  { const { ctx, page } = await pageAs(browser, "admin", 375); await page.goto("/observability"); await page.waitForTimeout(6000); const ov = await page.evaluate(() => document.documentElement.scrollWidth > window.innerWidth + 1);
    const tb = await page.locator("table").evaluateAll((els) => els.map((e) => { const p = e.parentElement as HTMLElement; return [e.scrollWidth, p.clientWidth, getComputedStyle(p).overflowX]; })); const t = (await page.locator("main").innerText()); out.push(`375: tràn ngang trang ${ov}; bảng [scrollWidth, khung, overflow-x] ${JSON.stringify(tb.slice(0, 6))}; có «Quyết định»: ${/Quyết định/.test(t)}`); await page.screenshot({ path: `${OUT}/s105-d07-375.png`, fullPage: true }); await ctx.close(); }
  { const { ctx, page } = await pageAs(browser, "admin"); let n = 0; await page.route(/observability\/decision/, (r) => { n++; return r.fulfill({ status: 500, contentType: "application/json", body: JSON.stringify({ code: "INTERNAL", message: "x" }) }); });
    await page.goto("/observability"); await page.waitForTimeout(6000); const retry = await page.getByRole("button", { name: /Thử lại/ }).count(); out.push(`12 lỗi: yêu cầu ${n}, nút «Thử lại» ${retry}`); if (retry) { await page.unroute(/observability\/decision/); await page.getByRole("button", { name: /Thử lại/ }).first().click(); await page.waitForTimeout(3000); out.push(`12 sau thử lại: «Thử lại» còn ${await page.getByRole("button", { name: /Thử lại/ }).count()}`); } await ctx.close(); }
  console.log("VERDICT D07", JSON.stringify(out));
});
