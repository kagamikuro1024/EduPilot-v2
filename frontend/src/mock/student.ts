// Dữ liệu mô phỏng cho các màn sinh viên (US-PROTO-01): buổi học, bài tập, bài nộp của SV B,
// khuyến nghị "việc nên làm". Mốc ngày/điểm theo SRS 4.1 — không tự đổi số.

import { BT03_SUBMITTED_AT } from "./assess";
import { COURSE_1, COURSE_2, NOW, STUDENT_B, TERM_START, fmtShortDate, type Student } from "./core";

const DAY = 86_400_000;

/** Ngày buổi thứ `n` của lớp 1 (Thứ Năm hằng tuần, buổi 10 = 29/10). */
export function sessionDate(n: number) {
  return new Date(TERM_START.getTime() + (n - 1) * 7 * DAY);
}

/** Buổi 1 lớp 2 (Thứ Ba 13/10); lớp 2 đang ở tuần 3. */
export const TERM_START_2 = new Date("2026-10-13T00:00:00+07:00");

export function sessionDateOf(courseId: string, n: number) {
  return courseId === COURSE_2 ? new Date(TERM_START_2.getTime() + (n - 1) * 7 * DAY) : sessionDate(n);
}

export const SESSION_COUNT = 15;
export const SESSION_HOURS: Record<string, { start: string; end: string }> = {
  [COURSE_1]: { start: "09:00", end: "11:30" },
  [COURSE_2]: { start: "07:00", end: "08:30" },
};

/** Buổi lớp 1 mà SV B vắng: buổi 3 (10/09) và buổi 7 (08/10). */
export const B_ABSENT_SESSIONS = [3, 7];
export const B_ABSENT_DATES = B_ABSENT_SESSIONS.map((n) => fmtShortDate(sessionDate(n)));

/** Khoảng thời gian còn lại tới `d`, tính từ NOW. */
export function until(d: Date) {
  const m = Math.round((d.getTime() - NOW.getTime()) / 60000);
  if (m <= 0) return "đã hết hạn";
  if (m < 60) return `${m} phút`;
  const h = Math.floor(m / 60);
  if (h < 48) return `${h} giờ`;
  return `${Math.round(h / 24)} ngày`;
}

// ---- bài tập / bài kiểm tra của lớp 1 -------------------------------------------------------

export type AssignmentKind = "essay" | "quiz";

export type Assignment = {
  id: string;
  code: string;
  title: string;
  kind: AssignmentKind;
  due: Date;
  /** điểm đã công bố cho cả lớp (BT01, BT02) */
  published: boolean;
  /** phút làm bài (quiz) */
  minutes?: number;
  lateDays?: number;
  description: string;
};

export const ASSIGNMENTS: Assignment[] = [
  {
    id: "bt01",
    code: "Bài tập 01",
    title: "Mô hình đe doạ",
    kind: "essay",
    due: new Date("2026-09-17T23:59:00+07:00"),
    published: true,
    description: "Dựng mô hình đe doạ STRIDE cho một ứng dụng web nội bộ.",
  },
  {
    id: "bt02",
    code: "Bài tập 02",
    title: "Tấn công mạng phổ biến",
    kind: "essay",
    due: new Date("2026-10-01T23:59:00+07:00"),
    published: true,
    description: "So sánh ba kiểu tấn công: SQL injection, XSS và chiếm phiên đăng nhập.",
  },
  {
    id: "bt03",
    code: "Bài tập 03",
    title: "Phân tích một vụ tấn công thực tế",
    kind: "essay",
    due: new Date("2026-10-22T23:59:00+07:00"),
    published: false,
    lateDays: 2,
    description: "Chọn một vụ tấn công đã công bố, phân tích theo 4 tiêu chí của rubric.",
  },
  {
    id: "quiz01",
    code: "QUIZ01",
    title: "Mật mã đối xứng",
    kind: "quiz",
    due: new Date("2026-10-30T03:20:00+07:00"),
    published: false,
    minutes: 20,
    description: "10 câu về AES, chế độ ECB/CBC và quản lý khoá. Bài có tính điểm.",
  },
];

export function assignmentById(id: string) {
  return ASSIGNMENTS.find((a) => a.id === id);
}

/** Bài nộp BT03 của SV B: mốc nộp lấy từ N6 (`BT03_SUBMITTED_AT` = 23/10 08:10), muộn 1 ngày. */
export const BT03_SUBMISSION = {
  studentId: STUDENT_B.id,
  at: BT03_SUBMITTED_AT,
  lateDays: 1,
  file: "bt03-tran-thu-uyen.pdf",
  sizeKb: 418,
};

// ---- kỳ thi ---------------------------------------------------------------------------------

export const MIDTERM = { title: "Thi giữa kỳ", at: new Date("2026-11-05T09:00:00+07:00"), end: "10:30", room: "P.302 – G2", week: 11 };
export const FINAL_EXAM = { title: "Thi cuối kỳ", at: sessionDate(SESSION_COUNT + 1), end: "11:00", room: "Theo lịch của phòng đào tạo", week: 16 };

// ---- "việc nên làm" (luật cứng, không LLM — INTEGRATION mục 4) -------------------------------

export type Recommendation = {
  title: string;
  reason: string;
  minutes: number;
  href: string;
  cta: string;
};

/** Một khuyến nghị duy nhất: hạn gần nhất chưa làm → chủ đề sai nhiều nhất khi luyện đề. */
export function recommendationFor(student: Student, quizDone: boolean): Recommendation {
  const quiz = ASSIGNMENTS[3];
  if (!quizDone && student.id !== "sv-1") {
    return {
      title: `${quiz.code} ${quiz.title} đóng sau ${until(quiz.due)}`,
      reason: "Bạn chưa làm bài này",
      minutes: quiz.minutes ?? 20,
      href: "/practice/at-quiz01",
      cta: "Làm bài",
    };
  }
  return {
    title: "Ôn lại Mật mã đối xứng",
    reason: "Bạn sai 4/7 câu gần nhất",
    minutes: 15,
    href: "/practice/at-symmetric",
    cta: "Luyện 10 câu",
  };
}

/** Học dở: thread và phiên chat gần nhất của sinh viên; nhãn thời gian do màn tính bằng `agoLabel` (N6). */
export const CONTINUE_LEARNING = [
  { title: "CBC khác ECB ở điểm nào?", context: "Threads · bạn đọc dở", at: new Date(NOW.getTime() - 2 * 3600_000), href: "/threads/t-cbc" },
  { title: "Cách chọn độ dài khoá RSA", context: "Chat riêng · phiên trước", at: new Date("2026-10-28T20:10:00+07:00"), href: "/chat" },
];

export const COURSE_IDS = [COURSE_1, COURSE_2];
