import { useInfiniteQuery, useQuery } from "@tanstack/react-query";
import { apiClient } from "@/shared/data";

// Hợp đồng thật: backend-go/api/openapi.yaml (`GET /courses/{id}/grade-scheme/status`).
export type SchemeState = "NO_POLICY" | "NOT_EXTRACTED" | "DRAFT_UNCLEAR" | "DRAFT_READY" | "CONFIRMED";
export type SchemeStatus = {
  state: SchemeState;
  unclear_count: number;
  has_policy: boolean;
  confirmed_at?: string;
  scheme_no?: number;
  pending_draft: boolean;
  gradebook: { status: "OPEN" | "LOCKED"; snapshot_version: number };
  finalize_blocked_reason?: string;
};

export const schemeStatusKey = (course: string) => ["grade-scheme-status", course] as const;

/** Trạng thái công thức của lớp (Staff). Lỗi / chưa tải → không có banner: không chặn việc đang làm. */
export const useSchemeStatus = (course: string) =>
  useQuery({
    queryKey: schemeStatusKey(course),
    staleTime: 30_000,
    retry: false,
    refetchOnWindowFocus: false,
    queryFn: async ({ signal }) => (await apiClient.get<SchemeStatus>(`/courses/${course}/grade-scheme/status`, { signal })).data,
  });

// ---- công thức: đọc / sửa / xem thử / xác nhận (US-P6-04) -----------------------------------------------------------------------------------------
export type Citation = { page: number; quote: string };
export type ItemSource = "MANUAL" | "EXAM" | "ASSIGNMENT";
export type ItemDoc = { key: string; name: string; max_score: string; source: ItemSource; exam_id?: string; assignment_id?: string };
export type ComponentDoc = { key: string; name: string; grp: "QT" | "CK"; weight: string; items: ItemDoc[]; citations: Citation[]; filled_by_user: boolean };
export type RuleKind = "BONUS" | "PENALTY" | "ROUNDING" | "FAIL_CONDITION" | "LETTER_SCALE" | "MISSING_SCORE";
export type RuleDoc = { kind: RuleKind; applicable: boolean; params: Record<string, unknown>; citations: Citation[]; filled_by_user: boolean };
export type Unclear = { source: "CODE" | "LLM"; code?: string; target: string; message: string };
export type SchemeDoc = {
  id: string;
  scheme_no: number;
  status: "DRAFT" | "CONFIRMED" | "SUPERSEDED";
  version: number;
  source_title?: string;
  source_document_id?: string;
  confirmed_at?: string;
  components: ComponentDoc[];
  rules: RuleDoc[];
  unclear_items: Unclear[];
  warnings: string[];
};
export type SchemeDiff = {
  components_added: string[];
  components_removed: string[];
  weight_changed: Array<{ key: string; from: string; to: string }>;
  rules_changed: Array<{ kind: string; from: unknown; to: unknown }>;
  weight_sum_from: string;
  weight_sum_to: string;
};
export type SchemesView = { confirmed: SchemeDoc | null; draft: SchemeDoc | null; diff?: SchemeDiff };

/** Dạng gửi lên `PUT` / `impact` (không có trích dẫn / cờ: máy chủ tự tính). */
export type PutItem = { key: string; name: string; max_score: string; source: ItemSource; exam_id?: string; assignment_id?: string };
export type PutComponent = { key: string; name: string; grp: "QT" | "CK"; weight: string; items: PutItem[] };
export type PutRule = { kind: RuleKind; applicable: boolean; params: Record<string, unknown> };
export type Working = { components: PutComponent[]; rules: PutRule[] };

export type Impact = {
  students_total: number;
  students_computable: number;
  changed_vs_confirmed: number;
  entries_carried: number;
  entries_dropped: Array<{ item_key: string; count: number; reason: "ITEM_REMOVED" | "MAX_CHANGED" | "SOURCE_CHANGED" }>;
  samples: Array<{ student: { id: string; full_name: string }; before: { hp: string; letter: string } | null; after: { hp: string; letter: string } }>;
};

export const schemesKey = (course: string, diff: boolean) => ["grade-schemes", course, diff] as const;

export const useSchemes = (course: string, diff: boolean, enabled = true) =>
  useQuery({
    queryKey: schemesKey(course, diff),
    enabled,
    staleTime: 0,
    refetchOnWindowFocus: false,
    queryFn: async ({ signal }) => (await apiClient.get<SchemesView>(`/courses/${course}/grade-scheme`, { signal, query: diff ? { diff: "confirmed" } : undefined })).data,
  });

export const toWorking = (d: SchemeDoc): Working => ({
  components: d.components.map((c) => ({ key: c.key, name: c.name, grp: c.grp, weight: c.weight, items: c.items.map((i) => ({ key: i.key, name: i.name, max_score: i.max_score, source: i.source, exam_id: i.exam_id, assignment_id: i.assignment_id })) })),
  rules: d.rules.map((r) => ({ kind: r.kind, applicable: r.applicable, params: r.params })),
});

