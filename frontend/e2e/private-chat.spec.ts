import { expect, test } from "@playwright/test";
import { asDemo } from "./support/session";
import { MID, SID, chatApi, frame, json, msg, noContent, session, sse } from "./support/chat-fixtures";

// US-P3-05: /chat THẬT (gateway giả theo hợp đồng thật, support/chat-fixtures.ts).
const open = async (page: import("@playwright/test").Page, context: import("@playwright/test").BrowserContext, h: Parameters<typeof chatApi>[1] = {}) => {
  await asDemo(context, "student", { person: "sv-2" });
  const calls = await chatApi(page, h);
  await page.goto("/chat");
  await page.locator("[data-part=chat-thread]").waitFor();
  return calls;
};
const composer = (page: import("@playwright/test").Page) => page.getByRole("textbox", { name: "Câu hỏi của bạn" });
const sentStream = (...extra: string[]) =>
  sse(frame("status", { message_id: MID, stage: "received" }, "1:1-0"), frame("status", { message_id: MID, stage: "searching" }, "1:2-0"),
    frame("token", { off: 0, t: "Theo quy chế [1], " }, "1:3-0"), frame("token", { off: 18, t: "không được." }, "1:4-0"), ...extra,
    frame("done", { message_id: MID, citations: msg().citations, low_confidence: false, degraded: false }, "1:5-0"));

test.beforeEach(async ({}, info) => {
  test.skip(info.project.name !== "desktop", "mobile có ca 375 riêng");
});

test("layout / empty: một Panel, gợi ý bấm được, không nút nhờ giảng viên", async ({ page, context }) => {
  await open(page, context);
  await expect(page.getByText("Hỏi bất cứ điều gì về lớp này.")).toBeVisible();
  await expect(page.getByText(/Nhờ giảng viên/)).toHaveCount(0);
  expect(await page.locator("[data-part=chat-thread]").evaluate((e) => !!e.closest("[data-ep-panel]"))).toBe(true);
  const w = await page.locator("[data-part=chat-composer]").evaluate((e) => e.getBoundingClientRect().width);
  expect(w).toBeLessThanOrEqual(840);
});

