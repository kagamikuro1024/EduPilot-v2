"use client";

import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { ApiError, apiClient } from "@/shared/data";

// Hợp đồng thật: backend-go/api/openapi.yaml (`/courses/{id}/insights/knowledge-gap/*`).
export type Topic = {
  rank: number;
  name: string;
  why: string;
  score: number;
  signals: { students: number; questions: number; low_confidence_rate: number; not_helpful_rate: number; practice_wrong_rate: number };
  recommendation: string;
  examples: string[];
  few_asked?: boolean;
  named_by: "llm" | "fallback";
  chapter: string | null;
  week: number | null;
  keywords: string[];
};
export type Report = {
  id: string;
  period_from: string | null;
  period_to: string | null;
  data_watermark: string;
  created_at: string;
  payload: { topics: Topic[]; uncovered: { text: string; count: number }[]; counts: { questions: number; students: number; clusters: number }; period_days: number };
};
export type Latest = {
  report: Report;
  has_new_data: boolean;
  previous?: { id: string; created_at: string; new_topics: string[]; moved?: { name: string; from: number; to: number }[] };
};
export type Generated = { job_id?: string; report_id?: string; reused?: boolean };

export const latestKey = (course: string) => ["insights", "latest", course] as const;
const base = (course: string) => `/courses/${course}/insights/knowledge-gap`;

/** Báo cáo mới nhất; lớp chưa có báo cáo (404) → `null`, không phải lỗi. */
export function useLatest(course: string | null) {
  return useQuery({
    queryKey: latestKey(course ?? ""),
    enabled: !!course,
    queryFn: async ({ signal }): Promise<Latest | null> => {
      try {
        return (await apiClient.get<Latest>(`${base(course!)}/latest`, { signal })).data;
      } catch (e) {
        if (e instanceof ApiError && e.status === 404) return null;
        throw e;
      }
    },
  });
}

export function useGenerate(course: string) {
  return useMutation({ mutationFn: async () => (await apiClient.post<Generated>(`${base(course)}/generate`, {})).data });
}

export function useRefreshLatest(course: string) {
  const qc = useQueryClient();
  return () => qc.invalidateQueries({ queryKey: latestKey(course) });
}

/** Tải Markdown của báo cáo (Blob) rồi kích hoạt tải xuống. */
export async function downloadMarkdown(course: string, reportId: string): Promise<void> {
  const r = await apiClient.get<Blob>(`${base(course)}/${reportId}/export`, { query: { format: "md" }, blob: true });
  const url = URL.createObjectURL(r.data);
  const a = document.createElement("a");
  a.href = url;
  a.download = "lo-hong-kien-thuc.md";
  a.click();
  URL.revokeObjectURL(url);
}
