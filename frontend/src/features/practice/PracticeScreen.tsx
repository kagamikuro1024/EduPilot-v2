"use client";

import { useState } from "react";
import { ago, at } from "@/mock/core";
import { HISTORY, IN_PROGRESS, WEAK_TOPICS } from "@/mock/practice";
import { QUIZ_KEY, QUIZ_SEED, type QuizState } from "@/mock/practice";
import { ASSIGNMENTS, until } from "@/mock/student";
import { DOCS } from "@/mock/library";
import { useDemoSlice } from "@/shared/state/demo";
import {
  ActionList,
  ActionRow,
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

const TOPICS = [
  { topic: "Mật mã đối xứng", attemptId: "at-symmetric", count: 10 },
  { topic: "Hàm băm và chữ ký số", attemptId: "at-symmetric", count: 6 },
  { topic: "An toàn ứng dụng web", attemptId: "at-symmetric", count: 8 },
  { topic: "Tường lửa và phân đoạn mạng", attemptId: "at-symmetric", count: 8 },
];

/** Luyện đề: chọn cách luyện có ích nhất lúc này (DESIGN §14.18). */
export function PracticeScreen() {
  const [mode, setMode] = useState<"topic" | "exam">("topic");
  const [quiz] = useDemoSlice<QuizState>(QUIZ_KEY, QUIZ_SEED);
  const weakest = WEAK_TOPICS[0];
  const quiz01 = ASSIGNMENTS[3];
  const papers = DOCS.filter((d) => d.kind === "exam");

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
                <ButtonLink href="/practice/at-symmetric" variant="primary">
                  Luyện 10 câu
                </ButtonLink>
              }
            />
          </ActionList>
        </Section>

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
                const w = WEAK_TOPICS.find((x) => x.topic === t.topic);
                return (
                  <ActionRow
                    key={t.topic}
                    href={`/practice/${t.attemptId}`}
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
                <ActionRow key={p.id} href="/practice/at-symmetric" title={p.title} context={p.summary} meta={`Đề cũ · ${p.pages} trang`} />
              ))}
            </ActionList>
          )}
        </Section>

        <Section title="Lượt đang dở">
          <ActionList label="Lượt đang dở">
            <ActionRow
              href={`/practice/${IN_PROGRESS.id}`}
              title={IN_PROGRESS.title}
              context={`Dừng ở câu ${IN_PROGRESS.at}/${IN_PROGRESS.total}`}
              meta={ago(at(-IN_PROGRESS.minsAgo))}
            />
            {HISTORY.slice(0, 2).map((h) => (
              <ActionRow
                key={h.id}
                href={`/practice/${h.id}`}
                title={h.title}
                context={`Đúng ${h.score}/${h.questions.length} câu`}
                meta={`${ago(at(-(h.minsAgo ?? 0)))} · xem lại`}
              />
            ))}
          </ActionList>
        </Section>
      </PageState>
      <p className={s.footNote}>Kết quả luyện đề không tính vào điểm quá trình, trừ các bài có ghi “tính điểm”.</p>
    </Page>
  );
}
