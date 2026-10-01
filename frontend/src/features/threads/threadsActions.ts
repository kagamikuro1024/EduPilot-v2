// Hàm thuần biến đổi trạng thái phiên của Threads (`ThreadsLive`). Không đụng React, không đọc đồng hồ:
// mọi mốc thời gian truyền vào (`nowMs`, giờ giả lập) để kiểm thử và để một thao tác ghi nhiều bài có mốc ổn định.
import { STAFF } from "@/mock/core";
import type { Note } from "@/mock/notes";
import {
  AI_TYPING_MS,
  TA_REPLY_AFTER_MS,
  aiStreamMs,
  findThread,
  pickTemplate,
  planAnswer,
  planHint,
  taReplyFor,
  threadNo,
  type AiPlan,
  type InsightThreadLike,
  type PostRole,
  type ThreadDraft,
  type ThreadPost,
  type ThreadsLive,
  type ThreadView,
} from "@/mock/threads";

export type Actor = { id: string; name: string; role: Exclude<PostRole, "ai"> };

const k = (threadId: string, postId: string) => `${threadId}:${postId}`;

/** Bỏ `@AI` khỏi câu hỏi trước khi chọn mẫu (nó chỉ là lời gọi). */
export const stripMention = (text: string) => text.replace(/@ai\b/gi, "").trim();
export const hasMention = (text: string) => /@ai\b/i.test(text);

export const NOTICE_NO_MORE_HINTS = "Trợ lý đã đưa hết gợi ý có trong tài liệu của lớp. Hãy trả lời các câu hỏi ngược ở trên hoặc nhập câu hỏi cụ thể rồi bấm lại.";
export const NOTICE_REPORTED = "Mình đã báo giảng viên; câu trả lời sẽ hiện ngay trong thread này.";

function aiPost(threadId: string, id: string, ms: number, plan: AiPlan, opts: { main: boolean; replyToName?: string }): ThreadPost {
  return {
    id,
    threadId,
    authorId: "ai",
    author: "Trợ lý AI của lớp",
    role: "ai",
    body: plan.body,
    ms,
    ai: {
      kind: plan.kind,
      main: opts.main,
      state: plan.state,
      citations: plan.citations,
      template: plan.template,
      replyToName: opts.replyToName,
      // nhánh không khớp hiện ngay toàn văn (không chảy chữ) nên không cần mốc bắt đầu
      startedMs: plan.kind === "fallback" ? undefined : ms,
    },
  };
}

/** Đăng thread mới; nếu bật nhờ AI thì tạo luôn câu trả lời AI chính (mẫu theo từ khoá, hoặc nhánh không khớp). */
export function createThread(
  live: ThreadsLive,
  input: { actor: Actor; courseId: string; week: number; title: string; topic: string; body: string; askAi: boolean },
  nowMs: number,
): { live: ThreadsLive; id: string } {
  // số thứ tự trong phiên: t-new-1, t-new-2… (không dùng thời gian)
  const id = `t-new-${Math.max(0, ...live.threads.map((t) => threadNo(t.id))) + 1}`;
  const posts = [...live.posts];
  if (input.askAi) posts.push(aiPost(id, "ai-main", nowMs + 1, planAnswer(`${input.title}\n${input.body}`, input.topic), { main: true }));
  return {
    id,
    live: {
      ...live,
      form: undefined,
      threads: [
        {
          id,
          courseId: input.courseId,
          title: input.title,
          topic: input.topic,
          week: input.week,
          body: input.body,
          ms: nowMs,
          askAi: input.askAi,
          authorId: input.actor.id,
          author: input.actor.name,
          role: input.actor.role,
        },
        ...live.threads,
      ],
      posts,
    },
  };
}

/**
 * Gửi một phản hồi của người. `withAi`: gọi AI trả lời ngay dưới (nút `Hỏi trợ lý AI` có chữ, hoặc có `@AI`).
 * Phản hồi đầu tiên của SINH VIÊN trong phiên ở thread này → hẹn phản hồi trễ của trợ giảng (J2: mỗi thread một lần,
 * kể cả khi đi qua `Hỏi trợ lý AI`; GV / TA không kích hoạt).
 */
