import { expect, test, type Page, type Route } from "@playwright/test";
import { settleGoto } from "./support/hydrate";
import AxeBuilder from "@axe-core/playwright";
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

// ───────── US-PE-05 — làm bài trắc nghiệm `/exams/[id]/take` (gateway giả; trạng thái dùng chung giữa các tab của một test) ─────────
const E1 = "e-take";
const OPT = (q: string, n: number) => Array.from({ length: n }, (_, i) => ({ id: `${q}-o${i + 1}`, body: `Đáp án ${"ABCD"[i]} của ${q}` }));
const TAKE_ITEMS = [
  { item_id: "q1", position: 1, type: "MCQ_SINGLE", points: "4.00", stem: "Câu hỏi một", options: OPT("q1", 3), code: null, answer: null as Json | null },
  { item_id: "q2", position: 2, type: "MCQ_MULTI", points: "4.00", stem: "Câu hỏi hai", options: OPT("q2", 4), code: null, answer: null as Json | null },
  { item_id: "q3", position: 3, type: "TRUE_FALSE", points: "2.00", stem: "Câu hỏi ba", options: [], code: null, answer: null as Json | null },
];

type CodeSample = { name: string; verdict: string; time_ms: number; memory_kb: number; input?: string; expected?: string; got?: string };
type CodeOut = { compile_ok: boolean; compile_log?: string; samples: CodeSample[] };
type CodeSub = { id: string; language: string; source: string; created_at: string; polls: number };
type CodeState = { drafts: Record<string, { source: string; rev: number; at: string }>; runs: Record<string, { language: string; polls: number }>; subs: CodeSub[]; runOut: CodeOut; subOut: CodeOut; pollsBeforeDone: number; runStatus: number; submitStatus: number };
type TakeState = { code: CodeState | null; started: boolean; submitted: null | { at: string; reason: string }; writer: string | null; answers: Record<string, Json>; skewMs: number; durationMs: number; deadline: number; calls: Array<{ m: string; url: string; tab: string | null; key: string | null; body: unknown }>; offlineSave: boolean; graceMs: number; events: Json[]; instructions: string | null };
function codeItem(c: CodeState) {
  const langs = Object.keys(c.drafts);
  const latest = langs.sort((a, b) => c.drafts[b].at.localeCompare(c.drafts[a].at))[0] ?? null;
  return {
    item_id: "c1", position: 1, type: "CODE", points: "10.00", stem: "Đọc hai số **a**, **b** và in tổng.", options: [], answer: null,
    code: {
      languages: ["c11", "cpp17"], time_limit_ms: 1000, memory_limit_mb: 256, starter_code: { cpp17: "// khởi đầu\n" },
      samples: [{ name: "sample1", input: "1 2\n", expected: "3\n" }], language: latest,
      drafts: Object.fromEntries(Object.entries(c.drafts).map(([l, d]) => [l, { source: d.source, rev: d.rev, saved_at: d.at }])),
    },
  };
}
const newCode = (): CodeState => ({
  drafts: {}, runs: {}, subs: [], pollsBeforeDone: 1, runStatus: 202, submitStatus: 202,
  runOut: { compile_ok: true, samples: [{ name: "sample1", verdict: "AC", time_ms: 3, memory_kb: 1024 }] },
  subOut: { compile_ok: true, samples: [{ name: "sample1", verdict: "AC", time_ms: 3, memory_kb: 1024 }] },
});

function fakeTake(over: Partial<TakeState> = {}) {
  const st: TakeState = { code: null, started: false, submitted: null, writer: null, answers: {}, skewMs: 0, durationMs: 45 * 60_000, deadline: 0, calls: [], offlineSave: false, graceMs: 0, events: [], instructions: null, ...over };
  const srvNow = () => Date.now() + st.skewMs;
  const runningView = (tab: string | null) => ({
    attempt: { id: "a-take", exam_id: E1, status: "IN_PROGRESS", started_at: new Date(st.deadline - st.durationMs).toISOString(), deadline_at: new Date(st.deadline).toISOString(), server_time: new Date(srvNow()).toISOString(), writer: { is_you: !!tab && tab === st.writer } },
    exam: { id: E1, title: "Tuần 9", instructions: null, kind: st.code ? "CODE" : "MCQ", duration_minutes: 45, closes_at: "2036-12-01T03:00:00Z", multi_scoring: "PARTIAL" },
    items: st.code ? [codeItem(st.code)] : TAKE_ITEMS.map((it) => ({ ...it, answer: st.answers[it.item_id] ?? null })),
  });
  const expireIfDue = () => {
    if (st.started && !st.submitted && srvNow() >= st.deadline + st.graceMs) st.submitted = { at: new Date(st.deadline).toISOString(), reason: "TIMEOUT" };
  };
  const json = (route: Route, status: number, body: unknown) => route.fulfill({ status, contentType: "application/json", headers: cors, body: JSON.stringify(body) });
  const err = (route: Route, status: number, code: string, details: Json = {}) => json(route, status, { code, message: code, details, trace_id: "t" });
  async function install(page: Page) {
    await page.route(new RegExp(`/api/v1/courses/${C1}/exams/${E1}/attempts`), async (route) => {
      const req = route.request();
      if (req.method() === "OPTIONS") return route.fulfill({ status: 204, headers: { ...cors, "Access-Control-Allow-Headers": "*", "Access-Control-Allow-Methods": "*" } });
      const h = req.headers();
      const tab = h["x-exam-tab"] ?? null;
      const path = new URL(req.url()).pathname.split("/attempts")[1] || "";
      st.calls.push({ m: req.method(), url: path, tab, key: h["idempotency-key"] ?? null, body: req.postDataJSON?.() ?? null });
      expireIfDue();
      if (req.method() === "POST" && path.endsWith("/events")) {
        st.events.push(...((req.postDataJSON() as { events: Json[] }).events ?? []));
        st.calls.push({ m: "EVENTS", url: path, tab, key: null, body: req.postData() });
        return route.fulfill({ status: 204, headers: cors });
      }
      if (st.code && /\/(code|runs|submissions)(\/|$)/.test(path) && (await codeRoutes(route, req, path, tab))) return;
      if (req.method() === "GET" && path === "/mine") {
        if (!st.started) return json(route, 200, { attempt: null, exam: { id: E1, title: "Tuần 9", instructions: st.instructions, kind: st.code ? "CODE" : "MCQ", duration_minutes: 45, max_score: "10.00", status: "OPEN", opens_at: "2026-12-01T01:00:00Z", closes_at: "2036-12-01T03:00:00Z", my_attempt: null, my_score: null } });
        if (st.submitted) return json(route, 200, { attempt: { id: "a-take", status: "GRADED", submitted_at: st.submitted.at, submit_reason: st.submitted.reason }, exam: { id: E1, title: "Tuần 9", closes_at: "2036-12-01T03:00:00Z", status: "OPEN" } });
        return json(route, 200, runningView(tab));
      }
      if (req.method() === "POST" && path === "") {
        st.started = true;
        st.writer = tab;
        st.deadline ||= srvNow() + st.durationMs;
        return json(route, 201, runningView(tab));
      }
      if (req.method() === "PUT" && path.endsWith("/answers")) {
        if (st.offlineSave) return route.abort("failed");
        if (st.submitted) return err(route, 409, "ATTEMPT_CLOSED");
        if (tab !== st.writer) return err(route, 409, "ATTEMPT_OTHER_TAB", { writer_seen_at: new Date().toISOString() });
        for (const it of (req.postDataJSON() as { items: Array<{ item_id: string; answer: Json }> }).items) st.answers[it.item_id] = it.answer;
        return json(route, 200, { saved_at: new Date().toISOString(), server_time: new Date(srvNow()).toISOString(), deadline_at: new Date(st.deadline).toISOString() });
      }
      if (req.method() === "POST" && path.endsWith("/takeover")) {
        st.writer = tab;
        return json(route, 200, runningView(tab).attempt);
      }
      if (req.method() === "POST" && path.endsWith("/submit")) {
        st.submitted = { at: new Date().toISOString(), reason: "MANUAL" };
        return json(route, 200, { status: "GRADED", submitted_at: st.submitted.at, answered: Object.keys(st.answers).length, total: 3 });
      }
      return err(route, 404, "NOT_FOUND");
    });
  }
  async function codeRoutes(route: Route, req: ReturnType<Route["request"]>, path: string, tab: string | null): Promise<boolean> {
    const c = st.code!;
    const m = req.method();
    const view = (id: string, status: string, language: string, at: string, out: CodeOut) => ({ id, status, language, created_at: at, compile_ok: status === "DONE" ? out.compile_ok : null, ...(status === "DONE" && out.compile_log ? { compile_log: out.compile_log } : {}), samples: status === "DONE" && out.compile_ok ? out.samples : [] });
    const writeGuard = () => (st.submitted ? err(route, 409, "ATTEMPT_CLOSED") : tab !== st.writer ? err(route, 409, "ATTEMPT_OTHER_TAB", { writer_seen_at: new Date().toISOString() }) : null);
    if (m === "PUT" && path.endsWith("/draft")) {
      const g = writeGuard();
      if (g) return void (await g), true;
      const b = req.postDataJSON() as { language: string; source: string; base_rev: number };
      const cur = c.drafts[b.language];
      if ((cur?.rev ?? 0) !== b.base_rev) return void (await err(route, 409, "DRAFT_CONFLICT", { current_rev: cur?.rev ?? 0, updated_at: cur?.at ?? null })), true;
      const at = new Date().toISOString();
      c.drafts[b.language] = { source: b.source, rev: (cur?.rev ?? 0) + 1, at };
      await json(route, 200, { rev: c.drafts[b.language].rev, saved_at: at, server_time: new Date(srvNow()).toISOString(), deadline_at: new Date(st.deadline).toISOString() });
      return true;
    }
    if (m === "POST" && path.endsWith("/run")) {
      const g = writeGuard();
      if (g) return void (await g), true;
      if (c.runStatus !== 202) return void (await err(route, c.runStatus, c.runStatus === 429 ? "RATE_LIMITED" : "JUDGE_UNAVAILABLE", {})), true;
      const b = req.postDataJSON() as { language: string };
      const id = `run-${Object.keys(c.runs).length + 1}`;
      c.runs[id] = { language: b.language, polls: 0 };
      await json(route, 202, { run_id: id });
      return true;
    }
    const runM = /\/runs\/([^/]+)$/.exec(path);
    if (m === "GET" && runM) {
      const r = c.runs[runM[1]];
      if (!r) return void (await err(route, 404, "NOT_FOUND")), true;
      r.polls++;
      await json(route, 200, view(runM[1], r.polls > c.pollsBeforeDone ? "DONE" : "RUNNING", r.language, new Date().toISOString(), c.runOut));
      return true;
    }
    if (m === "POST" && path.endsWith("/submit")) {
      const g = writeGuard();
      if (g) return void (await g), true;
      if (c.submitStatus !== 202) return void (await err(route, c.submitStatus, c.submitStatus === 429 ? "RATE_LIMITED" : "SUBMISSION_LIMIT_REACHED", {})), true;
      const b = req.postDataJSON() as { language: string; source: string };
      const sub = { id: `sub-${c.subs.length + 1}`, language: b.language, source: b.source, created_at: new Date().toISOString(), polls: 0 };
      c.subs.push(sub);
      await json(route, 202, { submission_id: sub.id });
      return true;
    }
    const subView = (x: CodeSub, full: boolean) => {
      const done = x.polls > c.pollsBeforeDone;
      return { ...view(x.id, done ? "DONE" : "RUNNING", x.language, x.created_at, c.subOut), is_final: x === c.subs[c.subs.length - 1], ...(full ? { source: x.source } : {}) };
    };
    if (m === "GET" && path.endsWith("/submissions")) {
      c.subs.forEach((x) => x.polls++);
      await json(route, 200, { items: [...c.subs].reverse().map((x) => subView(x, false)), next_cursor: null });
      return true;
    }
    const subM = /\/submissions\/([^/]+)$/.exec(path);
    if (m === "GET" && subM) {
      const x = c.subs.find((y) => y.id === subM[1]);
      if (!x) return void (await err(route, 404, "NOT_FOUND")), true;
      x.polls++;
      await json(route, 200, subView(x, true));
      return true;
    }
    return false;
  }
  return { st, install };
}
async function takeSetup(page: Page, f: ReturnType<typeof fakeTake>) {
  await studentSetup(page, []);
  await f.install(page); // đăng ký SAU studentSetup: route đăng ký sau được khớp trước
}
const confirmBox = (page: Page) => page.getByRole("dialog");
const timer = (page: Page) => page.getByRole("timer");
const startExam = async (page: Page) => {
  await page.goto(`/exams/${E1}/take?course=${C1}`);
  await expect(page.getByText("Bạn có 45 phút. Đồng hồ chạy ngay khi bạn bấm Bắt đầu và không dừng lại nếu bạn thoát.")).toBeVisible();
  await page.locator("[data-part=integrity-notice]").scrollIntoViewIfNeeded(); // câu minh bạch phải hiện trước khi Bắt đầu bấm được
  await page.getByRole("button", { name: "Bắt đầu làm bài" }).click();
  await expect(page.getByText("Câu 1/3")).toBeVisible();
};

