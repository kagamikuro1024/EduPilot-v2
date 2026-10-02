// MỘT nguồn cho mỗi con số / mốc thời gian của prototype (SRS 4.8, FR-X13).
// Màn chỉ GỌI các hàm này; cấm tự đếm lại hay viết cứng chuỗi như "12 ngày trước", "5", "4".
// Mọi hàm thuần: nhận dữ liệu và "bây giờ" (ms giả lập, xem `shared/state/clock.ts`) làm tham số.
import { bt03Submissions, type ReviewKind, type Submission } from "./assess";
import { COURSE_1, NOW, studentsOf, type Student } from "./core";
import type { Ticket } from "./state";

const MIN = 60000;
const DAY = 86400000;

// ---- N6: mốc thời gian ----------------------------------------------------------------------------------------------------

const pad = (n: number) => String(n).padStart(2, "0");
const hhmm = (d: Date) => `${pad(d.getHours())}:${pad(d.getMinutes())}`;
const dayStart = (d: Date) => new Date(d.getFullYear(), d.getMonth(), d.getDate()).getTime();

/**
 * "vừa xong" · "N phút trước" · "N giờ trước" (cùng ngày) · "hôm qua HH:mm" · "N ngày trước" (làm tròn xuống).
 * `now` mặc định là 09:20 29/10; màn có đồng hồ chạy truyền `useSimNow()`.
 */
export function agoLabel(at: Date | number, now: Date | number = NOW): string {
  const a = new Date(at);
  const n = new Date(now);
  const diff = n.getTime() - a.getTime();
  if (diff < MIN) return "vừa xong";
  const calendarDays = Math.round((dayStart(n) - dayStart(a)) / DAY);
  if (calendarDays <= 0) return diff < 60 * MIN ? `${Math.floor(diff / MIN)} phút trước` : `${Math.floor(diff / (60 * MIN))} giờ trước`;
  if (calendarDays === 1) return `hôm qua ${hhmm(a)}`;
  return `${Math.floor(diff / DAY)} ngày trước`;
}

/** Mốc cố định của kịch bản (SRS 4.8 N6). */
export { BT03_SUBMITTED_AT } from "./assess";
export const ASSIGNED_AT = new Date("2026-10-28T16:40:00+07:00");
export const CH5_UPLOADED_AT = new Date("2026-10-28T14:00:00+07:00");

// ---- N1: phiếu hỗ trợ -------------------------------------------------------------------------------------------------------

/** Tuổi phiếu (phút) tại `nowMs`: phiếu seed tính từ 09:20; phiếu mới tạo trong phiên mang `createdMs`. */
export function ticketAgeMin(t: Ticket, nowMs: number): number {
  return t.createdMs !== undefined ? Math.max(0, Math.floor((nowMs - t.createdMs) / MIN)) : t.ageMin + Math.max(0, Math.floor((nowMs - NOW.getTime()) / MIN));
}

/** Nhãn "Đang chờ giảng viên · …" của SV: "vừa gửi" trong phút đầu, sau đó theo `agoLabel` (N6). */
export function sentLabel(t: Ticket | undefined, nowMs: number): string {
  const age = t ? ticketAgeMin(t, nowMs) : 0;
  return age < 1 ? "vừa gửi" : agoLabel(nowMs - age * MIN, nowMs);
}

export type TicketStats = { open: number; overdue24: number; createdIn7d: number };

export function ticketStats(tickets: Ticket[], courseIds: string[], nowMs: number): TicketStats {
  const mine = tickets.filter((t) => courseIds.includes(t.courseId));
  const open = mine.filter((t) => t.status === "open");
  return {
    open: open.length,
    overdue24: open.filter((t) => ticketAgeMin(t, nowMs) >= 1440).length,
    createdIn7d: mine.filter((t) => ticketAgeMin(t, nowMs) <= 10080).length,
  };
}

// ---- N2: tỉ lệ AI tự trả lời ------------------------------------------------------------------------------------------------

/** `round((Q − E) / Q × 100)`; Q = 0 → 0. */
export function aiShare(questions: number, escalated: number): number {
  return questions > 0 ? Math.round(((questions - escalated) / questions) * 100) : 0;
}

// ---- N3, N4: sinh viên ----------------------------------------------------------------------------------------------------

/** Thứ tự sinh viên ở mọi màn: `sv-n` tăng dần theo n. */
export const studentNo = (id: string) => Number(id.slice(3));
export const byStudentOrder = (a: { id: string }, b: { id: string }) => studentNo(a.id) - studentNo(b.id);
export const orderStudents = <T extends { id: string }>(list: T[]): T[] => [...list].sort(byStudentOrder);

/** Tập "Cần chú ý": `risk ≠ none`; `high` trước, rồi số buổi vắng giảm dần, rồi thứ tự N4. */
export function attentionSet(roster: Student[]): Student[] {
  return roster
    .filter((s) => s.risk !== "none")
    .sort((a, b) => Number(b.risk === "high") - Number(a.risk === "high") || b.absences - a.absences || byStudentOrder(a, b));
}

// ---- N9, N10: badge và việc chờ -----------------------------------------------------------------------------------------------

export type NavBadges = { inbox: number; grading: number };

/** Hộp thư = phiếu `open` (N1); Chấm bài = bài chưa duyệt thuộc "Cần xem kỹ" (N8, đếm do màn chấm truyền vào). */
export function navBadges(stats: TicketStats, reviewPending: number): NavBadges {
  return { inbox: stats.open, grading: reviewPending };
}

// ---- N8: bài "Cần xem kỹ" chưa duyệt ---------------------------------------------------------------------------------------

/** Đã duyệt? Bài của B theo `Bt03State`; các bài khác theo dữ liệu gốc hoặc id GV vừa `Duyệt bài` (`grading.approved`). Nguồn duy nhất cho hàng chờ, badge, thẻ Hôm nay. */
export function isSubmissionApproved(x: Submission, bt03Status: "draft" | "approved" | "published", approvedIds: readonly string[]): boolean {
  return x.studentId === "sv-2" ? bt03Status !== "draft" : x.approved || approvedIds.includes(x.id);
}

/** Loại cờ của các bài `Cần xem kỹ` CHƯA duyệt (bài của B chỉ còn khi GV chưa `Duyệt bài`). Lớp 2 chưa có bài. */
export function reviewPending(bt03Status: "draft" | "approved" | "published", approvedIds: readonly string[] = []): ReviewKind[] {
  return bt03Submissions(studentsOf(COURSE_1))
    .filter((x) => x.flagKind && !isSubmissionApproved(x, bt03Status, approvedIds))
    .map((x) => x.flagKind!);
}

// ---- N7: mốc "Cập nhật" của /me -------------------------------------------------------------------------------------------

/** "Thứ Năm, 29 tháng 10 09:20" từ mốc ms giả lập. */
const WEEKDAY = ["Chủ nhật", "Thứ Hai", "Thứ Ba", "Thứ Tư", "Thứ Năm", "Thứ Sáu", "Thứ Bảy"];
export function fmtStamp(ms: number): string {
  const d = new Date(ms);
  return `${WEEKDAY[d.getDay()]}, ${d.getDate()} tháng ${d.getMonth() + 1} ${hhmm(d)}`;
}
