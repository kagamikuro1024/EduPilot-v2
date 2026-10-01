// Dữ liệu chấm bài dùng chung giữa /grading (US-PROTO-03) và /assignments/bt03 (US-PROTO-01).
import type { Bt03State } from "./state";

export const BT03_CRITERIA = ["Xác định tác nhân và bề mặt tấn công", "Phân tích tấn công", "Đánh giá tác động", "Biện pháp phòng thủ"] as const;
export const BT03_MAX_PER_CRITERION = 2.5;

/** Bản chấm nháp của AI (lượt 1). Lượt 2 chấm tiêu chí 2 = 2,5 → lệch 1,5 → "Cần xem kỹ". */
export const BT03_SEED: Bt03State = {
  status: "draft",
  scores: [2.5, 1.0, 2.0, 2.0],
  comments: [
    "Bài nêu rõ nhóm tác nhân và các điểm vào của hệ thống bị tấn công.",
    "Mô tả chuỗi tấn công còn thiếu bước leo thang đặc quyền.",
    "Đánh giá được thiệt hại dữ liệu nhưng chưa định lượng thời gian gián đoạn.",
    "Có đề xuất phân đoạn mạng và xác thực đa yếu tố, chưa nêu cách giám sát.",
  ],
};
/** Lượt chấm thứ hai của AI cho tiêu chí 2 (để hiện thông báo lệch). */
export const BT03_SECOND_PASS_CRITERION_2 = 2.5;
