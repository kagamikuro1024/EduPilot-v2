"use client";

import { useSession } from "@/shared/session/session";
import { DemoCalendar } from "./DemoCalendar";
import { RealCalendar } from "./RealCalendar";

/** Phiên đăng nhập thật có lớp → lịch thật; phiên mô phỏng → bản cũ. */
export function CalendarScreen() {
  const { realCourseId, role } = useSession();
  if (realCourseId && realCourseId !== "all" && (role === "student" || role === "teacher" || role === "ta")) return <RealCalendar courseId={realCourseId} staff={role !== "student"} />;
  return <DemoCalendar />;
}
