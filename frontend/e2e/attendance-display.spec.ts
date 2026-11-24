import AxeBuilder from "@axe-core/playwright";
import { expect, test, type Page } from "@playwright/test";
import { COURSE, err, json } from "./support/att-fixtures";
import { asDemo } from "./support/session";

// US-AS-04: màn chiếu điểm danh (DESIGN §14.8.1). Gateway giả theo hợp đồng thật; `rotate_seconds` ngắn để vòng đổi mã chạy trong vài giây.
const SESSION = "00000000-0000-7000-8000-0000000005a2";
const WID = "00000000-0000-7000-8000-0000000007b1";
const BASE = `**/api/v1/courses/${COURSE}`;
const URL_DISPLAY = `/attendance/display?course=${COURSE}&session=${SESSION}`;

const win = (over: Record<string, unknown> = {}) => ({ id: WID, round: 1, opened_at: new Date().toISOString(), closes_at: new Date(Date.now() + 600_000).toISOString(), closed_at: null, headcount: 40, checked_in: 3, over_headcount: false, ...over });
const disp = (n: number, stepMs: number, rotate = 1, over: Record<string, unknown> = {}) => ({
  qr_url: `https://app.test/checkin?c=${COURSE}#tok${n}`,
  short_code: n === 1 ? "ABC123" : "DEF456",
  step_expires_at: new Date(Date.now() + stepMs).toISOString(),
  server_time: new Date().toISOString(),
  rotate_seconds: rotate,
  round: 1,
  checked_in: 3,
  headcount: 40,
  closes_at: new Date(Date.now() + 600_000).toISOString(),
  ...over,
});

async function mock(page: Page, display: (n: number) => ReturnType<typeof json>) {
  const log = { display: 0, close: 0, windows: 0 };
  await page.route(`${BASE}/sessions/${SESSION}/checkin-windows`, (r) => { log.windows++; return r.fulfill(json({ items: [win()] })); });
  await page.route(`${BASE}/sessions/${SESSION}/checkin-windows/${WID}/display`, (r) => r.fulfill(display(++log.display)));
  await page.route(`${BASE}/checkin-windows/${WID}/close`, (r) => { log.close++; return r.fulfill(json({ window: win({ closed_at: new Date().toISOString(), checked_in: 7 }), written: { present: 6, late: 1 }, unchecked: 33, skipped_existing: 0 })); });
  return log;
}

test("attendance display: QR + mã ngắn «ABC 123» + đếm; đổi mã đúng một lần mỗi bước; nút Đóng gọi API và hiện «Đã đóng» với số cuối", async ({ page, context }, info) => {
  test.skip(info.project.name !== "desktop", "đo một lần");
  await asDemo(context, "teacher");
  const log = await mock(page, (n) => json(disp(n, n === 1 ? 1200 : 5000)));
  await page.goto(URL_DISPLAY);
  await expect(page.getByRole("heading", { name: "Màn chiếu điểm danh", level: 1 })).toBeHidden();
  await expect(page.getByRole("img", { name: "Mã QR điểm danh" })).toBeVisible();
  await expect(page.locator("[data-part=short-code]")).toHaveText("ABC 123");
  await expect(page.locator("[data-part=count]")).toHaveText("3 / 40");
  await expect(page.locator("[data-part=short-code]")).toHaveText("DEF 456", { timeout: 5000 }); // bước kế
  expect(log.display).toBe(2); // một lần mỗi bước, không hỏi dồn
  await page.getByRole("button", { name: "Đóng điểm danh" }).click();
  await expect(page.locator("[data-part=closed]")).toHaveText("Đã đóng");
  await expect(page.locator("[data-part=count]")).toHaveText("7 / 40");
  expect(log.close).toBe(1);
});

test("attendance display: mất mạng giữ mã cuối tới hết hạn rồi ô trống «Mất kết nối»; 409 WINDOW_CLOSED → «Đã đóng»", async ({ page, context }, info) => {
  test.skip(info.project.name !== "desktop", "đo một lần");
  await asDemo(context, "teacher");
  await mock(page, (n) => (n === 1 ? json(disp(1, 400, 1)) : n < 8 ? err(503, "UNAVAILABLE") : err(409, "WINDOW_CLOSED")));
  await page.goto(URL_DISPLAY);
  await expect(page.getByRole("img", { name: "Mã QR điểm danh" })).toBeVisible();
  await expect(page.locator("[data-part=offline]")).toHaveText("Mất kết nối", { timeout: 6000 });
  await expect(page.getByRole("img", { name: "Mã QR điểm danh" })).toHaveCount(0);
  await expect(page.locator("[data-part=closed]")).toHaveText("Đã đóng", { timeout: 20000 });
});

test("attendance display: Esc thoát về /attendance; sinh viên không gây yêu cầu /api/v1 nào", async ({ page, context }, info) => {
  test.skip(info.project.name !== "desktop", "đo một lần");
  await asDemo(context, "teacher");
  await page.route("**/api/v1/courses/**/sessions**", (r) => r.fulfill(json({ items: [] }))); // màn /attendance sau khi thoát
  await mock(page, (n) => json(disp(n, 20000, 10)));
  await page.goto(URL_DISPLAY);
  await expect(page.locator("[data-part=short-code]")).toBeVisible();
  await page.keyboard.press("Escape");
  await expect(page).toHaveURL(new RegExp(`/attendance\\?session=${SESSION}$`));

  await page.context().clearCookies();
  await asDemo(context, "student");
  const calls: string[] = [];
  page.on("request", (r) => r.url().includes("/api/v1/") && calls.push(new URL(r.url()).pathname));
  await page.goto(URL_DISPLAY);
  await expect(page.getByText("Bạn không có quyền xem màn này")).toBeVisible();
  expect(calls.filter((p) => p.includes("/checkin-windows"))).toEqual([]);
});

