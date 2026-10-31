import { useQuery } from "@tanstack/react-query";
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
export type ItemDoc = { key: string; name: string; max_score: string; source: "MANUAL" | "EXAM" | "ASSIGNMENT"; exam_id?: string };
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
export type PutComponent = { key: string; name: string; grp: "QT" | "CK"; weight: string; items: Array<{ key: string; name: string; max_score: string; source: "MANUAL" | "EXAM"; exam_id?: string }> };
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
  components: d.components.map((c) => ({ key: c.key, name: c.name, grp: c.grp, weight: c.weight, items: c.items.filter((i) => i.source !== "ASSIGNMENT").map((i) => ({ key: i.key, name: i.name, max_score: i.max_score, source: i.source as "MANUAL" | "EXAM", exam_id: i.exam_id })) })),
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
