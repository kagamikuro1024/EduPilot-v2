import { expect, test, type Page } from "@playwright/test";
import { acquireFakeApi, releaseFakeApi } from "./support/fake-api-lock";
import { err, getLog, reset, script } from "./support/llm-fixtures";
import { asJwt } from "./support/session";

// US-P10-01..03: /observability THẬT — ADMIN thấy toàn hệ thống + bảng yêu cầu (không cột nội dung), mở nội dung đã che cần lý do (một dòng nhật ký mỗi lần mở);
// TEACHER / TA chỉ thấy tổng hợp 7 ngày của lớp mình. Máy chủ giả theo hợp đồng thật.
const SUM = "GET /api/v1/admin/observability/summary";
const REQS = "GET /api/v1/admin/observability/requests";
const OPEN = "POST /api/v1/admin/observability/requests/00000000-0000-7000-8000-0000000b0001/open";
const COURSE = "00000000-0000-7000-8000-00000000c001";
test.describe.configure({ mode: "serial" });
test.beforeAll(acquireFakeApi);
test.afterAll(releaseFakeApi);
test.beforeEach(async ({}, info) => {
  test.skip(info.project.name !== "desktop", "tự đặt bề rộng; máy chủ giả dùng chung");
});

const day = (d: string, over: Record<string, unknown> = {}) => ({ day: d, requests: 120, p95_ms: 2400, errors: 3, escalations: 2, pii_masked: 14, cost: "5200.5", ...over });
const summary = () => ({
  window: "24h",
  totals: { requests: 480, p50_ms: 900, p95_ms: 3100, errors: 12, error_rate: 0.025, fallbacks: 20, fallback_rate: 0.0417, cost: "21040.2", pii_masked: 60, window_seconds: 86400 },
  grading: { jobs_total: { QUEUED: 3, RUNNING: 2, DONE: 40, FAILED: 4 }, queue_depth: 3, dead_total: 4, duration_p50_s: 12, duration_p95_s: 95 },
  days: [day("2026-10-29"), day("2026-10-28", { requests: 95, errors: 0 })],
  tasks: [{ task: "CHAT", requests: 300, cost: "9000", tokens_in: 120000, tokens_out: 20000 }, { task: "GRADING", requests: 40, cost: "11000", tokens_in: 90000, tokens_out: 10000 }],
});
const reqRow = (n: number, over: Record<string, unknown> = {}) => ({
  id: `00000000-0000-7000-8000-0000000b000${n}`, at: "2026-10-29T02:10:00Z", task: "CHAT", lane: "INTERACTIVE", provider: "gemini", model: "gemini-flash", latency_ms: 1800, status: "ok", error_kind: null,
  degraded: false, fallback_index: 0, pii_masked_count: 2, course_id: COURSE, course_code: "761987", ...over,
});
const detail = () => ({ id: reqRow(1).id, at: "2026-10-29T02:10:00Z", task: "CHAT", provider: "gemini", model: "gemini-flash", status: "ok", trace_id: "t".repeat(32), trace_url: null, masked_prompt: "Cho em hỏi về [SV_1] vắng mấy buổi?", tools: ["get_my_attendance"], confidence: "0.910" });

async function openAdmin(page: Page, extra: Record<string, unknown[]> = {}) {
  await reset(page);
  await script(page, {
    [SUM]: [{ body: summary() }],
    [REQS]: [{ body: { items: [reqRow(1), reqRow(2, { status: "error", error_kind: "NETWORK", task: "GRADING", course_id: null, course_code: null, pii_masked_count: 0 })], next_cursor: null } }],
    ...extra,
  } as never);
  await asJwt(page, "ADMIN", { email: "admin@ptit.edu.vn" });
  await page.goto("/observability");
}

