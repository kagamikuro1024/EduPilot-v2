"use client";

import dynamic from "next/dynamic";
import { useSession } from "@/shared/session/session";

const StudentAssignment = dynamic(() => import("./StudentAssignment").then((m) => m.StudentAssignment), { loading: () => null });

/** `/assignments/[id]` là màn của sinh viên (nav chặn vai khác). Lớp đang chọn quyết định bài nào đọc được. */
export function AssignmentScreen({ id }: { id: string }) {
  const { realCourseId, realPending } = useSession();
  if (realPending || !realCourseId || realCourseId === "all") return null;
  return <StudentAssignment key={`${realCourseId}:${id}`} courseId={realCourseId} id={id} />;
}
