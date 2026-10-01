import { COURSES, DEMO_STUDENT_IDS, type Role } from "@/mock/core";

// Phiên MÔ PHỎNG: vai trò, người và lớp đang chọn lưu trong cookie (không phải token) để
// server layout đọc được và trang không nháy khi tải lại. Phase P2 thay bằng JWT thật.
export const ROLE_COOKIE = "ep_demo_role";
export const PERSON_COOKIE = "ep_demo_person";
export const COURSE_COOKIE = "ep_demo_course";
/** Giá trị cookie lớp cho "Tất cả lớp của tôi" (chỉ GV / TA). */
export const ALL_COURSES = "all";

const ROLES: readonly string[] = ["student", "ta", "teacher", "admin"];

/** Cookie vai trò hợp lệ → Role; thiếu hoặc sai → null (chưa "đăng nhập"). */
export function parseRole(value: string | undefined): Role | null {
  return value && ROLES.includes(value) ? (value as Role) : null;
}

/** Người vào (sv-1…sv-4) cho vai Sinh viên; mặc định Sinh viên B. */
export function parsePerson(value: string | undefined): string {
  return (DEMO_STUDENT_IDS as readonly string[]).includes(value ?? "") ? (value as string) : "sv-2";
}

export function parseCourse(value: string | undefined): string {
  if (value === ALL_COURSES) return ALL_COURSES;
  return (COURSES.find((c) => c.id === value) ?? COURSES[0]).id;
}

/** Chỉ gọi ở trình duyệt. */
export function writeDemoCookie(name: string, value: string) {
  document.cookie = `${name}=${encodeURIComponent(value)}; path=/; max-age=31536000; samesite=lax`;
}

export function clearDemoSession() {
  for (const name of [ROLE_COOKIE, PERSON_COOKIE, COURSE_COOKIE]) document.cookie = `${name}=; path=/; max-age=0; samesite=lax`;
}
