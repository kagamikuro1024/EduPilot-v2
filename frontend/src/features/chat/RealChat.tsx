"use client";

import { useQueryClient } from "@tanstack/react-query";
import { ShieldCheck, ThumbsDown, ThumbsUp, Trash2 } from "lucide-react";
import Link from "next/link";
import { useEffect, useRef, useState } from "react";
import { ApiError, apiClient, useAutosaveDraft, useUndoableAction } from "@/shared/data";
import { CitationList, Markdown } from "@/shared/domain";
import { useSession } from "@/shared/session/session";
import { Button, Composer, Page, PageHeader, PageState, Panel, Skeleton } from "@/shared/ui";
import {
  LOCK_KEY,
  messagesKey,
  sessionsKey,
  useChatMessages,
  useChatSessions,
  useExamLock,
  type ChatBlock,
  type ChatCitation,
  type ChatMessage,
  type ChatSession,
} from "./chatApi";
import { ChatSessionSheet } from "./ChatSessionSheet";
import { useChatRun, type Live } from "./useChatRun";
import s from "./ChatScreen.module.css";
import r from "./RealChat.module.css";

type Feedback = "HELPFUL" | "NOT_HELPFUL" | null;

const BLOCK_TITLE: Record<string, string> = { upcoming_events: "Lịch sắp tới", exam_schedule: "Lịch thi", library_results: "Tài liệu" };
const HM = new Intl.DateTimeFormat("vi-VN", { hour: "2-digit", minute: "2-digit" });
const DAY = new Intl.DateTimeFormat("vi-VN", { day: "2-digit", month: "2-digit", hour: "2-digit", minute: "2-digit" });
const newKey = () => crypto.randomUUID();

function errorLine(code: string | null, retryAfter?: number): string {
  if (code === "OVERLOADED") return retryAfter ? `AI đang bận. Thử lại sau khoảng ${retryAfter} giây.` : "AI đang bận. Thử lại sau ít giây.";
  if (code === "INTERRUPTED") return "Câu trả lời bị gián đoạn.";
  if (code === "NETWORK") return "Mất kết nối. Câu trả lời vẫn được lưu nếu AI đã trả lời xong.";
  return "AI đang gián đoạn. Thử lại sau.";
}

