"use client";

import dynamic from "next/dynamic";
import { useSession } from "@/shared/session/session";
import { EmptyState, Page, PageHeader, Panel } from "@/shared/ui";

const RealMe = dynamic(() => import("./RealMe").then((m) => m.RealMe), { loading: () => null });

export function MeScreen() {
  const { realCourseId, realCourses, realPending } = useSession();
  if (realPending) return null;
  if (!realCourseId || realCourseId === "all")
    return (
      <Page>
        <PageHeader title="Kết quả của tôi" />
        <Panel><EmptyState title="Chưa có lớp nào." /></Panel>
      </Page>
    );
  return <RealMe key={realCourseId} courseId={realCourseId} classCode={realCourses?.find((c) => c.id === realCourseId)?.class_code ?? ""} />;
}
