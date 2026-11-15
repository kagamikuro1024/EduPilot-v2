import { expect, test } from "@playwright/test";
import { chatApi } from "./support/chat-fixtures";
import { json, practiceApi } from "./support/practice-fixtures";
import { asDemo } from "./support/session";

// US-P9-10 (liêm chính): luyện đề KHÔNG khoá chat (Q-P9-2, FR-24) và không bị khoá bởi bài thi; chat chỉ khoá khi có lượt bài thi PE đang làm.
// Phần máy chủ (không lộ đáp án, `search_library` ẩn EXAM_PAPER khi khoá) do test Go kiểm: TestPracticeNoAnswerLeak, TestLibraryToolHidesExamPaperWhileLocked.
test.beforeEach(async ({ context }, info) => {
  test.skip(info.project.name !== "desktop", "ca desktop");
  await asDemo(context, "student", { person: "sv-2" });
});

const composer = (page: import("@playwright/test").Page) => page.getByRole("textbox", { name: "Câu hỏi của bạn" });

test("đang có lượt bài thi: chat bị khoá nhưng luyện đề vẫn bắt đầu được", async ({ page, context }) => {
  const until = new Date(Date.now() + 40 * 60_000).toISOString();
  await context.route("**/api/v1/me/exam-lock", (r) => r.fulfill(json({ locked: true, until })));
  await chatApi(page);
  await page.goto("/chat");
  await expect(composer(page)).toBeDisabled();

  await practiceApi(page);
  await context.route("**/api/v1/me/exam-lock", (r) => r.fulfill(json({ locked: true, until })));
  await page.goto("/practice");
  await page.getByRole("listitem").filter({ hasText: "Hàm băm" }).getByRole("button").click();
  await page.waitForURL(/\/practice\/att-1/);
  await expect(page.getByRole("heading", { name: "Hàm băm", level: 1 })).toBeVisible();
});

test("đang thi thử: chat riêng không bị khoá", async ({ page, context }) => {
  await practiceApi(page, {}, { mockCount: 5 });
  await context.route("**/api/v1/me/exam-lock", (r) => r.fulfill(json({ locked: false })));
  await page.goto("/practice");
  await page.getByRole("tab", { name: /Thi thử/ }).click();
  await page.getByRole("button", { name: "Bắt đầu thi thử" }).click();
  await expect(page.locator("[data-part=practice-mock]")).toBeVisible();
  await chatApi(page);
  await page.goto("/chat");
  await expect(composer(page)).toBeEnabled();
});
