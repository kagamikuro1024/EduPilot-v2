"use client";

import { useSession } from "@/shared/session/session";
import { DemoChatScreen, CHAT_DRAFT_KEY } from "./DemoChat";
import { RealChat } from "./RealChat";

export { CHAT_DRAFT_KEY };

/** Phiên đăng nhập thật có lớp → chat thật (SSE); phiên mô phỏng (demo) → bản mô phỏng cũ. */
export function ChatScreen() {
  const { realCourseId, role } = useSession();
  if (role === "student" && realCourseId && realCourseId !== "all") return <RealChat courseId={realCourseId} />;
  return <DemoChatScreen />;
}
