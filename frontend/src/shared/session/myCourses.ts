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

export function clearStoredCourse() {
  try {
    localStorage.removeItem(COURSE_STORAGE_KEY);
  } catch { /* như trên */ }
}

export function writeStoredCourse(id: string) {
  try {
    localStorage.setItem(COURSE_STORAGE_KEY, id);
  } catch { /* như trên */ }
}

type MyCoursesData = { courses: RealCourse[]; acked: boolean; optOutAvailable: boolean };
type MyCoursesPage = CursorPage<Item> & { tracking_notice_acked?: boolean; study_time_opt_out_available?: boolean };

// Một truy vấn, hai người đọc (lớp thật; hộp thoại thời gian học): chung khoá nên chỉ MỘT yêu cầu `GET /me/courses`.
const myCoursesQuery = (enabled: boolean) => ({
  queryKey: ["me", "courses"] as const,
  enabled,
  queryFn: async ({ signal }: { signal: AbortSignal }): Promise<MyCoursesData> => {
    const page = (await apiClient.get<MyCoursesPage>("/me/courses", { signal, query: { limit: 100 } })).data;
    return {
      courses: page.items.map((i) => ({ ...i.course, role_in_course: i.role_in_course, enrollment_status: i.enrollment_status })),
      acked: page.tracking_notice_acked ?? true, // máy chủ cũ / gateway giả không có cờ: coi như đã xác nhận (không hộp thoại, không nhịp)
      optOutAvailable: page.study_time_opt_out_available ?? false,
    };
  },
});

/** Hộp thoại minh bạch thời gian học (US-P5-03 AC7): `undefined` khi chưa tải xong. */
export function useTrackingNotice(enabled: boolean) {
  return useQuery({ ...myCoursesQuery(enabled), select: (d) => ({ acked: d.acked, optOutAvailable: d.optOutAvailable }) });
}

/**
 * Lớp thật của phiên `jwt`. `data` là `undefined` khi chưa tải xong / lỗi — khung giữ nguyên cách chạy cũ (màn mô phỏng) cho tới khi có dữ liệu thật.
 * Một trang tối đa 100 lớp (ponytail: đủ cho một người; thêm "tải tiếp" khi có người vượt).
 */
export function useMyCourses(enabled: boolean) {
  return useQuery({ ...myCoursesQuery(enabled), select: (d) => d.courses });
}

/** Màn mô phỏng chưa dựng lại dùng lớp mô phỏng tương ứng theo `class_code` (761987 ↔ int1006-1, 761988 ↔ int1006-2); lớp khác → lớp đầu. */
export function mockCourseFor(classCode: string): Course {
  return COURSES.find((c) => c.code === classCode) ?? COURSES[0];
}

/**
 * Lần đầu (chưa nhớ lớp) giảng viên / trợ giảng có ≥ 2 lớp (FEAT-uiux-91 US-06 AC2): lớp mặc định là lớp ĐẦU TIÊN theo thứ tự `GET /me/courses`
 * mà `GET /me/today` có ≥ 1 mục thuộc lớp đó; không lớp nào có thì `null` (dùng lớp đầu danh sách). Không đổi API.
 */
export function useFirstCourseWithData(enabled: boolean, courses: RealCourse[] | null) {
  return useQuery({
    queryKey: ["me", "today", "first-course"] as const,
    enabled: enabled && !!courses,
    staleTime: 60_000,
    queryFn: async ({ signal }) => {
      const t = (await apiClient.get<{ actions?: Array<{ course: { id: string } | null }>; attention?: Array<{ course: { id: string } }>; upcoming?: Array<{ course: { id: string } }> }>("/me/today", { signal })).data;
      const withData = new Set<string>([...(t.actions ?? []).map((a) => a.course?.id), ...(t.attention ?? []).map((a) => a.course.id), ...(t.upcoming ?? []).map((a) => a.course.id)].filter((x): x is string => !!x));
      return courses?.find((c) => withData.has(c.id))?.id ?? null;
    },
  });
}
