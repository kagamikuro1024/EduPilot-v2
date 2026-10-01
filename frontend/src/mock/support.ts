// Hộp thư hỗ trợ (ticket). Người giữ: US-PROTO-02. Chỉ dùng chung `Ticket`, `TICKETS_SEED`, `ticketD3`.
import { COURSE_1, STUDENT_B } from "./core";
import type { Ticket } from "./state";

/** Câu D3 nguyên văn (DEMO_SCRIPT mục 3). */
export const D3_TEXT = "Thi cuối kỳ có được mang một tờ A4 ghi chú viết tay vào phòng thi không ạ?";

/** Ticket sinh ra khi SV B gửi D3 ở /chat (AI dưới ngưỡng → tự chuyển giảng viên, Q1). */
export function ticketD3(): Ticket {
  return { id: "tk-d3", courseId: COURSE_1, studentId: STUDENT_B.id, question: D3_TEXT, ageMin: 0, status: "open", reason: "Độ tin cậy 0,42 < 0,80 — không tài liệu nào trả lời được" };
}

/** Ticket gốc: US-PROTO-02 điền (5 mở + 1 đã có người nhận, theo SRS 4.1). */
export const TICKETS_SEED: Ticket[] = [];
