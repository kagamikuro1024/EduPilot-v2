// Nhật ký sửa điểm (SRS 4.5: mỗi thay đổi điểm đều ghi lại người sửa và thời điểm). MỘT kho trong `ep_demo_state`;
// nơi sửa điểm / duyệt / công bố gọi `logGrade`, hộp "Lịch sử sửa điểm" của sổ điểm đọc `gradeAuditOf`.
import { simNowMs } from "@/shared/state/clock";
import { writeSlice } from "@/shared/state/demo";

export const GRADE_AUDIT_KEY = "audit.grades";

export type GradeAudit = { id: string; studentId: string; atMs: number; by: string; text: string };

export function logGrade(studentId: string, by: string, text: string) {
  const atMs = simNowMs();
  writeSlice<GradeAudit[]>(GRADE_AUDIT_KEY, (prev) => [{ id: `ga-${atMs}-${(prev ?? []).length}`, studentId, atMs, by, text }, ...(prev ?? [])]);
}

export function gradeAuditOf(all: GradeAudit[], studentId: string): GradeAudit[] {
  return all.filter((x) => x.studentId === studentId).sort((a, b) => b.atMs - a.atMs);
}

/** "29/10 09:24" — cùng khuôn với các dòng lịch sử có sẵn. */
export function fmtAuditAt(ms: number): string {
  const d = new Date(ms);
  const p = (n: number) => String(n).padStart(2, "0");
  return `${p(d.getDate())}/${p(d.getMonth() + 1)} ${p(d.getHours())}:${p(d.getMinutes())}`;
}
