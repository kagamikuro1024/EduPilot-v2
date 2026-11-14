import { expect, test, type Page } from "@playwright/test";
import { A1, SUB_B, assignmentsApi, defaultRows, gradingApi, resultView } from "./support/assignment-fixtures";
import { BASE_URL } from "./support/env";
import { asDemo } from "./support/session";

// FEAT-grading-guidance (10.6 L4): bảng «Lưu ý chấm» ở ngăn bài tập, hộp «bản cũ» ở hàng chờ, phụ đề ở màn duyệt. Gateway giả theo hợp đồng thật (openapi.yaml).
const desktopOnly = (info: { project: { name: string } }) => test.skip(info.project.name !== "desktop", "đo một lần");
const cors = { "Access-Control-Allow-Origin": BASE_URL, "Access-Control-Allow-Credentials": "true", Vary: "Origin" };
const json = (body: unknown, status = 200) => ({ status, contentType: "application/json", headers: cors, body: JSON.stringify(body) });
const ITEM = "00000000-0000-7000-8000-0000000a9001";

const item = (over: Record<string, unknown> = {}) => ({
  id: ITEM, criterion_id: "c1", text: "Chấm chặt hơn khi chỉ nêu tên tác nhân mà chưa phân tích.", text_original: null, direction: "STRICTER", support: 6, needs_recheck: false,
  status: "DRAFT", pending_retire: false, approved_in_version: null, retired_in_version: null, version: 1, created_at: "2026-11-14T03:00:00.000Z", ...over,
});
const overview = (over: Record<string, unknown> = {}) => ({
  enabled: true, current_version: 1, items: [item()], next_cursor: null, counts: { draft: 1 }, stale: { count: 0, editing: 0, reviewed: 0, versions: [] },
  reflect: { eligible: true, results: 8, edits: 7 }, ...over,
});

async function openDrawer(page: Page, context: Parameters<typeof asDemo>[0], ov: () => unknown, h: NonNullable<Parameters<typeof assignmentsApi>[1]>["h"] = {}) {
  await asDemo(context, "teacher");
  const api = await assignmentsApi(page, {
    h: {
      "GET assignments/:id/guidance": (r) => r.fulfill(json(ov())),
      "GET assignments/:id/guidance/stats": (r) => r.fulfill(json({ items: [] })),
      ...h,
    },
  });
  await page.goto("/grading?tab=assignments");
  await page.locator("[data-part=assignment-row]").first().click();
  return api;
}

test("ngăn bài tập: Duyệt → Chờ áp dụng → Áp dụng gửi expected_version; không câu mô tả", async ({ page, context }, info) => {
  desktopOnly(info);
  let cur = overview();
  const api = await openDrawer(page, context, () => cur, {
    [`POST assignments/:id/guidance/items/${ITEM}/decide`]: (r) => {
      cur = overview({ items: [item({ status: "APPROVED", version: 2 })], counts: { approved: 1 } });
      return r.fulfill(json(cur.items[0]));
    },
    "POST assignments/:id/guidance/apply": (r) => {
      cur = overview({ current_version: 2, items: [item({ status: "APPROVED", approved_in_version: 2, version: 2 })], counts: { approved: 1 } });
      return r.fulfill(json({ version: 2, stale: { count: 0, editing: 0, reviewed: 0, versions: [] } }));
    },
  });
  const panel = page.locator("[data-part=guidance-panel]");
  await expect(panel).toContainText("Bản v1");
  await expect(panel.locator("[data-part=guidance-item]")).toContainText("Chấm chặt hơn khi chỉ nêu tên tác nhân");
  await expect(panel.getByRole("button", { name: "Áp dụng" })).toBeDisabled();
  await panel.getByRole("button", { name: "Duyệt" }).click();
  await expect(panel.getByText("Chờ áp dụng")).toBeVisible();
  const decide = api.calls.find((c) => c.path.endsWith(`/items/${ITEM}/decide`))!;
  expect(JSON.parse(decide.body)).toEqual({ decision: "APPROVE", version: 1 });
  await panel.getByRole("button", { name: "Áp dụng (1)" }).click();
  await expect(panel.getByText("Đã áp dụng bản v2")).toBeVisible();
  expect(JSON.parse(api.calls.find((c) => c.path.endsWith("/guidance/apply"))!.body)).toEqual({ expected_version: 1 });
  await expect(panel.getByText("Bản v2", { exact: true })).toBeVisible();
  await expect(page.getByRole("status").filter({ hasText: "Thành công" })).toHaveCount(0);
});

