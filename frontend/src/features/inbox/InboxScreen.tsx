"use client";

import dynamic from "next/dynamic";
import { useSession } from "@/shared/session/session";

// U2 (US-P10-08): tải chunk ngay khi module màn được nạp, song song với làm mới phiên — không đợi tới lúc có phiên mới bắt đầu tải
const loadRealInbox = () => import("./RealInbox").then((m) => m.RealInbox);
if (typeof window !== "undefined") void loadRealInbox();
const RealInbox = dynamic(loadRealInbox, { loading: () => null });

/** Hộp thư hỗ trợ chỉ có thật: TA / giảng viên có lớp. Vai khác không có đường vào (điều hướng không hiện mục này). */
export function InboxScreen() {
  const { realCourseId, role, realPending } = useSession();
  if (realPending || !realCourseId || (role !== "teacher" && role !== "ta")) return null;
  return <RealInbox courseId={realCourseId} />;
}
