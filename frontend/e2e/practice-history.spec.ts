import { expect, test } from "@playwright/test";
import { practiceApi } from "./support/practice-fixtures";
import { asDemo } from "./support/session";

// DEF-R10S-11 (TC-09-19, US-P9-09 AC7): sinh viên chưa có lượt nào → «Chưa có lượt luyện» + nút «Luyện ngay» về /practice. Desktop + mobile.
test("history rỗng: có nút Luyện ngay về /practice", async ({ page, context }) => {
  await asDemo(context, "student", { person: "sv-2" });
  await practiceApi(page);
  await page.goto("/practice/history");
  await expect(page.getByText("Chưa có lượt luyện nào")).toBeVisible();
  await page.getByRole("link", { name: "Luyện ngay" }).click();
  await expect(page).toHaveURL(/\/practice$/);
});
