// Dữ liệu GIẢ cố định cho trang mẫu /dev/panels (US-UI-01): không người thật, không gọi API.
export const TEACHER_TASKS = [
  { id: "t1", tone: "red", title: "12 bài thi chờ chấm — Kiểm tra tuần 9", context: "Hạn công bố điểm: thứ Sáu. 4 bài có câu code cần xem lại.", meta: "Lớp 761987", action: "Chấm bài", primary: true },
  { id: "t2", tone: "amber", title: "3 yêu cầu vào lớp đang chờ duyệt", context: "Email chưa khớp danh sách lớp — cần giảng viên xem.", meta: "Lớp 761988", action: "Xem yêu cầu", primary: false },
  { id: "t3", tone: "blue", title: "Duyệt 5 câu hỏi AI gợi ý", context: "Chủ đề Mật mã học; chưa có câu nào vào bài thi.", meta: "Ngân hàng câu hỏi", action: "Duyệt câu", primary: false },
  { id: "t4", tone: "neutral", title: "1 yêu cầu xem lại điểm", context: "Sinh viên hỏi về câu 4 của bài kiểm tra tuần 8.", meta: "Lớp 761987", action: "Trả lời", primary: false },
] as const;

export const TEACHER_STATS = [
  { id: "s1", value: "12", label: "bài chờ chấm" },
  { id: "s2", value: "3", label: "yêu cầu vào lớp" },
] as const;

export const ATTENTION = [
  { id: "a1", title: "Nhóm 4 sinh viên vắng từ 2 buổi liên tiếp", context: "Lớp 761987 · buổi 7 và buổi 8", meta: "Điểm danh" },
  { id: "a2", title: "Điểm giữa kỳ của lớp 761988 thấp hơn lớp 761987 1,2 điểm", context: "Phần Mật mã bất đối xứng sai nhiều nhất (câu 5, đúng 33 %)", meta: "Bài kiểm tra tuần 8" },
  { id: "a3", title: "2 sinh viên chưa nộp bài lab 3", context: "Hạn nộp đã qua 1 ngày", meta: "Lớp 761987" },
] as const;

export const UPCOMING = [
  { id: "u1", title: "Buổi 9 — Mã hoá đối xứng", context: "Thứ Năm · 09:00 · Phòng 302", meta: "Lớp 761987" },
  { id: "u2", title: "Hạn nộp bài lab 3", context: "Thứ Sáu · 23:59", meta: "Cả hai lớp" },
  { id: "u3", title: "Bài kiểm tra tuần 10 mở", context: "Thứ Hai · 08:00 · làm bài 45 phút", meta: "Lớp 761987" },
] as const;

export const STUDENT_NEXT = {
  title: "Làm tiếp bài kiểm tra tuần 9 — Mật mã",
  reason: "Bạn đã làm 6 trên 10 câu; bài tự lưu, còn 18 phút trước giờ đóng.",
  minutes: "≈ 15 phút",
  action: "Làm tiếp",
} as const;

export const STUDENT_DAY = [
  { id: "d1", time: "08:00", title: "Buổi 9 — Mã hoá đối xứng", context: "Phòng 302", need: false },
  { id: "d2", time: "17:00", title: "Bài kiểm tra tuần 9 đóng", context: "Bài của bạn còn 4 câu chưa làm", need: true },
  { id: "d3", time: "23:59", title: "Hạn nộp bài lab 3", context: "Bạn đã nộp bản đầu; có thể nộp lại", need: false },
] as const;
