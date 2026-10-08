import { expect, test, type Page, type Route } from "@playwright/test";
import { settleGoto } from "./support/hydrate";
import { loadAudit, runAudit } from "./support/audit";
import { BASE_URL } from "./support/env";
import { asJwt } from "./support/session";

// US-PE-03 AC13 — `/questions` thật (ngân hàng câu hỏi). Gateway giả bằng page.route; bản chạy với gateway thật là `@real` (không chạy ở CI).

test.beforeEach(({ page }) => settleGoto(page));

const C1 = "00000000-0000-7000-8000-00000000c001";
const cors = { "Access-Control-Allow-Origin": BASE_URL, "Access-Control-Allow-Credentials": "true", Vary: "Origin" };

type Json = Record<string, unknown>;
type Q = Json & { id: string; title: string; topic: string; type: string; difficulty: string; review_status: string; origin: string; version: number; stem: string; options: Json[]; answer_key: Json | null; explanation: string | null; used_in_exams: Json[]; archived_at: null; code?: Json };

function fakeBank() {
  let n = 100;
  const tests: Record<string, Json[]> = {};
  const calls = { put: [] as string[], review: [] as string[], verify: 0, importDry: 0, importReal: 0, suggest: 0, statusFilter: [] as string[] };
  const mk = (over: Partial<Q>): Q => ({
    id: `q-${++n}`, title: "Câu", topic: "Mật mã", type: "MCQ_SINGLE", difficulty: "MEDIUM", review_status: "DRAFT", origin: "MANUAL", version: 1, stem: "Đề", options: [], answer_key: null, explanation: null,
    used_in_exams: [], archived_at: null, reviewed_by: null, reviewed_at: null, ai_job_id: null, created_at: "2026-10-01T00:00:00Z", updated_at: "2026-10-01T00:00:00Z", ...over,
  });
  const opt = (id: string, body: string, position: number) => ({ id, position, body, pinned_last: false });
  const qs: Q[] = [
    mk({ id: "q-1", title: "AES là thuật toán gì?", stem: "Đề: <script>alert(1)</script> và **AES** dùng khoá `128` bit", review_status: "APPROVED", version: 3, options: [opt("o1", "Đối xứng", 1), opt("o2", "Bất đối xứng", 2)], answer_key: { option_ids: ["o1"] }, used_in_exams: [{ id: "e-1", title: "Kiểm tra tuần 9", status: "SCHEDULED" }] }),
    mk({ id: "q-2", title: "RSA dựa trên bài toán nào?", topic: "Mật mã bất đối xứng", difficulty: "HARD", review_status: "PENDING", origin: "AI_DRAFT", options: [opt("o3", "Phân tích thừa số", 1), opt("o4", "Logarit rời rạc", 2)], answer_key: { option_ids: ["o3"] } }),
    mk({ id: "q-3", title: "Tính a+b", topic: "Cơ bản", type: "CODE", stem: "Đọc hai số, in tổng.", code: undefined }),
  ];
  const code = (): Json => ({ languages: ["cpp17"], time_limit_ms: 1000, memory_limit_mb: 256, output_limit_kb: 1024, checker: "EXACT", float_eps: null, starter_code: {}, reference: null, reference_verified_version: null, reference_verified_at: null, tests_version: 1, tests: { total: 0, samples: 0, hidden: 0, total_weight: 0 } });
  qs[2].code = code();
  const row = (q: Q) => ({ id: q.id, type: q.type, title: q.title, topic: q.topic, difficulty: q.difficulty, review_status: q.review_status, origin: q.origin, used_in_exams: q.used_in_exams.length, version: q.version, archived_at: null, updated_at: q.updated_at });
  const stats = (q: Q) => {
    const t = (tests[q.id] ??= []).filter((x) => x.approved);
    (q.code as Json).tests = { total: t.length, samples: t.filter((x) => x.is_sample).length, hidden: t.filter((x) => !x.is_sample).length, total_weight: t.reduce((a, x) => a + (x.weight as number), 0) };
  };
  const json = (route: Route, body: unknown, status = 200) => route.fulfill({ status, contentType: "application/json", headers: cors, body: JSON.stringify(body) });
  const err = (route: Route, status: number, code: string, message: string, details?: unknown) => json(route, { code, message, trace_id: "t", details }, status);

  async function install(page: Page) {
    await page.route(/\/api\/v1\/jobs\/[^/]+$/, (route) => {
      const id = route.request().url().split("/").pop()!;
      if (id === "job-verify") return json(route, { id, kind: "code.verify_reference", status: "SUCCEEDED", progress: 100, result: { ok: true, compile_ok: true, tests_version: 2, per_test: (tests["q-3"] ?? []).map((t) => ({ test_id: t.id, name: t.name, verdict: "AC", time_ms: 3 })) } });
      return json(route, { id, kind: "question.suggest", status: "SUCCEEDED", progress: 100, result: { kind: "MCQ", requested: 2, created: ["q-900"], dropped: [] } });
    });
    await page.route(/\/api\/v1\/courses\/[^/]+\/questions/, async (route) => {
      const req = route.request();
      const url = new URL(req.url());
      const path = url.pathname.replace(/^.*\/questions/, "");
      const m = req.method();
      const body = (() => {
        try {
          return req.postDataJSON() as Json;
        } catch {
          return undefined; // multipart (nhập zip)
        }
      })();
      const parts = path.split("/").filter(Boolean);
      const q = parts[0] && parts[0] !== "suggest" ? qs.find((x) => x.id === parts[0]) : undefined;
      if (m === "GET" && parts.length === 0) {
        const st = url.searchParams.get("review_status");
        if (st) calls.statusFilter.push(st);
        const text = url.searchParams.get("q")?.toLowerCase();
        return json(route, { items: qs.filter((x) => (!st || x.review_status === st) && (!text || x.title.toLowerCase().includes(text))).map(row), next_cursor: null });
      }
      if (m === "POST" && parts.length === 0) {
        const b = body as Json;
        if (b.type !== "CODE" && b.type !== "TRUE_FALSE" && (b.correct as number[]).length === 0) return err(route, 422, "VALIDATION_FAILED", "Dữ liệu chưa hợp lệ.", [{ field: "correct", code: "NO_CORRECT_OPTION", message: "Cần chọn ít nhất một đáp án đúng." }]);
        const opts = ((b.options as Json[]) ?? []).map((o, i) => opt(`n${n}-${i}`, String(o.body), i + 1));
        const created = mk({ title: String(b.title), topic: String(b.topic), type: String(b.type), stem: String(b.stem), options: opts, answer_key: b.type === "CODE" ? null : { option_ids: ((b.correct as number[]) ?? []).map((i) => opts[i].id) }, code: b.type === "CODE" ? code() : undefined });
        qs.unshift(created);
        return json(route, created, 201);
      }
      if (!q && parts[0] === "suggest") {
        calls.suggest++;
        qs.unshift(mk({ id: "q-900", title: "Câu AI vừa soạn", origin: "AI_DRAFT", review_status: "PENDING" }));
        return json(route, { job_id: "job-suggest" }, 202);
      }
      if (!q) return err(route, 404, "NOT_FOUND", "Không tìm thấy.");
      const sub = parts[1];
      if (m === "GET" && !sub) {
        if (q.code) stats(q);
        return json(route, q);
      }
      if (m === "PUT" && !sub) {
        calls.put.push(String(q.id));
        if (q.used_in_exams.length > 0 && body?.stem !== q.stem) return err(route, 409, "QUESTION_IN_USE", "Câu hỏi đang được dùng.", { exams: q.used_in_exams });
        Object.assign(q, { title: body?.title, topic: body?.topic, stem: body?.stem, version: q.version + 1 });
        return json(route, q);
      }
      if (sub === "review") {
        calls.review.push(String(body?.decision));
        if (body?.decision === "APPROVE" && q.type === "CODE") {
          const c = q.code as Json;
          if (c.reference_verified_version !== c.tests_version) return err(route, 422, "VALIDATION_FAILED", "Dữ liệu chưa hợp lệ.", [{ field: "reference", code: "REFERENCE_NOT_VERIFIED", message: "Chạy lời giải mẫu qua bộ test hiện tại trước khi duyệt." }]);
        }
        q.review_status = { REQUEST: "PENDING", APPROVE: "APPROVED", REJECT: "REJECTED" }[String(body?.decision)] as string;
        q.version++;
        return json(route, q);
      }
      if (sub === "code" && m === "PUT") {
        const c = q.code as Json;
        Object.assign(c, { languages: body?.languages, time_limit_ms: body?.time_limit_ms, memory_limit_mb: body?.memory_limit_mb, reference: body?.reference ?? null, tests_version: (c.tests_version as number) + 1, reference_verified_version: null });
        q.version++;
        stats(q);
        return json(route, q);
      }
      if (sub === "testcases" && !parts[2]) {
        if (m === "GET") return json(route, { items: tests[q.id] ?? [], next_cursor: null });
        const t = { id: `t-${++n}`, position: (tests[q.id] ??= []).length + 1, name: body?.name, is_sample: body?.is_sample, weight: body?.weight, input: body?.input, input_truncated: false, expected: body?.expected, expected_truncated: false, input_bytes: 1, expected_bytes: 1, source: "MANUAL", approved: true };
        tests[q.id].push(t);
        (q.code as Json).tests_version = ((q.code as Json).tests_version as number) + 1;
        return json(route, t, 201);
      }
      if (parts[2] === "import") {
        const dry = url.searchParams.get("dry_run") === "true";
        if (dry) {
          calls.importDry++;
          return json(route, { would_create: 2, created: 0, errors: [] });
        }
        calls.importReal++;
        const list = (tests[q.id] ??= []);
        for (const [name, sample] of [["sample1", true], ["t2", false]] as const) list.push({ id: `t-${++n}`, position: list.length + 1, name, is_sample: sample, weight: 1, input: "1", input_truncated: false, expected: "1", expected_truncated: false, input_bytes: 1, expected_bytes: 1, source: "IMPORT", approved: true });
        (q.code as Json).tests_version = ((q.code as Json).tests_version as number) + 1;
        return json(route, { would_create: 2, created: 2, errors: [] }, 201);
      }
      if (parts[1] === "reference") {
        calls.verify++;
        const c = q.code as Json;
        c.reference_verified_version = c.tests_version;
        return json(route, { job_id: "job-verify" }, 202);
      }
      return err(route, 404, "NOT_FOUND", "Không tìm thấy.");
    });
  }
  return { install, calls, qs };
}

