"use client";

import { Pin } from "lucide-react";
import { useRouter, useSearchParams } from "next/navigation";
import { useEffect, useRef, useState } from "react";
import { CHAT_DRAFT_KEY } from "@/features/chat/ChatScreen";
import { agoLabel } from "@/mock/derive";
import { pushNote } from "@/mock/notes";
import { describePii, findPii, redactPii } from "@/mock/pii";
import { KEYS, type InsightThread } from "@/mock/state";
import {
  THREADS_LIVE_KEY,
  THREADS_LIVE_SEED,
  THREAD_TOPICS,
  hasPendingAi,
  replyCount,
  threadsOf,
  type ListLabel,
  type NewThreadForm,
  type ThreadView,
  type ThreadsLive,
} from "@/mock/threads";
import { useSession } from "@/shared/session/session";
import { simNowMs, useSimNow } from "@/shared/state/clock";
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
  InlineNotice,
  Input,
  Page,
  PageHeader,
  PageState,
  Panel,
  Section,
  Select,
  Skeleton,
  StatusText,
  Textarea,
  Toolbar,
  type StatusTone,
} from "@/shared/ui";
import { PIIChannelDialog } from "./PIIChannelDialog";
import { createThread } from "./threadsActions";
import s from "./Threads.module.css";

export const LIST_LABEL: Record<ListLabel, { tone: StatusTone; text: string } | null> = {
  verified: { tone: "green", text: "Đã được giảng viên xác nhận" },
  corrected: { tone: "green", text: "Đã được giảng viên sửa & xác nhận" },
  pending: { tone: "amber", text: "Chờ xác nhận" },
  waiting: { tone: "amber", text: "Đang chờ giảng viên" },
  none: { tone: "neutral", text: "Chưa có câu trả lời" },
  staff: null,
};

const EMPTY_FORM: NewThreadForm = { title: "", topic: "", content: "", askAi: true };
const STATUS_OPTIONS = [{ value: "pending", label: "Chờ xác nhận" }];
const EXCERPT = 120;

