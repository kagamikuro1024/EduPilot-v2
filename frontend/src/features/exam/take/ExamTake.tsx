"use client";

import { useSearchParams } from "next/navigation";
import { useEffect, useRef, useState } from "react";
import { ApiError, ApiErrorNotice, useIdempotentMutation, type ApiResult } from "@/shared/data";
import { Markdown } from "@/shared/domain";
import { makeExamClock } from "@/shared/lib/examClock";
import { useSession } from "@/shared/session/session";
import { Button, EmptyState, InlineNotice, Page, PageHeader, Skeleton } from "@/shared/ui";
import { fmtClock, fmtWhen } from "../examApi";
import { IntegrityNotice } from "./IntegrityNotice";
import { TakeRunning } from "./TakeRunning";
import { getMine, getResult, isNone, isRunning, newTabId, sleep, startAttempt, type Mine, type NoAttempt, type Running, type Submitted } from "./takeApi";
import { readStoredTab } from "./useWriter";
import s from "./Take.module.css";

type Load = { kind: "loading" } | { kind: "error"; error: unknown } | { kind: "ready"; course: string; mine: Mine; prevWriter: boolean; startedHere: boolean };
const hhmmss = (iso: string) => new Date(Date.parse(iso) + 7 * 3600_000).toISOString().slice(11, 19);

/**
 * `/exams/[id]/take` (sinh viên): theo trạng thái — chưa bắt đầu → màn giới thiệu; đang làm → màn làm bài; đã nộp → tóm tắt (không đề, không đúng / sai).
 * Lớp của bài không có trong đường dẫn: thử lớp gợi ý (`?course=`), lớp đang chọn rồi các lớp sinh viên còn lại, lấy lớp đầu tiên trả về bài.
 */
export function ExamTake({ id }: { id: string }) {
  const hint = useSearchParams().get("course");
  const { realCourses, realCourseId, identity } = useSession();
  const [clock] = useState(makeExamClock);
  const [tab] = useState(newTabId); // mã tab: sinh mỗi lần tải trang, chỉ ở bộ nhớ của trang
  const [load, setLoad] = useState<Load>({ kind: "loading" });
  const stored = useRef<string | null>(null);

  const courses = (realCourses ?? []).filter((c) => c.role_in_course === "STUDENT").map((c) => c.id);
  const order = [hint, realCourseId && realCourseId !== "all" ? realCourseId : null, ...courses].filter((c, i, a): c is string => !!c && courses.includes(c) && a.indexOf(c) === i);
  const orderKey = order.join(",");

  useEffect(() => {
    if (!realCourses) return;
    let alive = true;
    stored.current = readStoredTab(id);
    (async () => {
      let last: unknown = null;
      for (const c of orderKey.split(",").filter(Boolean)) {
        try {
          // lần đầu hỏi bằng id tab đã nhớ: `writer.is_you` cho biết người ghi cũ có phải chính trang vừa tải lại không (SRS 4.3.5)
          const mine = await getMine(clock, c, id, stored.current ?? undefined);
          if (alive) setLoad({ kind: "ready", course: c, mine, prevWriter: isRunning(mine) && mine.attempt.writer.is_you, startedHere: false });
          return;
        } catch (e) {
          last = e;
          if (!(e instanceof ApiError && (e.status === 404 || e.status === 403))) break;
        }
      }
      if (alive) setLoad({ kind: "error", error: last ?? new ApiError({ status: 404, code: "NOT_FOUND" }) });
    })();
    return () => {
      alive = false;
    };
  }, [realCourses, orderKey, id, clock]);

  /** Hỏi lại máy chủ (sau khi nộp / hết giờ): lặp tới khi `until` đúng. */
  async function refresh(course: string, until: (m: Mine) => boolean) {
    for (let n = 0; n < 15; n++) {
      try {
        const mine = await getMine(clock, course, id, tab);
        if (until(mine)) {
          setLoad({ kind: "ready", course, mine, prevWriter: false, startedHere: false });
          return;
        }
      } catch {
        /* mạng chập chờn: thử lại */
      }
      await sleep(2000);
    }
  }

  if (load.kind === "loading") return <Page><PageHeader title="Bài thi" back={{ href: "/exams", label: "Bài thi" }} /><Skeleton lines={5} /></Page>;
  if (load.kind === "error") {
    const gone = load.error instanceof ApiError && (load.error.status === 404 || load.error.status === 403);
    return (
      <Page>
        <PageHeader title="Bài thi" back={{ href: "/exams", label: "Bài thi" }} />
        {gone ? <EmptyState title="Không tìm thấy bài thi">Bài thi không có hoặc không thuộc lớp của bạn.</EmptyState> : <ApiErrorNotice error={load.error} onRetry={() => setLoad({ kind: "loading" })} />}
      </Page>
    );
  }
  const { course, mine } = load;
  if (isNone(mine)) {
    return <Intro mine={mine} onStarted={(r) => setLoad({ kind: "ready", course, mine: r, prevWriter: false, startedHere: true })} start={(key) => startAttempt(clock, course, id, tab, key)} />;
  }
  if (isRunning(mine)) {
    return (
      <TakeRunning
        key={mine.attempt.id}
        courseId={course}
        examId={id}
        initial={mine}
        tab={tab}
        clock={clock}
        userId={identity?.sub ?? "anon"}
        prevWriter={load.prevWriter}
        startedHere={load.startedHere}
        onFinished={() => void refresh(course, (m) => !isRunning(m))}
        onExpired={() => void refresh(course, (m) => !isRunning(m))}
      />
    );
  }
  return <Done course={course} id={id} mine={mine as Submitted} />;
}

