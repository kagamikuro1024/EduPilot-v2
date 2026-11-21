import { expect, test, type Page } from "@playwright/test";
import { asJwt } from "./support/session";
import { fail, insightsApi, ok, REPORT } from "./support/insights-fixtures";

const E1 = "00000000-0000-7000-8000-0000000000e1";
const E2 = "00000000-0000-7000-8000-0000000000e2";
const S1 = "00000000-0000-7000-8000-0000000000f1";
const TREND = [
  { exam_id: E1, title: "Tuần 1", opens_at: "2026-11-02T02:00:00Z", submissions: 6, pass_rate: "0.50", mean: "5.83", median: "5.50" },
  { exam_id: E2, title: "Tuần 2", opens_at: "2026-11-09T02:00:00Z", submissions: 4, pass_rate: "0.75", mean: "7.50", median: "7.00" },
];
const overview = (examId: string) => ({
  scope: "exam", exams: [{ id: examId, title: examId === E1 ? "Tuần 1" : "Tuần 2", opens_at: "2026-11-09T02:00:00Z", max_score: "10.00" }], submissions: examId === E1 ? 6 : 4,
  pass_rate: examId === E1 ? "0.50" : "0.75", mean: examId === E1 ? "5.83" : "7.50", median: examId === E1 ? "5.50" : "7.00",
  distribution: Array.from({ length: 10 }, (_, i) => ({ from: `${i}.00`, to: `${i + 1}.00`, count: i === 7 ? 3 : i === 5 ? 1 : 0 })),
});
const HARD = [
  { item_id: "00000000-0000-7000-8000-0000000000a1", exam_id: E2, position: 1, type: "MCQ", title: "Bắt tay TCP", topic: "Bắt tay ba bước TCP", attempts: 4, wrong_rate: "0.7500", ce_rate: null },
  { item_id: "00000000-0000-7000-8000-0000000000a2", exam_id: E2, position: 2, type: "CODE", title: "Cài đặt socket", topic: "", attempts: 4, wrong_rate: "0.5000", ce_rate: "0.2500" },
];
const DECL = [{ student_id: S1, name: "Sinh Viên 04", student_code: "B21DCCN004", scores: [{ exam_id: E1, title: "B2", score: "9.00" }, { exam_id: E2, title: "B3", score: "7.00" }, { exam_id: E2, title: "B4", score: "5.00" }] }];

type Mock = { trend?: unknown; compare?: unknown[]; latest?: unknown | null; overviewHandler?: Parameters<Page["route"]>[1] };

async function open(page: Page, m: Mock = {}) {
  await asJwt(page, "TEACHER");
  await insightsApi(page, { latest: m.latest === undefined ? REPORT : m.latest });
  const base = "**/api/v1/courses/*/analytics/exams";
  await page.route(`${base}/trend`, (r) => ok(r, { items: m.trend ?? TREND }));
  await page.route(`${base}/overview**`, m.overviewHandler ?? ((r) => ok(r, overview(new URL(r.request().url()).searchParams.get("exam_id") ?? E2))));
  await page.route(`${base}/hard-items**`, (r) => ok(r, { items: HARD, next_cursor: null }));
  await page.route(`${base}/declining**`, (r) => ok(r, { items: DECL, next_cursor: null }));
  await page.route("**/api/v1/analytics/exams/compare", (r) => ok(r, { items: m.compare ?? [] }));
  await page.goto("/analytics");
  await page.getByRole("tab", { name: "Điểm thi" }).click();
}

test("tab Điểm thi: bài mới nhất, một dòng chữ, phân bố, xu hướng, câu sai, điểm đang giảm (AC1, AC5)", async ({ page }) => {
  await open(page);
  await expect(page.getByText("Trung bình 7,5 · trung vị 7 · đạt 75% · 4 bài nộp")).toBeVisible();
  await expect(page.getByRole("figure", { name: "Phân bố điểm" })).toBeVisible();
  await expect(page.getByRole("heading", { name: "Xu hướng qua các bài thi" })).toBeVisible();
  await expect(page.getByRole("table", { name: "Câu sai nhiều nhất" }).getByText("Bài code")).toBeVisible();
  await expect(page.getByRole("table", { name: "Câu sai nhiều nhất" }).getByText("75%")).toBeVisible();
  await expect(page.getByRole("link", { name: "Sinh Viên 04" })).toHaveAttribute("href", `/students/${S1}`);
  await expect(page.getByText("9 → 7 → 5")).toBeVisible();
});