test("take: clock skew — máy lệch 5 phút vẫn hiện đúng thời gian còn lại", async ({ page }) => {
  const f = fakeTake({ skewMs: 5 * 60_000 }); // máy chủ nhanh hơn máy khách 5 phút
  await takeSetup(page, f);
  await startExam(page);
  await expect(timer(page)).toHaveText(/^(45:00|44:5\d)$/);
});

test("take: timeout autosubmit — hết giờ máy chủ đã nộp, màn hình báo 'Hết giờ'", async ({ page }) => {
  const f = fakeTake({ durationMs: 3000 });
  await takeSetup(page, f);
  await startExam(page);
  await page.getByLabel("Đáp án A của q1").check();
  await expect(page.getByText(/^Hết giờ — bài của bạn đã được nộp lúc \d\d:\d\d:\d\d\.$/)).toBeVisible({ timeout: 15_000 });
  await expect(page.getByText(/Điểm sẽ hiện khi bài thi đóng với cả lớp/)).toBeVisible();
  expect(await page.locator("#main").innerText()).not.toMatch(/đúng|sai|\d+,\d+ \/ /i); // không lộ điểm hay đúng / sai
});

test("take: submit confirm and after — hộp xác nhận nêu số câu; nộp xong chỉ còn tóm tắt", async ({ page }) => {
  const f = fakeTake();
  await takeSetup(page, f);
  await startExam(page);
  await page.getByLabel("Đáp án B của q1").check();
  await page.getByRole("button", { name: "Nộp bài" }).first().click();
  await expect(confirmBox(page).getByText("Bạn đã trả lời 1/3 câu. Còn 2 câu chưa trả lời. Sau khi nộp bạn không sửa được.")).toBeVisible();
  await confirmBox(page).getByRole("button", { name: "Làm tiếp" }).click();
  await expect(confirmBox(page)).toBeHidden();
  await page.getByRole("button", { name: "Nộp bài" }).first().click();
  await confirmBox(page).getByRole("button", { name: "Nộp bài" }).click();
  await expect(page.getByText(/^Đã nộp lúc \d\d:\d\d:\d\d\.$/)).toBeVisible();
  await expect(page.getByText(/Điểm sẽ hiện khi bài thi đóng với cả lớp/)).toBeVisible();
  await expect(page.getByRole("radio")).toHaveCount(0); // đề không còn hiện
  const submit = f.st.calls.find((c) => c.url.endsWith("/submit"));
  expect(submit?.key).toMatch(/^[0-9a-f-]{36}$/);
  expect(submit?.tab).toBeTruthy();
  expect(f.st.answers.q1).toEqual({ option_ids: ["q1-o2"] }); // bản cuối đã lên máy chủ trước khi nộp
});

test("take: two tabs — tab thứ hai chỉ đọc, 'Làm tiếp ở đây' giành quyền ghi, tab đầu thành chỉ đọc", async ({ context, page }) => {
  const f = fakeTake();
  await takeSetup(page, f);
  await startExam(page);
  await page.getByLabel("Đáp án A của q1").check();
  await expect.poll(() => f.st.answers.q1).toEqual({ option_ids: ["q1-o1"] });
  const b = await context.newPage();
  await takeSetup(b, f);
  await b.goto(`/exams/${E1}/take?course=${C1}`);
  await expect(b.getByText("Bài đang mở ở nơi khác")).toBeVisible();
  await expect(b.getByRole("radio").first()).toBeDisabled();
  await b.getByRole("button", { name: "Làm tiếp ở đây" }).click();
  await expect(b.getByText("Bài đang mở ở nơi khác")).toBeHidden();
  await expect(page.getByText("Bài đang mở ở nơi khác")).toBeVisible(); // tab đầu nhận tin qua BroadcastChannel
  await expect(page.getByRole("radio").first()).toBeDisabled();
  await b.getByLabel("Đáp án C của q1").check();
  await expect.poll(() => f.st.answers.q1).toEqual({ option_ids: ["q1-o3"] });
});

