import { expect, test, type BrowserContext, type Page } from "@playwright/test";
import { asDemo } from "./support/session";
import { COURSE, PID, TID, aiPost, err, json, precheckOk, row, threadApi, view } from "./support/thread-fixtures";

// US-P3-06: /threads THẬT (gateway giả theo hợp đồng thật, support/thread-fixtures.ts).
const T = `/courses/${COURSE}/threads`;
const list = (items: unknown[] = [row()]) => json({ items, next_cursor: null });

async function openList(page: Page, context: BrowserContext, h: Parameters<typeof threadApi>[1] = {}, role: "student" | "ta" = "student") {
  await asDemo(context, role, role === "student" ? { person: "sv-2" } : {});
  const calls = await threadApi(page, { [`GET ${T}`]: list(), [`POST ${T}/precheck`]: json(precheckOk), ...h });
  await page.goto("/threads");
  await page.locator("[data-part=thread-list], [data-ep-panel]").first().waitFor();
  return calls;
}
const ask = (page: Page) => page.getByRole("button", { name: "Đặt câu hỏi" }).first();
const fillForm = async (page: Page, title: string, body: string) => {
  await ask(page).click();
  await page.getByLabel(/Tiêu đề/).fill(title);
  await page.getByLabel(/Nội dung/).fill(body);
};

test.beforeEach(async ({}, info) => {
  test.skip(info.project.name !== "desktop", "ca 375 riêng");
});

test("layout: danh sách là một nguồn cấp, nhãn trạng thái, hành động chính duy nhất Đăng", async ({ page, context }) => {
  await openList(page, context, { [`GET ${T}`]: list([row(), row({ id: "x2", title: "Giao thức TCP", answer_state: "VERIFIED" }), row({ id: "x3", title: "Hỏi thêm", answer_state: null })]) });
  const rows = page.locator("[data-part=thread-list]");
  await expect(rows.getByText("Chờ xác nhận").first()).toBeVisible();
  await expect(rows.getByText("Đã được giảng viên xác nhận")).toBeVisible();
  await ask(page).click();
  await expect(page.locator("main [data-variant=primary]:visible")).toHaveCount(1);
  await expect(page.locator("[data-part=thread-form]").getByRole("button", { name: "Đăng" })).toBeVisible();
});

test("empty: câu dạy bước kế + Đặt câu hỏi", async ({ page, context }) => {
  await openList(page, context, { [`GET ${T}`]: list([]) });
  await expect(page.getByText("Chưa có câu hỏi nào. Đặt câu hỏi đầu tiên.")).toBeVisible();
});

test("precheck debounce: gõ liên tục chỉ phát yêu cầu sau khi ngừng ≥ 800 ms; notice while typing hiện một dòng loại", async ({ page, context }) => {
  const calls = await openList(page, context, { [`POST ${T}/precheck`]: json({ allowed: false, reasons: [{ type: "MSSV", count: 1 }, { type: "EMAIL", count: 1 }], redacted_text: "x", redacted_title: "", personal_question: false }) });
  await ask(page).click();
  const body = page.getByLabel(/Nội dung/);
  await body.pressSequentially("MSSV 20229001 mail a@b.vn thi lại", { delay: 60 }); // ≈ 2 s liên tục
  const during = calls.filter((c) => c.path.endsWith("/precheck")).length;
  expect(during).toBeLessThanOrEqual(1);
  await expect(page.getByText("Phát hiện MSSV, Email")).toBeVisible();
  expect(calls.filter((c) => c.path.endsWith("/precheck")).length).toBeLessThanOrEqual(2);
  await expect(page.getByRole("dialog")).toHaveCount(0); // chưa mở hộp thoại
  await body.fill("");
  await expect(page.getByText(/Phát hiện/)).toHaveCount(0);
});

const piiBlocked = err(422, "PII_DETECTED", { reasons: [{ type: "MSSV", count: 1 }], redacted_text: "[đã ẩn]", redacted_title: "", personal_question: false });

test("dialog two paths: chặn 422 → đúng hai nút hành động; ghi pii BLOCKED ở máy chủ, không có thread nào", async ({ page, context }) => {
  await openList(page, context, { [`POST ${T}`]: piiBlocked });
  await fillForm(page, "Hỏi", "MSSV 20229001 được mấy điểm?");
  await page.locator("[data-part=thread-form]").getByRole("button", { name: "Đăng" }).click();
  const dlg = page.getByRole("dialog");
  await expect(dlg).toBeVisible();
  await expect(dlg).toContainText("Chúng tôi tìm thấy: 1 MSSV."); // viết hoa đúng nhãn (QC BUG-3)
  const names = await dlg.getByRole("button").allTextContents();
  expect(names.filter((n) => /Chuyển sang chat riêng|Ẩn thông tin rồi đăng/.test(n))).toHaveLength(2);
  expect(names.map((n) => n.trim()).filter((n) => n && !/Chuyển sang chat riêng|Ẩn thông tin rồi đăng/.test(n))).toEqual([]);
  await page.keyboard.press("Escape"); // đóng chỉ quay lại ô soạn
  await expect(dlg).toHaveCount(0);
  await expect(page.getByLabel(/Nội dung/)).toHaveValue("MSSV 20229001 được mấy điểm?");
});