test("chọn bài khác gọi lại overview với exam_id (AC1)", async ({ page }) => {
  const asked: string[] = [];
  await open(page, { overviewHandler: (r) => { asked.push(r.request().url()); return ok(r, overview(new URL(r.request().url()).searchParams.get("exam_id") ?? E2)); } });
  await page.locator("main").getByLabel("Bài thi", { exact: true }).selectOption(E1);
  await expect(page.getByText("Trung bình 5,83 · trung vị 5,5 · đạt 50% · 6 bài nộp")).toBeVisible();
  expect(asked.some((u) => u.includes(`exam_id=${E1}`))).toBe(true);
});

test("theo tuần: gửi week, tuần không có bài → «Chưa có bài thi» (AC3)", async ({ page }) => {
  const asked: string[] = [];
  await open(page, { overviewHandler: (r) => { asked.push(r.request().url()); return ok(r, { scope: "week", exams: [], submissions: 0, pass_rate: null, mean: null, median: null, distribution: null }); } });
  await page.getByRole("radio", { name: "Theo tuần" }).click();
  await page.locator("main").getByLabel("Tuần", { exact: true }).fill("2026-W20");
  await expect(page.getByText("Chưa có bài thi", { exact: true })).toBeVisible();
  expect(asked.some((u) => u.includes("week=2026-W20"))).toBe(true);
});

test("liên kết «Xem ở Báo cáo lỗ hổng» chỉ khi chủ đề trùng báo cáo; lớp chưa có báo cáo → không liên kết (AC4)", async ({ page }) => {
  await open(page);
  const t = page.getByRole("table", { name: "Câu sai nhiều nhất" });
  await expect(t.getByRole("link", { name: "Xem ở Báo cáo lỗ hổng" })).toHaveCount(1);
});

test("lớp chưa có báo cáo: không liên kết (AC4)", async ({ page }) => {
  await open(page, { latest: null });
  await expect(page.getByRole("table", { name: "Câu sai nhiều nhất" })).toBeVisible();
  await expect(page.getByRole("link", { name: "Xem ở Báo cáo lỗ hổng" })).toHaveCount(0);
});

test("so sánh lớp: chỉ khi ≥ 2 lớp; lớp dưới 5 bài nộp «Chưa đủ» (AC6)", async ({ page }) => {
  await open(page, { compare: [
    { course_id: "c1", class_code: "761987", name: "An ninh mạng", exams: 2, submissions: 10, enough: true, mean: "6.50", median: "6.00", pass_rate: "0.70" },
    { course_id: "c2", class_code: "761988", name: "Mạng máy tính", exams: 1, submissions: 4, enough: false, mean: null, median: null, pass_rate: null },
  ] });
  const t = page.getByRole("table", { name: /So sánh điểm thi/ });
  await expect(t.getByText("6,5")).toBeVisible();
  await expect(t.getByText("Chưa đủ")).toHaveCount(3);
});

test("một lớp: không có bảng so sánh (AC6)", async ({ page }) => {
  await open(page, { compare: [{ course_id: "c1", class_code: "761987", name: "An ninh mạng", exams: 2, submissions: 10, enough: true, mean: "6.50", median: "6.00", pass_rate: "0.70" }] });
  await expect(page.getByRole("heading", { name: "Điểm thi", exact: true })).toBeVisible();
  await expect(page.getByRole("heading", { name: "So sánh các lớp" })).toHaveCount(0);
});

test("lớp chưa có bài thi công bố → EmptyState một dòng (AC8)", async ({ page }) => {
  await open(page, { trend: [] });
  await expect(page.getByText("Chưa có bài thi công bố")).toBeVisible();
});

test("overview lỗi → thông báo + Thử lại (AC8)", async ({ page }) => {
  await open(page, { overviewHandler: (r) => fail(r, 500, "INTERNAL") });
  await expect(page.getByRole("alert").filter({ hasText: /\S/ })).toHaveCount(1); // bỏ vùng thông báo đường dẫn rỗng của Next
  await expect(page.getByRole("button", { name: "Thử lại" })).toBeVisible();
});

test("375 px: biểu đồ co theo bề rộng, không tràn ngang (AC10)", async ({ page }, info) => {
  test.skip(info.project.name !== "mobile", "chỉ bề rộng di động");
  await open(page);
  await expect(page.getByRole("figure", { name: "Phân bố điểm" })).toBeVisible();
  expect(await page.evaluate(() => document.documentElement.scrollWidth <= window.innerWidth)).toBe(true);
});
