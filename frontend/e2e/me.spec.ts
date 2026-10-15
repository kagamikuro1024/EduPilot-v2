import { expect, test, type Page } from "@playwright/test";
import { COURSE, err, json } from "./support/cal-fixtures";
import { asDemo } from "./support/session";

// US-P5-06: `/me` THẬT của sinh viên. Gateway giả theo hợp đồng thật (openapi.yaml): `…/me/attendance`, `…/me/participation`, `…/me/study-time`.
const C2 = "00000000-0000-7000-8000-00000000c002";
const att = (over: Record<string, unknown> = {}) => ({
  attendance: {
    recorded: 9, present: 6, late: 1, excused: 0, absent: 2,
    sessions: [
      { session_no: 7, starts_at: "2026-10-08T02:00:00Z", status: "ABSENT" },
      { session_no: 5, starts_at: "2026-09-24T02:00:00Z", status: "LATE" },
      { session_no: 3, starts_at: "2026-09-10T02:00:00Z", status: "ABSENT" },
      { session_no: 1, starts_at: "2026-08-27T02:00:00Z", status: "PRESENT" },
    ],
    ...over,
  },
  next_cursor: null,
});
const part = (over: Record<string, unknown> = {}) => ({
  participation: { speak_count: 2, points_total: "0.50", events: [{ date: "2026-10-01", type: "SPEAK", points: "0.25" }, { date: "2026-09-17", type: "SPEAK", points: "0.25" }], ...over },
  next_cursor: null,
});
const weeks = Array.from({ length: 8 }, (_, i) => ({ week_start: new Date(Date.UTC(2026, 7, 17 + i * 7)).toISOString().slice(0, 10), minutes: 120 + i * 10, by_area: { CHAT: 60 } }));
const study = (over: Record<string, unknown> = {}) => ({ weeks, this_week_minutes: 45, tracking: { enabled: true, locked: false }, ...over });

type Handlers = Partial<Record<"attendance" | "participation" | "study-time", () => unknown>>;
/** `calls` đếm theo lớp + đường; mỗi handler trả thân JSON, `null` = 500. */
async function api(page: Page, course: string, h: Handlers = {}, calls: string[] = []) {
  await page.route(`**/api/v1/courses/${course}/me/**`, (r) => {
    const req = r.request();
    if (req.method() === "OPTIONS") return r.fulfill({ status: 204, headers: json({}).headers });
    const leaf = new URL(req.url()).pathname.split("/").pop() as keyof Handlers;
    calls.push(`${course}:${leaf}`);
    const fallback = { attendance: att, participation: part, "study-time": study }[leaf];
    const body = (h[leaf] ?? fallback)();
    return body === null ? r.fulfill(err(500, "INTERNAL")) : r.fulfill(json(body));
  });
  return calls;
}

test.beforeEach(async ({ context }, info) => {
  test.skip(info.project.name !== "desktop", "ca desktop; me 375 tự đặt bề rộng");
  await asDemo(context, "student", { person: "sv-2" });
});

test("me content and layout: chuyên cần một câu + buổi vắng / muộn; điểm cộng một câu; thời gian học ở CUỐI; không khối điểm giả", async ({ page }) => {
  await api(page, COURSE);
  await page.goto("/me");
  await expect(page.getByRole("heading", { name: "Kết quả của tôi" })).toBeVisible();
  await expect(page.getByText("Bạn có mặt 7/9 buổi, vắng 2.")).toBeVisible();
  await expect(page.getByText(/^Buổi 7 · 08\/10$/)).toBeVisible();
  await expect(page.getByText(/^Buổi 5 · 24\/09$/)).toBeVisible();
  await expect(page.getByText(/^Buổi 1 /)).toHaveCount(0); // buổi có mặt không liệt kê
  await expect(page.getByText("Phát biểu 2 lần · +0,50.")).toBeVisible();
  await expect(page.getByText("01/10")).toBeVisible();
  await expect(page.getByText("Tuần này: 45 phút")).toBeVisible();
  const titles = await page.locator("main section h2, main [data-ep-panel] h2").allInnerTexts();
  expect(titles).toEqual(["Chuyên cần", "Điểm cộng đã ghi", "Thời gian học"]);
  for (const gone of [/Điểm quá trình/, /Giải trình/, /nếu cuối kỳ/i, /Việc sắp tới/]) await expect(page.getByText(gone)).toHaveCount(0);
});

test("me no gamification: không streak, xếp hạng, vòng tiến độ, huy hiệu, confetti, so sánh bạn học", async ({ page }) => {
  await api(page, COURSE);
  await page.goto("/me");
  await expect(page.getByText("Tuần này: 45 phút")).toBeVisible();
  const text = (await page.locator("main").innerText()).toLowerCase();
  for (const bad of ["streak", "rank", "xếp hạng", "huy hiệu", "confetti", "so với bạn", "trung bình lớp"]) expect(text, bad).not.toContain(bad);
  await expect(page.locator("main [role=progressbar], main progress, main [class*=ring], main [class*=confetti]")).toHaveCount(0);
});

