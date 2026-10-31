"use client";

import { apiClient, useCursorList } from "@/shared/data";

// Hợp đồng thật: backend-go/api/openapi.yaml (`/courses/{id}/submissions…`).
// `version` là khoá lạc quan của dòng (int64 của máy chủ). Giá trị thật cỡ 1,8e15 < 2^53 nên `number` của JS giữ đúng;
// gửi lại NGUYÊN VĂN giá trị đã nhận, không tự tính.
export type Unmatched = {
  id: string;
  file_name: string;
  source: string;
  received_at: string;
  reason: string;
  reason_text: string;
  assignment_id: string | null;
  suggestion: { student_id: string; full_name: string } | null;
  version: number;
};

export const SOURCE_LABEL: Record<string, string> = { UPLOAD: "Tải lên", IMAP: "Thư", TEAMS: "Teams", IN_APP: "Trong EduPilot" };

export const unmatchedKey = (course: string) => ["unmatched", course] as const;

export const useUnmatched = (course: string) =>
  useCursorList<Unmatched>(unmatchedKey(course), `/courses/${course}/submissions`, { limit: 30, query: { match_status: "UNMATCHED" } });

const path = (course: string, sid: string) => `/courses/${course}/submissions/${sid}`;

export const matchSubmission = async (course: string, sid: string, body: { assignment_id: string; student_id: string; version: number }) =>
  (await apiClient.put<{ id: string; assignment_id: string; student_id: string; version: number }>(`${path(course, sid)}/match`, body)).data;

export const dismissSubmission = async (course: string, sid: string, reason: string, version: number) => {
  await apiClient.post(`${path(course, sid)}/dismiss`, { reason, version });
};

export const undismissSubmission = async (course: string, sid: string) => {
  await apiClient.delete(`${path(course, sid)}/dismiss`);
};

/** URL ký 5 phút; chỉ lấy khi người dùng bấm (không nhúng sẵn vào DOM). */
export const submissionFileUrl = async (course: string, sid: string) =>
  (await apiClient.get<{ url: string; expires_in: number }>(`${path(course, sid)}/file`)).data.url;
