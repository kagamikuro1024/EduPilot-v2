import { expect, test } from "@playwright/test";
import type { Page } from "@playwright/test";
import { asJwt } from "./support/session";
import { frame, json, sse } from "./support/chat-fixtures";

// FEAT-escalation-notify: chuông thật (US-P4-02 AC2, AC9, AC11). Gateway giả bằng page.route theo hợp đồng thật (`GET /notifications`, `GET /events`).
const COURSE = "00000000-0000-7000-8000-00000000c001";
const note = (over: Record<string, unknown> = {}) => ({
  id: "00000000-0000-7000-8000-0000000000b1", type: "TICKET_NEW", title: "1 câu hỏi đang chờ giảng viên", body: "INT1006", link: `/inbox?course=${COURSE}`, course_id: COURSE,
  read_at: null, created_at: new Date().toISOString(), ...over,
});
const READY = 'retry: 3000\n\nevent: ready\ndata: {"conn_id":"c"}\n\n';
const bell = (page: Page) => page.getByRole("button", { name: /^Thông báo/ });

async function setup(page: Page, h: { list?: (n: number) => unknown; events?: Parameters<Page["route"]>[1] }) {
  await asJwt(page, "TEACHER", { email: "gv@ptit.edu.vn", fullName: "Giảng Viên Thử" });
  let gets = 0;
  await page.route("**/api/v1/notifications*", (r) => {
    if (r.request().method() === "OPTIONS") return r.fulfill({ status: 204, headers: json({}).headers });
    gets++;
    const body = h.list ? h.list(gets) : { items: [], next_cursor: null, unread_count: 0 };
    return typeof body === "object" && body && "status" in (body as object) ? r.fulfill(body as never) : r.fulfill(json(body));
  });
  await page.route("**/api/v1/events", h.events ?? ((r) => r.fulfill(sse(READY))));
  return () => gets;
}

test.beforeEach(async ({}, info) => {
  test.skip(info.project.name !== "desktop", "mobile do ca 375 riêng");
});

test("bell realtime: khung SSE `notification` đổi chuông không cần tải lại trang", async ({ page }) => {
  const frameData = { ...note(), read: false, unread_count: 1 };
  let loaded: () => number = () => 0;
  loaded = await setup(page, {
    // khung tới SAU khi danh sách đầu đã tải (khung đẩy cập nhật bộ nhớ đệm đã có)
    events: async (r) => {
      while (loaded() < 1) await new Promise((res) => setTimeout(res, 50));
      await r.fulfill(sse(READY + frame("notification", frameData, "1000-1")));
    },
  });
  await page.goto("/");
  await expect(page.locator("[data-part=bell-dot]")).toHaveCount(1);
  await expect(page.locator("[data-part=bell-live]")).toHaveText("Có 1 thông báo mới");
  await bell(page).click();
  const first = page.locator("[data-part=notifications] li").first();
  await expect(first).toContainText("1 câu hỏi đang chờ giảng viên");
  await expect(first).toContainText("Hộp thư hỗ trợ");
  expect(await page.locator("[data-part=notifications]").innerText()).not.toMatch(/INT1006 ·|Nguyễn|MSSV/);
});

test("bell third tab falls back to polling: 429 SSE_LIMIT_REACHED không báo lỗi, chuông vẫn đúng từ GET /notifications", async ({ page }) => {
  await setup(page, {
    list: () => ({ items: [note({ title: "2 câu hỏi đang chờ giảng viên" })], next_cursor: null, unread_count: 2 }),
    events: (r) => r.fulfill(json({ code: "SSE_LIMIT_REACHED", message: "m", trace_id: "b".repeat(32), retry_after: 30 }, 429)),
  });
  await page.goto("/");
  await expect(page.locator("[data-part=bell-dot]")).toHaveCount(1);
  await bell(page).click();
  await expect(page.locator("[data-part=notifications] li").first()).toContainText("2 câu hỏi đang chờ giảng viên");
  await expect(page.getByRole("alert").filter({ hasText: /\S/ })).toHaveCount(0);
});

test("bell refresh error: làm mới thất bại giữ danh sách cũ, chỉ một dòng nhạt Chưa cập nhật được", async ({ page }) => {
  const gets = await setup(page, {
    list: (n) => (n === 1 ? { items: [note()], next_cursor: null, unread_count: 1 } : json({ code: "INTERNAL", message: "m", trace_id: "b".repeat(32) }, 500)),
  });
  await page.goto("/");
  await expect(page.locator("[data-part=bell-dot]")).toHaveCount(1);
  await page.evaluate(() => window.dispatchEvent(new Event("visibilitychange"))); // lấy lại focus → làm mới
  await expect.poll(() => gets(), { timeout: 15_000 }).toBeGreaterThan(1);
  await bell(page).click();
  await expect(page.getByText("Chưa cập nhật được")).toBeVisible();
  await expect(page.locator("[data-part=notifications] li").first()).toContainText("1 câu hỏi đang chờ giảng viên");
  await expect(page.getByText("Chưa tải được thông báo.")).toHaveCount(0);
});
