import type { Page, Route } from "@playwright/test";
import { BASE_URL } from "./env";

// Sổ điểm THẬT (US-P6-06): gateway giả theo HỢP ĐỒNG THẬT (`…/gradebook`, `…/grade-entries`, `…/grade-items/{id}/publish`, `…/students/{id}/explain|history`).
// Giữ trạng thái như máy chủ: `PUT` kiểm version, tăng version, trả kết quả của hàng (công thức đơn giản chỉ để dựng dữ liệu thử: HP = 0,4 × tuần + 0,6 × cuối kỳ).
const cors = { "Access-Control-Allow-Origin": BASE_URL, "Access-Control-Allow-Credentials": "true", Vary: "Origin" };
const j = (body: unknown, status = 200) => ({ status, contentType: "application/json", headers: cors, body: JSON.stringify(body) });
const e = (status: number, code: string, details?: unknown) => j({ code, message: code, trace_id: "t", ...(details ? { details } : {}) }, status);

export const WEEKLY = "00000000-0000-7000-8000-0000000008a1";
export const FINAL = "00000000-0000-7000-8000-0000000008a2";
export const EXAMCOL = "00000000-0000-7000-8000-0000000008a3";

const stuId = (i: number) => `00000000-0000-7000-8000-${String(100000000000 + i)}`;
export const student = (i: number) => ({ id: stuId(i), full_name: `Sinh Viên ${String(i).padStart(4, "0")}`, student_code: `B22DCCN${String(100 + i)}` });

type Entry = { score: string; version: number; source: string; updated_by?: string };
export type Opts = {
  students?: number;
  /** sổ đã chốt mà điểm sống khác bản chốt ở sinh viên 1 */
  drift?: boolean;
  scheme?: boolean;
  locked?: boolean;
  warnings?: Array<"PENALTY_NOT_IN_SCHEME" | "BONUS_NOT_IN_SCHEME">;
  status?: string;
  /** kết quả `preview` của nhập XLSX (mặc định: 3 dòng hợp lệ, không lỗi) */
  importOut?: Record<string, unknown>;
  h?: Record<string, (route: Route, next: (r: Route) => Promise<void> | void) => Promise<void> | void>;
};

export const COLUMNS = (published = false) => [
  { item_id: WEEKLY, key: "weekly", name: "Kiểm tra tuần", component_key: "weekly", grp: "QT", max_score: "10", source: "MANUAL", published },
  { item_id: FINAL, key: "final", name: "Thi cuối kỳ", component_key: "final", grp: "CK", max_score: "10", source: "MANUAL", published: false },
  { item_id: EXAMCOL, key: "exam", name: "Bài thi giữa kỳ", component_key: "weekly", grp: "QT", max_score: "10", source: "EXAM", published: true },
];

