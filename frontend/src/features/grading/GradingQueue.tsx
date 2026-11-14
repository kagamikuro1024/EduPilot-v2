"use client";

import { useState } from "react";
import { useRouter, useSearchParams } from "next/navigation";
import { Plus } from "lucide-react";
import { EmptyState, Page, PageHeader, Panel, Tabs } from "@/shared/ui";
import { AssignmentsTab } from "@/features/assignments/AssignmentsTab";
import { useSession } from "@/shared/session/session";
import { QueueTab } from "./QueueTab";
import { UnmatchedTab } from "./UnmatchedTab";
import s from "./Grading.module.css";

type Tab = "queue" | "assignments" | "unmatched";

/** `/grading` (TA, TEACHER): ba tab — hàng chờ chấm, bài tập, bài chưa khớp. Tab nằm ở URL (`?tab=`). */
export function GradingQueue() {
  const { role, realCourseId } = useSession();
  const params = useSearchParams();
  const router = useRouter();
  const [creating, setCreating] = useState(false);

  const asked = params.get("tab");
  const tab: Tab = asked === "assignments" || asked === "unmatched" ? asked : "queue";
  const setTab = (v: Tab) => router.replace(`/grading?tab=${v}`, { scroll: false });
  const teacher = role === "teacher";

  return (
    <Page kind="work">
      <PageHeader title="Chấm bài" primary={tab === "assignments" && teacher ? { label: "Tạo bài tập", icon: <Plus aria-hidden />, onClick: () => setCreating(true) } : undefined} />

      <div className={s.tabsRow}>
        <Tabs
          label="Phần của màn chấm bài"
          value={tab}
          onChange={setTab}
          options={[
            { value: "queue", label: "Hàng chờ" },
            { value: "assignments", label: "Bài tập" },
            { value: "unmatched", label: "Chưa khớp" },
          ]}
        />
      </div>

      {!realCourseId || realCourseId === "all" ? (
        <Panel>
          <EmptyState title="Chọn một lớp để chấm bài." />
        </Panel>
      ) : tab === "assignments" ? (
        <AssignmentsTab courseId={realCourseId} teacher={teacher} creating={creating} onCreatingChange={setCreating} />
      ) : tab === "unmatched" ? (
        <UnmatchedTab courseId={realCourseId} />
      ) : (
        <QueueTab courseId={realCourseId} teacher={teacher} />
      )}
    </Page>
  );
}
