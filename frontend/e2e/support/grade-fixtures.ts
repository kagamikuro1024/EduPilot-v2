import type { Page, Route } from "@playwright/test";
import { BASE_URL } from "./env";

// Công thức điểm THẬT (US-P6-04) với gateway giả theo HỢP ĐỒNG THẬT (`…/grade-scheme`, `…/impact`, `…/confirm`, `…/documents/{id}/chunks`).
// Gateway giả giữ bản nháp như máy chủ: `PUT` kiểm version, tính lại mục chưa rõ (thiếu làm tròn, tổng trọng số), version + 1.
export const COURSE = "00000000-0000-7000-8000-00000000c001";
export const COURSE_2 = "00000000-0000-7000-8000-00000000c002";
const DOC = "00000000-0000-7000-8000-0000000007d1";
const cors = { "Access-Control-Allow-Origin": BASE_URL, "Access-Control-Allow-Credentials": "true", Vary: "Origin" };
const j = (body: unknown, status = 200) => ({ status, contentType: "application/json", headers: cors, body: JSON.stringify(body) });
const e = (status: number, code: string, details?: unknown) => j({ code, message: code, trace_id: "t", ...(details ? { details } : {}) }, status);

type Comp = { key: string; name: string; grp: "QT" | "CK"; weight: string; items: Array<{ key: string; name: string; max_score: string; source: string }>; citations: Array<{ page: number; quote: string }>; filled_by_user: boolean };
type Rule = { kind: string; applicable: boolean; params: Record<string, unknown>; citations: Array<{ page: number; quote: string }>; filled_by_user: boolean };
export type Doc = { id: string; scheme_no: number; status: string; version: number; source_title?: string; source_document_id?: string; confirmed_at?: string; components: Comp[]; rules: Rule[]; unclear_items: unknown[]; warnings: string[] };

export const Q = {
  weekly: "Điểm quá trình gồm bài kiểm tra hằng tuần, chiếm 20% điểm học phần.",
  mid: "Bài kiểm tra giữa kỳ chiếm 20% điểm học phần.",
  final: "Thi kết thúc học phần chiếm 60% điểm học phần.",
  bonus: "Mỗi lần phát biểu được cộng 0,25 điểm, tổng không quá 1,0 điểm.",
  letters: "Thang điểm chữ: A từ 8,5; B từ 7,0; C từ 5,5; D từ 4,0; dưới 4,0 là F.",
};
const comp = (key: string, name: string, grp: "QT" | "CK", weight: string, page: number, quote: string): Comp => ({ key, name, grp, weight, items: [{ key, name, max_score: "10", source: "MANUAL" }], citations: [{ page, quote }], filled_by_user: false });
const rule = (kind: string, params: Record<string, unknown>, page: number, quote: string, applicable = true): Rule => ({ kind, applicable, params, citations: [{ page, quote }], filled_by_user: false });

/** Bản nháp kiểu lớp 2: đủ mọi mục trừ làm tròn. */
export const draftLop2 = (over: Partial<Doc> = {}): Doc => ({
  id: "00000000-0000-7000-8000-0000000000d2", scheme_no: 1, status: "DRAFT", version: 1, source_title: "quy-che-lop2", source_document_id: DOC,
  components: [comp("weekly", "Kiểm tra hằng tuần", "QT", "20", 1, Q.weekly), comp("midterm", "Giữa kỳ", "QT", "20", 1, Q.mid), comp("final", "Thi cuối kỳ", "CK", "60", 1, Q.final)],
  rules: [
    rule("BONUS", { source: "PARTICIPATION", per_speak: "0.25", cap: "1.0" }, 2, Q.bonus),
    rule("PENALTY", { source: "ABSENCE", free: 2, per: "0.5" }, 2, "Từ buổi vắng thứ ba, mỗi buổi trừ 0,5 điểm quá trình."),
    rule("LETTER_SCALE", { bands: [{ min: "8.5", letter: "A" }, { min: "4", letter: "D" }, { min: "0", letter: "F" }] }, 2, Q.letters),
  ],
  unclear_items: [{ source: "CODE", code: "ROUNDING_MISSING", target: "rules.ROUNDING", message: "Quy chế chưa nêu cách làm tròn: điền bước làm tròn." }],
  warnings: [], ...over,
});
/** Bản đã xác nhận (đủ mục, làm tròn 0,5). */
export const confirmedLop1 = (over: Partial<Doc> = {}): Doc => {
  const d = draftLop2({ id: "00000000-0000-7000-8000-0000000000d1", status: "CONFIRMED", confirmed_at: "2026-10-05T03:00:00.000Z", unclear_items: [], ...over });
  d.rules.push(rule("ROUNDING", { step: "0.5", mode: "HALF_UP" }, 3, "Điểm học phần làm tròn đến 0,5."));
  return d;
};

