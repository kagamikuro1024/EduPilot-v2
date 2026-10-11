"use client";

import { useSession } from "@/shared/session/session";
import dynamic from "next/dynamic";

const DemoThreadsScreen = dynamic(() => import("./DemoThreads").then((m) => m.DemoThreadsScreen), { loading: () => null });
const RealThreads = dynamic(() => import("./RealThreads").then((m) => m.RealThreads), { loading: () => null });

/** Phiên đăng nhập thật có lớp → Threads thật; phiên mô phỏng (demo) → bản mô phỏng cũ. */
export function ThreadsScreen() {
  const { realCourseId, role, realPending } = useSession();
  if (realPending) return null;
  if (realCourseId && realCourseId !== "all" && role !== "admin") return <RealThreads courseId={realCourseId} />;
  return <DemoThreadsScreen />;
}
