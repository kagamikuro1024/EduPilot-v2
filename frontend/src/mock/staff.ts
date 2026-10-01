// "Hôm nay" của giảng viên / trợ giảng: việc cần quyết định, dải lịch sắp tới, các bước thiết lập lớp mới.
// Nguồn việc theo FLOWS.md F14; việc đã xử lý tự biến mất vì mọi mục đều tính từ trạng thái giả lập.
import { COURSE_1, COURSE_2, courseById, studentById } from "./core";
import { CURRENT_SESSION, type AttendanceState, type Bt03State, type MembersState, type SchemesState, type Ticket } from "./state";

/** "3 ngày 4 giờ", "26 phút" — thời gian chờ nói đủ chính xác để quyết định (khác `ago` ở chỗ không làm tròn). */
export function waitText(minutes: number): string {
  const d = Math.floor(minutes / 1440);
  const h = Math.floor((minutes % 1440) / 60);
  if (d > 0) return h > 0 ? `${d} ngày ${h} giờ` : `${d} ngày`;
  if (h > 0) return `${h} giờ`;
  return `${minutes} phút`;
}

export type TaskStep = { label: string; href: string; done: boolean };

export type StaffTask = {
  id: string;
  tone: "red" | "amber" | "blue" | "neutral";
  title: string;
  context: string;
  /** tên lớp của việc — mỗi việc phải ghi (SRS 4.4) */
  course: string;
  href: string;
  actionLabel: string;
  /** chỉ việc khẩn nhất mới có nút đỏ */
  urgent?: boolean;
  steps?: TaskStep[];
};

/** Quá 72 giờ chưa ai nhận → nổi lên đầu "Hôm nay" (DEMO_SCRIPT bước 3). */
export const OVERDUE_MIN = 72 * 60;

export function staffTasks(input: {
  role: "ta" | "teacher";
  /** lớp người này phụ trách (TA chỉ có 761987) — việc của lớp khác không hiện */
  courseIds: string[];
  tickets: Ticket[];
  attendance: AttendanceState;
  members: MembersState;
  bt03: Bt03State;
  schemes: SchemesState;
}): StaffTask[] {
  const { role, courseIds, tickets, attendance, members, bt03, schemes } = input;
  const out: StaffTask[] = [];
  const c1 = courseById(COURSE_1).label;
  const c2 = courseById(COURSE_2).label;
  const has1 = courseIds.includes(COURSE_1);
  const has2 = courseIds.includes(COURSE_2);

  const open = tickets.filter((t) => t.status === "open" && courseIds.includes(t.courseId)).sort((a, b) => b.ageMin - a.ageMin);
  const oldest = open[0];
  if (oldest) {
    const who = studentById(oldest.studentId)?.name ?? "Sinh viên";
    out.push({
      id: "inbox",
      tone: "red",
      title: `Câu hỏi của ${who} đã chờ ${waitText(oldest.ageMin)}`,
      context: open.length > 1 ? `“${oldest.question}” · còn ${open.length - 1} câu khác đang chờ` : `“${oldest.question}”`,
      course: courseById(oldest.courseId).label,
      href: "/inbox",
      actionLabel: "Trả lời",
      urgent: oldest.ageMin >= OVERDUE_MIN,
    });
  }

  if (has1 && !attendance[COURSE_1]?.[CURRENT_SESSION]?.finalized) {
    out.push({
      id: "attendance",
      tone: "amber",
      title: `Điểm danh buổi ${CURRENT_SESSION} đang diễn ra`,
      context: `Lớp bắt đầu lúc 09:00, 30 sinh viên chưa được chốt điểm danh`,
      course: c1,
      href: "/attendance",
      actionLabel: "Điểm danh",
    });
  }

  if (has1 && bt03.status !== "published") {
    const draft = bt03.status === "draft";
    out.push({
      id: "grading",
      tone: "amber",
      title: draft ? "4 bài Bài tập 03 cần xem kỹ" : "1 bài đã duyệt chưa công bố",
      context: draft ? "Hai lượt chấm lệch hơn 1 điểm ở tiêu chí “Phân tích tấn công”" : "Điểm chỉ đến tay sinh viên sau khi bạn công bố",
      course: c1,
      href: "/grading",
      actionLabel: draft ? "Xem bài" : role === "teacher" ? "Công bố" : "Xem bài",
    });
  }

  if (has1) {
    out.push({
      id: "thread",
      tone: "blue",
      title: "1 câu trả lời của AI chờ bạn xác nhận",
      context: "Thread “CBC khác ECB ở điểm nào?” — sinh viên chỉ thấy nhãn chờ xác nhận",
      course: c1,
      href: "/threads",
      actionLabel: "Xem thread",
    });
  }

  if (role === "teacher" && has2) {
    const steps: TaskStep[] = [
      { label: `Chia sẻ mã tham gia ${members.joinCodes[COURSE_2]}`, href: "/class/members", done: false },
      { label: "Tải quy chế để lập công thức điểm", href: "/gradebook/scheme", done: schemes[COURSE_2]?.status === "confirmed" },
      { label: "Tạo lịch buổi học", href: "/calendar", done: false },
      { label: "Tải tài liệu bài giảng", href: "/documents", done: false },
    ];
    if (steps.some((s) => !s.done)) {
      out.push({
        id: "setup",
        tone: "neutral",
        title: `Thiết lập lớp mới — ${courseById(COURSE_2).code}`,
        context: `Còn ${steps.filter((s) => !s.done).length}/${steps.length} bước trước buổi đầu tiên`,
        course: c2,
        href: "/class/members",
        actionLabel: "Mở lớp",
        steps,
      });
    }
  }

  const pending = has2 ? members.pending[COURSE_2]?.length ?? 0 : 0;
  if (pending > 0) {
    out.push({
      id: "members",
      tone: "neutral",
      title: `${pending} yêu cầu vào lớp chờ duyệt`,
      context: "Lớp bật duyệt trước khi vào, sinh viên đang chờ bạn đồng ý",
      course: c2,
      href: "/class/members",
      actionLabel: "Duyệt",
    });
  }

  return out;
}

// ---- dải lịch sắp tới -------------------------------------------------------------------------

export type UpcomingItem = { id: string; when: string; title: string; note: string; now?: boolean };

/** Vài mốc sắp tới, không phải lịch đầy đủ (lịch đầy đủ ở /calendar). */
export function upcoming(courseId: string): UpcomingItem[] {
  const course = courseById(courseId);
  if (courseId !== COURSE_1) {
    return [{ id: "setup", when: "Chưa có", title: "Lớp chưa có lịch buổi học", note: `${course.schedule} · ${course.room}` }];
  }
  return [
    { id: "s10", when: "Hôm nay 09:00", title: `Buổi ${CURRENT_SESSION} · 09:00–11:30`, note: `${course.room} · đang diễn ra`, now: true },
    { id: "quiz", when: "Mai 03:20", title: "QUIZ01 “Mật mã đối xứng” đóng", note: "Còn 18 giờ · 12 sinh viên chưa làm" },
    { id: "mid", when: "Thứ Năm 5/11", title: `Thi giữa kỳ · buổi ${CURRENT_SESSION + 1}`, note: "Phòng thi công bố trước một tuần" },
  ];
}
