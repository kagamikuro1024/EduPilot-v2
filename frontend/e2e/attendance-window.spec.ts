import { expect, test, type BrowserContext, type Page } from "@playwright/test";
import { NOW, attApi, err, gridBody, json, sessionRow, sid, student, students, COURSE } from "./support/att-fixtures";
import { asDemo } from "./support/session";

// US-AS-04…07, 11: điểm danh bằng mã trên /attendance (DESIGN §14.8 «Self check-in on this route»). Gateway giả theo hợp đồng thật.
const SESSION = sessionRow(2).id;
const WID = "00000000-0000-7000-8000-0000000007b1";
const BASE = `**/api/v1/courses/${COURSE}`;
const policy = { self_checkin: true, rotate_seconds: 10, late_after_min: 15, require: "WEB", geo_check: false };
const win = (over: Record<string, unknown> = {}) => ({ id: WID, round: 1, opened_at: new Date(NOW - 60_000).toISOString(), closes_at: new Date(NOW + 600_000).toISOString(), closed_at: null, headcount: null, checked_in: 2, over_headcount: false, ...over });
const mark = (i: number, status: string) => student(i, { record: { status, note: null, version: 1, marked_by: { full_name: "Hệ thống" }, marked_at: new Date(NOW).toISOString() } });
const crow = (i: number, over: Record<string, unknown> = {}) => ({ id: `00000000-0000-7000-8000-0000000008${String(i).padStart(2, "0")}`, window_id: WID, round: 1, student: { id: sid(i), full_name: `Sinh Viên ${String(i).padStart(2, "0")}`, student_code: `B22DCCN${100 + i}` }, created_at: "2026-10-07T00:03:00.000Z", channel: "WEB", assurance: 3, flags: [], cleared: false, ...over });

type Calls = { absent: number; clear: string[]; opened: number; puts: string[]; patch: string[] };
async function setup(page: Page, context: BrowserContext, o: { windows?: unknown[]; rows?: unknown[]; grid?: unknown; put?: (r: import("@playwright/test").Route) => Promise<void> | void } = {}) {
  await asDemo(context, "teacher");
  await page.clock.setFixedTime(NOW);
  const calls: Calls = { absent: 0, clear: [], opened: 0, puts: [], patch: [] };
  await attApi(page, { grid: o.grid, h: o.put ? { "PUT attendance": (r) => o.put!(r) } : undefined });
  await page.route(`${BASE}`, (r) => r.fulfill(json({ id: COURSE, class_code: "761988", settings: { escalation_threshold: "0.80", version: 1 }, attendance: policy })));
  await page.route(`${BASE}/sessions/${SESSION}/checkin-windows`, (r) => {
    if (r.request().method() === "POST") {
      calls.opened++;
      return r.fulfill(json({ id: WID, round: 1, closes_at: new Date(NOW + 600_000).toISOString(), rotate_seconds: 10 }, 201));
    }
    return r.fulfill(json({ items: o.windows ?? [] }));
  });
  await page.route(`${BASE}/sessions/${SESSION}/checkins**`, (r) => r.fulfill(json({ items: o.rows ?? [], next_cursor: null })));
  await page.route(`${BASE}/sessions/${SESSION}/absent-unchecked`, (r) => {
    calls.absent++;
    return r.fulfill(json({ marked: 28 }));
  });
  await page.route(`${BASE}/checkins/*/clear-flags`, (r) => {
    calls.clear.push(new URL(r.request().url()).pathname.split("/").at(-2)!);
    return r.fulfill(json(crow(1, { cleared: true, flags: [] })));
  });
  await page.route(`${BASE}/checkin-windows/${WID}`, (r) => {
    calls.patch.push(r.request().postData() ?? "");
    return r.fulfill(json(win({ headcount: 31 })));
  });
  await page.route(`**/api/v1/courses/${COURSE}/sessions/${SESSION}/checkin-windows/${WID}/display`, (r) => r.fulfill(err(404, "NOT_FOUND")));
  await page.goto("/attendance");
  await expect(page.locator("[data-part=student-row]")).not.toHaveCount(0);
  return calls;
}
const row = (page: Page, i: number) => page.locator(`[data-part=student-row][data-student-id="${sid(i)}"]`);
const desktopOnly = (info: { project: { name: string } }) => test.skip(info.project.name !== "desktop", "đo một lần");

test("attendance checkin: lớp chưa mở cửa sổ → «Mở điểm danh» tạo cửa sổ rồi sang màn chiếu; lưới vẫn «Có mặt» mặc định", async ({ page, context }, info) => {
  desktopOnly(info);
  const calls = await setup(page, context);
  await expect(page.locator("[data-part=window-line]")).toHaveCount(0);
  await expect(row(page, 1).getByRole("radio", { name: "Có mặt — Sinh Viên 01", exact: true })).toBeChecked();
  await page.getByRole("button", { name: "Mở điểm danh" }).click();
  await expect(page).toHaveURL(new RegExp(`/attendance/display\\?course=${COURSE}&session=${SESSION}$`));
  expect(calls.opened).toBe(1);
});

