"use client";

import { useSession } from "@/shared/session/session";
import { DemoLibraryScreen } from "./DemoLibrary";
import { RealLibrary } from "./RealLibrary";
import { RealLibraryDetail } from "./RealLibraryDetail";

const real = (s: ReturnType<typeof useSession>) => (s.role === "student" && s.realCourseId && s.realCourseId !== "all" ? s.realCourseId : null);

/** Phiên đăng nhập thật của sinh viên có lớp → thư viện thật; phiên mô phỏng → bản cũ. */
export function LibraryScreen() {
  const course = real(useSession());
  return course ? <RealLibrary courseId={course} /> : <DemoLibraryScreen />;
}

export function LibraryDetail({ id }: { id: string }) {
  const course = real(useSession());
  return course ? <RealLibraryDetail courseId={course} id={id} /> : <DemoLibraryScreen />;
}
