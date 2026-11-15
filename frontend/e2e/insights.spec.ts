import { expect, test, type Page } from "@playwright/test";
import { asJwt } from "./support/session";
import { COURSE, cors, fail, insightsApi, ok, REPORT } from "./support/insights-fixtures";

const GEN = "**/api/v1/courses/*/insights/knowledge-gap/generate";
const JOB = { id: "00000000-0000-7000-8000-0000000000b1" };

async function open(page: Page, latest: unknown | null) {
  await asJwt(page, "TEACHER");
  const calls = await insightsApi(page, { latest });
  await page.goto("/insights");
  return calls;
}

test("lớp chưa có báo cáo: một EmptyState + đúng một nút «Tạo báo cáo mới», không vẽ báo cáo mô phỏng (AC2)", async ({ page }) => {
  await open(page, null);
  await expect(page.getByText("Chưa có báo cáo")).toBeVisible();
  await expect(page.getByRole("button", { name: "Tạo báo cáo mới" })).toHaveCount(1);
  await expect(page.getByText("Chủ đề nên dạy lại")).toHaveCount(0);
});

test("375 px: mục gập «Câu hỏi mẫu» / «Chủ đề khác» có vùng chạm ≥ 44 px (TC-05-21)", async ({ page }) => {
  await page.setViewportSize({ width: 375, height: 812 });
  const base = REPORT.report.payload.topics;
  const topics = Array.from({ length: 10 }, (_, i) => ({ ...base[i % 2], rank: i + 1, name: `Chủ đề thử ${i + 1}` })); // > 8 chủ đề → có «Chủ đề khác»
  await open(page, { ...REPORT, report: { ...REPORT.report, payload: { ...REPORT.report.payload, topics } } });
  await expect(page.getByText(/^Chủ đề khác \(\d+\)$/)).toBeVisible();
  const boxes = await page.locator("main summary").evaluateAll((els) => els.map((e) => Math.round(e.getBoundingClientRect().height)));
  expect(boxes.length).toBeGreaterThan(1);
  for (const h of boxes) expect(h).toBeGreaterThanOrEqual(44);
});

test("có báo cáo: chủ đề xếp hạng, câu mẫu gập, «Ít người hỏi» không có câu mẫu, chưa đề cập, so sánh, dòng ẩn danh (AC3, 6, 7, 11)", async ({ page }) => {
  await open(page, REPORT);
  await expect(page.getByRole("heading", { name: "Bắt tay ba bước TCP" })).toBeVisible();
  const names = await page.locator("h3.ep-item-title").allTextContents();
  expect(names).toEqual(["Bắt tay ba bước TCP", "Truy vấn SQL nhiều bảng", "Docker compose"]);
  await expect(page.getByText("Câu hỏi đã ẩn danh, dùng để cải thiện giảng dạy")).toBeVisible();
  // câu mẫu: gập; chủ đề < 3 sinh viên: nhãn, không mục gập
  await expect(page.getByText("Bắt tay ba bước TCP là gì?")).toBeHidden();
  await page.getByText("Câu hỏi mẫu (3)").first().click();
  await expect(page.getByText("Bắt tay ba bước TCP là gì?")).toBeVisible();
  await expect(page.getByText("Ít người hỏi")).toHaveCount(1);
  await expect(page.getByText("Câu hỏi mẫu")).toHaveCount(2);
  await expect(page.getByText("Chưa đặt được tên chủ đề")).toBeVisible();
  await expect(page.getByText("Cách cấu hình bộ định tuyến lượng tử")).toBeVisible();
  await expect(page.getByText("10 lần")).toBeVisible();
  await expect(page.getByRole("heading", { name: "So với báo cáo ngày 22/10/2026" })).toBeVisible();
  await expect(page.getByText("Chủ đề mới: Docker compose")).toBeVisible();
  await expect(page.getByText(/Giảm hạng: Truy vấn SQL nhiều bảng \(hạng 1 → 2\)/)).toBeVisible();
});