test("take: duplicate tab — bản sao tab không bao giờ thành người ghi thứ hai", async ({ context, page }) => {
  const f = fakeTake();
  await takeSetup(page, f);
  await startExam(page);
  const writerTab = f.st.writer;
  const b = await context.newPage();
  await takeSetup(b, f);
  await b.goto(`/exams/${E1}/take?course=${C1}`);
  await expect(b.getByText("Bài đang mở ở nơi khác")).toBeVisible();
  await b.waitForTimeout(800);
  expect(f.st.calls.some((c) => c.url.endsWith("/takeover"))).toBe(false); // không tự takeover
  expect(f.st.writer).toBe(writerTab);
  await page.getByLabel("Đáp án A của q1").check();
  await expect.poll(() => f.st.answers.q1).toEqual({ option_ids: ["q1-o1"] }); // tab đầu vẫn ghi được
});

test("take: reload writer — tải lại trang giữ quyền ghi bằng takeover({reload:true}), không hiện chỉ-đọc", async ({ page }) => {
  const f = fakeTake();
  await takeSetup(page, f);
  await startExam(page);
  await page.getByLabel("Đáp án B của q1").check();
  await expect.poll(() => f.st.answers.q1).toEqual({ option_ids: ["q1-o2"] });
  const oldTab = f.st.writer;
  await page.reload();
  await expect(page.getByText("Câu 1/3")).toBeVisible();
  await expect.poll(() => f.st.calls.find((c) => c.url.endsWith("/takeover"))?.body).toEqual({ reload: true });
  expect(f.st.writer).not.toBe(oldTab);
  await expect(page.getByText("Bài đang mở ở nơi khác")).toBeHidden();
  await expect(page.getByLabel("Đáp án B của q1")).toBeChecked(); // đáp án đã lưu hiện lại
  await page.getByLabel("Đáp án C của q1").check();
  await expect.poll(() => f.st.answers.q1).toEqual({ option_ids: ["q1-o3"] });
});

test("take: offline mid exam — mất mạng vẫn chọn được; có mạng lại thì tự lưu", async ({ context, page }) => {
  const f = fakeTake();
  await takeSetup(page, f);
  await startExam(page);
  await context.setOffline(true);
  await page.getByLabel("Đáp án C của q1").check();
  await expect(page.getByText("Mất mạng — bài vẫn được giữ trên máy bạn").first()).toBeVisible();
  await expect(page.getByLabel("Đáp án C của q1")).toBeChecked();
  expect(f.st.answers.q1).toBeUndefined();
  await context.setOffline(false);
  await expect(page.locator("[data-part=save-status]")).toHaveText(/^Đã lưu lúc \d\d:\d\d:\d\d$/, { timeout: 20_000 });
  expect(f.st.answers.q1).toEqual({ option_ids: ["q1-o3"] });
});

test("take: mcq 375 — mỗi lần một câu, chọn bằng bàn phím, vùng chạm ≥ 44 px, không tràn ngang", async ({ page }) => {
  await page.setViewportSize({ width: 375, height: 760 });
  const f = fakeTake();
  await takeSetup(page, f);
  await startExam(page);
  await expect(page.getByText("Câu hỏi một")).toBeVisible();
  await expect(page.getByText("Câu hỏi hai")).toHaveCount(0); // một câu mỗi lần
  await page.keyboard.press("b");
  await expect(page.getByLabel("Đáp án B của q1")).toBeChecked();
  await page.keyboard.press("ArrowRight");
  await expect(page.getByText("Câu 2/3")).toBeVisible();
  await page.keyboard.press("1");
  await page.keyboard.press("3");
  await expect(page.getByLabel("Đáp án A của q2")).toBeChecked();
  await expect(page.getByLabel("Đáp án C của q2")).toBeChecked();
  await page.getByRole("button", { name: "Danh sách câu" }).click();
  await expect(page.getByRole("dialog").getByText("Đã làm 2/3 câu.")).toBeVisible();
  await page.getByRole("dialog").getByRole("button", { name: /Câu 3/ }).click();
  await expect(page.getByText("Câu 3/3")).toBeVisible();
  expect(await page.locator("#main").innerText()).not.toMatch(/judge|sandbox|verdict|RAG|PII/i);
  await expect.poll(() => page.evaluate(() => document.documentElement.scrollWidth - window.innerWidth), { timeout: 5000 }).toBeLessThanOrEqual(0);
  const { AUDIT_SRC, TOUCH_SRC } = await loadAudit();
  expect(await runAudit(page, AUDIT_SRC)).toEqual({ ox: 0, cut: [], ell: [] });
  expect(await page.evaluate(TOUCH_SRC)).toEqual([]);
});

// ───────── US-PE-06 — làm bài lập trình (gateway giả cùng trạng thái dùng chung) ─────────
const codeSetup = async (page: Page, over: Partial<TakeState> = {}) => {
  const f = fakeTake({ code: newCode(), ...over });
  await takeSetup(page, f);
  return f;
};
const startCode = async (page: Page, width = 1280) => {
  await page.setViewportSize({ width, height: 900 });
  await page.goto(`/exams/${E1}/take?course=${C1}`);
  await expect(page.getByText("Bài này có phần lập trình, cần màn hình ≥ 1024 px.")).toBeVisible();
  await page.locator("[data-part=integrity-notice]").scrollIntoViewIfNeeded();
  await page.getByRole("button", { name: "Bắt đầu làm bài" }).click();
  await expect(page.getByText("Câu 1/1")).toBeVisible();
};
const editor = (page: Page) => page.getByRole("textbox", { name: "Mã nguồn bài 1" });

test("take: code viewport gate — < 1024 px đọc được đề nhưng không có ô soạn mã; ≥ 1024 px có hai cột", async ({ page }) => {
  await codeSetup(page);
  await startCode(page, 1440);
  const strip = page.getByText("Bài lập trình cần màn hình rộng hơn (từ 1.024 px). Hãy mở bài thi này trên máy tính — bài của bạn vẫn là một lượt duy nhất và tự đồng bộ.");
  for (const w of [390, 1023, 1024, 1440]) {
    await page.setViewportSize({ width: w, height: 900 });
    await expect(page.getByText("Đọc hai số")).toBeVisible();
    await expect(page.getByLabel("Test mẫu")).toContainText("sample1");
    if (w < 1024) {
      await expect(strip).toBeVisible();
      await expect(page.locator("textarea")).toHaveCount(0);
    } else {
      await expect(strip).toBeHidden();
      await expect(editor(page)).toBeVisible();
      const { AUDIT_SRC } = await loadAudit();
      expect(await runAudit(page, AUDIT_SRC)).toEqual({ ox: 0, cut: [], ell: [] });
    }
  }
});

