"use client";

import type { ExamClock } from "@/shared/lib/examClock";
import { useSession } from "@/shared/session/session";
import { MockRun } from "./MockRun";
import { PracticeResult } from "./PracticeResult";
import { TopicRun } from "./TopicRun";
import type { PAttempt } from "./practiceApi";

/** Thân màn làm bài theo trạng thái lượt: đang làm theo chủ đề → từng câu có phản hồi; đang thi thử → màn thi có đồng hồ; đã đóng → kết quả. */
export function PracticeRunBody({ course, attempt, tab, clock, nonce, onDone, onTakeOver }: { course: string; attempt: PAttempt; tab: string; clock: ExamClock; nonce: number; onDone: (a: PAttempt) => void; onTakeOver: () => void }) {
  const { identity } = useSession();
  if (attempt.status !== "IN_PROGRESS") return <PracticeResult courseId={course} attempt={attempt} />;
  if (attempt.mode === "MOCK") {
    return <MockRun key={`${attempt.id}:${nonce}`} courseId={course} initial={attempt} tab={tab} clock={clock} userId={identity?.sub ?? "anon"} onDone={onDone} onTakeOver={onTakeOver} />;
  }
  return <TopicRun key={attempt.id} courseId={course} initial={attempt} tab={tab} userId={identity?.sub ?? "anon"} onDone={onDone} />;
}