/** Threads: đọc và đặt câu hỏi công khai của lớp (DESIGN §14.3). Danh sách là chính; form mở tại chỗ sau `Đặt câu hỏi`. */
export function ThreadsScreen() {
  const { role, user, course, courses, isAll } = useSession();
  const router = useRouter();
  const params = useSearchParams();
  const now = useSimNow(30000);
  const [live, setLive] = useDemoSlice<ThreadsLive>(THREADS_LIVE_KEY, THREADS_LIVE_SEED);
  const [insight] = useDemoSlice<InsightThread[]>(KEYS.insightThreads, []);
  const [, setChatDraft] = useDemoSlice<string>(CHAT_DRAFT_KEY, "");

  const isStaff = role === "teacher" || role === "ta";
  const canPost = role === "student" || isStaff;
  const topicChoices = THREAD_TOPICS.filter((t) => isStaff || t !== "Thông báo");

  const [open, setOpen] = useState(false);
  const [form, setForm] = useState<NewThreadForm>(EMPTY_FORM);
  const [guard, setGuard] = useState<{ title: string; content: string } | null>(null);
  const [query, setQuery] = useState("");
  const [topics, setTopics] = useState<string[]>([]);
  const [status, setStatus] = useState<string[]>(params.get("filter") === "pending" ? ["pending"] : []);

  // Bản nháp form lưu trong phiên: rời trang quay lại, hoặc sang chat riêng rồi về, chữ vẫn còn.
  const restored = useRef(false);
  useEffect(() => {
    if (restored.current || !live.form) return;
    restored.current = true;
    setForm(live.form);
    setOpen(true);
  }, [live.form]);
  useEffect(() => {
    const t = window.setTimeout(() => {
      const hasText = form.title !== "" || form.content !== "" || form.topic !== "";
      setLive((prev) => {
        if (!hasText) return prev.form ? { ...prev, form: undefined } : prev;
        const f = prev.form;
        return f && f.title === form.title && f.topic === form.topic && f.content === form.content && f.askAi === form.askAi ? prev : { ...prev, form };
      });
    }, 300);
    return () => window.clearTimeout(t);
  }, [form, setLive]);

  const courseIds = isAll ? courses.map((c) => c.id) : [course.id];
  const views = courseIds.flatMap((id) => threadsOf(id, live, insight, now));
  const rows = views
    .filter((t) => (topics.length === 0 ? true : topics.includes(t.topic)))
    .filter((t) => (status.includes("pending") ? hasPendingAi(t) : true))
    .filter((t) => (query.trim() ? t.title.toLowerCase().includes(query.trim().toLowerCase()) : true))
    .sort((a, b) => Number(b.pinned) - Number(a.pinned) || b.lastMs - a.lastMs);
  const pendingCount = views.filter(hasPendingAi).length;

  const hits = findPii(`${form.title}\n${form.content}`);
  const missing = [!form.title.trim() && "tiêu đề", !form.topic && "chủ đề", !form.content.trim() && "nội dung"].filter(Boolean) as string[];
  const missingText = missing.length ? `Còn thiếu: ${missing.join(", ")}` : "";

  function publish(title: string, body: string, hidden = 0) {
    const nowMs = simNowMs();
    const actorRole = role === "teacher" || role === "ta" ? role : "student";
    let id = "";
    setLive((prev) => {
      const r = createThread(prev, { actor: { id: user.id, name: user.name, role: actorRole }, courseId: course.id, week: course.week, title, topic: form.topic, body, askAi: form.askAi }, nowMs);
      id = r.id;
      return hidden > 0 ? { ...r.live, flash: `Đã ẩn ${hidden} thông tin cá nhân` } : r.live;
    });
    pushNote({ to: { roles: ["teacher", "ta"], courseId: course.id }, title: `Câu hỏi mới trong Threads: «${title}»`, meta: "Threads", href: `/threads/${id}`, ms: nowMs });
    setForm(EMPTY_FORM);
    setOpen(false);
    setGuard(null);
    // Đăng xong chuyển ngay sang thread vừa tạo (01-AC4).
    router.push(`/threads/${id}`);
  }

  function submit() {
    if (missing.length) return;
    const title = form.title.trim();
    const content = form.content.trim();
    if (hits.length > 0) setGuard({ title, content });
    else publish(title, content);
  }

  function toPrivateChat() {
    if (!guard) return;
    setChatDraft(`${guard.title}\n\n${guard.content}`);
    setLive((prev) => ({ ...prev, form: undefined }));
    setForm(EMPTY_FORM);
    setGuard(null);
    setOpen(false);
    router.push("/chat");
  }

  return (
    <Page>
      <PageHeader
        title="Threads"
        description={`Câu hỏi công khai của ${isAll ? "các lớp bạn phụ trách" : course.label}. Câu trả lời có nhãn xác nhận là đã được giảng viên duyệt.`}
        meta={`${rows.length} thread · tuần ${course.week}`}
        actions={
          canPost && !open ? (
            <Button variant="primary" onClick={() => setOpen(true)} aria-expanded={open}>
              Đặt câu hỏi
            </Button>
          ) : undefined
        }
      />

      {canPost && open && (
        <Section title="Đặt câu hỏi" description="Hỏi về nội dung môn học. Đừng ghi mã số sinh viên, số điện thoại hay điểm cá nhân — chuyện riêng hỏi ở Chat riêng." panel action={<Button variant="ghost" size="sm" onClick={() => setOpen(false)}>Thu gọn</Button>}>
          <form
            className={s.createPanel}
            data-part="thread-form"
            onSubmit={(e) => {
              e.preventDefault();
              submit();
            }}
          >
            <Field label="Tiêu đề câu hỏi" required helper="Tóm tắt ngắn gọn thắc mắc của bạn">
              {(id, describedBy) => (
                <Input id={id} aria-describedby={describedBy} value={form.title} onChange={(e) => setForm((f) => ({ ...f, title: e.target.value }))} placeholder="Ví dụ: Vì sao CBC cần véc-tơ khởi tạo ngẫu nhiên?" />
              )}
            </Field>

            <Field label="Chủ đề" required>
              {(id) => (
                <Select id={id} value={form.topic} onChange={(e) => setForm((f) => ({ ...f, topic: e.target.value }))}>
                  <option value="">Chọn chủ đề</option>
                  {topicChoices.map((t) => (
                    <option key={t} value={t}>
                      {t}
                    </option>
                  ))}
                </Select>
              )}
            </Field>

            <Field label="Nội dung chi tiết" required helper="Mô tả bối cảnh và thắc mắc cụ thể.">
              {(id, describedBy) => (
                <Textarea id={id} aria-describedby={describedBy} rows={4} value={form.content} onChange={(e) => setForm((f) => ({ ...f, content: e.target.value }))} placeholder="Mô tả chi tiết thắc mắc của bạn về kiến thức môn học…" />
              )}
            </Field>

            {hits.length > 0 && <InlineNotice tone="warning" compact>Có vẻ bài có thông tin cá nhân. Bạn sẽ được hỏi trước khi đăng.</InlineNotice>}

            <div className={s.createActions}>
              <Checkbox label="Nhờ AI trả lời gợi ý (Socratic) ngay sau khi đăng" checked={form.askAi} onChange={(e) => setForm((f) => ({ ...f, askAi: e.target.checked }))} />
              <div className={s.submitGroup}>
                {missingText && (
                  <p id="thread-form-missing" className={s.missing}>
                    {missingText}
                  </p>
                )}
                <Button type="submit" variant="primary" disabled={missing.length > 0} aria-describedby={missingText ? "thread-form-missing" : undefined}>
                  Đăng câu hỏi
                </Button>
              </div>
            </div>
          </form>
        </Section>
      )}

      <Section title="Câu hỏi của lớp">
        <Toolbar
          end={
            <>
              {isStaff && <FilterChips label="Lọc theo trạng thái" value={status} onChange={setStatus} options={STATUS_OPTIONS.map((o) => ({ ...o, label: `${o.label} ${pendingCount}` }))} />}
              <FilterChips label="Lọc theo chủ đề" value={topics} onChange={setTopics} options={THREAD_TOPICS.map((t) => ({ value: t, label: t }))} />
            </>
          }
        >
          <Input className={s.search} type="search" value={query} placeholder="Tìm trong tiêu đề thread" aria-label="Tìm thread" onChange={(e) => setQuery(e.target.value)} />
        </Toolbar>

        <Panel>
        <PageState
          loading={<Skeleton lines={6} />}
          empty={
            <EmptyState title="Chưa có câu hỏi nào trong tuần này" action={canPost ? <Button variant="primary" onClick={() => setOpen(true)}>Đặt câu hỏi</Button> : undefined}>
              Khi bạn hoặc bạn cùng lớp đặt câu hỏi, thread sẽ hiện ở đây kèm trạng thái xác nhận của giảng viên.
            </EmptyState>
          }
        >
          {rows.length === 0 ? (
            views.length === 0 ? (
              <EmptyState title="Chưa có câu hỏi nào trong tuần này" action={canPost ? <Button variant="primary" onClick={() => setOpen(true)}>Đặt câu hỏi</Button> : undefined}>
                Khi bạn hoặc bạn cùng lớp đặt câu hỏi, thread sẽ hiện ở đây kèm trạng thái xác nhận của giảng viên.
              </EmptyState>
            ) : (
              <EmptyState title="Không có thread nào khớp bộ lọc" action={<ButtonLink href="/threads">Bỏ bộ lọc</ButtonLink>}>
                Thử bỏ bớt chủ đề hoặc tìm bằng từ khoá khác.
              </EmptyState>
            )
          ) : (
            <ActionList label="Thread của lớp">
              {rows.map((t) => (
                <ThreadRow key={t.id} thread={t} mine={t.question.authorId === user.id} now={now} />
              ))}
            </ActionList>
          )}
        </PageState>
        </Panel>
      </Section>

      <PIIChannelDialog
        open={Boolean(guard)}
        summary={describePii(hits)}
        onClose={() => setGuard(null)}
        onPrivateChat={role === "student" ? toPrivateChat : undefined}
        onRedactedPost={() => guard && publish(redactPii(guard.title), redactPii(guard.content), hits.length)}
      />
    </Page>
  );
}

function ThreadRow({ thread, mine, now }: { thread: ThreadView; mine: boolean; now: number }) {
  const label = LIST_LABEL[thread.label];
  const body = thread.question.body;
  const n = replyCount(thread);
  return (
    <ActionRow
      href={`/threads/${thread.id}`}
      data={{ "data-part": "thread-row" }}
      lead={thread.pinned ? <Pin className={s.pin} aria-hidden /> : undefined}
      title={
        <>
          {thread.title}
          {mine && (
            <>
              {" "}
              <StatusText tone="neutral" chip>
                Của bạn
              </StatusText>
            </>
          )}
        </>
      }
      context={<span className={s.clamp}>{body.length > EXCERPT ? `${body.slice(0, EXCERPT).trimEnd()}…` : body}</span>}
      meta={
        <>
          {thread.pinned ? "Ghim · " : ""}
          {`${thread.topic} · Tuần ${thread.week} · `}
          {n > 0 ? `${n} phản hồi · ` : ""}
          {agoLabel(thread.lastMs, now)}
          {label && (
            <>
              {" · "}
              <StatusText tone={label.tone}>{label.text}</StatusText>
            </>
          )}
        </>
      }
    />
  );
}
