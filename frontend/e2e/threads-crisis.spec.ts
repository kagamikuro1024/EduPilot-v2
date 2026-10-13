import { expect, test, type BrowserContext, type Page } from "@playwright/test";
import { asDemo } from "./support/session";
import { COURSE, json, precheckOk, row, threadApi } from "./support/thread-fixtures";

// US-P4-07 AC4: 422 CRISIS_SUPPORT ở Threads → hộp hỗ trợ; nháp không mất; precheck không bao giờ chặn. Gateway giả theo hợp đồng thật.
const T = `/courses/${COURSE}/threads`;
const MSG = "Mình rất tiếc khi nghe điều này. Bạn không phải một mình, hãy nói chuyện với giảng viên của bạn nhé. Phòng Công tác sinh viên, phòng 101 nhà A1";
const crisis = json({ code: "CRISIS_SUPPORT", message: MSG, trace_id: "b".repeat(32), details: { actions: ["OPEN_PRIVATE_CHAT"] } }, 422);
const TEXT = "Em không muốn sống nữa, học hành áp lực quá";

test.beforeEach(async ({}, info) => {
  test.skip(info.project.name !== "desktop", "ca desktop");
});

async function open(page: Page, context: BrowserContext, role: "student" | "ta", extra: Parameters<typeof threadApi>[1] = {}) {
  await asDemo(context, role, role === "student" ? { person: "sv-2" } : {});
  const calls = await threadApi(page, { [`GET ${T}`]: json({ pinned: [], items: [row()], next_cursor: null }), [`POST ${T}/precheck`]: json(precheckOk), [`POST ${T}`]: crisis, ...extra });
  await page.goto("/threads");
  await page.getByRole("button", { name: "Đặt câu hỏi" }).first().click();
  await page.getByLabel(/Tiêu đề/).fill("Em cần nói chuyện");
  await page.getByLabel(/Nội dung/).fill(TEXT);
  await page.locator("[data-part=thread-form]").getByRole("button", { name: "Đăng" }).click();
  return calls;
}

test("crisis dialog keeps draft: hộp hỗ trợ nêu đúng lời của máy chủ, Quay lại sửa giữ nguyên chữ, precheck không chặn", async ({ page, context }) => {
  const calls = await open(page, context, "student");
  const dialog = page.getByRole("dialog");
  await expect(dialog).toContainText(MSG);
  await expect(dialog.getByRole("button", { name: "Mở chat riêng" })).toBeVisible();
  await dialog.getByRole("button", { name: "Quay lại sửa" }).click();
  await expect(page.getByRole("dialog")).toHaveCount(0);
  await expect(page.getByLabel(/Nội dung/)).toHaveValue(TEXT);
  await expect(page.getByLabel(/Tiêu đề/)).toHaveValue("Em cần nói chuyện");
  expect(calls.filter((c) => c.path.endsWith("/precheck")).every((c) => c.method === "POST")).toBe(true);
  await expect(page.getByText(/Phát hiện/)).toHaveCount(0); // precheck cho qua: không có dòng cảnh báo
});

test("crisis dialog: Mở chat riêng mang nháp sang chat riêng", async ({ page, context }) => {
  await context.route("**/api/v1/chat/sessions?**", (r) => r.fulfill(json({ items: [], next_cursor: null })));
  await page.route("**/api/v1/chat/sessions/00000000-0000-7000-8000-0000000000d9/messages**", (r) => r.fulfill(json({ items: [], next_cursor: null })));
  await open(page, context, "student", { "POST /chat/sessions/from-draft": json({ session_id: "00000000-0000-7000-8000-0000000000d9", draft: { title: "Em cần nói chuyện", body: TEXT } }, 201) });
  await page.getByRole("dialog").getByRole("button", { name: "Mở chat riêng" }).click();
  await expect(page).toHaveURL(/\/chat\?session=00000000-0000-7000-8000-0000000000d9/);
  await expect(page.getByRole("textbox", { name: "Câu hỏi của bạn" })).toHaveValue(TEXT);
});

test("crisis dialog: trợ giảng không có chat riêng, chỉ còn Quay lại sửa", async ({ page, context }) => {
  await open(page, context, "ta");
  const dialog = page.getByRole("dialog");
  await expect(dialog).toContainText(MSG);
  await expect(dialog.getByRole("button", { name: "Mở chat riêng" })).toHaveCount(0);
  await dialog.getByRole("button", { name: "Quay lại sửa" }).click();
  await expect(page.getByLabel(/Nội dung/)).toHaveValue(TEXT);
});
