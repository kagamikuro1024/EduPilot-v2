// QC SmkS: TC-UX16-9/10 (P-11-64 AC4) — cột «Trạng thái» sổ điểm: ô nào có chữ, tiêu đề phụ «Thiếu điểm: N»
import { test } from "@playwright/test";
import { open, go, courseId } from "./r91lib";
test.skip(!process.env.QC_REAL, "QC_REAL");
for (const code of (process.env.QC_CLASSES ?? "QR91A").split(",")) test(`probe sổ điểm ${code}`, async ({ browser }) => {
  const { page } = await open(browser, "teacher");
  const c = await courseId(page, code);
  await go(page, `/gradebook?course=${c}`, undefined, 4000);
  const g = await page.evaluate(() => {
    const ths = [...document.querySelectorAll("main thead th")].map((t) => (t as HTMLElement).innerText.trim());
    const si = ths.indexOf("Trạng thái");
    const cells = si < 0 ? [] : [...document.querySelectorAll("main tbody tr")].map((r) => (r.querySelectorAll("td")[si] as HTMLElement | undefined)?.innerText.replace(/\s+/g, " ").trim() ?? "");
    return { ths, si, nonEmpty: cells.filter(Boolean), nRows: cells.length, thieu: /Thiếu điểm: \d+/.exec((document.querySelector("main") as HTMLElement).innerText)?.[0] ?? null };
  });
  console.log(`PROBE ${code}: cột Trạng thái=${g.si >= 0} số hàng=${g.nRows} ô có chữ=${JSON.stringify(g.nonEmpty.slice(0, 6))} (${g.nonEmpty.length}) tiêu đề phụ Thiếu=${g.thieu} | th=${JSON.stringify(g.ths)}`);
});