async function setup(page: Page, role: "TEACHER" | "TA" = "TEACHER") {
  await asJwt(page, role);
  const course = { id: C1, class_code: "761987", subject_code: "INT1006", name: "An ninh mạng", semester: "2026-2027-HK1", status: "ACTIVE" };
  await page.route("**/api/v1/me/courses**", (r) => r.fulfill({ status: 200, contentType: "application/json", headers: cors, body: JSON.stringify({ items: [{ course, role_in_course: role, enrollment_status: "ACTIVE" }], next_cursor: null }) }));
  await page.route("**/api/v1/me/today**", (r) => r.fulfill({ status: 200, contentType: "application/json", headers: cors, body: JSON.stringify({ count: 0, actions: [] }) }));
  await page.route("**/api/v1/notifications**", (r) => r.fulfill({ status: 200, contentType: "application/json", headers: cors, body: JSON.stringify({ items: [], next_cursor: null, unread_count: 0 }) }));
  const bank = fakeBank();
  await bank.install(page);
  return bank;
}

/** Dòng danh sách: ≥ 720 px là bảng, < 720 px là danh sách — chỉ một bản hiển thị, nên chọn bản đang thấy. */
const row = (page: Page, text: string) => page.locator("main").getByText(text).locator("visible=true").first();

const createMenu = (page: Page, item: string) => async () => {
  await page.getByRole("button", { name: "Tạo câu hỏi" }).first().click();
  await page.getByRole("menuitem", { name: item }).click();
};

