// Hợp đồng thật: backend-go/api/openapi.yaml (`GET /courses/{id}/students`).
export type StudentRow = {
  student: { id: string; full_name: string; student_code: string | null };
  attendance: { recorded: number; present: number; late: number; excused: number; absent: number };
  participation: { speak_count: number; points_total: string };
  study: { minutes_7d: number; tracked: boolean };
  activity: { asks_last_week: number; posts_last_week: number };
  risk: { level: "NONE" | "WATCH" | "AT_RISK"; computed_at: string | null };
};
export type StudentFilter = "attention" | "absent" | "inactive";
export const FILTERS: Array<{ value: StudentFilter; label: string }> = [
  { value: "attention", label: "Cần chú ý" },
  { value: "absent", label: "Vắng nhiều" },
  { value: "inactive", label: "Ít hoạt động" },
];
export const isFilter = (v: string): v is StudentFilter => FILTERS.some((f) => f.value === v);

/** "+0,50" / "−0,25" / "0,00": điểm cộng phát biểu, dấu thập phân kiểu Việt. */
export function signedPoints(raw: string): string {
  const n = Number(raw);
  const body = Math.abs(n).toFixed(2).replace(".", ",");
  return n > 0 ? `+${body}` : n < 0 ? `−${body}` : body;
}

// Hồ sơ 360 (US-P5-05): `GET /courses/{id}/students/{uid}/profile` (MỖI lần gọi = một dòng audit nên không tự gọi lại: xem RealProfile).
export type Profile = {
  student: { id: string; full_name: string; student_code: string | null; joined_at: string };
  risk: { level: "NONE" | "WATCH" | "AT_RISK"; sentence: string | null; reasons: Array<{ code: string; text: string }>; computed_at: string | null };
  attendance: { recorded: number; present: number; late: number; excused: number; absent: number };
  participation: { speak_count: number; points_total: string };
  study: { minutes_7d: number; weeks: Array<{ week_start: string; minutes: number }>; tracked: boolean };
  activity: { asks_last_week: number; posts_last_week: number };
};
export type AttendanceRow = { session: { id: string; session_no: number; starts_at: string }; status: "PRESENT" | "LATE" | "EXCUSED" | "ABSENT"; note: string | null };
export type Observation = {
  id: string;
  author: { id: string; full_name: string };
  kind: "NOTE" | "AI_SUMMARY";
  tags: string[];
  body: string;
  ai_edited: boolean;
  version: number;
  created_at: string;
  updated_at: string;
};
export type AiDraft = { summary: string; strengths: string[]; concerns: string[]; suggestions: string[] };

/** Bản nháp AI thành văn bản sửa được: một đoạn tóm tắt rồi ba nhóm gạch đầu dòng. */
export function draftText(d: AiDraft): string {
  const part = (title: string, xs: string[]) => (xs.length ? `\n\n${title}:\n${xs.map((x) => `- ${x}`).join("\n")}` : "");
  return d.summary + part("Điểm mạnh", d.strengths) + part("Cần lưu ý", d.concerns) + part("Gợi ý", d.suggestions);
}

export const STATUS_LABEL: Record<AttendanceRow["status"], string> = { PRESENT: "Có mặt", LATE: "Muộn", EXCUSED: "Vắng phép", ABSENT: "Vắng" };
