"use client";

import { useQuery } from "@tanstack/react-query";
import { apiClient, useCursorList } from "@/shared/data";

// Hợp đồng thật: backend-go/api/openapi.yaml (`/courses/{id}/assignments…`, 8 thao tác). Điểm là CHUỖI thập phân, không đổi sang số thực.
export type Channel = "IN_APP" | "UPLOAD" | "IMAP" | "TEAMS";
export type Criterion = { id: string; name: string; max_score: string; guidance: string };
export type Counts = { submitted: number; total: number; graded: number; published: number };
export type GradeLink = { component_name: string; component_key: string };

export type Assignment = {
  id: string;
  title: string;
  description: string;
  type: string;
  max_score: string;
  due_at: string;
  allow_late: boolean;
  late_penalty_per_day: string;
  late_max_days: number;
  appeal_days: number;
  channels: Channel[];
  submission_code: string;
  reference_answer: string | null;
  rubric: Criterion[];
  teams_class_id: string | null;
  teams_assignment_id: string | null;
  published_at: string | null;
  version: number;
  created_at: string;
  counts?: Counts;
  grade_link?: GradeLink;
  rubric_locked: boolean;
  late_rule_locked: boolean;
};

export type RubricIn = { id?: string; name: string; max_score: string; guidance: string };
export type AssignmentInput = {
  title: string;
  description: string;
  due_at: string;
  max_score: string;
  allow_late: boolean;
  late_penalty_per_day: string;
  late_max_days: number;
  appeal_days: number;
  channels: Channel[];
  submission_code?: string;
  reference_answer: string | null;
  teams_class_id: string | null;
  teams_assignment_id: string | null;
  rubric?: RubricIn[];
  version?: number;
};

export const CHANNEL_LABEL: Record<Channel, string> = { IN_APP: "Trong EduPilot", UPLOAD: "ZIP", IMAP: "Thư", TEAMS: "Teams" };
export const CHANNELS: Channel[] = ["IN_APP", "UPLOAD", "IMAP", "TEAMS"];

// ---- điểm: cộng bằng đơn vị 1/100 điểm ----------------------------------------------------------------------------------
/** "2,5" / "2.5" → 250 phần trăm điểm; chuỗi sai → null. Bội 0,25 và 0,05 đều là số nguyên ở đơn vị này. */
export function toUnits(s: string): number | null {
  const t = s.trim().replace(",", ".");
  if (!/^\d+(\.\d{1,2})?$/.test(t)) return null;
  const [int, frac = ""] = t.split(".");
  return Number(int) * 100 + Number(`${frac}00`.slice(0, 2));
}
/** 250 → "2.5" (dạng gửi lên máy chủ). */
export const fromUnits = (n: number): string => (n % 100 === 0 ? String(n / 100) : (n / 100).toFixed(2).replace(/0$/, ""));
/** Hiển thị kiểu Việt: "2,5". */
export const vi = (s: string): string => s.replace(".", ",");
/** Tổng điểm tiêu chí; một ô sai → null. */
export function sumUnits(scores: string[]): number | null {
  let total = 0;
  for (const s of scores) {
    const u = toUnits(s);
    if (u === null) return null;
    total += u;
  }
  return total;
}

// ---- đọc ----------------------------------------------------------------------------------------------------------------
export const assignmentsKey = (course: string) => ["assignments", course] as const;
const path = (course: string) => `/courses/${course}/assignments`;

export const useAssignments = (course: string) => useCursorList<Assignment>(assignmentsKey(course), path(course), { limit: 30 });

export const useAssignment = (course: string, id: string | null) =>
  useQuery({
    queryKey: [...assignmentsKey(course), id],
    enabled: Boolean(id),
    queryFn: async ({ signal }) => (await apiClient.get<Assignment>(`${path(course)}/${id}`, { signal })).data,
  });

// ---- ghi -----------------------------------------------------------------------------------------------------------------
export const createAssignment = async (course: string, body: AssignmentInput, idempotencyKey: string) =>
  (await apiClient.post<Assignment>(path(course), body, { idempotencyKey })).data;

export const updateAssignment = async (course: string, id: string, body: AssignmentInput) =>
  (await apiClient.put<Assignment>(`${path(course)}/${id}`, body)).data;

export const putRubric = async (course: string, id: string, version: number, criteria: RubricIn[]) =>
  (await apiClient.put<Assignment>(`${path(course)}/${id}/rubric`, { version, criteria })).data;

export const publishAssignment = async (course: string, id: string, idempotencyKey: string) =>
  (await apiClient.post<{ assignment: Assignment; students: number }>(`${path(course)}/${id}/publish`, {}, { idempotencyKey })).data;

export const deleteAssignment = async (course: string, id: string) => {
  await apiClient.delete(`${path(course)}/${id}`);
};

export const suggestRubric = async (course: string, id: string, idempotencyKey: string) =>
  (await apiClient.post<{ criteria: Array<{ name: string; guidance: string; max_score: string }> }>(`${path(course)}/${id}/rubric/suggest`, {}, { idempotencyKey })).data;
