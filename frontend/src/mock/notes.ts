// Thông báo chuông (SRS 4.9, FR-X17): MỘT kho trong `ep_demo_state` (khoá `KEYS.notes` ở state.ts).
// Nơi xảy ra sự kiện liên vai gọi `pushNote`; khung chuông đọc `notesFor`. Mục "trạng thái" (ví dụ "đang chờ chấm")
// KHÔNG nằm ở đây — chúng được tính từ dữ liệu nên tự biến khi sự kiện xảy ra.
import { simNowMs } from "@/shared/state/clock";
import { writeSlice } from "@/shared/state/demo";
import { courseHref, type Course, type Role } from "./core";

export type NoteTarget = {
  /** vai nhận; bỏ trống = mọi vai (kết hợp với `studentId` / `courseId` để thu hẹp) */
  roles?: Role[];
  /** chỉ sinh viên này nhận */
  studentId?: string;
  /** chỉ người có quyền với lớp này nhận (GV / TA / SV thành viên) */
  courseId?: string;
};

export type Note = {
  id: string;
  to: NoteTarget;
  title: string;
  /** dòng phụ: nguồn (Chat riêng, Bài tập, Lớp học, Threads, Hộp thư hỗ trợ…) */
  meta: string;
  /** đích đã mang đủ lớp + đối tượng, ví dụ `/inbox?ticket=tk-d3`, `/class/members?course=int1006-2&tab=pending` */
  href: string;
  /** mốc giả lập lúc xảy ra (ms) — chuỗi tương đối do derive.agoLabel tính, không ghi sẵn ở đây */
  ms: number;
  /** chữ thay "vừa xong" trong phút đầu, ví dụ "vừa gửi" (SRS 4.9) */
  just?: string;
  /** khoá người xem đã đọc: sinh viên = studentId, còn lại = tên vai */
  readBy: string[];
};

export function pushNote(n: Omit<Note, "readBy" | "ms" | "id"> & { id?: string; ms?: number }) {
  const ms = n.ms ?? simNowMs();
  // thông báo cho một lớp thì đích luôn mang `course=` để mở đúng lớp
  const href = n.to.courseId ? courseHref(n.href, n.to.courseId) : n.href;
  const note: Note = { ...n, href, id: n.id ?? `note-${ms}-${Math.floor(Math.random() * 1e6)}`, ms, readBy: [] };
  writeSlice<Note[]>("notes", (prev) => (prev?.some((x) => x.id === note.id) ? prev : [note, ...(prev ?? [])]));
}

export function markNoteRead(id: string, viewer: string) {
  writeSlice<Note[]>("notes", (prev) => (prev ?? []).map((n) => (n.id === id && !n.readBy.includes(viewer) ? { ...n, readBy: [...n.readBy, viewer] } : n)));
}

/** Khoá người xem: SV = studentId, còn lại = vai. */
export const viewerKey = (role: Role, studentId?: string) => (role === "student" ? (studentId ?? "student") : role);

export function notesFor(all: Note[], role: Role, studentId: string | undefined, courseIds: string[]): Note[] {
  return all
    .filter((n) => (!n.to.roles || n.to.roles.includes(role)) && (!n.to.studentId || n.to.studentId === studentId) && (!n.to.courseId || courseIds.includes(n.to.courseId)))
    .sort((a, b) => b.ms - a.ms);
}

// ---- Sự kiện liên vai (SRS 4.9): câu chữ nằm ở ĐÂY, nơi xảy ra chỉ gọi một dòng ----------------------------------------

/** SV hỏi mà AI không trả lời được → phiếu mới cho giảng viên và trợ giảng của lớp. */
export function noteNewTicket(courseId: string, ticketId: string) {
  pushNote({
    id: `n-${ticketId}`,
    to: { roles: ["teacher", "ta"], courseId },
    title: "1 câu hỏi mới cần xử lý",
    meta: "Hộp thư hỗ trợ",
    just: "vừa gửi",
    href: `/inbox?ticket=${ticketId}`,
  });
}

/** Câu hỏi của SV được chuyển cho giảng viên (D3) → mục trên chuông của chính SV đó. */
export function noteTicketSent(studentId: string, ticketId: string) {
  pushNote({ id: `n-${ticketId}-sent`, to: { studentId }, title: "Câu hỏi của bạn đang chờ giảng viên", meta: "Chat riêng", just: "vừa gửi", href: "/chat" });
}

/** Giảng viên gửi trả lời phiếu → sinh viên đã hỏi. */
export function noteTicketAnswered(studentId: string, ticketId: string) {
  pushNote({ id: `n-${ticketId}-answered`, to: { studentId }, title: "Giảng viên đã trả lời câu hỏi của bạn", meta: "Chat riêng", href: "/chat" });
}

/** `Công bố` điểm Bài tập 03 → mỗi sinh viên có bài trong đợt. */
export function noteBt03Published(studentId: string) {
  pushNote({ id: `n-bt03-published-${studentId}`, to: { studentId }, title: "Điểm Bài tập 03 đã được công bố", meta: "Bài tập", href: "/assignments/bt03" });
}

/** `Duyệt` / `Từ chối` yêu cầu vào lớp → sinh viên được quyết định. */
export function noteJoinDecided(studentId: string, course: Course, approved: boolean) {
  pushNote(
    approved
      ? { to: { studentId }, title: `Bạn đã được duyệt vào lớp ${course.label}`, meta: "Lớp học", href: "/" }
      : { to: { studentId }, title: `Yêu cầu vào lớp ${course.code} chưa được chấp nhận`, meta: "Lớp học", href: "/join" },
  );
}

/** SV gửi yêu cầu vào lớp cần duyệt → giảng viên (và trợ giảng) lớp đó. */
export function noteJoinRequest(studentName: string, course: Course) {
  pushNote({
    to: { roles: ["teacher", "ta"], courseId: course.id },
    title: `${studentName} xin vào lớp ${course.code}`,
    meta: "Lớp học",
    just: "vừa gửi",
    href: `/class/members?course=${course.id}&tab=pending`,
  });
}
