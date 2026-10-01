"use client";

import { Check, Pin } from "lucide-react";
import { useState } from "react";
import { ago, at } from "@/mock/core";
import { KEYS, type InsightThread } from "@/mock/state";
import { NEW_THREADS_KEY, REPORT_REASONS, threadById, type NewThread, type ThreadPost } from "@/mock/threads";
import { useUndoLine } from "@/shared/lib/useUndoLine";
import { useSession } from "@/shared/session/session";
import { useDemoSlice } from "@/shared/state/demo";
import {
  Button,
  ButtonLink,
  EmptyState,
  OverflowMenu,
  Page,
  PageHeader,
  PageState,
  Popover,
  Section,
  Skeleton,
  StatusText,
  Textarea,
} from "@/shared/ui";
import s from "./Threads.module.css";

/** Một thread và câu trả lời của nó (DESIGN §14.4). */
export function ThreadDetail({ id }: { id: string }) {
  const { role } = useSession();
  const [posts, setPosts] = useDemoSlice<NewThread[]>(NEW_THREADS_KEY, []);
  const [pinned] = useDemoSlice<InsightThread[]>(KEYS.insightThreads, []);
  const [removed, setRemoved] = useState<string[]>([]);
  const [edits, setEdits] = useState<Record<string, string>>({});
  const [verified, setVerified] = useState<string[]>([]);
  const undo = useUndoLine();

  const seeded = threadById(id);
  const mine = posts.find((p) => p.id === id);
  const fromInsights = pinned.find((p) => p.id === id);

  if (!seeded && !mine && !fromInsights) {
    return (
      <Page>
        <PageHeader title="Thread" back={{ href: "/threads", label: "Threads" }} />
        <EmptyState title="Không tìm thấy thread này" action={<ButtonLink href="/threads" variant="primary">Về Threads</ButtonLink>}>
          Thread có thể đã bị xoá, hoặc thuộc lớp khác lớp bạn đang xem.
        </EmptyState>
      </Page>
    );
  }

  const title = seeded?.title ?? mine?.title ?? fromInsights?.title ?? "";
  const topic = seeded?.topic ?? fromInsights?.topic ?? "Câu hỏi của bạn";
  const list: ThreadPost[] = seeded
    ? seeded.posts
    : mine
      ? [
          { id: "p1", author: "Bạn", role: "student", body: mine.body, minsAgo: 0, mine: true },
          ...(mine.answered
            ? [
                {
                  id: "p2",
                  author: "Trợ lý AI của lớp",
                  role: "ai" as const,
                  body: "Mình đã soạn một câu trả lời cho câu hỏi này. Giảng viên sẽ xác nhận trước khi câu trả lời được đánh dấu là chính thức.",
                  minsAgo: 0,
                  state: "pending" as const,
                },
              ]
            : []),
        ]
    : [{ id: "p1", author: "Giảng viên", role: "teacher", body: fromInsights?.body ?? "", minsAgo: 0 }];

  return (
    <Page>
      <PageHeader
        title={title}
        back={{ href: "/threads", label: "Threads" }}
        meta={
          <>
            {fromInsights || seeded?.pinned ? (
              <span className={s.pinNote}>
                <Pin aria-hidden /> Ghim
              </span>
            ) : null}
            <span>{topic}</span>
            {seeded && <span>Tuần {seeded.week}</span>}
          </>
        }
      />
      <PageState loading={<Skeleton lines={8} />} empty={<EmptyState title="Thread này chưa có nội dung">Nội dung sẽ hiện khi người đăng gửi câu hỏi.</EmptyState>}>
        <Section>
          <ol className={s.posts}>
            {list
              .filter((p) => !removed.includes(p.id))
              .map((p) => (
                <li key={p.id} className={[s.post, p.state === "verified" || verified.includes(p.id) ? s.verified : "", p.state === "pending" && !verified.includes(p.id) ? s.draft : ""].join(" ")}>
                  <div className={s.postHead}>
                    <span className={s.author}>{p.author}</span>
                    <span className={s.postMeta}>{p.minsAgo > 0 ? ago(at(-p.minsAgo)) : "vừa xong"}</span>
                    {(p.state === "verified" || verified.includes(p.id)) && (
                      <span className={s.verifiedLabel}>
                        <Check aria-hidden /> Đã được giảng viên xác nhận{p.verifiedBy ? ` · ${p.verifiedBy}` : ""}
                      </span>
                    )}
                    {p.state === "pending" && !verified.includes(p.id) && <StatusText tone="amber">Chờ xác nhận</StatusText>}
                  </div>
                  <p className={s.body}>{edits[p.id] ?? p.body}</p>
                  {p.citations && (
                    <ul className={s.cites}>
                      {p.citations.map((c) => (
                        <li key={c.title}>
                          <span className={s.citeTitle}>{c.title}</span>
                          <span className={s.citeMeta}>{c.locator}</span>
                        </li>
                      ))}
                    </ul>
                  )}
                  <div className={s.postActions}>
                    {role === "student" ? (
                      <>
                        <ReportButton />
                        {p.mine && (
                          <EditOwnPost
                            body={edits[p.id] ?? p.body}
                            onSave={(text) => {
                              setEdits((e) => ({ ...e, [p.id]: text }));
                              if (mine) setPosts((prev) => prev.map((x) => (x.id === mine.id ? { ...x, body: text } : x)));
                            }}
                            onDelete={() => {
                              setRemoved((r) => [...r, p.id]);
                              if (mine) setPosts((prev) => prev.filter((x) => x.id !== mine.id));
                              undo.push("Đã xoá bài của bạn", () => setRemoved((r) => r.filter((x) => x !== p.id)));
                            }}
                          />
                        )}
                      </>
                    ) : (
                      p.role === "ai" &&
                      p.state === "pending" &&
                      !verified.includes(p.id) && (
                        <>
                          <Button
                            size="sm"
                            variant="primary"
                            onClick={() => {
                              setVerified((v) => [...v, p.id]);
                              undo.push("Đã xác nhận câu trả lời", () => setVerified((v) => v.filter((x) => x !== p.id)));
                            }}
                          >
                            Xác nhận
                          </Button>
                          <Button size="sm">Chỉnh sửa</Button>
                          <OverflowMenu
                            items={[
                              {
                                label: "Loại khỏi tri thức",
                                danger: true,
                                onSelect: () => {
                                  setRemoved((r) => [...r, p.id]);
                                  undo.push("Đã loại câu trả lời khỏi tri thức", () => setRemoved((r) => r.filter((x) => x !== p.id)));
                                },
                              },
                            ]}
                          />
                        </>
                      )
                    )}
                  </div>
                </li>
              ))}
          </ol>
          {undo.node}
        </Section>
      </PageState>
    </Page>
  );
}