test("questions bank: bảng, bộ lọc, tạo MCQ (lỗi tại ô → sửa → tạo), câu AI có nhãn, câu đã dùng bị khoá", async ({ page }) => {
  const bank = await setup(page);
  await page.goto("/questions");
  await expect(page.getByRole("heading", { name: "Ngân hàng câu hỏi" })).toBeVisible();
  const table = page.getByRole("table", { name: /Ngân hàng câu hỏi lớp 761987/ });
  await expect(row(page, "AES là thuật toán gì?")).toBeVisible();
  await expect(row(page, "RSA dựa trên bài toán nào?")).toBeVisible();
  await expect(page.getByText("Chờ duyệt").first()).toBeVisible();
  void table;

  // bộ lọc trạng thái gọi API với review_status
  await page.getByLabel("Trạng thái").selectOption("PENDING");
  await expect(page.getByText("AES là thuật toán gì?")).toHaveCount(0);
  await expect(row(page, "RSA dựa trên bài toán nào?")).toBeVisible();
  expect(bank.calls.statusFilter).toContain("PENDING");
  await page.getByLabel("Trạng thái").selectOption("");

  // câu AI: nhãn nguồn chỉ hiện cho Staff
  await row(page, "RSA dựa trên bài toán nào?").click();
  await expect(page.getByRole("dialog")).toContainText("AI soạn nháp");
  await expect(page.getByRole("button", { name: "Duyệt", exact: true })).toBeVisible();
  await page.getByRole("button", { name: "Đóng" }).click();

  // tạo MCQ: thiếu đáp án đúng → lỗi ngay tại ô, giữ nguyên chữ đã gõ
  await createMenu(page, "Câu trắc nghiệm")();
  const dlg = page.getByRole("dialog", { name: "Câu trắc nghiệm mới" });
  await dlg.getByLabel("Tiêu đề").fill("Hàm băm nào an toàn?");
  await dlg.getByLabel("Chủ đề").fill("Hàm băm");
  await dlg.getByLabel("Đề bài").fill("Chọn hàm băm còn an toàn.");
  await dlg.getByLabel("Nội dung đáp án 1").fill("MD5");
  await dlg.getByLabel("Nội dung đáp án 2").fill("SHA-256");
  await dlg.getByRole("button", { name: "Tạo câu hỏi" }).click();
  await expect(dlg.getByRole("alert")).toContainText("Chọn đúng số đáp án đúng");
  await expect(dlg.getByLabel("Tiêu đề")).toHaveValue("Hàm băm nào an toàn?");
  await dlg.getByLabel("Đáp án 2 là đáp án đúng").check();
  await dlg.getByRole("button", { name: "Tạo câu hỏi" }).click();
  await expect(page.getByRole("dialog", { name: "Hàm băm nào an toàn?" })).toBeVisible();
  await expect(page.getByRole("dialog")).toContainText("Nháp");
  await page.getByRole("button", { name: "Đóng" }).click();
  await expect(row(page, "Hàm băm nào an toàn?")).toBeVisible();

  // câu đang dùng trong bài thi: đề HTML hiện thành chữ, thông báo khoá, sửa đề → 409 giữ chữ
  await row(page, "AES là thuật toán gì?").click();
  const d = page.getByRole("dialog", { name: "AES là thuật toán gì?" });
  await expect(d.locator("[data-part=markdown]")).toContainText("<script>alert(1)</script>");
  expect(await d.locator("[data-part=markdown] script").count()).toBe(0);
  await expect(d.locator("[data-part=markdown] strong")).toHaveText("AES");
  await expect(d).toContainText("Đang dùng trong bài thi Kiểm tra tuần 9 — nhân bản để sửa");
  await d.getByRole("button", { name: "Chỉnh sửa" }).click();
  await d.getByLabel("Đề bài").fill("Đề đã sửa");
  await d.getByRole("button", { name: "Lưu thay đổi" }).click();
  await expect(d.getByRole("alert").or(d.getByText("Câu hỏi đang được dùng trong bài thi"))).toBeVisible();
  await expect(d.getByLabel("Đề bài")).toHaveValue("Đề đã sửa");
});