test("take: code editor — Tab / Shift+Tab / Esc rồi Tab, dán nhiều dòng, 64 KiB, vượt giới hạn, INP, axe", async ({ page }) => {
  await codeSetup(page);
  await startCode(page);
  const ed = editor(page);
  await expect(ed).toHaveAttribute("spellcheck", "false");
  await expect(ed).toHaveAttribute("autocapitalize", "off");
  // Tab thụt 4 dấu cách tại con trỏ
  await ed.fill("");
  await ed.press("Tab");
  await ed.pressSequentially("x");
  await expect(ed).toHaveValue("    x");
  // nhiều dòng: Tab thụt cả khối, Shift+Tab bỏ thụt
  await ed.fill("a\nb");
  await ed.press("ControlOrMeta+A");
  await ed.press("Tab");
  await expect(ed).toHaveValue("    a\n    b");
  await ed.press("Shift+Tab");
  await expect(ed).toHaveValue("a\nb");
  // Esc rồi Tab rời ô (không bẫy bàn phím) và không đổi chữ
  await ed.press("Escape");
  await ed.press("Tab");
  expect(await page.evaluate(() => document.activeElement?.tagName)).not.toBe("TEXTAREA");
  await expect(ed).toHaveValue("a\nb");
  // dán nhiều dòng giữ nguyên, tiếng Việt không vỡ
  const multi = "// chú thích tiếng Việt: Đặng Thị Ngọc\nint main(){\n\treturn 0;\n}";
  await ed.fill(multi);
  await expect(ed).toHaveValue(multi);
  // 64 KiB vừa đủ; gõ thêm bị chặn kèm lời
  await ed.fill("a".repeat(65536));
  await expect(page.getByText(/65\.536\/65\.536/)).toBeVisible();
  await ed.press("Control+End");
  await ed.pressSequentially("bcd");
  expect((await ed.inputValue()).length).toBe(65536);
  await expect(page.getByText(/Đã đủ 65\.536 byte/)).toBeVisible();
  // dán vượt giới hạn: bị cắt đúng 65.536 byte
  await ed.fill("é".repeat(40000)); // 2 byte mỗi ký tự
  expect(new TextEncoder().encode(await ed.inputValue()).length).toBeLessThanOrEqual(65536);
  // dòng rất dài cuộn ngang TRONG ô, không làm trang tràn ngang
  await ed.fill("x".repeat(5000));
  expect(await page.evaluate(() => document.documentElement.scrollWidth - window.innerWidth)).toBeLessThanOrEqual(0);
  // INP: gõ liên tục trên ô gần đầy
  // dán một lần ~56 KiB / 8.000 dòng (một sự kiện input, như người dùng dán thật; `fill` của công cụ chậm với nhiều dòng)
  const big = "int x;\n".repeat(8000);
  await ed.evaluate((el, v) => {
    Object.getOwnPropertyDescriptor(HTMLTextAreaElement.prototype, "value")!.set!.call(el, v);
    el.dispatchEvent(new Event("input", { bubbles: true }));
  }, big);
  await expect(ed).toHaveValue(big);
  await page.evaluate(() => {
    const w = window as unknown as { __inp: number[] };
    w.__inp = [];
    new PerformanceObserver((l) => l.getEntries().forEach((e) => w.__inp.push(e.duration))).observe({ type: "event", durationThreshold: 16, buffered: false } as PerformanceObserverInit);
  });
  await ed.press("Control+End");
  await ed.pressSequentially("int y = 1;", { delay: 10 });
  const worst = await page.evaluate(() => Math.max(0, ...(window as unknown as { __inp: number[] }).__inp));
  expect(worst).toBeLessThanOrEqual(200);
  await ed.fill("int main(){}");
  const a = await new AxeBuilder({ page }).withTags(["wcag2a", "wcag2aa", "wcag21aa", "wcag22aa"]).analyze();
  expect(a.violations.filter((v) => v.impact === "critical" || v.impact === "serious")).toEqual([]);
});

test("take: language switch keeps drafts — mỗi ngôn ngữ một bản nháp, đổi qua lại không mất chữ, tải lại nhớ ngôn ngữ gần nhất", async ({ page }) => {
  const f = await codeSetup(page);
  await startCode(page);
  const ed = editor(page);
  await page.getByLabel("Ngôn ngữ").selectOption("cpp17");
  await expect(ed).toHaveValue("// khởi đầu\n"); // lần đầu nạp mã khởi đầu của ngôn ngữ đó
  await ed.fill("// bản C++");
  await page.getByLabel("Ngôn ngữ").selectOption("c11");
  await expect(ed).toHaveValue("");
  await ed.fill("/* bản C */");
  await page.getByLabel("Ngôn ngữ").selectOption("cpp17");
  await expect(ed).toHaveValue("// bản C++");
  await page.getByLabel("Ngôn ngữ").selectOption("c11");
  await expect(ed).toHaveValue("/* bản C */");
  await expect.poll(() => [f.st.code!.drafts.cpp17?.source, f.st.code!.drafts.c11?.source], { timeout: 8000 }).toEqual(["// bản C++", "/* bản C */"]);
  await page.reload();
  await expect(page.getByText("Câu 1/1")).toBeVisible();
  await expect(page.getByLabel("Ngôn ngữ")).toHaveValue("c11"); // bản lưu gần nhất
  await expect(editor(page)).toHaveValue("/* bản C */");
  await page.getByLabel("Ngôn ngữ").selectOption("cpp17");
  await expect(editor(page)).toHaveValue("// bản C++");
});

test("take: draft conflict resolution — bản máy chủ mới hơn: hai nút, không mất chữ, không ghi đè lặng lẽ", async ({ page }) => {
  const f = await codeSetup(page);
  await startCode(page);
  const ed = editor(page);
  await page.getByLabel("Ngôn ngữ").selectOption("cpp17");
  await ed.fill("// của tôi 1");
  await expect.poll(() => f.st.code!.drafts.cpp17?.rev, { timeout: 8000 }).toBe(1);
  // tab khác đã lưu bản mới hơn
  f.st.code!.drafts.cpp17 = { source: "// bản của tab khác", rev: 2, at: new Date().toISOString() };
  await ed.fill("// của tôi 2");
  await expect(page.getByText(/^Bản trên máy chủ mới hơn \(lưu lúc \d\d:\d\d:\d\d\)\.$/)).toBeVisible({ timeout: 8000 });
  await expect(page.getByRole("button", { name: "Dùng bản trên máy này" })).toBeVisible();
  await expect(ed).toHaveValue("// của tôi 2"); // chữ đang gõ giữ nguyên
  expect(f.st.code!.drafts.cpp17.source).toBe("// bản của tab khác"); // máy chủ không bị ghi đè
  await page.getByRole("button", { name: "Dùng bản đã lưu" }).click();
  await expect(ed).toHaveValue("// bản của tab khác");
  await expect(page.getByRole("button", { name: "Dùng bản đã lưu" })).toBeHidden();
  // lần nữa: chọn bản trên máy
  f.st.code!.drafts.cpp17 = { source: "// tab khác lần ba", rev: 3, at: new Date().toISOString() };
  await ed.fill("// của tôi 3");
  await expect(page.getByRole("button", { name: "Dùng bản trên máy này" })).toBeVisible({ timeout: 8000 });
  await page.getByRole("button", { name: "Dùng bản trên máy này" }).click();
  await expect.poll(() => f.st.code!.drafts.cpp17.source, { timeout: 8000 }).toBe("// của tôi 3");
  expect(f.st.code!.drafts.cpp17.rev).toBe(4);
});

test("take: run result display — AC / WA / CE / TLE có nhãn tiếng Việt, WA hiện đầu vào / mong đợi / của bạn, không từ kỹ thuật", async ({ page }) => {
  const f = await codeSetup(page, {});
  await startCode(page);
  await editor(page).fill("int main(){}");
  const click = async () => page.getByRole("button", { name: "Chạy thử" }).click();
  const main = page.locator("#main");
  // AC
  await click();
  await expect(main.getByText("Đúng 1/1 test mẫu.")).toBeVisible();
  await expect(main.getByText("sample1")).toHaveCount(2); // đề (test mẫu) + kết quả
  // WA
  f.st.code!.runOut = { compile_ok: true, samples: [{ name: "sample1", verdict: "WA", time_ms: 4, memory_kb: 2048, input: "1 2\n", expected: "3\n", got: "4 " }] };
  await click();
  await expect(main.getByText("Sai kết quả")).toBeVisible();
  await expect(main.getByText("Đầu vào")).toBeVisible();
  await expect(main.getByText("Kết quả mong đợi")).toBeVisible();
  await expect(main.getByText("Kết quả của bạn")).toBeVisible();
  await expect(main.locator("pre", { hasText: "4·" })).toBeVisible(); // khoảng trắng hiện bằng ký hiệu
  // CE
  f.st.code!.runOut = { compile_ok: false, compile_log: "main.cpp:1:12: error: expected '}' at end of input", samples: [] };
  await click();
  await expect(main.getByText("Lỗi biên dịch")).toBeVisible();
  await expect(main.getByText("main.cpp:1:12: error")).toBeVisible();
  // TLE
  f.st.code!.runOut = { compile_ok: true, samples: [{ name: "sample1", verdict: "TLE", time_ms: 1000, memory_kb: 900 }] };
  await click();
  await expect(main.getByText("Quá thời gian")).toBeVisible();
  // lỗi hạn mức / máy chấm tắt có câu tiếng Việt
  f.st.code!.runStatus = 429;
  await click();
  await expect(page.getByText(/Bạn thao tác hơi nhanh/)).toBeVisible();
  f.st.code!.runStatus = 503;
  await click();
  await expect(page.getByText(/Hệ thống chấm bài chưa sẵn sàng/)).toBeVisible();
  expect(await main.innerText()).not.toMatch(/sandbox|judge|verdict|go-judge|\bTLE\b|\bWA\b|\bCE\b|stderr/i);
});

