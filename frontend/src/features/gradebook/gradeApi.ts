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