export const IMPACT = {
  students_total: 30, students_computable: 30, changed_vs_confirmed: 12, entries_carried: 60, entries_dropped: [],
  samples: [
    { student: { id: "s6", full_name: "Sinh Viên 06" }, before: { hp: "10", letter: "A" }, after: { hp: "9.8", letter: "A" } },
    { student: { id: "s15", full_name: "Sinh Viên 15" }, before: { hp: "7", letter: "B" }, after: { hp: "6.8", letter: "B" } },
    { student: { id: "s30", full_name: "Sinh Viên 30" }, before: { hp: "3", letter: "F" }, after: { hp: "3.2", letter: "F" } },
  ],
};

export type Call = { method: string; path: string; body: string; headers: Record<string, string> };
type Next = (route: Route) => Promise<void> | void;
type Handler = (route: Route, dflt: Next) => Promise<void> | void;
type Opts = { draft?: Doc | null; confirmed?: Doc | null; impact?: unknown; h?: Record<string, Handler>; view?: unknown };

const recompute = (d: Doc) => {
  const un: unknown[] = [];
  const sum = d.components.reduce((a, c) => a + Number(c.weight), 0);
  if (Math.abs(sum - 100) > 0.001) un.push({ source: "CODE", code: "WEIGHT_SUM", target: "components", message: `Tổng trọng số là ${sum}, cần 100.` });
  if (!d.rules.some((r) => r.kind === "ROUNDING")) un.push({ source: "CODE", code: "ROUNDING_MISSING", target: "rules.ROUNDING", message: "Quy chế chưa nêu cách làm tròn: điền bước làm tròn." });
  d.unclear_items = un;
};