test("questions bank: bài lập trình — cấu hình, nhập zip (kiểm trước), thêm test, chạy lời giải mẫu, duyệt", async ({ page }) => {
  const bank = await setup(page);
  await page.goto("/questions");
  await row(page, "Tính a+b").click();
  const d = page.getByRole("dialog", { name: "Tính a+b" });
  // chưa kiểm lời giải mẫu → Duyệt bị máy chủ từ chối kèm lý do tiếng Việt
  await d.getByRole("button", { name: "Duyệt", exact: true }).click();
  await expect(d).toContainText("Chạy lời giải mẫu qua bộ test hiện tại trước khi duyệt.");
  // cấu hình + lời giải mẫu
  await d.getByRole("textbox", { name: "Lời giải mẫu" }).fill("int main(){}");
  await d.getByRole("button", { name: "Lưu cấu hình" }).click();
  await expect(d.getByText(/Chưa kiểm với bộ test v2/)).toBeVisible();
  // zip: kiểm trước rồi nhập
  await d.getByLabel("Tệp zip chứa test").setInputFiles({ name: "tests.zip", mimeType: "application/zip", buffer: Buffer.from("PK") });
  await d.getByRole("button", { name: "Kiểm tra zip" }).click();
  await expect(d).toContainText("Kiểm tra xong: sẽ thêm 2 test.");
  expect(bank.calls.importDry).toBe(1);
  await d.getByRole("button", { name: "Nhập zip" }).click();
  await expect(d).toContainText("Đã thêm 2 test.");
  await expect(d.getByText("sample1")).toBeVisible();
  // thêm một test tay
  await d.getByRole("button", { name: "Thêm test" }).click();
  await d.getByLabel("Tên test").fill("t3");
  await d.getByLabel("Đầu vào").fill("3 4");
  await d.getByLabel("Đầu ra mong đợi").fill("7");
  await d.getByRole("button", { name: "Thêm test" }).last().click();
  await expect(d.getByText("t3")).toBeVisible();
  // chạy lời giải mẫu → bảng verdict từng test
  await d.getByRole("button", { name: "Chạy lời giải mẫu" }).click();
  await expect(d.getByText("Mọi test đều đúng — lời giải mẫu đã được kiểm.")).toBeVisible();
  await expect(d.getByRole("table", { name: "Kết quả từng test" })).toContainText("Đúng");
  expect(bank.calls.verify).toBe(1);
  // duyệt
  await d.getByRole("button", { name: "Duyệt", exact: true }).click();
  await expect(d).toContainText("Đã duyệt");
  expect(bank.calls.review).toEqual(["APPROVE", "APPROVE"]);
});

test("questions bank: Gợi ý từ AI soạn nháp → câu ở trạng thái Chờ duyệt, dòng thông báo tại chỗ", async ({ page }) => {
  const bank = await setup(page, "TA");
  await page.goto("/questions");
  await createMenu(page, "Gợi ý từ AI")();
  const form = page.getByRole("form", { name: "Gợi ý câu hỏi từ AI" });
  await form.getByLabel("Chủ đề").fill("Mật mã đối xứng");
  await form.getByRole("button", { name: "Soạn nháp" }).click();
  await expect(page.getByText(/AI đã soạn 1 câu nháp · đang chờ bạn duyệt/)).toBeVisible();
  await expect(row(page, "Câu AI vừa soạn")).toBeVisible();
  expect(bank.calls.suggest).toBe(1);
});

test("questions bank: trạng thái rỗng và lỗi có Thử lại", async ({ page }) => {
  await asJwt(page, "TEACHER");
  await page.route("**/api/v1/me/courses**", (r) => r.fulfill({ status: 200, contentType: "application/json", headers: cors, body: JSON.stringify({ items: [{ course: { id: C1, class_code: "761987", subject_code: "INT1006", name: "An ninh mạng", semester: "2026-2027-HK1", status: "ACTIVE" }, role_in_course: "TEACHER", enrollment_status: "ACTIVE" }], next_cursor: null }) }));
  let fail = true;
  await page.route(/\/api\/v1\/courses\/[^/]+\/questions/, (route) => {
    const body = fail ? { code: "INTERNAL", message: "x", trace_id: "t" } : { items: [], next_cursor: null };
    return route.fulfill({ status: fail ? 500 : 200, contentType: "application/json", headers: cors, body: JSON.stringify(body) });
  });
  await page.goto("/questions");
  await expect(page.getByRole("button", { name: "Thử lại" })).toBeVisible();
  fail = false;
  await page.getByRole("button", { name: "Thử lại" }).click();
  await expect(page.getByText("Chưa có câu hỏi nào. Tạo câu đầu tiên hoặc nhờ AI gợi ý nháp.")).toBeVisible();
  await expect(page.getByRole("button", { name: "Tạo câu hỏi" }).first()).toBeVisible();
});

test("questions bank: bố cục ở 1440 / 1024 / 390 — không tràn ngang, không cắt chữ", async ({ page }) => {
  const { AUDIT_SRC } = await loadAudit();
  await setup(page);
  for (const width of [1440, 1024, 390]) {
    await page.setViewportSize({ width, height: 900 });
    await page.goto("/questions");
    await expect(row(page, "AES là thuật toán gì?")).toBeVisible();
    const a = await runAudit(page, AUDIT_SRC);
    expect(a.ox, `ox @${width}`).toBe(0);
    expect(a.cut, `cut @${width}`).toEqual([]);
    expect(a.ell, `ell @${width}`).toEqual([]);
  }
});

// ===== US-PE-04: `/exams` — soạn, lên lịch, gia hạn (Giảng viên / TA) và danh sách của sinh viên =====

type E = Json & { id: string; title: string; status: string; effective_status: string; version: number; items: Json[]; opens_at: string | null; closes_at: string | null; duration_minutes: number | null };

