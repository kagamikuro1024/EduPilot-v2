"use client";

// Hợp đồng thật: backend-go/api/openapi.yaml (`/admin/courses*`). Admin không thấy mã tham gia và không thấy danh sách sinh viên.
export type Person = { id: string; full_name: string };
export type AdminCourse = {
  id: string;
  class_code: string;
  subject_code: string;
  name: string;
  semester: string;
  status: "ACTIVE" | "ARCHIVED";
  teacher: Person | null;
  assistants_count: number;
  students_active: number;
  students_pending: number;
  capacity: number | null;
  version: number;
};
export type StaffUser = { id: string; full_name: string; email: string; status: string };
export type AssignResult = { teacher: Person | null; assistants: Person[]; changed: { teacher: boolean; added_ta: string[]; removed_ta: string[] } };

export const COURSES_KEY = ["admin", "courses"] as const;

/** "30 / 30" (đang học / sĩ số); chưa đặt sĩ số thì chỉ số đang học. */
export function sizeText(c: Pick<AdminCourse, "students_active" | "capacity">) {
  return c.capacity === null ? String(c.students_active) : `${c.students_active} / ${c.capacity}`;
}
