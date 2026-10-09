import type { Page, Route } from "@playwright/test";
import { BASE_URL } from "./env";

/** Gateway giả đọc-chỉ cho các màn Staff có API thật (bài thi, câu hỏi, thành viên): chỉ phục vụ đo bố cục, không ghi gì. */
const cors = { "Access-Control-Allow-Origin": BASE_URL, "Access-Control-Allow-Credentials": "true", Vary: "Origin" };
const j = (route: Route, body: unknown, status = 200) => route.fulfill({ status, contentType: "application/json", headers: cors, body: JSON.stringify(body) });
const T = (d: number) => new Date(Date.UTC(2026, 11, d, 1, 0)).toISOString();

export const STAFF_EXAM = "e-open";
export const STAFF_DRAFT = "e-draft";
const row = (id: string, title: string, status: string, o: Record<string, unknown> = {}) => ({
  id, title, kind: "MCQ", status, effective_status: status, opens_at: T(1), closes_at: T(2), duration_minutes: 45, items_count: 3, attempts: { started: 12, graded: 9 },
  published_at: null, version: 1, created_at: "2026-10-01T00:00:00Z", ...o,
});
const EXAMS = [
  row(STAFF_EXAM, "Giữa kỳ", "OPEN"),
  row("e-soon", "Tuần 10", "SCHEDULED", { opens_at: T(8), closes_at: T(9) }),
  row("e-closed", "Tuần 7", "CLOSED", { opens_at: T(-7), closes_at: T(-6) }),
  row(STAFF_DRAFT, "Tuần 11 (nháp)", "DRAFT", { opens_at: null, closes_at: null, duration_minutes: null, items_count: 2, attempts: { started: 0, graded: 0 } }),
];
const detail = (e: ReturnType<typeof row>) => ({
  ...e, instructions: "Đọc kỹ đề.", shuffle_questions: true, shuffle_options: true, max_score: "10.00", rounding_step: "0.01", multi_scoring: "PARTIAL", reveal_answers: true, appeal_days: 7,
  publish_hold: false, regrading: false, created_by: "u", updated_at: "2026-10-01T00:00:00Z",
  items: [
    { id: "i1", question_id: "q1", position: 1, points: "5.00", type: "MCQ_SINGLE", title: "AES là thuật toán gì?", topic: "Mật mã", difficulty: "EASY", review_status: "APPROVED", archived: false },
    { id: "i2", question_id: "q2", position: 2, points: "5.00", type: "TRUE_FALSE", title: "RSA dựa trên phân tích nguyên tố", topic: "Mật mã", difficulty: "MEDIUM", review_status: "APPROVED", archived: false },
  ],
});
const QUESTIONS = ["AES là thuật toán gì?", "RSA dựa trên bài toán nào?", "Hàm băm có tính chất nào?"].map((title, i) => ({
  id: `q${i + 1}`, type: "MCQ_SINGLE", title, topic: i === 2 ? "Hàm băm" : "Mật mã", difficulty: "EASY", review_status: i === 1 ? "PENDING" : "APPROVED", origin: "MANUAL", used_in_exams: i, version: 1, archived_at: null, updated_at: "2026-10-01T00:00:00Z",
}));
const RESULTS = Array.from({ length: 6 }, (_, i) => ({
  attempt_id: `a${i}`, student: { id: `s${i}`, full_name: `Sinh viên ${i + 1}`, student_code: `B20DC00${i + 1}` }, status: i === 5 ? "NOT_STARTED" : "GRADED", auto_score: "7.50", score: i === 5 ? null : "7.50", adjusted: false,
  submitted_at: T(1), submit_reason: "MANUAL", flags: { similarity: i === 0 ? 2 : 0, tab_hidden: 0, paste: 0 },
}));
const PAIR = {
  id: "p1", problem_id: "pr1", problem_title: "Tính tổng", run_id: "r1", a: { attempt_id: "a0", name: "Sinh viên 1" }, b: { attempt_id: "a1", name: "Sinh viên 2" }, score: "0.91", shared_fingerprints: 14,
  flagged: true, review_state: "NEW", note: null, reviewed_at: null, created_at: "2026-12-02T00:00:00Z",
};
const MEMBERS = [
  { user_id: "m1", full_name: "Nguyễn Văn A", email: "a@x.edu.vn", student_code: "B20DC001", role_in_course: "STUDENT", status: "ACTIVE", joined_via: "ROSTER", warning: null, status_changed_at: "2026-10-01T00:00:00Z" },
  { user_id: "m2", full_name: "Trần Thị B", email: "b@x.edu.vn", student_code: "B20DC002", role_in_course: "STUDENT", status: "ACTIVE", joined_via: "CODE", warning: null, status_changed_at: "2026-10-02T00:00:00Z" },
];