test("me study trend small no judgement: biểu đồ thấp (≈ 96 px), không mục tiêu, không tô đỏ tuần thấp", async ({ page }) => {
  await api(page, COURSE, { "study-time": () => study({ weeks: weeks.map((w, i) => ({ ...w, minutes: i === 3 ? 5 : w.minutes })) }) });
  await page.goto("/me");
  const chart = page.locator("main svg").last();
  await expect(chart).toBeVisible();
  const box = (await chart.boundingBox())!;
  expect(box.height).toBeLessThan(140);
  expect(await page.locator("main").innerText()).not.toMatch(/mục tiêu|chưa đạt|dưới mức/i);
  await expect(page.locator("main svg [data-tone=red], main svg [class*=red]")).toHaveCount(0);
});

test("me study time off shows state: tắt đo → 'Đang tắt' thay biểu đồ; chưa có dữ liệu → khối ẩn", async ({ page }) => {
  await api(page, COURSE, { "study-time": () => study({ weeks: [], tracking: { enabled: false, locked: false } }) });
  await page.goto("/me");
  await expect(page.getByText("Đang tắt")).toBeVisible();
  await expect(page.locator("main svg")).toHaveCount(0);
  await page.unrouteAll({ behavior: "ignoreErrors" });
  await asDemo(page.context(), "student", { person: "sv-2" });
  await api(page, COURSE, { "study-time": () => study({ weeks: [], this_week_minutes: 0 }) });
  await page.goto("/me");
  await expect(page.getByText("Bạn có mặt 7/9 buổi, vắng 2.")).toBeVisible();
  await expect(page.getByRole("heading", { name: "Thời gian học" })).toHaveCount(0);
  await expect(page.getByText(/0 phút/)).toHaveCount(0);
});

test("me states: lớp chưa điểm danh → câu rỗng không nút; lỗi → Thử lại rồi tải được", async ({ page }) => {
  let broken = true;
  await api(page, COURSE, { attendance: () => (broken ? null : att({ recorded: 0, present: 0, late: 0, absent: 0, sessions: [] })) });
  await page.goto("/me");
  await expect(page.getByText(/Không tải được kết quả của bạn/)).toBeVisible();
  broken = false;
  await page.getByRole("button", { name: "Thử lại" }).click();
  await expect(page.getByText("Chưa có buổi nào được điểm danh.")).toBeVisible();
  await expect(page.locator("main section", { hasText: "Chuyên cần" }).getByRole("button")).toHaveCount(0);
  await expect(page.locator("main section", { hasText: "Chuyên cần" }).getByRole("link")).toHaveCount(0);
});

test("me offline keeps content: lỗi khi làm mới không xoá nội dung đã tải", async ({ page }) => {
  let down = false;
  await api(page, COURSE, { attendance: () => (down ? null : att()) });
  await page.goto("/me");
  await expect(page.getByText("Bạn có mặt 7/9 buổi, vắng 2.")).toBeVisible();
  down = true;
  await page.evaluate(() => window.dispatchEvent(new Event("focus")));
  await page.evaluate(() => document.dispatchEvent(new Event("visibilitychange")));
  await page.waitForTimeout(400);
  await expect(page.getByText("Bạn có mặt 7/9 buổi, vắng 2.")).toBeVisible();
});

test("me staff blocked: giảng viên / TA vào /me thấy màn chặn quyền (không chuyển hướng) và không gọi API của sinh viên", async ({ page, context }) => {
  await context.clearCookies();
  await page.unrouteAll({ behavior: "ignoreErrors" });
  await asDemo(context, "teacher");
  const calls = await api(page, COURSE);
  await page.goto("/me");
  await expect(page.getByRole("heading", { name: "Bạn không có quyền xem màn này" })).toBeVisible();
  expect(calls).toEqual([]);
});

test("me two courses isolated: đổi lớp ở bộ chọn → số đổi theo, không lẫn", async ({ page, context }) => {
  await context.clearCookies();
  await page.unrouteAll({ behavior: "ignoreErrors" });
  await asDemo(context, "student", { person: "sv-1" }); // học cả hai lớp
  const calls: string[] = [];
  await api(page, COURSE, {}, calls);
  await api(page, C2, { attendance: () => att({ recorded: 4, present: 4, late: 0, absent: 0, sessions: [] }), participation: () => part({ speak_count: 0, points_total: "0.00", events: [] }), "study-time": () => study({ tracking: { enabled: false, locked: true }, weeks: [] }) }, calls);
  await page.goto("/me");
  await expect(page.getByText("Bạn có mặt 7/9 buổi, vắng 2.")).toBeVisible();
  await page.getByRole("button", { name: /^Chọn lớp, đang xem/ }).click();
  await page.getByRole("dialog", { name: "Chọn lớp" }).getByText("761988").click();
  await expect(page.getByText("Bạn có mặt 4/4 buổi, vắng 0.")).toBeVisible();
  await expect(page.getByText("Bạn có mặt 7/9 buổi, vắng 2.")).toHaveCount(0);
  await expect(page.getByText("Đang tắt")).toBeVisible();
  expect(calls.some((c) => c.startsWith(C2))).toBe(true);
});

test("me 375: một cột, không tràn ngang, chữ Việt dài không vỡ", async ({ page }) => {
  await page.setViewportSize({ width: 375, height: 812 });
  await api(page, COURSE, { attendance: () => att({ sessions: [{ session_no: 12, starts_at: "2026-10-08T02:00:00Z", status: "EXCUSED" }] }) });
  await page.goto("/me");
  await expect(page.getByText("Bạn có mặt 7/9 buổi, vắng 2.")).toBeVisible();
  expect(await page.evaluate(() => document.documentElement.scrollWidth - window.innerWidth)).toBeLessThanOrEqual(0);
});
