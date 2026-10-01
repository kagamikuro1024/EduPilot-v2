import { courseById, type Role } from "@/mock/core";

// Phiên MÔ PHỎNG: vai trò và lớp đang chọn lưu trong cookie (không phải token) để
// server layout đọc được và trang không nháy khi tải lại. Phase P2 thay bằng JWT thật.
export const ROLE_COOKIE = "ep_demo_role";
export const COURSE_COOKIE = "ep_demo_course";

export function parseRole(value: string | undefined): Role {
  return value === "student" || value === "ta" || value === "teacher" || value === "admin" ? value : "teacher";
}

export function parseCourse(value: string | undefined) {
  return courseById(value ?? "").id;
}
