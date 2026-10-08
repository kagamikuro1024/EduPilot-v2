// Hợp đồng thật: backend-go/api/openapi.yaml (`/courses/{id}/exams/{eid}/similarity…`, thao tác 52–54 + chi tiết).
export type ReviewState = "NEW" | "CLEARED" | "FOLLOW_UP";
export type Pair = {
  id: string;
  problem_id: string;
  problem_title: string;
  run_id: string;
  a: { attempt_id: string; name: string };
  b: { attempt_id: string; name: string };
  score: string;
  shared_fingerprints: number;
  flagged: boolean;
  review_state: ReviewState;
  note: string | null;
  reviewed_at: string | null;
  created_at: string;
};
export type Side = { language: string; source: string; match_lines: number[] };
export type PairDetail = { pair: Pair; a: Side; b: Side };

export const simPath = (course: string, exam: string) => `/courses/${course}/exams/${exam}/similarity`;

/** "0.873" → "87 %" (làm tròn xuống: không phóng đại độ giống). */
export const percent = (score: string) => `${Math.floor(Number(score) * 100)} %`;

export const STATE_LABEL: Record<ReviewState, string> = { NEW: "Chưa xem", CLEARED: "Đã xem — không có vấn đề", FOLLOW_UP: "Cần trao đổi" };
