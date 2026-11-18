// QC SmkS: probe UX14-9 — nút «Thử lại» có biến mất không khi bộ lọc lỗi tải
import { test } from "@playwright/test";
import { open, go, courseId } from "./r91lib";
test.skip(!process.env.QC_REAL, "QC_REAL");
test("probe grading error retry", async ({ browser }) => {
  const { page } = await open(browser, "teacher");
  const c = await courseId(page, process.env.QC_CLASS ?? "761987");
  let n = 0, fail = true;
  page.on("request", (r) => { if (/\/submissions\?/.test(r.url())) n++; });
  await go(page, `/grading?course=${c}`, undefined, 2500);
  await page.getByLabel("Bài tập").selectOption({ label: "Bài tập 03 — Phân tích một vụ tấn công thực tế" });
  await page.waitForTimeout(1500);
  await page.route(/\/submissions\?/, (r) => (fail ? r.abort() : r.continue()));
  await page.getByLabel("Trạng thái").selectOption({ label: "Đã duyệt" });
  await page.waitForTimeout(4000);
  await page.getByRole("button", { name: "Thử lại" }).waitFor({ timeout: 15000 });
  console.log("PROBE visible; trạng thái=", await page.getByLabel("Trạng thái").inputValue());
  fail = false;
  for (let i = 0; i < 10; i++) { console.log(`PROBE sau-fail=false t=${i * 0.3}s count=${await page.getByRole("button", { name: "Thử lại" }).count()} req=${n}`); await page.waitForTimeout(300); }
  await page.getByRole("button", { name: "Thử lại" }).click({ timeout: 8000 }).then(() => console.log("PROBE click OK"), (e) => console.log("PROBE click FAIL", String(e).slice(0, 200)));
  await page.waitForTimeout(2500);
  console.log("PROBE cuối count=", await page.getByRole("button", { name: "Thử lại" }).count(), "main=", (await page.locator("main").innerText()).replace(/\s+/g, " ").slice(0, 200));
});
