"use client";

import { useQuery } from "@tanstack/react-query";
import { useRouter, useSearchParams } from "next/navigation";
import { useState } from "react";
import { ApiError, ApiErrorNotice, useCursorList, useIdempotentMutation } from "@/shared/data";
import { fmtWhen } from "@/features/exam/examApi";
import { ActionList, ActionRow, Button, ButtonLink, EmptyState, Field, Input, Page, PageHeader, PageState, Panel, Section, Skeleton, Tabs } from "@/shared/ui";
import { getOverview, MIN_COUNT, newTabId, pKey, pPath, startPractice, useStudentCourse, type Mode, type PAttempt, type PHistoryRow, type POverview } from "./practiceApi";
import s from "./Practice.module.css";

const href = (id: string, course: string) => `/practice/${id}?course=${course}`;

/** Luyện đề: chọn cách luyện có ích nhất lúc này (DESIGN §14.18). */
export function PracticeHome() {
  const c = useStudentCourse();
  if (c.state === "loading") return <Page><PageHeader title="Luyện đề" /><Panel><Skeleton lines={6} /></Panel></Page>;
  if (c.state === "none") return <Page><PageHeader title="Luyện đề" /><Panel><EmptyState title="Chưa có lớp">Vào lớp bằng mã lớp để luyện đề.</EmptyState></Panel></Page>;
  return <Home courseId={c.id} />;
}

