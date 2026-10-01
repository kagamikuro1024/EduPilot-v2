"use client";

import { HISTORY, WEAK_TOPICS } from "@/mock/practice";
import { ago, at } from "@/mock/core";
import {
  ActionList,
  ActionRow,
  BarList,
  ButtonLink,
  EmptyState,
  Page,
  PageHeader,
  PageState,
  Section,
  Skeleton,
  StatusText,
} from "@/shared/ui";
import s from "./Practice.module.css";

/** Lịch sử luyện tập: nhìn ra chủ đề cần luyện tiếp (DESIGN §14.20). */
export function PracticeHistory() {
  return (
    <Page>
      <PageHeader
        title="Lịch sử luyện tập"
        back={{ href: "/practice", label: "Luyện đề" }}
        description={`${HISTORY.length} lượt gần đây. Bấm một lượt để xem lại từng câu và giải thích.`}
      />
      <PageState
        loading={<Skeleton lines={6} />}
        empty={
          <EmptyState title="Chưa có lượt luyện nào" action={<ButtonLink href="/practice/at-symmetric" variant="primary">Luyện 10 câu</ButtonLink>}>
            Sau lượt đầu tiên, bạn sẽ thấy ở đây mình hay sai chủ đề nào.
          </EmptyState>
        }
      >
        <Section title="Các lượt đã làm">
          <ActionList label="Lượt luyện tập">
            {HISTORY.map((h) => {
              const total = h.questions.length;
              const score = h.score ?? 0;
              return (
                <ActionRow
                  key={h.id}
                  href={`/practice/${h.id}`}
                  title={h.title}
                  context={`Đúng ${score}/${total} câu`}
                  meta={
                    <>
                      {`${ago(at(-(h.minsAgo ?? 0)))} · `}
                      <StatusText tone={score / total >= 0.7 ? "green" : score / total >= 0.5 ? "amber" : "red"}>
                        {score / total >= 0.7 ? "Nắm khá chắc" : score / total >= 0.5 ? "Cần ôn thêm" : "Nên luyện lại chủ đề này"}
                      </StatusText>
                    </>
                  }
                />
              );
            })}
          </ActionList>
        </Section>

        <Section title="Chủ đề bạn hay sai" description="Tính trên các câu đã làm trong 6 lượt gần nhất.">
          <div className={s.weak}>
            <BarList
              items={WEAK_TOPICS.map((w) => ({ label: w.topic, value: Math.round((w.wrong / w.total) * 100), tone: w.wrong / w.total >= 0.5 ? "red" : "ink" }))}
              max={100}
              format={(v) => `${v}% sai`}
            />
          </div>
          <div>
            <ButtonLink href="/practice/at-symmetric" variant="primary">
              Luyện {WEAK_TOPICS[0].topic}
            </ButtonLink>
          </div>
        </Section>
      </PageState>
    </Page>
  );
}
