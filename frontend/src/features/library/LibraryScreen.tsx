"use client";

import { useSession } from "@/shared/session/session";
import dynamic from "next/dynamic";

const DemoLibraryScreen = dynamic(() => import("./DemoLibrary").then((m) => m.DemoLibraryScreen), { loading: () => null });
const RealLibrary = dynamic(() => import("./RealLibrary").then((m) => m.RealLibrary), { loading: () => null });
const RealLibraryDetail = dynamic(() => import("./RealLibraryDetail").then((m) => m.RealLibraryDetail), { loading: () => null });

const real = (s: ReturnType<typeof useSession>) => (s.role === "student" && s.realCourseId && s.realCourseId !== "all" ? s.realCourseId : null);

/** Phiên đăng nhập thật của sinh viên có lớp → thư viện thật; phiên mô phỏng → bản cũ. */
export function LibraryScreen() {
  const s = useSession();
  const course = real(s);
  if (s.realPending) return null;
  return course ? <RealLibrary courseId={course} /> : <DemoLibraryScreen />;
}

export function LibraryDetail({ id }: { id: string }) {
  const s = useSession();
  const course = real(s);
  if (s.realPending) return null;
  return course ? <RealLibraryDetail courseId={course} id={id} /> : <DemoLibraryScreen />;
}
