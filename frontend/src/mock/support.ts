// Hộp thư hỗ trợ (ticket). Người giữ: US-PROTO-02. Chỉ dùng chung `Ticket`, `TICKETS_SEED`, `ticketD3`.
import { COURSE_1, STUDENT_B } from "./core";
import type { Ticket } from "./state";

/** Câu D3 nguyên văn (DEMO_SCRIPT mục 3). */
export const D3_TEXT = "Thi cuối kỳ có được mang một tờ A4 ghi chú viết tay vào phòng thi không ạ?";

/** Ticket sinh ra khi SV B gửi D3 ở /chat (AI dưới ngưỡng → tự chuyển giảng viên, Q1). */
export function ticketD3(createdMs?: number): Ticket {
  return { id: "tk-d3", courseId: COURSE_1, studentId: STUDENT_B.id, question: D3_TEXT, ageMin: 0, createdMs, status: "open", reason: "Độ tin cậy 0,42 < 0,80 — không tài liệu nào trả lời được" };
}

/** Ticket gốc (SRS 4.1): 5 mở — tuổi 26 phút, 3 giờ, 1 ngày, 2 ngày, 3 ngày 4 giờ — + 1 đã có người nhận. */
export const TICKETS_SEED: Ticket[] = [
  {
    id: "tk-1",
    courseId: COURSE_1,
    studentId: "sv-1",
    question: "Trong bài 4, vì sao chế độ CBC bắt buộc có IV ngẫu nhiên mà ECB thì không ạ?",
    ageMin: 26,
    status: "open",
    reason: "Độ tin cậy 0,61 < 0,80",
  },
  {
    id: "tk-2",
    courseId: COURSE_1,
    studentId: "sv-7",
    question: "Bài tập 03 em dùng SHA-1 để ký số cho ví dụ được không, hay phải đổi sang SHA-256 ạ?",
    ageMin: 180,
    status: "open",
    reason: "Sinh viên bấm Nhờ giảng viên hỗ trợ",
  },
  {
    id: "tk-3",
    courseId: COURSE_1,
    studentId: "sv-3",
    question: "Em thiếu Bài tập 02 thì còn nộp bù được không ạ?",
    ageMin: 1440,
    status: "open",
    reason: "Câu hỏi về quy chế lớp — không tài liệu nào trả lời được",
  },
  {
    id: "tk-4",
    courseId: COURSE_1,
    studentId: "sv-13",
    question: "QUIZ01 có hỏi phần RSA và hạ tầng khoá công khai (PKI) không ạ?",
    ageMin: 2880,
    status: "open",
    reason: "Độ tin cậy 0,47 < 0,80",
  },
  {
    id: "tk-5",
    courseId: COURSE_1,
    studentId: "sv-5",
    question: "Nhóm em muốn phân tích vụ lộ dữ liệu qua lỗi SQL Injection của một trang thương mại điện tử năm 2024, dùng nguồn báo chí có được tính là vụ tấn công thực tế không ạ?",
    ageMin: 4560,
    status: "open",
    reason: "Độ tin cậy 0,39 < 0,80",
  },
  {
    id: "tk-6",
    courseId: COURSE_1,
    studentId: "sv-9",
    question: "Em làm theo hướng dẫn tuần 8 để nối VPN vào phòng lab nhưng máy báo lỗi chứng chỉ không hợp lệ, em phải sửa ở đâu ạ?",
    ageMin: 45,
    status: "claimed",
    reason: "Độ tin cậy 0,55 < 0,80",
    claimedBy: "Phạm Quốc Bảo",
    claimedAt: "09:12",
  },
];

/**
 * Gộp ticket gốc với phần đã ghi trong trạng thái giả lập: bản đã ghi thắng, ticket mới (D3) nối sau.
 * Cần vì màn khác ghi lát `tickets` với giá trị gốc rỗng (AppShell, /chat) — gộp để không mất ticket gốc.
 */
export function mergeTickets(stored: Ticket[]): Ticket[] {
  return [
    ...TICKETS_SEED.map((t) => stored.find((x) => x.id === t.id) ?? t),
    ...stored.filter((t) => !TICKETS_SEED.some((s) => s.id === t.id)),
  ];
}