function Home({ courseId }: { courseId: string }) {
  const router = useRouter();
  const want = useSearchParams().get("topic");
  const [mode, setMode] = useState<"topic" | "mock">("topic");
  const [count, setCount] = useState("20");
  const [minutes, setMinutes] = useState("30");
  const [tab] = useState(newTabId);
  const ov = useQuery({ queryKey: pKey(courseId, "overview"), queryFn: ({ signal }) => getOverview(courseId, signal) });
  const recent = useCursorList<PHistoryRow>(pKey(courseId, "history", "recent"), `${pPath(courseId)}/history`, { limit: 3 });
  const start = useIdempotentMutation<{ mode: Mode; topic?: string; count?: number; duration_minutes?: number }, PAttempt>((body, key) => startPractice(courseId, body, tab, key));
  const [starting, setStarting] = useState<string | null>(null);
  const go = (label: string, body: Parameters<typeof start.mutate>[0]) => {
    setStarting(label);
    start.mutate(body).then((r) => router.push(href(r.data.id, courseId)), () => setStarting(null));
  };
  const total = (ov.data?.topics ?? []).reduce((n, t) => n + t.available, 0);
  const rec = ov.data && recommendation(ov.data, want);
  const open = ov.data?.in_progress ?? [];
  const noQuestions = ov.data && ov.data.topics.length === 0;
  const countN = Number(count);
  const minN = Number(minutes);
  const mockBad = !Number.isInteger(countN) || countN < MIN_COUNT || countN > 50 || !Number.isInteger(minN) || minN < 10 || minN > 180;

  return (
    <Page>
      <PageHeader title="Luyện đề" actions={<ButtonLink href="/practice/history">Lịch sử luyện tập</ButtonLink>} />
      <PageState
        query={ov}
        isEmpty={() => Boolean(noQuestions)}
        loading={<Panel><Skeleton lines={6} /></Panel>}
        empty={<Panel><EmptyState title="Giảng viên chưa mở câu hỏi" /></Panel>}
      >
        {open.length > 0 && (
          <Section panel title="Lượt đang dở">
            <ActionList label="Lượt đang dở">
              {open.map((o) => (
                <ActionRow
                  key={o.id}
                  tone="amber"
                  title={o.mode === "MOCK" ? "Thi thử" : o.topic}
                  context={`Dừng ở câu ${Math.min(o.current_index + 1, o.item_count)}/${o.item_count}`}
                  meta={o.deadline_at ? `Hết giờ lúc ${fmtWhen(o.deadline_at)}` : undefined}
                  action={<ButtonLink href={href(o.id, courseId)} variant="primary">Tiếp tục</ButtonLink>}
                />
              ))}
            </ActionList>
          </Section>
        )}

        {rec && (
          <Section panel title="Nên luyện ngay">
            <ActionList label="Khuyến nghị luyện tập">
              <ActionRow
                tone="red"
                redThread
                title={`Luyện ${rec.topic}`}
                meta={`${rec.count} câu · phản hồi ngay sau mỗi câu`}
                action={<Button variant="primary" loading={starting === "rec"} disabled={starting !== null} onClick={() => go("rec", { mode: "TOPIC", topic: rec.topic, count: rec.count })}>Luyện {rec.count} câu</Button>}
              />
            </ActionList>
          </Section>
        )}

        <Section panel title="Chọn cách luyện">
          <Tabs label="Cách luyện" value={mode} onChange={setMode} options={[{ value: "topic", label: "Theo chủ đề", count: ov.data?.topics.length ?? 0 }, { value: "mock", label: "Thi thử" }]} />
          {mode === "topic" ? (
            <ActionList label="Chủ đề luyện tập">
              {(ov.data?.topics ?? []).map((t) => {
                const enough = t.available >= MIN_COUNT;
                const n = Math.min(10, t.available);
                return (
                  <ActionRow
                    key={t.topic}
                    title={t.topic}
                    context={`${t.available} câu`}
                    action={<Button disabled={!enough || starting !== null} loading={starting === t.topic} title={enough ? undefined : `Cần ít nhất ${MIN_COUNT} câu`} onClick={() => go(t.topic, { mode: "TOPIC", topic: t.topic, count: n })}>Luyện {n} câu</Button>}
                  />
                );
              })}
            </ActionList>
          ) : (
            <form className={s.mockForm} onSubmit={(e) => { e.preventDefault(); go("mock", { mode: "MOCK", count: countN, duration_minutes: minN }); }} noValidate aria-label="Thi thử">
              <Field label="Số câu">{(id, by) => <Input id={id} aria-describedby={by} type="number" inputMode="numeric" min={MIN_COUNT} max={50} value={count} placeholder="20" onChange={(e) => setCount(e.target.value)} />}</Field>
              <Field label="Số phút">{(id, by) => <Input id={id} aria-describedby={by} type="number" inputMode="numeric" min={10} max={180} value={minutes} placeholder="30" onChange={(e) => setMinutes(e.target.value)} />}</Field>
              <Button type="submit" variant="primary" loading={starting === "mock"} disabled={mockBad || total < MIN_COUNT || starting !== null} title={total < MIN_COUNT ? `Lớp chưa đủ ${MIN_COUNT} câu` : undefined}>Bắt đầu thi thử</Button>
            </form>
          )}
          {start.error && <StartError error={start.error} />}
        </Section>

        {(recent.items.length > 0) && (
          <Section panel title="Lượt gần đây">
            <ActionList label="Lượt gần đây">
              {recent.items.map((r) => (
                <ActionRow
                  key={r.id}
                  href={href(r.id, courseId)}
                  title={r.mode === "MOCK" ? "Thi thử" : r.topic}
                  context={r.status === "IN_PROGRESS" ? "Đang làm" : `Đúng ${r.correct ?? 0}/${r.item_count}`}
                  meta={fmtWhen(r.started_at)}
                />
              ))}
            </ActionList>
          </Section>
        )}
      </PageState>
    </Page>
  );
}

function StartError({ error }: { error: ApiError }) {
  if (error.code === "NOT_ENOUGH_QUESTIONS") return <p className={s.hint} role="alert">Chủ đề này chưa đủ câu để luyện.</p>;
  return <ApiErrorNotice error={error} />;
}

/** Chủ đề khuyến nghị: `?topic=` (từ Hôm nay) nếu có câu đủ để luyện, nếu không thì khuyến nghị của máy chủ. */
function recommendation(ov: POverview, want: string | null): { topic: string; count: number } | null {
  const t = want ? ov.topics.find((x) => x.topic === want && x.available >= MIN_COUNT) : undefined;
  if (t) return { topic: t.topic, count: Math.min(10, t.available) };
  return ov.recommended && ov.recommended.count >= MIN_COUNT ? ov.recommended : null;
}
