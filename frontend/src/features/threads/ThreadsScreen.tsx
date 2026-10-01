"use client";

import { Pin } from "lucide-react";
import { useRouter } from "next/navigation";
import { useState } from "react";
import { redact, scanPersonal } from "@/mock/chat";
import { KEYS, type InsightThread } from "@/mock/state";
import { NEW_THREADS_KEY, THREADS, THREAD_TOPICS, type NewThread, type Thread } from "@/mock/threads";
import { CHAT_DRAFT_KEY } from "@/features/chat/ChatScreen";
import { useSession } from "@/shared/session/session";
import { useDemoSlice } from "@/shared/state/demo";
import {
  ActionList,
  ActionRow,
  Button,
  ButtonLink,
  Checkbox,
  EmptyState,
  Field,
  FilterChips,
  Input,
  Page,
  PageHeader,
  PageState,
  Section,
  Select,
  Skeleton,
  StatusText,
  Textarea,
  Toolbar,
} from "@/shared/ui";
import { PIIChannelDialog } from "./PIIChannelDialog";
import s from "./Threads.module.css";

const ANSWER_LABEL = {
  verified: { tone: "green", text: "Đã được giảng viên xác nhận" },
  pending: { tone: "amber", text: "Chờ xác nhận" },
  none: { tone: "neutral", text: "Chưa có câu trả lời" },
} as const;

/** Threads: đọc và đặt câu hỏi công khai của lớp (DESIGN §14.3, Proposals #15, #16). */
export function ThreadsScreen() {
  const { role, user, course } = useSession();
  const router = useRouter();
  const [posts, setPosts] = useDemoSlice<NewThread[]>(NEW_THREADS_KEY, []);
  const [pinned] = useDemoSlice<InsightThread[]>(KEYS.insightThreads, []);
  const [, setChatDraft] = useDemoSlice<string>(CHAT_DRAFT_KEY, "");

  // Form tạo thread mới (Proposal #15)
  const [title, setTitle] = useState("");
  const [topic, setTopic] = useState(THREAD_TOPICS[0] ?? "Mật mã đối xứng");
  const [content, setContent] = useState("");
  const [askAi, setAskAi] = useState(true);

  const [guard, setGuard] = useState<{ title: string; content: string; reasons: string[] } | null>(null);
  const [query, setQuery] = useState("");
  const [topics, setTopics] = useState<string[]>([]);

  const coursePinned = pinned.filter((p) => p.courseId === course.id);

  function publish(cleanTitle: string, cleanContent: string) {
    const id = `t-new-${Date.now()}`;
    const newPost: NewThread = {
      id,
      courseId: course.id,
      title: cleanTitle.trim(),
      body: cleanContent.trim(),
      topic,
      answered: askAi,
      askAi,
      authorName: user.name || "Bạn",
      authorRole: role === "teacher" || role === "ta" ? "teacher" : "student",
      createdAtMinsAgo: 0,
    };
    setPosts((prev) => [newPost, ...prev]);
    setTitle("");
    setContent("");
    setGuard(null);
    // Chuyển hướng ngay lập tức sang /threads/${id} (Proposal #15, 01-AC4)
    router.push(`/threads/${id}`);
  }

  function submit() {
    const t = title.trim();
    const c = content.trim();
    if (!t || !c) return;
    const fullText = `${t}\n${c}`;
    const found = scanPersonal(fullText);
    if (found.count > 0) {
      setGuard({ title: t, content: c, reasons: found.reasons });
    } else {
      publish(t, c);
    }
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
          <div className={s.createPanel}>
            <Field label="Tiêu đề câu hỏi" required helper="Tóm tắt ngắn gọn thắc mắc của bạn">
              {(id) => (
                <Input
                  id={id}
                  value={title}
                  onChange={(e) => setTitle(e.target.value)}
                  placeholder="Ví dụ: Vì sao CBC cần véc-tơ khởi tạo ngẫu nhiên?"
                />
              )}
            </Field>

            <Field label="Chủ đề môn học" required>
              {(id) => (
                <Select id={id} value={topic} onChange={(e) => setTopic(e.target.value)}>
                  {THREAD_TOPICS.map((t) => (
                    <option key={t} value={t}>
                      {t}
                    </option>
                  ))}
                </Select>
              )}
            </Field>

            <Field label="Nội dung chi tiết câu hỏi" required helper="Mô tả bối cảnh và thắc mắc cụ thể. Đừng ghi MSSV hay điểm cá nhân.">
              {(id) => (
                <Textarea
                  id={id}
                  rows={4}
                  value={content}
                  onChange={(e) => setContent(e.target.value)}
                  placeholder="Mô tả chi tiết thắc mắc của bạn về kiến thức môn học..."
                />
              )}
            </Field>

            <div className={s.createActions}>
              <div className={s.checkboxRow}>
                <Checkbox
                  label="Nhờ AI trả lời gợi ý (Socratic) ngay sau khi đăng"
                  checked={askAi}
                  onChange={(e) => setAskAi(e.target.checked)}
                />
              </div>
              <Button variant="primary" onClick={submit} disabled={!title.trim() || !content.trim()}>
                Đăng câu hỏi
              </Button>
            </div>
          </div>
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
        text={guard ? `${guard.title}\n\n${guard.content}` : ""}
        reasons={guard?.reasons ?? []}
        onClose={() => setGuard(null)}
        onPrivateChat={() => {
          if (guard) {
            setChatDraft(`${guard.title}\n\n${guard.content}`);
            setGuard(null);
            router.push("/chat");
          }
        }}
        onRedactedPost={() => {
          if (guard) {
            const cleanTitle = redact(guard.title);
            const cleanContent = redact(guard.content);
            publish(cleanTitle, cleanContent);
          }
        }}
      />
    </Page>
  );
}

function SeedRow({ thread }: { thread: Thread }) {
  const a = ANSWER_LABEL[thread.answer];
  return (
    <ActionRow
      href={`/threads/${thread.id}`}
      title={thread.title}
      context={thread.posts[0]?.body ?? ""}
      meta={
        <>
          {thread.pinned ? "Ghim · " : ""}
          {`${thread.topic} · Tuần ${thread.week} · ${thread.replies} phản hồi · `}
          <StatusText tone={a.tone}>{a.text}</StatusText>
        </>
      }
    />
  );
}
