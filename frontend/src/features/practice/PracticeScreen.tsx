"use client";

import { useRouter } from "next/navigation";
import { useState } from "react";
import { agoLabel } from "@/mock/derive";
import {
  ATTEMPTS,
  HISTORY_KEY,
  HISTORY_SEED,
  quizKey,
  QUIZ_SEED,
  RUN_KEY,
  RUN_SEED,
  SEED_RUN,
  SEED_STUDENT,
  historyRows,
  newRun,
  weakTopics,
  type QuizState,
  type Run,
  type RunRecord,
} from "@/mock/practice";
import { ASSIGNMENTS, until } from "@/mock/student";
import { DOCS } from "@/mock/library";
import { useSession } from "@/shared/session/session";
import { simNowMs, useSimNow } from "@/shared/state/clock";
import { useDemoSlice } from "@/shared/state/demo";
import {
  ActionList,
  ActionRow,
  Button,
  ButtonLink,
  EmptyState,
  Page,
  PageHeader,
  PageState,
  Section,
  Skeleton,
  StatusText,
  Tabs,
} from "@/shared/ui";
import s from "./Practice.module.css";

// Prototype: mọi chủ đề đều mở lượt luyện Mật mã đối xứng (chỉ chủ đề này có ngân hàng câu đầy đủ).
const TOPICS = [
  { topic: "Mật mã đối xứng", count: 10 },
  { topic: "Hàm băm và chữ ký số", count: 6 },
  { topic: "An toàn ứng dụng web", count: 8 },
  { topic: "Tường lửa và phân đoạn mạng", count: 8 },
];

const SYMMETRIC = ATTEMPTS[0];

/** Luyện đề: chọn cách luyện có ích nhất lúc này (DESIGN §14.18). */
export function PracticeScreen() {
  const router = useRouter();
  const { studentId } = useSession();
  const [mode, setMode] = useState<"topic" | "exam">("topic");
  const [quiz] = useDemoSlice<QuizState>(quizKey(studentId), QUIZ_SEED);
  const [run, setRun] = useDemoSlice<Run | null>(`${RUN_KEY}.${studentId}`, RUN_SEED);
  const [done] = useDemoSlice<RunRecord[]>(`${HISTORY_KEY}.${studentId}`, HISTORY_SEED);
  const now = useSimNow();
  // Lịch sử, chủ đề yếu và lượt dở seed chỉ thuộc về sinh viên B (FR-X18).
  const seeded = studentId === SEED_STUDENT;
  const weak = weakTopics(done, seeded);
  const weakest = weak[0];
  const rows = historyRows(done, seeded);
  const quiz01 = ASSIGNMENTS[3];
  const papers = DOCS.filter((d) => d.kind === "exam");
  const pending = run ? (run.finishedMs ? null : run) : seeded ? SEED_RUN : null;
  const pendingAt = pending ? pending.answers.findIndex((a) => a === null) : -1;

  function startFresh() {
    setRun(newRun(SYMMETRIC, simNowMs()));
    router.push(`/practice/${SYMMETRIC.id}`);
  }

  function resume() {
    if (!run && pending) setRun(pending);
    router.push(`/practice/${pending?.attemptId ?? SYMMETRIC.id}`);
  }

  return (
    <Page>
      <PageHeader
        title="Luyện đề"
        description="Luyện theo chủ đề bạn hay sai, hoặc thi thử với đề cũ."
        actions={<ButtonLink href="/practice/history">Lịch sử luyện tập</ButtonLink>}
      />
      <PageState
        loading={<Skeleton lines={6} />}
        empty={
          <EmptyState title="Bạn chưa luyện lần nào" action={<ButtonLink href="/practice/at-symmetric" variant="primary">Luyện 10 câu đầu tiên</ButtonLink>}>
            Mỗi lượt khoảng 15 phút. Sau lượt đầu, hệ thống sẽ gợi ý chủ đề bạn cần ôn thêm.
          </EmptyState>
        }
      >
        {weakest && (
          <Section title="Nên luyện ngay">
            <ActionList label="Khuyến nghị luyện tập">
              <ActionRow
                tone="red"
                href="/practice/at-symmetric"
                redThread
                title={`Ôn lại ${weakest.topic}`}
                context={`Bạn sai ${weakest.wrong}/${weakest.total} câu ở các lượt gần nhất`}
                meta="10 câu · khoảng 15 phút · có giải thích ngay sau mỗi câu"
                action={
                  <Button variant="primary" onClick={startFresh}>
                    Luyện 10 câu
                  </Button>
                }
              />
            </ActionList>
          </Section>
        )}

        <Section title="Chọn cách luyện">
          <Tabs
            label="Cách luyện"
            value={mode}
            onChange={setMode}
            options={[
              { value: "topic", label: "Theo chủ đề", count: TOPICS.length },
              { value: "exam", label: "Thi thử", count: papers.length + 1 },
            ]}
          />
          {mode === "topic" ? (
            <ActionList label="Chủ đề luyện tập">
              {TOPICS.map((t) => {
                const w = weak.find((x) => x.topic === t.topic);
                return (
                  <ActionRow
                    key={t.topic}
                    onSelect={startFresh}
                    title={t.topic}
                    context={w ? `Bạn sai ${w.wrong}/${w.total} câu đã làm` : "Bạn chưa luyện chủ đề này"}
                    meta={`${t.count} câu · phản hồi ngay sau mỗi câu`}
                  />
                );
              })}
            </ActionList>
          ) : (
            <ActionList label="Thi thử">
              <ActionRow
                tone="amber"
                href="/practice/at-quiz01"
                title={`${quiz01.code} — ${quiz01.title}`}
                context={quiz.status === "submitted" ? "Bạn đã nộp bài" : `Bài tính điểm · đóng sau ${until(quiz01.due)}`}
                meta={`${quiz01.minutes} phút · không có phản hồi trong lúc làm`}
                action={quiz.status === "submitted" ? <StatusText tone="green">Đã nộp</StatusText> : undefined}
              />
              {papers.map((p) => (
                <ActionRow key={p.id} href={`/practice/${p.practiceAttemptId ?? SYMMETRIC.id}`} title={p.title} context={p.summary} meta={`Đề cũ · ${p.pages} trang`} />
              ))}
            </ActionList>
          )}
        </Section>

        {(pending || rows.length > 0) && (
          <Section title={pending ? "Lượt đang dở" : "Lượt gần đây"}>
            <ActionList label={pending ? "Lượt đang dở" : "Lượt gần đây"}>
              {pending && (
                <ActionRow
                  title={pending.title}
                  context={`Dừng ở câu ${pendingAt < 0 ? pending.qids.length : pendingAt + 1}/${pending.qids.length}`}
                  meta={agoLabel(pending.startedMs, now)}
                  action={<Button onClick={resume}>Tiếp tục</Button>}
                />
              )}
              {rows.slice(0, 2).map((r) => (
                <ActionRow
                  key={r.id}
                  href={`/practice/${r.id}`}
                  title={`${r.topic} · ${r.total} câu`}
                  context={`${r.score}/${r.total}`}
                  meta={`${agoLabel(r.atMs, now)} · xem lại`}
                />
              ))}
            </ActionList>
          </Section>
        )}
      </PageState>
      <p className={s.footNote}>Kết quả luyện đề không tính vào điểm quá trình, trừ các bài có ghi “tính điểm”.</p>
    </Page>
  );
}
