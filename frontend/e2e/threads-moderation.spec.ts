import { expect, test } from "@playwright/test";
import AxeBuilder from "@axe-core/playwright";
import { noContent } from "./support/cal-fixtures";
import { asDemo } from "./support/session";
import { COURSE, PID, TID, aiPost, err, json, precheckOk, row, threadApi, view } from "./support/thread-fixtures";

// US-P4-06: kiểm duyệt Threads trên giao diện (báo cáo, sửa / xoá của chủ bài, ẩn / ghim của Staff, hàng chờ). Gateway giả theo hợp đồng thật.
const T = `/courses/${COURSE}/threads`;
const me = { full_name: "Sinh Viên Hai", role: "STUDENT", is_me: true };
const human = (over: Record<string, unknown> = {}) => ({
  id: "00000000-0000-7000-8000-0000000000c1", kind: "HUMAN", author: { full_name: "Lê Văn A", role: "STUDENT", is_me: false }, body: "Em nghĩ là Điều 5.", verification_state: null, citations: [], version: 1, created_at: "2026-10-11T01:30:00Z", ...over,
});
const withThread = (over: Record<string, unknown>, posts: unknown[] = []) => view(posts, { version: 1, ...over });

test.beforeEach(async ({}, info) => {
  test.skip(info.project.name !== "desktop" && !info.title.startsWith("375"), "ca 375 riêng");
});

test("report undo: Báo cáo → Đã báo cáo · Hoàn tác (5 s) → DELETE; báo lại → Bạn đã báo cáo bài này, không phải lỗi", async ({ page, context }) => {
  await asDemo(context, "student", { person: "sv-2" });
  let reported = false;
  const calls = await threadApi(page, {
    [`GET ${T}/${TID}`]: json(withThread({}, [human()])),
    [`GET ${T}/${TID}/similar`]: json({ items: [] }),
    [`POST ${T}/${TID}/report`]: (r) => { const first = !reported; reported = true; return r.fulfill(first ? json({ reported: true }, 201) : err(409, "ALREADY_REPORTED")); },
    [`DELETE ${T}/${TID}/report`]: noContent,
  });
  await page.goto(`/threads/${TID}`);
  await page.getByRole("button", { name: "Tuỳ chọn bài" }).first().click();
  await page.getByRole("menuitem", { name: "Báo cáo" }).click();
  await page.getByLabel("Lý do báo cáo").selectOption("OTHER");
  await page.getByLabel("Ghi chú").fill("Quảng cáo khoá học");
  await page.getByRole("button", { name: "Gửi báo cáo" }).click();
  const line = page.getByRole("status").filter({ hasText: "Đã báo cáo" });
  await expect(line).toContainText("Đã báo cáo");
  expect(JSON.parse(calls.find((c) => c.method === "POST" && c.path.endsWith("/report"))!.body)).toEqual({ reason: "OTHER", note: "Quảng cáo khoá học" });
  await expect(page.locator("[data-ep-toast], [role=alertdialog]")).toHaveCount(0);
  await line.getByRole("button", { name: "Hoàn tác" }).click();
  await expect.poll(() => calls.some((c) => c.method === "DELETE" && c.path === `${T}/${TID}/report`)).toBe(true);
  await page.getByRole("button", { name: "Tuỳ chọn bài" }).first().click();
  await page.getByRole("menuitem", { name: "Báo cáo" }).click();
  await page.getByRole("button", { name: "Gửi báo cáo" }).click();
  await expect(page.getByRole("status").filter({ hasText: "Bạn đã báo cáo bài này" })).toBeVisible();
  await expect(page.getByText("Bạn đã báo cáo bài này")).toHaveCount(1);
});