test("blocked then redact: 422 → Ẩn thông tin rồi đăng → 201, khoá Idempotency-Key MỚI và redact:true", async ({ page, context }) => {
  let n = 0;
  const calls = await openList(page, context, { [`POST ${T}`]: (r) => (++n === 1 ? r.fulfill(piiBlocked) : r.fulfill(json(view([]), 201))) });
  await fillForm(page, "Hỏi", "MSSV 20229001 được mấy điểm?");
  await page.locator("[data-part=thread-form]").getByRole("button", { name: "Đăng" }).click();
  await page.getByRole("dialog").getByRole("button", { name: "Ẩn thông tin rồi đăng" }).click();
  await expect(page.locator("[data-part=thread-form]")).toHaveCount(0);
  const posts = calls.filter((c) => c.method === "POST" && c.path === T);
  expect(posts).toHaveLength(2);
  expect(JSON.parse(posts[0].body).redact).toBe(false);
  expect(JSON.parse(posts[1].body).redact).toBe(true);
  expect(posts[1].headers["idempotency-key"]).not.toBe(posts[0].headers["idempotency-key"]);
});

test("switch keeps text: Chuyển sang chat riêng giữ đúng từng byte, mở /chat?session=…", async ({ page, context }) => {
  const body = "Dòng một có MSSV 20229001\n\n  Dòng ba thụt lề\tvà ký tự lạ ☃";
  await context.route("**/api/v1/chat/sessions?**", (r) => r.fulfill(json({ items: [], next_cursor: null })));
  await openList(page, context, {
    [`POST ${T}`]: piiBlocked,
    "POST /chat/sessions/from-draft": json({ session_id: "00000000-0000-7000-8000-0000000000d9", draft: { title: "Hỏi", body } }, 201),
  });
  await page.route("**/api/v1/chat/sessions/00000000-0000-7000-8000-0000000000d9/messages**", (r) => r.fulfill(json({ items: [], next_cursor: null })));
  await fillForm(page, "Hỏi", body);
  await page.locator("[data-part=thread-form]").getByRole("button", { name: "Đăng" }).click();
  await page.getByRole("dialog").getByRole("button", { name: "Chuyển sang chat riêng" }).click();
  await expect(page).toHaveURL(/\/chat\?session=00000000-0000-7000-8000-0000000000d9/);
  await expect(page.getByRole("textbox", { name: "Câu hỏi của bạn" })).toHaveValue(body);
});

test("personal question: Ẩn thông tin rồi đăng bị khoá, còn Chuyển sang chat riêng", async ({ page, context }) => {
  await openList(page, context, { [`POST ${T}`]: err(422, "PII_DETECTED", { reasons: [{ type: "PERSONAL_QUESTION", count: 1 }], redacted_text: "x", redacted_title: "", personal_question: true }) });
  await fillForm(page, "Điểm", "Em được mấy điểm giữa kỳ?");
  await page.locator("[data-part=thread-form]").getByRole("button", { name: "Đăng" }).click();
  const dlg = page.getByRole("dialog");
  await expect(dlg.getByRole("button", { name: "Ẩn thông tin rồi đăng" })).toBeDisabled();
  await expect(dlg.getByRole("button", { name: "Chuyển sang chat riêng" })).toBeEnabled();
  await expect(dlg.getByRole("button", { name: "Ẩn thông tin rồi đăng" })).toHaveAttribute("title", "Câu hỏi về điểm của riêng bạn không đăng công khai được.");
});

test("offline keeps draft: mất mạng giữ chữ, gửi lại cùng khoá; reload keeps draft", async ({ page, context }) => {
  let offline = true; // apiClient tự thử lại POST có khoá → mất mạng kéo dài tới khi test bật lại
  const calls = await openList(page, context, { [`POST ${T}`]: (r) => (offline ? r.abort("failed") : r.fulfill(json(view([]), 201))) });
  await fillForm(page, "Hỏi", "Câu hỏi sạch về Điều 5");
  await page.locator("[data-part=thread-form]").getByRole("button", { name: "Đăng" }).click();
  await expect(page.getByRole("alert").filter({ hasText: /kết nối/i })).toBeVisible();
  await expect(page.getByLabel(/Nội dung/)).toHaveValue("Câu hỏi sạch về Điều 5");
  offline = false;
  await page.getByRole("button", { name: "Gửi lại" }).click();
  await expect(page.locator("[data-part=thread-form]")).toHaveCount(0);
  const posts = calls.filter((c) => c.method === "POST" && c.path === T);
  expect(new Set(posts.map((c) => c.headers["idempotency-key"])).size).toBe(1);
});

