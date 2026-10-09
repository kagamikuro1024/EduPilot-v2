"use client";

import { HISTORY_KEY, HISTORY_SEED, SEED_STUDENT, historyRows, weakTopics, type RunRecord } from "@/mock/practice";
import { agoLabel } from "@/mock/derive";
import { useSession } from "@/shared/session/session";
import { useSimNow } from "@/shared/state/clock";
import { useDemoSlice } from "@/shared/state/demo";
import { ActionList, ActionRow, BarList, ButtonLink, EmptyState, Page, PageHeader, PageState, Panel, Section, Skeleton, StatusText } from "@/shared/ui";
import s from "./Practice.module.css";

/** Lịch sử luyện tập: nhìn ra chủ đề cần luyện tiếp (DESIGN §14.20). */
export function PracticeHistory() {
  const { studentId } = useSession();
  const [done] = useDemoSlice<RunRecord[]>(`${HISTORY_KEY}.${studentId}`, HISTORY_SEED);
  const now = useSimNow();
  const rows = historyRows(done, studentId === SEED_STUDENT);
  const weak = weakTopics(done, studentId === SEED_STUDENT);
  return (
    <Page>
      <PageHeader
        title="Lịch sử luyện tập"
        back={{ href: "/practice", label: "Luyện đề" }}
        description={rows.length > 0 ? `${rows.length} lượt gần đây. Bấm một lượt để xem lại từng câu và giải thích.` : "Chưa có lượt luyện nào."}
      />
      <PageState
        state={rows.length === 0 ? "empty" : undefined}
        loading={<Skeleton lines={6} />}
        empty={
          <Panel><EmptyState title="Chưa có lượt luyện nào" action={<ButtonLink href="/practice/at-symmetric" variant="primary">Luyện 10 câu</ButtonLink>}>
            Sau lượt đầu tiên, bạn sẽ thấy ở đây mình hay sai chủ đề nào.
          </EmptyState></Panel>
        }
      >
        <Section panel title="Các lượt đã làm">
          <ActionList label="Lượt luyện tập">
            {rows.map((r) => {
              const share = r.score / r.total;
              return (
                <ActionRow
                  key={r.id}
                  href={`/practice/${r.id}`}
                  title={`${r.topic} · ${r.total} câu`}
                  context={`${r.score}/${r.total}`}
                  meta={
                    <>
                      {`${agoLabel(r.atMs, now)} · `}
                      <StatusText tone={share >= 0.7 ? "green" : share >= 0.5 ? "amber" : "red"}>
                        {share >= 0.7 ? "Nắm khá chắc" : share >= 0.5 ? "Cần ôn thêm" : "Nên luyện lại chủ đề này"}
                      </StatusText>
                    </>
                  }
                />
              );
            })}
          </ActionList>
        </Section>

        {weak.length > 0 && (
          <Section panel title="Chủ đề bạn hay sai" description="Tính trên các câu đã làm trong các lượt gần nhất.">
            <div className={s.weak}>
              <BarList
                items={weak.map((w) => ({ label: w.topic, value: Math.round((w.wrong / w.total) * 100), tone: w.wrong / w.total >= 0.5 ? "red" : "ink" }))}
                max={100}
                format={(v) => `${v}% sai`}
              />
            </div>
            <div>
              <ButtonLink href="/practice/at-symmetric" variant="primary">
                Luyện {weak[0].topic}
              </ButtonLink>
            </div>
          </Section>
        )}
      </PageState>
    </Page>
  );
}
