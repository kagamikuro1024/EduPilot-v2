"use client";

import { useQuery } from "@tanstack/react-query";
import { apiClient } from "@/shared/data";

// Hợp đồng thật: backend-go/api/openapi.yaml (`/courses/{id}/analytics/exams/*`, `/analytics/exams/compare`). Chỉ bài PUBLISHED; số là chuỗi thập phân.
export type OverviewExam = { id: string; title: string; opens_at: string; max_score: string };
export type Overview = {
  scope: "exam" | "week";
  exams: OverviewExam[];
  submissions: number;
  pass_rate: string | null;
  mean: string | null;
  median: string | null;
  distribution: { from: string; to: string; count: number }[] | null;
};
export type TrendPoint = { exam_id: string; title: string; opens_at: string; submissions: number; pass_rate: string | null; mean: string | null; median: string | null };
export type HardItem = { item_id: string; exam_id: string; position: number; type: "MCQ" | "CODE"; title: string; topic: string; attempts: number; wrong_rate: string; ce_rate: string | null };
export type Declining = { student_id: string; name: string; student_code: string | null; scores: { exam_id: string; title: string; score: string }[] };
export type CompareRow = { course_id: string; class_code: string; name: string; exams: number; submissions: number; enough: boolean; mean: string | null; median: string | null; pass_rate: string | null };

const base = (c: string) => `/courses/${c}/analytics/exams`;

export type Scope = { examId?: string; week?: string };

export function useOverview(course: string, scope: Scope) {
  return useQuery({
    queryKey: ["analytics", "exams", "overview", course, scope.examId ?? "", scope.week ?? ""],
    staleTime: 0, // điểm không cache: mở lại là hỏi lại
    queryFn: async ({ signal }) => (await apiClient.get<Overview>(`${base(course)}/overview`, { signal, query: { exam_id: scope.examId, week: scope.week } })).data,
  });
}

export function useTrend(course: string) {
  return useQuery({ queryKey: ["analytics", "exams", "trend", course], staleTime: 0, queryFn: async ({ signal }) => (await apiClient.get<{ items: TrendPoint[] }>(`${base(course)}/trend`, { signal })).data.items });
}

export function useCompare(enabled: boolean) {
  return useQuery({ queryKey: ["analytics", "exams", "compare"], enabled, staleTime: 0, queryFn: async ({ signal }) => (await apiClient.get<{ items: CompareRow[] }>("/analytics/exams/compare", { signal })).data.items });
}

export const path = { hard: (c: string) => `${base(c)}/hard-items`, declining: (c: string) => `${base(c)}/declining` };

const nf = new Intl.NumberFormat("vi-VN", { maximumFractionDigits: 2 });
export const num = (v: string | null) => (v === null ? "—" : nf.format(Number(v)));
export const pct = (v: string | null) => (v === null ? "—" : `${nf.format(Math.round(Number(v) * 100))}%`);