/** Gateway giả có trạng thái. `h["PUT grade-scheme"]` v.v. thay cho hành vi mặc định (lỗi mạng, 409…). */
export async function gradeApi(p: Page, opts: Opts = {}) {
  const calls: Call[] = [];
  let draft: Doc | null = opts.draft === undefined ? draftLop2() : opts.draft;
  let confirmed: Doc | null = opts.confirmed ?? null;
  const handle = async (route: Route, skipCustom = false): Promise<void> => {
    const req = route.request();
    if (req.method() === "OPTIONS") return route.fulfill({ status: 204, headers: { ...cors, "Access-Control-Allow-Headers": "*", "Access-Control-Allow-Methods": "*" } });
    const url = new URL(req.url());
    const path = url.pathname.replace("/api/v1", "");
    const body = req.postData() ?? "";
    const tail = path.replace(/^\/courses\/[^/]+\//, "");
    const key = `${req.method()} ${tail.startsWith("documents") ? (tail.endsWith("/chunks") || tail.includes("/chunks?") ? "chunks" : "document") : tail}`;
    calls.push({ method: req.method(), path: path + url.search, body, headers: req.headers() });
    const custom = opts.h?.[key];
    if (custom && !skipCustom) return custom(route, (r) => handle(r, true));
    if (key === "GET grade-scheme") {
      if (opts.view) return route.fulfill(j(opts.view));
      const view: Record<string, unknown> = { confirmed, draft };
      if (url.searchParams.get("diff") === "confirmed" && confirmed && draft) {
        view.diff = { components_added: [], components_removed: [], weight_changed: [{ key: "midterm", from: "20", to: "15" }], rules_changed: [{ kind: "ROUNDING", from: { applicable: true, params: { step: "0.5" } }, to: null }], weight_sum_from: "100", weight_sum_to: "95" };
      }
      return route.fulfill(j(view));
    }
    if (key === "PUT grade-scheme") {
      const b = JSON.parse(body) as { version: number; components: Array<{ key: string; name: string; grp: "QT" | "CK"; weight: string; items?: Comp["items"] }>; rules: Array<{ kind: string; applicable?: boolean; params?: Record<string, unknown> }> };
      if (!draft) return route.fulfill(e(409, "NO_DRAFT"));
      if (b.version !== draft.version) return route.fulfill(e(409, "VERSION_CONFLICT", { current_version: draft.version, current: draft }));
      draft = {
        ...draft, version: draft.version + 1,
        components: b.components.map((c) => ({ ...c, items: c.items?.length ? c.items : [{ key: c.key, name: c.name, max_score: "10", source: "MANUAL" }], citations: draft!.components.find((x) => x.key === c.key && x.weight === c.weight && x.name === c.name)?.citations ?? [], filled_by_user: !draft!.components.some((x) => x.key === c.key && x.weight === c.weight && x.name === c.name) })),
        rules: b.rules.map((r) => ({ kind: r.kind, applicable: r.applicable ?? true, params: r.params ?? {}, citations: [], filled_by_user: true })),
      };
      recompute(draft);
      return route.fulfill(j(draft));
    }
    if (key === "POST grade-scheme/impact") return route.fulfill(j(opts.impact ?? IMPACT));
    if (key === "POST grade-scheme/confirm") {
      const b = JSON.parse(body) as { version: number };
      if (!draft) return route.fulfill(e(409, "NO_DRAFT"));
      if (b.version !== draft.version) return route.fulfill(e(409, "VERSION_CONFLICT", { current_version: draft.version, current: draft }));
      if (draft.unclear_items.length) return route.fulfill(e(422, "SCHEME_HAS_UNCLEAR", { codes: ["ROUNDING_MISSING"] }));
      confirmed = { ...draft, status: "CONFIRMED", confirmed_at: "2026-10-31T03:00:00.000Z", version: draft.version + 1 };
      draft = null;
      return route.fulfill(j({ scheme_id: confirmed.id, scheme_no: confirmed.scheme_no, confirmed_at: confirmed.confirmed_at, impact: opts.impact ?? IMPACT }));
    }
    if (key === "GET document") return route.fulfill(j({ id: DOC, title: "quy-che-lop2", preview_url: `${BASE_URL}/__pdf/quy-che.pdf?sig=1` }));
    if (key === "GET chunks") {
      return route.fulfill(j({
        items: [
          { id: "c1", ord: 0, page_no: 1, heading: null, text: `Điều 1. Cách tính điểm.\n${Q.weekly} ${Q.mid} ${Q.final}`, audience: "ALL" },
          { id: "c2", ord: 1, page_no: 2, heading: null, text: `Điều 2. Điểm cộng, trừ.\n${Q.bonus} Từ buổi vắng thứ ba, mỗi buổi trừ 0,5 điểm quá trình. ${Q.letters}`, audience: "ALL" },
        ],
        next_cursor: null,
      }));
    }
    return route.fulfill(e(404, "NOT_FOUND"));
  };
  await p.route(new RegExp(`/api/v1/courses/[^/]+/(grade-scheme|documents/${DOC}(/chunks)?(\\?|$))`), (r) => handle(r)); // chỉ tài liệu nguồn của công thức: không bắt các route tài liệu khác
  return { calls, set: (d: { draft?: Doc | null; confirmed?: Doc | null }) => { if (d.draft !== undefined) draft = d.draft; if (d.confirmed !== undefined) confirmed = d.confirmed; }, puts: () => calls.filter((c) => c.method === "PUT") };
}
