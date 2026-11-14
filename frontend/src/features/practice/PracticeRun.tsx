"use client";

import { useSearchParams } from "next/navigation";
import { useEffect, useState } from "react";
import { ApiError, ApiErrorNotice } from "@/shared/data";
import { makeExamClock } from "@/shared/lib/examClock";
import { useSession } from "@/shared/session/session";
import { EmptyState, Page, PageHeader, Panel, Skeleton } from "@/shared/ui";
import { MockRun } from "./MockRun";
import { PracticeResult } from "./PracticeResult";
import { TopicRun } from "./TopicRun";
import { getAttempt, newTabId, useStudentCourse, type PAttempt } from "./practiceApi";

type Load = { kind: "loading" } | { kind: "error"; error: unknown } | { kind: "ready"; course: string; attempt: PAttempt };

/**
 * `/practice/[attemptId]`: theo trạng thái — đang làm theo chủ đề → từng câu có phản hồi; đang thi thử → màn thi có đồng hồ; đã đóng → kết quả.
 * Lớp của lượt không có trong đường dẫn: thử lớp gợi ý (`?course=`), rồi các lớp sinh viên còn lại; mở lượt MOCK bằng trang này là giành quyền ghi cho tab này.
 */
export function PracticeRun({ attemptId }: { attemptId: string }) {
  const hint = useSearchParams().get("course");
  const c = useStudentCourse();
  const { identity } = useSession();
  const [clock] = useState(makeExamClock);
  const [tab] = useState(newTabId);
  const [load, setLoad] = useState<Load>({ kind: "loading" });
  const [nonce, setNonce] = useState(0);
  const ids = c.state === "ready" ? c.ids : null;
  const order = ids ? [hint, ...ids].filter((x, i, a): x is string => !!x && ids.includes(x) && a.indexOf(x) === i) : [];
  const key = order.join(",");

  useEffect(() => {
    if (!ids) return;
    let alive = true;
    (async () => {
      let last: unknown = null;
      for (const course of key.split(",").filter(Boolean)) {
        try {
          const attempt = await getAttempt(course, attemptId, clock, tab);
          if (alive) setLoad({ kind: "ready", course, attempt });
          return;
        } catch (e) {
          last = e;
          if (!(e instanceof ApiError && (e.status === 404 || e.status === 403))) break;
        }
      }
      if (alive) setLoad({ kind: "error", error: last ?? new ApiError({ status: 404, code: "NOT_FOUND" }) });
    })();
    return () => { alive = false; };
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [key, attemptId, nonce]);

  const back = { href: "/practice", label: "Luyện đề" };
  if (c.state === "none") return <Page><PageHeader title="Làm bài" back={back} /><Panel><EmptyState title="Chưa có lớp">Vào lớp bằng mã lớp để luyện đề.</EmptyState></Panel></Page>;
  if (load.kind === "loading") return <Page><PageHeader title="Làm bài" back={back} /><Panel><Skeleton lines={6} /></Panel></Page>;
  if (load.kind === "error") {
    const gone = load.error instanceof ApiError && (load.error.status === 404 || load.error.status === 403);
    return (
      <Page>
        <PageHeader title="Làm bài" back={back} />
        {gone ? <Panel><EmptyState title="Không tìm thấy lượt luyện">Lượt này không có hoặc không phải của bạn.</EmptyState></Panel> : <ApiErrorNotice error={load.error} onRetry={() => { setLoad({ kind: "loading" }); setNonce((n) => n + 1); }} />}
      </Page>
    );
  }
  const { course, attempt } = load;
  const done = (a: PAttempt) => setLoad({ kind: "ready", course, attempt: a });
  if (attempt.status !== "IN_PROGRESS") return <PracticeResult courseId={course} attempt={attempt} />;
  if (attempt.mode === "MOCK") {
    return <MockRun key={`${attempt.id}:${nonce}`} courseId={course} initial={attempt} tab={tab} clock={clock} userId={identity?.sub ?? "anon"} onDone={done} onTakeOver={() => { setLoad({ kind: "loading" }); setNonce((n) => n + 1); }} />;
  }
  return <TopicRun key={attempt.id} courseId={course} initial={attempt} tab={tab} onDone={done} />;
}