/** Chat riêng thật (DESIGN §14.2, SRS FEAT-private-chat-pii 4.7): phiên + lịch sử + luồng SSE có thể nối lại. */
export function RealChat({ courseId }: { courseId: string }) {
  const qc = useQueryClient();
  const { identity } = useSession();
  const [sid, setSid] = useState<string | null>(null);
  const [sheet, setSheet] = useState(false);
  const [hidden, setHidden] = useState<Set<string>>(new Set());
  const [fb, setFb] = useState<Record<string, Feedback>>({});
  const [sendError, setSendError] = useState<ApiError | null>(null);
  const draft = useAutosaveDraft(`chat:${courseId}:${sid ?? "moi"}`, { userId: identity?.sub });
  const keyRef = useRef<{ text: string; key: string } | null>(null);
  const endRef = useRef<HTMLDivElement>(null);
  const resumed = useRef<string | null>(null);

  const sessions = useChatSessions(courseId);
  const messages = useChatMessages(sid);
  const lock = useExamLock();
  const run = useChatRun(async () => {
    await Promise.all([qc.invalidateQueries({ queryKey: messagesKey(sid ?? "") }), qc.invalidateQueries({ queryKey: sessionsKey(courseId) })]);
  });
  const live = run.live;
  const busy = live?.status === "streaming";
  const locked = lock.data?.locked === true;
  const until = lock.data?.until ? new Date(lock.data.until) : null;

  // tới hạn khoá thì tự mở lại
  useEffect(() => {
    if (!until) return;
    const ms = until.getTime() - Date.now(); // eslint-disable-line react-hooks/purity -- trong effect, không phải lúc render
    const t = setTimeout(() => void lock.refetch(), Math.max(1000, ms + 500));
    return () => clearTimeout(t);
  }, [until?.getTime()]); // eslint-disable-line react-hooks/exhaustive-deps

  const list = messages.data ?? [];
  // mở lại giữa lúc đang sinh: nối lại luồng của tin STREAMING (không mất, không lặp)
  const streamingRow = list.find((m) => m.role === "ASSISTANT" && m.stream_status === "STREAMING");
  useEffect(() => {
    if (streamingRow && !live && resumed.current !== `${streamingRow.id}:${streamingRow.attempt}`) {
      resumed.current = `${streamingRow.id}:${streamingRow.attempt}`;
      void run.resume(streamingRow.id).catch(() => undefined);
    }
  }, [streamingRow?.id, streamingRow?.attempt, live]); // eslint-disable-line react-hooks/exhaustive-deps

  useEffect(() => {
    endRef.current?.scrollIntoView({ block: "end" });
  }, [list.length, live?.text.length, live?.status]);

  async function submit() {
    const text = draft.value.trim();
    if (!text || busy || locked) return;
    setSendError(null);
    if (keyRef.current?.text !== text) keyRef.current = { text, key: newKey() }; // gửi lại cùng chữ = cùng khoá (máy chủ chống trùng)
    try {
      let id = sid;
      if (!id) {
        id = (await apiClient.post<ChatSession>("/chat/sessions", { course_id: courseId })).data.id;
        setSid(id);
      }
      await run.send(id, text, keyRef.current.key, () => {
        draft.clear();
        keyRef.current = null;
      });
    } catch (e) {
      const err = e instanceof ApiError ? e : new ApiError({ status: 0, code: "NETWORK" });
      if (err.code === "EXAM_IN_PROGRESS") void qc.invalidateQueries({ queryKey: LOCK_KEY });
      setSendError(err);
    }
  }

  async function retry(mid: string) {
    setSendError(null);
    try {
      await run.retry(mid);
    } catch (e) {
      const err = e instanceof ApiError ? e : new ApiError({ status: 0, code: "NETWORK" });
      if (err.code === "EXAM_IN_PROGRESS") void qc.invalidateQueries({ queryKey: LOCK_KEY });
      setSendError(err);
    }
  }

  const vote = useUndoableAction<{ mid: string; value: Feedback; prev: Feedback }>({
    apply: (a) => setFb((p) => ({ ...p, [a.mid]: a.value })),
    rollback: (a) => setFb((p) => ({ ...p, [a.mid]: a.prev })),
    commit: (a) => apiClient.put(`/chat/messages/${a.mid}/feedback`, { value: a.value }),
    compensate: (a) => apiClient.put(`/chat/messages/${a.mid}/feedback`, { value: a.prev }),
    label: () => "Đã ghi nhận",
    itemKey: (a) => a.mid,
  });

  const del = useUndoableAction<string>({
    apply: (id) => setHidden((p) => new Set(p).add(id)),
    rollback: (id) =>
      setHidden((p) => {
        const n = new Set(p);
        n.delete(id);
        return n;
      }),
    commit: (id) => apiClient.delete(`/chat/sessions/${id}`),
    compensate: async (id) => {
      await apiClient.post(`/chat/sessions/${id}/restore`);
      await qc.invalidateQueries({ queryKey: sessionsKey(courseId) });
    },
    label: () => "Đã xoá phiên",
    itemKey: (id) => id,
  });

  const items = (sessions.data ?? []).filter((x) => !hidden.has(x.id));
  const open = (id: string | null) => {
    run.clear();
    setSid(id);
    setSheet(false);
    setSendError(null);
  };

  const showEmpty = !sid || (messages.isSuccess && list.length === 0 && !live);

  return (
    <Page width="full" className={s.page}>
      <PageHeader
        title="Chat riêng"
        actions={
          <Button onClick={() => open(null)} disabled={!sid && !live}>
            Phiên mới
          </Button>
        }
      />
      <div className={`${s.layout} ${r.layout}`}>
        <nav className={s.history} data-part="chat-history" aria-label="Phiên trước">
          <Panel>
            <p className={s.historyLabel}>Phiên trước</p>
            <SessionList items={items} current={sid} loading={sessions.isPending} onOpen={open} onDelete={(id) => void del.run(id)} />
            {del.node}
          </Panel>
        </nav>

        <div className={s.column}>
          <button type="button" className={s.sessionsBtn} data-part="chat-sessions-button" onClick={() => setSheet(true)} aria-haspopup="dialog">
            Phiên trước ({items.length})
          </button>
          <Panel>
            <div className={s.inner}>
              <PageState
                query={sid ? messages : undefined}
                loading={
                  <div className={s.thread} data-part="chat-thread">
                    <Skeleton lines={2} />
                    <Skeleton lines={4} />
                  </div>
                }
              >
                <div className={s.thread} data-part="chat-thread" aria-live="polite">
                  {showEmpty && !live ? (
                    <div className={s.intro}>
                      <p className="ep-item-title">Hỏi bất cứ điều gì về lớp này.</p>
                      <Button onClick={() => draft.setValue("Quy chế thi cuối kỳ nói gì về tài liệu được mang vào phòng thi?")}>
                        Quy chế thi cuối kỳ nói gì về tài liệu?
                      </Button>
                    </div>
                  ) : (
                    <>
                      {list.map((m) => (live?.mid === m.id ? null : <Row key={m.id} m={m} fb={fb[m.id] ?? m.feedback} onVote={(v) => void vote.run({ mid: m.id, value: v, prev: fb[m.id] ?? m.feedback })} onRetry={() => void retry(m.id)} disabled={busy || locked} />))}
                      {live && <LiveRow live={live} onRetry={live.mid ? () => void retry(live.mid!) : undefined} />}
                    </>
                  )}
                  {vote.node}
                  <div ref={endRef} />
                </div>
              </PageState>

              <div className={s.composer} data-part="chat-composer">
                {locked && (
                  <p className={r.lock} role="status">
                    Chat tạm khóa trong lúc bạn làm bài thi.{until ? ` Dùng lại được sau ${HM.format(until)}.` : ""}
                  </p>
                )}
                <Composer
                  value={draft.value}
                  onChange={draft.setValue}
                  onSubmit={() => void submit()}
                  busy={busy}
                  onStop={() => void run.stop()}
                  disabled={locked}
                  placeholder="Hỏi về điểm, chuyên cần, lịch hoặc nội dung bài học…"
                  label="Câu hỏi của bạn"
                  error={sendError ? sendError.userMessage : undefined}
                  onRetry={sendError ? () => void submit() : undefined}
                />
              </div>
            </div>
          </Panel>
        </div>
      </div>

      <ChatSessionSheet open={sheet} onClose={() => setSheet(false)} onNew={() => open(null)}>
        <SessionList items={items} current={sid} loading={sessions.isPending} onOpen={open} onDelete={(id) => void del.run(id)} />
      </ChatSessionSheet>
    </Page>
  );
}

