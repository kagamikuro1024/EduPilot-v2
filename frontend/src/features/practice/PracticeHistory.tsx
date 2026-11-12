"use client";

import { useQuery } from "@tanstack/react-query";
import { useCursorList } from "@/shared/data";
import { fmtWhen } from "@/features/exam/examApi";
import { ActionList, ActionRow, BarList, Button, ButtonLink, EmptyState, Page, PageHeader, PageState, Panel, Section, Skeleton, StatusText } from "@/shared/ui";
import { getWeakness, pKey, pPath, useStudentCourse, type PHistoryRow } from "./practiceApi";
import s from "./Practice.module.css";

/** Lịch sử luyện tập: nhìn ra chủ đề cần luyện tiếp (DESIGN §14.20). */
export function PracticeHistory() {
  const c = useStudentCourse();
  if (c.state === "loading") return <Page><PageHeader title="Lịch sử luyện tập" back={{ href: "/practice", label: "Luyện đề" }} /><Panel><Skeleton lines={6} /></Panel></Page>;
  if (c.state === "none") return <Page><PageHeader title="Lịch sử luyện tập" back={{ href: "/practice", label: "Luyện đề" }} /><Panel><EmptyState title="Chưa có lớp">Vào lớp bằng mã lớp để luyện đề.</EmptyState></Panel></Page>;
  return <History courseId={c.id} />;
}

function History({ courseId }: { courseId: string }) {
  const list = useCursorList<PHistoryRow>(pKey(courseId, "history"), `${pPath(courseId)}/history`, { limit: 30 });
  const weak = useQuery({ queryKey: pKey(courseId, "weakness"), queryFn: ({ signal }) => getWeakness(courseId, signal) });
  const topics = weak.data?.topics ?? [];
  return (
    <Page>
      <PageHeader title="Lịch sử luyện tập" back={{ href: "/practice", label: "Luyện đề" }} />
      <PageState
        query={list}
        isEmpty={() => list.items.length === 0}
        loading={<Panel><Skeleton lines={6} /></Panel>}
        empty={<Panel><EmptyState title="Chưa có lượt luyện nào" action={<ButtonLink href="/practice" variant="primary">Luyện đề</ButtonLink>}>Sau lượt đầu tiên, bạn sẽ thấy ở đây mình hay sai chủ đề nào.</EmptyState></Panel>}
      >
        {topics.length > 0 && (
          <Section panel title="Chủ đề bạn hay sai">
            <div className={s.weak}>
              <BarList items={topics.map((w) => ({ label: w.topic, value: Math.round((w.wrong / w.answered) * 100), tone: w.wrong / w.answered >= 0.5 ? "red" : "ink" }))} max={100} format={(v) => `${v}% sai`} />
            </div>
            <ActionList label="Tài liệu nên đọc lại">
              {topics.flatMap((w) => w.documents.map((d) => <ActionRow key={`${w.topic}:${d.document_id}`} href={`/library/${d.document_id}`} title={d.title} context={w.topic} />))}
            </ActionList>
            <div>
              <ButtonLink href={`/practice?topic=${encodeURIComponent(topics[0].topic)}`} variant="primary">Luyện {topics[0].topic}</ButtonLink>
            </div>
          </Section>
        )}

        <Section panel title="Các lượt đã làm">
          <ActionList label="Lượt luyện tập">
            {list.items.map((r) => {
              const share = r.correct === null ? null : r.correct / r.item_count;
              return (
                <ActionRow
                  key={r.id}
                  href={`/practice/${r.id}?course=${courseId}`}
                  title={`${r.mode === "MOCK" ? "Thi thử" : r.topic} · ${r.item_count} câu`}
                  context={r.status === "IN_PROGRESS" ? "Đang làm" : `Đúng ${r.correct ?? 0}/${r.item_count}`}
                  meta={
                    <>
                      {`${fmtWhen(r.started_at)}${r.status === "EXPIRED" ? " · hết giờ" : ""}`}
                      {share !== null && (
                        <>
                          {" · "}
                          <StatusText tone={share >= 0.7 ? "green" : share >= 0.5 ? "amber" : "red"}>{share >= 0.7 ? "Nắm khá chắc" : share >= 0.5 ? "Cần ôn thêm" : "Nên luyện lại"}</StatusText>
                        </>
                      )}
                    </>
                  }
                />
              );
            })}
          </ActionList>
          {list.hasNextPage && <div><Button loading={list.isFetchingNextPage} onClick={() => void list.fetchNextPage()}>Xem thêm</Button></div>}
        </Section>
      </PageState>
    </Page>
  );
}
