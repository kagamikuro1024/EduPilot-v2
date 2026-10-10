"use client";

import { useSession } from "@/shared/session/session";
import { DemoDocuments } from "./DemoDocuments";
import { RealDocuments } from "./RealDocuments";

/** Phiên đăng nhập thật của Giảng viên / TA có lớp → quản lý tài liệu thật; phiên mô phỏng → bản cũ. */
export function Documents() {
  const { realCourseId, role } = useSession();
  if (realCourseId && realCourseId !== "all" && (role === "teacher" || role === "ta")) return <RealDocuments courseId={realCourseId} canDelete={role === "teacher"} />;
  return <DemoDocuments />;
}
