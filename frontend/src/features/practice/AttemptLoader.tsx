"use client";

import { useSearchParams } from "next/navigation";
import { useEffect, useState } from "react";
import dynamic from "next/dynamic";
import { ApiError, ApiErrorNotice } from "@/shared/data";
import { makeExamClock } from "@/shared/lib/examClock";
import { EmptyState, Page, PageHeader, Panel } from "@/shared/ui";
import { RunShell } from "./PracticeShell";
import { getAttempt, locateAttempt, newTabId, useStudentCourse, type PAttempt } from "./practiceApi";

type Load = { kind: "loading" } | { kind: "error"; error: unknown } | { kind: "ready"; course: string; attempt: PAttempt };

// Thân màn làm bài (MockRun / TopicRun / kết quả) nặng: nạp lười. Phần TẢI LƯỢT ở file này nhập tĩnh để GET lượt chạy ngay khi trang dựng, song song với
// /me/courses — không đợi thêm một vòng tải chunk (DEF-R10S-20).
const RunBody = dynamic(() => import("./PracticeRun").then((m) => m.PracticeRunBody), { loading: RunShell });

/**
 * `/practice/[attemptId]`: tải lượt rồi dựng thân theo trạng thái. Lớp từ `?course=` (đọc ở lượt dựng đầu); thiếu / sai thì hỏi MỘT lần lớp của lượt
 * (`GET /me/practice/attempts/{id}`), không dò từng lớp. Mở lượt MOCK bằng trang này là giành quyền ghi cho tab này.
 */
export function AttemptLoader({ attemptId }: { attemptId: string }) {
  useEffect(() => { void import("./PracticeRun"); }, []); // chunk thân tải SONG SONG với GET lượt (không đợi lượt về mới bắt đầu tải)
  const params = useSearchParams();
  const [hint] = useState(() => params.get("course")); // đọc ở lượt dựng đầu: tải lượt chạy SONG SONG với /me/courses, không đợi danh sách lớp
  const c = useStudentCourse();
  const [clock] = useState(makeExamClock);
  const [tab] = useState(newTabId);
  const [load, setLoad] = useState<Load>({ kind: "loading" });
  const [nonce, setNonce] = useState(0);
  // Lớp từ ?course= (liên kết trong app luôn mang theo); thiếu hoặc sai thì hỏi MỘT lần lớp của lượt — không dò từng lớp (luật 13).
  useEffect(() => {
    let alive = true;
    (async () => {
      const gone = (e: unknown) => e instanceof ApiError && (e.status === 404 || e.status === 403);
      try {
        let course = hint;
        let attempt: PAttempt | null = null;
        if (course) {
          try {
            attempt = await getAttempt(course, attemptId, clock, tab);
          } catch (e) {
            if (!gone(e)) throw e;
          }
        }
        if (!attempt) {
          course = await locateAttempt(attemptId);
          attempt = await getAttempt(course, attemptId, clock, tab);
        }
        if (alive) setLoad({ kind: "ready", course: course!, attempt });
      } catch (e) {
        if (alive) setLoad({ kind: "error", error: e });
      }
    })();
    return () => { alive = false; };
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [hint, attemptId, nonce]);

  const back = { href: "/practice", label: "Luyện đề" };
  if (c.state === "none") return <Page kind="work"><PageHeader title="Làm bài" back={back} /><Panel><EmptyState title="Chưa có lớp" /></Panel></Page>;
  if (load.kind === "loading") return <RunShell />;
  if (load.kind === "error") {
    const gone = load.error instanceof ApiError && (load.error.status === 404 || load.error.status === 403);
    return (
      <Page kind="work">
        <PageHeader title="Làm bài" back={back} />
        {gone ? <Panel><EmptyState title="Không tìm thấy lượt luyện" /></Panel> : <ApiErrorNotice error={load.error} onRetry={() => { setLoad({ kind: "loading" }); setNonce((n) => n + 1); }} />}
      </Page>
    );
  }
  const { course, attempt } = load;
  return (
    <RunBody
      course={course}
      attempt={attempt}
      tab={tab}
      clock={clock}
      nonce={nonce}
      onDone={(a) => setLoad({ kind: "ready", course, attempt: a })}
      onTakeOver={() => { setLoad({ kind: "loading" }); setNonce((n) => n + 1); }}
    />
  );
}
