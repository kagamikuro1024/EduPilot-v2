"use client";

import dynamic from "next/dynamic";
import { useSession } from "@/shared/session/session";

const DemoInbox = dynamic(() => import("./InboxView").then((m) => m.InboxView), { loading: () => null });
const RealInbox = dynamic(() => import("./RealInbox").then((m) => m.RealInbox), { loading: () => null });

/** Phiên đăng nhập thật có lớp → hộp thư thật; phiên mô phỏng → bản mô phỏng cũ. */
export function InboxScreen() {
  const { realCourseId, role, realPending } = useSession();
  if (realPending) return null;
  if (realCourseId && role !== "admin" && role !== "student") return <RealInbox courseId={realCourseId} />;
  return <DemoInbox />;
}
