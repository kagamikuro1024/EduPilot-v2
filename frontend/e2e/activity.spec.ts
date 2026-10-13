import { expect, test, type BrowserContext, type Page } from "@playwright/test";
import { BASE_URL } from "./support/env";
import { asDemo } from "./support/session";

// US-P5-03: heartbeat thời gian học, hộp thoại minh bạch, công tắc đo ở /settings và /class/settings. Gateway giả theo hợp đồng thật (openapi.yaml);
// đồng hồ giả của Playwright điều khiển chu kỳ 30 s / idle 120 s.
const C1 = "00000000-0000-7000-8000-00000000c001";
const CORS = { "Access-Control-Allow-Origin": BASE_URL, "Access-Control-Allow-Credentials": "true", Vary: "Origin", "Access-Control-Allow-Headers": "authorization,content-type,x-request-id,idempotency-key,accept", "Access-Control-Allow-Methods": "GET,POST,PUT,DELETE,OPTIONS" };
const json = (body: unknown, status = 200) => ({ status, contentType: "application/json", headers: CORS, body: JSON.stringify(body) });
const item = (role: string) => ({ course: { id: C1, class_code: "761987", subject_code: "INT1006", name: "An ninh mạng", semester: "2026-2027-HK1", status: "ACTIVE" }, role_in_course: role, enrollment_status: "ACTIVE" });

type Beat = { course_id: string; area: string; tab_id: string };

test.beforeEach(async ({}, info) => {
  test.skip(info.project.name !== "desktop", "ca desktop");
});

/** Vào trang với đồng hồ giả; `acked` = đã bấm "Đã hiểu". Pha ngẫu nhiên của hook = 0 nên nhịp đầu rơi ngay. Trả mảng thân các nhịp đã gửi. */
async function open(page: Page, context: BrowserContext, path: string, o: { acked?: boolean; role?: "student" | "ta" | "teacher"; optOut?: boolean } = {}) {
  const beats: Beat[] = [];
  const acked = { v: o.acked ?? true };
  await asDemo(context, o.role ?? "student", { person: "sv-2" });
  await page.addInitScript(() => { Math.random = () => 0; });
  await page.clock.install({ time: new Date() });
  await page.route("**/api/v1/me/courses**", (r) => (r.request().method() === "OPTIONS" ? r.fulfill({ status: 204, headers: CORS }) : r.fulfill(json({ items: [item(o.role === "ta" ? "TA" : o.role === "teacher" ? "TEACHER" : "STUDENT")], next_cursor: null, tracking_notice_acked: acked.v, study_time_opt_out_available: o.optOut ?? true }))));
  await page.route("**/api/v1/activity/heartbeat", (r) => {
    if (r.request().method() === "OPTIONS") return r.fulfill({ status: 204, headers: CORS });
    beats.push(JSON.parse(r.request().postData() ?? "{}"));
    return r.fulfill({ status: 204, headers: CORS });
  });
  await page.route("**/api/v1/me/tracking-notice/ack", (r) => {
    if (r.request().method() === "OPTIONS") return r.fulfill({ status: 204, headers: CORS });
    acked.v = true;
    return r.fulfill({ status: 204, headers: CORS });
  });
  const loaded = page.waitForResponse((r) => r.url().includes("/me/courses"));
  await page.goto(path);
  await loaded;
  await page.waitForTimeout(300); // thời gian THẬT để React dựng lại hook sau khi lớp / cờ đã về; đồng hồ giả chỉ chạy khi test ra lệnh
  return beats;
}

/** Yêu cầu giả được xử lý bất đồng bộ (thời gian THẬT): đợi số nhịp đạt `n`, hoặc — khi khẳng định "không thêm" — đợi một nhịp thật rồi đọc. */
const settled = async (beats: Beat[], n: number) => expect.poll(() => beats.length).toBe(n);
const quiet = async (page: Page, beats: Beat[], n: number) => {
  await page.waitForTimeout(300);
  expect(beats.length).toBe(n);
};

const hide = (page: Page, hidden: boolean) =>
  page.evaluate((h) => {
    Object.defineProperty(document, "visibilityState", { configurable: true, get: () => (h ? "hidden" : "visible") });
    document.dispatchEvent(new Event("visibilitychange"));
  }, hidden);

