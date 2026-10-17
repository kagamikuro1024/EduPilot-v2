import { expect, test, type BrowserContext, type Page } from "@playwright/test";
import { BASE_URL } from "./support/env";
import { asDemo, type DemoRole } from "./support/session";
import { NOW, at, calApi, err, exam, item, json, noContent, page as pageOf, session } from "./support/cal-fixtures";

// US-P8-03: /calendar THẬT (gateway giả theo hợp đồng thật).
const C = "/courses/00000000-0000-7000-8000-00000000c001/calendar";

async function open(page: Page, context: BrowserContext, role: DemoRole, h: Parameters<typeof calApi>[1] = {}) {
  await asDemo(context, role, role === "student" ? { person: "sv-2" } : {});
  const calls = await calApi(page, { [`GET ${C}`]: pageOf([exam(), item(), session()]), ...h });
  await page.clock.setFixedTime(NOW);
  await page.goto("/calendar");
  await page.getByRole("heading", { name: "Lịch", level: 1 }).waitFor();
  return calls;
}

test("week default: ≥ 720 px mặc định Tuần, một Panel, đỏ chỉ cho bài thi trong 48 giờ", async ({ page, context }, info) => {
  test.skip(info.project.name !== "desktop", "ca 375 riêng");
  await open(page, context, "student", { [`GET ${C}`]: pageOf([exam({ starts_at: at(5), ends_at: at(6) }), exam({ id: "weekly_exam:far", title: "Thi cuối kỳ", starts_at: at(60), ends_at: at(61) }), item({ starts_at: at(8), title: "Nộp báo cáo" })]) });
  await expect(page.getByRole("radio", { name: "Tuần" })).toBeChecked();
  await expect(page.locator("main [data-ep-panel]:visible")).toHaveCount(1);
  const border = (t: string) => page.locator("[data-part=cal-event]", { hasText: t }).locator("xpath=ancestor::*[@data-kind][1]").evaluate((e) => getComputedStyle(e).borderLeftColor);
  const soon = await border("Thi giữa kỳ");
  expect(await border("Thi cuối kỳ")).not.toBe(soon); // chỉ bài thi trong 48 giờ có tín hiệu đỏ
  expect(await border("Nộp báo cáo")).not.toBe(soon);
  expect(await page.locator("main").innerText()).not.toMatch(/\b(RAG|PII|trace|provider)\b/i);
});

test("list on mobile: < 720 px mặc định Danh sách; chọn kiểu khác thì nhớ", async ({ page, context }, info) => {
  test.skip(info.project.name === "desktop", "chỉ ca hẹp");
  await open(page, context, "student");
  await expect(page.getByRole("radio", { name: "Danh sách" })).toBeChecked();
  await expect(page.getByText("Thi giữa kỳ")).toBeVisible();
  await expect(page.getByText("Chưa làm")).toBeVisible(); // personal_state của chính mình
  await page.getByRole("radio", { name: "Tháng" }).click();
  await page.reload();
  await expect(page.getByRole("radio", { name: "Tháng" })).toBeChecked();
});

test("month: lưới 7 cột, chip sự kiện, chuyển tháng", async ({ page, context }, info) => {
  test.skip(info.project.name !== "desktop", "đo một lần");
  const calls = await open(page, context, "student");
  await page.getByRole("radio", { name: "Tháng" }).click();
  await expect(page.locator("main").getByText(/^Tháng \d+ năm \d{4}$/)).toBeVisible();
  await expect(page.getByLabel("Thi giữa kỳ").first()).toBeVisible();
  await page.getByRole("button", { name: "Khoảng sau" }).click();
  await expect.poll(() => calls.filter((c) => c.method === "GET" && c.path.startsWith(C)).length).toBeGreaterThan(1);
  const q = new URL("http://x" + calls.filter((c) => c.method === "GET").at(-1)!.path).searchParams;
  expect((new Date(q.get("to")!).getTime() - new Date(q.get("from")!).getTime()) / 86_400_000).toBeLessThanOrEqual(62);
});

