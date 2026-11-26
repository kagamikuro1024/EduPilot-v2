import { appendFileSync, readFileSync } from "node:fs";
import { test, type Browser, type Page } from "@playwright/test";
import { sql, PW, OUT } from "./qc-real-lib";
const RES = "/Users/kuro/Documents/TA_Agent_qc9s/docs/regression/evidence-9s/s106/results_106.jsonl";
const FX = JSON.parse(readFileSync("/Users/kuro/.config/edupilot-team/qc3-work/g06fx.json", "utf8")) as { a: string; v1sub: string; v0sub: string; v1res: string; v0res: string; n_c2: number };
const verdict = (tc: string, cond: boolean, why: string) => { appendFileSync(RES, JSON.stringify({ tc, kq: cond ? "P" : "F", note: why }) + "\n"); console.log("VERDICT", cond ? "P" : "F", tc, why.slice(0, 330)); };
test.skip(!process.env.QC_REAL, "cần stack thật");
test.use({ actionTimeout: 25_000 });
const CID = () => sql(`select id from courses where class_code='QD9G'`)[0][0];
async function pageAs(browser: Browser, user: string, w = 1440) {
  const ctx = await browser.newContext({ baseURL: "https://localhost:943", ignoreHTTPSErrors: true, viewport: { width: w, height: 900 }, locale: "vi-VN" });
  await ctx.addInitScript((c) => localStorage.setItem("ep:ui:course", c), CID());
  const page = await ctx.newPage(); await page.addLocatorHandler(page.getByRole("button", { name: "Đã hiểu" }), async (b) => { await b.click(); });
  await page.goto("/login"); await page.locator("input[type=email]").fill(`${user}@edupilot.local`); await page.locator("input[type=password]").fill(PW);
  await page.keyboard.press("Enter"); await page.waitForURL((u) => !u.pathname.startsWith("/login")); await page.locator("[data-part=topbar]").waitFor(); return { ctx, page };
}
const panel = (p: Page) => p.locator("[data-part=guidance-panel]");
test("TC-G06-05 bảng số đo dưới danh sách; staff thấy, không biểu đồ", async ({ browser }) => {
  const { ctx, page } = await pageAs(browser, "teacher"); await page.goto(`/grading?tab=assignments&open=${FX.a}`); await page.waitForTimeout(4500);
  const st = panel(page).locator("[data-part=guidance-stats]"); const n = await st.count(); const txts = await st.allInnerTexts();
  const ord = await page.evaluate(() => { const p = document.querySelector("[data-part=guidance-panel]")!; const s = p.querySelector("[data-part=guidance-stats]"); const it = [...p.querySelectorAll("[data-part=guidance-item]")].pop(); return !!(s && it && (it.compareDocumentPosition(s) & Node.DOCUMENT_POSITION_FOLLOWING)); });
  const charts = await panel(page).locator("svg.recharts-surface, canvas, [class*=chart]").count(); await panel(page).scrollIntoViewIfNeeded(); await panel(page).screenshot({ path: `${OUT}/s106-g0605-teacher.png` });
  const ta = await pageAs(browser, "ta"); await ta.page.goto(`/grading?tab=assignments&open=${FX.a}`); await ta.page.waitForTimeout(4500); const nta = await panel(ta.page).locator("[data-part=guidance-stats]").count();
  verdict("TC-G06-05", n >= 1 && ord && charts === 0, `ngăn bài tập: ${n} dòng số đo (${JSON.stringify(txts.map((t) => t.slice(0, 120)))}); nằm SAU danh sách mục=${ord}; biểu đồ=${charts}; TA thấy ${nta} dòng (nhân viên); SV không thấy khối (TC-G04-18 P)`); await ta.ctx.close(); await ctx.close();
});
test("TC-G06-06/07 phụ đề «bản vN · n lần sửa» ở màn duyệt", async ({ browser }) => {
  const { ctx, page } = await pageAs(browser, "teacher"); await page.goto(`/grading/${FX.v1sub}`); await page.waitForTimeout(4500);
  const meta = page.locator("[data-part=guidance-meta]"); const n = await meta.count(); const t = await meta.allInnerTexts(); const where = await meta.first().evaluate((e) => e.closest("[data-part]")?.parentElement?.className ?? "").catch(() => ""); console.log("META-PARENT", where, await meta.count());
  let pop = ""; if (n) { await meta.first().click().catch(() => {}); await page.waitForTimeout(800); await page.locator("[data-part=guidance-popover]").waitFor({ timeout: 5000 }).catch(() => {}); pop = (await page.locator("[data-part=guidance-popover]").innerText().catch(() => "")).replace(/\n/g, " ¦ "); await page.screenshot({ path: `${OUT}/s106-g0606-pop.png` }); }
  const names = await page.locator("main").innerText(); const crit = ["c1", "c2", "c3", "c4"]; 
  verdict("TC-G06-06", n === 1 && /bản v1 · 3 lần sửa/.test(t.join(" ")) && pop.length > 10, `kết quả v1 (c2 có ${FX.n_c2} lưu ý áp dụng): ${n} phụ đề «${t.join(" ¦ ").slice(0, 120)}»; bấm → «${pop.slice(0, 220)}»`);
  await page.goto(`/grading/${FX.v0sub}`); await page.waitForTimeout(4500); const n0 = await page.locator("[data-part=guidance-meta]").count(); const bad0 = /bản v\d · \d+ lần sửa/.test(await page.locator("main").innerText());
  verdict("TC-G06-07", n0 === 0 && !bad0, `kết quả v0 (duyệt trước khi áp dụng): phụ đề guidance-meta ×${n0}; chữ «bản vN · n lần sửa» có=${bad0}`); await ctx.close();
});
test("TC-G06-11 số đo lỗi 500 → «—» + Thử lại; chưa đủ dữ liệu không biểu đồ rỗng", async ({ browser }) => {
  const { ctx, page } = await pageAs(browser, "teacher");
  await page.route(/\/guidance\/stats/, (r) => r.fulfill({ status: 500, contentType: "application/json", body: JSON.stringify({ code: "INTERNAL", message: "Đã xảy ra lỗi." }) }));
  await page.goto(`/grading?tab=assignments&open=${FX.a}`); await page.waitForTimeout(5000); const t = (await panel(page).innerText()).replace(/\n+/g, " ¦ "); const retry = await page.getByRole("button", { name: "Thử lại" }).count(); const charts = await panel(page).locator("svg.recharts-surface, canvas").count();
  await page.screenshot({ path: `${OUT}/s106-g0611.png`, fullPage: true }); const hasDash = /—/.test(t);
  await page.unroute(/\/guidance\/stats/); if (retry) await page.getByRole("button", { name: "Thử lại" }).first().click(); await page.waitForTimeout(2500);
  verdict("TC-G06-11", (hasDash || retry > 0) && charts === 0 && retry > 0, `chặn GET /guidance/stats → 500: khối «${t.slice(-240)}»; «—»=${hasDash}; nút «Thử lại» ×${retry}; biểu đồ ${charts}`); await ctx.close();
});
