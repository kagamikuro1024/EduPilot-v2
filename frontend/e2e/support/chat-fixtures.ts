import type { Page, Route } from "@playwright/test";
import { BASE_URL } from "./env";

// Chat riêng THẬT (US-P3-05) với gateway giả bằng page.route: theo HỢP ĐỒNG THẬT (backend-go/api/openapi.yaml `/chat/*`). Không dùng máy chủ giả dùng chung.
export const COURSE = "00000000-0000-7000-8000-00000000c001";
export const SID = "00000000-0000-7000-8000-0000000000d1";
export const MID = "00000000-0000-7000-8000-0000000000e1";
const CORS = { "Access-Control-Allow-Origin": BASE_URL, "Access-Control-Allow-Credentials": "true", Vary: "Origin", "Access-Control-Allow-Headers": "authorization,content-type,x-request-id,idempotency-key,last-event-id,accept", "Access-Control-Allow-Methods": "GET,POST,PUT,DELETE,OPTIONS" };

export const json = (body: unknown, status = 200) => ({ status, contentType: "application/json", headers: CORS, body: JSON.stringify(body) });
export const noContent = { status: 204, headers: CORS, body: "" };
export const frame = (event: string, data: unknown, id?: string) => `${id ? `id: ${id}\n` : ""}event: ${event}\ndata: ${JSON.stringify(data)}\n\n`;
export const sse = (...frames: string[]) => ({ status: 200, headers: { ...CORS, "Content-Type": "text/event-stream", "Cache-Control": "no-cache" }, body: frames.join("") });
export const session = (over: Record<string, unknown> = {}) => ({ id: SID, title: "Quy chế thi", last_message_at: "2026-10-11T02:00:00Z", document_id: null, ...over });
export const msg = (over: Record<string, unknown> = {}) => ({
  id: MID, role: "ASSISTANT", content: "Theo quy chế [1], sinh viên không được mang tài liệu.", streaming: false, stream_status: "DONE", attempt: 1,
  citations: [{ n: 1, document_id: "00000000-0000-7000-8000-0000000000f1", title: "Quy chế học vụ", page_no: 3, snippet: "Sinh viên không được mang tài liệu vào phòng thi." }],
  blocks: [], low_confidence: false, degraded: false, masked_count: 0, feedback: null, error_code: null, created_at: "2026-10-11T02:00:00Z", ...over,
});

type Handler = (route: Route) => Promise<void> | void;
type Fulfill = Parameters<Route["fulfill"]>[0];
/** Gắn các route chat; `h` ghi đè theo khoá "METHOD /đường-dẫn-sau-/api/v1" (khớp đuôi). OPTIONS luôn 204. */
export async function chatApi(page: Page, h: Record<string, Handler | Fulfill> = {}) {
  const calls: Array<{ method: string; path: string; body: string; headers: Record<string, string> }> = [];
  await page.route("**/api/v1/chat/**", async (route) => {
    const req = route.request();
    if (req.method() === "OPTIONS") return route.fulfill({ status: 204, headers: CORS });
    const path = new URL(req.url()).pathname.replace("/api/v1", "");
    calls.push({ method: req.method(), path, body: req.postData() ?? "", headers: req.headers() });
    const key = Object.keys(h).find((k) => k === `${req.method()} ${path}`) ?? Object.keys(h).find((k) => k.endsWith("*") && `${req.method()} ${path}`.startsWith(k.slice(0, -1)));
    const step = key ? h[key] : undefined;
    if (typeof step === "function") return step(route);
    if (step) return route.fulfill(step);
    return route.fulfill(json({ code: "NOT_FOUND", message: "m", trace_id: "b".repeat(32) }, 404));
  });
  return calls;
}
