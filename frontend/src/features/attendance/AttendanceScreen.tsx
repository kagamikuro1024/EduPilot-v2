"use client";

import dynamic from "next/dynamic";
import { useSession } from "@/shared/session/session";

const RealAttendance = dynamic(() => import("./RealAttendance").then((m) => m.RealAttendance), { loading: () => null });

/** Điểm danh chỉ có thật: TA / giảng viên có lớp. */
export function AttendanceScreen() {
  const { realCourseId, realCourses, role, realPending, identity } = useSession();
  if (realPending || !realCourseId || realCourseId === "all" || (role !== "teacher" && role !== "ta")) return null;
  const classCode = realCourses?.find((c) => c.id === realCourseId)?.class_code ?? "";
  return <RealAttendance courseId={realCourseId} userId={identity?.sub ?? "anon"} classCode={classCode} />;
}
