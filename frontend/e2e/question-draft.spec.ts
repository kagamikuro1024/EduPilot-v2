import { expect, test, type Page, type Route } from "@playwright/test";
import { BASE_URL } from "./support/env";
import { asDemo } from "./support/session";

// DEF-R5-03 (nháp câu hỏi phải khôi phục sau khi mở lại) và DEF-R5-04 (Enter trên hàng mở Drawer, không đóng ngay).
const cors = { "Access-Control-Allow-Origin": BASE_URL, "Access-Control-Allow-Credentials": "true", Vary: "Origin", "Access-Control-Allow-Headers": "*", "Access-Control-Allow-Methods": "*" };
const json = (body: unknown, status = 200) => ({ status, contentType: "application/json", headers: cors, body: JSON.stringify(body) });
const ROW = { id: "q-a", type: "MCQ_SINGLE", title: "Câu A", topic: "Mật mã", difficulty: "MEDIUM", review_status: "PENDING", origin: "AI_DRAFT", used_in_exams: 0, version: 1, archived_at: null, updated_at: "2026-10-01T00:00:00Z" };
const DETAIL = {
  ...ROW,
  used_in_exams: [],
  stem: "Đề gốc của máy chủ",
  explanation: null,
  options: [0, 1].map((i) => ({ id: `o${i}`, position: i, body: `Đáp án ${i}`, pinned_last: false })),
  answer_key: { option_ids: ["o0"] },
  citations: [],
};

test.beforeEach(async ({ context, page }, info) => {
  test.skip(info.project.name !== "desktop", "ca desktop");
  await asDemo(context, "teacher");
  await page.route(/\/api\/v1\/courses\/[^/]+\/(questions|documents|guidance-flags)/, async (route: Route) => {
    const req = route.request();
    if (req.method() === "OPTIONS") return route.fulfill({ status: 204, headers: cors });
    const path = new URL(req.url()).pathname.replace(/^.*\/courses\/[^/]+/, "");
    if (path === "/documents") return route.fulfill(json({ items: [], next_cursor: null }));
    if (path === "/guidance-flags") return route.fulfill(json({ items: [] }));
    if (path === "/questions") return route.fulfill(json({ items: [ROW], next_cursor: null }));
    if (path === "/questions/q-a") return route.fulfill(json(DETAIL));
    return route.fulfill(json({ code: "NOT_FOUND", message: "m", trace_id: "t" }, 404));
  });
});

const rowA = (page: Page) => page.getByRole("row", { name: /Câu A/ });

test("sửa câu hỏi: chữ gõ dở còn nguyên và có dòng «Đã khôi phục» khi mở lại sau khi tải lại trang (R5-03)", async ({ page }) => {
  await page.goto("/questions");
  await rowA(page).click();
  await page.getByRole("button", { name: "Chỉnh sửa" }).click();
  const stem = page.getByLabel("Đề bài");
  await expect(stem).toHaveValue("Đề gốc của máy chủ");
  await stem.fill("Đề đang sửa dở");
  await page.evaluate(() => window.dispatchEvent(new Event("pagehide"))); // ghi ngay thay vì chờ nhịp 2 s
  await page.reload();
  await rowA(page).click();
  await page.getByRole("button", { name: "Chỉnh sửa" }).click();
  await expect(page.getByText("Đã khôi phục phần bạn đang sửa dở.")).toBeVisible();
  await expect(page.getByLabel("Đề bài")).toHaveValue("Đề đang sửa dở");
});

test("sửa câu hỏi chưa có nháp: giá trị của máy chủ, không có dòng «Đã khôi phục»", async ({ page }) => {
  await page.goto("/questions");
  await rowA(page).click();
  await page.getByRole("button", { name: "Chỉnh sửa" }).click();
  await expect(page.getByLabel("Đề bài")).toHaveValue("Đề gốc của máy chủ");
  await page.getByLabel("Đề bài").fill("Gõ mới");
  await expect(page.getByLabel("Đề bài")).toHaveValue("Gõ mới");
  await expect(page.getByText("Đã khôi phục phần bạn đang sửa dở.")).toHaveCount(0);
});
