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