test("attendance checkin: cửa sổ đang mở — dòng «Đang mở · k/n · còn», ô trống + «Chưa điểm danh», «Điểm danh lúc», chip cờ ≤ 2 rồi +n, «Đã xem», lọc «Cần xem»", async ({ page, context }, info) => {
  desktopOnly(info);
  const list = [mark(1, "PRESENT"), mark(2, "PRESENT"), ...students(30).slice(2).map((s, i) => student(i + 3))];
  const flagged = [
    crow(1, { flags: [{ code: "DEVICE_SHARED", label: "Máy dùng chung" }, { code: "GEO_OUT", label: "Ngoài vùng" }, { code: "TOKEN_LATE", label: "Mã cũ" }], channel: "CODE", assurance: 1 }),
    crow(2),
    crow(3, { channel: "CODE", assurance: 1 }),
  ];
  const calls = await setup(page, context, { windows: [win()], rows: flagged, grid: gridBody(2, list) });
  const line = page.locator("[data-part=window-line]");
  await expect(line).toContainText("Đang mở");
  await expect(page.locator("[data-part=window-count]")).toHaveText("· 3/30");
  await expect(line).toContainText("còn 10:00");
  await expect(line.getByRole("link", { name: "Mở màn chiếu" })).toHaveAttribute("href", `/attendance/display?course=${COURSE}&session=${SESSION}`);
  // hàng chưa điểm danh: ô trống, không mặc định Có mặt
  await expect(row(page, 5)).toContainText("Chưa điểm danh");
  for (const label of ["Có mặt", "Muộn", "Vắng phép", "Vắng"]) await expect(row(page, 5).getByRole("radio", { name: `${label} — Sinh Viên 05`, exact: true })).not.toBeChecked();
  // hàng đã quét: giờ + chip
  const r1 = row(page, 1).locator("[data-part=checkin-meta]");
  await expect(r1).toContainText("Điểm danh lúc 07:03");
  await expect(r1).toContainText("Máy dùng chung");
  await expect(r1).toContainText("Ngoài vùng");
  await expect(r1).not.toContainText("Mã cũ"); // chỉ 2 chip rồi +n
  await expect(r1).toContainText("+3");
  const r3 = row(page, 3).locator("[data-part=checkin-meta]");
  await expect(r3).toContainText("Chưa xác minh · Nhập mã");
  await expect(r3).not.toContainText("Đã xem"); // không cờ ⇒ không có «Đã xem»
  await expect(row(page, 2).locator("[data-part=checkin-meta]")).not.toContainText("Đã xem");
  // lọc
  await page.getByRole("button", { name: "Cần xem (1)" }).click();
  await expect(page).toHaveURL(/filter=review/);
  await expect(page.locator("[data-part=student-row]")).toHaveCount(1);
  await r1.getByRole("button", { name: "Đã xem" }).click();
  await expect.poll(() => calls.clear.length).toBe(1);
});

// DEF-R10.8-03 (TC-AS08-20, luật 14): «Đóng» lỗi rồi tự thử lại là CÙNG một thao tác ⇒ cùng Idempotency-Key tới khi thành công.
test("attendance checkin: «Đóng» lỗi 503 rồi thử lại — mọi lượt POST …/close dùng MỘT Idempotency-Key; xong thì «Đã ghi»", async ({ page, context }, info) => {
  desktopOnly(info);
  const list = [mark(1, "PRESENT"), mark(2, "PRESENT"), ...students(30).slice(2).map((s, i) => student(i + 3))];
  const keys: string[] = [];
  await page.route(`${BASE}/checkin-windows/${WID}/close`, (r) => {
    keys.push(r.request().headers()["idempotency-key"] ?? "");
    if (keys.length < 3) return r.fulfill(err(503, "NOT_READY"));
    return r.fulfill(json({ window: win({ closed_at: new Date(NOW).toISOString() }), written: { present: 2, late: 0 }, unchecked: 28, skipped_existing: 0 }));
  });
  await setup(page, context, { windows: [win()], rows: [crow(1)], grid: gridBody(2, list) });
  await page.getByRole("button", { name: "Đóng", exact: true }).click();
  await expect(page.locator("[data-part=close-queued]")).toContainText("Chưa đóng");
  await page.evaluate(() => window.dispatchEvent(new Event("online"))); // lần tự thử lại thứ nhất (vẫn 503)
  await expect.poll(() => keys.length).toBe(2);
  await expect(page.locator("[data-part=close-queued]")).toContainText("Chưa đóng");
  await page.evaluate(() => window.dispatchEvent(new Event("online"))); // lần thứ hai: 200
  await expect.poll(() => keys.length).toBe(3);
  expect(keys.every((k) => k.length >= 8)).toBe(true);
  expect(new Set(keys).size, `khoá: ${keys.join(", ")}`).toBe(1);
});