test("take: submission history — Đang chấm → kết quả, nhãn Lần nộp tính điểm, xem mã và Dùng lại mã này", async ({ page }) => {
  const f = await codeSetup(page, {});
  await startCode(page);
  await editor(page).fill("int main(){return 1;}");
  await page.getByRole("button", { name: "Nộp lời giải" }).click();
  const row = (n: number) => page.locator("[data-part=submission]").nth(n);
  await expect(row(0)).toContainText("Đang chấm");
  await expect(row(0)).toContainText("Biên dịch được · 1/1 test mẫu đúng", { timeout: 10_000 });
  await editor(page).fill("int main(){return 2;}");
  await page.getByRole("button", { name: "Nộp lời giải" }).click();
  await expect(page.locator("[data-part=submission]")).toHaveCount(2);
  await expect(page.getByText("Lần nộp tính điểm")).toHaveCount(1); // chỉ bản mới nhất
  await expect(row(0)).toContainText("Lần nộp tính điểm"); // mới nhất trước
  expect(f.st.code!.subs.map((x) => x.source)).toEqual(["int main(){return 1;}", "int main(){return 2;}"]);
  await row(1).getByRole("button", { name: "Xem mã" }).click();
  await expect(row(1).locator("pre").first()).toHaveText("int main(){return 1;}");
  await row(1).getByRole("button", { name: "Dùng lại mã này" }).click();
  await expect(editor(page)).toHaveValue("int main(){return 1;}");
  // giãn cách / giới hạn có lời
  f.st.code!.submitStatus = 429;
  await page.getByRole("button", { name: "Nộp lời giải" }).click();
  await expect(page.getByText(/Bạn thao tác hơi nhanh/)).toBeVisible();
  f.st.code!.submitStatus = 409;
  await page.getByRole("button", { name: "Nộp lời giải" }).click();
  await expect(page.getByText(/Bạn đã nộp đủ số lần cho bài này/)).toBeVisible();
});

test("take: timeout while typing — hết giờ khi đang gõ: gửi ngay bản nháp, ô chỉ đọc còn nguyên chữ", async ({ page }) => {
  const f = await codeSetup(page, { durationMs: 7000, graceMs: 10_000 }); // như máy chủ thật: nhận bản lưu tới hạn + 10 s
  await startCode(page);
  const ed = editor(page);
  await ed.click();
  await page.keyboard.type("x".repeat(60), { delay: 200 }); // gõ vượt qua giờ chót (≈ 7 s)
  await expect(page.getByText(/Hết giờ — phần bạn gõ sau giờ không được tính/).first()).toBeVisible();
  await expect(ed).toHaveAttribute("readonly", "");
  const text = await ed.inputValue();
  expect(text.length).toBeGreaterThan(10);
  expect(text.length).toBeLessThan(60); // ký tự gõ sau giờ bị bỏ
  await expect.poll(() => f.st.code!.drafts.c11?.source, { timeout: 8000 }).toBe(text); // bản cuối lên máy chủ NGAY khi hết giờ
  await expect(page.getByRole("button", { name: "Nộp lời giải" })).toBeDisabled();
});

test("take: sse drop during judge — không có kênh sự kiện vẫn nhận kết quả đúng một lần", async ({ page }) => {
  await page.route("**/api/v1/events**", (r) => r.abort());
  const f = await codeSetup(page, {});
  f.st.code!.pollsBeforeDone = 2;
  await startCode(page);
  await editor(page).fill("int main(){}");
  await page.getByRole("button", { name: "Chạy thử" }).click();
  await expect(page.getByText("Đang chạy thử…")).toBeVisible();
  await expect(page.getByText("Đúng 1/1 test mẫu.")).toBeVisible({ timeout: 12_000 });
  await expect(page.locator("[data-part=code-wide] li", { hasText: "Đúng" })).toHaveCount(1);
  const polls = f.st.code!.runs["run-1"].polls;
  expect(polls).toBeGreaterThanOrEqual(3);
  await page.waitForTimeout(2600);
  expect(f.st.code!.runs["run-1"].polls).toBe(polls); // xong thì thôi thăm dò
});

// ───────── US-PE-07 — liêm chính: câu minh bạch, ghi tín hiệu, so độ giống ─────────
const INTEGRITY = "Trong giờ làm bài, chat AI tạm khoá. Hệ thống ghi lại số lần bạn rời trang hoặc dán nội dung để giảng viên xem khi cần; đây không phải giám thị và không tự trừ điểm của bạn.";

test("take: integrity notice — đúng chữ ở màn bắt đầu và dải cố định trong giờ; Bắt đầu chỉ bấm được sau khi thấy câu này; không từ kỹ thuật", async ({ page }) => {
  const f = fakeTake();
  await takeSetup(page, f);
  await page.goto(`/exams/${E1}/take?course=${C1}`);
  const start = page.getByRole("button", { name: "Bắt đầu làm bài" });
  const notice = page.locator("[data-part=integrity-notice]");
  await expect(notice.getByText(INTEGRITY, { exact: true })).toBeVisible();
  await expect(start).toBeEnabled(); // câu đã nằm trong khung nhìn → đã "thấy"
  await notice.getByText("Tìm hiểu thêm").click();
  await expect(notice.getByText("Không ghi nội dung bạn dán, không dùng camera, không ghi màn hình.")).toBeVisible();
  expect(await page.locator("#main").innerText()).not.toMatch(/RAG|PII|provider|trace|gian lận|vi phạm/i);
  await start.click();
  await expect(page.getByText("Câu 1/3")).toBeVisible();
  await expect(page.locator("[data-part=integrity-notice]").getByText(INTEGRITY, { exact: true })).toBeVisible(); // dải cố định trong giờ
});

test("take: integrity notice gating — ở 375 px câu nằm dưới màn hình thì nút Bắt đầu bị khoá tới khi cuộn tới câu", async ({ page }) => {
  await page.setViewportSize({ width: 375, height: 420 });
  const f = fakeTake({ instructions: Array.from({ length: 40 }, (_, i) => `Dòng hướng dẫn số ${i + 1}.`).join("\n\n") }); // đẩy câu minh bạch xuống dưới màn hình
  await takeSetup(page, f);
  await page.goto(`/exams/${E1}/take?course=${C1}`);
  const start = page.getByRole("button", { name: "Bắt đầu làm bài" });
  await expect(start).toBeDisabled();
  await page.locator("[data-part=integrity-notice]").scrollIntoViewIfNeeded();
  await expect(start).toBeEnabled();
});

test("take: integrity events — rời tab / dán / mất mạng được ghi gộp; chỉ độ dài đoạn dán, không bao giờ nội dung", async ({ page }) => {
  const f = fakeTake();
  await takeSetup(page, f);
  await startExam(page);
  await page.evaluate(() => {
    const hide = (v: "hidden" | "visible") => {
      Object.defineProperty(document, "visibilityState", { value: v, configurable: true });
      document.dispatchEvent(new Event("visibilitychange"));
    };
    hide("hidden");
    return new Promise<void>((res) => setTimeout(() => { hide("visible"); res(); }, 300));
  });
  await page.evaluate(() => {
    const dt = new DataTransfer();
    dt.setData("text", "SECRET-PASTE-CONTENT");
    document.body.dispatchEvent(new ClipboardEvent("paste", { clipboardData: dt, bubbles: true }));
    window.dispatchEvent(new Event("offline"));
    window.dispatchEvent(new Event("online"));
    window.dispatchEvent(new Event("pagehide")); // gửi lô ngay thay vì chờ 15 s
  });
  await expect.poll(() => f.st.events.length, { timeout: 8000 }).toBeGreaterThanOrEqual(5);
  const types = f.st.events.map((e) => (e as { type: string }).type);
  expect(types).toEqual(expect.arrayContaining(["TAB_HIDDEN", "TAB_VISIBLE", "PASTE", "OFFLINE", "ONLINE"]));
  const hidden = f.st.events.find((e) => (e as { type: string }).type === "TAB_HIDDEN") as { meta: { duration_ms: number } };
  expect(hidden.meta.duration_ms).toBeGreaterThanOrEqual(250);
  const paste = f.st.events.find((e) => (e as { type: string }).type === "PASTE") as { meta: { chars: number } };
  expect(paste.meta.chars).toBe("SECRET-PASTE-CONTENT".length);
  const wire = f.st.calls.filter((c) => c.m === "EVENTS").map((c) => c.body).join("");
  expect(wire).not.toContain("SECRET");
});