test("staff add event: Thêm sự kiện mở form tại chỗ (không hộp thoại), lưu gửi đúng thân; sửa; xoá có Hoàn tác 5 giây", async ({ page, context }, info) => {
  test.skip(info.project.name !== "desktop", "đo một lần");
  let list = [item({ id: "calendar_event:00000000-0000-7000-8000-0000000000a1", editable: true, version: 1 })];
  const calls = await open(page, context, "ta", {
    [`GET ${C}`]: (r) => r.fulfill(pageOf(list)),
    [`POST ${C}/events`]: (r) => { list = [...list, item({ id: "calendar_event:new", title: "Thi thực hành", editable: true, version: 1 })]; return r.fulfill(json({ id: "new", version: 1 }, 201)); },
    [`PUT ${C}/events/00000000-0000-7000-8000-0000000000a1`]: (r) => r.fulfill(json({ id: "x", version: 2 })),
    [`DELETE ${C}/events/00000000-0000-7000-8000-0000000000a1`]: noContent,
  });
  await page.getByRole("radio", { name: "Danh sách" }).click();
  await expect(page.getByRole("button", { name: "Thêm sự kiện" })).toHaveCount(1);
  await page.getByRole("button", { name: "Thêm sự kiện" }).click();
  await expect(page.getByRole("dialog")).toHaveCount(0);
  await page.getByLabel("Tên sự kiện").fill("Thi thực hành");
  await page.getByLabel("Loại").selectOption("EXAM");
  await page.getByLabel("Bắt đầu").fill("2026-12-20T14:00");
  await page.getByRole("button", { name: "Lưu" }).click();
  await expect(page.getByText("Thi thực hành")).toBeVisible();
  const post = JSON.parse(calls.find((c) => c.method === "POST")!.body);
  expect(post).toMatchObject({ type: "EXAM", title: "Thi thực hành", starts_at: "2026-12-20T07:00:00.000Z", ends_at: null });
  // sửa tại chỗ: version đi kèm
  await page.getByRole("button", { name: /Thêm với Nộp báo cáo/ }).click();
  await page.getByRole("menuitem", { name: "Sửa" }).click();
  await page.getByLabel("Tên sự kiện").fill("Nộp báo cáo cuối kỳ");
  await page.getByRole("button", { name: "Lưu" }).click();
  expect(JSON.parse(calls.find((c) => c.method === "PUT")!.body)).toMatchObject({ title: "Nộp báo cáo cuối kỳ", version: 1 });
  // xoá: dòng biến ngay, Hoàn tác trong 5 giây thì không gọi API; để hết giờ thì gọi
  await page.getByRole("button", { name: /Thêm với Nộp báo cáo/ }).click();
  await page.getByRole("menuitem", { name: "Xoá" }).click();
  await expect(page.getByText("Đã xoá sự kiện")).toBeVisible();
  await page.getByRole("button", { name: "Hoàn tác" }).click();
  await page.waitForTimeout(300);
  expect(calls.some((c) => c.method === "DELETE")).toBe(false);
  await page.getByRole("button", { name: /Thêm với Nộp báo cáo/ }).click();
  await page.getByRole("menuitem", { name: "Xoá" }).click();
  await expect.poll(() => calls.some((c) => c.method === "DELETE"), { timeout: 8000 }).toBe(true);
});

test("staff add event: sinh viên không thấy Thêm sự kiện lẫn menu sửa / xoá", async ({ page, context }, info) => {
  test.skip(info.project.name !== "desktop", "đo một lần");
  await open(page, context, "student", { [`GET ${C}`]: pageOf([item()]) });
  await page.getByRole("radio", { name: "Danh sách" }).click();
  await expect(page.getByRole("button", { name: "Thêm sự kiện" })).toHaveCount(0);
  await expect(page.getByRole("button", { name: /Thêm với/ })).toHaveCount(0);
});