export async function gradebookApi(p: Page, opts: Opts = {}) {
  const n = opts.students ?? 30;
  const entries = new Map<string, Entry>();
  const columns = COLUMNS();
  const calls: Array<{ method: string; path: string; body: string; idem: string }> = [];
  const scheme = opts.scheme ?? true;
  let locked = opts.locked ?? false;
  let snapVersion = opts.locked ? 2 : 0;
  const versions: Array<{ version: number; students: number; locked_at: string; locked_by: string; unlock_reason: string | null }> = opts.locked ? [{ version: 2, students: n, locked_at: "2026-10-30T03:00:00Z", locked_by: "Lê Thu Hà", unlock_reason: "Phúc khảo sv04" }, { version: 1, students: n, locked_at: "2026-10-29T03:00:00Z", locked_by: "Lê Thu Hà", unlock_reason: null }] : [];
  let lastReason = "";
  const result = (i: number) => {
    const w = entries.get(`${stuId(i)}:${WEEKLY}`);
    const f = entries.get(`${stuId(i)}:${FINAL}`);
    if (!w || !f) return { status: "INCOMPLETE", verdict: "INCOMPLETE:final", missing: f ? ["weekly"] : ["final"] };
    const hp = Math.round((Number(w.score) * 0.4 + Number(f.score) * 0.6) * 2) / 2;
    return { status: "COMPLETE", qt: w.score, hp: String(hp), letter: hp >= 8.5 ? "A" : hp >= 5 ? "C" : "F", verdict: hp < 3 ? "LIET:final<3.0" : "OK", missing: [] };
  };
  const row = (i: number) => ({
    student: student(i),
    entries: Object.fromEntries([WEEKLY, FINAL].flatMap((c) => { const x = entries.get(`${stuId(i)}:${c}`); return x ? [[c, x]] : []; })),
    bonus: "0", penalty: "0", result: scheme ? result(i) : null, ...(locked ? { locked: { version: snapVersion } } : {}), ...(locked && opts.drift && i === 1 ? { drift: true } : {}),
  });
  const handle = async (route: Route, skip = false): Promise<void> => {
    const req = route.request();
    if (req.method() === "OPTIONS") return route.fulfill({ status: 204, headers: { ...cors, "Access-Control-Allow-Headers": "*", "Access-Control-Allow-Methods": "*" } });
    const url = new URL(req.url());
    const path = url.pathname.replace("/api/v1", "");
    const body = req.postData() ?? "";
    const tail = path.replace(/^\/courses\/[^/]+\//, "").replace(/\/students\/[^/]+\//, "/students/*/").replace(/^(gradebook\/snapshots)\/\d+$/, "$1/*");
    const key = `${req.method()} ${tail.replace(/^grade-items\/[^/]+\//, "grade-items/*/")}`;
    calls.push({ method: req.method(), path: path + url.search, body, idem: req.headers()["idempotency-key"] ?? "" });
    const custom = opts.h?.[key];
    if (custom && !skip) return custom(route, (r) => handle(r, true));
    if (key === "GET grade-scheme/status") {
      const state = opts.status ?? (scheme ? "CONFIRMED" : "NO_POLICY");
      return route.fulfill(j({ state, unclear_count: 0, has_policy: state !== "NO_POLICY", pending_draft: false, gradebook: { status: locked ? "LOCKED" : "OPEN", snapshot_version: snapVersion }, ...(state !== "CONFIRMED" ? { finalize_blocked_reason: "Chưa có công thức điểm đã xác nhận" } : {}) }));
    }
    if (key === "GET gradebook") {
      const cursor = Number(url.searchParams.get("cursor") ?? 0);
      const limit = Number(url.searchParams.get("limit") ?? 30);
      const idx = Array.from({ length: Math.min(limit, n - cursor) }, (_, k) => cursor + k + 1);
      const next = cursor + limit < n ? String(cursor + limit) : null;
      return route.fulfill(j({
        scheme: scheme ? { id: "s1", scheme_no: 1 } : null, state: { status: locked ? "LOCKED" : "OPEN", snapshot_version: snapVersion },
        columns: scheme ? columns : [], rows: idx.map(row), next_cursor: next,
        ...(cursor === 0 && opts.warnings?.length ? { warnings: opts.warnings, warning_students: Object.fromEntries(opts.warnings.map((w) => [w, [{ id: stuId(1), full_name: student(1).full_name }]])) } : {}),
      }));
    }
    if (key === "PUT grade-entries") {
      const b = JSON.parse(body) as { item_id: string; student_id: string; score: string | null; version: number | null };
      const k = `${b.student_id}:${b.item_id}`;
      const cur = entries.get(k);
      if (locked) return route.fulfill(e(409, "GRADEBOOK_LOCKED"));
      if (b.item_id === EXAMCOL) return route.fulfill(e(409, "ITEM_NOT_EDITABLE"));
      if (cur && b.version !== cur.version) return route.fulfill(e(409, "VERSION_CONFLICT", { current_version: cur.version, current: { score: cur.score, version: cur.version, updated_by: { name: "Thầy Nam" }, updated_at: new Date(Date.now() - 120_000).toISOString() } }));
      if (!cur && b.version !== null) return route.fulfill(e(409, "VERSION_CONFLICT", { current_version: 0, current: { score: null, version: 0 } }));
      const i = Number(b.student_id.slice(-5));
      if (b.score === null) entries.delete(k);
      else {
        if (Number(b.score) > 10) return route.fulfill(e(422, "SCORE_OUT_OF_RANGE"));
        if (!/^\d{1,3}(\.\d{1,2})?$/.test(b.score)) return route.fulfill(e(422, "SCORE_PRECISION"));
        entries.set(k, { score: b.score, version: (cur?.version ?? 0) + 1, source: "MANUAL" });
      }
      const out = entries.get(k);
      return route.fulfill(j({ entry: out ? { item_id: b.item_id, student_id: b.student_id, ...out } : null, row_result: result(i), bonus: "0", penalty: "0" }));
    }
    if (key === "POST uploads/presign") return route.fulfill(j({ upload_id: "00000000-0000-7000-8000-0000000000a1", method: "PUT", url: `${BASE_URL}/__blob/up1`, headers: {}, blob_key: "k", expires_in: 600 }));
    if (key === "POST grade-entries/import") {
      const b = JSON.parse(body) as { mode: string; mapping: { student_code_col?: string }; skip_errors?: boolean };
      const base = { rows_total: 0, rows_ok: 0, overwrites: 0, unchanged: 0, errors: [], sample: [], headers: ["MSSV", "Họ tên", "Điểm"], suggested_mapping: { header_row: 1, student_code_col: "A", score_col: "C" }, written: 0, skipped_errors: 0 };
      if (!b.mapping.student_code_col) return route.fulfill(j(base));
      const sample = [1, 2, 3].map((i) => ({ student: student(i), before: i === 1 ? "5" : null, after: String(6 + i) }));
      const out = { ...base, rows_total: 3, rows_ok: 3, overwrites: 1, sample, ...opts.importOut };
      if (b.mode === "commit") {
        const ok = (out.errors as unknown[]).length === 0 || b.skip_errors;
        if (!ok) return route.fulfill(e(409, "IMPORT_HAS_ERRORS"));
        for (const r of sample) entries.set(`${r.student.id}:${WEEKLY}`, { score: r.after, version: 1, source: "IMPORT" });
        return route.fulfill(j({ ...out, written: sample.length, skipped_errors: (out.errors as unknown[]).length }));
      }
      return route.fulfill(j(out));
    }
    if (key === "GET gradebook/preview") {
      const all = Array.from({ length: n }, (_, k) => result(k + 1));
      const distribution: Record<string, number> = {};
      for (const r of all) if (r.letter) distribution[r.letter] = (distribution[r.letter] ?? 0) + 1;
      return route.fulfill(j({ summary: { students: n, complete: all.filter((r) => r.status === "COMPLETE").length, incomplete: all.filter((r) => r.status !== "COMPLETE").length, liet: all.filter((r) => r.verdict.startsWith("LIET")).length, distribution }, items: [], next_cursor: null }));
    }
    if (key === "POST gradebook/finalize") {
      if (locked) return route.fulfill(e(409, "GRADEBOOK_LOCKED"));
      const missing = Array.from({ length: n }, (_, k) => k + 1).filter((i) => result(i).status !== "COMPLETE");
      if (missing.length) return route.fulfill(e(409, "GRADES_INCOMPLETE", { total: missing.length, students: missing.slice(0, 30).map((i) => ({ id: stuId(i), missing: [entries.has(`${stuId(i)}:${WEEKLY}`) ? "final" : "weekly"] })) }));
      locked = true;
      snapVersion += 1;
      versions.unshift({ version: snapVersion, students: n, locked_at: new Date().toISOString(), locked_by: "Lê Thu Hà", unlock_reason: lastReason || null });
      return route.fulfill(j({ version: snapVersion, students: n, locked_at: versions[0].locked_at }));
    }
    if (key === "POST gradebook/unlock") {
      const b = JSON.parse(body) as { reason: string };
      if (!locked) return route.fulfill(e(409, "NOT_LOCKED"));
      if ([...b.reason.trim()].length < 10) return route.fulfill(e(422, "VALIDATION_FAILED"));
      locked = false;
      lastReason = b.reason;
      return route.fulfill(j({ status: "OPEN", snapshot_version: snapVersion }));
    }
    if (key === "GET gradebook/snapshots") return route.fulfill(j({ items: versions }));
    if (key === "GET gradebook/snapshots/*") {
      const v = Number(path.split("/").at(-1));
      return route.fulfill(j({ version: v, locked_at: "2026-10-30T03:00:00Z", items: Array.from({ length: Math.min(n, 5) }, (_, k) => ({ student: student(k + 1), numeric_grade: "7", letter: "B", failed_reason: null, qt: "7", ck: "7", bonus: "0", penalty: "0", items: {} })), next_cursor: null }));
    }
    if (key === "GET gradebook/export.xlsx") return route.fulfill({ status: 200, headers: { ...cors, "Content-Type": "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet", "Content-Disposition": 'attachment; filename="so-diem-L1.xlsx"' }, body: "PK" });
    if (key === "POST grade-items/*/publish") {
      const b = JSON.parse(body) as { published: boolean };
      return route.fulfill(j({ item_id: path.split("/").at(-2), published: b.published }));
    }
    if (key === "GET gradebook/students/*/explain") {
      return route.fulfill(j({
        student: student(2), result: { status: "COMPLETE", hp: "7", letter: "B", verdict: "OK", missing: [] },
        lines: [
          { label: "QT₀", expr: "(8 + 7,5) ÷ 2", value: "7,75", filled_by_user: false },
          { label: "Điểm cộng", expr: "min(0,5; 1,0)", value: "0,50", quote: "Mỗi lần phát biểu được cộng 0,25 điểm.", page: 2, filled_by_user: false },
          { label: "Làm tròn", expr: "round(7,20 ÷ 0,5) × 0,5", value: "7,0", filled_by_user: true },
        ],
      }));
    }
    if (key === "GET gradebook/students/*/history") {
      return route.fulfill(j({ items: [{ at: "2026-10-30T03:00:00Z", by: "Lê Thu Hà", action: "set", item: "weekly", from: "8", to: "9" }], next_cursor: null }));
    }
    return route.fulfill(e(404, "NOT_FOUND"));
  };
  await p.route(/\/api\/v1\/courses\/[^/]+\/(gradebook|grade-entries|grade-items\/[^/]+\/publish|grade-scheme\/status|uploads\/presign)/, (r) => handle(r));
  await p.route(/\/__blob\//, (r) => r.fulfill({ status: 200, headers: cors, body: "" }));
  return { calls, entries, puts: () => calls.filter((c) => c.method === "PUT") };
}
