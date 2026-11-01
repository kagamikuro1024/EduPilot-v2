import { expect, test, type Page } from "@playwright/test";
import { COURSE, err, json } from "./support/cal-fixtures";
import { asDemo } from "./support/session";
import { A1, A2, studentApi, studentItem } from "./support/assignment-fixtures";

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

type Handlers = Partial<Record<"attendance" | "participation" | "study-time" | "grades" | "what-if", () => unknown>>;
/** `calls` đếm theo lớp + đường; mỗi handler trả thân JSON, `null` = 500. */
async function api(page: Page, course: string, h: Handlers = {}, calls: string[] = []) {
  await page.route(`**/api/v1/courses/${course}/me/**`, (r) => {
    const req = r.request();
    if (req.method() === "OPTIONS") return r.fulfill({ status: 204, headers: json({}).headers });
    const leaf = new URL(req.url()).pathname.split("/").pop() as keyof Handlers;
    calls.push(`${course}:${leaf}`);
    const fallback = { attendance: att, participation: part, "study-time": study, grades: () => ({ status: "NO_SCHEME" }), "what-if": () => ({ hp: "7", letter: "B" }) }[leaf];
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
  expect(titles).toEqual(["Điểm", "Chuyên cần", "Điểm cộng đã ghi", "Thời gian học"]); // #26: khối Điểm thật (US-P6-09)
  for (const gone of [/Điểm quá trình/, /Giải trình/, /nếu cuối kỳ/i, /Việc sắp tới/]) await expect(page.getByText(gone)).toHaveCount(0);
});

test("me no gamification (no gamify): không streak, xếp hạng, vòng tiến độ, huy hiệu, confetti, so sánh bạn học", async ({ page }) => {
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

test("me staff blocked: TA / giảng viên / admin vào /me thấy màn chặn chung (không chuyển hướng), có nút Về Hôm nay, không gọi API của sinh viên", async ({ page, context }) => {
  for (const role of ["teacher", "ta", "admin"] as const) {
    await context.clearCookies();
    await page.unrouteAll({ behavior: "ignoreErrors" });
    await asDemo(context, role);
    const calls = await api(page, COURSE);
    await page.goto("/me");
    await expect(page.getByRole("heading", { name: "Bạn không có quyền xem màn này" }), role).toBeVisible();
    await expect(page).toHaveURL(/\/me$/);
    expect(calls, role).toEqual([]);
    await page.getByRole("link", { name: "Về Hôm nay" }).click();
    await expect(page).toHaveURL(/\/$/);
  }
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

// ---- US-P6-09: khối Điểm trên /me ------------------------------------------------------------------------------------------------------------
const comp = (key: string, name: string, weight: string, score?: string) => ({ key, name, weight, status: score ? "PUBLISHED" : "PENDING", ...(score ? { score } : {}) });
const inProgress = (over: Record<string, unknown> = {}) => ({
  status: "IN_PROGRESS",
  components: [comp("midterm", "Giữa kỳ", "20", "8.25"), comp("weekly", "Kiểm tra hằng tuần", "20", "8"), comp("final", "Thi cuối kỳ", "60")],
  qt: "8.25",
  calculation: [
    { label: "Giữa kỳ (20%)", expr: "8,25", value: "8,25" },
    { label: "Điểm cộng", expr: "min(0,5; 1,0)", value: "0,5", source_quote: "Mỗi lần phát biểu được cộng 0,25 điểm.", page: 2 },
    { label: "Điểm quá trình", expr: "min(10; max(0; 8,25 + 0,5 − 0))", value: "8,25" },
  ],
  official_note: "Điểm chính thức nằm ở hệ thống quản lý đào tạo của trường.",
  ...over,
});
const finalGrade = () => ({ status: "FINAL", components: [comp("final", "Thi cuối kỳ", "60", "7")], qt: "7", final: { numeric_grade: "7.0", letter: "B", version: 1, locked_at: "2026-10-30T03:00:00Z" }, calculation: [{ label: "Điểm học phần thô", expr: "(40 × 7 + 60 × 7) ÷ 100", value: "7" }], official_note: "Điểm chính thức nằm ở hệ thống quản lý đào tạo của trường." });

test("grade section: câu nhận định theo trạng thái, thành phần đã công bố / chưa công bố, đúng một dòng ghi chú cố định", async ({ page }) => {
  await api(page, COURSE, { grades: () => inProgress() });
  await page.goto("/me");
  await expect(page.locator("[data-part=grade-sentence]")).toHaveText("Điểm quá trình hiện tại của bạn là 8,25 trên 10.");
  const sec = page.locator("main section", { hasText: "Giữa kỳ · 20%" });
  await expect(sec).toContainText("8,25");
  await expect(sec).toContainText("Chưa công bố");
  await expect(page.locator("[data-part=official-note]")).toHaveCount(1);
  await page.unrouteAll({ behavior: "ignoreErrors" });
  await asDemo(page.context(), "student", { person: "sv-2" });
  await api(page, COURSE, { grades: finalGrade });
  await page.goto("/me");
  await expect(page.locator("[data-part=grade-sentence]")).toHaveText("Điểm học phần đã chốt: 7,0 (B).");
  await expect(page.locator("[data-part=what-if]")).toHaveCount(0);
  await page.unrouteAll({ behavior: "ignoreErrors" });
  await asDemo(page.context(), "student", { person: "sv-2" });
  await api(page, COURSE, { grades: () => ({ status: "NO_SCHEME" }) });
  await page.goto("/me");
  await expect(page.getByText("Lớp chưa có công thức điểm chính thức.")).toBeVisible();
  await expect(page.locator("[data-part=official-note]")).toHaveCount(0);
});

test("calculation linear: phép tính từng dòng cùng thứ tự, Căn cứ mở đoạn quy chế tại chỗ, số dấu phẩy, không gamify", async ({ page }) => {
  await api(page, COURSE, { grades: () => inProgress() });
  await page.goto("/me");
  const calc = page.locator("[data-part=grade-calculation]");
  await expect(calc.locator("li")).toHaveCount(3);
  await expect(calc.locator("li").nth(0)).toContainText("Giữa kỳ (20%)");
  await expect(calc.locator("li").nth(1)).toContainText("min(0,5; 1,0)");
  await expect(calc.locator("li").nth(2)).toContainText("= 8,25");
  await calc.getByRole("button", { name: /Căn cứ · tr\. 2/ }).click();
  await expect(calc).toContainText("Mỗi lần phát biểu được cộng 0,25 điểm.");
  const text = (await page.locator("main").innerText()).toLowerCase();
  for (const bad of ["streak", "xếp hạng", "huy hiệu", "confetti", "so với bạn", "trung bình lớp", "tuyệt vời", "cố lên", "chúc mừng"]) expect(text, bad).not.toContain(bad);
  expect(text).not.toMatch(/\d\.\d/); // số hiển thị bằng dấu phẩy
  await expect(page.locator("main [role=progressbar], main progress")).toHaveCount(0);
});

test("what if inline: gõ số → tính tại chỗ bằng máy chủ, ngoài thang báo tại ô và không gọi, mạng lỗi giữ số đã gõ", async ({ page }) => {
  let down = false;
  const bodies: string[] = [];
  await api(page, COURSE, { grades: () => inProgress() });
  await page.route(`**/api/v1/courses/${COURSE}/me/grades/what-if`, (r) => {
    const req = r.request();
    if (req.method() === "OPTIONS") return r.fulfill({ status: 204, headers: json({}).headers });
    bodies.push(req.postData() ?? "");
    return down ? r.fulfill(err(500, "INTERNAL")) : r.fulfill(json({ hp: "8", letter: "B+", calculation: [] }));
  });
  await page.goto("/me");
  const input = page.getByLabel("Điểm cuối kỳ giả định");
  await input.fill("8,0");
  await expect(page.locator("[data-part=what-if-result]")).toHaveText("Điểm học phần sẽ là 8 (B+).");
  expect(JSON.parse(bodies.at(-1)!)).toEqual({ final_exam_score: "8.0" });
  const n = bodies.length;
  await input.fill("11");
  await expect(page.getByRole("alert").filter({ hasText: "Nhập số từ 0 đến 10" })).toBeVisible();
  await page.waitForTimeout(600);
  expect(bodies.length).toBe(n);
  down = true;
  await input.fill("9");
  await expect(page.getByRole("button", { name: "Thử lại" })).toBeVisible();
  await expect(input).toHaveValue("9");
  down = false;
  await page.getByRole("button", { name: "Thử lại" }).click();
  await expect(page.locator("[data-part=what-if-result]")).toBeVisible();
  await expect(page.getByText(/nếu cuối kỳ/i)).toHaveCount(1);
});

test("grade states: lỗi → nguyên nhân + Thử lại; chưa công bố gì → câu rỗng; mất mạng không xoá nội dung khác", async ({ page }) => {
  let broken = true;
  await api(page, COURSE, { grades: () => (broken ? null : { status: "IN_PROGRESS", components: [comp("midterm", "Giữa kỳ", "20")], official_note: "x" }) });
  await page.goto("/me");
  await expect(page.getByText("Không tải được điểm của bạn.")).toBeVisible();
  await expect(page.getByText("Bạn có mặt 7/9 buổi, vắng 2.")).toBeVisible();
  broken = false;
  await page.locator("main section", { hasText: "Không tải được điểm của bạn." }).getByRole("button", { name: "Thử lại" }).click();
  await expect(page.getByText("Chưa có điểm được công bố.")).toBeVisible();
});

test("grade mobile 375: khối điểm một cột, không tràn ngang", async ({ page }) => {
  await page.setViewportSize({ width: 375, height: 812 });
  await api(page, COURSE, { grades: () => inProgress({ calculation: [{ label: "Điểm quá trình trước điểm cộng / trừ", expr: "(20 × 8,25 + 20 × 8 + 20 × 7,5 + 20 × 9) ÷ 80", value: "8,19" }] }) });
  await page.goto("/me");
  await expect(page.locator("[data-part=grade-sentence]")).toBeVisible();
  expect(await page.evaluate(() => document.documentElement.scrollWidth - window.innerWidth)).toBeLessThanOrEqual(0);
});

test("assignments section: mỗi bài một dòng tên · hạn · trạng thái, mở sang trang bài tập (US-P7-03 AC10)", async ({ page }) => {
  await api(page, COURSE);
  await studentApi(page, {
    items: [
      studentItem(),
      studentItem({ id: A2, title: "Bài tập 04 · Mã độc", my: { state: "SUBMITTED", late_days: 0, submitted_at: "2026-10-23T01:10:00.000Z" } }),
    ],
  });
  await page.goto("/me");
  const rows = page.locator("[data-part=me-assignment-row]");
  await expect(rows).toHaveCount(2);
  await expect(rows.nth(0)).toContainText("Bài tập 03 · Phân tích nhật ký");
  await expect(rows.nth(0)).toContainText("Hạn Thứ Sáu, 20/11 · 14:00");
  await expect(rows.nth(0)).toContainText("Chưa nộp");
  await expect(rows.nth(1)).toContainText("Đã nộp");
  await expect(rows.nth(0).getByRole("link")).toHaveAttribute("href", `/assignments/${A1}?course=${COURSE}`); // mang lớp theo để màn đích gọi API ngay (TL-1)
  const titles = await page.locator("main section h2, main [data-ep-panel] h2").allInnerTexts();
  expect(titles).toEqual(["Điểm", "Chuyên cần", "Điểm cộng đã ghi", "Bài tập", "Thời gian học"]);
});
