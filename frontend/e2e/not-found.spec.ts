import { expect, test } from "@playwright/test";
import { API_URL, BASE_URL } from "./support/env";
import { settleGoto } from "./support/hydrate";
import { mockStudentToday } from "./support/screens";
import { asDemo } from "./support/session";

// DEF-R1.5-01 (TC-00-57/59, FR-X16): đã đăng nhập mở đường dẫn không có thật → màn «Không tìm thấy trang» trong khung app + «Về Hôm nay»; không bị đưa về /login.
test.beforeEach(({ page }) => settleGoto(page));

for (const role of ["student", "teacher"] as const) {
  test(`404: ${role} đã đăng nhập mở /khong-co-trang ⇒ Không tìm thấy trang, giữ URL, Về Hôm nay quay lại`, async ({ page, context }) => {
    await asDemo(context, role);
    await mockStudentToday(page);
    const refresh: string[] = [];
    page.on("request", (q) => q.url().includes("/auth/refresh") && refresh.push(q.url()));
    await page.goto("/khong-co-trang");
    await expect(page.getByRole("heading", { level: 1, name: "Không tìm thấy trang" })).toBeVisible();
    expect(refresh).toEqual([`${API_URL}/api/v1/auth/refresh`]); // trang kết xuất theo request cũng gọi đúng origin dựng (không đọc biến môi trường lúc chạy)
    await expect(page).toHaveURL(`${BASE_URL}/khong-co-trang`);
    await page.getByRole("link", { name: "Về Hôm nay" }).click();
    await expect(page).toHaveURL(`${BASE_URL}/`);
  });
}

test("404: chưa đăng nhập mở /khong-co-trang ⇒ /login?next=", async ({ page }) => {
  await page.route("**/api/v1/auth/refresh", (r) => r.fulfill({ status: 401, contentType: "application/json", body: JSON.stringify({ code: "UNAUTHENTICATED", message: "x", trace_id: "c".repeat(32) }) }));
  await page.goto("/khong-co-trang");
  await expect(page).toHaveURL(/\/login\?next=%2Fkhong-co-trang$/);
});
