"use client";

import dynamic from "next/dynamic";
import { useSession } from "@/shared/session/session";
import { EmptyState, Page, PageHeader, Panel } from "@/shared/ui";
import { StudentsView } from "./StudentsView";

const RealStudents = dynamic(() => import("./RealStudents").then((m) => m.RealStudents), { loading: () => null });

/** Phiên có lớp thật → danh sách thật; phiên mô phỏng (không lớp thật) giữ màn mô phỏng. */
export function StudentsScreen() {
  const { realCourseId, realCourses, realPending, role } = useSession();
  if (realPending) return null; // chưa biết là phiên thật hay mô phỏng: không dựng bản nào
  if ((role !== "teacher" && role !== "ta") || !realCourseId) return <StudentsView />;
  if (realCourseId === "all")
    return (
      <Page width="wide">
        <PageHeader title="Sinh viên" />
        <Panel><EmptyState title="Chọn một lớp để xem sinh viên." /></Panel>
      </Page>
    );
  return <RealStudents key={realCourseId} courseId={realCourseId} classCode={realCourses?.find((c) => c.id === realCourseId)?.class_code ?? ""} />;
}
