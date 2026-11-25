import { expect, test } from "@playwright/test";
import { needReal, noOverflow, shot, sql, uiLogin } from "./qc-real-lib";

// QC sprint 11 — P-11-15 ca 4: «Kết quả của tôi» hiện «Chi tiết» mở danh sách thành phần (lớp QCT3, fixture p10_detail_fixture.py).
needReal();

test("@real ME-1 «Chi tiết» mở danh sách các cột điểm (P-11-15)", async ({ page }) => {
  const c3 = sql("select id from courses where class_code='QCT3'")[0][0];
  await uiLogin(page, "sv31");
  await page.goto(`/me?course=${c3}`);
  await expect(page.getByText(/Giữa kỳ/).first()).toBeVisible({ timeout: 30_000 });
  const btn = page.getByRole("button", { name: "Chi tiết" }).first();
  await expect(btn).toBeVisible();
  await btn.click();
  const list = page.getByRole("list", { name: "Chi tiết" });
  await expect(list).toBeVisible();
  const items = await list.getByRole("listitem").allInnerTexts();
  expect(items.map((s) => s.trim())).toEqual(["Giữa kỳ lần 1 7", "Giữa kỳ lần 2 9"]);
  const body = await page.locator("main").innerText();
  expect(body, "không chuỗi công thức thô").not.toMatch(/min\(|max\(|÷|\d\s*[×x]\s*\d/);
  await shot(page, "me1-detail");
  await page.setViewportSize({ width: 375, height: 812 });
  await noOverflow(page, true);
  await btn.click();
  await expect(list).toHaveCount(0);
});
