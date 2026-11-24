import { test, type Browser } from "@playwright/test";
import { PW, sql } from "./qc-real-lib";
const OUT = "/Users/kuro/Documents/TA_Agent_qc9s/docs/regression/evidence-9s/shots/seed08";
test.skip(!process.env.QC_REAL, "cần stack thật");
test.use({ actionTimeout: 25_000 });
async function pageAs(browser: Browser, user: string, w = 1440) {
  const cid = sql(`select id from courses where class_code='761987'`)[0][0];
  const ctx = await browser.newContext({ baseURL: "https://localhost:4743", ignoreHTTPSErrors: true, viewport: { width: w, height: 900 }, locale: "vi-VN" });
  await ctx.addInitScript((c) => localStorage.setItem("ep:ui:course", c), cid);
  const page = await ctx.newPage(); await page.addLocatorHandler(page.getByRole("button", { name: "Đã hiểu" }), async (b) => { await b.click(); });
  await page.goto("/login"); await page.locator("input[type=email]").fill(`${user}@edupilot.local`); await page.locator("input[type=password]").fill(PW);
  await page.keyboard.press("Enter"); await page.waitForURL((u) => !u.pathname.startsWith("/login")); await page.locator("[data-part=topbar]").waitFor(); return { ctx, page };
}
const q = (t: string) => sql(t);
const SUB = () => sql(`select s.id from submissions s join users u on u.id=s.student_id join assignments a on a.id=s.assignment_id where u.email='sv.kha@edupilot.local' and a.submission_code='BT03'`)[0][0];
test("SEED-08 D: công bố B", async ({ browser }) => {
  const t = await pageAs(browser, "teacher"); const page = t.page; await page.goto("/grading?tab=queue"); await page.waitForTimeout(3000);
  await page.locator("select").first().selectOption({ label: "Bài tập 03 — Phân tích một vụ tấn công thực tế" }); await page.waitForTimeout(2000);
  await page.locator("select").nth(1).selectOption("REVIEWED"); await page.waitForTimeout(2500);
  console.log("ROWS", (await page.locator("main").innerText()).replace(/\n+/g, " ¦ ").slice(0, 900));
  const row = page.locator("tr", { hasText: "20229002" }).first(); console.log("ROWCOUNT", await row.count());
  const cb = row.locator("input[type=checkbox]").first(); console.log("CB", await cb.count()); if (await cb.count()) await cb.check();
  await page.screenshot({ path: `${OUT}/d0-chon-b.png`, fullPage: true });
  await page.getByRole("button", { name: /Công bố/ }).first().click(); await page.waitForTimeout(1500);
  console.log("DLG", (await page.getByRole("dialog").innerText().catch(() => "không dialog")).replace(/\n+/g, " ¦ ")); await page.screenshot({ path: `${OUT}/d1-hop-cong-bo.png`, fullPage: true });
  await page.getByRole("dialog").getByRole("button", { name: "Công bố" }).click(); await page.waitForTimeout(5000);
  console.log("PUBDB", JSON.stringify(sql(`select r.status, r.total, r.published_at is not null from grading_results r where r.submission_id='${SUB()}' order by round desc limit 1`)));
  await page.screenshot({ path: `${OUT}/d2-sau-cong-bo.png`, fullPage: true });
  const sv = await pageAs(browser, "sv.kha"); await sv.page.goto("/assignments"); await sv.page.waitForTimeout(4000);
  console.log("SVLIST", (await sv.page.locator("main").innerText()).replace(/\n+/g, " ¦ ").slice(0, 700)); await sv.page.screenshot({ path: `${OUT}/e0-sv-sau-cong-bo.png`, fullPage: true });
  await sv.page.getByRole("link", { name: /Xem/ }).first().click().catch(async () => { await sv.page.getByRole("button", { name: /Xem/ }).first().click(); }); await sv.page.waitForTimeout(4000);
  console.log("SVURL", sv.page.url()); console.log("SVDETAIL", (await sv.page.locator("main").innerText()).replace(/\n+/g, " ¦ ").slice(0, 3500)); await sv.page.screenshot({ path: `${OUT}/e1-sv-chi-tiet.png`, fullPage: true });
});
test("SEED-08 E: SV nộp muộn đang chấm (sv08)", async ({ browser }) => {
  const sv = await pageAs(browser, "sv08"); await sv.page.goto("/assignments"); await sv.page.waitForTimeout(4000);
  console.log("LIST", (await sv.page.locator("main").innerText()).replace(/\n+/g, " ¦ ").slice(0, 600));
  await sv.page.getByRole("link", { name: /Xem/ }).first().click().catch(async () => { await sv.page.getByRole("button", { name: /Xem/ }).first().click(); }); await sv.page.waitForTimeout(4000);
  console.log("DETAIL", (await sv.page.locator("main").innerText()).replace(/\n+/g, " ¦ ").slice(0, 4500)); await sv.page.screenshot({ path: `${OUT}/f0-sv08-dang-cham.png`, fullPage: true });
});
test("SEED-08 F: Kết quả của tôi (bước 6)", async ({ browser }) => {
  const sv = await pageAs(browser, "sv.kha"); await sv.page.goto("/me"); await sv.page.waitForTimeout(5000);
  console.log("ME", (await sv.page.locator("main").innerText()).replace(/\n+/g, " ¦ ").slice(0, 3800)); await sv.page.screenshot({ path: `${OUT}/g0-sv-ket-qua.png`, fullPage: true });
});