function ReportButton() {
  const [sent, setSent] = useState(false);
  if (sent) return <StatusText tone="green">Đã gửi báo cáo</StatusText>;
  return (
    <Popover
      label="Báo cáo bài viết"
      width={240}
      trigger={(p) => (
        <Button size="sm" onClick={p.toggle} aria-expanded={p["aria-expanded"]}>
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

/** Sửa tại chỗ bài của chính mình (không dùng modal cho việc sửa thường). */
function EditOwnPost({ body, onSave, onDelete }: { body: string; onSave: (text: string) => void; onDelete: () => void }) {
  const [editing, setEditing] = useState(false);
  const [text, setText] = useState(body);
  if (editing) {
    return (
      <div className={s.editBox}>
        <Textarea value={text} rows={4} aria-label="Sửa bài của bạn" onChange={(e) => setText(e.target.value)} />
        <div className={s.editActions}>
          <Button
            size="sm"
            variant="primary"
            onClick={() => {
              onSave(text);
              setEditing(false);
            }}
          >
            Lưu
          </Button>
          <Button
            size="sm"
            variant="ghost"
            onClick={() => {
              setText(body);
              setEditing(false);
            }}
          >
            Huỷ
          </Button>
        </div>
      </div>
    );
  }
  return (
    <>
      <Button size="sm" onClick={() => setEditing(true)}>
        Sửa
      </Button>
      <OverflowMenu items={[{ label: "Xoá bài của tôi", danger: true, onSelect: onDelete }]} />
    </>
  );
}
