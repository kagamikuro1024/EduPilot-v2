"use client";

import { useSession } from "@/shared/session/session";
import { EmptyState, Page, PageHeader, Panel } from "@/shared/ui";
import { KnowledgeGap } from "./KnowledgeGap";

/** Lỗ hổng kiến thức của lớp đang chọn (giảng viên / TA): đọc báo cáo thật; không có bản mô phỏng. */
export function InsightsScreen() {
  const { realCourses, realCourseId, realPending } = useSession();
  const courseId = realCourses ? (realCourseId && realCourseId !== "all" ? realCourseId : (realCourses[0]?.id ?? null)) : null;
  if (realPending) return null;
  if (!courseId) {
    return (
      <Page kind="read">
        <PageHeader title="Lỗ hổng kiến thức" />
        <Panel>
          <EmptyState title="Chưa có lớp nào" />
        </Panel>
      </Page>
    );
  }
  return <KnowledgeGap courseId={courseId} />;
}
