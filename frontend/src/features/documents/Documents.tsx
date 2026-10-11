"use client";

import { useSession } from "@/shared/session/session";
import dynamic from "next/dynamic";

const DemoDocuments = dynamic(() => import("./DemoDocuments").then((m) => m.DemoDocuments), { loading: () => null });
const RealDocuments = dynamic(() => import("./RealDocuments").then((m) => m.RealDocuments), { loading: () => null });

/** Phiên đăng nhập thật của Giảng viên / TA có lớp → quản lý tài liệu thật; phiên mô phỏng → bản cũ. */
export function Documents() {
  const { realCourseId, role, realPending } = useSession();
  if (realPending) return null;
  if (realCourseId && realCourseId !== "all" && (role === "teacher" || role === "ta")) return <RealDocuments courseId={realCourseId} canDelete={role === "teacher"} />;
  return <DemoDocuments />;
}