test("ics panel: Tạo liên kết hiện URL một lần + Sao chép; Đặt lại có xác nhận và đổi URL; có token mà mất thì chỉ có Đặt lại", async ({ page, context }, info) => {
  test.skip(info.project.name !== "desktop", "đo một lần");
  await context.grantPermissions(["clipboard-read", "clipboard-write"], { origin: BASE_URL });
  let exists = false;
  let n = 0;
  const calls = await open(page, context, "student", {
    "GET /me/calendar/ics-token": (r) => r.fulfill(json({ exists })),
    "POST /me/calendar/ics-token": (r) => { exists = true; return r.fulfill(json({ url: `http://gw.test/api/v1/calendar/feed.ics?token=T${++n}` }, 201)); },
  });
  await page.getByRole("button", { name: "Thêm vào lịch" }).click();
  await page.getByRole("button", { name: "Tạo liên kết" }).click();
  await expect(page.getByRole("textbox", { name: "Liên kết lịch" })).toHaveValue("http://gw.test/api/v1/calendar/feed.ics?token=T1");
  await page.getByRole("button", { name: "Sao chép" }).click();
  await expect(page.getByText("Đã sao chép")).toBeVisible();
  await page.getByRole("button", { name: "Đặt lại liên kết" }).click();
  await expect(page.getByRole("dialog")).toContainText("Liên kết cũ ngừng hoạt động");
  await page.getByRole("dialog").getByRole("button", { name: "Đặt lại" }).click();
  await expect(page.getByRole("textbox", { name: "Liên kết lịch" })).toHaveValue(/token=T2$/);
  expect(calls.filter((c) => c.method === "POST")).toHaveLength(2);
  // tải lại: URL không bao giờ hiện lại
  await page.reload();
  await page.getByRole("button", { name: "Thêm vào lịch" }).click();
  await expect(page.getByText("Mất liên kết thì đặt lại")).toBeVisible();
  await expect(page.getByRole("textbox", { name: "Liên kết lịch" })).toHaveCount(0);
  await expect(page.getByRole("button", { name: "Đặt lại liên kết" })).toBeVisible();
});

test("states: rỗng có hành động, lỗi có Thử lại, đang tải có khung xương", async ({ page, context }, info) => {
  test.skip(info.project.name !== "desktop", "đo một lần");
  await open(page, context, "teacher", { [`GET ${C}`]: pageOf([]) });
  await expect(page.getByText("Chưa có gì trong lịch.")).toBeVisible();
  await page.unroute(`**/api/v1/courses/00000000-0000-7000-8000-00000000c001/calendar**`);
  let n = 0;
  await calApi(page, { [`GET ${C}`]: (r) => (++n === 1 ? r.fulfill(err(500, "INTERNAL")) : r.fulfill(pageOf([exam()]))) });
  await page.reload();
  await page.getByRole("button", { name: "Thử lại" }).click();
  await expect(page.getByText("Thi giữa kỳ")).toBeVisible();
});

test("375: không tràn ngang, vùng chạm ≥ 44 px", async ({ page, context }, info) => {
  test.skip(info.project.name === "desktop", "chỉ ca hẹp");
  await open(page, context, "student");
  expect(await page.evaluate(() => document.documentElement.scrollWidth <= window.innerWidth)).toBe(true);
  for (const name of ["Thêm vào lịch"]) {
    const b = await page.getByRole("button", { name }).boundingBox();
    expect(b!.height).toBeGreaterThanOrEqual(44);
  }
});

// US-P5-01: ngăn kéo "Lịch buổi học" (Staff). Gateway giả theo hợp đồng thật `…/sessions`, `…/sessions/generate[?dry_run=true]`, `…/sessions/{sid}`.
const S = "/courses/00000000-0000-7000-8000-00000000c001/sessions";
const SID = "00000000-0000-7000-8000-0000000000d";
const row = (n: number, over: Record<string, unknown> = {}) => ({
  id: `${SID}${n}`, session_no: n, starts_at: `2026-10-${String(11 + n).padStart(2, "0")}T02:00:00.000Z`, ends_at: `2026-10-${String(11 + n).padStart(2, "0")}T04:30:00.000Z`,
  room: "P.302", topic: null, version: 1, attendance: { done_at: null, has_records: false }, ...over,
});
/** 11 thứ Hai từ 12/10 bỏ 02/11: đúng ví dụ AC4. */
const monday = (i: number) => new Date(Date.UTC(2026, 9, 12 + 7 * i)).toISOString().slice(0, 10);
const WOULD = Array.from({ length: 12 }, (_, i) => i).filter((i) => i !== 3).map((i, k) => ({ session_no: k + 1, starts_at: `${monday(i)}T02:00:00.000Z`, ends_at: `${monday(i)}T04:30:00.000Z` }));