test("heartbeat hidden tab stops: ẩn tab → không nhịp nào; hiện lại + thao tác → gửi tiếp", async ({ page, context }) => {
  const beats = await open(page, context, "/chat");
  await page.clock.runFor(1_000);
  await settled(beats, 1);
  await hide(page, true);
  await page.clock.runFor(150_000);
  await quiet(page, beats, 1);
  await hide(page, false);
  await page.keyboard.press("Shift");
  await page.clock.runFor(1_000);
  await settled(beats, 2);
});

test("heartbeat idle 2min stops: không thao tác 2 phút → ngừng; nhịp cuối trước mốc idle", async ({ page, context }) => {
  const beats = await open(page, context, "/threads");
  await page.clock.runFor(130_000);
  await page.waitForTimeout(300);
  const atIdle = beats.length;
  expect(atIdle).toBeGreaterThanOrEqual(4);
  expect(atIdle).toBeLessThanOrEqual(5);
  await page.clock.runFor(240_000);
  await quiet(page, beats, atIdle);
});

test("heartbeat resumes on input: nghỉ quá 2 phút rồi chạm / gõ → nhịp tiếp trong ≤ 1 s", async ({ page, context }) => {
  const beats = await open(page, context, "/library");
  await page.clock.runFor(200_000);
  await page.waitForTimeout(300);
  const before = beats.length;
  await page.mouse.click(300, 300);
  await page.clock.runFor(1_000);
  await settled(beats, before + 1);
});

test("heartbeat cleanup on route change: rời khu vực đo → dọn bộ hẹn giờ, không nhịp nào nữa", async ({ page, context }) => {
  const beats = await open(page, context, "/chat");
  await page.clock.runFor(1_000);
  await settled(beats, 1);
  await page.locator("[data-part=sidebar]").getByRole("link", { name: "Hôm nay" }).first().click();
  await page.keyboard.press("Shift");
  await page.clock.runFor(100_000);
  await quiet(page, beats, 1);
});

test("heartbeat areas only: chỉ course_id, area, tab_id; area đúng khu vực của trang", async ({ page, context }) => {
  const beats = await open(page, context, "/chat");
  await page.clock.runFor(1_000);
  for (const [path, area] of [["/threads", "THREADS"], ["/library", "LIBRARY"], ["/calendar", "CALENDAR"]] as const) {
    await page.locator("[data-part=sidebar]").locator(`a[href="${path}"]`).click();
    await page.keyboard.press("Shift");
    await page.clock.runFor(31_000);
    await expect.poll(() => beats.at(-1)?.area, { message: path }).toBe(area);
  }
  for (const b of beats) {
    expect(Object.keys(b).sort()).toEqual(["area", "course_id", "tab_id"]);
    expect(b.course_id).toBe(C1);
    expect(["CHAT", "THREADS", "PRACTICE", "LIBRARY", "CALENDAR"]).toContain(b.area);
    expect(b.tab_id).toMatch(/^[0-9a-f-]{36}$/);
  }
  expect(new Set(beats.map((b) => b.tab_id)).size).toBeGreaterThanOrEqual(1);
});

test("heartbeat not on exam or me: /me, /exams, Hôm nay không gửi nhịp", async ({ page, context }) => {
  const beats = await open(page, context, "/me");
  await page.clock.runFor(100_000);
  await page.goto("/exams");
  await page.clock.runFor(100_000);
  await page.goto("/");
  await page.clock.runFor(100_000);
  await page.waitForTimeout(300);
  expect(beats).toEqual([]);
});

