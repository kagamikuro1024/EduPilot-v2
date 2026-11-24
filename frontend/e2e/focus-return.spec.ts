import { expect, test, type Page } from "@playwright/test";
import { draftLop2, gradeApi } from "./support/grade-fixtures";
import { gradebookApi } from "./support/gradebook-fixtures";
import { asDemo } from "./support/session";

// DEF-R8-A2: đóng Drawer / hộp thoại thì focus về đúng nút đã mở (không rơi về <body>).
test.beforeEach(async ({ context }, info) => {
  test.skip(info.project.name !== "desktop", "ca desktop");
  await asDemo(context, "teacher");
});

test("Drawer trích dẫn (375 px): Esc trả focus về nút «Trang N»", async ({ page }) => {
  await page.setViewportSize({ width: 375, height: 812 });
  await gradeApi(page, { draft: draftLop2({ unclear_items: [] }) });
  await page.goto("/gradebook/scheme");
  const opener = page.locator("[data-block=BONUS]").getByRole("button", { name: "Trang 2" });
  await opener.focus();
  await page.keyboard.press("Enter");
  await expect(page.getByRole("dialog")).toContainText("Mỗi lần phát biểu được cộng 0,25 điểm");
  await page.keyboard.press("Escape");
  await expect(page.getByRole("dialog")).toHaveCount(0);
  await expect(opener).toBeFocused();
});

test("Nhập XLSX: Esc trả focus về nút menu «Thêm hành động»", async ({ page }: { page: Page }) => {
  await gradebookApi(page, {});
  await page.goto("/gradebook");
  const menu = page.getByRole("button", { name: "Thêm hành động" });
  await menu.click();
  await page.getByRole("menuitem", { name: "Nhập từ XLSX" }).click();
  await expect(page.getByRole("dialog", { name: "Nhập điểm từ XLSX" })).toBeVisible();
  await page.keyboard.press("Escape");
  await expect(page.getByRole("dialog")).toHaveCount(0);
  await expect(menu).toBeFocused();
});
