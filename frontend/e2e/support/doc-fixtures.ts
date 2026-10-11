import type { Page, Route } from "@playwright/test";
import { BASE_URL } from "./env";
import { COURSE } from "./chat-fixtures";

// Tài liệu / thư viện THẬT (US-P8-02) với gateway giả bằng page.route theo HỢP ĐỒNG THẬT (backend-go/api/openapi.yaml).
export { COURSE };
export const DID = "00000000-0000-7000-8000-0000000000d1";
const CORS = { "Access-Control-Allow-Origin": BASE_URL, "Access-Control-Allow-Credentials": "true", Vary: "Origin", "Access-Control-Allow-Headers": "authorization,content-type,x-request-id,idempotency-key,accept,if-none-match", "Access-Control-Allow-Methods": "GET,POST,PUT,PATCH,DELETE,OPTIONS", "Access-Control-Expose-Headers": "ETag" };
export const json = (body: unknown, status = 200, headers: Record<string, string> = {}) => ({ status, contentType: "application/json", headers: { ...CORS, ...headers }, body: JSON.stringify(body) });
export const err = (status: number, code: string) => json({ code, message: "m", trace_id: "b".repeat(32) }, status);
export const noContent = { status: 204, headers: CORS, body: "" };

export const doc = (over: Record<string, unknown> = {}) => ({
  id: DID, course_id: COURSE, title: "Quy chế học vụ", type: "LECTURE", filename: "quy-che.pdf", mime_type: "application/pdf", size_bytes: 1000, status: "READY", error: null, page_count: 3,
  visible_to_students: true, use_for_rag: true, category: null, week_no: 3, version: 1, created_at: "2026-10-10T01:00:00Z", updated_at: "2026-10-10T02:00:00Z", ...over,
});
export const stats = (over: Record<string, unknown> = {}) => ({ total: 1, by_type: { LECTURE: 1 }, by_status: { READY: 1 }, chunks: 3, embedded_chunks: 3, pages: 3, bytes: 1000, has_course_policy: true, last_upload_at: "2026-10-10T01:00:00Z", ...over });
export const libItem = (over: Record<string, unknown> = {}) => ({ id: DID, title: "Quy chế học vụ", type: "LECTURE", file_kind: "PDF", category: null, week_no: 3, updated_at: "2026-10-10T02:00:00Z", snippet: "", can_ask_ai: true, ...over });
export const libDetail = (over: Record<string, unknown> = {}) => ({ ...libItem(), size_bytes: 1000, page_count: 3, preview_url: "http://localhost:9/preview.pdf", ...over });
export const chunk = (n: number, over: Record<string, unknown> = {}) => ({ id: `00000000-0000-7000-8000-00000000c0${n}0`.slice(0, 36), ord: n, page_no: n + 1, heading: null, text: `Nội dung đoạn ${n + 1}`, audience: "ALL", ...over });

type Handler = (route: Route) => Promise<void> | void;
type Fulfill = Parameters<Route["fulfill"]>[0];
export async function docApi(page: Page, h: Record<string, Handler | Fulfill> = {}) {
  const calls: Array<{ method: string; path: string; body: string; headers: Record<string, string> }> = [];
  const handle = async (route: Route) => {
    const req = route.request();
    if (req.method() === "OPTIONS") return route.fulfill({ status: 204, headers: CORS });
    const u = new URL(req.url());
    const path = u.pathname.replace("/api/v1", "");
    calls.push({ method: req.method(), path: path + u.search, body: req.postData() ?? "", headers: req.headers() });
    const step = h[`${req.method()} ${path}`];
    if (typeof step === "function") return step(route);
    if (step) return route.fulfill(step);
    return route.fulfill(err(404, "NOT_FOUND"));
  };
  for (const g of [`**/api/v1/courses/${COURSE}/documents**`, `**/api/v1/courses/${COURSE}/library**`, `**/api/v1/courses/${COURSE}/uploads/**`, "**/api/v1/chat/sessions", "**/api/v1/jobs/**"]) await page.route(g, handle);
  return calls;
}