test("ngăn bài tập: sửa chữ khi duyệt gửi text; 422 hiện lỗi tại chỗ; bài chấm bản cũ → Chấm lại", async ({ page, context }, info) => {
  desktopOnly(info);
  let bad = true;
  const stale = { count: 3, editing: 1, reviewed: 2, versions: [1] };
  const api = await openDrawer(page, context, () => overview({ stale }), {
    [`POST assignments/:id/guidance/items/${ITEM}/decide`]: (r) =>
      bad ? ((bad = false), r.fulfill(json({ code: "GUIDANCE_TEXT_REJECTED", message: "m", trace_id: "t" }, 422))) : r.fulfill(json(item({ status: "APPROVED", version: 2 }))),
    "POST assignments/:id/regrade-stale": (r) => r.fulfill(json({ job_id: "00000000-0000-7000-8000-0000000e0009", queued: 3, skipped: { reviewed: 2, edited: 1, max_round: 0 } }, 202)),
  });
  const panel = page.locator("[data-part=guidance-panel]");
  await expect(panel).toContainText("3 bài chấm theo bản cũ · 1 đang sửa dở");
  await panel.getByRole("button", { name: "Sửa" }).click();
  await panel.getByLabel("Chữ lưu ý").fill("Chấm chặt hơn, trừ nếu muộn.");
  await panel.getByRole("button", { name: "Duyệt" }).click();
  await expect(panel).toContainText("Chữ này không dùng được");
  await expect(panel.getByLabel("Chữ lưu ý")).toHaveValue("Chấm chặt hơn, trừ nếu muộn."); // chữ đã gõ không mất
  await panel.getByLabel("Chữ lưu ý").fill("Chấm chặt hơn khi chưa phân tích.");
  await panel.getByRole("button", { name: "Duyệt" }).click();
  await expect.poll(() => api.calls.filter((c) => c.path.endsWith(`/items/${ITEM}/decide`)).length).toBe(2);
  expect(JSON.parse(api.calls.filter((c) => c.path.endsWith(`/items/${ITEM}/decide`))[1].body)).toEqual({ decision: "APPROVE", version: 1, text: "Chấm chặt hơn khi chưa phân tích." });
  await panel.getByRole("button", { name: "Chấm lại" }).click();
  await expect.poll(() => api.calls.some((c) => c.path.endsWith("/regrade-stale"))).toBe(true);
});

test("lớp tắt tính năng: ngăn bài tập không có khối «Lưu ý chấm»", async ({ page, context }, info) => {
  desktopOnly(info);
  await openDrawer(page, context, () => overview({ enabled: false }));
  await expect(page.getByRole("dialog").getByLabel("Tiêu đề")).toBeVisible();
  await expect(page.locator("[data-part=guidance-panel]")).toHaveCount(0);
});

