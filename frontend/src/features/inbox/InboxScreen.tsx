"use client";

import dynamic from "next/dynamic";
import { useSession } from "@/shared/session/session";

const RealInbox = dynamic(() => import("./RealInbox").then((m) => m.RealInbox), { loading: () => null });

/** Hộp thư hỗ trợ chỉ có thật: TA / giảng viên có lớp. Vai khác không có đường vào (điều hướng không hiện mục này). */
export function InboxScreen() {
  const { realCourseId, role, realPending } = useSession();
  if (realPending || !realCourseId || (role !== "teacher" && role !== "ta")) return null;
  return <RealInbox courseId={realCourseId} />;
}
