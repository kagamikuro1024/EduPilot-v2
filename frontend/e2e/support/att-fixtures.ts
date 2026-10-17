import type { Page, Route } from "@playwright/test";
import { COURSE, err, json, noContent } from "./cal-fixtures";

// Điểm danh THẬT (US-P5-02) với gateway giả bằng page.route theo HỢP ĐỒNG THẬT (`…/sessions/{sid}/attendance`, `…/participation`).
export { COURSE, err, json, noContent };
const SID = "00000000-0000-7000-8000-0000000005a";
const STU = "00000000-0000-7000-8000-0000000006";
/** Buổi `n`: buổi 2 = đang diễn ra lúc `NOW` của spec. */
export const NOW = Date.parse("2026-10-07T03:00:00Z"); // trước giờ thật để JWT giả (exp = giờ thật + 15 phút) không hết hạn (như cal-fixtures)
export const sessionRow = (n: number, over: Record<string, unknown> = {}) => ({
  id: `${SID}${n}`, session_no: n, starts_at: `2026-10-${String(5 + (n - 1) * 2).padStart(2, "0")}T02:00:00.000Z`, ends_at: `2026-10-${String(5 + (n - 1) * 2).padStart(2, "0")}T04:30:00.000Z`,
  room: "P.402", topic: null, version: 1, attendance: { done_at: null, has_records: false }, ...over,
});
export const sid = (i: number) => `${STU}${String(i).padStart(3, "0")}`;
export const student = (i: number, over: Record<string, unknown> = {}) => ({ id: sid(i), full_name: `Sinh Viên ${String(i).padStart(2, "0")}`, student_code: `B22DCCN${100 + i}`, record: null, speak_count: 0, ...over });
export const students = (n = 30) => Array.from({ length: n }, (_, i) => student(i + 1));
export const gridBody = (sessionNo = 2, list: unknown[] = students(), done: string | null = null) => {
  const r = sessionRow(sessionNo);
  return { session: { id: r.id, session_no: sessionNo, starts_at: r.starts_at, ends_at: r.ends_at, room: r.room, topic: null, attendance_done_at: done }, students: list };
};
export const conflict = (i: number, mine: string, theirs: string, by = "Thầy Nam") => ({ student_id: sid(i), mine: { status: mine }, current: { status: theirs, version: 2, marked_by: { full_name: by }, marked_at: "2026-10-07T02:30:00Z" } });

type Next = (route: Route) => Promise<void> | void;
type Handler = (route: Route, dflt: Next) => Promise<void> | void;
export type Call = { method: string; path: string; body: string };

/**
 * Gateway giả có trạng thái: `PUT` áp từng hàng như máy chủ thật (đổi trạng thái → version + 1), `participation` khử trùng theo `client_event_id`.
 * `h["PUT attendance"]` thay cho hành vi mặc định (lỗi 5xx, 409…).
 */
export async function attApi(p: Page, opts: { sessions?: unknown[]; grid?: unknown; h?: Record<string, Handler> } = {}) {
  const calls: Call[] = [];
  const stored = new Map<string, { status: string; version: number }>();
  const events = new Map<string, string>();
  let doneAt: string | null = null;
  const handle = async (route: Route, skipCustom = false): Promise<void> => {
    const req = route.request();
    if (req.method() === "OPTIONS") return route.fulfill({ status: 204, headers: json({}).headers });
    const path = new URL(req.url()).pathname.replace("/api/v1", "");
    const body = req.postData() ?? "";
    if (!skipCustom) calls.push({ method: req.method(), path, body }); // dflt() chạy lại cùng yêu cầu: không đếm đôi
    const key = `${req.method()} ${path.endsWith("/attendance") ? "attendance" : path.includes("/participation") ? "participation" : "sessions"}`;
    const custom = skipCustom ? undefined : opts.h?.[key];
    if (custom) return custom(route, (r) => handle(r, true));
    if (path.endsWith("/attendance")) {
      if (req.method() === "GET") return route.fulfill(json(opts.grid ?? gridBody(Number(path.split("/sessions/")[1].charAt(path.split("/sessions/")[1].indexOf("/") - 1)))));
      const b = JSON.parse(body) as { records: Array<{ student_id: string; status: string }>; complete: boolean };
      const out = b.records.map((r) => {
        const cur = stored.get(r.student_id);
        const next = cur && cur.status === r.status ? cur : { status: r.status, version: (cur?.version ?? 0) + 1 };
        stored.set(r.student_id, next);
        return { student_id: r.student_id, status: r.status, version: next.version };
      });
      if (b.complete) doneAt = new Date().toISOString();
      return route.fulfill(json({ records: out, done_at: doneAt }));
    }
    if (path.includes("/participation")) {
      if (req.method() === "DELETE") return route.fulfill(noContent);
      const b = JSON.parse(body) as { client_event_id: string };
      const had = events.has(b.client_event_id);
      if (!had) events.set(b.client_event_id, `00000000-0000-7000-8000-00000000e${String(events.size).padStart(3, "0")}`);
      return route.fulfill(json({ id: events.get(b.client_event_id), client_event_id: b.client_event_id }, had ? 200 : 201));
    }
    if (req.method() === "GET") return route.fulfill(json({ items: opts.sessions ?? [sessionRow(1), sessionRow(2), sessionRow(3)], next_cursor: null }));
    return route.fulfill(err(404, "NOT_FOUND"));
  };
  await p.route(`**/api/v1/courses/${COURSE}/sessions**`, (r) => handle(r));
  await p.route(`**/api/v1/courses/${COURSE}/participation**`, (r) => handle(r));
  return { calls, puts: () => calls.filter((c) => c.method === "PUT"), stored };
}
