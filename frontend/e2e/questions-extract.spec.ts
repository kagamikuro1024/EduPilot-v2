import { expect, test, type Route } from "@playwright/test";
import { BASE_URL } from "./support/env";
import { asDemo } from "./support/session";

// US-P9-02 + US-P9-04: `/questions` THẬT — Trích từ đề → việc nền → câu Chờ duyệt → chọn hàng → Duyệt hàng loạt (kết quả từng id). Gateway giả có trạng thái.
const cors = { "Access-Control-Allow-Origin": BASE_URL, "Access-Control-Allow-Credentials": "true", Vary: "Origin", "Access-Control-Allow-Headers": "*", "Access-Control-Allow-Methods": "*" };
const json = (body: unknown, status = 200) => ({ status, contentType: "application/json", headers: cors, body: JSON.stringify(body) });
const PAPER = "00000000-0000-7000-8000-0000000000d1";
const KEY = "00000000-0000-7000-8000-0000000000d2";
const JOB = "00000000-0000-7000-8000-0000000000a9";

test.beforeEach(async ({ context }, info) => {
  test.skip(info.project.name !== "desktop", "ca desktop");
  await asDemo(context, "teacher");
});

type Q = { id: string; review_status: "PENDING" | "APPROVED"; origin: string; title: string };
const row = (q: Q) => ({ id: q.id, type: "MCQ_SINGLE", title: q.title, topic: "Mật mã", difficulty: "MEDIUM", review_status: q.review_status, origin: q.origin, used_in_exams: 0, version: 1, archived_at: null, updated_at: "2026-10-01T00:00:00Z" });
const doc = (id: string, title: string, type: string) => ({ id, course_id: "c", title, type, filename: `${title}.pdf`, status: "READY", error: null, page_count: 6, visible_to_students: false, use_for_rag: true, category: null, week_no: null, version: 1, updated_at: "2026-10-01T00:00:00Z" });

test("trích từ đề → câu chờ duyệt → duyệt hàng loạt: tiến độ trong luồng, lý do bỏ qua, kết quả từng id, Idempotency-Key", async ({ page }) => {
  const qs: Q[] = [{ id: "q-old", review_status: "APPROVED", origin: "MANUAL", title: "Câu đã có" }];
  const calls: Array<{ method: string; path: string; body: string; idem: string }> = [];
  await page.route(/\/api\/v1\/courses\/[^/]+\/(questions|documents)/, async (route: Route) => {
    const req = route.request();
    if (req.method() === "OPTIONS") return route.fulfill({ status: 204, headers: cors });
    const url = new URL(req.url());
    const path = url.pathname.replace(/^.*\/courses\/[^/]+/, "");
    const body = req.postData() ?? "";
    calls.push({ method: req.method(), path, body, idem: req.headers()["idempotency-key"] ?? "" });
    if (path === "/documents" && req.method() === "GET") {
      const type = url.searchParams.get("type");
      return route.fulfill(json({ items: type === "EXAM_PAPER" ? [doc(PAPER, "Đề cuối kỳ 2025", type)] : type === "ANSWER_KEY" ? [doc(KEY, "Đáp án cuối kỳ 2025", type)] : [], next_cursor: null }));
    }
    if (path === "/questions" && req.method() === "GET") return route.fulfill(json({ items: qs.map(row), next_cursor: null }));
    if (path === "/questions/extract") return route.fulfill(json({ job_id: JOB }, 202));
    if (path === "/questions/bulk") {
      const b = JSON.parse(body) as { action: string; ids: string[] };
      return route.fulfill(json({ results: b.ids.map((id) => {
        if (id === "q-3") return { id, ok: false, code: "CITATION_REQUIRED" };
        qs.find((q) => q.id === id)!.review_status = "APPROVED";
        return { id, ok: true };
      }) }));
    }
    return route.fulfill(json({ code: "NOT_FOUND", message: "m", trace_id: "t" }, 404));
  });
  await page.route(/\/api\/v1\/jobs\/[^/]+$/, (route) => {
    for (const [i, id] of ["q-1", "q-2", "q-3"].entries()) if (!qs.some((q) => q.id === id)) qs.push({ id, review_status: "PENDING", origin: "EXTRACTED", title: `Câu trích ${i + 1}` });
    return route.fulfill(json({ id: JOB, kind: "question.extract", status: "SUCCEEDED", progress: 100, result: { created: ["q-1", "q-2", "q-3"], skipped: [{ number: 7, page: 2, reason: "NO_ANSWER_MATCH" }] } }));
  });

  await page.goto("/questions");
  await expect(page.getByRole("heading", { name: "Ngân hàng câu hỏi", level: 1 })).toBeVisible();
  await page.getByRole("button", { name: "Tạo câu hỏi" }).first().click();
  await page.getByRole("menuitem", { name: /Trích từ đề/ }).click();
  const form = page.getByRole("form", { name: "Trích câu hỏi từ đề cũ" });
  await expect(form.getByRole("button", { name: "Trích câu hỏi" })).toBeDisabled();
  await form.getByLabel("Đề cũ").selectOption({ label: "Đề cuối kỳ 2025" });
  await form.getByLabel("Đáp án").selectOption({ label: "Đáp án cuối kỳ 2025" });
  await form.getByRole("button", { name: "Trích câu hỏi" }).click();
  await expect(form.getByText("Đã tạo 3 câu chờ duyệt · bỏ qua 1 câu")).toBeVisible();
  await form.getByText("Xem lý do").click();
  await expect(form.getByText("Câu 7 (trang 2): Đáp án không khớp số câu")).toBeVisible();
  const extract = calls.find((c) => c.path === "/questions/extract")!;
  expect(extract.idem).not.toBe("");
  expect(JSON.parse(extract.body)).toEqual({ document_id: PAPER, answer_key_document_id: KEY });

  // chọn 3 câu mới → Duyệt: một câu thiếu trích dẫn ở lại vùng chọn kèm lý do, hai câu kia thành Đã duyệt
  for (const t of ["Câu trích 1", "Câu trích 2", "Câu trích 3"]) await page.getByRole("row", { name: new RegExp(t) }).getByRole("checkbox").check();
  await expect(page.getByRole("toolbar", { name: "Thao tác với câu đã chọn" })).toContainText("Đã chọn 3");
  await page.getByRole("toolbar").getByRole("button", { name: "Duyệt", exact: true }).click();
  await expect(page.getByRole("toolbar")).toContainText("Đã chọn 1");
  await expect(page.getByRole("row", { name: /Câu trích 3/ })).toContainText("Câu cần có nguồn trích dẫn trước khi duyệt.");
  await expect(page.getByRole("row", { name: /Câu trích 1/ })).toContainText("Đã duyệt");
  const bulk = calls.find((c) => c.path === "/questions/bulk")!;
  expect(bulk.idem).not.toBe("");
  expect(JSON.parse(bulk.body)).toEqual({ action: "APPROVE", ids: ["q-1", "q-2", "q-3"] });
});
