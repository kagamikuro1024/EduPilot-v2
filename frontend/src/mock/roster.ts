// Danh sách lớp mô phỏng — chỉ còn /gradebook dùng (US-P5-05 AC10: /students, /students/[id] đã thật).
import { studentById, studentsOf, type Student } from "./core";
import { orderStudents } from "./derive";
import type { MembersState } from "./state";

/** Thành viên hiện tại của một lớp: dữ liệu gốc + người vừa được duyệt ở /class/members. */
export function rosterOf(courseId: string, members: MembersState): Student[] {
  const extra = (members.joined[courseId] ?? []).map(studentById).filter((s): s is Student => s !== undefined && !s.courseIds.includes(courseId));
  // Thứ tự chuẩn của mọi màn: `sv-n` tăng dần (SRS 4.8 N4) — không sắp theo tên.
  return orderStudents([...studentsOf(courseId), ...extra]);
}
