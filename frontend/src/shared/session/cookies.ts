import { courseById, type Role } from "@/mock/core";

// Phiên MÔ PHỎNG: vai trò và lớp đang chọn lưu trong cookie (không phải token) để
// server layout đọc được và trang không nháy khi tải lại. Phase P2 thay bằng JWT thật.
export const ROLE_COOKIE = "ep_demo_role";
export const COURSE_COOKIE = "ep_demo_course";

const ROLES: readonly string[] = ["student", "ta", "teacher", "admin"];

/** Cookie vai trò hợp lệ → Role; thiếu hoặc sai → null (chưa "đăng nhập"). */
export function parseRole(value: string | undefined): Role | null {
  return value && ROLES.includes(value) ? (value as Role) : null;
}

export function parseCourse(value: string | undefined) {
  return courseById(value ?? "").id;
}

/** Chỉ gọi ở trình duyệt. */
export function writeDemoCookie(name: string, value: string) {
  document.cookie = `${name}=${encodeURIComponent(value)}; path=/; max-age=31536000; samesite=lax`;
}

export function clearDemoSession() {
  for (const name of [ROLE_COOKIE, COURSE_COOKIE]) document.cookie = `${name}=; path=/; max-age=0; samesite=lax`;
}