test("hàng chờ: «N bài chấm theo bản cũ» + Chấm lại; công bố bị chặn MIXED_GUIDANCE nói rõ", async ({ page, context }, info) => {
  desktopOnly(info);
  await asDemo(context, "teacher");
  const reviewed = defaultRows().map((r, i) => (i > 0 && r.result ? { ...r, result: { ...r.result, status: "REVIEWED", run_state: "DONE", needs_review: false } } : r));
  const api = await gradingApi(page, {
    rows: reviewed,
    h: {
      "GET assignments/:id/guidance": (r) => r.fulfill(json(overview({ stale: { count: 3, editing: 0, reviewed: 2, versions: [1] } }))),
      "POST assignments/:id/regrade-stale": (r) => r.fulfill(json({ job_id: "00000000-0000-7000-8000-0000000e0009", queued: 3, skipped: { reviewed: 2, edited: 0, max_round: 0 } }, 202)),
      "POST grading-results/publish": (r) => r.fulfill(json({ code: "MIXED_GUIDANCE", message: "m", trace_id: "t", details: { versions: { "1": 1 } } }, 409)),
    },
  });
  await page.goto("/grading?tab=queue");
  await page.getByLabel("Bài tập", { exact: true }).selectOption(A1);
  await expect(page.getByText("3 bài chấm theo bản cũ")).toBeVisible();
  await page.getByRole("button", { name: "Chấm lại" }).click();
  await expect(page.getByText("Đã xếp 3 bài chấm lại")).toBeVisible();
  expect(api.calls.some((c) => c.path.endsWith(`/assignments/${A1}/regrade-stale`))).toBe(true);

  await page.getByRole("button", { name: "Cần xem kỹ" }).click();
  await page.getByLabel("Trạng thái").selectOption("REVIEWED");
  await page.locator("[data-part=queue-row]").first().getByRole("checkbox").check();
  await page.getByRole("button", { name: "Công bố" }).click();
  await expect(page.getByRole("dialog")).toContainText("2 bài đã duyệt chấm theo bản lưu ý cũ");
  await page.getByRole("dialog").getByRole("button", { name: "Công bố" }).click();
  await expect(page.getByText(/Còn bài nháp chấm theo bản lưu ý cũ/)).toBeVisible();
  await expect(page.getByRole("dialog")).toHaveCount(0);
});

test("màn duyệt: «bản v1 · 1 lưu ý» mở danh sách theo tiêu chí; chip Bản cũ + Chấm lại xác nhận rồi gửi", async ({ page, context }, info) => {
  desktopOnly(info);
  await asDemo(context, "teacher");
  const v = resultView({ guidance: { version: 1, current_version: 2, stale: true, items_by_criterion: { c2: ["Chỉ cho điểm tối đa khi có ví dụ thực tế."] } } });
  const api = await gradingApi(page, { views: { [SUB_B]: v }, h: { "POST submissions/:id/grade": (r) => r.fulfill(json({ job_id: "00000000-0000-7000-8000-0000000e0009", queued: 1 }, 202)) } });
  await page.goto(`/grading/${SUB_B}?status=DRAFT&needs=1`);
  const meta = page.locator("[data-part=guidance-meta]");
  await expect(meta).toHaveText("bản v1 · 1 lưu ý");
  await meta.click();
  await expect(page.locator("[data-part=guidance-popover]")).toContainText("Phân tích tấn công");
  await expect(page.locator("[data-part=guidance-popover]")).toContainText("Chỉ cho điểm tối đa khi có ví dụ thực tế.");
  await page.keyboard.press("Escape");
  await expect(page.getByText("Bản cũ", { exact: true })).toBeVisible();
  await page.getByRole("button", { name: "Chấm lại" }).click();
  await page.getByRole("dialog").getByRole("button", { name: "Chấm lại" }).click();
  await expect.poll(() => api.calls.some((c) => c.method === "POST" && c.path.endsWith(`/submissions/${SUB_B}/grade`))).toBe(true);
});

test("màn duyệt: lớp tắt tính năng (không có khối guidance) → không phụ đề, không chip", async ({ page, context }, info) => {
  desktopOnly(info);
  await asDemo(context, "teacher");
  await gradingApi(page);
  await page.goto(`/grading/${SUB_B}?status=DRAFT&needs=1`);
  await expect(page.locator("[data-part=submission-text]")).toBeVisible();
  await expect(page.locator("[data-part=guidance-meta]")).toHaveCount(0);
  await expect(page.getByText("Bản cũ", { exact: true })).toHaveCount(0);
});
