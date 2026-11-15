// QC US-P6-07: chụp + dump Dialog nhập XLSX (TC-P6-07-44…48, 59) và menu ⋯ theo vai. Chép vào frontend/e2e/ khi chạy, xoá sau.
import { expect, test } from "@playwright/test";
import { mkdirSync, writeFileSync } from "node:fs";
import { gradebookApi } from "./support/gradebook-fixtures";
import { asDemo } from "./support/session";
const OUT = "../docs/sprints/8/qc/shots"; mkdirSync(OUT, { recursive: true });
const XLSX = { name: "giua-ky.xlsx", mimeType: "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet", buffer: Buffer.from("PK\x03\x04 thử") };
const errors = [
  { row: 3, student_code: "99999999", code: "STUDENT_NOT_IN_COURSE", message: "MSSV không có trong lớp" },
  { row: 6, student_code: "20229777", code: "STUDENT_CODE_UNVERIFIED", message: "MSSV chưa có trong danh sách lớp" },
];
test.beforeEach(async ({}, i) => test.skip(i.project.name !== "desktop", "QC tự đặt bề rộng"));
for (const vp of [1440, 375]) test(`import dialog ${vp}`, async ({ page, context }) => {
  await page.setViewportSize({ width: vp, height: vp === 375 ? 812 : 900 });
  await asDemo(context, "teacher");
  await gradebookApi(page, { importOut: { rows_ok: 1, errors } } as any);
  await page.goto("/gradebook");
  await expect(page.getByRole("heading", { name: "Sổ điểm", level: 1 })).toBeVisible();
  await page.getByRole("button", { name: "Thêm hành động" }).click();
  const items = await page.getByRole("menuitem").allInnerTexts();
  await page.getByRole("menuitem", { name: "Nhập từ XLSX" }).click();
  const dlg = page.getByRole("dialog", { name: "Nhập điểm từ XLSX" });
  const out: string[] = [`menu: ${items.join(" | ")}`];
  out.push(`--- bước 1 ---\n${await dlg.innerText()}`); await page.screenshot({ path: `${OUT}/p607-step1-${vp}.png` });
  await dlg.locator("[data-part=import-input]").setInputFiles(XLSX);
  await dlg.getByRole("button", { name: "Tiếp tục" }).click();
  out.push(`--- bước 2 ---\n${await dlg.innerText()}`); await page.screenshot({ path: `${OUT}/p607-step2-${vp}.png` });
  await dlg.getByRole("button", { name: "Xem trước" }).click();
  await page.waitForTimeout(500);
  out.push(`--- bước 3 ---\n${await dlg.innerText()}`); await page.screenshot({ path: `${OUT}/p607-step3-${vp}.png` });
  out.push(`scroll=${await page.evaluate(() => [document.documentElement.scrollWidth, innerWidth])}`);
  out.push(`checked(Công bố)=${await dlg.getByLabel("Công bố cột cho sinh viên").isChecked()}`);
  writeFileSync(`${OUT}/p607-dialog-${vp}.txt`, out.join("\n"));
  await page.keyboard.press("Escape");
  await expect(page.getByRole("button", { name: "Thêm hành động" })).toBeFocused();
});
test("menu theo vai", async ({ page, context }) => {
  await page.setViewportSize({ width: 1440, height: 900 });
  const r: string[] = [];
  for (const role of ["ta", "teacher"] as const) {
    const p = await context.newPage(); await asDemo(context, role); await gradebookApi(p, {}); await p.goto("/gradebook");
    await expect(p.getByRole("heading", { name: "Sổ điểm", level: 1 })).toBeVisible();
    const has = await p.getByRole("button", { name: "Thêm hành động" }).count();
    if (has) { await p.getByRole("button", { name: "Thêm hành động" }).click(); r.push(`${role}: ${(await p.getByRole("menuitem").allInnerTexts()).join(" | ")}`); } else r.push(`${role}: không có menu`);
  }
  writeFileSync(`${OUT}/p607-menu-roles.txt`, r.join("\n"));
});
