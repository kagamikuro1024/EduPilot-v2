import { expect, test } from "@playwright/test";
import type { Page } from "@playwright/test";
import { asDemo, asJwt } from "./support/session";
import { SID, chatApi, frame, json, msg, sse } from "./support/chat-fixtures";

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

// ---------- US-P4-03: hộp thư hỗ trợ thật (hàng chờ, nhận, trả lời, đóng) ----------
const TID = "00000000-0000-7000-8000-0000000000f7";
const SUB = "00000000-0000-7000-8000-0000000000a0"; // `sub` của JWT giả
const ticket = (over: Record<string, unknown> = {}) => ({
  id: TID, status: "OPEN", kind: "QUESTION", student: { id: "00000000-0000-7000-8000-0000000000a1", full_name: "Sinh Viên Thử" }, claimed_by: null, answered_by: null, age_minutes: 95, sla_stage: 0,
  reopened_count: 0, save_as_knowledge: false, course: { id: COURSE, class_code: "761987" }, version: 1, created_at: "2026-10-11T02:00:00Z", ...over,
});
const row = (t: ReturnType<typeof ticket>) => ({ id: t.id, status: t.status, kind: t.kind, student: t.student, question_preview: "Điểm chuyên cần tính thế nào?", age_minutes: t.age_minutes, claimed_by: t.claimed_by, sla_stage: 0, course: t.course });
const detail = (t: ReturnType<typeof ticket>) => ({
  ticket: t, question: { id: "q1", content: "Điểm chuyên cần tính thế nào?", created_at: t.created_at },
  ai_answer: { id: "a1", content: "Mình chưa tìm thấy quy định này.", created_at: t.created_at, confidence: "0.42" }, transcript: [],
});

/** Gateway giả cho hàng chờ: trạng thái ticket nằm trong closure, mỗi POST đổi nó như máy chủ thật. */
async function inboxApi(page: Page, opts: { answer?: (route: Parameters<Parameters<Page["route"]>[1]>[0]) => unknown; claim?: number } = {}) {
  const state = { t: ticket() };
  const calls: Array<{ method: string; path: string; body: string; headers: Record<string, string> }> = [];
  await page.route("**/api/v1/courses/*/tickets**", async (r) => {
    const req = r.request();
    if (req.method() === "OPTIONS") return r.fulfill({ status: 204, headers: json({}).headers });
    const path = new URL(req.url()).pathname.replace("/api/v1", "");
    if (req.method() === "GET") return r.fulfill(json(path.endsWith("/tickets") ? { items: [row(state.t)], next_cursor: null } : detail(state.t)));
    calls.push({ method: req.method(), path, body: req.postData() ?? "", headers: req.headers() });
    if (path.endsWith("/claim")) {
      if (opts.claim) state.t = ticket({ status: "CLAIMED", claimed_by: { id: "other", full_name: "Cô Hoa" }, version: 2 }); // người kia nhận trước
      if (opts.claim) return r.fulfill(json({ code: "TICKET_ALREADY_CLAIMED", message: "m", trace_id: "b".repeat(32), details: { claimed_by: { id: "x", full_name: "Cô Hoa" } } }, opts.claim));
      state.t = ticket({ status: "CLAIMED", claimed_by: { id: SUB, full_name: "Giảng Viên Thử" }, version: 2 });
      return r.fulfill(json({ ticket: state.t }));
    }
    if (path.endsWith("/answer")) {
      if (opts.answer) return opts.answer(r);
      state.t = ticket({ status: "ANSWERED", answered_by: { id: SUB, full_name: "Giảng Viên Thử" }, version: 3 });
      return r.fulfill(json({ ticket: state.t, message: { id: "m1", content: "x", created_at: state.t.created_at } }));
    }
    return r.fulfill(json({ ticket: state.t }));
  });
  return { calls, state };
}

