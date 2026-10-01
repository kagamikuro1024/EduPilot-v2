"use client";

import { Pin } from "lucide-react";
import { useRouter } from "next/navigation";
import { useState } from "react";
import { scanPersonal } from "@/mock/chat";
import { ago, at } from "@/mock/core";
import { KEYS, type InsightThread } from "@/mock/state";
import { NEW_THREADS_KEY, THREADS, THREAD_TOPICS, type NewThread, type Thread } from "@/mock/threads";
import { CHAT_DRAFT_KEY } from "@/features/chat/ChatScreen";
import { useSession } from "@/shared/session/session";
import { useDemoSlice } from "@/shared/state/demo";
import {
  ActionList,
  ActionRow,
  ButtonLink,
  Composer,
  EmptyState,
  FilterChips,
  Input,
  Page,
  PageHeader,
  PageState,
  Section,
  Skeleton,
  StatusText,
  Toolbar,
} from "@/shared/ui";
import { PIIChannelDialog } from "./PIIChannelDialog";
import s from "./Threads.module.css";

const ANSWER_LABEL = {
  verified: { tone: "green", text: "Đã được giảng viên xác nhận" },
  pending: { tone: "amber", text: "Chờ xác nhận" },
  none: { tone: "neutral", text: "Chưa có câu trả lời" },
} as const;

/** Threads: đọc và đặt câu hỏi công khai của lớp (DESIGN §14.3). */
export function ThreadsScreen() {
  const { role, course } = useSession();
  const router = useRouter();
  const [posts, setPosts] = useDemoSlice<NewThread[]>(NEW_THREADS_KEY, []);
  const [pinned] = useDemoSlice<InsightThread[]>(KEYS.insightThreads, []);
  const [, setChatDraft] = useDemoSlice<string>(CHAT_DRAFT_KEY, "");
  const [draft, setDraft] = useState("");
  const [guard, setGuard] = useState<{ text: string; reasons: string[] } | null>(null);
  const [query, setQuery] = useState("");
  const [topics, setTopics] = useState<string[]>([]);

  const coursePinned = pinned.filter((p) => p.courseId === course.id);

  function publish(text: string) {
    const id = `t-new-${Date.now()}`;
    const title = text.length > 80 ? `${text.slice(0, 80).trimEnd()}…` : text;
    setPosts((prev) => [{ id, title, body: text, topic: "Câu hỏi của bạn", answered: false }, ...prev]);
    setDraft("");
    setGuard(null);
    window.setTimeout(() => setPosts((prev) => prev.map((p) => (p.id === id ? { ...p, answered: true } : p))), 2000);
  }

  function submit() {
    const text = draft.trim();
    if (!text) return;
    const found = scanPersonal(text);
    if (found.count > 0) setGuard({ text, reasons: found.reasons });
    else publish(text);
  }

  const rows = THREADS.filter((t) => t.courseId === course.id)
    .filter((t) => (topics.length === 0 ? true : topics.includes(t.topic)))
    .filter((t) => (query.trim() ? t.title.toLowerCase().includes(query.trim().toLowerCase()) : true))
    .sort((a, b) => Number(Boolean(b.pinned)) - Number(Boolean(a.pinned)) || a.minsAgo - b.minsAgo);

  return (
    <Page>
      <PageHeader
        title="Threads"
        description={`Câu hỏi công khai của ${course.label}. Câu trả lời có nhãn xác nhận là đã được giảng viên duyệt.`}
        meta={`${rows.length + coursePinned.length + posts.length} thread · tuần ${course.week}`}
      />

      {role === "student" && (
        <Section title="Đặt câu hỏi" description="Hỏi về nội dung môn học. Đừng ghi mã số sinh viên hay điểm cá nhân — chuyện riêng hỏi ở Chat riêng.">
          <Composer
            value={draft}
            onChange={setDraft}
            onSubmit={submit}
            submitLabel="Đăng câu hỏi"
            label="Nội dung câu hỏi"
            placeholder="Ví dụ: Vì sao CBC cần véc-tơ khởi tạo ngẫu nhiên?"
          />
        </Section>
      )}

      <Section title="Câu hỏi của lớp">
        <Toolbar end={<FilterChips label="Lọc theo chủ đề" value={topics} onChange={setTopics} options={THREAD_TOPICS.map((t) => ({ value: t, label: t }))} />}>
          <Input
            className={s.search}
            type="search"
            value={query}
            placeholder="Tìm trong tiêu đề thread"
            aria-label="Tìm thread"
            onChange={(e) => setQuery(e.target.value)}
          />
        </Toolbar>

        <PageState
          loading={<Skeleton lines={6} />}
          empty={
            <EmptyState title="Chưa có câu hỏi nào trong tuần này" action={<ButtonLink href="/threads" variant="primary">Đặt câu hỏi</ButtonLink>}>
              Khi bạn hoặc bạn cùng lớp đặt câu hỏi, thread sẽ hiện ở đây kèm trạng thái xác nhận của giảng viên.
            </EmptyState>
          }
        >
          {rows.length === 0 && posts.length === 0 && coursePinned.length === 0 ? (
            <EmptyState title="Không có thread nào khớp bộ lọc" action={<ButtonLink href="/threads">Bỏ bộ lọc</ButtonLink>}>
              Thử bỏ bớt chủ đề hoặc tìm bằng từ khoá khác.
            </EmptyState>
          ) : (
            <ActionList label="Thread của lớp">
              {coursePinned.map((p) => (
                <ActionRow
                  key={p.id}
                  lead={<Pin className={s.pin} aria-hidden />}
                  href={`/threads/${p.id}`}
                  title={p.title}
                  context={p.body}
                  meta={`Ghim · ${p.topic} · giảng viên tạo từ Insights`}
                />
              ))}

              {rows
                .filter((t) => t.pinned)
                .map((t) => (
                  <SeedRow key={t.id} thread={t} />
                ))}
              {posts.map((p) => (
                <ActionRow
                  key={p.id}
                  href={`/threads/${p.id}`}
                  title={p.title}
                  context="Bài của bạn"
                  meta={
                    <>
                      {`Tuần ${course.week} · vừa xong · `}
                      <StatusText tone={p.answered ? "amber" : "neutral"}>{p.answered ? "Chờ xác nhận" : "Đang soạn câu trả lời"}</StatusText>
                    </>
                  }
                />
              ))}
              {rows
                .filter((t) => !t.pinned)
                .map((t) => (
                  <SeedRow key={t.id} thread={t} />
                ))}
            </ActionList>
          )}
        </PageState>
      </Section>

      <PIIChannelDialog
        open={Boolean(guard)}
        text={guard?.text ?? ""}
        reasons={guard?.reasons ?? []}
        onClose={() => setGuard(null)}
        onPrivateChat={() => {
          setChatDraft(guard?.text ?? "");
          setGuard(null);
          router.push("/chat");
        }}
        onRedactedPost={publish}
      />
    </Page>
  );
}

function SeedRow({ thread }: { thread: Thread }) {
  const a = ANSWER_LABEL[thread.answer];
  return (
    <ActionRow
      lead={thread.pinned ? <Pin className={s.pin} aria-hidden /> : undefined}
      href={`/threads/${thread.id}`}
      title={thread.title}
      context={thread.posts[0].body}
      meta={
        <>
          {`Tuần ${thread.week} · ${thread.topic} · ${thread.replies} trả lời · ${ago(at(-thread.minsAgo))} · `}
          <StatusText tone={a.tone}>{a.text}</StatusText>
        </>
      }
    />
  );
}