test("tạo báo cáo: chỉ báo giai đoạn, xong tải lại báo cáo (AC4)", async ({ page }) => {
  let state = 0;
  let latest: unknown = null;
  await asJwt(page, "TEACHER");
  await insightsApi(page, {});
  await page.route("**/api/v1/courses/*/insights/knowledge-gap/latest", (r) => (latest ? ok(r, latest) : fail(r, 404, "NOT_FOUND")));
  await page.route(GEN, (r) => {
    expect(r.request().headers()["idempotency-key"]).toMatch(/^[A-Za-z0-9._:-]{8,128}$/);
    return ok(r, { job_id: JOB.id }, 202);
  });
  await page.route(`**/api/v1/jobs/${JOB.id}`, (r) => {
    state += 1;
    if (state >= 3) latest = REPORT;
    return ok(r, { id: JOB.id, status: state >= 3 ? "SUCCEEDED" : "RUNNING", progress: state >= 3 ? 100 : 40, result: state >= 3 ? { report_id: REPORT.report.id } : undefined });
  });
  await page.goto("/insights");
  await page.getByRole("button", { name: "Tạo báo cáo mới" }).click();
  await expect(page.getByText("Đang nhóm câu hỏi theo chủ đề…")).toBeVisible();
  await expect(page.getByRole("heading", { name: "Bắt tay ba bước TCP" })).toBeVisible({ timeout: 15_000 });
});

test("không có dữ liệu mới → reused; dưới 20 câu → 409; vượt trần → 429 (AC5)", async ({ page }) => {
  await open(page, REPORT);
  await page.route(GEN, (r) => ok(r, { report_id: REPORT.report.id, reused: true }));
  await page.getByRole("button", { name: "Tạo báo cáo mới" }).click();
  await expect(page.getByText("Chưa có câu hỏi mới từ ngày 14/11/2026")).toBeVisible();

  await page.unroute(GEN);
  await page.route(GEN, (r) => fail(r, 409, "NOT_ENOUGH_DATA"));
  await page.getByRole("button", { name: "Tạo báo cáo mới" }).click();
  await expect(page.getByText("Chưa đủ câu hỏi để gom (cần ≥ 20).")).toBeVisible();

  await page.unroute(GEN);
  await page.route(GEN, (r) => fail(r, 429, "RATE_LIMITED"));
  await page.getByRole("button", { name: "Tạo báo cáo mới" }).click();
  await expect(page.getByRole("alert").filter({ hasText: /\S/ })).toBeVisible();
});

test("có dữ liệu mới: «Tạo báo cáo mới» là nút chính (AC3)", async ({ page }) => {
  await open(page, { ...REPORT, has_new_data: true });
  await expect(page.locator('[data-ep-header-actions] [data-variant="primary"]')).toHaveText(/Tạo báo cáo mới/);
});

test("tạo thread ghim: nháp sửa được → Đăng tạo thread rồi ghim; lỗi giữ chữ (AC8, AC13)", async ({ page }) => {
  await open(page, REPORT);
  let posted: { title: string; body: string } | null = null;
  let pinned = false;
  await page.route("**/api/v1/courses/*/threads", (r) => {
    if (r.request().method() !== "POST") return r.fallback();
    posted = r.request().postDataJSON();
    return ok(r, { thread: { id: "00000000-0000-7000-8000-0000000000c1" } }, 201);
  });
  await page.route("**/api/v1/courses/*/threads/*/pin", (r) => {
    pinned = true;
    return r.fulfill({ status: 204, headers: cors });
  });
  await page.getByRole("button", { name: "Tạo thread ghim" }).first().click();
  const dlg = page.getByRole("group", { name: "Tạo thread ghim" });
  await expect(dlg.getByLabel("Tiêu đề")).toHaveValue("Chủ đề đang vướng: Bắt tay ba bước TCP");
  const body = dlg.getByLabel("Nội dung");
  await expect(body).toHaveValue(/Ôn lại Bắt tay ba bước TCP/);
  await body.fill("Tuần này ta ôn lại bắt tay ba bước.");
  await dlg.getByRole("button", { name: "Đăng" }).click();
  await expect(dlg.getByText("Đã đăng và ghim thread.")).toBeVisible();
  expect(posted).toMatchObject({ title: "Chủ đề đang vướng: Bắt tay ba bước TCP", body: "Tuần này ta ôn lại bắt tay ba bước.", week_no: 3 });
  expect(JSON.stringify(posted)).not.toContain("là gì?"); // không chứa câu hỏi mẫu
  expect(pinned).toBe(true);
});

