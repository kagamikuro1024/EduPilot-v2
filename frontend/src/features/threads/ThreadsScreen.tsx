"use client";

import { useSession } from "@/shared/session/session";
import dynamic from "next/dynamic";
import { Suspense } from "react";

const DemoThreadsScreen = dynamic(() => import("./DemoThreads").then((m) => m.DemoThreadsScreen), { loading: () => null });
// U2 (US-P10-08): tải chunk ngay khi module màn được nạp, song song với làm mới phiên — không đợi tới lúc có phiên mới bắt đầu tải
const loadRealThreads = () => import("./RealThreads").then((m) => m.RealThreads);
if (typeof window !== "undefined") void loadRealThreads();
const RealThreads = dynamic(loadRealThreads, { loading: () => null });

/** Phiên đăng nhập thật có lớp → Threads thật; phiên mô phỏng (demo) → bản mô phỏng cũ. */
export function ThreadsScreen() {
  const { realCourseId, role, realPending } = useSession();
  if (realPending) return null;
  if (realCourseId && realCourseId !== "all" && role !== "admin") return <Suspense fallback={null}><RealThreads courseId={realCourseId} /></Suspense>;
  return <DemoThreadsScreen />;
}
