import { expect, test } from "@playwright/test";
import { acquireFakeApi, releaseFakeApi } from "./support/fake-api-lock";
import { reset, script } from "./support/llm-fixtures";
import { asJwt } from "./support/session";

// DEF-R10.7-03 (X3): /practice của sinh viên giữ khung tab «Theo chủ đề / Thi thử» khi danh sách lớp còn đang tải.
test.describe.configure({ mode: "serial" });
test.beforeAll(acquireFakeApi);
test.afterAll(releaseFakeApi);
test.beforeEach(async ({}, info) => {
  test.skip(info.project.name !== "desktop", "tự đặt bề rộng; máy chủ giả dùng chung");
});

test("practice đang tải lớp: tab «Theo chủ đề», «Thi thử» và «Lịch sử luyện tập» vẫn hiện", async ({ page }) => {
  await page.setViewportSize({ width: 375, height: 812 });
  await reset(page);
  const slow = { body: { items: [], next_cursor: null }, delay: 4000 };
  await script(page, { "GET /api/v1/me/courses": [slow, slow, slow] } as never);
  await asJwt(page, "STUDENT", { email: "sv@ptit.edu.vn" });
  await page.goto("/practice");
  await expect(page.getByRole("heading", { name: "Luyện đề", level: 1 })).toBeVisible();
  await expect(page.getByRole("tab", { name: /Theo chủ đề/ })).toBeVisible();
  await expect(page.getByRole("tab", { name: /Thi thử/ })).toBeVisible();
  await expect(page.getByRole("link", { name: "Lịch sử luyện tập" })).toBeVisible();
});
