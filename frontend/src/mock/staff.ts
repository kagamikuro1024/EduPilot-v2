// "Hôm nay" của giảng viên / trợ giảng: việc cần quyết định, dải lịch sắp tới, các bước thiết lập lớp mới.
// Thứ tự và cách đếm việc theo SRS 4.3.1 J3; mọi con số lấy từ `mock/derive.ts` (FR-X13), không đếm lại ở màn.
import { REVIEW_KIND_SHORT, type ReviewKind } from "./assess";
import { COURSE_1, COURSE_2, courseById, studentById } from "./core";
import { agoLabel, reviewPending, ticketAgeMin } from "./derive";
import type { PendingAi, ThreadTask } from "./threads";
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
  /** móc đo `data-task-id` (J3): `inbox` · `attendance` · `review` · `thread-new-<n>` · `ai-pending` · `setup` · `members` */
  id: string;
  tone: "red" | "amber" | "blue" | "neutral";
  title: string;
  context: string;
  /** nhãn trạng thái của việc (`Chờ xác nhận`, `Cần giảng viên trả lời`) */
  tag?: string;
  /** tên lớp của việc — mỗi việc phải ghi (SRS 4.4) */
  course: string;
  href: string;
  actionLabel: string;
  /** chỉ việc khẩn nhất mới có nút đỏ */
  urgent?: boolean;
  steps?: TaskStep[];
};

/** "1 lệch hai lượt chấm · 1 bài ngắn bất thường · …" — đếm theo loại cờ, giữ thứ tự gặp (SRS 4.8 N8). */
function reviewLine(kinds: ReviewKind[]): string {
  const counts = new Map<ReviewKind, number>();
  for (const k of kinds) counts.set(k, (counts.get(k) ?? 0) + 1);
  return [...counts].map(([k, n]) => `${n} ${REVIEW_KIND_SHORT[k]}`).join(" · ");
}

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
  /** việc "Câu hỏi mới" của Threads, mới nhất trước (`threadTasks`) */
  threads: ThreadTask[];
  /** câu AI chờ xác nhận còn lại sau khi trừ các thread đã có việc riêng (`pendingAi`, N10) */
  pendingAi: PendingAi[];
  /** giờ giả lập (ms) — tuổi phiếu và "vừa xong" lớn dần theo đồng hồ */
  nowMs: number;
}): StaffTask[] {
  const { role, courseIds, tickets, attendance, members, bt03, schemes, threads, pendingAi, nowMs } = input;
  const out: StaffTask[] = [];
  const c1 = courseById(COURSE_1).label;
  const c2 = courseById(COURSE_2).label;
  const has1 = courseIds.includes(COURSE_1);
  const has2 = courseIds.includes(COURSE_2);

  const open = tickets
    .filter((t) => t.status === "open" && courseIds.includes(t.courseId))
    .sort((a, b) => ticketAgeMin(b, nowMs) - ticketAgeMin(a, nowMs));
  const oldest = open[0];
  if (oldest) {
    const who = studentById(oldest.studentId)?.name ?? "Sinh viên";
    const age = ticketAgeMin(oldest, nowMs);
    out.push({
      id: "inbox",
      tone: "red",
      title: `Câu hỏi của ${who} đã chờ ${waitText(age)}`,
      context: open.length > 1 ? `“${oldest.question}” · còn ${open.length - 1} câu khác đang chờ` : `“${oldest.question}”`,
      course: courseById(oldest.courseId).label,
      href: `/inbox?ticket=${oldest.id}`,
      actionLabel: "Trả lời",
      urgent: age >= OVERDUE_MIN,
    });
  }

  if (has1 && !attendance[COURSE_1]?.[CURRENT_SESSION]?.finalized) {
    out.push({
      id: "attendance",
      tone: "amber",
      title: `Điểm danh buổi ${CURRENT_SESSION} đang diễn ra`,
      context: `Lớp bắt đầu lúc 09:00, 30 sinh viên chưa được chốt điểm danh`,
      course: c1,
      href: `/attendance?session=${CURRENT_SESSION}`,
      actionLabel: "Điểm danh",
    });
  }

  const kinds = has1 ? reviewPending(bt03.status) : [];
  if (kinds.length > 0) {
    out.push({
      id: "review",
      tone: "amber",
      title: `${kinds.length} bài Bài tập 03 cần xem kỹ`,
      context: reviewLine(kinds),
      course: c1,
      href: "/grading?filter=review",
      actionLabel: "Xem bài",
    });
  }

  for (const t of threads) {
    out.push({
      id: `thread-new-${t.no}`,
      tone: t.label === "Cần giảng viên trả lời" ? "amber" : "blue",
      title: `Câu hỏi mới: «${t.title}»`,
      context: `${t.topic} · ${agoLabel(t.ms, nowMs)}`,
      tag: t.label,
      course: courseById(t.courseId).label,
      href: `/threads/${t.threadId}`,
      actionLabel: "Xem thread",
    });
  }

  if (pendingAi.length > 0) {
    out.push({
      id: "ai-pending",
      tone: "blue",
      title: `${pendingAi.length} câu trả lời của AI chờ bạn xác nhận`,
      context: "Sinh viên chỉ thấy nhãn chờ xác nhận cho tới khi bạn duyệt câu trả lời",
      course: has1 ? c1 : c2,
      href: pendingAi.length === 1 ? `/threads/${pendingAi[0].threadId}` : "/threads?filter=pending",
      actionLabel: "Xem thread",
    });
  }

  if (role === "teacher" && has2) {
    const to = (path: string) => `${path}?course=${COURSE_2}`;
    const steps: TaskStep[] = [
      { label: `Chia sẻ mã tham gia ${members.joinCodes[COURSE_2]}`, href: to("/class/members"), done: false },
      { label: "Tải quy chế để lập công thức điểm", href: to("/gradebook/scheme"), done: schemes[COURSE_2]?.status === "confirmed" },
      { label: "Tạo lịch buổi học", href: to("/calendar"), done: false },
      { label: "Tải tài liệu bài giảng", href: to("/documents"), done: false },
    ];
    if (steps.some((s) => !s.done)) {
      out.push({
        id: "setup",
        tone: "neutral",
        title: `Thiết lập lớp mới — ${courseById(COURSE_2).code}`,
        context: `Còn ${steps.filter((s) => !s.done).length}/${steps.length} bước trước buổi đầu tiên`,
        course: c2,
        href: to("/class/members"),
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
      href: `/class/members?course=${COURSE_2}&tab=pending`,
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
