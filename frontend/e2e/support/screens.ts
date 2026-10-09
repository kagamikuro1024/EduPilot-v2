import type { BrowserContext, Page } from "@playwright/test";
import { BASE_URL } from "./env";
import { asDemo } from "./session";
import { mockAdminApi, mockLlmApi, mockStaffApi, STAFF_DRAFT, STAFF_EXAM } from "./staffMock";
import { mockTake, TAKE_COURSE, TAKE_EXAM } from "./takeMock";

/** 13 màn chính của sprint 5.5 (SRS 8.2) với dữ liệu giả CỐ ĐỊNH — dùng cho ảnh trước / sau (`shots.spec.ts`) và phép đo diện tích đỏ (`panels.spec.ts`). */
export const FROZEN = new Date("2026-10-29T09:20:00+07:00");
export const LONG_EXP = 366 * 86400; // đồng hồ đóng băng sau giờ thật 19 ngày: token mặc định (15 phút) sẽ hết hạn

const cors = { "Access-Control-Allow-Origin": BASE_URL, "Access-Control-Allow-Credentials": "true", Vary: "Origin" };
const json = (body: unknown) => ({ status: 200, contentType: "application/json", headers: cors, body: JSON.stringify(body) });
const COURSE = { id: "00000000-0000-7000-8000-00000000c001", class_code: "761987" };

export async function mockStudentToday(page: Page) {
  const body = {
    no_course: false, email_verified: true, continue: [],
    recommended: { id: "r1", kind: "EXAM_OPEN", title: "Làm bài thi Tuần 9", reason: "Bài thi đóng lúc 21:00 hôm nay.", urgency: "high", href: "/exams", course: null, estimate_minutes: 45 },
    timeline: [{ at: "2026-10-29T02:00:00Z", ends_at: "2026-10-29T04:30:00Z", title: "Buổi 10 · An ninh mạng", place: "P.302", state: "NOW", course: COURSE }],
  };
  for (const url of ["**/api/v1/me/today**", "**/api/v1/courses/*/today**"]) await page.route(url, (r) => r.fulfill(json(body)));
}

export async function mockStaffToday(page: Page) {
  const body = {
    count: 2,
    actions: [
      { id: "a1", kind: "COURSE_SETUP", title: "Thiết lập lớp mới · 761987", reason: "1/4 bước xong: chia sẻ mã lớp → tải quy chế → tạo lịch → tải tài liệu.", urgency: "high", href: "/class/settings", course: COURSE, steps: [] },
      { id: "a2", kind: "JOIN_REQUESTS", title: "3 yêu cầu vào lớp", reason: "Sinh viên chờ duyệt.", urgency: "normal", href: "/class/members", course: COURSE },
    ],
    attention: [],
    upcoming: [{ at: "2026-10-29T02:00:00Z", title: "Buổi 10 · An ninh mạng", place: "P.302", course: COURSE }],
  };
  for (const url of ["**/api/v1/me/today**", "**/api/v1/courses/*/today**"]) await page.route(url, (r) => r.fulfill(json(body)));
}

export async function mockStudentExams(page: Page) {
  const row = (id: string, title: string, status: string, my: unknown, score: string | null) => ({ id, title, status, my_attempt: my, my_score: score, instructions: null, kind: "MCQ", duration_minutes: 45, max_score: "10.00", opens_at: "2026-10-29T01:00:00Z", closes_at: "2026-10-29T14:00:00Z" });
  const items = [
    row("s-1", "Tuần 9", "OPEN", null, null),
    row("s-3", "Tuần 10", "SCHEDULED", null, null),
    row("s-4", "Tuần 8", "PUBLISHED", { id: "a-2", status: "GRADED", deadline_at: "2026-10-22T02:00:00Z", submitted_at: "2026-10-22T01:50:00Z" }, "8.50"),
  ];
  await page.route(/\/api\/v1\/courses\/[^/]+\/exams(\?|$)/, (r) => r.fulfill(json({ items, next_cursor: null })));
}

export type Screen = { name: string; path: string; setup: (page: Page, context: BrowserContext) => Promise<void>; waitFor: string };
const student = async (page: Page, context: BrowserContext) => { await asDemo(context, "student", { expIn: LONG_EXP }); void page; };

export const SCREENS: Screen[] = [
  { name: "home-student", path: "/", waitFor: "main h2", setup: async (page, context) => { await student(page, context); await mockStudentToday(page); } },
  { name: "exams-student", path: "/exams", waitFor: "main h2", setup: async (page, context) => { await student(page, context); await mockStudentExams(page); } },
  { name: "take-pre", path: `/exams/${TAKE_EXAM}/take?course=${TAKE_COURSE}`, waitFor: "main button", setup: async (page) => { await mockTake(page, "intro", { now: FROZEN.getTime(), expIn: LONG_EXP }); } },
  { name: "take-run", path: `/exams/${TAKE_EXAM}/take?course=${TAKE_COURSE}`, waitFor: "main [role=radio], main input[type=radio]", setup: async (page) => { await mockTake(page, "running", { now: FROZEN.getTime(), expIn: LONG_EXP }); } },
  { name: "home-teacher", path: "/", waitFor: "main h2", setup: async (page, context) => { await asDemo(context, "teacher", { expIn: LONG_EXP }); await mockStaffToday(page); } },
  { name: "members", path: "/class/members", waitFor: "main table", setup: async (page, context) => { await asDemo(context, "teacher", { expIn: LONG_EXP }); await mockStaffApi(page); } },
  { name: "questions", path: "/questions", waitFor: "main table", setup: async (page, context) => { await asDemo(context, "teacher", { expIn: LONG_EXP }); await mockStaffApi(page); } },
  { name: "exam-editor", path: `/exams/${STAFF_DRAFT}`, waitFor: "main form", setup: async (page, context) => { await asDemo(context, "teacher", { expIn: LONG_EXP }); await mockStaffApi(page); } },
  { name: "exam-results", path: `/exams/${STAFF_EXAM}/results`, waitFor: "main table", setup: async (page, context) => { await asDemo(context, "teacher", { expIn: LONG_EXP }); await mockStaffApi(page); } },
  { name: "admin-users", path: "/admin/users", waitFor: "main table", setup: async (page, context) => { await asDemo(context, "admin", { expIn: LONG_EXP }); await mockAdminApi(page); } },
  { name: "admin-courses", path: "/admin/courses", waitFor: "main table", setup: async (page, context) => { await asDemo(context, "admin", { expIn: LONG_EXP }); await mockAdminApi(page); } },
  { name: "settings-llm", path: "/settings/llm", waitFor: "main h2", setup: async (page, context) => { await asDemo(context, "admin", { expIn: LONG_EXP }); await mockLlmApi(page); } },
  { name: "login", path: "/login", waitFor: "main form, form", setup: async () => {} },
];

/** Mở một màn ở bề rộng `w`, đóng băng đồng hồ, chờ phông chữ + dữ liệu. */
export async function openScreen(page: Page, context: BrowserContext, s: Screen, w: number) {
  await page.setViewportSize({ width: w, height: w === 375 ? 812 : 900 });
  await page.clock.setFixedTime(FROZEN);
  await s.setup(page, context);
  await page.goto(s.path);
  await page.locator(s.waitFor).first().waitFor({ state: "attached" });
  await page.evaluate(() => document.fonts.ready);
  await page.waitForTimeout(400);
}