test("ADMIN: dải 24 giờ, bảng 7 ngày, chi phí theo tác vụ, bảng yêu cầu KHÔNG có cột nội dung", async ({ page }) => {
  await openAdmin(page);
  await expect(page.locator("[data-part=obs-today]")).toContainText("480 yêu cầu");
  await expect(page.locator("[data-part=obs-today]")).toContainText("lỗi 2,5%");
  await expect(page.locator("[data-part=obs-grading]")).toHaveText("Chấm bài: 3 chờ · 2 đang chạy · 4 lỗi hết lượt thử · p95 95 s"); // US-P7-07 AC10 (DEF-R9S-06)
  await expect(page.getByRole("table", { name: "Số liệu theo ngày" }).locator("tbody tr")).toHaveCount(2);
  await expect(page.getByRole("table", { name: "Chi phí 7 ngày theo tác vụ" })).toContainText("Chấm bài");
  const t = page.getByRole("table", { name: /Yêu cầu gửi tới model/ });
  await expect(t.locator("tbody tr")).toHaveCount(2);
  await expect(t).toContainText("Lớp 761987");
  await expect(t).toContainText("Ngoài lớp");
  await expect(t.locator("thead")).not.toContainText(/nội dung|prompt/i);
  expect(await page.content()).not.toContain("[SV_1]"); // chưa mở thì nội dung không có trong trang
});

test("mở nội dung: cần lý do ≥ 10 ký tự, POST mang lý do, hiện nội dung đã che + công cụ; mỗi lần mở một POST", async ({ page }) => {
  await openAdmin(page, { [OPEN]: [{ body: detail() }] });
  await page.getByRole("table", { name: /Yêu cầu gửi tới model/ }).locator("tbody tr").first().click();
  const open = page.getByRole("button", { name: "Xem chi tiết" });
  await expect(open).toBeDisabled();
  await page.getByLabel("Lý do").fill("ngắn");
  await expect(open).toBeDisabled();
  await page.getByLabel("Lý do").fill("Giảng viên khiếu nại câu trả lời sai");
  await open.click();
  await expect(page.getByText("Cho em hỏi về [SV_1] vắng mấy buổi?")).toBeVisible();
  await expect(page.getByText("get_my_attendance")).toBeVisible();
  const posts = (await getLog(page, "/api/v1/admin/observability/requests/00000000-0000-7000-8000-0000000b0001/open")).filter((e) => e.method === "POST");
  expect(posts).toHaveLength(1);
  expect(JSON.parse(posts[0].body)).toEqual({ reason: "Giảng viên khiếu nại câu trả lời sai" });
});

test("mã theo dõi: không có JAEGER (trace_url null) → hiện mã + nút sao chép, không liên kết rỗng (TC-02-13)", async ({ page, context }) => {
  await context.grantPermissions(["clipboard-read", "clipboard-write"]);
  await openAdmin(page, { [OPEN]: [{ body: detail() }] });
  await page.getByRole("table", { name: /Yêu cầu gửi tới model/ }).locator("tbody tr").first().click();
  await page.getByLabel("Lý do").fill("Giảng viên khiếu nại câu trả lời sai");
  await page.getByRole("button", { name: "Xem chi tiết" }).click();
  const drawer = page.getByRole("dialog");
  await expect(drawer.getByText("t".repeat(32))).toBeVisible();
  await expect(drawer.locator('a[href*="undefined"], a[href=""]')).toHaveCount(0);
  await drawer.getByRole("button", { name: "Sao chép mã theo dõi" }).click();
  expect(await page.evaluate(() => navigator.clipboard.readText())).toBe("t".repeat(32));
  await expect(drawer.getByRole("button", { name: "Đã sao chép" })).toBeVisible();
});

test("nội dung hết hạn lưu (410 PAYLOAD_EXPIRED) → một dòng, không lỗi chung", async ({ page }) => {
  await openAdmin(page, { [OPEN]: [err(410, "PAYLOAD_EXPIRED")] });
  await page.getByRole("table", { name: /Yêu cầu gửi tới model/ }).locator("tbody tr").first().click();
  await page.getByLabel("Lý do").fill("Giảng viên khiếu nại câu trả lời sai");
  await page.getByRole("button", { name: "Xem chi tiết" }).click();
  await expect(page.getByText("Nội dung đã hết hạn lưu.")).toBeVisible();
});

test("lọc yêu cầu: «Có vấn đề» gọi lại với filter=problem", async ({ page }) => {
  await openAdmin(page);
  await page.getByRole("radio", { name: "Có vấn đề" }).click();
  await expect.poll(async () => (await getLog(page, "/api/v1/admin/observability/requests")).some((e) => e.search.includes("filter=problem"))).toBe(true);
});

test("lỗi API tổng hợp → một dòng + Thử lại, không spinner riêng", async ({ page }) => {
  await openAdmin(page, { [SUM]: [err(503, "SERVICE_UNAVAILABLE")] });
  await expect(page.getByRole("button", { name: "Thử lại" }).first()).toBeVisible({ timeout: 20_000 });
});