test("reload keeps draft", async ({ page, context }) => {
  await openList(page, context);
  await fillForm(page, "Tiêu đề nháp", "Nội dung nháp");
  await page.waitForTimeout(2500);
  await page.reload();
  await ask(page).click();
  await expect(page.getByLabel(/Tiêu đề/)).toHaveValue("Tiêu đề nháp");
  await expect(page.getByLabel(/Nội dung/)).toHaveValue("Nội dung nháp");
});

test("exam lock: ô soạn khoá kèm giờ mở lại", async ({ page, context }) => {
  await asDemo(context, "student", { person: "sv-2" });
  await context.route("**/api/v1/me/exam-lock", (r) => r.fulfill(json({ locked: true, until: new Date(Date.now() + 30 * 60_000).toISOString() })));
  await threadApi(page, { [`GET ${T}`]: list() });
  await page.goto("/threads");
  await ask(page).click();
  await expect(page.getByText(/Chat tạm khóa trong lúc bạn làm bài thi\. Dùng lại được sau/)).toBeVisible();
  await expect(page.getByLabel(/Nội dung/)).toBeDisabled();
});

test("student detail: nhãn AI + Chờ xác nhận, nguồn mở tại chỗ, KHÔNG có độ tin cậy / nút duyệt", async ({ page, context }) => {
  await asDemo(context, "student", { person: "sv-2" });
  await threadApi(page, { [`GET ${T}/${TID}`]: json(view()), [`GET ${T}/${TID}/similar`]: json({ items: [{ id: "x9", title: "Thi lại mấy lần", preview: "p" }] }) });
  await page.goto(`/threads/${TID}`);
  await expect(page.locator("[data-part=ai-post]")).toContainText("AI");
  await expect(page.locator("[data-part=ai-post]")).toContainText("Chờ xác nhận");
  await page.getByRole("button", { name: /Quy chế học vụ/ }).click();
  await expect(page.getByText("Sinh viên được thi lại một lần.").last()).toBeVisible();
  await expect(page.getByText(/Độ tin cậy/)).toHaveCount(0);
  await expect(page.getByRole("button", { name: "Xác nhận" })).toHaveCount(0);
  await expect(page.getByRole("link", { name: "Thi lại mấy lần" })).toBeVisible();
});

test("staff review: Xác nhận / Chỉnh sửa / Loại cạnh bản nháp, độ tin cậy 0,72", async ({ page, context }) => {
  await asDemo(context, "ta");
  let state = "PENDING";
  const calls = await threadApi(page, {
    [`GET ${T}/${TID}`]: (r) => r.fulfill(json(view([aiPost({ verification_state: state, confidence: "0.720", rejected: state === "REJECTED" })], { ai_state: "ANSWERED" }))),
    [`GET ${T}/${TID}/similar`]: json({ items: [] }),
    [`POST /courses/${COURSE}/posts/${PID}/verify`]: (r) => { state = "VERIFIED"; return r.fulfill(json(aiPost({ verification_state: "VERIFIED" }))); },
    [`PUT /courses/${COURSE}/posts/${PID}/correct`]: (r) => { state = "CORRECTED"; return r.fulfill(json(aiPost({ verification_state: "CORRECTED" }))); },
    [`POST /courses/${COURSE}/posts/${PID}/reject`]: (r) => { state = "REJECTED"; return r.fulfill(json(aiPost({ verification_state: "REJECTED" }))); },
  });
  await page.goto(`/threads/${TID}`);
  await expect(page.getByText("Độ tin cậy 0,72")).toBeVisible();
  await page.getByRole("button", { name: "Xác nhận" }).click();
  await expect(page.locator("[data-part=ai-post]")).toContainText("Đã được giảng viên xác nhận");
  await page.getByRole("button", { name: "Chỉnh sửa" }).click();
  await page.getByLabel("Bản sửa").fill("Bản đã sửa.");
  await page.getByRole("button", { name: "Lưu bản sửa" }).click();
  await expect(page.locator("[data-part=ai-post]")).toContainText("Đã sửa bởi giảng viên");
  expect(JSON.parse(calls.find((c) => c.method === "PUT")!.body)).toEqual({ body: "Bản đã sửa.", version: 1 });
  await page.getByRole("button", { name: "Thêm hành động" }).click();
  await page.getByRole("menuitem", { name: "Loại" }).click();
  await expect(page.getByText("Đã loại một câu trả lời AI.")).toBeVisible();
});

test("375: danh sách và soạn không tràn ngang", async ({ page, context }) => {
  await page.setViewportSize({ width: 375, height: 812 });
  await openList(page, context);
  await ask(page).click();
  expect(await page.evaluate(() => document.documentElement.scrollWidth - document.documentElement.clientWidth)).toBeLessThanOrEqual(0);
});
