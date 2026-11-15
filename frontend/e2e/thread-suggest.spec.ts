import { expect, test, type Page } from "@playwright/test";
import { asDemo } from "./support/session";
import { COURSE, json, precheckOk, row, threadApi } from "./support/thread-fixtures";

// FEAT-grading-guidance M4 (10.6 L5): gợi ý «câu tương tự đã xác nhận» dưới ô tiêu đề khi đặt câu hỏi. Chữ đang soạn đi trong THÂN (POST), không trong URL; không hiện gì khi trống / lỗi.
const T = `/courses/${COURSE}/threads`;
const SIM = `POST ${T}/similar-confirmed`;
const T1 = "00000000-0000-7000-8000-0000000000c1";

test.beforeEach(async ({}, info) => {
  test.skip(info.project.name !== "desktop", "ca desktop");
});

async function compose(page: Page, context: Parameters<typeof asDemo>[0], h: Parameters<typeof threadApi>[1], title = "Thi lại môn này", body = "Em được thi lại mấy lần trong học kỳ") {
  await asDemo(context, "student", { person: "sv-2" });
  const calls = await threadApi(page, { [`GET ${T}`]: json({ items: [row()], next_cursor: null }), [`POST ${T}/precheck`]: json(precheckOk), ...h });
  await page.goto("/threads");
  await page.getByRole("button", { name: "Đặt câu hỏi" }).first().click();
  await page.getByLabel(/Tiêu đề/).fill(title);
  await page.getByLabel(/Nội dung/).fill(body);
  return calls;
}

test("sau precheck sạch: ≤ 3 thread đã xác nhận dưới ô tiêu đề, là liên kết; chữ đi trong thân, URL không có chữ", async ({ page, context }) => {
  const calls = await compose(page, context, { [SIM]: json([{ thread_id: T1, title: "Thi lại được mấy lần?" }, { thread_id: "00000000-0000-7000-8000-0000000000c2", title: "Điều kiện thi lại" }]) });
  const list = page.locator("[data-part=similar-confirmed]");
  await expect(list.getByRole("link")).toHaveCount(2);
  await expect(list.getByRole("link", { name: "Thi lại được mấy lần?" })).toHaveAttribute("href", `/threads/${T1}`);
  const sim = calls.find((c) => c.path.endsWith("/threads/similar-confirmed"))!;
  expect(sim.method).toBe("POST");
  expect(JSON.parse(sim.body)).toEqual({ q: "Thi lại môn này\nEm được thi lại mấy lần trong học kỳ" });
  expect(sim.path).not.toContain("thi");
  // vị trí: sau ô tiêu đề, trước ô nội dung
  const order = await page.evaluate(() => {
    const a = document.querySelector("[data-part=similar-confirmed]")!;
    const body = document.querySelector("textarea")!;
    return a.compareDocumentPosition(body) & Node.DOCUMENT_POSITION_FOLLOWING;
  });
  expect(order).toBeTruthy();
});

test("trống / lỗi: không hiện gì, không báo lỗi", async ({ page, context }) => {
  await compose(page, context, { [SIM]: json([]) });
  await page.waitForTimeout(1200);
  await expect(page.locator("[data-part=similar-confirmed]")).toHaveCount(0);
  await page.unrouteAll({ behavior: "ignoreErrors" });
  await compose(page, context, { [SIM]: json({ code: "INTERNAL", message: "m", trace_id: "t" }, 500) });
  await page.waitForTimeout(1200);
  await expect(page.locator("[data-part=similar-confirmed]")).toHaveCount(0);
  await expect(page.locator("[data-part=thread-form] [role=alert]")).toHaveCount(0);
});

test("precheck phát hiện PII hoặc câu hỏi riêng tư: không gửi chữ tới similar-confirmed", async ({ page, context }) => {
  const calls = await compose(page, context, {
    [`POST ${T}/precheck`]: json({ allowed: false, reasons: [{ type: "MSSV", count: 1 }], redacted_text: "x", redacted_title: "", personal_question: false }),
    [SIM]: json([{ thread_id: T1, title: "Không được hiện" }]),
  });
  await expect(page.locator("[data-part=pii-notice]")).toBeVisible();
  await page.waitForTimeout(500);
  expect(calls.some((c) => c.path.endsWith("/similar-confirmed"))).toBe(false);
  await expect(page.locator("[data-part=similar-confirmed]")).toHaveCount(0);
});