test("TEACHER: chỉ tổng hợp 7 ngày của lớp mình, không bảng yêu cầu, không gọi API admin", async ({ page }) => {
  await reset(page);
  await script(page, {
    "GET /api/v1/me/courses": [{ body: { items: [{ course: { id: COURSE, class_code: "761987", subject_code: "INT1006", name: "An ninh mạng", semester: "2026-2027-HK1", status: "ACTIVE" }, role_in_course: "TEACHER", enrollment_status: "ACTIVE" }], next_cursor: null } }],
    [`GET /api/v1/courses/${COURSE}/observability/summary`]: [{ body: { course_id: COURSE, window: "7d", requests: 210, error_rate: 0.01, escalations: 4, pii_masked: 30, helpful: 20, not_helpful: 3, thread_correct: 9, thread_reject: 1, graded_changed: 2, graded_total: 8, change_rate: 0.25, mean_abs_deviation: "0.60", days: [] } }],
  } as never);
  await asJwt(page, "TEACHER", { email: "gv@ptit.edu.vn" });
  await page.goto("/observability");
  await expect(page.getByText("7 ngày qua")).toBeVisible();
  await expect(page.getByText("210")).toBeVisible();
  await expect(page.getByText("2 / 8 (25%)")).toBeVisible();
  await expect(page.getByRole("table", { name: /Yêu cầu gửi tới model/ })).toHaveCount(0);
  expect((await getLog(page, "/api/v1/admin/observability")).length).toBe(0);
});

// DEF-R10.7-01 (khung §23): ADMIN — mọi mục cách nhau 48 px; GV — meta không lặp lớp đang chọn, lớp chưa có yêu cầu dùng EmptyState.
test("ADMIN: các mục kề nhau cách đều 48 px (mục → mục), dòng «24 giờ qua» nằm trong mục 7 ngày (DEF-R10.7-01)", async ({ page }) => {
  await openAdmin(page);
  await expect(page.getByRole("heading", { name: "Yêu cầu", level: 2 })).toBeVisible();
  await expect(page.locator("[data-part=obs-today]")).toBeVisible();
  const gaps = await page.locator("main [data-ep-section]").evaluateAll((els) => els.slice(1).map((e, i) => Math.round(e.getBoundingClientRect().top - els[i].getBoundingClientRect().bottom)));
  expect(gaps.length).toBeGreaterThanOrEqual(3);
  expect(gaps).toEqual(gaps.map(() => 48));
});

async function openTeacher(page: Page, summaryBody: Record<string, unknown>) {
  await reset(page);
  await script(page, {
    "GET /api/v1/me/courses": [{ body: { items: [{ course: { id: COURSE, class_code: "761987", subject_code: "INT1006", name: "An ninh mạng", semester: "2026-2027-HK1", status: "ACTIVE" }, role_in_course: "TEACHER", enrollment_status: "ACTIVE" }], next_cursor: null } }],
    [`GET /api/v1/courses/${COURSE}/observability/summary`]: [{ body: { course_id: COURSE, window: "7d", ...summaryBody } }],
  } as never);
  await asJwt(page, "TEACHER", { email: "gv@ptit.edu.vn" });
  await page.goto("/observability");
}
const ZERO = { requests: 0, error_rate: 0, escalations: 0, pii_masked: 0, helpful: 0, not_helpful: 0, thread_correct: 0, thread_reject: 0, graded_changed: 0, graded_total: 0, change_rate: 0, mean_abs_deviation: "0", days: [] };

test("TEACHER: meta đầu trang không lặp lớp đang chọn; lớp chưa có yêu cầu AI → EmptyState (DEF-R10.7-01)", async ({ page }) => {
  await openTeacher(page, ZERO);
  await expect(page.getByRole("heading", { name: "Quan sát AI", level: 1 })).toBeVisible();
  await expect(page.locator("main [data-ep-empty]")).toHaveCount(1);
  await expect(page.locator("main [data-ep-empty]")).toContainText("Chưa có yêu cầu AI nào");
  await expect(page.locator("[data-ep-header-meta]")).toHaveCount(0); // bộ chọn lớp ở topbar đã nêu lớp; đầu trang không lặp lại
});