const SIM_C = "e-sim";
function simPair(over: Json = {}) {
  return {
    id: "p-1", problem_id: "q-1", problem_title: "Tính tổng", run_id: "r-1", a: { attempt_id: "at-1", name: "Nguyễn Văn A" }, b: { attempt_id: "at-2", name: "Trần Thị B" },
    score: "0.873", shared_fingerprints: 40, flagged: true, review_state: "NEW", note: null, reviewed_at: null, created_at: "2026-12-01T05:00:00Z", ...over,
  };
}
async function simSetup(page: Page, role: "TEACHER" | "TA" = "TEACHER") {
  await setup(page, role);
  const st = { pair: simPair() as Json, reviews: [] as Json[], runs: 0, jobPolls: 0 };
  const j = (route: Route, status: number, body: unknown) => route.fulfill({ status, contentType: "application/json", headers: cors, body: JSON.stringify(body) });
  await page.route(new RegExp(`/api/v1/courses/${C1}/exams/${SIM_C}/similarity`), async (route) => {
    const req = route.request();
    const path = new URL(req.url()).pathname.split("/similarity")[1] || "";
    if (req.method() === "OPTIONS") return route.fulfill({ status: 204, headers: { ...cors, "Access-Control-Allow-Headers": "*", "Access-Control-Allow-Methods": "*" } });
    if (req.method() === "GET" && path === "") return j(route, 200, { items: [st.pair], next_cursor: null });
    if (req.method() === "POST" && path === "/run") { st.runs++; return j(route, 202, { job_id: "job-sim" }); }
    if (req.method() === "GET" && path === "/p-1") {
      return j(route, 200, {
        pair: st.pair,
        a: { language: "cpp17", source: "int main(){\n  int n;\n  scanf(\"%d\", &n);\n  return 0;\n}", match_lines: [2, 3] },
        b: { language: "cpp17", source: "int main(){\n  int count;\n  scanf(\"%d\", &count);\n  puts(\"x\");\n}", match_lines: [2, 3] },
      });
    }
    if (req.method() === "PUT" && path === "/p-1/review") {
      const b = req.postDataJSON() as { state: string; note: string | null };
      st.reviews.push(b);
      st.pair = { ...st.pair, review_state: b.state, note: b.note, reviewed_at: "2026-12-01T06:00:00Z" };
      return j(route, 200, st.pair);
    }
    return j(route, 404, { code: "NOT_FOUND", message: "x", trace_id: "t" });
  });
  await page.route("**/api/v1/jobs/job-sim", (route) => { st.jobPolls++; return j(route, 200, { id: "job-sim", kind: "exam.similarity", status: st.jobPolls > 1 ? "SUCCEEDED" : "RUNNING", progress: 50, result: null }); });
  return st;
}

test("similarity review — cặp hiện, hai mã cạnh nhau có tô phần khớp, đánh dấu Đã xem; không có hành động trừ điểm", async ({ page }) => {
  const st = await simSetup(page);
  await page.goto(`/exams/${SIM_C}/similarity?course=${C1}`);
  await expect(page.getByRole("heading", { name: "Nghi giống nhau", level: 1 })).toBeVisible();
  await expect(page.getByText("Độ giống chỉ là gợi ý — nhiều bài đúng cùng một cách làm tự nhiên giống nhau. Quyết định là của thầy/cô.")).toBeVisible();
  const pairRow = row(page, "Nguyễn Văn A · Trần Thị B");
  await expect(pairRow).toBeVisible();
  await expect(row(page, "87 %")).toBeVisible();
  await expect(row(page, "Nên xem")).toBeVisible();
  await pairRow.click();
  const pv = page.locator("[data-part=pair-view]");
  await expect(pv.getByText("Nguyễn Văn A")).toBeVisible();
  await expect(pv.getByText("Trần Thị B")).toBeVisible();
  await expect(pv.locator("[data-hit]")).toHaveCount(4); // 2 dòng khớp mỗi bên, tô sáng
  await pv.getByLabel("Ghi chú (tối đa 500 ký tự)").fill("Cùng cách đọc dữ liệu");
  await pv.getByRole("button", { name: "Đã xem — không có vấn đề" }).click();
  await expect.poll(() => st.reviews.length).toBe(1);
  expect(st.reviews[0]).toEqual({ state: "CLEARED", note: "Cùng cách đọc dữ liệu" });
  await expect(pv.getByText(/^Đã xem — không có vấn đề · /)).toBeVisible();
  await pv.getByRole("button", { name: "Cần trao đổi" }).click();
  await expect.poll(() => st.reviews.length).toBe(2);
  const text = await page.locator("#main").innerText();
  expect(text).not.toMatch(/trừ điểm|gian lận|vi phạm|nghi gian/i);
  await expect(page.getByRole("button", { name: /trừ điểm/i })).toHaveCount(0);
  // chạy lại: việc nền, tiến độ rồi danh sách làm mới
  await page.getByRole("button", { name: "Chạy lại so sánh" }).click();
  await expect.poll(() => st.runs).toBe(1);
});

test("similarity review — TA không mở được trang so độ giống", async ({ page }) => {
  await simSetup(page, "TA");
  await page.goto(`/exams/${SIM_C}/similarity?course=${C1}`);
  await expect(page.getByText(/Trang này dành cho/)).toBeVisible();
  await expect(page.getByText("Nguyễn Văn A")).toHaveCount(0);
});

// ---- US-PE-08: kết quả (Staff `/exams/[id]/results`, sinh viên `/exams/[id]/take` sau công bố) -----------------------------------------
const E_R = "00000000-0000-7000-8000-0000000e0801";
const A_1 = "00000000-0000-7000-8000-0000000a0801";
const ST = (n: number, name: string, code: string) => ({ id: `00000000-0000-7000-8000-00000000500${n}`, full_name: name, student_code: code });
function resultsRows(teacher: boolean): Json[] {
  const flags = (similarity: number, tab_hidden: number) => (teacher ? { flags: { similarity, tab_hidden, paste: 0 } } : {});
  return [
    { attempt_id: A_1, student: ST(1, "Nguyễn Văn A", "B20DC000001"), status: "GRADED", auto_score: "7.75", score: "7.75", adjusted: false, submitted_at: "2026-12-01T02:40:00Z", submit_reason: "MANUAL", ...flags(1, 4) },
    { attempt_id: "att-b", student: ST(2, "Trần Thị B", "B20DC000002"), status: "GRADED", auto_score: "9.00", score: "9.50", adjusted: true, submitted_at: "2026-12-01T02:59:59Z", submit_reason: "CLOSED", ...flags(0, 0) },
    { attempt_id: "att-c", student: ST(3, "Lê Văn C", "B20DC000003"), status: "GRADING", auto_score: null, score: null, adjusted: false, submitted_at: "2026-12-01T02:50:00Z", submit_reason: "TIMEOUT", ...flags(0, 0) },
    { attempt_id: null, student: ST(4, "Phạm Thị D", "B20DC000004"), status: "ABSENT", auto_score: null, score: null, adjusted: false, submitted_at: null, submit_reason: null, ...flags(0, 0) },
  ];
}
const detailItems = (): Json[] => [
  { item_id: "i-1", position: 1, type: "MCQ_SINGLE", stem: "2+2=?", options: [{ id: "o1", body: "3" }, { id: "o2", body: "4" }], earned: "1.00", max: "1.00", correct: true, mine: { option_ids: ["o2"] }, answer: { option_ids: ["o2"] }, explanation: "Phép cộng.", overridden: false, samples: [], hidden: null, final_submission: null, compile_log: null },
  { item_id: "i-2", position: 2, type: "CODE", stem: "Đọc a b, in a+b.", options: [], earned: "0.75", max: "1.00", correct: null, mine: null, answer: null, explanation: null, overridden: false, samples: [{ name: "sample1", verdict: "AC", time_ms: 3, memory_kb: 1200, input: "1 2\n", expected: "3\n" }], hidden: { passed: 3, total: 4 }, final_submission: { id: "s-1", language: "cpp17", source: "int main(){}", created_at: "2026-12-01T02:30:00Z", compile_ok: true }, compile_log: null },
];