test("owner edit delete: sửa gửi version, PII → hai lối, xoá hỏi hậu quả bằng số rồi về danh sách", async ({ page, context }) => {
  await asDemo(context, "student", { person: "sv-2" });
  const calls = await threadApi(page, {
    [`GET ${T}`]: json({ pinned: [], items: [], next_cursor: null }),
    [`GET ${T}/${TID}`]: json(withThread({ author: me }, [human()])),
    [`GET ${T}/${TID}/similar`]: json({ items: [] }),
    [`POST ${T}/precheck`]: json(precheckOk),
    [`PUT ${T}/${TID}`]: (r) => {
      const body = JSON.parse(r.request().postData() ?? "{}");
      return r.fulfill(body.redact || !body.body.includes("20229001") ? json(view([], { version: 2, edited: true })) : err(422, "PII_DETECTED", { reasons: [{ type: "MSSV", count: 1 }], personal_question: false }));
    },
    [`DELETE ${T}/${TID}`]: noContent,
  });
  await page.goto(`/threads/${TID}`);
  await page.getByRole("button", { name: "Tuỳ chọn bài" }).first().click();
  await page.getByRole("menuitem", { name: "Sửa" }).click();
  await page.getByLabel(/Nội dung/).fill("MSSV 20229001 hỏi Điều 5");
  await page.getByRole("button", { name: "Lưu" }).click();
  const dialog = page.getByRole("dialog");
  await expect(dialog.getByRole("button", { name: "Chuyển sang chat riêng" })).toBeVisible();
  await dialog.getByRole("button", { name: "Ẩn thông tin rồi lưu" }).click();
  await expect.poll(() => calls.filter((c) => c.method === "PUT").length).toBe(2);
  const last = JSON.parse(calls.filter((c) => c.method === "PUT").at(-1)!.body);
  expect(last).toMatchObject({ version: 1, redact: true });
  await page.getByRole("button", { name: "Tuỳ chọn bài" }).first().click();
  await page.getByRole("menuitem", { name: "Xoá" }).click();
  await expect(page.getByRole("dialog")).toContainText("Câu hỏi và 1 bình luận sẽ bị xoá");
  await page.getByRole("dialog").getByRole("button", { name: "Xoá" }).click();
  await expect(page).toHaveURL(/\/threads$/);
  expect(calls.some((c) => c.method === "DELETE" && c.path === `${T}/${TID}`)).toBe(true);
});

test("staff hide pin: Ẩn bài cần lý do + version, Bỏ ẩn, Ghim; chủ bài thấy một dòng lý do", async ({ page, context }) => {
  await asDemo(context, "ta");
  let hidden: { reason: string; auto: boolean } | null = null;
  const calls = await threadApi(page, {
    [`GET ${T}/${TID}`]: (r) => r.fulfill(json(withThread({ hidden }, [aiPost({ verification_state: "PENDING" })]))),
    [`GET ${T}/${TID}/similar`]: json({ items: [] }),
    [`POST ${T}/${TID}/hide`]: (r) => { hidden = { reason: JSON.parse(r.request().postData() ?? "{}").reason, auto: false }; return r.fulfill(json({ id: TID, hidden: true, version: 2 })); },
    [`POST ${T}/${TID}/unhide`]: (r) => { hidden = null; return r.fulfill(json({ id: TID, hidden: false, version: 3 })); },
    [`POST ${T}/${TID}/pin`]: noContent,
  });
  await page.goto(`/threads/${TID}`);
  await page.getByRole("button", { name: "Tuỳ chọn bài" }).first().click();
  await page.getByRole("menuitem", { name: "Ẩn bài" }).click();
  await expect(page.getByRole("button", { name: "Ẩn bài" })).toBeDisabled();
  await page.getByLabel("Lý do ẩn").fill("Lạc đề");
  await page.getByRole("button", { name: "Ẩn bài" }).click();
  await expect(page.getByText("Đã ẩn · Lạc đề")).toBeVisible();
  expect(JSON.parse(calls.find((c) => c.path.endsWith("/hide"))!.body)).toEqual({ reason: "Lạc đề", version: 1 });
  await page.getByRole("button", { name: "Tuỳ chọn bài" }).first().click();
  await page.getByRole("menuitem", { name: "Bỏ ẩn" }).click();
  await expect(page.getByText("Đã ẩn · Lạc đề")).toHaveCount(0);
  await page.getByRole("button", { name: "Tuỳ chọn bài" }).first().click();
  await page.getByRole("menuitem", { name: "Ghim" }).click();
  await expect.poll(() => calls.some((c) => c.path.endsWith("/pin"))).toBe(true);
});

