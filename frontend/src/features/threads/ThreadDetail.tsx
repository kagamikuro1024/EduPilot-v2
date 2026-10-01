"use client";

import { Check, ChevronDown, ChevronUp, Edit3, MessageSquare, Pin, Sparkles } from "lucide-react";
import { useRouter } from "next/navigation";
import { useState } from "react";
import { scanPersonal } from "@/mock/chat";
import { ago, at } from "@/mock/core";
import { KEYS, type InsightThread } from "@/mock/state";
import {
  NEW_THREADS_KEY,
  REPORT_REASONS,
  THREAD_MODERATION_KEY,
  THREAD_REPLIES_KEY,
  threadById,
  type NewThread,
  type ThreadModeration,
  type ThreadPost,
  type ThreadReply,
} from "@/mock/threads";
import { CHAT_DRAFT_KEY } from "@/features/chat/ChatScreen";
import { useUndoLine } from "@/shared/lib/useUndoLine";
import { useSession } from "@/shared/session/session";
import { useDemoSlice } from "@/shared/state/demo";
import {
  Button,
  ButtonLink,
  EmptyState,
  Field,
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
import { PIIChannelDialog } from "./PIIChannelDialog";
import s from "./Threads.module.css";

const CITATION_SNIPPETS: Record<string, string> = {
  "Chương 3 — Mật mã đối xứng và chế độ vận hành":
    "Định lý chế độ khối (tr. 14–17): ECB không che giấu mẫu lặp; CBC đạt bảo mật ngữ nghĩa (semantic security) khi và chỉ khi IV được sinh ngẫu nhiên đồng đều và không thể dự đoán trước.",
  "Modern Network Security Threats":
    "Mục 2.4: Pattern leakage in Electronic Codebook (ECB) vs. Cipher Block Chaining (CBC) with unique Initialization Vector.",
};

/** Một thread và các phản hồi / câu trả lời của nó (DESIGN §14.4, Proposals #15, #16). */
export function ThreadDetail({ id }: { id: string }) {
  const { role, user } = useSession();
  const router = useRouter();
  const [posts, setPosts] = useDemoSlice<NewThread[]>(NEW_THREADS_KEY, []);
  const [pinned] = useDemoSlice<InsightThread[]>(KEYS.insightThreads, []);
  const [moderations, setModerations] = useDemoSlice<Record<string, ThreadModeration>>(THREAD_MODERATION_KEY, {});
  const [allReplies, setAllReplies] = useDemoSlice<Record<string, ThreadReply[]>>(THREAD_REPLIES_KEY, {});
  const [, setChatDraft] = useDemoSlice<string>(CHAT_DRAFT_KEY, "");

  const [edits, setEdits] = useState<Record<string, string>>({});
  const [expandedCites, setExpandedCites] = useState<Record<string, boolean>>({});
  const [editingAi, setEditingAi] = useState(false);
  const [aiEditText, setAiEditText] = useState("");
  const [showOriginalAi, setShowOriginalAi] = useState(false);

  // Reply composer state
  const [replyText, setReplyText] = useState("");
  const [replyGuard, setReplyGuard] = useState<{ text: string; reasons: string[] } | null>(null);

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

  // Danh sách posts gốc (câu hỏi + câu trả lời AI)
  const initialPosts: ThreadPost[] = seeded
    ? seeded.posts
    : mine
      ? [
          { id: "p1", author: mine.authorName || "Bạn", role: mine.authorRole || "student", body: mine.body, minsAgo: mine.createdAtMinsAgo || 0, mine: true },
          ...(mine.answered
            ? [
                {
                  id: "p2",
                  author: "Trợ lý AI của lớp",
                  role: "ai" as const,
                  body: "Theo bài giảng An ninh mạng, đối với vấn đề này cần xem xét bối cảnh thực thi và tính chất ngẫu nhiên của véc-tơ khởi tạo. Bạn có thể đối chiếu thêm bảng so sánh chế độ khối trong tài liệu Chương 3.",
                  minsAgo: 0,
                  state: "pending" as const,
                  citations: [
                    { title: "Chương 3 — Mật mã đối xứng và chế độ vận hành", locator: "trang 14–17", href: "/library" },
                  ],
                },
              ]
            : []),
        ]
      : [{ id: "p1", author: "Giảng viên", role: "teacher", body: fromInsights?.body ?? "", minsAgo: 0 }];

  // Tách câu hỏi gốc và câu trả lời AI
  const questionPost = initialPosts[0];
  const aiPost = initialPosts.find((p) => p.role === "ai");
  const otherPosts = initialPosts.slice(1).filter((p) => p.role !== "ai");

  // Moderation state cho AI post của thread này
  const modKey = `${id}:${aiPost?.id || "p2"}`;
  const currentMod = moderations[modKey];
  const aiState = currentMod?.state ?? aiPost?.state ?? "pending";
  const aiBody = currentMod?.editedBody ?? edits[aiPost?.id || ""] ?? aiPost?.body ?? "";
  const originalAiBody = currentMod?.originalBody ?? aiPost?.body ?? "";

  // Danh sách các replies thảo luận (bao gồm cả reply động do người dùng thêm)
  const threadReplies: ThreadReply[] = allReplies[id] ?? [];

  // GV / TA xác nhận câu trả lời AI
  function verifyAiAnswer() {
    if (!aiPost) return;
    const prevMod = currentMod;
    setModerations((prev) => ({
      ...prev,
      [modKey]: { state: "verified", verifiedBy: user.name, originalBody: originalAiBody, editedBody: aiBody },
    }));
    undo.push("Đã xác nhận câu trả lời của AI", () => {
      setModerations((prev) => {
        const next = { ...prev };
        if (prevMod) next[modKey] = prevMod;
        else delete next[modKey];
        return next;
      });
    });
  }

  // GV / TA lưu chỉnh sửa và xác nhận (Proposal #15, 02-AC8)
  function saveEditedAiAnswer() {
    if (!aiPost) return;
    const prevMod = currentMod;
    setModerations((prev) => ({
      ...prev,
      [modKey]: {
        state: "corrected",
        verifiedBy: user.name,
        originalBody: originalAiBody,
        editedBody: aiEditText.trim() || aiBody,
      },
    }));
    setEditingAi(false);
    undo.push("Đã sửa và xác nhận câu trả lời AI", () => {
      setModerations((prev) => {
        const next = { ...prev };
        if (prevMod) next[modKey] = prevMod;
        else delete next[modKey];
        return next;
      });
    });
  }

  // GV / TA loại bỏ câu trả lời AI khỏi tri thức
  function removeAiAnswer() {
    if (!aiPost) return;
    const prevMod = currentMod;
    setModerations((prev) => ({
      ...prev,
      [modKey]: { state: "removed", originalBody: originalAiBody },
    }));
    undo.push("Đã loại câu trả lời khỏi tri thức", () => {
      setModerations((prev) => {
        const next = { ...prev };
        if (prevMod) next[modKey] = prevMod;
        else delete next[modKey];
        return next;
      });
    });
  }

  // Gửi phản hồi thảo luận (Reply Composer)
  function publishReply(text: string) {
    const newReply: ThreadReply = {
      id: `rep-${Date.now()}`,
      threadId: id,
      author: user.name || (role === "student" ? "Bạn" : "Giảng viên"),
      role: role === "teacher" || role === "ta" ? "teacher" : "student",
      body: text.trim(),
      minsAgo: 0,
    };
    setAllReplies((prev) => ({
      ...prev,
      [id]: [...(prev[id] ?? []), newReply],
    }));
    setReplyText("");
    setReplyGuard(null);
  }

  function handleSendReply() {
    const text = replyText.trim();
    if (!text) return;
    const found = scanPersonal(text);
    if (found.count > 0) {
      setReplyGuard({ text, reasons: found.reasons });
    } else {
      publishReply(text);
    }
  }

  // Nút Hỏi trợ lý AI gợi ý phản hồi
  function handleAskAiHelper() {
    const suggestion =
      "Gợi ý từ AI: Đối với câu hỏi về chế độ mật mã, bạn hãy chú ý đến tính chất lan truyền lỗi (error propagation). Khi một khối bản mã bị lỗi, các khối bản rõ giải mã sau đó có bị ảnh hưởng hay không?";
    setReplyText((prev) => (prev ? `${prev}\n\n${suggestion}` : suggestion));
  }

  const isStaff = role === "teacher" || role === "ta";

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
        <div className={s.detailContainer}>
          {/* 1. KHỐI CÂU HỎI GỐC (Panel 1) */}
          {questionPost && (
            <div className={s.questionPanel}>
              <div className={s.postHead}>
                <span className={s.author}>{questionPost.author}</span>
                <span className={s.postMeta}>{questionPost.minsAgo > 0 ? ago(at(-questionPost.minsAgo)) : "vừa xong"}</span>
                <StatusText tone="neutral">Câu hỏi gốc</StatusText>
              </div>
              <p className={s.body}>{edits[questionPost.id] ?? questionPost.body}</p>

              <div className={s.postActions}>
                {role === "student" ? (
                  <>
                    <ReportButton />
                    {questionPost.mine && (
                      <EditOwnPost
                        body={edits[questionPost.id] ?? questionPost.body}
                        onSave={(text) => {
                          setEdits((e) => ({ ...e, [questionPost.id]: text }));
                          if (mine) setPosts((prev) => prev.map((x) => (x.id === mine.id ? { ...x, body: text } : x)));
                        }}
                        onDelete={() => {
                          if (mine) setPosts((prev) => prev.filter((x) => x.id !== mine.id));
                          router.push("/threads");
                        }}
                      />
                    )}
                  </>
                ) : null}
              </div>
            </div>
          )}

          {/* 2. KHỐI CÂU TRẢ LỜI AI (Panel 2 - nếu chưa bị loại bỏ) */}
          {aiPost && aiState !== "removed" && (
            <div className={[s.aiPanel, aiState === "verified" || aiState === "corrected" ? s.aiPanelVerified : ""].join(" ")}>
              <div className={s.postHead}>
                <span className={s.author}>
                  <Sparkles style={{ width: 15, height: 15, display: "inline", verticalAlign: "middle", marginRight: 4, color: "var(--ep-amber)" }} aria-hidden />
                  {aiPost.author}
                </span>
                <span className={s.postMeta}>{aiPost.minsAgo > 0 ? ago(at(-aiPost.minsAgo)) : "vừa xong"}</span>

                {aiState === "verified" && (
                  <span className={s.verifiedLabel}>
                    <Check aria-hidden /> Đã được giảng viên xác nhận
                  </span>
                )}
                {aiState === "corrected" && (
                  <span className={s.correctedLabel}>
                    <Check aria-hidden /> Đã được giảng viên sửa & xác nhận
                  </span>
                )}
                {aiState === "pending" && <StatusText tone="amber">Chờ xác nhận</StatusText>}
              </div>

              {/* Chế độ sửa câu trả lời AI (Proposal #15, 02-AC8) */}
              {editingAi ? (
                <div className={s.editBox}>
                  <Field label="Chỉnh sửa câu trả lời AI trước khi xác nhận">
                    {(fid) => <Textarea id={fid} rows={5} value={aiEditText} onChange={(e) => setAiEditText(e.target.value)} />}
                  </Field>
                  <div className={s.editActions}>
                    <Button variant="primary" size="sm" onClick={saveEditedAiAnswer}>
                      Lưu và xác nhận
                    </Button>
                    <Button variant="ghost" size="sm" onClick={() => setEditingAi(false)}>
                      Huỷ
                    </Button>
                  </div>
                </div>
              ) : (
                <>
                  <p className={s.body}>{aiBody}</p>

                  {/* Nút xem câu trả lời gốc nếu đã qua sửa */}
                  {aiState === "corrected" && originalAiBody && (
                    <div>
                      <button type="button" className={s.originalAnswerToggle} onClick={() => setShowOriginalAi((prev) => !prev)}>
                        {showOriginalAi ? "Ẩn câu trả lời AI gốc" : "Xem câu trả lời AI gốc (so sánh)"}
                      </button>
                      {showOriginalAi && <p className={s.originalAnswerBox}>{originalAiBody}</p>}
                    </div>
                  )}

                  {/* Trích dẫn nguồn mở rộng xem chi tiết được */}
                  {aiPost.citations && aiPost.citations.length > 0 && (
                    <div>
                      <p className={s.citesHeader}>Nguồn tham khảo trích dẫn:</p>
                      <ul className={s.cites}>
                        {aiPost.citations.map((c) => {
                          const isExpanded = expandedCites[c.title] ?? false;
                          const snippet = CITATION_SNIPPETS[c.title] || `Trích từ ${c.title}, ${c.locator}: Nội dung đối chiếu học thuật về quy chuẩn an toàn.`;
                          return (
                            <li key={c.title}>
                              <button
                                type="button"
                                className={s.citeItem}
                                onClick={() => setExpandedCites((prev) => ({ ...prev, [c.title]: !prev[c.title] }))}
                                aria-expanded={isExpanded}
                              >
                                <div style={{ display: "flex", justifyContent: "space-between", alignItems: "center" }}>
                                  <span className={s.citeTitle}>{c.title}</span>
                                  {isExpanded ? <ChevronUp style={{ width: 14, height: 14 }} aria-hidden /> : <ChevronDown style={{ width: 14, height: 14 }} aria-hidden />}
                                </div>
                                <span className={s.citeMeta}>{c.locator} · Bấm để xem đoạn trích</span>
                                {isExpanded && <p className={s.citeSnippet}>{snippet}</p>}
                              </button>
                            </li>
                          );
                        })}
                      </ul>
                    </div>
                  )}
                </>
              )}

              {/* Các thao tác kiểm duyệt của GV / TA */}
              {isStaff && !editingAi && aiState === "pending" && (
                <div className={s.postActions}>
                  <Button variant="primary" size="sm" onClick={verifyAiAnswer}>
                    Xác nhận
                  </Button>
                  <Button
                    size="sm"
                    icon={<Edit3 style={{ width: 14, height: 14 }} aria-hidden />}
                    onClick={() => {
                      setAiEditText(aiBody);
                      setEditingAi(true);
                    }}
                  >
                    Chỉnh sửa
                  </Button>
                  <OverflowMenu
                    items={[
                      {
                        label: "Loại khỏi tri thức",
                        danger: true,
                        onSelect: removeAiAnswer,
                      },
                    ]}
                  />
                </div>
              )}
            </div>
          )}

          {/* 3. KHỐI THẢO LUẬN / PHẢN HỒI KHÁC (Panel 3...) */}
          {(otherPosts.length > 0 || threadReplies.length > 0) && (
            <Section title="Thảo luận của lớp" description="Ý kiến và giải đáp từ sinh viên, trợ giảng và giảng viên">
              <div style={{ display: "grid", gap: "var(--ep-space-3)" }}>
                {otherPosts.map((p) => (
                  <div key={p.id} className={s.replyPanel}>
                    <div className={s.postHead}>
                      <span className={s.author}>{p.author}</span>
                      <span className={s.postMeta}>{p.minsAgo > 0 ? ago(at(-p.minsAgo)) : "vừa xong"}</span>
                      {p.role === "teacher" && <StatusText tone="green">Giảng viên</StatusText>}
                    </div>
                    <p className={s.body}>{p.body}</p>
                  </div>
                ))}
                {threadReplies.map((r) => (
                  <div key={r.id} className={s.replyPanel}>
                    <div className={s.postHead}>
                      <span className={s.author}>{r.author}</span>
                      <span className={s.postMeta}>{r.minsAgo > 0 ? ago(at(-r.minsAgo)) : "vừa xong"}</span>
                      {r.role === "teacher" && <StatusText tone="green">Giảng viên</StatusText>}
                    </div>
                    <p className={s.body}>{r.body}</p>
                  </div>
                ))}
              </div>
            </Section>
          )}

          {/* 4. KHỐI REPLY COMPOSER Ở CUỐI TRANG (Panel 4) */}
          <Section title="Phản hồi trong thread này" description="Đóng góp câu trả lời hoặc thảo luận mở rộng với cả lớp">
            <div className={s.replyComposerPanel}>
              <Field label="Nội dung phản hồi của bạn" helper="Hỏi đáp hoặc đóng góp ý kiến. Không ghi thông tin cá nhân.">
                {(fid) => (
                  <Textarea
                    id={fid}
                    rows={3}
                    value={replyText}
                    onChange={(e) => setReplyText(e.target.value)}
                    placeholder="Viết phản hồi hoặc đặt câu hỏi tiếp nối tại đây..."
                  />
                )}
              </Field>

              <div style={{ display: "flex", justifyContent: "space-between", alignItems: "center", flexWrap: "wrap", gap: "var(--ep-space-3)" }}>
                <Button
                  size="sm"
                  variant="secondary"
                  icon={<Sparkles style={{ width: 14, height: 14, color: "var(--ep-amber)" }} aria-hidden />}
                  onClick={handleAskAiHelper}
                >
                  Hỏi trợ lý AI
                </Button>
                <Button
                  variant="primary"
                  icon={<MessageSquare style={{ width: 15, height: 15 }} aria-hidden />}
                  onClick={handleSendReply}
                  disabled={!replyText.trim()}
                >
                  Gửi phản hồi
                </Button>
              </div>
            </div>
          </Section>

          {undo.node}
        </div>
      </PageState>

      <PIIChannelDialog
        open={Boolean(replyGuard)}
        text={replyGuard?.text ?? ""}
        reasons={replyGuard?.reasons ?? []}
        onClose={() => setReplyGuard(null)}
        onPrivateChat={() => {
          if (replyGuard) {
            setChatDraft(replyGuard.text);
            setReplyGuard(null);
            router.push("/chat");
          }
        }}
        onRedactedPost={(clean) => {
          if (replyGuard) {
            publishReply(clean);
          }
        }}
      />
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