const OTHER = "00000000-0000-7000-8000-00000000c002";
const item = (id: string, code: string) => ({ course: { id, class_code: code, subject_code: "INT1006", name: "An ninh mạng", semester: "2026-2027-HK1", status: "ACTIVE" }, role_in_course: "TEACHER", enrollment_status: "ACTIVE" });

test("attendance display: lớp lấy từ URL, không từ lớp đang chọn; lớp ngoài tài khoản → lỗi rõ, 0 yêu cầu cửa sổ", async ({ page, context }, info) => {
  test.skip(info.project.name !== "desktop", "đo một lần");
  await asDemo(context, "teacher");
  // lớp đang chọn của tab = OTHER (đứng đầu danh sách); màn chiếu mở cho COURSE
  await page.route("**/api/v1/me/courses**", (r) => r.fulfill(json({ items: [item(OTHER, "761999"), item(COURSE, "761988")], next_cursor: null })));
  const asked: string[] = [];
  page.on("request", (r) => r.url().includes("/checkin-windows") && asked.push(new URL(r.url()).pathname));
  await mock(page, (n) => json(disp(n, 20000, 10)));
  await page.goto(URL_DISPLAY);
  await expect(page.locator("[data-part=short-code]")).toHaveText("ABC 123");
  await expect(page).toHaveURL(new RegExp(`course=${COURSE}`)); // tham số không bị bỏ
  expect(asked.every((p) => p.includes(`/courses/${COURSE}/`))).toBe(true);

  const before = asked.length;
  await page.goto(`/attendance/display?course=00000000-0000-7000-8000-00000000c009&session=${SESSION}`);
  await expect(page.getByText("Lớp này không có trong tài khoản của bạn")).toBeVisible();
  await expect(page.locator("[data-part=short-code]")).toHaveCount(0);
  expect(asked.length).toBe(before);
});

test("attendance display: mã ngắn nằm trọn trong khung ở 1024×768, 1366×768, 1440×900, 1920×1080, 375×667 (DEF-GP10-02)", async ({ page, context }, info) => {
  test.skip(info.project.name !== "desktop", "đo một lần");
  await asDemo(context, "teacher");
  await mock(page, (n) => json(disp(n, 20000, 10, { short_code: "WWM8WW" })));
  for (const [w, h] of [[1024, 768], [1366, 768], [1440, 900], [1920, 1080], [375, 667]] as const) {
    await page.setViewportSize({ width: w, height: h });
    await page.goto(URL_DISPLAY);
    const code = page.locator("[data-part=short-code]");
    await expect(code).toBeVisible();
    const [c, q] = [await code.boundingBox(), await page.getByRole("img", { name: "Mã QR điểm danh" }).boundingBox()];
    expect(c!.y, `${w}×${h}: mã NGAY DƯỚI QR (US-AS-08 AC2)`).toBeGreaterThanOrEqual(q!.y + q!.height);
    expect(c!.x, `${w}×${h}: mã nằm trong bề ngang QR`).toBeGreaterThanOrEqual(q!.x - 1);
    expect(c!.x + c!.width, `${w}×${h}: mã nằm trong bề ngang QR`).toBeLessThanOrEqual(q!.x + q!.width + 1);
    expect(await page.evaluate(() => document.documentElement.scrollHeight <= innerHeight + 1), `${w}×${h}: không cuộn dọc`).toBe(true);
    expect(await code.evaluate((e) => e.scrollWidth <= e.clientWidth + 1 && e.getBoundingClientRect().right <= innerWidth), `${w}×${h}: mã trọn trong khung`).toBe(true);
    expect(await page.evaluate(() => document.documentElement.scrollWidth <= innerWidth + 1)).toBe(true);
  }
});

test("attendance display: axe 0 lỗi nghiêm trọng ở 1280×720 và 1920×1080 (mã đang hiện, và «Đã đóng»); QR vuông, vừa khung, không cuộn", async ({ page, context }, info) => {
  test.skip(info.project.name !== "desktop", "đo một lần");
  await asDemo(context, "teacher");
  await mock(page, (n) => json(disp(n, 20000, 10)));
  for (const [w, h] of [[1280, 720], [1920, 1080]] as const) {
    await page.setViewportSize({ width: w, height: h });
    await page.goto(URL_DISPLAY);
    await expect(page.getByRole("img", { name: "Mã QR điểm danh" })).toBeVisible();
    const box = await page.getByRole("img", { name: "Mã QR điểm danh" }).boundingBox();
    expect(Math.abs(box!.width - box!.height)).toBeLessThan(2); // vuông
    expect(box!.height).toBeLessThanOrEqual(h);
    expect(await page.evaluate(() => document.documentElement.scrollHeight <= innerHeight + 1 && document.documentElement.scrollWidth <= innerWidth + 1)).toBe(true);
    const res = await new AxeBuilder({ page }).withTags(["wcag2a", "wcag2aa", "wcag21aa", "wcag22aa"]).analyze();
    expect(res.violations.filter((v) => v.impact === "critical" || v.impact === "serious").map((v) => v.id)).toEqual([]);
    await page.getByRole("button", { name: "Đóng điểm danh" }).click();
    await expect(page.locator("[data-part=closed]")).toHaveText("Đã đóng");
    const res2 = await new AxeBuilder({ page }).withTags(["wcag2a", "wcag2aa", "wcag21aa", "wcag22aa"]).analyze();
    expect(res2.violations.filter((v) => v.impact === "critical" || v.impact === "serious").map((v) => v.id)).toEqual([]);
  }
});