test("attendance checkin: cửa sổ đã đóng — «Đã ghi a có mặt · b muộn · u chưa điểm danh», số đầu người, «Đánh vắng u SV» qua xác nhận", async ({ page, context }, info) => {
  desktopOnly(info);
  const list = [mark(1, "PRESENT"), mark(2, "LATE"), ...students(30).slice(2).map((s, i) => student(i + 3))];
  const closed = win({ closed_at: new Date(NOW - 1000).toISOString(), headcount: 31, over_headcount: true });
  const calls = await setup(page, context, { windows: [closed], rows: [crow(1), crow(2)], grid: gridBody(2, list) });
  const line = page.locator("[data-part=window-line]");
  await expect(page.locator("[data-part=window-summary]")).toHaveText("Đã ghi 1 có mặt · 1 muộn · 28 chưa điểm danh");
  await expect(line).toContainText("31 người trong lớp");
  // một việc một nút (§23): sau khi đóng không còn «Mở điểm danh» ở đầu trang (vòng kế tiếp nằm trong menu ⋯), «Đánh vắng u SV» chỉ ở một chỗ
  await expect(page.getByRole("button", { name: "Mở điểm danh" })).toHaveCount(0);
  await expect(page.getByRole("button", { name: /^Đánh vắng \d+ SV/ })).toHaveCount(1);
  await expect(page.locator("[data-part=over-headcount]")).toHaveText("Số điểm danh nhiều hơn số người đếm");
  await line.getByRole("button", { name: "Đánh vắng 28 SV chưa điểm danh" }).click();
  await expect(page.getByText("Đánh vắng 28 sinh viên?")).toBeVisible();
  await expect(page.getByText("28 sinh viên chưa điểm danh thành Vắng.")).toBeVisible();
  await page.getByRole("button", { name: "Đánh vắng", exact: true }).click();
  await expect.poll(() => calls.absent).toBe(1);
});

test("attendance checkin: Lưu điểm danh bị 422 CHECKIN_UNRESOLVED → cảnh báo vàng đầu panel + nút đánh vắng; không gửi hàng ô trống; ô đã sửa còn nguyên", async ({ page, context }, info) => {
  desktopOnly(info);
  const list = [mark(1, "PRESENT"), ...students(30).slice(1).map((s, i) => student(i + 2))];
  let body = "";
  await setup(page, context, {
    windows: [win({ closed_at: new Date(NOW - 1000).toISOString() })],
    rows: [crow(1)],
    grid: gridBody(2, list),
    put: (r) => {
      body = r.request().postData() ?? "";
      return r.fulfill(json({ code: "CHECKIN_UNRESOLVED", message: "m", trace_id: "b".repeat(32), details: { count: 29 } }, 422));
    },
  });
  await page.getByLabel(/Bảng điểm danh/).focus();
  await row(page, 1).getByRole("radio", { name: "Muộn — Sinh Viên 01", exact: true }).check();
  await page.getByRole("button", { name: /Lưu điểm danh/ }).first().click();
  await expect(page.getByText("Còn 29 sinh viên chưa điểm danh", { exact: true })).toBeVisible();
  await expect(page.getByRole("button", { name: "Đánh vắng 29 SV chưa điểm danh" })).toHaveCount(1); // nút ở cảnh báo, dòng cửa sổ không lặp lại
  const records = (JSON.parse(body) as { records: Array<{ student_id: string }> }).records;
  expect(records.map((x) => x.student_id)).toEqual([sid(1)]); // 29 ô trống không gửi
  await expect(row(page, 1).getByRole("radio", { name: "Muộn — Sinh Viên 01", exact: true })).toBeChecked();
});

// D-AS-05: ở 375 cảnh báo «Còn N sinh viên chưa điểm danh» không bị nút ép xuống từng chữ: chữ tối đa 2 dòng, nút nằm DƯỚI chữ và dàn hết bề rộng khối.
test("attendance checkin 375: cảnh báo CHECKIN_UNRESOLVED — chữ ≤ 2 dòng, nút Đánh vắng xếp dưới, full-width, không tràn ngang", async ({ page, context }, info) => {
  test.skip(info.project.name !== "mobile", "đo ở di động");
  await setup(page, context, {
    windows: [win({ closed_at: new Date(NOW - 1000).toISOString() })],
    rows: [crow(1)],
    grid: gridBody(2, [mark(1, "PRESENT"), ...students(30).slice(1).map((s, i) => student(i + 2))]),
    put: (r) => r.fulfill(json({ code: "CHECKIN_UNRESOLVED", message: "m", trace_id: "b".repeat(32), details: { count: 29 } }, 422)),
  });
  await page.getByRole("button", { name: /Lưu điểm danh/ }).first().click();
  const text = page.getByText("Còn 29 sinh viên chưa điểm danh", { exact: true });
  await expect(text).toBeVisible();
  const btn = page.getByRole("button", { name: "Đánh vắng 29 SV chưa điểm danh" }).first();
  const [t, b, n] = [await text.boundingBox(), await btn.boundingBox(), await page.getByRole("status").filter({ has: text }).boundingBox()];
  expect(t!.height).toBeLessThanOrEqual(2 * 22);
  expect(b!.y).toBeGreaterThanOrEqual(t!.y + t!.height);
  expect(b!.width).toBeGreaterThanOrEqual(n!.width * 0.8);
  expect(await page.evaluate(() => document.documentElement.scrollWidth <= window.innerWidth)).toBe(true);
});
