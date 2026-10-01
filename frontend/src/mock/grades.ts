// Tính điểm của bản mô phỏng — số nguyên phần trăm (hai chữ số thập phân), làm tròn nửa lên ở bước cuối
// (SRS 4.1). Đây là bản mô phỏng của `internal/grade` (code thuần, không LLM, không float64 cho điểm thật).

import { COURSE_1, RECORDED_SESSIONS, STUDENT_B, studentById, type Student } from "./core";
import type { AttendanceState, Bt03State } from "./state";
import { CURRENT_SESSION } from "./state";

export type Scheme = {
  /** trọng số quá trình, % (CK = 100 − qt) */
  qtWeight: number;
  /** cộng mỗi lần phát biểu (đơn vị 0,01) và trần */
  speakBonus: number;
  speakCap: number;
  /** trừ mỗi buổi vắng không phép từ buổi vắng thứ `absenceFrom` */
  absencePenalty: number;
  absenceFrom: number;
};

export const SCHEME_1: Scheme = { qtWeight: 40, speakBonus: 0.25, speakCap: 1.0, absencePenalty: 0.5, absenceFrom: 3 };
/** Bản nháp trích từ quy-che-lop2.pdf — còn thiếu quy tắc làm tròn (chặn xác nhận). */
export const SCHEME_2_DRAFT: Scheme = { qtWeight: 30, speakBonus: 0.2, speakCap: 0.6, absencePenalty: 0.5, absenceFrom: 3 };

const cents = (n: number) => Math.round(n * 100);
const fromCents = (c: number) => c / 100;
/** Làm tròn nửa lên đến 0,1 trên số nguyên phần trăm. */
const round1 = (c: number) => Math.floor((c + 5) / 10) / 10;

export type Qt = {
  /** trung bình bài tập đã công bố (2 chữ số) hoặc null nếu chưa có bài nào */
  avg: number | null;
  bonus: number;
  penalty: number;
  /** điểm quá trình đã làm tròn 0,1 */
  qt: number | null;
};

/** `scores`: điểm các bài tập ĐÃ CÔNG BỐ (bỏ qua null). Bài thiếu không tính vào trung bình (thiếu bài hiển thị riêng). */
export function calcQt(input: { scores: Array<number | null>; speaks: number; absences: number }, s: Scheme = SCHEME_1): Qt {
  const got = input.scores.filter((x): x is number => x !== null).map(cents);
  if (got.length === 0) return { avg: null, bonus: 0, penalty: 0, qt: null };
  // trung bình hai chữ số, nửa lên
  const avgC = Math.floor((got.reduce((a, b) => a + b, 0) * 10 / got.length + 5) / 10);
  const bonusC = Math.min(cents(s.speakCap), input.speaks * cents(s.speakBonus));
  const penaltyC = Math.max(0, input.absences - (s.absenceFrom - 1)) * cents(s.absencePenalty);
  const totalC = avgC + bonusC - penaltyC;
  return { avg: fromCents(avgC), bonus: fromCents(bonusC), penalty: fromCents(penaltyC), qt: round1(totalC) };
}

/** Điểm học phần dự kiến khi biết CK: qtWeight·QT + (100−qtWeight)·CK, làm tròn 0,1. */
export function calcFinal(qt: number, ck: number, s: Scheme = SCHEME_1): number {
  const c = (cents(qt) * s.qtWeight + cents(ck) * (100 - s.qtWeight)) / 100;
  return round1(Math.round(c));
}

// ---- BT03 ----------------------------------------------------------------------------------

export const LATE_PENALTY_BT03 = 0.5; // nộp muộn 1 ngày

/** Tổng BT03 sau khi trừ nộp muộn (không âm). */
export function bt03Total(b: Pick<Bt03State, "scores">): { raw: number; late: number; total: number } {
  const raw = fromCents(b.scores.reduce((a, x) => a + cents(x), 0));
  return { raw, late: LATE_PENALTY_BT03, total: Math.max(0, fromCents(cents(raw) - cents(LATE_PENALTY_BT03))) };
}

// ---- điểm bài tập gốc theo SV (tất định) ----------------------------------------------------

export type BtScores = { bt01: number | null; bt02: number | null; bt03: number | null };

/** Điểm BT01, BT02 đã công bố. BT03 luôn null ở đây (chỉ B có, và chỉ sau khi GV công bố — xem `btScoresFor`). */
export function baseBtScores(s: Student): BtScores {
  if (s.id === "sv-1") return { bt01: 8.5, bt02: 8.0, bt03: null };
  if (s.id === "sv-2") return { bt01: 7.0, bt02: 8.0, bt03: null };
  if (s.id === "sv-3") return { bt01: 5.5, bt02: null, bt03: null }; // thiếu BT02
  if (s.id === "sv-4") return { bt01: null, bt02: null, bt03: null };
  const n = Number(s.id.slice(3));
  const f = (k: number) => Math.round((5.5 + ((n * k) % 41) / 10) * 4) / 4; // 5,5 … 9,5 bước 0,25
  return { bt01: Math.min(10, f(7)), bt02: Math.min(10, f(11)), bt03: null };
}

export function btScoresFor(s: Student, bt03: Bt03State): BtScores {
  const base = baseBtScores(s);
  return s.id === STUDENT_B.id && bt03.status === "published" ? { ...base, bt03: bt03Total(bt03).total } : base;
}

// ---- thống kê chuyên cần / phát biểu có tính hệ quả của điểm danh buổi 10 ---------------------

export type AttendanceStats = { recorded: number; absences: number; speaks: number; late: number };

/** Chuyên cần + phát biểu của một SV lớp 1: dữ liệu gốc + buổi hôm nay nếu GV đã `Lưu điểm danh`. */
export function attendanceStats(s: Student, attendance: AttendanceState): AttendanceStats {
  const today = attendance[COURSE_1]?.[CURRENT_SESSION];
  let absences = s.absences;
  let speaks = s.speaks;
  let late = 0;
  let recorded = RECORDED_SESSIONS;
  if (today?.finalized) {
    recorded += 1;
    const m = today.marks[s.id] ?? "present";
    if (m === "absent") absences += 1;
    if (m === "late") late += 1;
    speaks += today.speaks[s.id] ?? 0;
  }
  return { recorded, absences, speaks, late };
}

/** QT (lớp 1, công thức đã xác nhận) của một SV theo trạng thái hiện tại. */
export function qtOf(studentId: string, attendance: AttendanceState, bt03: Bt03State): Qt & AttendanceStats & BtScores {
  const s = studentById(studentId) ?? STUDENT_B;
  const st = attendanceStats(s, attendance);
  const bt = btScoresFor(s, bt03);
  const q = calcQt({ scores: [bt.bt01, bt.bt02, bt.bt03], speaks: st.speaks, absences: st.absences });
  return { ...q, ...st, ...bt };
}