function fakeExams(bank: ReturnType<typeof fakeBank>) {
  let n = 0;
  const exams: E[] = [];
  const calls = { create: [] as Json[], put: [] as Json[], items: [] as Json[], schedule: 0, unschedule: 0, extend: [] as string[], preview: 0, del: 0 };
  const mk = (title: string, over: Partial<E> = {}): E => {
    const e = {
      id: `e-${++n}`, title, kind: "MCQ", status: "DRAFT", effective_status: "DRAFT", opens_at: null, closes_at: null, duration_minutes: null, items_count: 0, attempts: { started: 0, graded: 0 }, published_at: null,
      version: 1, created_at: `2026-10-0${Math.min(n, 9)}T00:00:00Z`, instructions: null, shuffle_questions: true, shuffle_options: true, max_score: "10.00", rounding_step: "0.01", multi_scoring: "PARTIAL",
      reveal_answers: true, appeal_days: 7, publish_hold: false, regrading: false, items: [], created_by: "u", updated_at: "2026-10-01T00:00:00Z", ...over,
    } as E;
    exams.push(e);
    return e;
  };
  const open = mk("Giữa kỳ", { status: "OPEN", effective_status: "OPEN", opens_at: "2026-12-01T01:00:00Z", closes_at: "2026-12-01T03:00:00Z", duration_minutes: 60, items_count: 5, attempts: { started: 12, graded: 0 } });
  const json = (route: Route, body: unknown, status = 200) => route.fulfill({ status, contentType: "application/json", headers: cors, body: JSON.stringify(body) });
  const err = (route: Route, status: number, code: string, message: string, details?: unknown) => json(route, { code, message, trace_id: "t", details }, status);
  const sync = (e: E) => {
    e.items_count = e.items.length;
    e.version++;
  };
  async function install(page: Page) {
    await page.route(/\/api\/v1\/courses\/[^/]+\/exams/, async (route) => {
      const req = route.request();
      const url = new URL(req.url());
      const parts = url.pathname.replace(/^.*\/exams/, "").split("/").filter(Boolean);
      const m = req.method();
      const body = (() => {
        try {
          return req.postDataJSON() as Json;
        } catch {
          return undefined;
        }
      })();
      const e = parts[0] ? exams.find((x) => x.id === parts[0]) : undefined;
      if (m === "GET" && parts.length === 0) return json(route, { items: exams, next_cursor: null });
      if (m === "POST" && parts.length === 0) {
        calls.create.push(body ?? {});
        return json(route, mk(String(body?.title)), 201);
      }
      if (!e) return err(route, 404, "NOT_FOUND", "Không tìm thấy.");
      const sub = parts[1];
      if (m === "GET" && !sub) return json(route, e);
      if (m === "DELETE" && !sub) {
        calls.del++;
        exams.splice(exams.indexOf(e), 1);
        return route.fulfill({ status: 204, headers: cors });
      }
      if (m === "PUT" && !sub) {
        calls.put.push(body ?? {});
        if (body?.version !== e.version) return err(route, 409, "VERSION_CONFLICT", "Phiên bản đã đổi.", { current_version: e.version, current: e });
        for (const k of ["title", "instructions", "opens_at", "closes_at", "duration_minutes", "reveal_answers", "appeal_days"]) if (k in (body ?? {})) e[k] = body![k];
        e.version++;
        return json(route, e);
      }
      if (m === "PUT" && sub === "items") {
        calls.items.push(body ?? {});
        const want = (body?.items as Array<{ question_id: string; points: string }>) ?? [];
        e.items = want.map((it, i) => {
          const q = bank.qs.find((x) => x.id === it.question_id)!;
          return { id: `i-${i}`, question_id: q.id, position: i + 1, points: it.points, type: q.type, title: q.title, topic: q.topic, difficulty: q.difficulty, review_status: q.review_status, archived: false };
        });
        sync(e);
        return json(route, e);
      }
      if (sub === "preview") {
        calls.preview++;
        const items = e.items.map((it, i) => {
          const q = bank.qs.find((x) => x.id === it.question_id)!;
          return { item_id: String(it.id), position: i + 1, type: q.type, points: String(it.points), stem: q.stem, options: (q.options as Array<{ id: string; body: string }>).map((o) => ({ id: o.id, body: o.body })), code: null };
        });
        return json(route, { preview: true, exam: { id: e.id, title: e.title, instructions: e.instructions, kind: "MCQ", opens_at: e.opens_at, closes_at: e.closes_at, duration_minutes: e.duration_minutes, max_score: "10.00", status: "DRAFT", my_attempt: null, my_score: null }, items });
      }
      if (sub === "schedule") {
        calls.schedule++;
        const problems: Json[] = [];
        if (!e.opens_at) problems.push({ field: "opens_at", code: "VALUE_REQUIRED", message: "Cần giờ mở bài." });
        if (!e.closes_at) problems.push({ field: "closes_at", code: "VALUE_REQUIRED", message: "Cần giờ đóng bài." });
        if (!e.duration_minutes) problems.push({ field: "duration_minutes", code: "VALUE_REQUIRED", message: "Cần thời lượng làm bài." });
        if (e.items.length === 0) problems.push({ field: "items", code: "NO_ITEMS", message: "Bài thi cần ít nhất một câu hỏi." });
        if (problems.length) return err(route, 422, "VALIDATION_FAILED", "Dữ liệu chưa hợp lệ.", problems);
        Object.assign(e, { status: "SCHEDULED", effective_status: "SCHEDULED", version: e.version + 1 });
        return json(route, e);
      }
      if (sub === "clone") {
        const c = mk(`${e.title} (bản sao)`, { kind: e.kind, items: [...e.items], items_count: e.items_count });
        return json(route, c, 201);
      }
      if (sub === "unschedule") {
        calls.unschedule++;
        Object.assign(e, { status: "DRAFT", effective_status: "DRAFT", version: e.version + 1 });
        return json(route, e);
      }
      if (sub === "extend") {
        calls.extend.push(String(body?.closes_at));
        if (new Date(String(body?.closes_at)) <= new Date(String(e.closes_at))) return err(route, 422, "VALIDATION_FAILED", "Dữ liệu chưa hợp lệ.", [{ field: "closes_at", code: "CLOSES_NOT_LATER", message: "Giờ đóng mới phải muộn hơn giờ đóng hiện tại." }]);
        e.closes_at = String(body?.closes_at);
        e.version++;
        return json(route, e);
      }
      return err(route, 404, "NOT_FOUND", "Không tìm thấy.");
    });
  }
  return { install, calls, exams, open };
}

