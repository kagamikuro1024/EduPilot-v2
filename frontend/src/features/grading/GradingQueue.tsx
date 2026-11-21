"use client";

import { useEffect, useState } from "react";
import { useRouter, useSearchParams } from "next/navigation";
import { Plus } from "lucide-react";
import { EmptyState, Page, PageHeader, Panel, Skeleton, Tabs } from "@/shared/ui";
import { AssignmentsTab } from "@/features/assignments/AssignmentsTab";
import { useSession } from "@/shared/session/session";
import { useQueue } from "./gradingApi";
import { QueueTab, type QueueHeader } from "./QueueTab";
import { UnmatchedTab } from "./UnmatchedTab";
import s from "./Grading.module.css";

type Tab = "queue" | "assignments" | "unmatched";

/** Không có `?tab=`: `Hàng chờ` khi còn việc chờ, ngược lại `Bài tập` (US-P7-02 AC1). Chọn xong mới dựng tab (không để QueueTab ghi `?tab=queue` trước). Dùng chung bộ nhớ với «mọi bài nộp» của QueueTab. */
function DefaultTab({ courseId, onPick }: { courseId: string; onPick: (t: Tab) => void }) {
  const all = useQueue(courseId, {});
  const ready = !all.isPending;
  const work = all.isError || all.items.some((r) => r.result?.status !== "PUBLISHED"); // lỗi tải: giữ Hàng chờ (tự báo lỗi)
  useEffect(() => { if (ready) onPick(work ? "queue" : "assignments"); }, [ready]); // eslint-disable-line react-hooks/exhaustive-deps
  return <Panel><Skeleton lines={6} /></Panel>;
}

/** `/grading` (TA, TEACHER): ba tab — hàng chờ chấm, bài tập, bài chưa khớp. Tab nằm ở URL (`?tab=`). */
export function GradingQueue() {
  const { role, realCourseId } = useSession();
  const params = useSearchParams();
  const router = useRouter();
  const [creating, setCreating] = useState(false);
  const [queueHdr, setQueueHdr] = useState<QueueHeader | null>(null);

  const asked = params.get("tab");
  const tab: Tab = asked === "assignments" || asked === "unmatched" ? asked : "queue";
  const setTab = (v: Tab) => router.replace(`/grading?tab=${v}`, { scroll: false });
  const teacher = role === "teacher";

  return (
    <Page kind="work">
      <PageHeader
        title="Chấm bài"
        primary={tab === "assignments" && teacher ? { label: "Tạo bài tập", icon: <Plus aria-hidden />, onClick: () => setCreating(true) } : tab === "queue" ? queueHdr?.primary : undefined}
        secondary={tab === "queue" ? queueHdr?.secondary : undefined}
      />

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
          <EmptyState title="Chọn một lớp để chấm bài" />
        </Panel>
      ) : !asked ? (
        <DefaultTab courseId={realCourseId} onPick={setTab} />
      ) : tab === "assignments" ? (
        <AssignmentsTab courseId={realCourseId} teacher={teacher} creating={creating} onCreatingChange={setCreating} />
      ) : tab === "unmatched" ? (
        <UnmatchedTab courseId={realCourseId} />
      ) : (
        <QueueTab courseId={realCourseId} teacher={teacher} onHeader={setQueueHdr} />
      )}
    </Page>
  );
}
