"use client";

import { useSession } from "@/shared/session/session";
import type { RealCourse } from "@/shared/session/myCourses";

// Hợp đồng thật: backend-go/api/openapi.yaml (`/courses/{id}/join-code`, `/join-settings`, `/members*`).
export type JoinInfo = {
  join_code: string;
  join_url: string;
  enabled: boolean;
  expires_at: string | null;
  require_approval: boolean;
  allowed_email_domain: string | null;
  capacity: number | null;
  active_students: number;
  pending: number;
  version: number;
};
export type Member = {
  user_id: string;
  full_name: string;
  email: string;
  student_code: string;
  role_in_course: "TEACHER" | "TA" | "STUDENT";
  status: "ACTIVE" | "PENDING" | "REMOVED";
  joined_via: "ADMIN" | "ROSTER" | "CODE";
  warning: "EMAIL_MISMATCH" | "EMAIL_UNVERIFIED" | null;
  status_changed_at: string;
};
export type MemberCounts = { active: number; pending: number };

export const classKey = (courseId: string, ...rest: string[]) => ["class", courseId, ...rest] as const;

export type ClassCourse = { state: "loading" } | { state: "none" } | { state: "ready"; course: RealCourse; canManage: boolean };

/**
 * Lớp mà màn "Quản lý lớp" đang làm việc: lớp thật đang chọn nếu người này là giảng viên / TA của lớp đó, nếu không thì lớp quản lý đầu tiên.
 * `canManage` = giảng viên (TA chỉ xem). Sinh viên không bao giờ tới đây (khung chặn quyền); Admin chưa chọn lớp nào ⇒ "none".
 */
export function useClassCourse(): ClassCourse {
  const { realCourses, realCourseId } = useSession();
  if (!realCourses) return { state: "loading" };
  const staff = realCourses.filter((c) => c.role_in_course !== "STUDENT");
  const course = staff.find((c) => c.id === realCourseId) ?? staff[0];
  return course ? { state: "ready", course, canManage: course.role_in_course === "TEACHER" } : { state: "none" };
}
