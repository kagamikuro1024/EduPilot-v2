import { expect, test } from "@playwright/test";
import { acquireFakeApi, releaseFakeApi } from "./support/fake-api-lock";
import { reset, script } from "./support/llm-fixtures";
import { asJwt } from "./support/session";

// DEF-R10.7-02 (TL 14.6): /gradebook của giảng viên giữ nguyên nút đầu trang («Chốt điểm» khoá, «Xem công thức», menu) khi danh sách lớp còn đang tải hoặc lỗi.
test.describe.configure({ mode: "serial" });
test.beforeAll(acquireFakeApi);
test.afterAll(releaseFakeApi);
test.beforeEach(async ({}, info) => {
  test.skip(info.project.name !== "desktop", "tự đặt bề rộng; máy chủ giả dùng chung");
});

for (const [name, step] of [["đang tải", { body: { items: [], next_cursor: null }, delay: 4000 }], ["lỗi", { status: 503, body: { code: "NOT_READY", message: "m", trace_id: "c".repeat(32) } }]] as const) {
  test(`gradebook ${name}: «Chốt điểm» khoá vẫn hiện, có «Xem công thức»`, async ({ page }) => {
    await reset(page);
    await script(page, { "GET /api/v1/me/courses": [step, step, step] } as never);
    await asJwt(page, "TEACHER", { email: "gv@ptit.edu.vn" });
    await page.goto("/gradebook");
    await expect(page.getByRole("heading", { name: "Sổ điểm", level: 1 })).toBeVisible();
    const btn = page.getByRole("button", { name: "Chốt điểm" });
    await expect(btn).toBeVisible();
    await expect(btn).toBeDisabled();
    await expect(page.getByRole("link", { name: "Xem công thức" })).toBeVisible();
  });
}