test("own hidden post: chủ bài thấy một dòng, không thấy người báo cáo", async ({ page, context }) => {
  await asDemo(context, "student", { person: "sv-2" });
  await threadApi(page, {
    [`GET ${T}/${TID}`]: json(withThread({ author: me, hidden: { reason: "Nhiều báo cáo, chờ duyệt", auto: true } }, [])),
    [`GET ${T}/${TID}/similar`]: json({ items: [] }),
  });
  await page.goto(`/threads/${TID}`);
  await expect(page.getByText("Bài của bạn đang được giảng viên xem xét")).toBeVisible();
  await expect(page.getByText(/báo cáo/i)).toHaveCount(0);
});

test("pinned + moderation: ghim nằm trên items; Staff có hàng chờ, sinh viên không có tab", async ({ page, context }) => {
  await asDemo(context, "ta");
  const item = { kind: "post", thread_id: TID, post_id: PID, title_or_excerpt: "Quảng cáo khoá học", reports: { total: 2, by_reason: { SPAM: 2 } }, hidden_at: null, hidden_by_auto: false, created_at: "2026-10-11T02:00:00Z" };
  await page.route(`**/api/v1/courses/${COURSE}/moderation/queue**`, (r) => (r.request().method() === "OPTIONS" ? r.fulfill(noContent) : r.fulfill(json({ items: [item], next_cursor: null }))));
  await threadApi(page, {
    [`GET ${T}`]: json({ pinned: [row({ id: "p1", title: "Nội quy lớp", pinned: true })], items: [row({ id: "x2", title: "Câu hỏi thường" })], next_cursor: null }),
  });
  await page.goto("/threads");
  const rows = page.locator("[data-part=thread-list] li");
  await expect(rows.nth(0)).toContainText("Nội quy lớp");
  await expect(rows.nth(1)).toContainText("Câu hỏi thường");
  await page.getByRole("tab", { name: "Cần xem" }).click();
  await expect(page.getByRole("row", { name: /Quảng cáo khoá học/ })).toContainText("2 · Spam hoặc quảng cáo");
  const a11y = await new AxeBuilder({ page }).analyze();
  expect(a11y.violations.filter((v) => v.impact === "critical" || v.impact === "serious")).toEqual([]);
});

test("student list has no moderation tab", async ({ page, context }) => {
  await asDemo(context, "student", { person: "sv-2" });
  await threadApi(page, { [`GET ${T}`]: json({ pinned: [], items: [row()], next_cursor: null }) });
  await page.goto("/threads");
  await expect(page.getByRole("tab", { name: "Cần xem" })).toHaveCount(0);
});

test("375: menu và biểu mẫu báo cáo không tràn ngang", async ({ page, context }, info) => {
  test.skip(info.project.name === "desktop", "chỉ ca 375");
  await asDemo(context, "student", { person: "sv-2" });
  await threadApi(page, { [`GET ${T}/${TID}`]: json(withThread({}, [human()])), [`GET ${T}/${TID}/similar`]: json({ items: [] }) });
  await page.goto(`/threads/${TID}`);
  await page.getByRole("button", { name: "Tuỳ chọn bài" }).first().click();
  await page.getByRole("menuitem", { name: "Báo cáo" }).click();
  await expect(page.getByLabel("Lý do báo cáo")).toBeVisible();
  expect(await page.evaluate(() => document.documentElement.scrollWidth - document.documentElement.clientWidth)).toBeLessThanOrEqual(0);
});