test("tracking notice first login: chưa Đã hiểu → hộp thoại, KHÔNG nhịp; Đã hiểu → POST ack, hộp thoại tắt, nhịp bắt đầu", async ({ page, context }) => {
  const beats = await open(page, context, "/chat", { acked: false });
  const dialog = page.getByRole("dialog", { name: "Thời gian học trên EduPilot" });
  await expect(dialog).toBeVisible();
  await expect(dialog).toContainText("EduPilot ghi thời gian bạn dùng Chat, Threads, Luyện đề, Thư viện và Lịch khi tab đang mở và bạn đang thao tác.");
  await expect(dialog).toContainText("Không ghi nội dung bạn xem hay gõ.");
  await expect(dialog).toContainText("Giảng viên xem được; không dùng để tính điểm.");
  await expect(dialog).toContainText("Bạn có thể tắt ở Cài đặt.");
  await page.clock.runFor(100_000);
  expect(beats).toEqual([]);
  const ack = page.waitForRequest((r) => r.url().endsWith("/me/tracking-notice/ack") && r.method() === "POST");
  await dialog.getByRole("button", { name: "Đã hiểu" }).click();
  await ack;
  await expect(page.getByRole("dialog")).toHaveCount(0);
  await page.keyboard.press("Shift");
  await page.clock.runFor(1_000);
  await settled(beats, 1);
});

test("tracking notice optout line hidden when no class allows opt-out", async ({ page, context }) => {
  await open(page, context, "/chat", { acked: false, optOut: false });
  await expect(page.getByRole("dialog")).toContainText("Giảng viên xem được; không dùng để tính điểm.");
  await expect(page.getByRole("dialog")).not.toContainText("Bạn có thể tắt ở Cài đặt.");
});

test("tracking notice not shown to staff: TA / giảng viên không hộp thoại, không nhịp", async ({ page, context }) => {
  for (const role of ["ta", "teacher"] as const) {
    const beats = await open(page, context, "/chat", { acked: false, role });
    await page.clock.runFor(60_000);
    await expect(page.getByRole("dialog")).toHaveCount(0);
    expect(beats, role).toEqual([]);
    await context.clearCookies();
  }
});

// ---- /settings ----
const settingsBody = (o: { track: boolean; locked: boolean; acked?: boolean }) => ({
  notify_ticket_by_mail: false, notify_answer_by_mail: true, remind_deadline_by_mail: true, track_study_time: o.track, reminders: { exam: true, class_session: false, other: true }, version: 3,
  study_time: { acked: o.acked ?? true, courses: [{ course_id: C1, class_code: "761987", name: "An ninh mạng", locked: o.locked }] },
});

test("study time opt out on allowed course: công tắc đổi được, lưu lạc quan kèm version, không toast", async ({ page, context }) => {
  await open(page, context, "/settings");
  let cur = settingsBody({ track: true, locked: false });
  const puts: unknown[] = [];
  await page.route("**/api/v1/me/settings", (r) => {
    if (r.request().method() === "OPTIONS") return r.fulfill({ status: 204, headers: CORS });
    if (r.request().method() === "PUT") {
      const b = JSON.parse(r.request().postData() ?? "{}");
      puts.push(b);
      cur = { ...cur, track_study_time: b.track_study_time, version: cur.version + 1 };
    }
    return r.fulfill(json(cur));
  });
  await page.goto("/settings");
  const sw = page.getByRole("switch", { name: "761987 · An ninh mạng" });
  await expect(sw).toBeChecked();
  await expect(sw).toBeEnabled();
  await sw.click();
  await expect(sw).not.toBeChecked();
  expect(puts).toEqual([{ track_study_time: false, version: 3 }]);
  await expect(page.locator("[data-ep-toast]")).toHaveCount(0);
});

test("study time locked on strict course: bật và khoá, tooltip lý do ở chính công tắc", async ({ page, context }) => {
  await open(page, context, "/settings");
  await page.route("**/api/v1/me/settings", (r) => (r.request().method() === "OPTIONS" ? r.fulfill({ status: 204, headers: CORS }) : r.fulfill(json(settingsBody({ track: false, locked: true })))));
  await page.goto("/settings");
  const sw = page.getByRole("switch", { name: "761987 · An ninh mạng" });
  await expect(sw).toBeChecked();
  await expect(sw).toBeDisabled();
  await expect(sw).toHaveAttribute("title", "Lớp này luôn ghi thời gian học");
  await expect(page.getByText("Lớp này luôn ghi thời gian học")).toHaveCount(0); // không dòng mô tả
});