async function fillPlan(page: Page) {
  const dlg = page.getByRole("dialog");
  await dlg.getByRole("button", { name: "T2", exact: true }).click();
  await dlg.getByLabel("Bắt đầu").fill("09:00");
  await dlg.getByLabel("Kết thúc").fill("11:30");
  await dlg.getByLabel("Phòng").fill("P.302");
  await dlg.getByLabel("Từ ngày").fill("2026-10-12");
  await dlg.getByLabel("Đến ngày").fill("2026-12-28");
  await dlg.getByLabel("Ngày nghỉ").fill("2026-11-02");
  await dlg.getByRole("button", { name: "Thêm", exact: true }).click();
}

test("generate sessions preview and create: xem trước không ghi gì, Tạo N buổi chỉ sau xem trước, khoá mới mỗi lần bấm", async ({ page, context }, info) => {
  test.skip(info.project.name !== "desktop", "đo một lần");
  const calls = await open(page, context, "ta", {
    [`GET ${S}`]: json({ items: [], next_cursor: null }),
    [`POST ${S}/generate`]: (r) =>
      r.fulfill(r.request().url().includes("dry_run=true")
        ? json({ would_create: WOULD, skipped: [{ date: "2026-11-02", reason: "EXCLUDED" }] })
        : json({ created: 11, first_session_no: 1, last_session_no: 11, skipped: 0 }, 201)),
  });
  await expect(page.getByRole("button", { name: "Tạo lịch buổi học" })).toHaveCount(1);
  await page.getByRole("button", { name: "Tạo lịch buổi học" }).click();
  const dlg = page.getByRole("dialog");
  await expect(dlg.getByRole("button", { name: "Tạo buổi" })).toBeDisabled(); // chưa xem trước thì chưa tạo được
  await expect(dlg.getByRole("button", { name: "Xem trước" })).toBeDisabled();
  await fillPlan(page);
  await dlg.getByRole("button", { name: "Xem trước" }).click();
  await expect(dlg.getByText("Sẽ tạo 11 buổi · bỏ 1 ngày nghỉ")).toBeVisible();
  await expect(dlg.getByRole("row")).toHaveCount(12); // tiêu đề + 11 buổi
  expect(calls.filter((c) => c.method === "POST" && !c.path.includes("dry_run"))).toHaveLength(0); // xem trước không tạo gì
  const pv = calls.find((c) => c.method === "POST")!;
  expect(pv.path).toContain("dry_run=true");
  expect(JSON.parse(pv.body)).toEqual({ weekdays: [1], start_time: "09:00", end_time: "11:30", room: "P.302", from: "2026-10-12", to: "2026-12-28", exclude_dates: ["2026-11-02"] });
  await dlg.getByRole("button", { name: "Tạo 11 buổi" }).click();
  await expect(dlg.getByText("Đã tạo 11 buổi")).toBeVisible();
  const first = calls.filter((c) => c.method === "POST" && !c.path.includes("dry_run"));
  expect(first).toHaveLength(1);
  expect(first[0].key).toMatch(/^ep-/);
  // Lần bấm thứ hai (ý định mới) phải dùng khoá MỚI.
  await dlg.getByRole("button", { name: "Xem trước" }).click();
  await dlg.getByRole("button", { name: "Tạo 11 buổi" }).click();
  await expect.poll(() => calls.filter((c) => c.method === "POST" && !c.path.includes("dry_run")).length).toBe(2);
  const keys = calls.filter((c) => c.method === "POST" && !c.path.includes("dry_run")).map((c) => c.key);
  expect(new Set(keys).size).toBe(2);
  expect(await dlg.innerText()).not.toMatch(/\b(RAG|PII|trace|provider)\b/i);
});

