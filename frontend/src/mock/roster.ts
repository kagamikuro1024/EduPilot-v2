// Danh sách lớp, lịch buổi học và hồ sơ 360 — US-PROTO-02 (/students, /attendance, /class/members).
import { COURSE_1, RECORDED_SESSIONS, TERM_START, fmtShortDate, studentById, studentsOf, type Student } from "./core";
import { orderStudents } from "./derive";
import { baseBtScores, type AttendanceStats } from "./grades";
import { CURRENT_SESSION, type MembersState } from "./state";

/** Thành viên hiện tại của một lớp: dữ liệu gốc + người vừa được duyệt ở /class/members. */
export function rosterOf(courseId: string, members: MembersState): Student[] {
  const extra = (members.joined[courseId] ?? []).map(studentById).filter((s): s is Student => s !== undefined && !s.courseIds.includes(courseId));
  // Thứ tự chuẩn của mọi màn: `sv-n` tăng dần (SRS 4.8 N4) — không sắp theo tên.
  return orderStudents([...studentsOf(courseId), ...extra]);
}

// ---- lịch buổi học ---------------------------------------------------------------------------

export const SESSION_COUNT = 15;

export type SessionState = "recorded" | "current" | "future";

export type ClassSession = { n: number; date: Date; label: string; state: SessionState };

/** Danh sách buổi của lớp đang chọn; lớp chưa có lịch (761988) trả về rỗng. */
export function sessionsOf(courseId: string): ClassSession[] {
  if (courseId !== COURSE_1) return [];
  return Array.from({ length: SESSION_COUNT }, (_, i) => {
    const n = i + 1;
    // buổi n = 27/08 + 7·(n−1) ngày (SRS 4.1)
    const date = new Date(TERM_START.getTime() + (n - 1) * 7 * 86400000);
    const state: SessionState = n < CURRENT_SESSION ? "recorded" : n === CURRENT_SESSION ? "current" : "future";
    return { n, date, label: `Buổi ${n} · ${fmtShortDate(date)}`, state };
  });
}

// ---- chuyên cần đã ghi (buổi 1–9) ------------------------------------------------------------

/** SV B vắng buổi 3 (10/09) và buổi 7 (08/10) — SRS 4.1. */
const ABSENT_FIXED: Record<string, number[]> = { "sv-2": [3, 7] };

/** Các buổi đã ghi mà sinh viên vắng; tất định để mọi màn hiện cùng một sự thật. */
export function absentSessions(s: Student): number[] {
  const fixed = ABSENT_FIXED[s.id];
  if (fixed) return fixed;
  const seed = Number(s.id.slice(3));
  const out: number[] = [];
  for (let k = 0; k < RECORDED_SESSIONS * 3 && out.length < s.absences; k++) {
    const n = ((seed * 3 + k * 4) % RECORDED_SESSIONS) + 1;
    if (!out.includes(n)) out.push(n);
  }
  return out.sort((a, b) => a - b);
}

// ---- hoạt động học ---------------------------------------------------------------------------

/** Phút học mỗi tuần, 8 tuần gần nhất, trung bình quanh `activityMin` (chỉ tham khảo — D17). */
export function activitySeries(s: Student): Array<{ x: string; y: number }> {
  const seed = Number(s.id.slice(3));
  return Array.from({ length: 8 }, (_, i) => {
    const week = 3 + i;
    const wave = ((seed * 7 + i * 13) % 11) / 10 - 0.5;
    const drift = s.gradeTrend < 0 ? 1 - i * 0.07 : 0.8 + i * 0.05;
    return { x: `T${week}`, y: Math.max(5, Math.round(s.activityMin * drift + s.activityMin * wave * 0.3)) };
  });
}

// ---- ghi chú quan sát (chỉ giảng viên/TA) -----------------------------------------------------

export type StudentNote = { id: string; at: string; by: string; text: string };

/** Khoá lát riêng của /students/[id]. */
export const NOTES_KEY = "students.notes";

export const NOTES_SEED: Record<string, StudentNote[]> = {
  "sv-3": [
    { id: "n-1", at: "15/10", by: "Phạm Quốc Bảo", text: "Đã nhắn hỏi lý do vắng buổi 6, em trả lời bị trùng lịch làm thêm buổi sáng." },
    { id: "n-2", at: "22/10", by: "Lê Thu Hà", text: "Hẹn gặp cuối buổi 9 để bàn kế hoạch nộp bù Bài tập 02, em chưa tới." },
  ],
};

// ---- câu rủi ro bằng tiếng Việt thường ---------------------------------------------------------

const fmt1 = (n: number) => n.toFixed(1).replace(".", ",");

/**
 * Một câu nói rõ vì sao sinh viên cần chú ý — không dùng nhãn kỹ thuật (DESIGN §14.7).
 * Suy ra từ chuyên cần thật + bài còn thiếu, nên khớp mọi màn khác.
 */
export function riskSentence(s: Student, stats: AttendanceStats): string | null {
  const parts: string[] = [];
  const penalty = Math.max(0, stats.absences - 2) * 0.5;
  if (stats.absences >= 3) {
    parts.push(penalty > 0 ? `vắng ${stats.absences}/${stats.recorded} buổi, đã bị trừ ${fmt1(penalty)} điểm` : `vắng ${stats.absences}/${stats.recorded} buổi`);
  }
  const bt = baseBtScores(s);
  if (bt.bt01 === null) parts.push("thiếu Bài tập 01");
  if (bt.bt02 === null) parts.push("thiếu Bài tập 02");
  if (s.gradeTrend <= -1.2) parts.push(`điểm giảm ${fmt1(-s.gradeTrend)} so với hai tuần trước`);
  if (s.activityMin < 45) parts.push("học dưới ngưỡng trong ba tuần liên tiếp");
  if (parts.length === 0) return null;
  // hai lý do nặng nhất là đủ để quyết định; dài hơn thì không ai đọc hết
  return `${parts.slice(0, 2).join("; ").replace(/^./, (c) => c.toUpperCase())}.`;
}

/** Nhóm chip lọc ở /students (DESIGN §14.6). */
export const STUDENT_FILTERS = [
  { value: "watch", label: "Cần chú ý" },
  { value: "absent", label: "Vắng nhiều" },
  { value: "drop", label: "Điểm giảm" },
  { value: "quiet", label: "Ít hoạt động" },
] as const;

export type StudentFilter = (typeof STUDENT_FILTERS)[number]["value"];

export function matchesFilter(f: StudentFilter, s: Student, stats: AttendanceStats): boolean {
  if (f === "watch") return s.risk !== "none";
  if (f === "absent") return stats.absences >= 3;
  if (f === "drop") return s.gradeTrend <= -1.2;
  return s.activityMin < 45;
}

