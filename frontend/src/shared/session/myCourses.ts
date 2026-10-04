"use client";

import { useQuery } from "@tanstack/react-query";
import { apiClient, type CursorPage } from "@/shared/data";
import { COURSES, type Course } from "@/mock/core";

/** Một lớp thật của người đăng nhập (`GET /me/courses`); không có mã tham gia. */
export type RealCourse = {
  id: string;
  class_code: string;
  subject_code: string;
  name: string;
  semester: string;
  status: "ACTIVE" | "ARCHIVED";
  role_in_course: "TEACHER" | "TA" | "STUDENT";
  enrollment_status: "ACTIVE" | "PENDING";
};

type Item = { course: Omit<RealCourse, "role_in_course" | "enrollment_status">; role_in_course: RealCourse["role_in_course"]; enrollment_status: RealCourse["enrollment_status"] };

/** Khoá lưu lựa chọn lớp ở trình duyệt (id lớp hoặc `all`; không phải bí mật). */
export const COURSE_STORAGE_KEY = "ep:ui:course";

export function readStoredCourse(): string | null {
  try {
    return localStorage.getItem(COURSE_STORAGE_KEY);
  } catch {
    return null; // bộ nhớ bị chặn: chỉ mất việc nhớ
  }
}

export function writeStoredCourse(id: string) {
  try {
    localStorage.setItem(COURSE_STORAGE_KEY, id);
  } catch { /* như trên */ }
}

/**
 * Lớp thật của phiên `jwt`. `data` là `undefined` khi chưa tải xong / lỗi — khung giữ nguyên cách chạy cũ (màn mô phỏng) cho tới khi có dữ liệu thật.
 * Một trang tối đa 100 lớp (ponytail: đủ cho một người; thêm "tải tiếp" khi có người vượt).
 */
export function useMyCourses(enabled: boolean) {
  return useQuery({
    queryKey: ["me", "courses"],
    enabled,
    queryFn: async ({ signal }): Promise<RealCourse[]> => {
      const page = (await apiClient.get<CursorPage<Item>>("/me/courses", { signal, query: { limit: 100 } })).data;
      return page.items.map((i) => ({ ...i.course, role_in_course: i.role_in_course, enrollment_status: i.enrollment_status }));
    },
  });
}

/** Màn mô phỏng chưa dựng lại dùng lớp mô phỏng tương ứng theo `class_code` (761987 ↔ int1006-1, 761988 ↔ int1006-2); lớp khác → lớp đầu. */
export function mockCourseFor(classCode: string): Course {
  return COURSES.find((c) => c.code === classCode) ?? COURSES[0];
}
