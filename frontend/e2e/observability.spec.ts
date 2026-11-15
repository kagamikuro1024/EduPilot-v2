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

test("nội dung hết hạn lưu (410 PAYLOAD_EXPIRED) → một dòng, không lỗi chung", async ({ page }) => {
  await openAdmin(page, { [OPEN]: [err(410, "PAYLOAD_EXPIRED")] });
  await page.getByRole("table", { name: /Yêu cầu gửi tới model/ }).locator("tbody tr").first().click();
  await page.getByLabel("Lý do").fill("Giảng viên khiếu nại câu trả lời sai");
  await page.getByRole("button", { name: "Xem chi tiết" }).click();
  await expect(page.getByText("Nội dung đã hết hạn lưu.")).toBeVisible();
});

test("lọc yêu cầu: «Có vấn đề» gọi lại với filter=problem; lỗi API → Thử lại", async ({ page }) => {
  await openAdmin(page, { [SUM]: [err(503, "SERVICE_UNAVAILABLE")] });
  await expect(page.getByRole("button", { name: "Thử lại" }).first()).toBeVisible({ timeout: 20_000 });
  await page.getByRole("radio", { name: "Có vấn đề" }).click();
  await expect.poll(async () => (await getLog(page, "/api/v1/admin/observability/requests")).some((e) => e.search.includes("filter=problem"))).toBe(true);
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