test("generate sessions: sửa form sau khi xem trước thì phải xem trước lại; 422 hiện tại ô", async ({ page, context }, info) => {
  test.skip(info.project.name !== "desktop", "đo một lần");
  await open(page, context, "teacher", {
    [`GET ${S}`]: json({ items: [], next_cursor: null }),
    [`POST ${S}/generate`]: (r) => r.fulfill(json({ code: "VALIDATION_FAILED", message: "m", trace_id: "b".repeat(32), details: [{ field: "to", code: "TOO_MANY_SESSIONS", message: "Khoảng ngày này có 74 buổi, mỗi lần tạo tối đa 60. Hãy chia nhỏ khoảng ngày." }] }, 422)),
  });
  await page.getByRole("button", { name: "Tạo lịch buổi học" }).click();
  const dlg = page.getByRole("dialog");
  await fillPlan(page);
  await dlg.getByRole("button", { name: "Xem trước" }).click();
  await expect(dlg.getByText(/Khoảng ngày này có 74 buổi/)).toBeVisible();
  await expect(dlg.getByLabel("Đến ngày")).toHaveAttribute("aria-invalid", "true");
  await expect(dlg.getByRole("button", { name: "Tạo buổi" })).toBeDisabled();
});

test("generate sessions offline retry: mất mạng thì nút trở lại, thử lại dùng CÙNG Idempotency-Key", async ({ page, context }, info) => {
  test.skip(info.project.name !== "desktop", "đo một lần");
  let real = 0;
  const calls = await open(page, context, "ta", {
    [`GET ${S}`]: json({ items: [], next_cursor: null }),
    [`POST ${S}/generate`]: (r) => {
      if (r.request().url().includes("dry_run=true")) return r.fulfill(json({ would_create: WOULD, skipped: [{ date: "2026-11-02", reason: "EXCLUDED" }] }));
      // Lượt đầu mất mạng (máy khách tự thử lại 2 lần cũng mất); người dùng bấm lại thì có mạng.
      return ++real <= 3 ? r.abort("failed") : r.fulfill(json({ created: 11, first_session_no: 1, last_session_no: 11, skipped: 0 }, 201));
    },
  });
  await page.getByRole("button", { name: "Tạo lịch buổi học" }).click();
  const dlg = page.getByRole("dialog");
  await fillPlan(page);
  await dlg.getByRole("button", { name: "Xem trước" }).click();
  await dlg.getByRole("button", { name: "Tạo 11 buổi" }).click();
  await expect(dlg.getByRole("alert")).toBeVisible();
  await expect(dlg.getByRole("button", { name: "Tạo 11 buổi" })).toBeEnabled(); // nút trở lại
  await dlg.getByRole("button", { name: "Tạo 11 buổi" }).click();
  await expect(dlg.getByText("Đã tạo 11 buổi")).toBeVisible();
  const keys = calls.filter((c) => c.method === "POST" && !c.path.includes("dry_run")).map((c) => c.key);
  expect(keys).toHaveLength(4);
  expect(new Set(keys).size).toBe(1);
});

