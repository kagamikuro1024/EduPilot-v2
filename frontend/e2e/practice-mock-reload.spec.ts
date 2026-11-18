import { expect, test } from "@playwright/test";
import { practiceApi } from "./support/practice-fixtures";
import { asDemo } from "./support/session";

// Desktop + mobile (390): thi thử tải lại giữa bài. Gateway giả có trạng thái (support/practice-fixtures.ts).
test.beforeEach(async ({ context }) => { await asDemo(context, "student", { person: "sv-2" }); });

// DEF-R10S-14 (TC-12-11, US-P9-07 AC6): thi thử 20 câu, tải lại ở câu 8 → quay ĐÚNG câu 8 (còn lựa chọn đã chọn, kể cả chưa gửi), chọn tiếp câu 9…20 rồi nộp → đủ 20/20.
// Trước sửa: MockRun không PATCH current_index → tải lại về câu 1, các lần «Câu sau» tiếp theo rơi vào câu cũ, 7 câu cuối không bao giờ được trả lời.
for (const gap of [250, 1200]) {
  test(`mock run: tải lại ở câu 8 (nhịp ${gap} ms) quay đúng câu, chọn tiếp và nộp — đủ 20/20`, async ({ page }) => {
    await practiceApi(page, {}, { mockCount: 20 });
    await page.goto("/practice");
    await page.getByRole("tab", { name: /Thi thử/ }).click();
    await page.getByRole("button", { name: "Bắt đầu thi thử" }).click();
    await page.waitForURL(/\/practice\/att-1/);
    await expect(page.locator("[data-part=practice-mock]")).toBeVisible();
    const pick = async (n: number) => {
      await page.getByRole("button", { name: new RegExp(`^Phương án B của câu ${n}`) }).click();
      await page.waitForTimeout(gap);
    };
    for (let n = 1; n <= 8; n++) {
      if (n > 1) await page.getByRole("button", { name: "Câu sau" }).click();
      await pick(n);
    }
    await page.reload();
    await expect(page.locator("[data-part=practice-mock]")).toBeVisible();
    await expect(page.getByText("Câu 8/20")).toBeVisible(); // quay đúng câu đang làm
    await expect(page.getByRole("button", { name: /^Phương án B của câu 8/ })).toHaveAttribute("aria-pressed", "true"); // lựa chọn chưa gửi lúc tải lại vẫn còn
    for (let n = 9; n <= 20; n++) {
      await page.getByRole("button", { name: "Câu sau" }).click();
      await pick(n);
    }
    await page.getByRole("button", { name: "Nộp bài" }).first().click();
    await expect(page.getByRole("dialog")).toContainText("Bạn đã trả lời 20/20 câu");
    await page.getByRole("dialog").getByRole("button", { name: "Nộp bài" }).click();
    await expect(page.getByRole("heading", { name: "Bạn đúng cả 20 câu", level: 1 })).toBeVisible();
  });
}
