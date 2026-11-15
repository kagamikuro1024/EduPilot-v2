"use client";

import { ApiErrorNotice } from "@/shared/data";
import { withCourse } from "@/shared/lib/courseHref";
import { useSession } from "@/shared/session/session";
import { ActionList, ActionRow, ButtonLink, EmptyState, Page, PageHeader, Section } from "@/shared/ui";
import { fmtLong } from "@/features/calendar/calendarApi";
import { dueLabel, useMyAssignments, type StudentListItem } from "./assignmentApi";

/** Một dòng chữ trạng thái (US-P7-03 AC1): không điểm nháp, không cờ, không tiến độ chấm. */
function statusLine(a: StudentListItem): string {
  const m = a.my;
  switch (m.state) {
    case "NOT_SUBMITTED":
      return "Chưa nộp";
    case "GRADING":
      return "Đang chấm";
    case "GRADED":
      return "Đã có điểm";
    default:
      return `Đã nộp${m.submitted_at ? ` ${fmtLong(m.submitted_at)}` : ""}${m.late_days > 0 ? ` · Nộp muộn ${m.late_days} ngày` : ""}`;
  }
}

const GROUPS: Array<{ title: string; part: string; of: (a: StudentListItem) => boolean; action: (a: StudentListItem) => string }> = [
  { title: "Cần nộp", part: "todo", of: (a) => a.my.state === "NOT_SUBMITTED", action: () => "Nộp bài" },
  { title: "Đã nộp, chờ điểm", part: "waiting", of: (a) => a.my.state === "SUBMITTED" || a.my.state === "GRADING", action: (a) => (a.my.state === "SUBMITTED" ? "Nộp lại" : "Xem") },
  { title: "Đã có điểm", part: "graded", of: (a) => a.my.state === "GRADED", action: () => "Xem" },
];

/** `/assignments` của sinh viên (P-11-14, US-P7-03 AC11): ba nhóm, mỗi nhóm một Panel; dữ liệu của `GET …/assignments` (đã công bố, không rubric / đáp án). */
export function StudentAssignmentList() {
  const { identity, realCourseId, realPending } = useSession();
  const courseId = realPending ? null : realCourseId;
  if (!identity || !courseId || courseId === "all") return <Page kind="work"><PageHeader title="Bài tập" /></Page>;
  return <List courseId={courseId} />;
}

function List({ courseId }: { courseId: string }) {
  const q = useMyAssignments(courseId);
  const now = Date.now();
  return (
    <Page kind="work">
      <PageHeader title="Bài tập" />
      {q.isPending ? (
        <Section title="Cần nộp" part="assignments-loading"><ActionList label="Bài tập" loading={3} /></Section>
      ) : q.isError ? (
        <ApiErrorNotice error={q.error} onRetry={() => void q.refetch()} />
      ) : q.items.length === 0 ? (
        <EmptyState title="Chưa có bài tập" />
      ) : (
        GROUPS.map((g) => {
          const rows = q.items.filter(g.of).sort((x, y) => new Date(x.due_at).getTime() - new Date(y.due_at).getTime());
          if (rows.length === 0) return null;
          return (
            <Section key={g.part} title={g.title} part={`assignments-${g.part}`}>
              <ActionList label={g.title}>
                {rows.map((a) => (
                  <ActionRow
                    key={a.id}
                    href={withCourse(`/assignments/${a.id}`, courseId)}
                    title={a.title}
                    context={statusLine(a)}
                    meta={`Hạn ${fmtLong(a.due_at)} · ${dueLabel(a.due_at, now)}`}
                    action={<ButtonLink href={withCourse(`/assignments/${a.id}`, courseId)} variant={g.part === "todo" ? "primary" : "secondary"} size="sm">{g.action(a)}</ButtonLink>}
                    data={{ "data-part": "assignment-row" }}
                  />
                ))}
              </ActionList>
            </Section>
          );
        })
      )}
    </Page>
  );
}
