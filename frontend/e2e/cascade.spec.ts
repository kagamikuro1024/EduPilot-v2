import { expect, test, type BrowserContext, type Page } from "@playwright/test";
import { asDemo } from "./support/session";
import { COURSE, err, json, precheckOk, row, threadApi, view } from "./support/thread-fixtures";

const T = `/courses/${COURSE}/threads`;
async function openList(page: Page, context: BrowserContext, h: Parameters<typeof threadApi>[1] = {}) {
  await asDemo(context, "student", { person: "sv-2" });
  const calls = await threadApi(page, { [`GET ${T}`]: json({ items: [row()], next_cursor: null }), [`POST ${T}/precheck`]: json(precheckOk), ...h });
  await page.goto("/threads");
  await page.locator("[data-part=thread-list], [data-ep-panel]").first().waitFor();
  return calls;
}
const fillForm = async (page: Page, title: string, body: string) => {
  await page.getByRole("button", { name: "Đặt câu hỏi" }).first().click();
  await page.getByLabel(/Tiêu đề/).fill(title);
  await page.getByLabel(/Nội dung/).fill(body);
};

// US-DM-03 AC6 (giao diện của cổng THREADS_PRECHECK): bài bị chặn — do luật, decision, Structured hay đóng-an-toàn — hiện CÙNG hộp thoại hai lối,
// không nêu nguyên nhân kỹ thuật, không mất chữ đã gõ; chờ > 800 ms có chỉ báo «Đang kiểm tra». Các nhánh cascade ở máy chủ:
// backend-go/internal/contract TestCascadeThreadsHTTP (HTTP thật + Postgres + provider giả) và internal/{thread,grading,agent} tests.
const BAD = /\b(AI|RAG|PII|decision|fallback|trace|provider|model|LLM|Structured|cascade)\b/i;
const personal = err(422, "PII_DETECTED", { reasons: [{ type: "PERSONAL_QUESTION", count: 1 }], redacted_text: "Bạn ở lớp mình bị cảnh cáo học vụ phải không?", redacted_title: "", personal_question: true });

test.beforeEach(async ({}, info) => {
  test.skip(info.project.name !== "desktop", "tự đặt bề rộng");
});

test("bài bị chặn bởi cổng decision: cùng hộp thoại hai lối, không từ kỹ thuật, chữ còn nguyên", async ({ page, context }) => {
  await openList(page, context, { [`POST ${T}`]: personal });
  const body = "Bạn ở lớp mình bị cảnh cáo học vụ phải không?";
  await fillForm(page, "Hỏi", body);
  await page.locator("[data-part=thread-form]").getByRole("button", { name: "Đăng" }).click();
  const dlg = page.getByRole("dialog");
  await expect(dlg).toBeVisible();
  const names = (await dlg.getByRole("button").allTextContents()).map((n) => n.trim()).filter(Boolean);
  expect(names.filter((n) => /Chuyển sang chat riêng|Ẩn thông tin rồi đăng/.test(n))).toHaveLength(2);
  expect(names.filter((n) => !/Chuyển sang chat riêng|Ẩn thông tin rồi đăng/.test(n))).toEqual([]);
  expect(await dlg.innerText()).not.toMatch(BAD);
  await page.keyboard.press("Escape");
  await expect(page.getByLabel(/Nội dung/)).toHaveValue(body);
});

test("kiểm duyệt chậm > 800 ms: hiện «Đang kiểm tra»; nhanh thì không", async ({ page, context }) => {
  let n = 0;
  await openList(page, context, {
    [`POST ${T}`]: async (route) => {
      n++;
      if (n === 1) await new Promise((r) => setTimeout(r, 1500));
      await route.fulfill(json(view([]), 201));
    },
  });
  await fillForm(page, "Hỏi", "Câu hỏi sạch về Điều 5");
  const form = page.locator("[data-part=thread-form]");
  await form.getByRole("button", { name: "Đăng" }).click();
  await expect(form.getByText("Đang kiểm tra")).toBeVisible({ timeout: 1400 });
  await expect(form).toHaveCount(0); // đăng xong, form đóng
});
