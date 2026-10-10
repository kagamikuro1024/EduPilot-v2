"use client";

import { MessageSquare, Pin, Sparkles } from "lucide-react";
import { useRouter } from "next/navigation";
import { useEffect, useMemo, useRef, useState } from "react";
import { CHAT_DRAFT_KEY } from "@/features/chat/ChatScreen";
import { describePii, findPii, redactPii } from "@/mock/pii";
import { KEYS, type InsightThread } from "@/mock/state";
import {
  THREADS_LIVE_KEY,
  THREADS_LIVE_SEED,
  TA_REPLY_AFTER_MS,
  TA_TYPING_AFTER_MS,
  findThread,
  postAgo,
  type ThreadPost,
  type ThreadsLive,
} from "@/mock/threads";
import { STAFF } from "@/mock/core";
import { useUndoLine } from "@/shared/lib/useUndoLine";
import { useSession } from "@/shared/session/session";
import { simNowMs, useSimNow } from "@/shared/state/clock";
import { useDemoSlice } from "@/shared/state/demo";
import { Button, ButtonLink, EmptyState, Page, PageHeader, PageState, Panel, PanelSection, Section, Skeleton, Textarea } from "@/shared/ui";
import { PIIChannelDialog } from "./PIIChannelDialog";
import { AiBlock, EditInline, PostActions, QuoteBlock, ReportButton, RoleChip } from "./ThreadParts";
import {
  addReply,
  aiBusy,
  askAiEmpty,
  deleteLiveThread,
  editPost,
  hasMention,
  removePost,
  restorePost,
  retryAi,
  saveDraft,
  setMod,
  stopAi,
  type Actor,
} from "./threadsActions";
import s from "./Threads.module.css";

const NOTICE_MS = 8000;

