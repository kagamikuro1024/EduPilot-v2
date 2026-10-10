"use client";

import { useSession } from "@/shared/session/session";
import { DemoThreadsScreen } from "./DemoThreads";
import { RealThreads } from "./RealThreads";

/** Phiên đăng nhập thật có lớp → Threads thật; phiên mô phỏng (demo) → bản mô phỏng cũ. */
export function ThreadsScreen() {
  const { realCourseId, role } = useSession();
  if (realCourseId && realCourseId !== "all" && role !== "admin") return <RealThreads courseId={realCourseId} />;
  return <DemoThreadsScreen />;
}