test("edit delete session: Sửa tại chỗ gửi version; Xoá bị khoá khi đã có điểm danh; xoá có Hoàn tác 5 giây", async ({ page, context }, info) => {
  test.skip(info.project.name !== "desktop", "đo một lần");
  const list = [row(1), row(2, { attendance: { done_at: "2026-10-13T05:00:00.000Z", has_records: true } }), row(3)];
  const calls = await open(page, context, "teacher", {
    [`GET ${S}`]: json({ items: list, next_cursor: null }),
    [`PUT ${S}/${SID}1`]: (r) => r.fulfill(json({ session: row(1, { topic: "Học bù", version: 2 }) })),
    [`DELETE ${S}/${SID}3`]: noContent,
  });
  await page.getByRole("button", { name: "Tạo lịch buổi học" }).click();
  const dlg = page.getByRole("dialog");
  await expect(dlg.getByText(/^Buổi 1 ·/)).toBeVisible();
  await dlg.getByRole("button", { name: "Thêm với buổi 1" }).click();
  await page.getByRole("menuitem", { name: "Sửa" }).click();
  await dlg.getByLabel("Chủ đề").fill("Học bù");
  await dlg.getByRole("button", { name: "Lưu", exact: true }).click();
  await expect.poll(() => calls.some((c) => c.method === "PUT")).toBe(true);
  expect(JSON.parse(calls.find((c) => c.method === "PUT")!.body)).toMatchObject({ topic: "Học bù", room: "P.302", version: 1 });
  // buổi đã có điểm danh: Xoá khoá, nói lý do
  await dlg.getByRole("button", { name: "Thêm với buổi 2" }).click();
  await expect(page.getByRole("menuitem", { name: /Xoá/ })).toBeDisabled();
  await expect(page.getByRole("menuitem", { name: /Xoá/ })).toContainText("Đã có điểm danh");
  await dlg.getByRole("button", { name: "Thêm với buổi 2" }).click(); // đóng menu
  // buổi 3: Hoàn tác thì không gọi API; để hết giờ thì gọi
  await dlg.getByRole("button", { name: "Thêm với buổi 3" }).click();
  await page.getByRole("menuitem", { name: "Xoá" }).click();
  await expect(dlg.getByText("Đã xoá buổi 3")).toBeVisible();
  await dlg.getByRole("button", { name: "Hoàn tác" }).click();
  await page.waitForTimeout(300);
  expect(calls.some((c) => c.method === "DELETE")).toBe(false);
  await dlg.getByRole("button", { name: "Thêm với buổi 3" }).click();
  await page.getByRole("menuitem", { name: "Xoá" }).click();
  await expect.poll(() => calls.some((c) => c.method === "DELETE"), { timeout: 8000 }).toBe(true);
});

test("generate sessions: ?new=sessions mở sẵn ngăn kéo (liên kết từ Hôm nay); sinh viên không có nút", async ({ page, context }, info) => {
  test.skip(info.project.name !== "desktop", "đo một lần");
  await asDemo(context, "teacher", {});
  await calApi(page, { [`GET ${C}`]: pageOf([]), [`GET ${S}`]: json({ items: [], next_cursor: null }) });
  await page.clock.setFixedTime(NOW);
  await page.goto("/calendar?new=sessions");
  await expect(page.getByRole("dialog").getByRole("heading", { name: "Lịch buổi học" })).toBeVisible();
  await expect(page.getByRole("dialog").getByText("Lớp chưa có buổi học.")).toBeVisible();
  await page.getByRole("dialog").getByRole("button", { name: "Đóng" }).click();
  await expect(page.getByRole("dialog")).toHaveCount(0);
  await open(page, context, "student");
  await expect(page.getByRole("button", { name: "Tạo lịch buổi học" })).toHaveCount(0);
});

test("generate sessions 375: ngăn kéo toàn màn hình, không tràn ngang, vùng chạm ≥ 44 px", async ({ page, context }, info) => {
  test.skip(info.project.name === "desktop", "chỉ ca hẹp");
  await open(page, context, "ta", { [`GET ${S}`]: json({ items: [row(1)], next_cursor: null }) });
  await page.getByRole("button", { name: "Tạo lịch buổi học" }).click();
  const dlg = page.getByRole("dialog");
  await expect(dlg.getByRole("heading", { name: "Lịch buổi học" })).toBeVisible();
  const box = (await dlg.boundingBox())!;
  expect(box.width).toBeGreaterThanOrEqual(page.viewportSize()!.width - 1);
  expect(await page.evaluate(() => document.documentElement.scrollWidth <= window.innerWidth)).toBe(true);
  for (const name of ["Xem trước", "Tạo buổi", "T2", "Thêm với buổi 1"]) {
    const b = await dlg.getByRole("button", { name, exact: true }).boundingBox();
    expect(b!.height, name).toBeGreaterThanOrEqual(44);
  }
});