/** Một thread: câu hỏi gốc → câu trả lời AI → MỘT vùng "Thảo luận (n)" có ô soạn ở cuối (SRS 4.3.1, DESIGN §14.4). */
export function DemoThreadDetail({ id }: { id: string }) {
  const { role, user, courses } = useSession();
  const router = useRouter();
  const [live, setLive] = useDemoSlice<ThreadsLive>(THREADS_LIVE_KEY, THREADS_LIVE_SEED);
  const [insight] = useDemoSlice<InsightThread[]>(KEYS.insightThreads, []);
  const [, setChatDraft] = useDemoSlice<string>(CHAT_DRAFT_KEY, "");
  const undo = useUndoLine();

  const taTyping = live.due.some((d) => d.threadId === id && !d.fired);
  const now = useSimNow(taTyping ? 500 : 1000);
  const view = useMemo(() => findThread(id, live, insight, now), [id, live, insight, now]);
  const busy = view ? aiBusy(view, now) : false;

  const isStaff = role === "teacher" || role === "ta";
  const actor: Actor = { id: user.id, name: role === "teacher" ? `${user.title ?? "TS."} ${user.name}` : user.name, role: role === "teacher" ? "teacher" : role === "ta" ? "ta" : "student" };

  // Bản nháp theo thread: lưu trong phiên (sống qua rời trang); `edit` là bản đang gõ, chưa gõ thì lấy bản đã lưu.
  const [edit, setEdit] = useState<{ text: string; quoteOf?: string } | null>(null);
  const draft = edit ?? live.drafts[id] ?? { text: "" };
  const text = draft.text;
  const quoteOf = draft.quoteOf;
  const setText = (t: string) => setEdit({ text: t, quoteOf });
  const setQuoteOf = (q: string | undefined) => setEdit({ text, quoteOf: q });
  const [localNotice, setNotice] = useState<string | null>(null);
  const notice = localNotice ?? live.flash ?? null;
  const [guard, setGuard] = useState<{ kind: "reply"; text: string; withAi: boolean } | { kind: "edit"; postId: string; text: string } | null>(null);
  const [editingId, setEditingId] = useState<string | null>(null);
  const composerRef = useRef<HTMLTextAreaElement>(null);

  useEffect(() => {
    if (!edit) return;
    const t = window.setTimeout(() => setLive((prev) => ((prev.drafts[id]?.text ?? "") === edit.text && prev.drafts[id]?.quoteOf === edit.quoteOf ? prev : saveDraft(prev, id, edit))), 300);
    return () => window.clearTimeout(t);
  }, [edit, id, setLive]);

  // Dòng báo tự mất sau 8 s (hoặc khi gõ vào ô soạn).
  useEffect(() => {
    if (!localNotice && !live.flash) return;
    const t = window.setTimeout(() => {
      setNotice(null);
      setLive((prev) => (prev.flash ? { ...prev, flash: undefined } : prev));
    }, NOTICE_MS);
    return () => window.clearTimeout(t);
  }, [localNotice, live.flash, setLive]);

  // Tự cuộn tới bài mới (và tới bài trong liên kết `#post-…` của chuông).
  const lastCount = useRef<number | null>(null);
  const postsTotal = view ? view.discussion.length + (view.mainAi ? 1 : 0) : 0;
  useEffect(() => {
    if (!view) return;
    const smooth = !window.matchMedia("(prefers-reduced-motion: reduce)").matches;
    if (lastCount.current === null) {
      lastCount.current = postsTotal;
      const hash = window.location.hash.slice(1);
      if (hash) document.getElementById(hash)?.scrollIntoView({ block: "center" });
      return;
    }
    if (postsTotal > lastCount.current) {
      const last = view.discussion[view.discussion.length - 1];
      document.getElementById(`post-${last?.id}`)?.scrollIntoView({ block: "nearest", behavior: smooth ? "smooth" : "auto" });
    }
    lastCount.current = postsTotal;
  }, [postsTotal, view]);

  if (!view || !courses.some((c) => c.id === view.courseId)) {
    return (
      <Page>
        <PageHeader title="Thread" back={{ href: "/threads", label: "Threads" }} />
        <Panel>
          <EmptyState title="Không tìm thấy thread này" action={<ButtonLink href="/threads" variant="primary">Về Threads</ButtonLink>}>
            Thread có thể đã bị xoá, hoặc thuộc lớp khác lớp bạn đang xem.
          </EmptyState>
        </Panel>
      </Page>
    );
  }

  const all = [view.question, ...(view.mainAi ? [view.mainAi] : []), ...view.discussion];
  const byId = (pid?: string) => all.find((p) => p.id === pid);
  const mineOf = (p: ThreadPost) => p.authorId === user.id && p.role !== "ai";
  const canModerate = isStaff;
  const taWriting = live.due.some((d) => d.threadId === id && !d.fired && now >= d.sentMs + TA_TYPING_AFTER_MS && now < d.sentMs + TA_REPLY_AFTER_MS);
  const hits = findPii(text);

  function update(fn: (p: ThreadsLive) => ThreadsLive) {
    setLive(fn);
  }

  function send(raw: string, withAi: boolean, hidden = 0) {
    if (!view) return;
    const nowMs = simNowMs();
    update((prev) => addReply(prev, findThread(id, prev, insight, nowMs) ?? view, { actor, text: raw, quoteOf, withAi: withAi || hasMention(raw) }, nowMs).live);
    setEdit({ text: "" });
    setNotice(hidden > 0 ? `Đã ẩn ${hidden} thông tin cá nhân` : null);
    setGuard(null);
  }

  function submit(withAi: boolean) {
    const raw = text.trim();
    if (!raw) return;
    if (findPii(raw).length > 0) setGuard({ kind: "reply", text: raw, withAi });
    else send(raw, withAi);
  }

  function askAi() {
    if (text.trim()) return submit(true);
    if (!view) return;
    const r = askAiEmpty(live, view, simNowMs());
    if (r.notice) setNotice(r.notice);
    else update(() => r.live);
  }

  function moderate(post: ThreadPost, state: "verified" | "corrected" | "removed", edited?: string) {
    const prev = live.mods[`${id}:${post.id}`];
    update((p) => setMod(p, id, post.id, { state, by: user.name, edited }));
    undo.push(state === "verified" ? "Đã xác nhận câu trả lời của AI" : state === "corrected" ? "Đã sửa và xác nhận câu trả lời AI" : "Đã loại câu trả lời khỏi tri thức", () =>
      update((p) => setMod(p, id, post.id, prev ?? null)),
    );
  }

  function saveAiEdit(post: ThreadPost, edited: string) {
    if (findPii(edited).length > 0) return setGuard({ kind: "edit", postId: post.id, text: edited });
    moderate(post, edited === (post.ai?.originalBody ?? post.body) ? "verified" : "corrected", edited);
  }

  function quote(p: ThreadPost) {
    setQuoteOf(p.id);
    composerRef.current?.focus();
    composerRef.current?.scrollIntoView({ block: "center" });
  }

  const quotePost = byId(quoteOf);
  const aiPostById = (pid: string) => view.mainAi?.id === pid ? view.mainAi : view.discussion.find((p) => p.id === pid);
  const disabledReason = busy ? "Trợ lý AI đang soạn…" : !text.trim() ? "Nhập nội dung phản hồi" : "";

  function bodyOf(p: ThreadPost) {
    if (editingId === p.id)
      return (
        <EditInline
          body={p.body}
          onCancel={() => setEditingId(null)}
          onSave={(t) => {
            update((prev) => editPost(prev, id, p.id, t));
            setEditingId(null);
          }}
        />
      );
    return (
      <>
        <p className={s.body}>{p.body}</p>
        {p.edited && <p className={s.edited}>Đã sửa</p>}
      </>
    );
  }

  return (
    <Page>
      <PageHeader
        title={view.title}
        back={{ href: "/threads", label: "Threads" }}
        meta={
          view.pinned ? (
            <span className={s.pinNote}>
              <Pin aria-hidden /> Ghim
            </span>
          ) : undefined
        }
      />

      <PageState loading={<Skeleton lines={8} />} empty={<EmptyState title="Thread này chưa có nội dung">Nội dung sẽ hiện khi người đăng gửi câu hỏi.</EmptyState>}>
        <div className={s.detailContainer}>
          <Panel>
          {/* 1. Câu hỏi gốc */}
          <PanelSection>
          <div className={s.questionPanel} data-part="thread-question" id={`post-${view.question.id}`}>
            <div className={s.postHead}>
              <span className={s.author}>
                {view.question.author}
                {mineOf(view.question) ? " (bạn)" : ""}
              </span>
              <span className={s.postMeta}>
                {`${view.topic} · Tuần ${view.week} · ${postAgo(view.question, now)} · ${view.participants} người tham gia`}
              </span>
            </div>
            {editingId === view.question.id ? (
              <EditInline
                body={view.question.body}
                onCancel={() => setEditingId(null)}
                onSave={(t) => {
                  update((prev) => editPost(prev, id, view.question.id, t));
                  setEditingId(null);
                }}
              />
            ) : (
              <p className={s.body}>{view.question.body}</p>
            )}
            {role === "student" && editingId !== view.question.id && (
              <PostActions>
                {mineOf(view.question) ? (
                  <>
                    <Button size="sm" onClick={() => setEditingId(view.question.id)}>
                      Sửa
                    </Button>
                    {view.origin === "live" && (
                      <Button
                        size="sm"
                        variant="ghost"
                        onClick={() => {
                          update((prev) => deleteLiveThread(prev, id));
                          router.push("/threads");
                        }}
                      >
                        Xoá bài của tôi
                      </Button>
                    )}
                  </>
                ) : (
                  <ReportButton />
                )}
              </PostActions>
            )}
          </div>
          </PanelSection>

          {/* 2. Câu trả lời AI chính */}
          {view.mainAi && (
            <PanelSection>
            <AiBlock
              panel
              post={view.mainAi}
              now={now}
              canModerate={canModerate}
              onStop={(n) => update((p) => stopAi(p, id, view.mainAi!.id, n))}
              onRetry={() => update((p) => retryAi(p, id, view.mainAi!.id, simNowMs()))}
              onVerify={() => moderate(view.mainAi!, "verified")}
              onSaveEdit={(t) => saveAiEdit(view.mainAi!, t)}
              onRemove={() => moderate(view.mainAi!, "removed")}
            />
            </PanelSection>
          )}
          </Panel>

          {/* 3. MỘT vùng thảo luận, ô soạn ở cuối chính vùng này */}
          <Section title={`Thảo luận (${view.discussion.length})`} id="discussion" panel>
            {view.discussion.length > 0 && (
              <ol className={s.posts}>
                {view.discussion.map((p) => (
                  <li key={p.id}>
                    {p.role === "ai" && p.ai ? (
                      <AiBlock
                        post={p}
                        now={now}
                        canModerate={canModerate}
                        onStop={(n) => update((prev) => stopAi(prev, id, p.id, n))}
                        onRetry={() => update((prev) => retryAi(prev, id, p.id, simNowMs()))}
                        onVerify={() => moderate(p, "verified")}
                        onSaveEdit={(t) => saveAiEdit(p, t)}
                        onRemove={() => moderate(p, "removed")}
                      />
                    ) : (
                      <article className={s.replyPanel} id={`post-${p.id}`} data-part="thread-post">
                        <div className={s.postHead}>
                          <span className={s.author}>
                            {p.author}
                            {mineOf(p) ? " (bạn)" : ""}
                          </span>
                          <RoleChip post={p} />
                          <span className={s.postMeta}>{postAgo(p, now)}</span>
                        </div>
                        {p.quoteOf && byId(p.quoteOf) && <QuoteBlock post={byId(p.quoteOf)!} now={now} />}
                        {bodyOf(p)}
                        {editingId !== p.id && (
                          <PostActions>
                            <Button size="sm" variant="ghost" onClick={() => quote(p)}>
                              Trả lời
                            </Button>
                            {mineOf(p) && role === "student" ? (
                              <>
                                <Button size="sm" variant="ghost" onClick={() => setEditingId(p.id)}>
                                  Sửa
                                </Button>
                                {p.ms !== undefined && (
                                  <Button
                                    size="sm"
                                    variant="ghost"
                                    onClick={() => {
                                      update((prev) => removePost(prev, id, p.id));
                                      undo.push("Đã xoá phản hồi của bạn", () => update((prev) => restorePost(prev, id, p.id)));
                                    }}
                                  >
                                    Xoá
                                  </Button>
                                )}
                              </>
                            ) : role === "student" ? (
                              <ReportButton />
                            ) : null}
                          </PostActions>
                        )}
                      </article>
                    )}
                  </li>
                ))}
              </ol>
            )}

            {taWriting && (
              <p className={s.typing} role="status">
                {STAFF.ta.name} đang trả lời…
                <span className={s.dots} aria-hidden>
                  <i />
                  <i />
                  <i />
                </span>
              </p>
            )}

            <div className={s.replyComposerPanel} data-part="reply-composer">
              <p className={s.composerLabel} id="reply-label">
                Phản hồi của bạn
              </p>
              {quotePost && (
                <div className={s.quoteDraft}>
                  <QuoteBlock post={quotePost} now={now} />
                  <Button size="sm" variant="ghost" onClick={() => setQuoteOf(undefined)}>
                    Bỏ trích
                  </Button>
                </div>
              )}
              {hits.length > 0 && <p className={s.hint}>Có vẻ bài có thông tin cá nhân. Bạn sẽ được hỏi trước khi đăng.</p>}
              <Textarea
                id="reply-body"
                aria-labelledby="reply-label"
                ref={composerRef}
                rows={3}
                value={text}
                onChange={(e) => {
                  setText(e.target.value);
                  if (notice) {
                    setNotice(null);
                    if (live.flash) setLive((prev) => ({ ...prev, flash: undefined }));
                  }
                }}
                aria-describedby={disabledReason ? "reply-hint" : undefined}
                placeholder="Viết phản hồi hoặc đặt câu hỏi tiếp nối tại đây…"
              />
              {notice && (
                <p className={s.notice} role="status">
                  {notice}
                </p>
              )}
              <div className={s.composerBar}>
                {disabledReason && (
                  <span id="reply-hint" className={s.missing}>
                    {disabledReason}
                  </span>
                )}
                <Button variant="secondary" icon={<Sparkles aria-hidden />} onClick={askAi} disabled={busy}>
                  Hỏi trợ lý AI
                </Button>
                <Button variant="primary" icon={<MessageSquare aria-hidden />} onClick={() => submit(false)} disabled={busy || !text.trim()} aria-describedby={disabledReason ? "reply-hint" : undefined}>
                  Gửi phản hồi
                </Button>
              </div>
            </div>
          </Section>

          {undo.node}
        </div>
      </PageState>

      <PIIChannelDialog
        open={Boolean(guard)}
        summary={describePii(findPii(guard?.text ?? ""))}
        onClose={() => setGuard(null)}
        onPrivateChat={
          role === "student" && guard?.kind === "reply"
            ? () => {
                setChatDraft(guard.text);
                setEdit({ text: "" });
                setLive((prev) => saveDraft(prev, id, { text: "" }));
                setGuard(null);
                router.push("/chat");
              }
            : undefined
        }
        onRedactedPost={() => {
          if (!guard) return;
          if (guard.kind === "reply") send(redactPii(guard.text), guard.withAi, findPii(guard.text).length);
          else {
            const post = aiPostById(guard.postId);
            const edited = redactPii(guard.text);
            setGuard(null);
            if (post) moderate(post, "corrected", edited);
          }
        }}
      />
    </Page>
  );
}