test("inbox: danh sách → chi tiết → Nhận → soạn → Gửi; trả lời có khoá Idempotency và nháp được xoá sau khi gửi", async ({ page, context }) => {
  await asDemo(context, "teacher");
  const api = await inboxApi(page);
  await page.goto("/inbox");
  const list = page.locator("[data-part=inbox-list]");
  await expect(list.getByText("Sinh Viên Thử")).toBeVisible();
  await expect(list.getByText("Điểm chuyên cần tính thế nào?")).toBeVisible();
  const detailPane = page.locator("[data-part=inbox-detail]");
  await expect(detailPane.getByText("Mình chưa tìm thấy quy định này.")).toBeVisible();
  await expect(detailPane.getByRole("textbox")).toHaveCount(0); // chưa nhận thì chưa soạn
  await detailPane.getByRole("button", { name: "Nhận", exact: true }).click();
  const box = detailPane.getByRole("textbox", { name: "Câu trả lời gửi sinh viên" });
  await expect(box).toBeVisible();
  expect(api.calls[0].path).toMatch(/\/tickets\/.+\/claim$/);
  await box.fill("Em xem lại bài 3 nhé.");
  await page.evaluate(() => window.dispatchEvent(new Event("pagehide"))); // nháp lưu ngay khi rời trang
  expect(await page.evaluate((k) => localStorage.getItem(k), `ep:draft:${SUB}:ticket:${TID}`)).toContain("Em xem lại bài 3 nhé.");
  await box.press("Enter");
  await expect.poll(() => api.calls.some((c) => c.path.endsWith("/answer"))).toBe(true);
  const post = api.calls.find((c) => c.path.endsWith("/answer"))!;
  expect(post.headers["idempotency-key"]).toMatch(/^ep-[0-9a-f-]{36}$/);
  expect(JSON.parse(post.body)).toMatchObject({ body: "Em xem lại bài 3 nhé.", version: 2 });
  await expect(detailPane.getByText("Giảng Viên Thử đã trả lời.")).toBeVisible();
  expect(await page.evaluate((k) => localStorage.getItem(k), `ep:draft:${SUB}:ticket:${TID}`)).toBeNull();
});

test("inbox PII: thông tin của sinh viên khác (422 PII_IN_ANSWER) giữ nguyên chữ đã gõ và báo ngay tại ô soạn", async ({ page, context }) => {
  await asDemo(context, "teacher");
  const api = await inboxApi(page, { answer: (r) => r.fulfill(json({ code: "PII_IN_ANSWER", message: "m", trace_id: "b".repeat(32), details: [{ start: 3, end: 9, kind: "NAME" }] }, 422)) });
  await page.goto("/inbox");
  const pane = page.locator("[data-part=inbox-detail]");
  await pane.getByRole("button", { name: "Nhận", exact: true }).click();
  const box = pane.getByRole("textbox", { name: "Câu trả lời gửi sinh viên" });
  await box.fill("Bạn Nguyễn Văn B cũng hỏi.");
  await box.press("Enter");
  await expect(pane.getByRole("alert")).toContainText(/thông tin|sinh viên/i);
  await expect(box).toHaveValue("Bạn Nguyễn Văn B cũng hỏi.");
  expect(api.calls.filter((c) => c.path.endsWith("/answer"))).toHaveLength(1);
});

test("inbox claim conflict: người khác đã nhận (409) → làm tươi, không có ô soạn, thấy tên người nhận", async ({ page, context }) => {
  await asDemo(context, "teacher");
  await inboxApi(page, { claim: 409 });
  await page.goto("/inbox");
  const pane = page.locator("[data-part=inbox-detail]");
  await pane.getByRole("button", { name: "Nhận", exact: true }).click();
  await expect(pane.getByText("Cô Hoa đã nhận.")).toBeVisible();
  await expect(pane.getByRole("textbox")).toHaveCount(0);
});

test("inbox 375: danh sách → chi tiết qua ?ticket=, nút Hộp thư quay lại, không tràn ngang", async ({ page, context }) => {
  await asDemo(context, "teacher");
  await inboxApi(page);
  await page.setViewportSize({ width: 375, height: 812 });
  await page.goto("/inbox");
  await page.locator("[data-ticket-id]").first().click();
  await expect(page).toHaveURL(/\?ticket=/);
  await expect(page.getByRole("button", { name: "Hộp thư", exact: true })).toBeVisible();
  expect(await page.evaluate(() => document.documentElement.scrollWidth <= window.innerWidth)).toBe(true);
  await page.getByRole("button", { name: "Hộp thư", exact: true }).click();
  await expect(page).not.toHaveURL(/\?ticket=/);
});