test("study time group hidden before acknowledgement", async ({ page, context }) => {
  await open(page, context, "/settings", { acked: false });
  await page.route("**/api/v1/me/settings", (r) => (r.request().method() === "OPTIONS" ? r.fulfill({ status: 204, headers: CORS }) : r.fulfill(json(settingsBody({ track: true, locked: false, acked: false })))));
  await page.goto("/settings");
  await expect(page.getByRole("heading", { name: "Nhắc trước 24 giờ" })).toBeVisible();
  await expect(page.getByRole("switch", { name: "761987 · An ninh mạng" })).toHaveCount(0);
});

// ---- /class/settings ----
const detail = (o: { allow: boolean; students: number }) => ({
  id: C1, class_code: "761987", subject_code: "INT1006", name: "An ninh mạng", semester: "2026-2027-HK1", status: "ACTIVE", allow_study_time_opt_out: o.allow, teachers: [{ full_name: "TS. Lê Thu Hà" }], my_role: "TEACHER",
  counts: { students_active: o.students, students_pending: 0 }, settings: { escalation_threshold: "0.80", version: 5 },
});

async function classSettings(page: Page, context: BrowserContext, role: "teacher" | "ta", d: ReturnType<typeof detail>, putStatus = 200) {
  await open(page, context, "/", { role });
  const puts: unknown[] = [];
  await page.route(`**/api/v1/courses/${C1}`, (r) => (r.request().method() === "OPTIONS" ? r.fulfill({ status: 204, headers: CORS }) : r.fulfill(json({ ...d, my_role: role === "ta" ? "TA" : "TEACHER" }))));
  await page.route(`**/api/v1/courses/${C1}/settings`, (r) => {
    if (r.request().method() === "OPTIONS") return r.fulfill({ status: 204, headers: CORS });
    puts.push(JSON.parse(r.request().postData() ?? "{}"));
    return r.fulfill(putStatus === 200 ? json({ ...d, allow_study_time_opt_out: false }) : json({ code: "COURSE_HAS_STUDENTS", message: "m", trace_id: "b".repeat(32) }, putStatus));
  });
  await page.route(`**/api/v1/courses/${C1}/join-code`, (r) => r.fulfill(json({ join_code: "ABCDEFG", join_url: "http://x/join", enabled: true, expires_at: null, require_approval: true, allowed_email_domain: null, capacity: null, active_students: d.counts.students_active, pending: 0, version: 1 })));
  await page.goto("/class/settings");
  return puts;
}

test("class settings opt out locks after first student: chưa có sinh viên → đổi được; đã có → khoá kèm lý do", async ({ page, context }) => {
  const puts = await classSettings(page, context, "teacher", detail({ allow: true, students: 0 }));
  const sw = page.getByRole("switch", { name: "Cho sinh viên tắt đo thời gian học" });
  await expect(sw).toBeEnabled();
  await sw.click();
  expect(puts).toEqual([{ allow_study_time_opt_out: false, version: 5 }]);
});

test("class settings opt out locked: đã có sinh viên → công tắc disabled, tooltip ở chính công tắc", async ({ page, context }) => {
  await classSettings(page, context, "teacher", detail({ allow: true, students: 1 }));
  const sw = page.getByRole("switch", { name: "Cho sinh viên tắt đo thời gian học" });
  await expect(sw).toBeDisabled();
  await expect(sw).toHaveAttribute("title", "Lớp đã có sinh viên");
});

test("class settings opt out race: 409 COURSE_HAS_STUDENTS → hoàn lại + một dòng", async ({ page, context }) => {
  await classSettings(page, context, "teacher", detail({ allow: true, students: 0 }), 409);
  const sw = page.getByRole("switch", { name: "Cho sinh viên tắt đo thời gian học" });
  await sw.click();
  await expect(page.getByText("Lớp đã có sinh viên")).toBeVisible();
  await expect(sw).toBeChecked();
});

test("class settings opt out TA read-only", async ({ page, context }) => {
  await classSettings(page, context, "ta", detail({ allow: true, students: 0 }));
  await expect(page.getByRole("switch", { name: "Cho sinh viên tắt đo thời gian học" })).toBeDisabled();
});
