"use client";

import dynamic from "next/dynamic";
import { useSession } from "@/shared/session/session";

// Cả hai bản nạp lười: Next prefetch chunk của mọi route trong thanh bên, nên bản tĩnh làm MỌI route vượt ngân sách JS 250 KB gzip (Lighthouse CI `resource-summary:script`); mỗi lúc chỉ nạp một bản.
const DemoChatScreen = dynamic(() => import("./DemoChat").then((m) => m.DemoChatScreen), { loading: () => null });
const RealChat = dynamic(() => import("./RealChat").then((m) => m.RealChat), { loading: () => null });

/** Phiên đăng nhập thật có lớp → chat thật (SSE); phiên mô phỏng (demo) → bản mô phỏng cũ. */
export function ChatScreen() {
  const { realCourseId, role, realPending } = useSession();
  if (realPending) return null;
  if (role === "student" && realCourseId && realCourseId !== "all") return <RealChat courseId={realCourseId} />;
  return <DemoChatScreen />;
}
