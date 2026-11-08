// Hợp đồng thật: backend-go/api/openapi.yaml (`GET /courses/{id}/me/attendance`, `…/me/participation`, `…/me/study-time`).
import { formatDay } from "@/shared/lib/datetime";
export type MyAttendance = {
  attendance: {
    recorded: number;
    present: number;
    late: number;
    excused: number;
    absent: number;
    sessions: Array<{ session_no: number; starts_at: string; status: "PRESENT" | "LATE" | "EXCUSED" | "ABSENT" }>;
  };
  next_cursor: string | null;
};
export type MyParticipation = {
  participation: { speak_count: number; points_total: string; events: Array<{ date: string; type: string; points: string }> };
  next_cursor: string | null;
};
export type MyStudyTime = {
  weeks: Array<{ week_start: string; minutes: number }>;
  this_week_minutes: number;
  tracking: { enabled: boolean; locked: boolean };
};

export const EVENT_LABEL: Record<string, string> = { SPEAK: "Phát biểu", BONUS: "Điểm cộng", PENALTY: "Trừ điểm" };
export const STATUS_LABEL: Record<string, string> = { PRESENT: "Có mặt", LATE: "Muộn", EXCUSED: "Vắng có phép", ABSENT: "Vắng" };

/** "+0,50" / "−0,25" / "0,00": dấu thập phân kiểu Việt. */
export function signedPoints(raw: string): string {
  const n = Number(raw);
  const body = Math.abs(n).toFixed(2).replace(".", ",");
  return n > 0 ? `+${body}` : n < 0 ? `−${body}` : body;
}

/** "Bạn có mặt 7/9 buổi, vắng 2." — muộn vẫn là có mặt; vắng có phép nêu riêng khi có. */
export function attendanceSentence(a: MyAttendance["attendance"]): string {
  const parts = [`Bạn có mặt ${a.present + a.late}/${a.recorded} buổi`, `vắng ${a.absent}`];
  if (a.excused > 0) parts.push(`vắng có phép ${a.excused}`);
  return `${parts.join(", ")}.`;
}

/** "08/10" theo giờ Việt Nam ("08/10/2025" nếu khác năm). */
export const dayMonthOf = (iso: string) => formatDay(iso);

// `GET /courses/{id}/me/grades` và `POST …/me/grades/what-if` (US-P6-09): DTO sinh viên — cột chưa công bố là PENDING và KHÔNG có `score`.
export type MeGrades = {
  status: "NO_SCHEME" | "IN_PROGRESS" | "FINAL";
  components?: Array<{ key: string; name: string; weight: string; status: "PUBLISHED" | "PENDING"; score?: string }>;
  qt?: string;
  final?: { numeric_grade: string; letter: string; failed_reason?: string; version: number; locked_at: string };
  calculation?: Array<{ label: string; expr: string; value: string; source_quote?: string; page?: number }>;
  official_note?: string;
};
export type MeWhatIf = { hp?: string; letter?: string; failed?: string };

const comma = (v: string) => v.replace(".", ",");

/** Câu nhận định theo trạng thái (không bình luận tốt / xấu, không động viên). */
export function gradeSentence(g: MeGrades): string {
  if (g.status === "NO_SCHEME") return "Lớp chưa có công thức điểm chính thức.";
  if (g.status === "FINAL" && g.final) return `Điểm học phần đã chốt: ${comma(g.final.numeric_grade)} (${g.final.letter}).`;
  if (g.qt) return `Điểm quá trình hiện tại của bạn là ${comma(g.qt)} trên 10.`;
  return "";
}