export const putScheme = async (course: string, version: number, w: Working) =>
  (await apiClient.put<SchemeDoc>(`/courses/${course}/grade-scheme`, { version, ...w })).data;

export const postImpact = async (course: string, w: Working, signal?: AbortSignal) =>
  (await apiClient.post<Impact>(`/courses/${course}/grade-scheme/impact`, w, { signal })).data;

export const confirmScheme = async (course: string, version: number, idempotencyKey: string) =>
  (await apiClient.post<{ scheme_id: string; scheme_no: number; confirmed_at: string; impact: Impact }>(`/courses/${course}/grade-scheme/confirm`, { version }, { idempotencyKey })).data;

/** Văn bản từng trang của tài liệu nguồn (Staff; `GET …/documents/{id}/chunks`, đoạn cùng trang nối lại). */
export const useDocPages = (course: string, docId: string | undefined) =>
  useQuery({
    queryKey: ["grade-doc-pages", course, docId],
    enabled: Boolean(docId),
    staleTime: 5 * 60_000,
    queryFn: async ({ signal }) => {
      const pages: Record<number, string> = {};
      let cursor: string | undefined;
      for (let i = 0; i < 50; i++) {
        const r = (await apiClient.get<{ items: Array<{ page_no: number | null; text: string }>; next_cursor: string | null }>(`/courses/${course}/documents/${docId}/chunks`, { signal, query: { limit: 100, cursor } })).data;
        for (const c of r.items) pages[c.page_no ?? 1] = pages[c.page_no ?? 1] ? `${pages[c.page_no ?? 1]}\n${c.text}` : c.text;
        if (!r.next_cursor) break;
        cursor = r.next_cursor;
      }
      return pages;
    },
  });

/** URL ký sẵn (xem trực tiếp, 5 phút) của tệp PDF nguồn: `GET …/documents/{id}` → `preview_url`. Không có (tệp mất / không phải PDF) → không có liên kết. */
export const useDocPreview = (course: string, docId: string | undefined) =>
  useQuery({
    queryKey: ["grade-doc-preview", course, docId],
    enabled: Boolean(docId),
    staleTime: 4 * 60_000,
    retry: false,
    queryFn: async ({ signal }) => (await apiClient.get<{ preview_url?: string }>(`/courses/${course}/documents/${docId}`, { signal })).data.preview_url ?? null,
  });

// ---- bảng điểm (US-P6-06) ------------------------------------------------------------------------------------------------------------------------
export type Column = { item_id: string; key: string; name: string; component_key: string; grp: "QT" | "CK"; max_score: string; source: "MANUAL" | "EXAM" | "ASSIGNMENT"; published: boolean };
export type GEntry = { score: string; version: number; source: string; updated_by?: string };
export type GResult = { status: "COMPLETE" | "INCOMPLETE" | "INVALID"; qt?: string; hp?: string; letter?: string; verdict: string; missing: string[] };
export type GRow = { student: { id: string; full_name: string; student_code: string }; entries: Record<string, GEntry>; bonus: string; penalty: string; result: GResult | null; locked?: { version: number }; drift?: boolean };
export type GradebookPage = {
  scheme: { id: string; scheme_no: number } | null;
  state: { status: "OPEN" | "LOCKED"; snapshot_version: number };
  columns: Column[];
  rows: GRow[];
  next_cursor: string | null;
  warnings?: Array<"PENALTY_NOT_IN_SCHEME" | "BONUS_NOT_IN_SCHEME">;
  warning_students?: Record<string, Array<{ id: string; full_name: string }>>;
};
export type PutEntryOut = { entry: { item_id: string; student_id: string; score: string; version: number; source: string } | null; row_result: GResult | null; bonus: string; penalty: string };
export type Explain = { student: { id: string; full_name: string; student_code: string }; result: GResult; lines: Array<{ label: string; expr: string; value: string; quote?: string; page?: number; filled_by_user: boolean }>; locked?: { version: number } };
export type HistoryPage = { items: Array<{ at: string; by: string; action: string; item: string; from: string | null; to: string | null }>; next_cursor: string | null };

export const gradebookKey = (course: string) => ["gradebook", course] as const;

export const useGradebook = (course: string) =>
  useInfiniteQuery({
    queryKey: gradebookKey(course),
    initialPageParam: null as string | null,
    staleTime: 0,
    refetchOnWindowFocus: false,
    queryFn: async ({ pageParam, signal }) => (await apiClient.get<GradebookPage>(`/courses/${course}/gradebook`, { signal, query: { limit: 100, cursor: pageParam } })).data,
    getNextPageParam: (last) => last.next_cursor ?? undefined,
  });

