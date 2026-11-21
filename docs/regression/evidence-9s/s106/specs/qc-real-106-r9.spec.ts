import { appendFileSync, readFileSync } from "node:fs";
import { test } from "@playwright/test";
import { sql, PW, OUT } from "./qc-real-lib";
test.skip(!process.env.QC_REAL, "cần stack thật");
const W = JSON.parse(readFileSync("/tmp/r9_w.json", "utf8")) as { course: string; aid: string };
const RES = "/Users/kuro/Documents/TA_Agent_qc9s/docs/regression/evidence-9s/results_9rerun.jsonl";
const verdict = (tc: string, cond: boolean, why: string) => { appendFileSync(RES, JSON.stringify({ tc, kq: cond ? "P" : "F", note: why }) + "\n"); console.log("VERDICT", cond ? "P" : "F", tc, why.slice(0, 300)); };
async function login(browser: any, user: string, course: string, w = 1440) {
  const ctx = await browser.newContext({ baseURL: "https://localhost:943", ignoreHTTPSErrors: true, viewport: { width: w, height: 900 }, locale: "vi-VN" });
  await ctx.addInitScript((c: string) => localStorage.setItem("ep:ui:course", c), course);
  const page = await ctx.newPage(); await page.addLocatorHandler(page.getByRole("button", { name: "Đã hiểu" }), async (b: any) => { await b.click(); });
  await page.goto("/login"); await page.locator("input[type=email]").fill(`${user}@edupilot.local`); await page.locator("input[type=password]").fill(PW); await page.keyboard.press("Enter");
  await page.waitForURL((u: URL) => !u.pathname.startsWith("/login")); await page.locator("[data-part=topbar]").waitFor(); return { ctx, page };
}
test("TC-P7-08-13 lớp có bài tập, chưa có bài nộp", async ({ browser }) => {
  const { ctx, page } = await login(browser, "teacher", W.course); await page.goto(`/grading?course=${W.course}&tab=queue`); await page.waitForTimeout(4500);
  const t = (await page.locator("main").innerText()).replace(/\n+/g, " | "); await page.screenshot({ path: `${OUT}/s9r-0813.png` });
  const btns = await page.locator("main").getByRole("button").allInnerTexts();
  verdict("TC-P7-08-13", /Chưa có bài nộp/.test(t) && (/Đồng bộ từ Teams/.test(t) || btns.some((b: string) => /Đồng bộ từ Teams/.test(b))) && btns.some((b: string) => /Tải bài lên/.test(b)), `lớp QD9W có 1 bài tập, 0 bài nộp: «${t.slice(0, 260)}»; nút: ${JSON.stringify(btns)}`); await ctx.close();
});
test("TC-P7-07-13 /observability ADMIN dải Chấm bài", async ({ browser }) => {
  const { ctx, page } = await login(browser, "admin", W.course); await page.goto("/observability"); await page.waitForTimeout(5000);
  const t = (await page.locator("main").innerText()).replace(/\n+/g, " | "); await page.screenshot({ path: `${OUT}/s9r-0713.png`, fullPage: true });
  const i = t.indexOf("Chấm bài"); verdict("TC-P7-07-13", i >= 0 && /(hàng đợi|Hàng đợi|đang chờ)/i.test(t.slice(i, i + 400)), `/observability: «Chấm bài» ${i >= 0 ? "có" : "không có"}; đoạn: «${i >= 0 ? t.slice(i, i + 300) : t.slice(0, 200)}»`); await ctx.close();
});
