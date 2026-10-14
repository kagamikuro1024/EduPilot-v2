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