async function setupExams(page: Page, role: "TEACHER" | "TA") {
  const bank = await setup(page, role);
  bank.qs[1].review_status = "APPROVED"; // hai câu đã duyệt để thử thứ tự
  const ex = fakeExams(bank);
  await ex.install(page);
  return { bank, ex };
}

test("exam editor: tạo bài, lỗi lên lịch có danh sách, chọn câu, đổi thứ tự, xem trước, lên lịch, hoàn tác, gia hạn", async ({ page }, info) => {
  test.skip(info.project.name !== "desktop", "trang soạn bài từ 720 px (SRS 7.1)");
  const { ex } = await setupExams(page, "TEACHER");
  await page.goto("/exams");
  await expect(page.getByRole("heading", { name: "Bài thi", level: 1 })).toBeVisible();
  await expect(page.getByRole("heading", { name: "Đang mở", level: 2 })).toBeVisible();
  await expect(page.getByText("Giữa kỳ")).toBeVisible();

  // tạo bài nháp chỉ bằng tiêu đề
  await page.getByRole("button", { name: "Tạo bài thi" }).click();
  const form = page.getByRole("form", { name: "Bài thi mới" });
  await form.getByLabel("Tiêu đề").fill("Kiểm tra tuần 9");
  await form.getByRole("button", { name: "Tạo bản nháp" }).click();
  await expect(page.getByRole("heading", { name: "Kiểm tra tuần 9", level: 1 })).toBeVisible();
  expect(ex.calls.create).toEqual([{ title: "Kiểm tra tuần 9" }]);

  // lên lịch khi thiếu giờ và câu → MỌI việc cần sửa hiện một lần, có liên kết tới đúng chỗ
  await page.getByRole("button", { name: "Lên lịch" }).click();
  const todo = page.getByRole("list", { name: "Việc cần sửa" });
  await expect(page.getByText("Cần sửa 4 việc trước khi lên lịch")).toBeVisible();
  await expect(todo.getByRole("listitem")).toHaveCount(4);
  await expect(todo).toContainText("Cần giờ mở bài.");
  await expect(todo).toContainText("Bài thi cần ít nhất một câu hỏi.");
  await todo.getByRole("button", { name: "Thêm câu hỏi" }).click();
  await expect(page.getByRole("tab", { name: /^Câu hỏi/ })).toHaveAttribute("aria-selected", "true");

  // chọn câu từ ngân hàng (chỉ câu đã duyệt), đổi thứ tự bằng nút Lên / Xuống, lưu
  await page.getByRole("button", { name: "Thêm câu từ ngân hàng" }).click();
  const drawer = page.getByRole("dialog", { name: "Chọn câu từ ngân hàng" });
  await expect(drawer.getByText("Tính a+b")).toHaveCount(0); // câu code chưa duyệt không có trong danh sách
  await drawer.getByRole("checkbox", { name: /AES là thuật toán gì\?/ }).check();
  await drawer.getByRole("checkbox", { name: /RSA dựa trên bài toán nào\?/ }).check();
  await drawer.getByRole("button", { name: "Thêm 2 câu" }).click();
  await page.getByRole("button", { name: "Đưa câu AES là thuật toán gì? xuống" }).click();
  await expect(page.getByRole("list", { name: "Câu hỏi của bài thi" }).getByRole("listitem").first()).toContainText("RSA dựa trên bài toán nào?");
  await page.getByLabel("Điểm của câu 1").fill("2,5");
  await page.getByRole("button", { name: "Lưu danh sách câu" }).click();
  await expect(page.getByText("Đã lưu danh sách câu hỏi")).toBeVisible();
  expect(ex.calls.items[0]).toMatchObject({ items: [{ question_id: "q-2", points: "2.50" }, { question_id: "q-1", points: "1.00" }] });
  await expect(page.getByText(/Tổng điểm các câu: 3,50/)).toBeVisible();

  // thông tin: giờ theo giờ Việt Nam, thời lượng
  await page.getByRole("tab", { name: "Thông tin" }).click();
  await page.getByLabel("Mở lúc").fill("2030-12-01T09:00");
  await page.getByLabel("Đóng lúc").fill("2030-12-01T10:30");
  await page.getByLabel("Thời lượng (phút)").fill("45");
  await page.getByRole("button", { name: "Lưu thông tin" }).click();
  await expect(page.getByText("Đã lưu thông tin bài thi")).toBeVisible();
  expect(ex.calls.put.at(-1)).toMatchObject({ opens_at: "2030-12-01T02:00:00.000Z", closes_at: "2030-12-01T03:30:00.000Z", duration_minutes: 45 });

  // xem trước: HTML thô là chữ, không có đáp án đúng
  await page.getByRole("tab", { name: "Xem trước" }).click();
  await expect(page.getByText("Chỉ xem: bài như sinh viên sẽ thấy")).toBeVisible();
  await expect(page.getByText("<script>alert(1)</script>")).toBeVisible();
  expect(await page.locator("main script").count()).toBe(0);
  await page.getByRole("button", { name: "Xáo lại" }).click();
  await expect.poll(() => ex.calls.preview).toBeGreaterThanOrEqual(2);

  // bố cục sạch ở 1440 và 1024
  const { AUDIT_SRC } = await loadAudit();
  for (const width of [1440, 1024]) {
    await page.setViewportSize({ width, height: 900 });
    const a = await runAudit(page, AUDIT_SRC);
    expect(a, `audit @${width}`).toEqual({ ox: 0, cut: [], ell: [] });
  }

  // lên lịch thành công: không hộp thoại xác nhận; dòng "Hoàn tác" tại chỗ; nút chính biến mất
  await page.getByRole("button", { name: "Lên lịch" }).click();
  await expect(page.getByText(/Đã lên lịch · mở .*Sinh viên sẽ nhận thông báo/)).toBeVisible();
  await expect(page.getByRole("button", { name: "Lên lịch" })).toHaveCount(0);
  await expect(page.getByRole("dialog")).toHaveCount(0);
  await expect(page.getByText("Sắp tới").first()).toBeVisible();
  await page.getByRole("button", { name: "Hoàn tác" }).click();
  await expect.poll(() => ex.calls.unschedule).toBe(1);
  await expect(page.getByRole("button", { name: "Lên lịch" })).toBeVisible();

  // bài đang mở: nút chính là Gia hạn; mốc nhanh +30 phút; giờ đóng phải muộn hơn
  await page.goto(`/exams/${ex.open.id}`);
  await expect(page.getByRole("button", { name: "Gia hạn" }).first()).toBeVisible();
  await expect(page.getByRole("button", { name: "Lên lịch" })).toHaveCount(0);
  await page.getByRole("button", { name: "Gia hạn" }).first().click();
  const extend = page.getByRole("form", { name: "Gia hạn bài thi" });
  await extend.getByLabel("Giờ đóng mới").fill("2026-12-01T10:00"); // = đóng cũ (03:00Z) → không muộn hơn
  await extend.getByRole("button", { name: "Gia hạn" }).click();
  await expect(extend.getByText("Giờ đóng mới phải muộn hơn giờ đóng hiện tại.")).toBeVisible();
  await extend.getByRole("button", { name: "+30 phút" }).click();
  await extend.getByRole("button", { name: "Gia hạn" }).click();
  await expect(page.getByText(/Đã gia hạn đến 10:30 01\/12/)).toBeVisible();
  expect(ex.calls.extend.at(-1)).toBe("2026-12-01T03:30:00.000Z");
});

