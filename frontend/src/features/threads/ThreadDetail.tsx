"use client";

import { useSession } from "@/shared/session/session";
import { DemoThreadDetail } from "./DemoThreadDetail";
import { RealThreadDetail } from "./RealThreadDetail";

/** Phiên đăng nhập thật có lớp → chi tiết thật; phiên mô phỏng → bản cũ. */
export function ThreadDetail({ id }: { id: string }) {
  const { realCourseId, role } = useSession();
  if (realCourseId && realCourseId !== "all" && role !== "admin") return <RealThreadDetail courseId={realCourseId} id={id} />;
  return <DemoThreadDetail id={id} />;
}
