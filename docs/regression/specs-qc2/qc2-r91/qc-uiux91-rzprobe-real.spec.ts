// QC SmkS: DEF-R9.1-06 — chấm đã chọn (Vắng phép) đổi xám khi rê chuột lên đúng ô (đặc hiệu `.cell:hover .radio` > `.radio:checked`)
import { test } from "@playwright/test";
import { open, go, courseId } from "./r91lib";
test.skip(!process.env.QC_REAL, "QC_REAL");
test("probe attendance hover vs checked dot", async ({ browser }) => {
  const { page } = await open(browser, "teacher");
  const c = await courseId(page, process.env.QC_CLASS ?? "761987");
  await go(page, `/attendance?course=${c}`, undefined, 3000);
  const row = page.locator("main tbody tr[data-part=student-row]").nth(4);
  for (const st of ["LATE", "EXCUSED", "ABSENT"]) {
    const dot = row.locator(`input[data-status=${st}]`);
    await dot.check({ force: true });
    await page.mouse.move(5, 5); await page.waitForTimeout(300);
    const away = await dot.evaluate((e) => getComputedStyle(e).borderColor);
    await dot.hover(); await page.waitForTimeout(300);
    const over = await dot.evaluate((e) => getComputedStyle(e).borderColor);
    console.log(`PROBE ${st} chuột-xa=${away} chuột-trên-ô=${over}`);
  }
  await row.locator("input[data-status=PRESENT]").check({ force: true });
});