test("exam editor: TA không có nút Lên lịch, kèm câu giải thích; nhân bản và xoá nháp", async ({ page }, info) => {
  test.skip(info.project.name !== "desktop", "trang soạn bài từ 720 px (SRS 7.1)");
  const { ex } = await setupExams(page, "TA");
  await page.goto("/exams");
  await page.getByRole("button", { name: "Tạo bài thi" }).click();
  await page.getByRole("form", { name: "Bài thi mới" }).getByLabel("Tiêu đề").fill("Nháp của TA");
  await page.getByRole("button", { name: "Tạo bản nháp" }).click();
  await expect(page.getByRole("heading", { name: "Nháp của TA", level: 1 })).toBeVisible();
  await expect(page.getByRole("button", { name: "Lên lịch" })).toHaveCount(0);
  await expect(page.getByText("Chỉ giảng viên lên lịch được")).toBeVisible();
  // TA không xoá được; nhân bản được
  await page.getByRole("button", { name: "Thêm hành động cho bài thi" }).click();
  await expect(page.getByRole("menuitem", { name: /Xoá bài thi nháp/ })).toHaveCount(0);
  await page.getByRole("menuitem", { name: /Nhân bản bài thi/ }).click();
  await expect(page.getByRole("heading", { name: "Nháp của TA (bản sao)", level: 1 })).toBeVisible();
  expect(ex.exams.length).toBe(3);
});

test("exam editor: giảng viên xoá bài nháp — hộp xác nhận nêu số câu", async ({ page }, info) => {
  test.skip(info.project.name !== "desktop", "trang soạn bài từ 720 px (SRS 7.1)");
  const { ex } = await setupExams(page, "TEACHER");
  await page.goto("/exams");
  await page.getByRole("button", { name: "Tạo bài thi" }).click();
  await page.getByRole("form", { name: "Bài thi mới" }).getByLabel("Tiêu đề").fill("Bỏ đi");
  await page.getByRole("button", { name: "Tạo bản nháp" }).click();
  await page.getByRole("button", { name: "Thêm hành động cho bài thi" }).click();
  await page.getByRole("menuitem", { name: /Xoá bài thi nháp/ }).click();
  await expect(page.getByRole("dialog")).toContainText("Bài thi nháp có 0 câu sẽ bị xoá.");
  await page.getByRole("button", { name: "Xoá bài thi" }).click();
  await expect(page.getByRole("heading", { name: "Bài thi", level: 1 })).toBeVisible();
  expect(ex.calls.del).toBe(1);
});

test("exam list: trạng thái rỗng và lỗi có Thử lại", async ({ page }) => {
  await asJwt(page, "TEACHER");
  const course = { id: C1, class_code: "761987", subject_code: "INT1006", name: "An ninh mạng", semester: "2026-2027-HK1", status: "ACTIVE" };
  await page.route("**/api/v1/me/courses**", (r) => r.fulfill({ status: 200, contentType: "application/json", headers: cors, body: JSON.stringify({ items: [{ course, role_in_course: "TEACHER", enrollment_status: "ACTIVE" }], next_cursor: null }) }));
  let fail = true;
  await page.route(/\/api\/v1\/courses\/[^/]+\/exams/, (route) => {
    const body = fail ? { code: "INTERNAL", message: "x", trace_id: "t" } : { items: [], next_cursor: null };
    return route.fulfill({ status: fail ? 500 : 200, contentType: "application/json", headers: cors, body: JSON.stringify(body) });
  });
  await page.goto("/exams");
  await expect(page.getByRole("button", { name: "Thử lại" })).toBeVisible();
  fail = false;
  await page.getByRole("button", { name: "Thử lại" }).click();
  await expect(page.getByText("Chưa có bài thi nào. Tạo bài thi đầu tiên từ ngân hàng câu hỏi.")).toBeVisible();
});

