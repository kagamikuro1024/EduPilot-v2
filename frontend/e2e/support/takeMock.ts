import type { Page, Route } from "@playwright/test";
import { BASE_URL } from "./env";
import { asJwt } from "./session";

export const TAKE_COURSE = "00000000-0000-7000-8000-00000000c001";
export const TAKE_EXAM = "e-take";
export type TakeKind = "intro" | "running" | "submitted" | "published";
const cors = { "Access-Control-Allow-Origin": BASE_URL, "Access-Control-Allow-Credentials": "true", Vary: "Origin" };
const j = (route: Route, status: number, body: unknown) => route.fulfill({ status, contentType: "application/json", headers: cors, body: JSON.stringify(body) });
const optsOf = (q: string) => [1, 2, 3].map((i) => ({ id: `${q}-o${i}`, body: `Đáp án ${"ABC"[i - 1]} của ${q}` }));
const examHead = { id: TAKE_EXAM, title: "Tuần 9", instructions: null, kind: "MCQ", duration_minutes: 45, closes_at: "2036-12-01T03:00:00Z", opens_at: "2026-12-01T01:00:00Z" };

/** Gateway giả tối thiểu cho `/exams/[id]/take` (đọc): một trong bốn trạng thái; chỉ phục vụ đo bố cục, không ghi gì. */
export async function mockTake(page: Page, kind: TakeKind, opts: { now?: number; expIn?: number } = {}) {
  await asJwt(page, "STUDENT", { expIn: opts.expIn });
  const course = { id: TAKE_COURSE, class_code: "761987", subject_code: "INT1006", name: "An ninh mạng", semester: "2026-2027-HK1", status: "ACTIVE" };
  await page.route("**/api/v1/me/courses**", (r) => j(r, 200, { items: [{ course, role_in_course: "STUDENT", enrollment_status: "ACTIVE" }], next_cursor: null }));
  await page.route("**/api/v1/me/today**", (r) => j(r, 200, { no_course: false, email_verified: true, recommended: null, timeline: [], continue: [] }));
  await page.route("**/api/v1/notifications**", (r) => j(r, 200, { items: [], next_cursor: null, unread_count: 0 }));
  await page.route(/\/api\/v1\/courses\/[^/]+\/exams/, async (route) => {
    const req = route.request();
    if (req.method() === "OPTIONS") return route.fulfill({ status: 204, headers: { ...cors, "Access-Control-Allow-Headers": "*", "Access-Control-Allow-Methods": "*" } });
    const path = new URL(req.url()).pathname.split(`/exams/${TAKE_EXAM}`)[1] ?? "";
    const runAttempt = () => { const now = opts.now ?? Date.now(); const dl = now + 40 * 60_000; return { id: "a-take", exam_id: TAKE_EXAM, status: "IN_PROGRESS", started_at: new Date(dl - 45 * 60_000).toISOString(), deadline_at: new Date(dl).toISOString(), server_time: new Date(now).toISOString(), writer: { is_you: true } }; };
    if (req.method() === "POST" && path.endsWith("/takeover")) return j(route, 200, runAttempt());
    if (req.method() === "POST" && path.endsWith("/events")) return route.fulfill({ status: 204, headers: cors });
    if (req.method() === "PUT" && path.endsWith("/answers")) return j(route, 200, { saved_at: new Date(opts.now ?? Date.now()).toISOString(), server_time: new Date(opts.now ?? Date.now()).toISOString(), deadline_at: runAttempt().deadline_at });
    if (path.endsWith("/attempts/mine")) {
      if (kind === "intro") return j(route, 200, { attempt: null, exam: { ...examHead, max_score: "10.00", status: "OPEN", my_attempt: null, my_score: null } });
      if (kind === "running") {
        const now = opts.now ?? Date.now();
        const dl = now + 40 * 60_000;
        const items = [1, 2, 3].map((n) => ({ item_id: `q${n}`, position: n, type: "MCQ_SINGLE", points: "4.00", stem: `Câu hỏi số ${n}: chọn đáp án đúng nhất`, options: optsOf(`q${n}`), code: null, answer: null }));
        return j(route, 200, { attempt: { id: "a-take", exam_id: TAKE_EXAM, status: "IN_PROGRESS", started_at: new Date(dl - 45 * 60_000).toISOString(), deadline_at: new Date(dl).toISOString(), server_time: new Date(now).toISOString(), writer: { is_you: true } }, exam: { ...examHead, multi_scoring: "PARTIAL" }, items });
      }
      return j(route, 200, { attempt: { id: "a-take", status: "GRADED", submitted_at: "2026-12-01T02:40:00Z", submit_reason: "MANUAL" }, exam: { ...examHead, status: kind === "published" ? "PUBLISHED" : "OPEN" } });
    }
    if (path.endsWith("/result")) {
      const items = [
        { item_id: "i-1", position: 1, type: "MCQ_SINGLE", stem: "2+2=?", options: [{ id: "o1", body: "3" }, { id: "o2", body: "4" }], earned: "1.00", max: "1.00", correct: true, mine: { option_ids: ["o2"] }, answer: { option_ids: ["o2"] }, explanation: "Phép cộng.", overridden: false, samples: [], hidden: null, final_submission: null, compile_log: null },
      ];
      return j(route, 200, { exam: { id: TAKE_EXAM, title: "Tuần 9", max_score: "10.00", published_at: "2026-12-02T00:00:00Z", reveal_answers: true, appeal_days: 7, appeal_open_until: "2036-12-09T00:00:00Z" }, score: "7.75", score_adjusted: false, appeal: { status: null, response: null }, items });
    }
    return j(route, 404, { code: "NOT_FOUND", message: "x", trace_id: "t" });
  });
}