export async function mockStaffApi(page: Page) {
  await page.route(/\/api\/v1\/courses\/[^/]+\/(exams|questions|members|join-code|share-sources)/, async (route) => {
    const req = route.request();
    if (req.method() === "OPTIONS") return route.fulfill({ status: 204, headers: { ...cors, "Access-Control-Allow-Headers": "*", "Access-Control-Allow-Methods": "*" } });
    const u = new URL(req.url());
    const p = u.pathname.replace(/^.*\/courses\/[^/]+/, "");
    const list = (items: unknown[]) => j(route, { items, next_cursor: null });
    if (p === "/join-code") return j(route, { join_code: "AN7K2MQ", join_url: "https://edupilot.example/join/AN7K2MQ", enabled: true, expires_at: null, require_approval: true, allowed_email_domain: null, capacity: 120, active_students: 30, pending: 2, version: 1 });
    if (p.startsWith("/share-sources")) return list([]);
    if (p.startsWith("/members")) return j(route, { items: u.searchParams.get("status") === "PENDING" ? [] : MEMBERS, next_cursor: null, counts: { active: MEMBERS.length, pending: 0 } });
    if (p.startsWith("/questions")) return list(QUESTIONS);
    if (p === "/exams") return list(EXAMS);
    const m = /^\/exams\/([^/]+)(\/.*)?$/.exec(p);
    if (!m) return j(route, { code: "NOT_FOUND", message: "x", trace_id: "t" }, 404);
    const e = EXAMS.find((x) => x.id === m[1]);
    const sub = m[2] ?? "";
    if (sub === "" && e) return j(route, detail(e));
    if (sub === "/preview" && e) {
      return j(route, { preview: true, exam: { id: e.id, title: e.title, instructions: "Đọc kỹ đề.", kind: "MCQ", opens_at: e.opens_at, closes_at: e.closes_at, duration_minutes: 45, max_score: "10.00", status: e.status, my_attempt: null, my_score: null },
        items: [{ item_id: "i1", position: 1, type: "MCQ_SINGLE", points: "5.00", stem: "AES là thuật toán gì?", options: [{ id: "o1", body: "Mã khối" }, { id: "o2", body: "Mã dòng" }], code: null }] });
    }
    if (sub === "/results") return j(route, { progress: { not_started: 1, in_progress: 0, grading: 0, graded: 5, absent: 0 }, items: RESULTS, next_cursor: null });
    if (sub === "/similarity") return list([PAIR]);
    if (sub === "/similarity/p1") return j(route, { pair: PAIR, a: { language: "cpp17", source: "int main() {\n  return 0;\n}\n", match_lines: [1, 2] }, b: { language: "cpp17", source: "int main() {\n  return 0;\n}\n", match_lines: [1, 2] } });
    if (sub === "/appeals" || sub.startsWith("/stats")) return list([]);
    return j(route, { code: "NOT_FOUND", message: "x", trace_id: "t" }, 404);
  });
}

const MODEL = { id: "m1", model: "gpt-4o-mini", kind: "chat", dims: null, price_in: "0.15", price_out: "0.60", enabled: true };
const PROVIDER = { id: "pv1", type: "openai", name: "OpenAI", base_url: null, enabled: true, has_key: true, key_status: "ok", rpm_limit: 600, tpm_limit: null, last_test: { ok: true, at: "2026-10-01T00:00:00Z", error_kind: null }, circuit: "closed", models: [MODEL], version: 1 };
const CHAIN = [{ model_id: "m1", provider_id: "pv1", provider_name: "OpenAI", model: "gpt-4o-mini" }];

/** `/admin/llm/*` cho `/settings/llm` (chỉ đọc). */
export async function mockLlmApi(page: Page) {
  await page.route(/\/api\/v1\/admin\/llm\//, async (route) => {
    const req = route.request();
    if (req.method() === "OPTIONS") return route.fulfill({ status: 204, headers: { ...cors, "Access-Control-Allow-Headers": "*", "Access-Control-Allow-Methods": "*" } });
    const p = new URL(req.url()).pathname;
    if (p.endsWith("/providers")) return j(route, { items: [PROVIDER], env_fallback: { active: false, providers: [] } });
    if (p.endsWith("/routes")) return j(route, { items: [{ task: "chat", lane: "INTERACTIVE", chain: CHAIN, params: {}, version: 1 }], embedding: null });
    if (p.endsWith("/budget")) return j(route, { scope: "global", daily_limit: "10.00", monthly_limit: "200.00", spent_today: "1.20", spent_month: "40.00", pct_today: 12, pct_month: 20, state: "ok", version: 1 });
    if (p.endsWith("/usage")) return j(route, { from: "2026-10-01T00:00:00Z", to: "2026-10-30T00:00:00Z", group: "task", items: [] });
    return j(route, { code: "NOT_FOUND", message: "x", trace_id: "t" }, 404);
  });
}

const USERS = [
  { id: "u1", email: "ha.lt@edupilot.test", full_name: "Lê Thu Hà", role: "TEACHER", status: "ACTIVE", last_login_at: "2026-10-09T02:00:00Z", version: 1 },
  { id: "u2", email: "bao.pq@edupilot.test", full_name: "Phạm Quốc Bảo", role: "TA", status: "INVITED", last_login_at: null, version: 1 },
  { id: "u3", email: "kh@edupilot.test", full_name: "Người bị khoá", role: "STUDENT", status: "DISABLED", last_login_at: "2026-09-01T02:00:00Z", version: 2 },
];
const COURSES = [
  { id: "c1", class_code: "761987", subject_code: "INT1006", name: "An ninh mạng", semester: "2026-2027-HK1", status: "ACTIVE", teacher: { id: "u1", full_name: "Lê Thu Hà" }, assistants_count: 1, students_active: 30, students_pending: 2, capacity: 40, version: 1 },
  { id: "c2", class_code: "761988", subject_code: "INT1007", name: "Mật mã học", semester: "2026-2027-HK1", status: "ARCHIVED", teacher: null, assistants_count: 0, students_active: 0, students_pending: 0, capacity: null, version: 1 },
];

/** `/admin/users` và `/admin/courses` (chỉ đọc). */
export async function mockAdminApi(page: Page) {
  await page.route(/\/api\/v1\/admin\/(users|courses)/, async (route) => {
    const req = route.request();
    if (req.method() === "OPTIONS") return route.fulfill({ status: 204, headers: { ...cors, "Access-Control-Allow-Headers": "*", "Access-Control-Allow-Methods": "*" } });
    const p = new URL(req.url()).pathname;
    return j(route, { items: p.endsWith("/users") ? USERS : COURSES, next_cursor: null });
  });
}
