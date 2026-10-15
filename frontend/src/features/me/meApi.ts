// Hợp đồng thật: backend-go/api/openapi.yaml (`GET /courses/{id}/me/attendance`, `…/me/participation`, `…/me/study-time`).
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

const ICT_DAY = new Intl.DateTimeFormat("en-GB", { day: "2-digit", month: "2-digit", timeZone: "Asia/Ho_Chi_Minh" });
/** "08/10" theo giờ Việt Nam, cố định dấu `/` (không phụ thuộc locale của trình duyệt). */
export const dayMonthOf = (iso: string) => ICT_DAY.format(new Date(iso));
