import { expect, test, type Route } from "@playwright/test";
import { BASE_URL } from "./support/env";
import { asDemo } from "./support/session";

// FEAT-grading-guidance M6 (10.6 L5): lý do loại câu hỏi chọn nhanh cạnh dòng «Hoàn tác»; chỉ hiện khi lớp bật «Nhớ lý do bỏ câu hỏi khi sinh đề». Gateway giả theo hợp đồng thật.
const cors = { "Access-Control-Allow-Origin": BASE_URL, "Access-Control-Allow-Credentials": "true", Vary: "Origin", "Access-Control-Allow-Headers": "*", "Access-Control-Allow-Methods": "*" };
const json = (body: unknown, status = 200) => ({ status, contentType: "application/json", headers: cors, body: JSON.stringify(body) });

test.beforeEach(async ({ context }, info) => {
  test.skip(info.project.name !== "desktop", "ca desktop");
  await asDemo(context, "teacher");
});

type Q = { id: string; review_status: "PENDING" | "APPROVED"; origin: string; title: string };
const row = (q: Q) => ({ id: q.id, type: "MCQ_SINGLE", title: q.title, topic: "Mật mã", difficulty: "MEDIUM", review_status: q.review_status, origin: q.origin, used_in_exams: 0, version: 1, archived_at: null, updated_at: "2026-10-01T00:00:00Z" });

async function open(page: import("@playwright/test").Page, hintsOn: boolean) {
  const bulks: Array<Record<string, unknown>> = [];
  const qs: Q[] = [{ id: "q-a", review_status: "PENDING", origin: "GENERATED", title: "Câu A" }];
  await page.route(/\/api\/v1\/courses\/[^/]+\/(questions|documents|guidance-flags)/, async (route: Route) => {
    const req = route.request();
    if (req.method() === "OPTIONS") return route.fulfill({ status: 204, headers: cors });
    const path = new URL(req.url()).pathname.replace(/^.*\/courses\/[^/]+/, "");
    if (path === "/documents") return route.fulfill(json({ items: [], next_cursor: null }));
    if (path === "/guidance-flags") return route.fulfill(json({ items: [{ feature: "QUESTION_HINTS", enabled: hintsOn, version: 1, can_edit: true }] }));
    if (path === "/questions" && req.method() === "GET") return route.fulfill(json({ items: qs.map(row), next_cursor: null }));
    if (path === "/questions/bulk") {
      const b = JSON.parse(req.postData() ?? "{}") as { ids: string[] };
      bulks.push(b);
      return route.fulfill(json({ results: b.ids.map((id) => ({ id, ok: true })) }));
    }
    return route.fulfill(json({ code: "NOT_FOUND", message: "m", trace_id: "t" }, 404));
  });
  await page.goto("/questions");
  await expect(page.getByRole("row", { name: /Câu A/ })).toBeVisible();
  await page.getByRole("row", { name: /Câu A/ }).getByRole("checkbox").check();
  await page.getByRole("toolbar").getByRole("button", { name: "Loại", exact: true }).click();
  return bulks;
}

test("lớp bật nhớ lý do: chọn «Mơ hồ» cạnh Hoàn tác → lệnh Loại gửi reason; bỏ chọn thì không có reason", async ({ page }) => {
  const bulks = await open(page, true);
  const group = page.getByRole("group", { name: "Lý do loại" });
  await expect(group.getByRole("button")).toHaveCount(6);
  await group.getByRole("button", { name: "Mơ hồ" }).click();
  await expect(group.getByRole("button", { name: "Mơ hồ" })).toHaveAttribute("aria-pressed", "true");
  await page.evaluate(() => window.dispatchEvent(new Event("pagehide"))); // gửi ngay thay vì chờ 5 s
  await expect.poll(() => bulks.length, { timeout: 2000 }).toBe(1);
  expect(bulks[0]).toEqual({ action: "REJECT", ids: ["q-a"], reason: "MO_HO" });
});

test("không chọn lý do: lệnh Loại như cũ (không reason)", async ({ page }) => {
  const bulks = await open(page, true);
  await expect(page.getByRole("group", { name: "Lý do loại" })).toBeVisible();
  await page.evaluate(() => window.dispatchEvent(new Event("pagehide")));
  await expect.poll(() => bulks.length, { timeout: 2000 }).toBe(1);
  expect(bulks[0]).toEqual({ action: "REJECT", ids: ["q-a"] });
});

test("lớp tắt: không có nhãn lý do", async ({ page }) => {
  await open(page, false);
  await expect(page.getByRole("button", { name: "Hoàn tác" })).toBeVisible();
  await expect(page.getByRole("group", { name: "Lý do loại" })).toHaveCount(0);
});