export const putEntry = async (course: string, body: { item_id: string; student_id: string; score: string | null; version: number | null }) =>
  (await apiClient.put<PutEntryOut>(`/courses/${course}/grade-entries`, body)).data;

export const publishItem = async (course: string, item: string, published: boolean) =>
  (await apiClient.post<{ item_id: string; published: boolean }>(`/courses/${course}/grade-items/${item}/publish`, { published })).data;

export const useExplain = (course: string, student: string | null) =>
  useQuery({
    queryKey: ["grade-explain", course, student],
    enabled: Boolean(student),
    staleTime: 0,
    queryFn: async ({ signal }) => (await apiClient.get<Explain>(`/courses/${course}/gradebook/students/${student}/explain`, { signal })).data,
  });

export const useHistory = (course: string, student: string | null) =>
  useInfiniteQuery({
    queryKey: ["grade-history", course, student],
    enabled: Boolean(student),
    initialPageParam: null as string | null,
    staleTime: 0,
    queryFn: async ({ pageParam, signal }) => (await apiClient.get<HistoryPage>(`/courses/${course}/gradebook/students/${student}/history`, { signal, query: { cursor: pageParam } })).data,
    getNextPageParam: (last) => last.next_cursor ?? undefined,
  });

// ---- nhập / xuất XLSX (US-P6-07) -----------------------------------------------------------------------------------------------------------------
export type ImportMapping = { header_row: number; student_code_col: string; score_col: string };
export type ImportRowError = { row: number; student_code: string; code: string; message: string };
export type ImportOut = {
  rows_total: number;
  rows_ok: number;
  overwrites: number;
  unchanged: number;
  errors: ImportRowError[];
  sample: Array<{ student: { id: string; full_name: string; student_code: string }; before: string | null; after: string }>;
  headers: string[];
  suggested_mapping: ImportMapping;
  written: number;
  skipped_errors: number;
};

export const importEntries = async (course: string, body: { upload_id: string; item_id: string; mode: "preview" | "commit"; mapping: Partial<ImportMapping> & { header_row: number }; skip_errors?: boolean }, idempotencyKey?: string) =>
  (await apiClient.post<ImportOut>(`/courses/${course}/grade-entries/import`, body, { idempotencyKey })).data;

export const exportXlsx = async (course: string) => (await apiClient.get<Blob>(`/courses/${course}/gradebook/export.xlsx`, { blob: true })).data;

// ---- chốt / mở khoá / bản chốt (US-P6-08) -----------------------------------------------------------------------------------------------------
export type PreviewSummary = { students: number; complete: number; incomplete: number; liet: number; distribution: Record<string, number> };
export type SnapshotVersion = { version: number; students: number; locked_at: string; locked_by: string; unlock_reason: string | null };
export type SnapshotRow = { student: { id: string; full_name: string; student_code: string }; numeric_grade: string; letter: string; failed_reason: string | null };
export type Incomplete = { total: number; students: Array<{ id: string; missing: string[] }> };

export const snapshotsKey = (course: string) => ["grade-snapshots", course] as const;

export const usePreviewSummary = (course: string, enabled: boolean) =>
  useQuery({
    queryKey: ["grade-preview-summary", course],
    enabled,
    staleTime: 0,
    queryFn: async ({ signal }) => (await apiClient.get<{ summary: PreviewSummary }>(`/courses/${course}/gradebook/preview`, { signal, query: { limit: 1 } })).data.summary,
  });

export const useSnapshots = (course: string, enabled: boolean) =>
  useQuery({
    queryKey: snapshotsKey(course),
    enabled,
    staleTime: 0,
    queryFn: async ({ signal }) => (await apiClient.get<{ items: SnapshotVersion[] }>(`/courses/${course}/gradebook/snapshots`, { signal })).data.items,
  });

export const useSnapshotRows = (course: string, version: number | null) =>
  useInfiniteQuery({
    queryKey: ["grade-snapshot", course, version],
    enabled: version !== null,
    initialPageParam: null as string | null,
    staleTime: 0,
    queryFn: async ({ pageParam, signal }) => (await apiClient.get<{ items: SnapshotRow[]; next_cursor: string | null }>(`/courses/${course}/gradebook/snapshots/${version}`, { signal, query: { limit: 50, cursor: pageParam } })).data,
    getNextPageParam: (last) => last.next_cursor ?? undefined,
  });

export const finalizeGradebook = async (course: string, idempotencyKey: string) =>
  (await apiClient.post<{ version: number; students: number; locked_at: string }>(`/courses/${course}/gradebook/finalize`, {}, { idempotencyKey })).data;

export const unlockGradebook = async (course: string, reason: string, idempotencyKey: string) =>
  (await apiClient.post<{ status: "OPEN"; snapshot_version: number }>(`/courses/${course}/gradebook/unlock`, { reason }, { idempotencyKey })).data;
