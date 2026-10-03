// Lịch (US-PROTO-01, /calendar — dùng cho cả sinh viên lẫn giảng viên / trợ giảng).
// Sự kiện dựng từ buổi học của lớp + hạn nộp + kỳ thi; buổi ôn tập do /insights tạo thêm.

import { COURSE_1, NOW, courseById } from "./core";
import type { CalendarExtra } from "./state";
import { ASSIGNMENTS, FINAL_EXAM, MIDTERM, SESSION_COUNT, SESSION_HOURS, sessionDateOf, until } from "./student";

export type EventKind = "session" | "deadline" | "exam" | "extra";

export const EVENT_LABEL: Record<EventKind, string> = {
  session: "Buổi học",
  deadline: "Hạn nộp",
  exam: "Kỳ thi",
  extra: "Buổi ôn tập",
};

/**
 * Nhãn ngắn cho ô lịch Tháng (ô hẹp ≈ 48–100 px, tối đa 2 dòng): phần trước dấu " · " và bỏ tiền tố "Hạn nộp " (loại sự kiện đã có ở
 * màu viền) — "QUIZ01 đóng · Mật mã đối xứng" → "QUIZ01 đóng", "Hạn nộp Bài tập 02 · …" → "Bài tập 02". Tên đầy đủ ở title/aria-label.
 */
export function shortLabel(title: string): string {
  return title.split(" · ")[0].replace(/^Hạn nộp /, "");
}

export type CalEvent = {
  id: string;
  kind: EventKind;
  title: string;
  start: Date;
  /** giờ kết thúc dạng "11:30" */
  endLabel?: string;
  where?: string;
  href?: string;
  /** câu chỉ sinh viên thấy ("bạn chưa làm") */
  studentNote?: string;
};

function hhmm(d: Date, time: string) {
  const [h, m] = time.split(":").map(Number);
  const out = new Date(d);
  out.setHours(h, m, 0, 0);
  return out;
}

/** Mọi sự kiện của một lớp, xếp theo thời gian. */
export function eventsFor(courseId: string, extras: CalendarExtra[], quizDone = false): CalEvent[] {
  const course = courseById(courseId);
  const hours = SESSION_HOURS[courseId] ?? SESSION_HOURS[COURSE_1];
  const out: CalEvent[] = [];

  for (let n = 1; n <= SESSION_COUNT; n++) {
    const day = sessionDateOf(courseId, n);
    out.push({
      id: `ss-${courseId}-${n}`,
      kind: "session",
      title: `Buổi ${n} · ${course.name}`,
      start: hhmm(day, hours.start),
      endLabel: hours.end,
      where: course.room,
    });
  }

  if (courseId === COURSE_1) {
    for (const a of ASSIGNMENTS) {
      out.push({
        id: `dl-${a.id}`,
        kind: "deadline",
        title: a.kind === "quiz" ? `${a.code} đóng · ${a.title}` : `Hạn nộp ${a.code} · ${a.title}`,
        start: a.due,
        href: a.kind === "quiz" ? "/practice/at-quiz01" : `/assignments/${a.id}`,
        studentNote: a.id === "quiz01" ? (quizDone ? "Bạn đã nộp" : `Bạn chưa làm · còn ${until(a.due)}`) : undefined,
      });
    }
    out.push({ id: "ex-mid", kind: "exam", title: `${MIDTERM.title} · tuần ${MIDTERM.week}`, start: MIDTERM.at, endLabel: MIDTERM.end, where: MIDTERM.room });
    out.push({ id: "ex-final", kind: "exam", title: `${FINAL_EXAM.title} · tuần ${FINAL_EXAM.week}`, start: FINAL_EXAM.at, endLabel: FINAL_EXAM.end, where: FINAL_EXAM.room });
  }

  for (const e of extras.filter((x) => x.courseId === courseId)) {
    const start = new Date(e.at);
    const end = new Date(start.getTime() + e.durationMin * 60000);
    out.push({
      id: e.id,
      kind: "extra",
      title: e.title,
      start,
      endLabel: `${end.getHours().toString().padStart(2, "0")}:${end.getMinutes().toString().padStart(2, "0")}`,
      where: course.room,
    });
  }

  return out.sort((a, b) => a.start.getTime() - b.start.getTime());
}

/** Thứ Hai của tuần chứa `d` (tuần bắt đầu từ Thứ Hai). */
export function startOfWeek(d: Date) {
  const out = new Date(d);
  out.setHours(0, 0, 0, 0);
  out.setDate(out.getDate() - ((out.getDay() + 6) % 7));
  return out;
}

export const WEEK_DAY_SHORT = ["T2", "T3", "T4", "T5", "T6", "T7", "CN"];

/** Buổi đang diễn ra lúc NOW (buổi 10 lớp 1). */
export function isNow(e: CalEvent) {
  if (e.kind !== "session" && e.kind !== "extra") return false;
  const end = e.endLabel ? hhmm(e.start, e.endLabel) : new Date(e.start.getTime() + 90 * 60000);
  return e.start <= NOW && NOW <= end;
}
