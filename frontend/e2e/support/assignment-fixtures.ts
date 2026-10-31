import type { Page, Route } from "@playwright/test";
import { BASE_URL } from "./env";

// Bài tập THẬT (US-P7-02) với gateway giả theo HỢP ĐỒNG THẬT (`…/assignments`, `…/{aid}`, `…/publish`, `…/rubric`, `…/rubric/suggest`).
// Gateway giả giữ trạng thái như máy chủ: kiểm `version`, tăng version sau mỗi lần ghi, nhớ `Idempotency-Key` của POST.
export const COURSE = "00000000-0000-7000-8000-00000000c001";
export const A1 = "00000000-0000-7000-8000-0000000a0001";
export const A2 = "00000000-0000-7000-8000-0000000a0002";
const cors = { "Access-Control-Allow-Origin": BASE_URL, "Access-Control-Allow-Credentials": "true", Vary: "Origin" };
const j = (body: unknown, status = 200) => ({ status, contentType: "application/json", headers: cors, body: JSON.stringify(body) });
const e = (status: number, code: string, details?: unknown) => j({ code, message: code, trace_id: "t", ...(details === undefined ? {} : { details }) }, status);

export type Criterion = { id: string; name: string; max_score: string; guidance: string };
export type A = {
  id: string; title: string; description: string; type: string; max_score: string; due_at: string; allow_late: boolean;
  late_penalty_per_day: string; late_max_days: number; appeal_days: number; channels: string[]; submission_code: string;
  reference_answer: string | null; rubric: Criterion[]; teams_class_id: string | null; teams_assignment_id: string | null;
  published_at: string | null; version: number; created_at: string;
  counts: { submitted: number; total: number; graded: number; published: number };
  grade_link?: { component_name: string; component_key: string };
  rubric_locked: boolean; late_rule_locked: boolean;
};

export const assignment = (over: Partial<A> = {}): A => ({
  id: A1,
  title: "Bài tập 03 · Phân tích nhật ký",
  description: "Phân tích nhật ký tấn công và viết báo cáo.",
  type: "ESSAY",
  max_score: "10",
  due_at: "2026-11-20T07:00:00.000Z",
  allow_late: true,
  late_penalty_per_day: "0.5",
  late_max_days: 3,
  appeal_days: 7,
  channels: ["IN_APP"],
  submission_code: "BT03",
  reference_answer: "Đáp án chuẩn của bài tập 03 dùng để gợi ý tiêu chí chấm cho giảng viên.",
  rubric: [
    { id: "c1", name: "Phát hiện dấu hiệu", max_score: "6", guidance: "Nêu đủ ba dấu hiệu." },
    { id: "c2", name: "Đề xuất xử lý", max_score: "4", guidance: "Đề xuất khả thi." },
  ],
  teams_class_id: null,
  teams_assignment_id: null,
  published_at: "2026-10-20T03:00:00.000Z",
  version: 3,
  created_at: "2026-10-18T03:00:00.000Z",
  counts: { submitted: 24, total: 30, graded: 20, published: 12 },
  grade_link: { component_name: "Kiểm tra hằng tuần", component_key: "weekly" },
  rubric_locked: false,
  late_rule_locked: false,
  ...over,
});

export const draftAssignment = (over: Partial<A> = {}): A =>
  assignment({
    id: A2, title: "Bài tập 04 · Mã độc", submission_code: "BT04", published_at: null, version: 1,
    counts: { submitted: 0, total: 30, graded: 0, published: 0 }, grade_link: undefined, due_at: "2026-12-01T07:00:00.000Z", ...over,
  });

export type Call = { method: string; path: string; body: string; headers: Record<string, string> };
type Next = (route: Route) => Promise<void> | void;
type Handler = (route: Route, dflt: Next) => Promise<void> | void;
type Opts = {
  items?: A[];
  /** trang hai của phân trang con trỏ */
  page2?: A[];
  suggest?: Array<{ name: string; guidance: string; max_score: string }>;
  h?: Record<string, Handler>;
};

const SUGGEST = [
  { name: "Mô tả hiện tượng", guidance: "Nêu đúng hiện tượng quan sát được.", max_score: "3" },
  { name: "Phân tích nguyên nhân", guidance: "Chỉ ra nguyên nhân kèm bằng chứng.", max_score: "4" },
  { name: "Đề xuất khắc phục", guidance: "Đề xuất có thể thực hiện được.", max_score: "3" },
];