function SessionList({ items, current, loading, onOpen, onDelete }: { items: ChatSession[]; current: string | null; loading: boolean; onOpen: (id: string) => void; onDelete: (id: string) => void }) {
  if (loading) return <Skeleton lines={3} />;
  if (items.length === 0) return <p className={s.historyEmpty}>Chưa có phiên nào</p>;
  return (
    <ul className={r.sheetList}>
      {items.map((x) => (
        <li key={x.id} className={r.rowActions}>
          <button type="button" className={[s.historyItem, r.rowMain, current === x.id ? s.historyOn : ""].join(" ")} aria-pressed={current === x.id} onClick={() => onOpen(x.id)}>
            <span className={s.historyTitle}>{x.title ?? "Cuộc trò chuyện mới"}</span>
            <span className={s.historyMeta}>{DAY.format(new Date(x.last_message_at))}</span>
          </button>
          <Button variant="ghost" size="sm" className={r.rowDel} aria-label={`Xoá phiên ${x.title ?? ""}`} onClick={() => onDelete(x.id)}>
            <Trash2 aria-hidden />
          </Button>
        </li>
      ))}
    </ul>
  );
}

function Row({ m, fb, onVote, onRetry, disabled }: { m: ChatMessage; fb: Feedback; onVote: (v: Feedback) => void; onRetry: () => void; disabled: boolean }) {
  if (m.role === "USER") return <div className={s.fromSv}><p className={s.who}>Bạn</p><p className={s.text}>{m.content}</p></div>;
  const failed = m.stream_status === "FAILED";
  const cancelled = m.stream_status === "CANCELLED";
  return (
    <div className={s.fromAi}>
      <p className={s.who}>Trợ lý AI</p>
      <Masked n={m.masked_count} />
      {m.stream_status === "DONE" ? <Markdown source={m.content} /> : m.content && <p className={r.plain}>{m.content}</p>}
      <Blocks blocks={m.blocks} />
      <Citations items={m.citations} />
      {(failed || cancelled) && (
        <div className={r.actions}>
          <span>{cancelled ? "Đã dừng." : errorLine(m.error_code)}</span>
          <Button size="sm" onClick={onRetry} disabled={disabled}>Thử lại</Button>
        </div>
      )}
      {m.stream_status === "DONE" && (
        <div className={s.feedback}>
          <button type="button" className={s.linkBtn} aria-pressed={fb === "HELPFUL"} onClick={() => onVote(fb === "HELPFUL" ? null : "HELPFUL")}>
            <ThumbsUp aria-hidden />
            Hữu ích
          </button>
          <button type="button" className={s.linkBtn} aria-pressed={fb === "NOT_HELPFUL"} onClick={() => onVote(fb === "NOT_HELPFUL" ? null : "NOT_HELPFUL")}>
            <ThumbsDown aria-hidden />
            Không hữu ích
          </button>
        </div>
      )}
    </div>
  );
}