test("tạo thread lỗi: hộp giữ nguyên chữ đã sửa, báo tại chỗ (AC13)", async ({ page }) => {
  await open(page, REPORT);
  await page.route("**/api/v1/courses/*/threads", (r) => (r.request().method() === "POST" ? fail(r, 422, "PII_DETECTED", "Có thông tin cá nhân") : r.fallback()));
  await page.getByRole("button", { name: "Tạo thread ghim" }).first().click();
  const dlg = page.getByRole("group", { name: "Tạo thread ghim" });
  await dlg.getByLabel("Nội dung").fill("Chữ đã sửa của giảng viên");
  await dlg.getByRole("button", { name: "Đăng" }).click();
  await expect(dlg.getByRole("alert")).toBeVisible();
  await expect(dlg.getByLabel("Nội dung")).toHaveValue("Chữ đã sửa của giảng viên");
});

test("buổi ôn tập: sự kiện lịch loại OTHER «Ôn tập: <chủ đề>» (AC9)", async ({ page }) => {
  await open(page, REPORT);
  let body: Record<string, unknown> | null = null;
  await page.route(`**/api/v1/courses/${COURSE.id}/calendar/events`, (r) => {
    if (r.request().method() !== "POST") return r.fallback();
    body = r.request().postDataJSON();
    return ok(r, { id: "00000000-0000-7000-8000-0000000000d1" }, 201);
  });
  await page.getByRole("button", { name: "Tạo buổi ôn tập" }).first().click();
  const dlg = page.getByRole("group", { name: "Tạo buổi ôn tập" });
  await expect(dlg.getByLabel("Tên buổi")).toHaveValue("Ôn tập: Bắt tay ba bước TCP");
  await expect(dlg.getByRole("button", { name: "Lưu" })).toBeDisabled();
  await dlg.getByLabel("Bắt đầu").fill("2026-11-20T18:00");
  await dlg.getByRole("button", { name: "Lưu" }).click();
  await expect(dlg.getByText("Đã thêm buổi ôn tập vào lịch của lớp.")).toBeVisible();
  expect(body).toMatchObject({ type: "OTHER", title: "Ôn tập: Bắt tay ba bước TCP", starts_at: "2026-11-20T11:00:00.000Z" });
});

test("xuất Markdown gọi đúng đường export (AC10)", async ({ page }) => {
  await open(page, REPORT);
  const asked = page.waitForRequest((r) => r.url().includes(`/insights/knowledge-gap/${REPORT.report.id}/export`) && r.url().includes("format=md"));
  await page.route("**/knowledge-gap/*/export**", (r) => r.fulfill({ status: 200, contentType: "text/markdown", headers: cors, body: "# Lỗ hổng" }));
  await page.locator("[data-ep-header-actions]").getByRole("button", { name: "Thêm" }).click();
  await page.getByRole("menuitem", { name: "Xuất Markdown" }).click();
  await asked;
});

test("API lỗi: một thông báo + Thử lại (AC13)", async ({ page }) => {
  await asJwt(page, "TEACHER");
  await insightsApi(page, {});
  await page.route("**/api/v1/courses/*/insights/knowledge-gap/latest", (r) => fail(r, 500, "INTERNAL"));
  await page.goto("/insights");
  await expect(page.getByRole("alert").filter({ hasText: /\S/ })).toHaveCount(1);
  await expect(page.getByRole("button", { name: "Thử lại" })).toBeVisible();
});

test("375 px: không tràn ngang, mục gập và nút đủ vùng chạm (AC14)", async ({ page }, info) => {
  test.skip(info.project.name !== "mobile", "chỉ bề rộng di động");
  await open(page, REPORT);
  await expect(page.getByRole("heading", { name: "Bắt tay ba bước TCP" })).toBeVisible();
  expect(await page.evaluate(() => document.documentElement.scrollWidth <= window.innerWidth)).toBe(true);
  for (const b of await page.getByRole("button", { name: /Tạo (thread ghim|buổi ôn tập)/ }).all()) expect((await b.boundingBox())!.height).toBeGreaterThanOrEqual(44);
});
