import { expect, test, type Page } from "@playwright/test";
import { BASE_URL } from "./support/env";
import { asJwt } from "./support/session";

const cors = { "Access-Control-Allow-Origin": BASE_URL, "Access-Control-Allow-Credentials": "true", Vary: "Origin" };
const json = (body: unknown) => ({ status: 200, contentType: "application/json", headers: cors, body: JSON.stringify(body) });
const KC = { id: "11111111-1111-7111-8111-111111111106", class_code: "KC106", subject_code: "KC", name: "Kiểm chứng 10.6", semester: "2026-2027-HK1", status: "ACTIVE" };
const MOCKED = { id: "22222222-2222-7222-8222-222222222987", class_code: "761987", subject_code: "INT1006", name: "An ninh mạng", semester: "2026-2027-HK1", status: "ACTIVE" };

async function open(page: Page, courses: (typeof KC)[], path: string) {
  await asJwt(page, "TEACHER");
  await page.route("**/api/v1/me/courses**", (r) => r.fulfill(json({ items: courses.map((course) => ({ course, role_in_course: "TEACHER", enrollment_status: "ACTIVE" })), next_cursor: null })));
  await page.route("**/api/v1/me/today**", (r) => r.fulfill(json({ count: 0, actions: [] })));
  await page.route("**/api/v1/notifications**", (r) => r.fulfill(json({ items: [], next_cursor: null, unread_count: 0 })));
  await page.goto(path);
}

// 10.7 (QC «lỗi Trung»): màn mô phỏng không mượn số liệu của lớp mô phỏng khác khi lớp thật không có trong mô phỏng
test("analytics: lớp thật ngoài mô phỏng hiện rỗng, không số liệu 761987; header không lặp lớp", async ({ page }) => {
  await open(page, [KC, MOCKED], "/analytics");
  await expect(page.getByText("Chưa có số liệu")).toBeVisible();
  await expect(page.getByText(/câu hỏi trong \d+ ngày/)).toHaveCount(0);
  await expect(page.locator("[data-ep-header-meta]")).not.toContainText("KC106");
});

test("analytics: lớp 761987 vẫn có số liệu", async ({ page }) => {
  await open(page, [MOCKED], "/analytics");
  await expect(page.getByText(/câu hỏi trong 7 ngày/).first()).toBeVisible();
});

test("observability: màn thật theo lớp đang chọn, không còn số mô phỏng", async ({ page }) => {
  // P10-01: màn là THẬT (số liệu của chính lớp qua API), nên tiêu đề phụ nêu đúng lớp đang chọn; số mô phỏng cũ «1.962» không được xuất hiện.
  await open(page, [KC, MOCKED], "/observability");
  await expect(page.getByRole("heading", { name: "Quan sát AI", level: 1 })).toBeVisible();
  await expect(page.getByText("1.962")).toHaveCount(0);
});
