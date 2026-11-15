"use client";

import { useSession } from "@/shared/session/session";
import { Page, PageHeader } from "@/shared/ui";
import { AdminView } from "./AdminView";
import { CourseView } from "./CourseView";

/** Quan sát AI: Quản trị viên thấy toàn hệ thống + bảng yêu cầu; giảng viên chỉ thấy bản tổng hợp một lớp của mình (không nội dung). */
export function ObservabilityScreen() {
  const { role, course, realCourses, realCourseId } = useSession();
  const isAdmin = role === "admin";
  const courseId = realCourses ? (realCourseId && realCourseId !== "all" ? realCourseId : (realCourses[0]?.id ?? null)) : null;
  return (
    <Page width="wide">
      <PageHeader title="Quan sát AI" meta={<span>{isAdmin ? "Toàn hệ thống" : course.label}</span>} />
      {isAdmin ? <AdminView /> : <CourseView courseId={courseId} />}
    </Page>
  );
}