function Intro({ mine, start, onStarted }: { mine: NoAttempt; start: (key: string) => Promise<ApiResult<Running>>; onStarted: (r: Running) => void }) {
  const e = mine.exam;
  const run = useIdempotentMutation<void, Running>((_v, key) => start(key));
  const [nowMs] = useState(() => Date.now()); // một lần khi mở màn giới thiệu
  const [seen, setSeen] = useState(false); // đã thấy câu minh bạch (AC6): nút Bắt đầu chỉ bấm được sau đó
  const minutesLeft = e.closes_at ? Math.max(0, Math.floor((Date.parse(e.closes_at) - nowMs) / 60_000)) : null;
  const short = e.status === "OPEN" && e.duration_minutes && minutesLeft !== null && minutesLeft < e.duration_minutes;
  return (
    <Page>
      <PageHeader title={e.title} back={{ href: "/exams", label: "Bài thi" }} />
      <div className={s.intro}>
        {e.instructions && <Markdown source={e.instructions} />}
        {e.status === "OPEN" && (
          <>
            <p className={s.introNote}>Bạn có {e.duration_minutes} phút. Đồng hồ chạy ngay khi bạn bấm Bắt đầu và không dừng lại nếu bạn thoát.</p>
            {e.kind !== "MCQ" && <p className={s.introNote}>Bài này có phần lập trình, cần màn hình ≥ 1024 px.</p>}
            {short && e.closes_at && <p className={s.introNote}>Bài thi đóng lúc {fmtClock(e.closes_at).slice(0, 5)}, bạn chỉ còn {minutesLeft} phút.</p>}
            <IntegrityNotice onSeen={() => setSeen(true)} />
            {run.error && <ApiErrorNotice error={run.error} onRetry={() => void run.retry().then((r) => r && onStarted(r.data), () => undefined)} />}
            <div>
              <Button variant="primary" loading={run.pending} disabled={!seen} onClick={() => void run.mutate().then((r) => onStarted(r.data), () => undefined)}>Bắt đầu làm bài</Button>
            </div>
          </>
        )}
        {e.status === "SCHEDULED" && <InlineNotice>Bài thi mở lúc {e.opens_at ? fmtWhen(e.opens_at) : "—"}. Bạn có {e.duration_minutes} phút để làm.</InlineNotice>}
        {(e.status === "CLOSED" || e.status === "PUBLISHED") && <InlineNotice>Bạn không làm bài này.</InlineNotice>}
      </div>
    </Page>
  );
}

function Done({ course, id, mine }: { course: string; id: string; mine: Submitted }) {
  const a = mine.attempt;
  const [score, setScore] = useState<string | null>(null);
  const published = mine.exam.status === "PUBLISHED";
  useEffect(() => {
    if (!published) return;
    void getResult(course, id, a.id).then((r) => setScore(r.score ? `${r.score.replace(".", ",")} / ${r.exam.max_score.replace(".", ",")}` : null), () => undefined);
  }, [published, course, id, a.id]);
  const when = hhmmss(a.submitted_at);
  return (
    <Page>
      <PageHeader title={mine.exam.title} back={{ href: "/exams", label: "Bài thi" }} />
      <div className={s.done} data-part="submitted">
        <InlineNotice tone="success" title={a.submit_reason === "MANUAL" ? `Đã nộp lúc ${when}.` : `Hết giờ — bài của bạn đã được nộp lúc ${when}.`}>
          {published
            ? score ? `Điểm của bạn: ${score}.` : "Điểm đã được công bố."
            : mine.exam.status === "CLOSED"
              ? "Điểm đang được chấm."
              : `Điểm sẽ hiện khi bài thi đóng với cả lớp${mine.exam.closes_at ? ` (${fmtWhen(mine.exam.closes_at)})` : ""}.`}
        </InlineNotice>
      </div>
    </Page>
  );
}