export function addReply(
  live: ThreadsLive,
  thread: ThreadView,
  input: { actor: Actor; text: string; quoteOf?: string; withAi: boolean },
  nowMs: number,
): { live: ThreadsLive; postId: string } {
  const postId = `r-${nowMs}`;
  const text = input.text.trim();
  const post: ThreadPost = { id: postId, threadId: thread.id, authorId: input.actor.id, author: input.actor.name, role: input.actor.role, body: text, ms: nowMs, quoteOf: input.quoteOf };
  const posts = [...live.posts, post];
  if (input.withAi) {
    const plan = planAnswer(stripMention(text), thread.topic);
    posts.push(aiPost(thread.id, `ai-${nowMs + 1}`, nowMs + 1, plan, { main: false, replyToName: input.actor.name }));
  }
  const due = [...live.due];
  if (input.actor.role === "student" && !due.some((d) => d.threadId === thread.id)) {
    due.push({ id: `due-${thread.id}`, threadId: thread.id, quoteOf: postId, sentMs: nowMs, studentId: input.actor.id });
  }
  const drafts = { ...live.drafts };
  delete drafts[thread.id];
  return { live: { ...live, posts, due, drafts }, postId };
}

/**
 * `Hỏi trợ lý AI` khi ô soạn trống (SRS 4.3.1 F, J1). Luôn có phản hồi nhìn thấy:
 * - chưa có câu AI chính → trả lời câu hỏi gốc (bài AI chính);
 * - đã có, mẫu khớp, chưa dùng `Gợi ý thêm` → đăng `Gợi ý thêm` (một lần mỗi thread);
 * - đã dùng hết / câu hỏi gốc không khớp mà đã báo giảng viên → không đăng, trả về dòng thông báo.
 */
export function askAiEmpty(live: ThreadsLive, thread: ThreadView, nowMs: number): { live: ThreadsLive; postId?: string; notice?: string } {
  const everyAi = [thread.mainAi, ...thread.discussion].filter((p): p is ThreadPost => Boolean(p?.ai));
  const question = `${thread.title}\n${thread.question.body}`;
  const matched = pickTemplate(question, thread.topic);
  if (!matched) {
    if (everyAi.some((p) => p.ai!.kind === "fallback")) return { live, notice: NOTICE_REPORTED };
    const postId = `ai-${nowMs}`;
    return { live: { ...live, posts: [...live.posts, aiPost(thread.id, postId, nowMs, planAnswer(question, thread.topic), { main: !thread.mainAi })] }, postId };
  }
  if (!thread.mainAi && everyAi.length === 0) {
    const postId = `ai-${nowMs}`;
    return { live: { ...live, posts: [...live.posts, aiPost(thread.id, postId, nowMs, planAnswer(question, thread.topic), { main: true })] }, postId };
  }
  if (everyAi.some((p) => p.ai!.kind === "hint")) return { live, notice: NOTICE_NO_MORE_HINTS };
  const postId = `ai-${nowMs}`;
  return { live: { ...live, posts: [...live.posts, aiPost(thread.id, postId, nowMs, planHint(question, thread.topic), { main: false })] }, postId };
}

function patchPost(live: ThreadsLive, threadId: string, postId: string, fn: (p: ThreadPost) => ThreadPost): ThreadsLive {
  return { ...live, posts: live.posts.map((p) => (p.threadId === threadId && p.id === postId ? fn(p) : p)) };
}

/** Dừng khi AI đang chảy chữ: giữ phần đã hiện. */
export const stopAi = (live: ThreadsLive, threadId: string, postId: string, chars: number) =>
  patchPost(live, threadId, postId, (p) => (p.ai ? { ...p, ai: { ...p.ai, stoppedAt: chars } } : p));

/** `Hỏi lại` sau khi dừng: soạn lại từ đầu. */
export const retryAi = (live: ThreadsLive, threadId: string, postId: string, nowMs: number) =>
  patchPost(live, threadId, postId, (p) => (p.ai ? { ...p, ms: nowMs, ai: { ...p.ai, stoppedAt: undefined, startedMs: nowMs } } : p));

