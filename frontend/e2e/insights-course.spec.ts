import { expect, test, type Page } from "@playwright/test";
import { BASE_URL } from "./support/env";
import { asJwt } from "./support/session";
import { REPORT } from "./support/insights-fixtures";

const cors = { "Access-Control-Allow-Origin": BASE_URL, "Access-Control-Allow-Credentials": "true", Vary: "Origin" };
const json = (body: unknown) => ({ status: 200, contentType: "application/json", headers: cors, body: JSON.stringify(body) });
const KC = { id: "11111111-1111-7111-8111-111111111106", class_code: "KC106", subject_code: "KC", name: "Kiểm chứng 10.6", semester: "2026-2027-HK1", status: "ACTIVE" };
const MOCKED = { id: "22222222-2222-7222-8222-222222222987", class_code: "761987", subject_code: "INT1006", name: "An ninh mạng", semester: "2026-2027-HK1", status: "ACTIVE" };

async function open(page: Page, courses: (typeof KC)[]) {
  await asJwt(page, "TEACHER");
  await page.route("**/api/v1/me/courses**", (r) => r.fulfill(json({ items: courses.map((course) => ({ course, role_in_course: "TEACHER", enrollment_status: "ACTIVE" })), next_cursor: null })));
  await page.route("**/api/v1/me/today**", (r) => r.fulfill(json({ count: 0, actions: [] })));
  await page.route("**/api/v1/notifications**", (r) => r.fulfill(json({ items: [], next_cursor: null, unread_count: 0 })));
  // báo cáo theo lớp: chỉ lớp 761987 có; lớp khác 404 (không mượn báo cáo của lớp nào)
  await page.route("**/api/v1/courses/*/insights/knowledge-gap/latest", (r) =>
    r.request().url().includes(MOCKED.id) ? r.fulfill(json(REPORT)) : r.fulfill({ ...json({ code: "NOT_FOUND", message: "Không có", trace_id: "t" }), status: 404 }),
  );
  await page.goto("/insights");
}

test("insights: lớp thật ngoài mô phỏng KHÔNG mượn báo cáo 761987 — rỗng, header là lớp thật (10.7 lô 1)", async ({ page }) => {
  await open(page, [KC, MOCKED]);
  await expect(page.getByText("Chưa có báo cáo")).toBeVisible();
  await expect(page.getByRole("button", { name: /^Chọn lớp, đang xem KC106/ })).toBeVisible(); // lớp thật ở thanh trên (header không còn lặp lớp)
  await expect(page.getByText("Chủ đề nên dạy lại")).toHaveCount(0);
  await expect(page.getByText("An ninh mạng – 761987")).toHaveCount(0);
});

test("insights: lớp có báo cáo thật thì hiện báo cáo của chính lớp đó", async ({ page }) => {
  await open(page, [MOCKED]);
  await expect(page.getByText("Chủ đề nên dạy lại")).toBeVisible();
});