test("answer appears in student chat: tin của giảng viên hiện tên + vai, không có Hữu ích / Không hữu ích, và đánh dấu đã đọc thông báo của phiên", async ({ page, context }) => {
  await asDemo(context, "student", { person: "sv-2" });
  const staff = msg({ id: "s1", content: "Em xem lại bài 3 nhé.", citations: [], blocks: [{ kind: "ticket", data: { ticket_id: TID, answered_by: { full_name: "Nguyễn Thị Hoa", role: "TEACHER" } } }] });
  const read: string[] = [];
  await page.route("**/api/v1/notifications/read", (r) => {
    if (r.request().method() === "OPTIONS") return r.fulfill({ status: 204, headers: json({}).headers });
    read.push(r.request().postData() ?? "");
    return r.fulfill(json({ updated: 1 }));
  });
  await chatApi(page, {
    "GET /chat/sessions": json({ items: [{ id: SID, title: "Câu hỏi", last_message_at: "2026-10-11T02:00:00Z", document_id: null }], next_cursor: null }),
    [`GET /chat/sessions/${SID}/messages`]: json({ items: [staff, msg({ id: "u1", role: "USER", content: "Hỏi gì đó", citations: [] })], next_cursor: null, open_tickets: 0 }),
  });
  await page.goto(`/chat?session=${SID}`);
  const thread = page.locator("[data-part=chat-thread]");
  await expect(thread.locator("[data-part=staff-answer]")).toContainText("Giảng viên · Nguyễn Thị Hoa");
  await expect(thread.locator("[data-part=staff-answer]")).toContainText("Em xem lại bài 3 nhé.");
  await expect(thread.getByRole("button", { name: "Hữu ích", exact: true })).toHaveCount(0);
  await expect.poll(() => read.length).toBe(1);
  expect(JSON.parse(read[0])).toEqual({ link: `/chat?session=${SID}` });
});

test("inbox draft per ticket: nháp còn nguyên sau khi tải lại trang", async ({ page, context }) => {
  await asDemo(context, "teacher");
  await inboxApi(page);
  await page.goto("/inbox");
  const pane = page.locator("[data-part=inbox-detail]");
  await pane.getByRole("button", { name: "Nhận", exact: true }).click();
  await pane.getByRole("textbox", { name: "Câu trả lời gửi sinh viên" }).fill("Nháp dang dở");
  await page.evaluate(() => window.dispatchEvent(new Event("pagehide")));
  await page.reload();
  await expect(page.locator("[data-part=inbox-detail]").getByRole("textbox", { name: "Câu trả lời gửi sinh viên" })).toHaveValue("Nháp dang dở");
});

test("inbox send failure keeps draft: lỗi 500 giữ chữ, Thử lại dùng lại cùng khoá Idempotency", async ({ page, context }) => {
  await asDemo(context, "teacher");
  let n = 0;
  const api = await inboxApi(page, {
    answer: (r) => {
      n++;
      if (n === 1) return r.fulfill(json({ code: "INTERNAL", message: "m", trace_id: "b".repeat(32) }, 500));
      api.state.t = ticket({ status: "ANSWERED", answered_by: { id: SUB, full_name: "Giảng Viên Thử" }, version: 3 });
      return r.fulfill(json({ ticket: api.state.t, message: { id: "m1", content: "x", created_at: "2026-10-11T02:00:00Z" } }));
    },
  });
  await page.goto("/inbox");
  const pane = page.locator("[data-part=inbox-detail]");
  await pane.getByRole("button", { name: "Nhận", exact: true }).click();
  const box = pane.getByRole("textbox", { name: "Câu trả lời gửi sinh viên" });
  await box.fill("Trả lời quan trọng");
  await box.press("Enter");
  await expect(pane.getByRole("button", { name: "Gửi lại" })).toBeVisible();
  await expect(box).toHaveValue("Trả lời quan trọng");
  await pane.getByRole("button", { name: "Gửi lại" }).click();
  await expect(pane.getByText("Giảng Viên Thử đã trả lời.")).toBeVisible();
  const posts = api.calls.filter((c) => c.path.endsWith("/answer"));
  expect(posts).toHaveLength(2);
  expect(posts[1].headers["idempotency-key"]).toBe(posts[0].headers["idempotency-key"]);
});

test("inbox states: rỗng có Xem tất cả, lỗi có Thử lại mà không mất dữ liệu", async ({ page, context }) => {
  await asDemo(context, "teacher");
  let mode: "empty" | "error" = "empty";
  await page.route("**/api/v1/courses/*/tickets**", (r) => {
    if (r.request().method() === "OPTIONS") return r.fulfill({ status: 204, headers: json({}).headers });
    return mode === "empty" ? r.fulfill(json({ items: [], next_cursor: null })) : r.fulfill(json({ code: "INTERNAL", message: "m", trace_id: "b".repeat(32) }, 500));
  });
  await page.goto("/inbox");
  await expect(page.getByText("Không có câu hỏi nào đang chờ.")).toBeVisible();
  await expect(page.getByRole("button", { name: "Xem tất cả" })).toBeVisible();
  mode = "error";
  await page.getByRole("button", { name: "Xem tất cả" }).click();
  await expect(page.getByRole("button", { name: "Thử lại" })).toBeVisible();
  mode = "empty";
  await page.getByRole("button", { name: "Thử lại" }).click();
  await expect(page.getByText("Không có câu hỏi nào đang chờ.")).toBeVisible();
});