async function resultsSetup(page: Page, role: "TEACHER" | "TA") {
  await setup(page, role);
  const teacher = role === "TEACHER";
  const st = { hold: true, version: 4, holds: [] as Json[], scores: [] as Json[], answers: [] as Json[], csv: 0, adjust: null as Json | null };
  const j = (route: Route, status: number, body: unknown) => route.fulfill({ status, contentType: "application/json", headers: cors, body: JSON.stringify(body) });
  const exam = () => ({ id: E_R, title: "Giữa kỳ", kind: "MIXED", status: "CLOSED", effective_status: "CLOSED", opens_at: "2026-12-01T01:00:00Z", closes_at: "2026-12-01T03:00:00Z", duration_minutes: 45, items_count: 2, attempts: { started: 3, graded: 2 }, published_at: null, version: st.version, created_at: "2026-11-20T00:00:00Z", instructions: null, shuffle_questions: true, shuffle_options: true, max_score: "10.00", rounding_step: "0.01", multi_scoring: "PARTIAL", reveal_answers: true, appeal_days: 7, publish_hold: st.hold, regrading: false, items: [], created_by: "u", updated_at: "2026-12-01T03:00:00Z" });
  await page.route(new RegExp(`/api/v1/courses/${C1}/exams/${E_R}`), async (route) => {
    const req = route.request();
    const path = new URL(req.url()).pathname.split(E_R)[1] || "";
    if (req.method() === "OPTIONS") return route.fulfill({ status: 204, headers: { ...cors, "Access-Control-Allow-Headers": "*", "Access-Control-Allow-Methods": "*" } });
    if (req.method() === "GET" && path === "") return j(route, 200, exam());
    if (req.method() === "GET" && path === "/results") return j(route, 200, { progress: { not_started: 0, in_progress: 0, grading: 1, graded: 2, absent: 1 }, items: resultsRows(teacher), next_cursor: null });
    if (req.method() === "GET" && path === `/results/${A_1}`) {
      return j(route, 200, { attempt_id: A_1, student: ST(1, "Nguyễn Văn A", "B20DC000001"), status: "GRADED", version: 3, auto_score: "7.75", score: st.adjust ? (st.adjust as { score: string }).score : "7.75", adjust: st.adjust, submitted_at: "2026-12-01T02:40:00Z", submit_reason: "MANUAL", items: detailItems(),
        submissions: [{ id: "s-1", item_id: "i-2", status: "DONE", verdict: "WA", language: "cpp17", source: "int main(){}", created_at: "2026-12-01T02:30:00Z", compile_ok: true, compile_log: null, tests: [{ position: 1, is_sample: true, verdict: "AC", time_ms: 3, memory_kb: 1200 }, { position: 2, is_sample: false, verdict: "WA", time_ms: 4, memory_kb: 1200 }] }],
        ...(teacher ? { integrity: { tab_hidden_count: 4, tab_hidden_ms: 61000, paste_count: 0, paste_chars: 0, offline_count: 0, takeover_count: 0, chat_blocked_count: 0 } } : {}), appeal: null });
    }
    if (req.method() === "PUT" && path === `/results/${A_1}/score`) { const b = req.postDataJSON() as { score: string | null }; st.scores.push(b); st.adjust = b.score ? { score: b.score, reason: "chấm tay", at: "2026-12-02T01:00:00Z" } : null; return j(route, 200, { attempt_id: A_1, auto_score: "7.75", score: b.score ?? "7.75", adjusted: b.score !== null, version: 4 }); }
    if (req.method() === "PUT" && path === "/publish-hold") { const b = req.postDataJSON() as { hold: boolean }; st.holds.push(b); st.hold = b.hold; st.version++; return j(route, 200, exam()); }
    if (req.method() === "GET" && path === "/events") return j(route, 200, { summary: {}, items: [{ id: "ev-1", type: "TAB_HIDDEN", occurred_at: "2026-12-01T02:10:00Z", client_at: null, meta: null }], next_cursor: null });
    if (req.method() === "GET" && path === "/stats") return j(route, 200, { distribution: Array.from({ length: 10 }, (_, i) => ({ from: `${i}.00`, to: `${i + 1}.00`, count: i === 7 ? 1 : i === 9 ? 1 : 0 })), mean: "8.38", median: "8.38", hardest: [{ item_id: "i-1", title: "2+2", correct_rate: "0.50" }], code: [{ item_id: "i-2", title: "a+b", mean_ratio: "0.75", ce_rate: "0.00" }] });
    if (req.method() === "GET" && path === "/results.csv") { st.csv++; return route.fulfill({ status: 200, contentType: "text/csv; charset=utf-8", headers: cors, body: "\ufeffmssv;ho_ten\nB20DC000001;Nguyễn Văn A\n" }); }
    if (req.method() === "GET" && path === "/appeals") return j(route, 200, { items: [{ id: "ap-1", attempt_id: A_1, status: "OPEN", reason: "Câu 2 chấm thiếu test", response: null, created_at: "2026-12-02T01:00:00Z", responded_at: null, score_before: null, score_after: null, version: 1, student: { full_name: "Nguyễn Văn A", student_code: "B20DC000001" } }], next_cursor: null });
    if (req.method() === "POST" && path === "/appeals/ap-1/answer") { st.answers.push(req.postDataJSON() as Json); return j(route, 200, { id: "ap-1", attempt_id: A_1, status: "UPHELD", reason: "x", response: "ok", created_at: "2026-12-02T01:00:00Z", responded_at: "2026-12-02T02:00:00Z", score_before: "7.75", score_after: null, version: 2 }); }
    return j(route, 404, { code: "NOT_FOUND", message: "x", trace_id: "t" });
  });
  return st;
}

test("results (Giảng viên): tiến độ, bảng điểm dấu phẩy, vắng, tín hiệu; Công bố có xác nhận nêu hậu quả; Drawer chi tiết + sửa điểm có lý do", async ({ page }) => {
  const st = await resultsSetup(page, "TEACHER");
  await page.goto(`/exams/${E_R}/results?course=${C1}`);
  await expect(page.getByRole("heading", { name: "Kết quả · Giữa kỳ", level: 1 })).toBeVisible();
  await expect(page.locator("[data-part=results-progress]")).toHaveText("Đang chấm 2/3 · Vắng 1");
  await expect(row(page, "Nguyễn Văn A")).toBeVisible();
  await expect(row(page, "7,75")).toBeVisible();
  await expect(row(page, "9,50 (đã sửa)")).toBeVisible();
  await expect(row(page, "Vắng").first()).toBeVisible();
  await expect(row(page, "Rời tab 4 · Giống nhau 1")).toBeVisible();
  await expect(page.getByRole("tab", { name: "Nghi giống nhau" })).toBeVisible();
  // Công bố: hộp xác nhận nêu hậu quả bằng số
  await page.getByRole("button", { name: "Công bố điểm" }).click();
  await expect(page.getByRole("dialog").getByText(/Công bố điểm cho 2 sinh viên\. Họ sẽ thấy điểm, đáp án và có thể gửi yêu cầu xem lại trong 7 ngày\./)).toBeVisible();
  await page.getByRole("dialog").getByRole("button", { name: "Công bố" }).click();
  await expect.poll(() => st.holds.length).toBe(1);
  expect(st.holds[0]).toEqual({ hold: false, version: 4 });
  // Drawer
  await row(page, "Nguyễn Văn A").click();
  const dr = page.locator("[data-part=result-drawer]");
  await expect(dr.getByText("Điểm chính thức")).toBeVisible();
  await expect(dr.getByText("Test ẩn: đạt 3 trên 4")).toBeVisible();
  await expect(dr.locator("[data-part=integrity]").getByRole("term").filter({ hasText: "Rời tab" })).toBeVisible();
  await expect(dr.getByText("Chỉ là tín hiệu để tham khảo, không phải kết luận.")).toBeVisible();
  const save = dr.getByRole("button", { name: "Lưu điểm" });
  await expect(save).toBeDisabled(); // cần lý do
  await dr.getByLabel("Điểm mới (đúng bước làm tròn)").fill("8,5");
  await dr.getByLabel("Lý do (bắt buộc, tối đa 500 ký tự)").first().fill("chấm tay câu 2");
  await save.click();
  await expect.poll(() => st.scores.length).toBe(1);
  expect(st.scores[0]).toEqual({ score: "8.5", reason: "chấm tay câu 2", version: 3 });
});

test("results (TA): đọc được bảng điểm, không có Tín hiệu / Công bố / Nghi giống nhau; Drawer không có tín hiệu liêm chính và không sửa điểm", async ({ page }) => {
  await resultsSetup(page, "TA");
  await page.goto(`/exams/${E_R}/results?course=${C1}`);
  await expect(row(page, "Nguyễn Văn A")).toBeVisible();
  await expect(page.getByRole("columnheader", { name: "Tín hiệu" })).toHaveCount(0);
  await expect(page.getByRole("button", { name: "Công bố điểm" })).toHaveCount(0);
  await expect(page.getByRole("tab", { name: "Nghi giống nhau" })).toHaveCount(0);
  await row(page, "Nguyễn Văn A").click();
  const dr = page.locator("[data-part=result-drawer]");
  await expect(dr.getByText("Điểm chính thức")).toBeVisible();
  await expect(dr.locator("[data-part=integrity]")).toHaveCount(0);
  await expect(dr.getByRole("button", { name: "Lưu điểm" })).toHaveCount(0);
  await expect(dr.locator("[data-part=override-panel]")).toHaveCount(0);
});

