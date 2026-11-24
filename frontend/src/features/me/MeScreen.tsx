"use client";

import dynamic from "next/dynamic";
import { useSession } from "@/shared/session/session";
import { EmptyState, Page, PageHeader, Panel } from "@/shared/ui";

// U2 (US-P10-08): tải chunk ngay khi module màn được nạp, song song với làm mới phiên — không đợi tới lúc có phiên mới bắt đầu tải
const loadRealMe = () => import("./RealMe").then((m) => m.RealMe);
if (typeof window !== "undefined") void loadRealMe();
const RealMe = dynamic(loadRealMe, { loading: () => null });

export function MeScreen() {
  const { realCourseId, realPending } = useSession();
  if (realPending) return null;
  if (!realCourseId || realCourseId === "all")
    return (
      <Page kind="read">
        <PageHeader title="Kết quả của tôi" />
        <Panel><EmptyState title="Chưa có lớp nào" /></Panel>
      </Page>
    );
  return <RealMe key={realCourseId} courseId={realCourseId} />;
}
