// QC US-P6-06: chụp + dump `/gradebook` (tc-US-P6-06.md TC-01, 04, 37, 60). Sao chép vào frontend/e2e/ khi chạy, xoá sau.
import { expect, test } from "@playwright/test";
import { mkdirSync, writeFileSync } from "node:fs";
import { gradebookApi } from "./support/gradebook-fixtures";
import { asDemo } from "./support/session";
const OUT = "../docs/sprints/8/qc/shots"; mkdirSync(OUT, { recursive: true });
test.beforeEach(async ({}, i) => test.skip(i.project.name !== "desktop", "QC tự đặt bề rộng"));
for (const [name, role, vp, opts] of [["teacher-1440", "teacher", 1440, { students: 30 }], ["teacher-375", "teacher", 375, { students: 30 }], ["ta-1440", "ta", 1440, { students: 30 }], ["noscheme-1440", "teacher", 1440, { scheme: false }]] as const) {
  test(`dump ${name}`, async ({ page, context }) => {
    await page.setViewportSize({ width: vp, height: vp === 375 ? 812 : 900 });
    await asDemo(context, role as any);
    await gradebookApi(page, opts as any);
    await page.goto("/gradebook");
    await expect(page.getByRole("heading", { name: "Sổ điểm", level: 1 })).toBeVisible();
    await page.waitForTimeout(900);
    const t = await page.locator("main").innerText();
    const panels = await page.locator("main [data-ep-panel]").count();
    const inputs = await page.locator("main input:visible").count();
    const rows = await page.locator("[data-part=grade-row]").count();
    const note = await page.getByText("Điểm chính thức nằm ở hệ thống quản lý đào tạo của trường.").count();
    const btns = await page.locator("main button:visible").evaluateAll((b: any[]) => b.map((x) => (x.textContent || x.getAttribute("aria-label") || "").trim() + (x.disabled || x.getAttribute("aria-disabled") === "true" ? " [khoá]" : "")));
    const sw = await page.evaluate(() => [document.documentElement.scrollWidth, innerWidth]);
    writeFileSync(`${OUT}/p606-${name}.txt`, `panels=${panels} rows=${rows} inputs=${inputs} officialNote=${note} scroll=${sw}\nbuttons: ${btns.join(" | ")}\n---\n${t.slice(0, 2500)}\n`);
    await page.screenshot({ path: `${OUT}/p606-${name}.png`, fullPage: false });
  });
}
