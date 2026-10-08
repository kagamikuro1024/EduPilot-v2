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