function LiveRow({ live, onRetry }: { live: Live; onRetry?: () => void }) {
  const done = live.status === "done";
  return (
    <>
      {live.userText && <div className={s.fromSv}><p className={s.who}>Bạn</p><p className={s.text}>{live.userText}</p></div>}
      <div className={s.fromAi} data-part="chat-live">
        <p className={s.who}>Trợ lý AI</p>
        <Masked n={live.masked} />
        <Blocks blocks={live.blocks} />
        {live.text ? done ? <Markdown source={live.text} /> : <p className={r.plain}>{live.text}{live.status === "streaming" && <span className={s.caret} aria-hidden />}</p> : live.status === "streaming" && <p className={s.who}>{live.stage === "searching" ? "Đang tìm trong tài liệu…" : "Đang trả lời…"}</p>}
        {done && <Citations items={live.citations} />}
        {live.status === "cancelled" && <div className={r.actions}><span>Đã dừng.</span></div>}
        {live.status === "failed" && (
          <div className={r.actions} role="alert">
            <span>{errorLine(live.error?.code ?? null, live.error?.retryAfter)}</span>
            {onRetry && <Button size="sm" onClick={onRetry}>Thử lại</Button>}
          </div>
        )}
      </div>
    </>
  );
}

function Masked({ n }: { n: number }) {
  const [why, setWhy] = useState(false);
  if (n <= 0) return null;
  return (
    <>
      <p className={s.privacy}>
        <ShieldCheck aria-hidden />
        Đã ẩn {n} thông tin cá nhân trước khi gửi cho AI
        <button type="button" className={s.linkBtn} onClick={() => setWhy((v) => !v)} aria-expanded={why}>
          Tìm hiểu
        </button>
      </p>
      {why && <p className={s.privacyWhy}>Tên, MSSV, email, số điện thoại được thay bằng ký hiệu trước khi gửi cho AI và khôi phục khi hiển thị cho bạn.</p>}
    </>
  );
}

function Blocks({ blocks }: { blocks: ChatBlock[] }) {
  return (
    <>
      {blocks.map((b, i) => {
        const data = b.data && typeof b.data === "object" ? (b.data as Record<string, unknown>) : {};
        const rows = Object.entries(data);
        if (rows.length === 0) return null;
        return (
          <section key={i} className={s.block} aria-label={BLOCK_TITLE[b.kind] ?? "Kết quả"}>
            <h3 className={r.blockTitle}>{BLOCK_TITLE[b.kind] ?? "Kết quả"}</h3>
            <dl className={r.list}>
              {rows.map(([k, v]) => (
                <div key={k}>
                  <dt>{k}</dt>
                  <dd>{Array.isArray(v) ? v.map((x) => (typeof x === "object" ? JSON.stringify(x) : String(x))).join(", ") : typeof v === "object" ? JSON.stringify(v) : String(v)}</dd>
                </div>
              ))}
            </dl>
          </section>
        );
      })}
    </>
  );
}

function Citations({ items }: { items: ChatCitation[] }) {
  if (!items || items.length === 0) return null;
  return (
    <CitationList
      items={items.map((c) => ({
        id: String(c.n),
        title: c.title,
        page: c.page_no ?? undefined,
        excerpt: (
          <>
            {c.snippet}{" "}
            <Link href={`/library/${c.document_id}`} className={s.linkBtn}>
              Xem tài liệu
            </Link>
          </>
        ),
      }))}
    />
  );
}