type SV = { id: string; title: string; status: string; my_attempt: Json | null; my_score: string | null; opens_at: string; closes_at: string };
function studentRows(): SV[] {
  const base = { instructions: null, kind: "MCQ", duration_minutes: 45, max_score: "10.00" };
  const row = (id: string, title: string, status: string, my_attempt: Json | null, my_score: string | null, o: string, c: string): SV => ({ ...base, id, title, status, my_attempt, my_score, opens_at: o, closes_at: c }) as SV;
  return [
    row("s-1", "Tuần 9", "OPEN", null, null, "2026-12-01T01:00:00Z", "2026-12-01T03:00:00Z"),
    row("s-2", "Giữa kỳ", "OPEN", { id: "a-1", status: "IN_PROGRESS", deadline_at: "2026-12-01T02:00:00Z", submitted_at: null }, null, "2026-12-01T01:00:00Z", "2026-12-01T03:00:00Z"),
    row("s-3", "Tuần 10", "SCHEDULED", null, null, "2026-12-08T01:00:00Z", "2026-12-08T03:00:00Z"),
    row("s-4", "Tuần 8", "PUBLISHED", { id: "a-2", status: "GRADED", deadline_at: "2026-11-24T02:00:00Z", submitted_at: "2026-11-24T01:50:00Z" }, "8.50", "2026-11-24T01:00:00Z", "2026-11-24T03:00:00Z"),
    row("s-5", "Tuần 7", "CLOSED", { id: "a-3", status: "GRADING", deadline_at: "2026-11-17T02:00:00Z", submitted_at: "2026-11-17T01:50:00Z" }, null, "2026-11-17T01:00:00Z", "2026-11-17T03:00:00Z"),
  ];
}

async function studentSetup(page: Page, rows: SV[]) {
  await asJwt(page, "STUDENT");
  const course = { id: C1, class_code: "761987", subject_code: "INT1006", name: "An ninh mạng", semester: "2026-2027-HK1", status: "ACTIVE" };
  await page.route("**/api/v1/me/courses**", (r) => r.fulfill({ status: 200, contentType: "application/json", headers: cors, body: JSON.stringify({ items: [{ course, role_in_course: "STUDENT", enrollment_status: "ACTIVE" }], next_cursor: null }) }));
  await page.route("**/api/v1/me/today**", (r) => r.fulfill({ status: 200, contentType: "application/json", headers: cors, body: JSON.stringify({ no_course: false, email_verified: true, recommended: null, timeline: [], continue: [] }) }));
  await page.route("**/api/v1/notifications**", (r) => r.fulfill({ status: 200, contentType: "application/json", headers: cors, body: JSON.stringify({ items: [], next_cursor: null, unread_count: 0 }) }));
  await page.route(/\/api\/v1\/courses\/[^/]+\/exams/, (route) => route.fulfill({ status: 200, contentType: "application/json", headers: cors, body: JSON.stringify({ items: rows, next_cursor: null }) }));
}

test("student exam list: mỗi hàng một hành động, nhóm theo trạng thái, không tràn ngang; ở 375 px vùng chạm ≥ 44 px", async ({ page }, info) => {
  await studentSetup(page, studentRows());
  await page.goto("/exams");
  await expect(page.getByRole("heading", { name: "Bài thi", level: 1 })).toBeVisible();
  for (const g of ["Đang mở", "Sắp tới", "Đã có điểm", "Đã đóng"]) await expect(page.getByRole("heading", { name: g, level: 2 })).toBeVisible();
  const row = (t: string) => page.getByRole("listitem").filter({ hasText: t });
  await expect(row("Tuần 9").getByRole("link", { name: "Bắt đầu làm bài" })).toHaveAttribute("href", "/exams/s-1/take");
  await expect(row("Giữa kỳ").getByRole("link", { name: "Tiếp tục" })).toHaveAttribute("href", "/exams/s-2/take");
  await expect(row("Tuần 8").getByRole("link", { name: "Xem kết quả" })).toHaveAttribute("href", "/exams/s-4/take");
  await expect(row("Tuần 8")).toContainText("Điểm 8,50 / 10,00");
  await expect(row("Tuần 10").getByRole("link")).toHaveCount(0); // sắp tới: chưa có hành động
  await expect(row("Tuần 7").getByRole("link")).toHaveCount(0);
  await expect(row("Tuần 7")).toContainText("Đã nộp — điểm hiện khi giảng viên công bố");
  // không lộ chữ kỹ thuật
  expect(await page.locator("main").innerText()).not.toMatch(/judge|sandbox|verdict|DRAFT|PUBLISHED/i);
  await expect.poll(() => page.evaluate(() => document.documentElement.scrollWidth - window.innerWidth), { timeout: 5000 }).toBeLessThanOrEqual(0);
  const { AUDIT_SRC, TOUCH_SRC } = await loadAudit();
  const a = await runAudit(page, AUDIT_SRC);
  expect(a).toEqual({ ox: 0, cut: [], ell: [] });
  if (info.project.name === "mobile") expect(await page.evaluate(TOUCH_SRC)).toEqual([]); // vùng chạm chỉ đo ở dự án 375 px
});

test("student exam list: trạng thái rỗng", async ({ page }) => {
  await studentSetup(page, []);
  await page.goto("/exams");
  await expect(page.getByText("Lớp của bạn chưa có bài thi nào.")).toBeVisible();
});

test("exam route access: sinh viên không mở được trang soạn bài; Staff không mở được trang làm bài", async ({ page }) => {
  await studentSetup(page, []);
  await page.goto("/exams/e-1");
  await expect(page.getByText("Trang này dành cho giảng viên và trợ giảng.")).toBeVisible();
  await page.unrouteAll({ behavior: "ignoreErrors" });
  await setupExams(page, "TEACHER");
  await page.goto("/exams/e-1/take");
  await expect(page.getByText("Trang này dành cho sinh viên.")).toBeVisible();
});
