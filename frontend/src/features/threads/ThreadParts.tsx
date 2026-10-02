"use client";

import { Check, ChevronDown, Edit3, Sparkles } from "lucide-react";
import { useState, type ReactNode } from "react";
import { AI_SOURCES_MS, AI_TYPING_MS, aiStreamMs, postAgo, ROLE_NAME, REPORT_REASONS, type ThreadPost } from "@/mock/threads";
import { useTimedReveal } from "@/shared/lib/useStreamedText";
import { Button, OverflowMenu, Popover, StatusText, Textarea } from "@/shared/ui";
import s from "./Threads.module.css";

/** "**đậm**" của mẫu AI → <strong>; dấu mở chưa có dấu đóng (đang chảy chữ) thì phần chữ tới giờ đã đậm. */
function Rich({ text }: { text: string }) {
  return (
    <>
      {text.split("**").map((part, i) => (i % 2 === 1 ? <strong key={i}>{part}</strong> : part))}
    </>
  );
}

/** Khối trích bài được trả lời (↳). */
export function QuoteBlock({ post, now }: { post: ThreadPost; now: number }) {
  const body = post.body.length > 160 ? `${post.body.slice(0, 160).trimEnd()}…` : post.body;
  return (
    <blockquote className={s.quote}>
      <span className={s.quoteWho}>
        {post.author} · {postAgo(post, now)}
      </span>
      <span className={s.quoteBody}>{body}</span>
    </blockquote>
  );
}

export function RoleChip({ post }: { post: ThreadPost }) {
  if (post.role === "teacher" || post.role === "ta") return <StatusText tone="green">{ROLE_NAME[post.role]}</StatusText>;
  return null;
}

/** Nguồn tham khảo (n): thu gọn, bấm mở; mỗi nguồn mở trong Thư viện. */
function Sources({ post }: { post: ThreadPost }) {
  const [open, setOpen] = useState(false);
  const list = post.ai?.citations ?? [];
  if (list.length === 0) return null;
  return (
    <div className={s.sources}>
      <button type="button" className={s.sourcesToggle} onClick={() => setOpen((o) => !o)} aria-expanded={open}>
        {`Nguồn tham khảo (${list.length})`}
        <ChevronDown aria-hidden data-open={open} />
      </button>
      {open && (
        <ul className={s.cites}>
          {list.map((c) => (
            <li key={`${c.title}-${c.locator}`}>
              <a className="ep-link" href={c.href}>
                {c.title}
              </a>
              <span className={s.citeMeta}> · {c.locator}</span>
            </li>
          ))}
        </ul>
      )}
    </div>
  );
}

function StateLabel({ post }: { post: ThreadPost }) {
  const a = post.ai!;
  if (a.state === "verified")
    return (
      <span className={s.verifiedLabel}>
        <Check aria-hidden /> Đã được giảng viên xác nhận{a.verifiedBy ? ` · ${a.verifiedBy}` : ""}
      </span>
    );
  if (a.state === "corrected")
    return (
      <span className={s.correctedLabel}>
        <Check aria-hidden /> Đã được giảng viên sửa &amp; xác nhận{a.verifiedBy ? ` · ${a.verifiedBy}` : ""}
      </span>
    );
  if (a.state === "waiting") return <StatusText tone="amber">Đang chờ giảng viên</StatusText>;
  return <StatusText tone="amber">Chờ xác nhận</StatusText>;
}

/**
 * Một bài của Trợ lý AI. Bài mới (có `startedMs`) chạy trình tự E: "đang soạn…" → chữ chảy (có `Dừng`) → nguồn → nhãn.
 * `canModerate`: GV / TA thấy `Xác nhận` · `Chỉnh sửa` · `Loại khỏi tri thức` khi bài `Chờ xác nhận`.
 */
