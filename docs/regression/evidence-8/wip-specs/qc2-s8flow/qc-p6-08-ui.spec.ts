// QC US-P6-08: chụp + dump sổ đã chốt, hộp Mở khoá, menu theo vai (TC-P6-08-10, 15, 36–38). Chép vào frontend/e2e/ khi chạy, xoá sau.
import { expect, test } from "@playwright/test";
import { mkdirSync, writeFileSync } from "node:fs";
import { gradebookApi } from "./support/gradebook-fixtures";
import { asDemo } from "./support/session";
const OUT = "../docs/sprints/8/qc/shots"; mkdirSync(OUT, { recursive: true });
test.beforeEach(async ({}, i) => test.skip(i.project.name !== "desktop", "QC tự đặt bề rộng"));
for (const [name, role, vp, opts] of [["locked-teacher-1440", "teacher", 1440, { students: 30, locked: true }], ["locked-teacher-375", "teacher", 375, { students: 30, locked: true }], ["locked-ta-1440", "ta", 1440, { students: 30, locked: true }], ["drift-teacher-1440", "teacher", 1440, { students: 30, locked: true, drift: true }]] as const) {
  test(`dump ${name}`, async ({ page, context }) => {
    await page.setViewportSize({ width: vp, height: vp === 375 ? 812 : 900 });
    await asDemo(context, role as any); await gradebookApi(page, opts as any);
    await page.goto("/gradebook");
    await expect(page.getByRole("heading", { name: "Sổ điểm", level: 1 })).toBeVisible();
    await page.waitForTimeout(800);
    const head = (await page.locator("main").innerText()).slice(0, 700);
    const inputs = await page.locator("main input[data-cell]").count();
    const menuBtn = await page.getByRole("button", { name: "Thêm hành động" }).count();
    let items = "(không menu)";
    if (menuBtn) { await page.getByRole("button", { name: "Thêm hành động" }).click(); items = (await page.getByRole("menuitem").allInnerTexts()).join(" | "); await page.keyboard.press("Escape"); }
    const note = await page.getByText("Điểm chính thức nằm ở hệ thống quản lý đào tạo của trường.").count();
    const finBtn = await page.getByRole("button", { name: "Chốt điểm" }).count();
    const drift = await page.getByText("Khác bản chốt").count();
    const sw = await page.evaluate(() => [document.documentElement.scrollWidth, innerWidth]);
    writeFileSync(`${OUT}/p608-${name}.txt`, `inputs=${inputs} menu=[${items}] officialNote=${note} chotDiemBtn=${finBtn} khacBanChot=${drift} scroll=${sw}\n---\n${head}\n`);
    await page.screenshot({ path: `${OUT}/p608-${name}.png` });
  });
}
test("hộp Mở khoá", async ({ page, context }) => {
  await page.setViewportSize({ width: 1440, height: 900 });
  await asDemo(context, "teacher"); await gradebookApi(page, { students: 30, locked: true } as any);
  await page.goto("/gradebook"); await expect(page.getByRole("heading", { name: "Sổ điểm", level: 1 })).toBeVisible();
  await page.getByRole("button", { name: "Thêm hành động" }).click();
  await page.getByRole("menuitem", { name: "Mở khoá" }).click();
  const dlg = page.getByRole("dialog"); await expect(dlg).toBeVisible();
  const txt = await dlg.innerText(); const out = [txt];
  const box = dlg.getByRole("textbox"); const ok = dlg.getByRole("button", { name: "Mở khoá" });
  for (const n of [0, 9, 10, 500, 501]) { await box.fill("x".repeat(n)); await page.waitForTimeout(100); out.push(`len ${n}: nút ${await ok.last().isDisabled() ? "KHOÁ" : "mở"}`); }
  await box.fill("          "); out.push(`10 dấu cách: nút ${await ok.last().isDisabled() ? "KHOÁ" : "mở"}`);
  await box.fill("Phúc khảo sv04: sửa điểm giữa kỳ"); await page.screenshot({ path: `${OUT}/p608-unlock-dialog.png` });
  writeFileSync(`${OUT}/p608-unlock-dialog.txt`, out.join("\n"));
});
