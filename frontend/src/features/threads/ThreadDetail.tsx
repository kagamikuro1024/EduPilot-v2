"use client";

import { useSession } from "@/shared/session/session";
import dynamic from "next/dynamic";

const DemoThreadDetail = dynamic(() => import("./DemoThreadDetail").then((m) => m.DemoThreadDetail), { loading: () => null });
const RealThreadDetail = dynamic(() => import("./RealThreadDetail").then((m) => m.RealThreadDetail), { loading: () => null });

/** Phiên đăng nhập thật có lớp → chi tiết thật; phiên mô phỏng → bản cũ. */
export function ThreadDetail({ id }: { id: string }) {
  const { realCourseId, role, realPending } = useSession();
  if (realPending) return null;
  if (realCourseId && realCourseId !== "all" && role !== "admin") return <RealThreadDetail courseId={realCourseId} id={id} />;
  return <DemoThreadDetail id={id} />;
}
