import type { Page, Route } from "@playwright/test";
import { BASE_URL } from "./env";
import { COURSE } from "./chat-fixtures";

// Lịch THẬT (US-P8-03) với gateway giả bằng page.route theo HỢP ĐỒNG THẬT (backend-go/api/openapi.yaml `/courses/{id}/calendar*`, `/me/calendar/ics-token`).
export { COURSE };
const CORS = { "Access-Control-Allow-Origin": BASE_URL, "Access-Control-Allow-Credentials": "true", Vary: "Origin", "Access-Control-Allow-Headers": "authorization,content-type,x-request-id,idempotency-key,accept,if-none-match", "Access-Control-Allow-Methods": "GET,POST,PUT,DELETE,OPTIONS", "Access-Control-Expose-Headers": "ETag" };
export const json = (body: unknown, status = 200) => ({ status, contentType: "application/json", headers: CORS, body: JSON.stringify(body) });
export const err = (status: number, code: string) => json({ code, message: "m", trace_id: "b".repeat(32) }, status);
export const noContent = { status: 204, headers: CORS, body: "" };

const HOUR = 3_600_000;
/** "Bây giờ" cố định của spec lịch (Thứ Tư 07/10/2026 10:00 giờ Việt Nam, trước mọi ngày chạy thật để JWT giả (exp = giờ thật + 15 phút) không hết hạn): tuần Thứ Hai–Chủ Nhật không phụ thuộc ngày chạy. Spec đặt `page.clock.setFixedTime(NOW)`. */
export const NOW = Date.parse("2026-10-07T03:00:00Z");
/** Giờ tương đối so với NOW. */
export const at = (hours: number) => new Date(NOW + hours * HOUR).toISOString();

export const item = (over: Record<string, unknown> = {}) => ({
  id: "calendar_event:00000000-0000-7000-8000-0000000000a1", source: "calendar_event", type: "OTHER", title: "Nộp báo cáo", starts_at: at(30), ends_at: null, location: "P.301",
  href: null, editable: false, personal_state: null, status: null, ...over,
});
export const exam = (over: Record<string, unknown> = {}) => item({ id: "weekly_exam:00000000-0000-7000-8000-0000000000b1", source: "weekly_exam", type: "EXAM", title: "Thi giữa kỳ", starts_at: at(5), ends_at: at(6), location: null, href: "/exams/00000000-0000-7000-8000-0000000000b1/take", personal_state: "NOT_STARTED", status: "SCHEDULED", ...over });
export const session = (over: Record<string, unknown> = {}) => item({ id: "class_session:00000000-0000-7000-8000-0000000000c1", source: "class_session", type: "CLASS_SESSION", title: "Buổi 3 · Mật mã", starts_at: at(80), ends_at: at(81.5), location: "P.402", ...over });
export const page = (items: unknown[]) => json({ items, next_cursor: null });

type Handler = (route: Route) => Promise<void> | void;
type Fulfill = Parameters<Route["fulfill"]>[0];
export async function calApi(p: Page, h: Record<string, Handler | Fulfill> = {}) {
  const calls: Array<{ method: string; path: string; body: string }> = [];
  const handle = async (route: Route) => {
    const req = route.request();
    if (req.method() === "OPTIONS") return route.fulfill({ status: 204, headers: CORS });
    const u = new URL(req.url());
    const path = u.pathname.replace("/api/v1", "");
    calls.push({ method: req.method(), path: path + u.search, body: req.postData() ?? "" });
    const step = h[`${req.method()} ${path}`];
    if (typeof step === "function") return step(route);
    if (step) return route.fulfill(step);
    return route.fulfill(err(404, "NOT_FOUND"));
  };
  for (const g of [`**/api/v1/courses/${COURSE}/calendar**`, "**/api/v1/me/calendar/**"]) await p.route(g, handle);
  return calls;
}
