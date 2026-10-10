import type { Page, Route } from "@playwright/test";
import { BASE_URL } from "./env";
import { COURSE } from "./chat-fixtures";

// Threads THẬT (US-P3-06) với gateway giả bằng page.route, theo HỢP ĐỒNG THẬT (backend-go/api/openapi.yaml `/courses/{id}/threads*`).
export { COURSE };
export const TID = "00000000-0000-7000-8000-0000000000a1";
export const PID = "00000000-0000-7000-8000-0000000000b1";
const CORS = { "Access-Control-Allow-Origin": BASE_URL, "Access-Control-Allow-Credentials": "true", Vary: "Origin", "Access-Control-Allow-Headers": "authorization,content-type,x-request-id,idempotency-key,accept", "Access-Control-Allow-Methods": "GET,POST,PUT,DELETE,OPTIONS" };
export const json = (body: unknown, status = 200) => ({ status, contentType: "application/json", headers: CORS, body: JSON.stringify(body) });
export const err = (status: number, code: string, details?: unknown) => json({ code, message: "m", trace_id: "b".repeat(32), details }, status);

export const row = (over: Record<string, unknown> = {}) => ({
  id: TID, title: "Điều 5 quy chế", preview: "Quy định thi lại ở Điều 5 là gì?", tags: ["quy-che"], week_no: 3, author: { full_name: "Ngô Ngọc Cẩm", role: "STUDENT", is_me: false },
  answer_state: "PENDING", reply_count: 1, last_activity_at: "2026-10-11T02:00:00Z", created_at: "2026-10-11T01:00:00Z", ...over,
});
export const aiPost = (over: Record<string, unknown> = {}) => ({
  id: PID, kind: "AI", author: null, body: "Theo quy chế [1], sinh viên được thi lại một lần.", verification_state: "PENDING", version: 1, created_at: "2026-10-11T01:05:00Z",
  citations: [{ n: 1, document_id: "00000000-0000-7000-8000-0000000000f1", title: "Quy chế học vụ", page_no: 4, snippet: "Sinh viên được thi lại một lần." }], ...over,
});
export const view = (posts: unknown[] = [aiPost()], over: Record<string, unknown> = {}) => ({
  thread: { id: TID, title: "Điều 5 quy chế", body: "Quy định thi lại ở Điều 5 là gì?", tags: ["quy-che"], week_no: 3, author: { full_name: "Ngô Ngọc Cẩm", role: "STUDENT", is_me: false }, reply_count: 1,
    created_at: "2026-10-11T01:00:00Z", last_activity_at: "2026-10-11T02:00:00Z", similar_of: null, ...over }, posts, next_cursor: null,
});
export const precheckOk = { allowed: true, reasons: [], redacted_text: "", redacted_title: "", personal_question: false };

type Handler = (route: Route) => Promise<void> | void;
type Fulfill = Parameters<Route["fulfill"]>[0];
export async function threadApi(page: Page, h: Record<string, Handler | Fulfill> = {}) {
  const calls: Array<{ method: string; path: string; body: string; headers: Record<string, string> }> = [];
  const handle = async (route: Route) => {
    const req = route.request();
    if (req.method() === "OPTIONS") return route.fulfill({ status: 204, headers: CORS });
    const path = new URL(req.url()).pathname.replace("/api/v1", "");
    calls.push({ method: req.method(), path, body: req.postData() ?? "", headers: req.headers() });
    const step = h[`${req.method()} ${path}`];
    if (typeof step === "function") return step(route);
    if (step) return route.fulfill(step);
    return route.fulfill(err(404, "NOT_FOUND"));
  };
  await page.route(`**/api/v1/courses/${COURSE}/threads**`, handle);
  await page.route(`**/api/v1/courses/${COURSE}/posts/**`, handle);
  await page.route("**/api/v1/chat/sessions/from-draft", handle);
  return calls;
}