export function setMod(live: ThreadsLive, threadId: string, postId: string, mod: ThreadsLive["mods"][string] | null): ThreadsLive {
  const mods = { ...live.mods };
  if (mod) mods[k(threadId, postId)] = mod;
  else delete mods[k(threadId, postId)];
  return { ...live, mods };
}

export function saveDraft(live: ThreadsLive, threadId: string, draft: ThreadDraft): ThreadsLive {
  const drafts = { ...live.drafts };
  if (draft.text || draft.quoteOf) drafts[threadId] = draft;
  else delete drafts[threadId];
  return { ...live, drafts };
}

export const editPost = (live: ThreadsLive, threadId: string, postId: string, text: string): ThreadsLive => ({ ...live, edits: { ...live.edits, [k(threadId, postId)]: text } });

export function removePost(live: ThreadsLive, threadId: string, postId: string): ThreadsLive {
  return { ...live, removed: [...live.removed, k(threadId, postId)] };
}

export function restorePost(live: ThreadsLive, threadId: string, postId: string): ThreadsLive {
  return { ...live, removed: live.removed.filter((x) => x !== k(threadId, postId)) };
}

/** Xoá hẳn thread tạo trong phiên (bài của chính mình). */
export function deleteLiveThread(live: ThreadsLive, threadId: string): ThreadsLive {
  return {
    ...live,
    threads: live.threads.filter((t) => t.id !== threadId),
    posts: live.posts.filter((p) => p.threadId !== threadId),
    due: live.due.filter((d) => d.threadId !== threadId),
  };
}

/**
 * Tới hạn thì trợ giảng "trả lời": thêm phản hồi có trích bài của sinh viên. Trả thêm các thông báo chuông cần đẩy
 * (nơi gọi `pushNote`, vì đây là hàm thuần). `live` giữ nguyên tham chiếu khi chưa có gì tới hạn.
 */
export function fireDueReplies(live: ThreadsLive, insight: InsightThreadLike[], nowMs: number): { live: ThreadsLive; notes: Array<Omit<Note, "readBy">> } {
  const dueNow = live.due.filter((d) => !d.fired && nowMs >= d.sentMs + TA_REPLY_AFTER_MS);
  if (dueNow.length === 0) return { live, notes: [] };
  let next = live;
  const notes: Array<Omit<Note, "readBy">> = [];
  for (const d of dueNow) {
    const thread = findThread(d.threadId, next, insight, nowMs);
    next = { ...next, due: next.due.map((x) => (x.id === d.id ? { ...x, fired: true } : x)) };
    if (!thread) continue;
    const postId = `ta-${d.id}`;
    const ms = d.sentMs + TA_REPLY_AFTER_MS;
    const post: ThreadPost = {
      id: postId,
      threadId: d.threadId,
      authorId: STAFF.ta.id,
      author: STAFF.ta.name,
      role: "ta",
      body: taReplyFor(`${thread.title}\n${thread.question.body}`, thread.topic, thread.week),
      ms,
      quoteOf: d.quoteOf,
    };
    next = { ...next, posts: [...next.posts, post] };
    notes.push({
      id: `note-${d.id}`,
      to: { roles: ["student"], studentId: d.studentId },
      title: `${STAFF.ta.name} đã trả lời trong «${thread.title}»`,
      meta: "Threads",
      href: `/threads/${d.threadId}#post-${postId}`,
      ms,
    });
  }
  return { live: next, notes };
}

/** Còn bài AI đang soạn / đang chảy chữ trong thread này? (khoá hai nút ở ô soạn) */
export function aiBusy(thread: ThreadView, nowMs: number): boolean {
  return [thread.mainAi, ...thread.discussion].some((p) => {
    const a = p?.ai;
    if (!p || !a || a.startedMs === undefined || a.stoppedAt !== undefined) return false;
    return nowMs - a.startedMs < AI_TYPING_MS + aiStreamMs(p.body);
  });
}