test("send: token chạy ra, trích nguồn mở tại chỗ, phản hồi có Hoàn tác, không placeholder", async ({ page, context }) => {
  let sent = false; // lịch sử chỉ có tin sau khi máy chủ giả nhận POST (không đếm số lần GET: React Query có thể gộp / huỷ lượt tải)
  const calls = await open(page, context, {
    "POST /chat/sessions": json(session(), 201),
    [`POST /chat/sessions/${SID}/messages`]: (r) => { sent = true; return r.fulfill(sentStream(frame("notice", { masked: 2 }, "1:4-1"))); },
    [`GET /chat/sessions/${SID}/messages`]: (r) => r.fulfill(json({ items: sent ? [msg({ masked_count: 2 }), msg({ id: "u1", role: "USER", content: "Quy chế thi nói gì?", citations: [] })] : [], next_cursor: null })),
    [`PUT /chat/messages/${MID}/feedback`]: noContent,
  });
  await composer(page).fill("Quy chế thi nói gì?");
  await composer(page).press("Enter");
  await expect(page.getByText("Đã ẩn 2 thông tin cá nhân trước khi gửi cho AI")).toBeVisible();
  await expect(page.getByText(/Theo quy chế/)).toBeVisible();
  await page.getByRole("button", { name: /Nguồn tham khảo|Quy chế học vụ/ }).first().click();
  await expect(page.getByText("Sinh viên không được mang tài liệu vào phòng thi.")).toBeVisible();
  await expect(page.getByRole("link", { name: "Xem tài liệu" })).toHaveAttribute("href", /\/library\/00000000-0000-7000-8000-0000000000f1/);
  await page.getByRole("button", { name: "Hữu ích", exact: true }).click();
  await expect(page.getByText("Đã ghi nhận")).toBeVisible();
  await expect(page.getByRole("button", { name: "Hoàn tác" })).toBeVisible();
  expect(await page.locator("body").innerText()).not.toMatch(/\[\[\s*(SV|MSSV|EMAIL|SDT|CCCD)/i);
  const post = calls.find((c) => c.method === "POST" && c.path.endsWith("/messages"))!;
  expect(post.headers["idempotency-key"]).toMatch(/^[0-9a-f-]{36}$/);
  expect(JSON.parse(post.body)).toEqual({ content: "Quy chế thi nói gì?" });
  await expect(composer(page)).toHaveValue("");
});

test("low confidence sentence only: một câu nhạt, không nút Nhờ giảng viên, không gọi /escalate", async ({ page, context }) => {
  let sent = false;
  const calls = await open(page, context, {
    "POST /chat/sessions": json(session(), 201),
    [`POST /chat/sessions/${SID}/messages`]: (r) => {
      sent = true;
      return r.fulfill(sse(frame("status", { message_id: MID, stage: "received" }, "1:1-0"), frame("token", { off: 0, t: "Theo tài liệu thì chưa rõ." }, "1:2-0"),
        frame("done", { message_id: MID, citations: [], low_confidence: true, degraded: false }, "1:3-0")));
    },
    [`GET /chat/sessions/${SID}/messages`]: (r) => r.fulfill(json({ items: sent ? [msg({ content: "Theo tài liệu thì chưa rõ.", citations: [], low_confidence: true }), msg({ id: "u1", role: "USER", content: "Hỏi khó?", citations: [] })] : [], next_cursor: null })),
  });
  await composer(page).fill("Hỏi khó?");
  await composer(page).press("Enter");
  await expect(page.getByText("AI chưa đủ chắc chắn về câu này")).toBeVisible();
  await expect(page.getByRole("button", { name: /Nhờ giảng viên/ })).toHaveCount(0);
  await expect(page.locator("[data-part=unsure] svg")).toHaveCount(0);
  expect(calls.some((c) => /escalate/.test(c.path))).toBe(false);
  expect(await page.locator("body").innerText()).not.toMatch(/\b0[,.]\d{2,3}\b.*(tin cậy|chắc)/i); // không có con số
});

test("overloaded retry: có dòng thời gian chờ + Thử lại, không thêm bong bóng người dùng, không nút nhờ giảng viên", async ({ page, context }) => {
  let retried = false;
  await open(page, context, {
    "POST /chat/sessions": json(session(), 201),
    [`POST /chat/sessions/${SID}/messages`]: (r) => r.fulfill(sse(frame("status", { message_id: MID, stage: "received" }, "1:1-0"), frame("error", { code: "OVERLOADED", message: "m", retry_after: 20 }, "1:2-0"))),
    [`POST /chat/messages/${MID}/retry`]: (r) => { retried = true; return r.fulfill(sse(frame("status", { message_id: MID, stage: "received" }, "2:1-0"), frame("token", { off: 0, t: "Xong." }, "2:2-0"), frame("done", { message_id: MID, citations: [], low_confidence: false, degraded: false }, "2:3-0"))); },
    [`GET /chat/sessions/${SID}/messages`]: (r) => r.fulfill(json({ items: [retried ? msg({ content: "Xong.", citations: [] }) : msg({ content: "", stream_status: "FAILED", error_code: "OVERLOADED", citations: [] }), msg({ id: "u1", role: "USER", content: "hỏi", citations: [] })], next_cursor: null })),
  });
  await composer(page).fill("hỏi");
  await composer(page).press("Enter");
  await expect(page.getByText("AI đang bận. Thử lại sau khoảng 20 giây.")).toBeVisible();
  await expect(page.getByText(/Nhờ giảng viên/)).toHaveCount(0);
  const bubbles = await page.locator("[data-part=chat-thread]").getByText("hỏi", { exact: true }).count();
  await page.getByRole("button", { name: "Thử lại" }).click();
  await expect(page.getByText("Xong.")).toBeVisible();
  expect(await page.locator("[data-part=chat-thread]").getByText("hỏi", { exact: true }).count()).toBeLessThanOrEqual(bubbles);
});

test("exam lock: ô soạn khoá kèm giờ mở lại, không nêu tên bài thi, nháp không mất", async ({ page, context }) => {
  await asDemo(context, "student", { person: "sv-2" });
  const until = new Date(Date.now() + 40 * 60_000).toISOString();
  await context.route("**/api/v1/me/exam-lock", (r) => r.fulfill(json({ locked: true, until })));
  await chatApi(page);
  await page.goto("/chat");
  await expect(page.getByText(/Chat tạm khóa trong lúc bạn làm bài thi\. Dùng lại được sau \d{2}:\d{2}\./)).toBeVisible();
  await expect(composer(page)).toBeDisabled();
  expect(await page.locator("main").innerText()).not.toMatch(/Kiểm tra tuần|INT1006|bài thi tuần/i);
});

test("409 EXAM_IN_PROGRESS khi gửi: giữ chữ trong ô soạn", async ({ page, context }) => {
  await open(page, context, {
    "POST /chat/sessions": json(session(), 201),
    [`POST /chat/sessions/${SID}/messages`]: json({ code: "EXAM_IN_PROGRESS", message: "m", trace_id: "b".repeat(32), details: { until: new Date(Date.now() + 600_000).toISOString() } }, 409),
  });
  await composer(page).fill("chữ không được mất");
  await composer(page).press("Enter");
  await expect(page.locator("[data-part=chat-composer]").getByRole("alert")).toContainText("Chat tạm khóa trong lúc bạn làm bài thi");
  await expect(composer(page)).toHaveValue("chữ không được mất");
});

test("draft: nháp tự lưu 2 s và khôi phục sau tải lại", async ({ page, context }) => {
  await open(page, context);
  await composer(page).fill("nháp của tôi");
  await page.waitForTimeout(2500);
  await page.reload();
  await page.locator("[data-part=chat-thread]").waitFor();
  await expect(composer(page)).toHaveValue("nháp của tôi");
});

test("reload mid-stream: tin đang sinh được nối lại bằng Last-Event-ID, câu trả lời đủ", async ({ page, context }) => {
  let gets = 0;
  const calls = await open(page, context, {
    [`GET /chat/sessions`]: json({ items: [session()], next_cursor: null }),
    [`GET /chat/sessions/${SID}/messages`]: (r) => {
      gets++;
      const asst = gets === 1 ? msg({ content: "Theo quy chế", stream_status: "STREAMING", streaming: true, citations: [] }) : msg({ content: "Theo quy chế đầy đủ.", citations: [] });
      return r.fulfill(json({ items: [asst, msg({ id: "u1", role: "USER", content: "hỏi", citations: [] })], next_cursor: null }));
    },
    [`GET /chat/messages/${MID}/stream`]: (r) => r.fulfill(sse(frame("status", { message_id: MID, stage: "received" }, "1:1-0"), frame("token", { off: 0, t: "Theo quy chế" }, "1:2-0"), frame("token", { off: 12, t: " đầy đủ." }, "1:3-0"), frame("done", { message_id: MID, citations: [], low_confidence: false, degraded: false }, "1:4-0"))),
  });
  await page.locator("[data-part=chat-history] button[aria-pressed]").click();
  await expect(page.getByText("Theo quy chế đầy đủ.")).toBeVisible();
  expect(calls.some((c) => c.path === `/chat/messages/${MID}/stream`)).toBe(true);
});

test("calendar tool block: lịch hiện thành danh sách dòng, không JSON thô", async ({ page, context }) => {
  const ev = [{ title: "Thi giữa kỳ", type: "EXAM", starts: "Chủ Nhật, 20/12 · 14:00", location: "P.301" }, { title: "Nộp báo cáo", type: "OTHER", starts: "Thứ Ba, 22/12 · 09:00" }];
  let sent = false;
  await open(page, context, {
    "POST /chat/sessions": json(session(), 201),
    [`POST /chat/sessions/${SID}/messages`]: (r) => { sent = true; return r.fulfill(sse(frame("status", { message_id: MID, stage: "received" }, "1:1-0"), frame("block", { kind: "exam_schedule", data: { events: ev } }, "1:2-0"), frame("token", { off: 0, t: "Lịch thi của bạn:" }, "1:3-0"), frame("done", { message_id: MID, citations: [], low_confidence: false, degraded: false }, "1:4-0"))); },
    [`GET /chat/sessions/${SID}/messages`]: (r) => r.fulfill(json({ items: sent ? [msg({ citations: [], content: "Lịch thi của bạn:", blocks: [{ kind: "exam_schedule", data: { events: ev } }] }), msg({ id: "u1", role: "USER", content: "Khi nào thi?", citations: [] })] : [], next_cursor: null })),
  });
  await composer(page).fill("Khi nào thi?");
  await composer(page).press("Enter");
  const block = page.locator("[data-part=calendar-block]").first();
  await expect(block).toBeVisible();
  await expect(block.getByRole("heading", { name: "Lịch thi" })).toBeVisible();
  await expect(block.getByRole("listitem")).toHaveCount(2);
  await expect(block).toContainText("Chủ Nhật, 20/12 · 14:00 Thi giữa kỳ · P.301");
  expect(await block.innerText()).not.toMatch(/[{}\[\]"]|events/);
});

test("375: không tràn ngang, vùng chạm ≥ 44 px", async ({ page, context }) => {
  await page.setViewportSize({ width: 375, height: 812 });
  await open(page, context);
  expect(await page.evaluate(() => document.documentElement.scrollWidth - document.documentElement.clientWidth)).toBeLessThanOrEqual(0);
  const small = await page.locator("main button:visible").evaluateAll((bs) => bs.filter((b) => b.getBoundingClientRect().height < 44 && !b.closest("[data-part=chat-thread]")).map((b) => b.textContent));
  expect(small).toEqual([]);
});
