import { expect, test, type Route } from "@playwright/test";
import { BASE_URL } from "./support/env";
import { asDemo } from "./support/session";

// DEF-R10S-18 (TC-12-13): «N câu chờ duyệt» là số của cả lớp, không phải số hàng đã tải (trang đầu ≤ 30 hàng). Gateway giả có lọc + phân trang con trỏ.
const cors = { "Access-Control-Allow-Origin": BASE_URL, "Access-Control-Allow-Credentials": "true", Vary: "Origin", "Access-Control-Allow-Headers": "*", "Access-Control-Allow-Methods": "*" };
const json = (body: unknown, status = 200) => ({ status, contentType: "application/json", headers: cors, body: JSON.stringify(body) });

test.beforeEach(async ({ context }) => { await asDemo(context, "teacher"); });

test("câu chờ duyệt: đếm theo cả lớp (43), không theo trang đã tải", async ({ page }) => {
  const all = Array.from({ length: 53 }, (_, i) => ({
    id: `q-${i}`, type: "MCQ_SINGLE", title: `Câu ${i}`, topic: "Mật mã", difficulty: "MEDIUM", review_status: i < 43 ? "PENDING" : "APPROVED", origin: "EXTRACTED",
    used_in_exams: 0, version: 1, archived_at: null, updated_at: "2026-11-01T00:00:00Z", created_at: `2026-11-01T00:00:${String(59 - (i % 60)).padStart(2, "0")}Z`,
  }));
  await page.route(/\/api\/v1\/courses\/[^/]+\/questions(\?.*)?$/, (route: Route) => {
    const req = route.request();
    if (req.method() === "OPTIONS") return route.fulfill({ status: 204, headers: cors });
    const u = new URL(req.url());
    const rs = u.searchParams.get("review_status");
    const rows = all.filter((q) => !rs || q.review_status === rs);
    const limit = Number(u.searchParams.get("limit") ?? 30);
    const from = Number(u.searchParams.get("cursor") ?? 0);
    const items = rows.slice(from, from + limit);
    return route.fulfill(json({ items, next_cursor: from + limit < rows.length ? String(from + limit) : null }));
  });
  await page.goto("/questions");
  await expect(page.getByText("43 câu chờ duyệt")).toBeVisible();
});
