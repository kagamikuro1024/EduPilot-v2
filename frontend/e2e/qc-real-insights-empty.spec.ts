import { expect, test } from "@playwright/test";
import { call, courseId, login, needReal, noOverflow, shot, sql, uiLogin } from "./qc-real-lib";

// TC-05-03 (làm TRƯỚC mọi báo cáo): lớp chưa có báo cáo → API 404, EmptyState + đúng một nút «Tạo báo cáo mới», không vẽ báo cáo mô phỏng.
needReal();

test("@real INS-EMPTY /insights lớp 761987 chưa có báo cáo (TC-05-03)", async ({ page, request }) => {
  expect(sql("select count(*) from insight_reports")[0][0], "tiền điều kiện: chưa có báo cáo nào").toBe("0");
  const tk = await login(request, "teacher");
  const cid = await courseId(request, "teacher");
  const api = await call(request, tk, "GET", `/courses/${cid}/insights/knowledge-gap/latest`);
  expect(api.status, "latest chưa có báo cáo → 404").toBe(404);
  const reqs: string[] = [];
  page.on("request", (r) => { if (r.url().includes("/api/v1/") && !r.url().includes("/auth/")) reqs.push(`${r.method()} ${r.url().split("/api/v1")[1]}`); });
  await uiLogin(page, "teacher");
  await page.goto("/insights");
  await expect(page.getByRole("button", { name: "Tạo báo cáo mới" })).toHaveCount(1);
  await shot(page, "ins-empty-1440");
  const body = await page.locator("main").innerText();
  expect(body, "không vẽ chủ đề mô phỏng").not.toMatch(/Bảo mật mạng|Chủ đề \d|Câu hỏi mẫu/);
  expect(reqs.some((r) => r.includes("knowledge-gap/latest")), "màn đọc latest").toBe(true);
  await page.setViewportSize({ width: 375, height: 812 });
  await page.reload();
  await expect(page.getByRole("button", { name: "Tạo báo cáo mới" })).toHaveCount(1);
  await noOverflow(page);
  await shot(page, "ins-empty-375");
});