/** Gateway giả có trạng thái. `h["PUT assignments/:id"]` v.v. thay cho hành vi mặc định (lỗi mạng, 409, chậm…). */
export async function assignmentsApi(p: Page, opts: Opts = {}) {
  const calls: Call[] = [];
  let items: A[] = (opts.items ?? [assignment(), draftAssignment()]).map((a) => ({ ...a }));
  const page2 = (opts.page2 ?? []).map((a) => ({ ...a }));
  const posted = new Map<string, unknown>(); // Idempotency-Key → phản hồi đã trả

  const handle = async (route: Route, skipCustom = false): Promise<void> => {
    const req = route.request();
    if (req.method() === "OPTIONS") return route.fulfill({ status: 204, headers: { ...cors, "Access-Control-Allow-Headers": "*", "Access-Control-Allow-Methods": "*" } });
    const url = new URL(req.url());
    const path = url.pathname.replace("/api/v1", "");
    const body = req.postData() ?? "";
    const tail = path.replace(/^\/courses\/[^/]+\//, "");
    const aid = tail.split("/")[1] ?? "";
    const key = `${req.method()} ${tail.replace(/assignments\/[^/]+/, "assignments/:id")}`;
    if (!skipCustom) calls.push({ method: req.method(), path: path + url.search, body, headers: req.headers() }); // dflt() vào lại: không đếm đôi
    const custom = opts.h?.[key];
    if (custom && !skipCustom) return custom(route, (r) => handle(r, true));
    const idem = req.headers()["idempotency-key"] ?? "";
    const find = () => items.find((a) => a.id === aid);
    const replace = (next: A) => {
      items = items.map((a) => (a.id === next.id ? next : a));
      return next;
    };

    if (key === "GET assignments") {
      if (url.searchParams.get("cursor")) return route.fulfill(j({ items: page2, next_cursor: null }));
      return route.fulfill(j({ items, next_cursor: page2.length ? "c2" : null }));
    }
    if (key === "POST assignments") {
      if (posted.has(idem)) return route.fulfill(j(posted.get(idem), 201));
      const b = JSON.parse(body) as Partial<A> & { rubric?: Criterion[] };
      const made = assignment({
        ...b,
        id: `00000000-0000-7000-8000-0000000a00${10 + items.length}`,
        rubric: (b.rubric ?? []).map((c, i) => ({ id: c.id || `c${i + 1}`, name: c.name, max_score: c.max_score, guidance: c.guidance ?? "" })),
        published_at: null,
        version: 1,
        counts: { submitted: 0, total: 30, graded: 0, published: 0 },
        grade_link: undefined,
      });
      items = [made, ...items];
      posted.set(idem, made);
      return route.fulfill(j(made, 201));
    }
    if (key === "GET assignments/:id") {
      const cur = find();
      return route.fulfill(cur ? j(cur) : e(404, "ASSIGNMENT_NOT_FOUND"));
    }
    if (key === "PUT assignments/:id") {
      const cur = find();
      if (!cur) return route.fulfill(e(404, "ASSIGNMENT_NOT_FOUND"));
      const b = JSON.parse(body) as Partial<A> & { version: number };
      if (b.version !== cur.version) return route.fulfill(e(409, "VERSION_CONFLICT", { current: cur }));
      return route.fulfill(j(replace({ ...cur, ...b, rubric: cur.rubric, version: cur.version + 1 })));
    }
    if (key === "DELETE assignments/:id") {
      const cur = find();
      if (!cur) return route.fulfill(e(404, "ASSIGNMENT_NOT_FOUND"));
      items = items.filter((a) => a.id !== aid);
      return route.fulfill({ status: 204, headers: cors });
    }
    if (key === "POST assignments/:id/publish") {
      const cur = find();
      if (!cur) return route.fulfill(e(404, "ASSIGNMENT_NOT_FOUND"));
      if (posted.has(idem)) return route.fulfill(j(posted.get(idem)));
      if (cur.published_at) return route.fulfill(e(409, "ALREADY_PUBLISHED"));
      if (cur.rubric.length === 0) return route.fulfill(e(422, "RUBRIC_REQUIRED"));
      const out = { assignment: replace({ ...cur, published_at: "2026-10-31T03:00:00.000Z", version: cur.version + 1 }), students: cur.counts.total };
      posted.set(idem, out);
      return route.fulfill(j(out));
    }
    if (key === "PUT assignments/:id/rubric") {
      const cur = find();
      if (!cur) return route.fulfill(e(404, "ASSIGNMENT_NOT_FOUND"));
      const b = JSON.parse(body) as { version: number; criteria: Criterion[] };
      if (b.version !== cur.version) return route.fulfill(e(409, "VERSION_CONFLICT", { current: cur }));
      const sum = b.criteria.reduce((a, c) => a + Math.round(Number(c.max_score) * 100), 0);
      if (sum !== Math.round(Number(cur.max_score) * 100)) {
        return route.fulfill(e(422, "RUBRIC_INVALID", { sum: String(sum / 100), max_score: cur.max_score }));
      }
      const rubric = b.criteria.map((c, i) => ({ id: c.id || `c${i + 1}`, name: c.name, max_score: c.max_score, guidance: c.guidance ?? "" }));
      return route.fulfill(j(replace({ ...cur, rubric, version: cur.version + 1 })));
    }
    if (key === "POST assignments/:id/rubric/suggest") {
      const cur = find();
      if (!cur) return route.fulfill(e(404, "ASSIGNMENT_NOT_FOUND"));
      if (!cur.reference_answer || cur.reference_answer.length < 50) return route.fulfill(e(422, "REFERENCE_ANSWER_REQUIRED"));
      return route.fulfill(j({ criteria: opts.suggest ?? SUGGEST }));
    }
    return route.fulfill(e(404, "NOT_FOUND"));
  };

  await p.route(/\/api\/v1\/courses\/[^/]+\/assignments(\?|$|\/)/, (r) => handle(r));
  return {
    calls,
    items: () => items,
    posts: () => calls.filter((c) => c.method === "POST"),
    puts: () => calls.filter((c) => c.method === "PUT"),
  };
}