test("results: thống kê, xuất CSV, phúc khảo trả lời một lần", async ({ page }) => {
  const st = await resultsSetup(page, "TEACHER");
  await page.goto(`/exams/${E_R}/results?course=${C1}`);
  await page.getByRole("button", { name: "Xuất CSV" }).first().click();
  await expect.poll(() => st.csv).toBe(1);
  await page.getByRole("tab", { name: "Thống kê" }).click();
  await expect(page.locator("[data-part=stats]").getByText("8,38").first()).toBeVisible();
  await expect(page.locator("[data-part=stats]").getByText("2+2")).toBeVisible();
  await page.getByRole("tab", { name: "Xem lại điểm" }).click();
  const ap = page.locator("[data-part=appeal]");
  await expect(ap.getByText("Câu 2 chấm thiếu test")).toBeVisible();
  await ap.getByLabel("Phản hồi cho sinh viên (bắt buộc, tối đa 1.000 ký tự)").fill("Đã xem lại, giữ nguyên điểm.");
  await ap.getByRole("button", { name: "Gửi phản hồi" }).click();
  await expect.poll(() => st.answers.length).toBe(1);
  expect(st.answers[0]).toMatchObject({ decision: "UPHELD", response: "Đã xem lại, giữ nguyên điểm.", score: null, version: 1 });
});

function studentResultBody(appeal: Json) {
  return {
    exam: { id: E_R, title: "Giữa kỳ", max_score: "10.00", published_at: "2026-12-02T00:00:00Z", reveal_answers: true, appeal_days: 7, appeal_open_until: "2036-12-09T00:00:00Z" },
    score: "7.75", score_adjusted: true, appeal, items: detailItems(),
  };
}
async function studentResultSetup(page: Page, examStatus: "PUBLISHED" | "CLOSED") {
  await studentSetup(page, []);
  const st = { appeals: [] as Json[], appeal: { status: null, response: null } as Json };
  const j = (route: Route, status: number, body: unknown) => route.fulfill({ status, contentType: "application/json", headers: cors, body: JSON.stringify(body) });
  await page.route(new RegExp(`/api/v1/courses/${C1}/exams/${E_R}/`), async (route) => {
    const req = route.request();
    const path = new URL(req.url()).pathname.split(E_R)[1];
    if (req.method() === "OPTIONS") return route.fulfill({ status: 204, headers: { ...cors, "Access-Control-Allow-Headers": "*", "Access-Control-Allow-Methods": "*" } });
    if (path === "/attempts/mine") return j(route, 200, { attempt: { id: A_1, status: "GRADED", submitted_at: "2026-12-01T02:40:00Z", submit_reason: "MANUAL" }, exam: { id: E_R, title: "Giữa kỳ", closes_at: "2026-12-01T03:00:00Z", status: examStatus } });
    if (path === `/attempts/${A_1}/result`) return j(route, 200, studentResultBody(st.appeal));
    if (path === `/attempts/${A_1}/appeal` && req.method() === "POST") { st.appeals.push({ body: req.postDataJSON(), key: req.headers()["idempotency-key"] ?? null }); st.appeal = { status: "OPEN", response: null }; return j(route, 201, { id: "ap-1" }); }
    return j(route, 404, { code: "NOT_FOUND", message: "x", trace_id: "t" });
  });
  return st;
}

test("student result: điểm dấu phẩy, từng câu, test ẩn chỉ số đạt, đáp án + giải thích; xin xem lại tại chỗ (không hộp thoại), chỉ một lần", async ({ page }) => {
  const st = await studentResultSetup(page, "PUBLISHED");
  await page.goto(`/exams/${E_R}/take?course=${C1}`);
  const r = page.locator("[data-part=student-result]");
  await expect(r.locator("[data-part=final-score]")).toHaveText("7,75 / 10,00");
  await expect(r.getByText("Điểm đã được giảng viên điều chỉnh.")).toBeVisible();
  await expect(r.getByText("Bạn chọn").first()).toBeVisible();
  await expect(r.getByText("Đáp án đúng").first()).toBeVisible();
  await expect(r.locator("[data-part=explanation]").getByText("Phép cộng.")).toBeVisible();
  await expect(r.locator("[data-part=hidden-count]")).toHaveText("Test ẩn: đạt 3 trên 4");
  await expect(r.getByText("sample1")).toBeVisible();
  const text = await r.innerText();
  expect(text).not.toMatch(/trọng số|weight|test ẩn \d|input|expected/i);
  await r.getByRole("button", { name: "Gửi yêu cầu xem lại" }).click();
  await expect(page.getByRole("dialog")).toHaveCount(0); // mở dần tại chỗ
  await r.getByLabel("Bạn muốn giảng viên xem lại điều gì? (tối đa 1.000 ký tự)").fill("Câu 2 thiếu test");
  await r.getByRole("button", { name: "Gửi yêu cầu", exact: true }).click();
  await expect.poll(() => st.appeals.length).toBe(1);
  expect((st.appeals[0] as { body: Json; key: string | null }).body).toEqual({ reason: "Câu 2 thiếu test" });
  expect((st.appeals[0] as { key: string | null }).key).toBeTruthy();
  await expect(r.getByText("Yêu cầu xem lại của bạn đã gửi, giảng viên sẽ trả lời.")).toBeVisible();
  await expect(r.getByRole("button", { name: "Gửi yêu cầu xem lại" })).toHaveCount(0);
});

test("student result: 375 px — phần trắc nghiệm dùng được, mã code cuộn ngang trong khối mã, trang không tràn ngang", async ({ page }) => {
  await page.setViewportSize({ width: 375, height: 800 });
  await studentResultSetup(page, "PUBLISHED");
  await page.goto(`/exams/${E_R}/take?course=${C1}`);
  const r = page.locator("[data-part=student-result]");
  await expect(r.locator("[data-part=final-score]")).toBeVisible();
  await r.locator("details summary").first().click();
  await expect(r.locator("pre").first()).toBeVisible();
  const overflow = await page.evaluate(() => document.documentElement.scrollWidth - window.innerWidth);
  expect(overflow).toBeLessThanOrEqual(0);
});

test("student result: chưa công bố — 'Điểm đang được chấm.', không có điểm hay đáp án", async ({ page }) => {
  await studentResultSetup(page, "CLOSED");
  await page.goto(`/exams/${E_R}/take?course=${C1}`);
  await expect(page.getByText("Điểm đang được chấm.")).toBeVisible();
  await expect(page.locator("[data-part=student-result]")).toHaveCount(0);
});

test("results: 1.000 dòng được ảo hoá — DOM chỉ dựng dòng nhìn thấy, cuộn tới cuối vẫn thấy dòng cuối", async ({ page }) => {
  await resultsSetup(page, "TEACHER");
  const many = Array.from({ length: 1000 }, (_, i) => ({ attempt_id: `att-${i}`, student: { id: `00000000-0000-7000-8000-${String(i).padStart(12, "0")}`, full_name: `Sinh viên ${String(i).padStart(4, "0")}`, student_code: `B20DC${String(i).padStart(6, "0")}` }, status: "GRADED", auto_score: "5.00", score: "5.00", adjusted: false, submitted_at: "2026-12-01T02:40:00Z", submit_reason: "MANUAL", flags: { similarity: 0, tab_hidden: 0, paste: 0 } }));
  await page.route(new RegExp(`/api/v1/courses/${C1}/exams/${E_R}/results(\\?|$)`), (route) =>
    route.request().method() === "OPTIONS"
      ? route.fulfill({ status: 204, headers: { ...cors, "Access-Control-Allow-Headers": "*", "Access-Control-Allow-Methods": "*" } })
      : route.fulfill({ status: 200, contentType: "application/json", headers: cors, body: JSON.stringify({ progress: { not_started: 0, in_progress: 0, grading: 0, graded: 1000, absent: 0 }, items: many, next_cursor: null }) }),
  );
  await page.goto(`/exams/${E_R}/results?course=${C1}`);
  await expect(row(page, "Sinh viên 0000")).toBeVisible();
  const rendered = await page.locator("main tbody tr, main [role=row]").count();
  expect(rendered).toBeLessThan(120);
  await page.locator("main").getByRole("table").first().evaluate((t) => {
    for (let el: HTMLElement | null = t as HTMLElement; el; el = el.parentElement) {
      if (el.scrollHeight > el.clientHeight + 50 && getComputedStyle(el).overflowY !== "visible") { el.scrollTop = el.scrollHeight; break; }
    }
  });
  await expect(row(page, "Sinh viên 0999")).toBeVisible();
});