export function AiBlock({
  post,
  now,
  canModerate,
  quote,
  onStop,
  onRetry,
  onVerify,
  onSaveEdit,
  onRemove,
  panel,
}: {
  post: ThreadPost;
  now: number;
  canModerate: boolean;
  quote?: ThreadPost;
  onStop: (chars: number) => void;
  onRetry: () => void;
  onVerify: () => void;
  onSaveEdit: (text: string) => void;
  onRemove: () => void;
  /** khối chính (có khung riêng) hay bài trong vùng thảo luận */
  panel?: boolean;
}) {
  const a = post.ai!;
  const reveal = useTimedReveal({ body: post.body, startedAt: a.startedMs, typingMs: AI_TYPING_MS, streamMs: aiStreamMs(post.body), sourcesMs: AI_SOURCES_MS, stoppedAt: a.stoppedAt });
  const [showOriginal, setShowOriginal] = useState(false);
  const [editing, setEditing] = useState(false);
  const [draft, setDraft] = useState(post.body);

  const done = reveal.phase === "done";
  const showSources = done || reveal.phase === "sourcing";
  const trimmed = draft.trim();
  const editError = trimmed === "" ? "Nội dung không được để trống" : trimmed === post.body.trim() ? "Chưa có thay đổi — dùng Xác nhận nếu nội dung đã đúng" : null;

  return (
    <article className={[panel ? s.aiPanel : s.aiPost, a.state === "verified" || a.state === "corrected" ? s.aiPanelVerified : ""].join(" ")} id={`post-${post.id}`} data-part="thread-ai">
      <div className={s.postHead}>
        <span className={s.author}>
          <Sparkles className={s.aiIcon} aria-hidden />
          {post.author}
        </span>
        {done && <StateLabel post={post} />}
      </div>
      {a.replyToName && <p className={s.replyTo}>↳ trả lời {a.replyToName}</p>}
      {quote && <QuoteBlock post={quote} now={now} />}

      {reveal.phase === "typing" && (
        <p className={s.typing} role="status">
          Trợ lý AI đang soạn…
          <span className={s.dots} aria-hidden>
            <i />
            <i />
            <i />
          </span>
        </p>
      )}

      {reveal.phase !== "typing" && !editing && (
        <>
          <p className={s.body}>
            <Rich text={reveal.text} />
          </p>
          {reveal.phase === "streaming" && (
            <div className={s.postActions}>
              <Button size="sm" variant="secondary" onClick={() => onStop(reveal.chars)}>
                Dừng
              </Button>
            </div>
          )}
          {reveal.phase === "stopped" && (
            <p className={s.stopped}>
              Đã dừng ·{" "}
              <button type="button" className="ep-link" data-inline onClick={onRetry}>
                Hỏi lại
              </button>
            </p>
          )}
          {a.state === "corrected" && a.originalBody && a.originalBody !== post.body && done && (
            <div>
              <button type="button" className={s.originalAnswerToggle} onClick={() => setShowOriginal((p) => !p)}>
                {showOriginal ? "Ẩn câu trả lời AI gốc" : "Xem câu trả lời AI gốc"}
              </button>
              {showOriginal && (
                <p className={s.originalAnswerBox}>
                  <Rich text={a.originalBody} />
                </p>
              )}
            </div>
          )}
          {showSources && <Sources post={post} />}
        </>
      )}

      {editing && (
        <div className={s.editBox}>
          <label className={s.editLabel} htmlFor={`edit-${post.id}`}>
            Chỉnh sửa câu trả lời AI trước khi xác nhận
          </label>
          <Textarea id={`edit-${post.id}`} rows={6} value={draft} aria-describedby={editError ? `edit-${post.id}-err` : undefined} onChange={(e) => setDraft(e.target.value)} />
          <div className={s.editActions}>
            <Button
              variant="primary"
              size="sm"
              disabled={Boolean(editError)}
              aria-describedby={editError ? `edit-${post.id}-err` : undefined}
              onClick={() => {
                onSaveEdit(trimmed);
                setEditing(false);
              }}
            >
              Lưu và xác nhận
            </Button>
            <Button variant="ghost" size="sm" onClick={() => setEditing(false)}>
              Huỷ
            </Button>
            {editError && (
              <span id={`edit-${post.id}-err`} className={s.missing}>
                {editError}
              </span>
            )}
          </div>
        </div>
      )}

      {canModerate && done && !editing && a.state === "pending" && (
        <div className={s.postActions}>
          <Button variant="primary" size="sm" onClick={onVerify}>
            Xác nhận
          </Button>
          <Button
            size="sm"
            icon={<Edit3 aria-hidden />}
            onClick={() => {
              setDraft(post.body);
              setEditing(true);
            }}
          >
            Chỉnh sửa
          </Button>
          <OverflowMenu items={[{ label: "Loại khỏi tri thức", danger: true, onSelect: onRemove }]} />
        </div>
      )}
    </article>
  );
}

export function ReportButton() {
  const [sent, setSent] = useState(false);
  if (sent) return <StatusText tone="green">Đã gửi báo cáo</StatusText>;
  return (
    <Popover
      label="Báo cáo bài viết"
      width={240}
      trigger={(p) => (
        <Button size="sm" variant="ghost" onClick={p.toggle} aria-expanded={p["aria-expanded"]}>
          Báo cáo
        </Button>
      )}
    >
      {(close) => (
        <div className={s.reportPanel}>
          <p className={s.reportLabel}>Lý do báo cáo</p>
          {REPORT_REASONS.map((r) => (
            <button
              key={r}
              type="button"
              className={s.reportItem}
              onClick={() => {
                setSent(true);
                close();
              }}
            >
              {r}
            </button>
          ))}
        </div>
      )}
    </Popover>
  );
}

/** Sửa bài của chính mình tại chỗ. */
export function EditInline({ body, onSave, onCancel }: { body: string; onSave: (text: string) => void; onCancel: () => void }) {
  const [text, setText] = useState(body);
  const t = text.trim();
  return (
    <div className={s.editBox}>
      <Textarea value={text} rows={4} aria-label="Sửa bài của bạn" onChange={(e) => setText(e.target.value)} />
      <div className={s.editActions}>
        <Button size="sm" variant="primary" disabled={!t || t === body.trim()} onClick={() => onSave(t)}>
          Lưu
        </Button>
        <Button size="sm" variant="ghost" onClick={onCancel}>
          Huỷ
        </Button>
      </div>
    </div>
  );
}

export function PostActions({ children }: { children: ReactNode }) {
  return <div className={s.postActions}>{children}</div>;
}
