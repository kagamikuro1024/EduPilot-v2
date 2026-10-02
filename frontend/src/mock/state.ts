// Hợp đồng trạng thái giả lập DÙNG CHUNG giữa các story (SRS FR-X3).
// Mọi trạng thái nằm trong MỘT khoá localStorage `ep_demo_state` (xem shared/state/demo.ts);
// mỗi "lát" (slice) có tên riêng ở KEYS. Lát dùng chung giữa các màn của nhiều story được khai báo ở đây
// (kiểu + giá trị gốc). Lát riêng của một màn: tự đặt tên khoá có tiền tố route, ví dụ "chat.sessions".
// KHÔNG đổi tên khoá / hình dạng ở đây mà không báo người giữ nền.

import { COURSE_1, COURSE_2, PENDING_STUDENT_IDS } from "./core";

export const KEYS = {
  /** Ticket hộp thư hỗ trợ. Ghi: /chat (D3 sinh ticket mới), /inbox (nhận, trả lời). Đọc: /chat, /, /inbox, chuông. */
  tickets: "tickets",
  /** Điểm danh + phát biểu theo lớp/buổi. Ghi: /attendance. Đọc: /me, /gradebook, /students/[id]. */
  attendance: "attendance",
  /** Thành viên và yêu cầu vào lớp. Ghi: /join (gửi yêu cầu), /class/members (duyệt). Đọc: bộ chọn lớp, /. */
  members: "members",
  /** Kết quả chấm BT03 của SV B. Ghi: /grading/[id], /grading. Đọc: /assignments/bt03, /me, /gradebook. */
  bt03: "bt03",
  /** Trạng thái công thức điểm từng lớp. Ghi: /gradebook/scheme. Đọc: /gradebook, /me. */
  schemes: "schemes",
  /** Thread ghim sinh từ Insights. Ghi: /insights. Đọc: /threads. */
  insightThreads: "insightThreads",
  /** Sự kiện thêm vào lịch (buổi ôn tập). Ghi: /insights. Đọc: /calendar. */
  calendarExtras: "calendarExtras",
  /** Thông báo chuông (mock/notes.ts). Ghi: nơi xảy ra sự kiện liên vai. Đọc: khung chuông ở AppShell. */
  notes: "notes",
  /** Giờ giả lập của sự kiện cuối làm đổi điểm quá trình (SRS 4.8 N7). Ghi: lưu điểm danh, công bố BT03. Đọc: /me. */
  meStamp: "meStamp",
} as const;

// ---- tickets -------------------------------------------------------------------------------

export type TicketStatus = "open" | "claimed" | "answered" | "closed";

export type Ticket = {
  id: string;
  courseId: string;
  studentId: string;
  /** câu hỏi gốc (đã che danh tính phía giảng viên không cần; mock giữ nguyên văn) */
  question: string;
  /** số phút trước "bây giờ" (NOW) khi ticket được tạo; ticket mới tạo = 0 */
  ageMin: number;
  /** ticket tạo trong phiên: mốc giả lập (ms) lúc tạo; có thì `ageMin` bị bỏ qua (xem derive.ticketAgeMin) */
  createdMs?: number;
  status: TicketStatus;
  /** lý do nêu cho giảng viên, ví dụ "Độ tin cậy 0,42 < 0,80" */
  reason: string;
  /** tên người đã nhận + giờ ("09:12") */
  claimedBy?: string;
  claimedAt?: string;
  /** câu trả lời của giảng viên */
  answer?: { by: string; text: string; saveAsKnowledge: boolean };
  /** SV bấm "Đã rõ" → đóng */
  closedBySv?: boolean;
};

// ---- điểm danh -----------------------------------------------------------------------------

export type Mark = "present" | "late" | "excused" | "absent";

export type SessionAttendance = {
  marks: Record<string, Mark>;
  /** số lần phát biểu ghi thêm trong buổi này (+0,25 mỗi lần) */
  speaks: Record<string, number>;
  /** đã bấm `Lưu điểm danh` (hoàn tất buổi) → mới tính vào điểm */
  finalized: boolean;
  /** bản đã chốt lúc bấm `Lưu điểm danh`: điểm của SV đọc bản này; ô sửa sau đó (kể cả lúc giả lập mất mạng) chưa đổi điểm cho tới lần lưu kế */
  committed?: { marks: Record<string, Mark>; speaks: Record<string, number> };
};

/** Bản đã chốt của một buổi (undefined nếu chưa `Lưu điểm danh`). Dữ liệu cũ chưa có `committed` thì lấy chính nó. */
export function committedOf(sa: SessionAttendance | undefined): { marks: Record<string, Mark>; speaks: Record<string, number> } | undefined {
  return sa?.finalized ? (sa.committed ?? sa) : undefined;
}

/** courseId → số buổi (1–15) → dữ liệu điểm danh. Thiếu = chưa ghi (mặc định có mặt). */
export type AttendanceState = Record<string, Record<number, SessionAttendance>>;
export const ATTENDANCE_SEED: AttendanceState = {};

/** Buổi lớp 1 đang diễn ra lúc 09:20 ngày 29/10 (mốc NOW). */
export const CURRENT_SESSION = 10;

// ---- thành viên / yêu cầu vào lớp ---------------------------------------------------------

export type MembersState = {
  /** SV đã được duyệt / vào thêm so với dữ liệu gốc (core.ts), theo lớp */
  joined: Record<string, string[]>;
  /** yêu cầu đang chờ duyệt, theo lớp */
  pending: Record<string, string[]>;
  /** yêu cầu bị từ chối, theo lớp */
  rejected: Record<string, string[]>;
  /** mã tham gia hiện hành của từng lớp (GV có thể tạo lại) */
  joinCodes: Record<string, string>;
};

export const MEMBERS_SEED: MembersState = {
  joined: {},
  pending: { [COURSE_1]: [], [COURSE_2]: PENDING_STUDENT_IDS },
  rejected: {},
  joinCodes: { [COURSE_1]: "AN7K2MQ", [COURSE_2]: "BX4P9TW" },
};

// ---- BT03 của SV B (chấm → duyệt → công bố) ------------------------------------------------

export type Bt03State = {
  status: "draft" | "approved" | "published";
  /** điểm 4 tiêu chí, bước 0,25, tối đa 2,5 mỗi tiêu chí (trước khi trừ nộp muộn) */
  scores: [number, number, number, number];
  comments: [string, string, string, string];
};

// ---- công thức điểm ------------------------------------------------------------------------

export type SchemeStatus = "none" | "draft" | "confirmed";
export type SchemesState = Record<string, { status: SchemeStatus; rounding?: string }>;
export const SCHEMES_SEED: SchemesState = {
  [COURSE_1]: { status: "confirmed" },
  // lớp 2: chưa tải quy chế → tải ở /gradebook/scheme → bản nháp (thiếu làm tròn) → điền → confirmed
  [COURSE_2]: { status: "none" },
};

// ---- do Insights sinh ra -------------------------------------------------------------------

export type InsightThread = { id: string; courseId: string; title: string; topic: string; body: string };
export type CalendarExtra = { id: string; courseId: string; title: string; /** ISO có múi giờ */ at: string; durationMin: number };
