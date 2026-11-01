"use client";

import { useQuery } from "@tanstack/react-query";
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

/** Số bản nộp sinh viên đã có cho một bài (US-P7-04 BUG-1): 0 khi chưa nộp lần nào. */
export const studentVersionsKey = (course: string, aid: string, uid: string) => ["assignment-versions", course, aid, uid] as const;
export const useStudentVersions = (course: string, aid: string, uid: string) =>
  useQuery({
    queryKey: studentVersionsKey(course, aid, uid),
    enabled: Boolean(aid && uid),
    staleTime: 10_000,
    retry: false,
    queryFn: async ({ signal }) => (await apiClient.get<{ versions: number }>(`/courses/${course}/assignments/${aid}/versions`, { signal, query: { student_id: uid } })).data,
  });

// ---- thư không xử lý được (US-P7-05 AC9): `GET /courses/{id}/mail-inbound?status=IGNORED` -------------------------------
export type MailInbound = {
  id: string;
  from_email: string;
  subject: string;
  received_at: string;
  status: "ACCEPTED" | "IGNORED" | "FAILED";
  reason: string | null;
  submission_id: string | null;
};

export const MAIL_REASON_TEXT: Record<string, string> = {
  AUTO_REPLY: "Thư tự động",
  NO_ATTACHMENT: "Không có tệp đính kèm",
  TOO_MANY_ATTACHMENTS: "Quá nhiều tệp đính kèm",
  ATTACHMENT_REJECTED: "Tệp bị từ chối",
  PARSE_ERROR: "Không đọc được thư",
};

export const mailInboundKey = (course: string) => ["mail-inbound", course, "IGNORED"] as const;
export const useIgnoredMail = (course: string) =>
  useCursorList<MailInbound>(mailInboundKey(course), `/courses/${course}/mail-inbound`, { limit: 20, query: { status: "IGNORED" } });
