"use client";

import dynamic from "next/dynamic";
import { useSession } from "@/shared/session/session";
import { EmptyState, Page, PageHeader, Panel } from "@/shared/ui";

const RealStudents = dynamic(() => import("./RealStudents").then((m) => m.RealStudents), { loading: () => null });

export function StudentsScreen() {
  const { realCourseId, realCourses, realPending, role } = useSession();
  if (realPending) return null; // chưa biết có lớp thật hay không: không dựng gì
  if ((role !== "teacher" && role !== "ta") || !realCourseId || realCourseId === "all")
    return (
      <Page width="wide">
        <PageHeader title="Sinh viên" />
        <Panel><EmptyState title={realCourseId === "all" ? "Chọn một lớp để xem sinh viên." : "Chưa có lớp nào."} /></Panel>
      </Page>
    );
  return <RealStudents key={realCourseId} courseId={realCourseId} classCode={realCourses?.find((c) => c.id === realCourseId)?.class_code ?? ""} />;
}
