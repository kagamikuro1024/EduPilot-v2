"use client";

import dynamic from "next/dynamic";
import { useSession } from "@/shared/session/session";
import { EmptyState, Page, PageHeader, Panel } from "@/shared/ui";

const RealProfile = dynamic(() => import("./RealProfile").then((m) => m.RealProfile), { loading: () => null });

export function StudentProfileScreen({ id }: { id: string }) {
  const { realCourseId, realCourses, realPending, role, identity } = useSession();
  if (realPending) return null;
  if ((role !== "teacher" && role !== "ta") || !realCourseId || realCourseId === "all")
    return (
      <Page>
        <PageHeader title="Hồ sơ sinh viên" back={{ href: "/students", label: "Sinh viên" }} />
        <Panel><EmptyState title={realCourseId === "all" ? "Chọn một lớp để xem hồ sơ." : "Chưa có lớp nào."} /></Panel>
      </Page>
    );
  return (
    <RealProfile
      key={`${realCourseId}:${id}`}
      courseId={realCourseId}
      studentId={id}
      classCode={realCourses?.find((c) => c.id === realCourseId)?.class_code ?? ""}
      userId={identity?.sub ?? "anon"}
      canManage={role === "teacher"}
    />
  );
}