test("inbox all courses: chế độ Tất cả lớp của tôi gọi /me/tickets và mỗi dòng ghi mã lớp", async ({ page, context }) => {
  await asDemo(context, "teacher");
  await page.addInitScript(() => localStorage.setItem("ep:ui:course", "all"));
  const urls: string[] = [];
  await page.route("**/api/v1/me/tickets**", (r) => {
    if (r.request().method() === "OPTIONS") return r.fulfill({ status: 204, headers: json({}).headers });
    urls.push(r.request().url());
    return r.fulfill(json({ items: [row(ticket()), { ...row(ticket({ id: "00000000-0000-7000-8000-0000000000f8" })), course: { id: "c2", class_code: "761988" } }], next_cursor: null }));
  });
  await page.goto("/inbox");
  const list = page.locator("[data-part=inbox-list]");
  await expect(list).toContainText("761987");
  await expect(list).toContainText("761988");
  expect(urls[0]).toContain("status=OPEN");
});

test("inbox shortcuts: J / K đổi ticket, R vào ô soạn, không kích hoạt khi đang gõ", async ({ page, context }) => {
  await asDemo(context, "teacher");
  const second = ticket({ id: "00000000-0000-7000-8000-0000000000f8", student: { id: "s2", full_name: "Sinh Viên Hai" } });
  await page.route("**/api/v1/courses/*/tickets**", (r) => {
    if (r.request().method() === "OPTIONS") return r.fulfill({ status: 204, headers: json({}).headers });
    const path = new URL(r.request().url()).pathname;
    if (path.endsWith("/tickets")) return r.fulfill(json({ items: [row(ticket()), row(second)], next_cursor: null }));
    return r.fulfill(json(detail(path.endsWith("f8") ? second : ticket({ status: "CLAIMED", claimed_by: { id: SUB, full_name: "Giảng Viên Thử" } }))));
  });
  await page.goto("/inbox");
  await expect(page.locator("[data-part=inbox-list]").getByText("Sinh Viên Hai")).toBeVisible();
  await page.keyboard.press("j");
  await expect(page).toHaveURL(/ticket=.*f8/);
  await expect(page.locator("[data-part=inbox-detail]").getByRole("heading", { name: "Sinh Viên Hai" })).toBeVisible();
  await page.keyboard.press("k");
  await expect(page).not.toHaveURL(/ticket=.*f8/);
  await expect(page).toHaveURL(/ticket=/);
  await page.keyboard.press("r");
  const box = page.locator("[data-part=inbox-detail]").getByRole("textbox", { name: "Câu trả lời gửi sinh viên" });
  await expect(box).toBeFocused();
  await page.keyboard.type("jjkk");
  await expect(box).toHaveValue("jjkk");
  await expect(page).toHaveURL(/ticket=(?!.*f8)/);
});

test("bell count: số chưa đọc trên chuông, tối đa 9+", async ({ page }) => {
  await setup(page, { list: () => ({ items: [note()], next_cursor: null, unread_count: 15 }) });
  await page.goto("/");
  await expect(page.locator("[data-part=bell-dot]")).toHaveText("9+");
  await expect(bell(page)).toHaveAttribute("aria-label", "Thông báo, 15 chưa đọc");
});

test("bell count small: 3 chưa đọc hiện đúng số 3", async ({ page }) => {
  await setup(page, { list: () => ({ items: [note()], next_cursor: null, unread_count: 3 }) });
  await page.goto("/");
  await expect(page.locator("[data-part=bell-dot]")).toHaveText("3");
});

test("bell no jump: thông báo mới chèn lên đầu lúc đang mở không làm nhảy mục đang đọc", async ({ page }) => {
  let push = false;
  const items = [1, 2, 3].map((i) => note({ id: `00000000-0000-7000-8000-00000000000${i}`, title: `Thông báo số ${i}` }));
  await setup(page, {
    list: () => ({ items, next_cursor: null, unread_count: 3 }),
    events: async (r) => {
      while (!push) await new Promise((res) => setTimeout(res, 50));
      await r.fulfill(sse(READY + frame("notification", { ...note({ id: "00000000-0000-7000-8000-0000000000ff", title: "Thông báo mới nhất" }), read: false, unread_count: 4 }, "1000-1")));
    },
  });
  await page.goto("/");
  await bell(page).click();
  const second = page.locator("[data-part=notifications] li", { hasText: "Thông báo số 2" });
  await expect(second).toBeVisible();
  await page.waitForTimeout(500); // khung mở xong animation
  const before = (await second.boundingBox())!.y;
  push = true;
  await expect(page.locator("[data-part=notifications] li").first()).toContainText("Thông báo mới nhất");
  await page.waitForTimeout(200);
  const after = (await second.boundingBox())!.y;
  expect(Math.abs(after - before)).toBeLessThan(1.5);
});
