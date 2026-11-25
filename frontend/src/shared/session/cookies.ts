/** Giá trị "Tất cả lớp của tôi" của bộ chọn lớp (chỉ GV / TA). Phiên đăng nhập luôn là JWT thật: không còn cookie phiên mô phỏng. */
export const ALL_COURSES = "all";

/** Khoá lưu lựa chọn lớp ở trình duyệt (id lớp hoặc `all`; không phải bí mật). Ở đây (không phải myCourses, "use client") để mã nội tuyến của layout máy chủ đọc được. */
export const COURSE_STORAGE_KEY = "ep:ui:course";
