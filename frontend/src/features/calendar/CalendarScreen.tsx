"use client";

import { useSession } from "@/shared/session/session";
import dynamic from "next/dynamic";

const DemoCalendar = dynamic(() => import("./DemoCalendar").then((m) => m.DemoCalendar), { loading: () => null });
const RealCalendar = dynamic(() => import("./RealCalendar").then((m) => m.RealCalendar), { loading: () => null });

/** Phiên đăng nhập thật có lớp → lịch thật; phiên mô phỏng → bản cũ. */
export function CalendarScreen() {
  const { realCourseId, role, realPending } = useSession();
  if (realPending) return null;
  if (realCourseId && realCourseId !== "all" && (role === "student" || role === "teacher" || role === "ta")) return <RealCalendar courseId={realCourseId} staff={role !== "student"} />;
  return <DemoCalendar />;
}
