"use client";

import { useQuery } from "@tanstack/react-query";
import { apiClient } from "@/shared/data";

// Hợp đồng thật: backend-go/api/openapi.yaml (`/courses/{id}/assignments/{aid}/guidance…`, `/regrade-stale`, `/guidance-flags`).
export type GuidanceStatus = "DRAFT" | "APPROVED" | "REJECTED" | "RETIRED";
export type GuidanceItem = {
  id: string;
  criterion_id: string;
  text: string;
  text_original: string | null;
  direction: "STRICTER" | "LENIENT" | "CLARIFY";
  support: number;
  needs_recheck: boolean;
  status: GuidanceStatus;
  pending_retire: boolean;
  approved_in_version: number | null;
  retired_in_version: number | null;
  version: number;
  created_at: string;
};
export type GuidanceStale = { count: number; editing: number; reviewed: number; versions: number[] };
export type GuidanceOverview = {
  enabled: boolean;
  current_version: number;
  items: GuidanceItem[];
  next_cursor: string | null;
  counts: Partial<Record<"draft" | "approved" | "pending_retire" | "rejected" | "retired", number>>;
  stale: GuidanceStale;
  reflect: { eligible: boolean; results: number; edits: number };
};
export type GuidanceFeature = "GRADING_GUIDANCE" | "THREAD_SUGGEST" | "QUESTION_HINTS" | "REGRESSION_CAPTURE";
export type GuidanceFlag = { feature: GuidanceFeature; enabled: boolean; version: number; can_edit: boolean };
export type GuidanceStatsRow = { version: number; n: number; mae_criterion: string | null; mae_total: string | null; bias_total: string | null; edited_criteria: number; edited_rate: string | null; criteria_counted: number };

const base = (course: string, aid: string) => `/courses/${course}/assignments/${aid}`;

export const guidanceKey = (aid: string) => ["guidance", aid] as const;

export const useGuidance = (course: string, aid: string, enabled = true) =>
  useQuery({
    queryKey: guidanceKey(aid),
    enabled,
    staleTime: 0,
    retry: false,
    queryFn: async ({ signal }) => (await apiClient.get<GuidanceOverview>(`${base(course, aid)}/guidance`, { signal, query: { limit: 100 } })).data,
  });

export const useGuidanceStats = (course: string, aid: string, enabled: boolean) =>
  useQuery({
    queryKey: ["guidance-stats", aid],
    enabled,
    staleTime: 0,
    retry: false,
    queryFn: async ({ signal }) => (await apiClient.get<{ items: GuidanceStatsRow[] }>(`${base(course, aid)}/guidance/stats`, { signal })).data.items,
  });

export const reflectGuidance = async (course: string, aid: string, idempotencyKey: string) =>
  (await apiClient.post<{ job_id: string }>(`${base(course, aid)}/guidance/reflect`, {}, { idempotencyKey })).data;

export const decideGuidance = async (course: string, aid: string, id: string, body: { decision: "APPROVE" | "REJECT" | "RETIRE" | "REVERT"; text?: string; version: number }) =>
  (await apiClient.post<GuidanceItem>(`${base(course, aid)}/guidance/items/${id}/decide`, body)).data;

export const applyGuidance = async (course: string, aid: string, expectedVersion: number, idempotencyKey: string) =>
  (await apiClient.post<{ version: number; stale: GuidanceStale }>(`${base(course, aid)}/guidance/apply`, { expected_version: expectedVersion }, { idempotencyKey })).data;

export type RegradeOut = { job_id?: string; queued: number; skipped: { reviewed: number; edited: number; max_round: number } };
export const regradeStale = async (course: string, aid: string, idempotencyKey: string) =>
  (await apiClient.post<RegradeOut>(`${base(course, aid)}/regrade-stale`, {}, { idempotencyKey })).data;

export const flagsKey = (course: string) => ["guidance-flags", course] as const;
export const useGuidanceFlags = (course: string, enabled: boolean) =>
  useQuery({
    queryKey: flagsKey(course),
    enabled,
    staleTime: 0,
    retry: false,
    queryFn: async ({ signal }) => (await apiClient.get<{ items: GuidanceFlag[] }>(`/courses/${course}/guidance-flags`, { signal })).data.items,
  });
export const setGuidanceFlag = async (course: string, feature: GuidanceFeature, enabled: boolean, version: number) =>
  (await apiClient.put<GuidanceFlag>(`/courses/${course}/guidance-flags/${feature}`, { enabled, version })).data;

/** Chữ hướng của lưu ý: chữ thường dùng cho giảng viên (không thuật ngữ kỹ thuật). */
export const DIRECTION_TEXT: Record<GuidanceItem["direction"], string> = { STRICTER: "Chấm chặt hơn", LENIENT: "Chấm rộng hơn", CLARIFY: "Làm rõ tiêu chí" };
