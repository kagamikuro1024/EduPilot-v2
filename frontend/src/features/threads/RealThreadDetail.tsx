"use client";

import { useQueryClient } from "@tanstack/react-query";
import Link from "next/link";
import { useState } from "react";
import { ApiError, apiClient, useAutosaveDraft } from "@/shared/data";
import { CitationList, Markdown } from "@/shared/domain";
import { useSession } from "@/shared/session/session";
import { Button, ButtonLink, Field, InlineNotice, OverflowMenu, Page, PageHeader, PageState, Panel, Skeleton, StatusText, Textarea } from "@/shared/ui";
import { ANSWER_LABEL, SKIP_LABEL, confidenceLabel, describeReasons, threadKey, useSimilar, useThread, type ThreadPost } from "./threadsApi";
import { usePostGate } from "./usePostGate";
import r from "./RealThreads.module.css";

const WHEN = new Intl.DateTimeFormat("vi-VN", { day: "2-digit", month: "2-digit", hour: "2-digit", minute: "2-digit" });

/** Chi tiết thread thật (DESIGN §14.4): câu hỏi, bài AI kèm nguồn và nhãn xác nhận, bình luận, thread tương tự; Staff có Xác nhận / Chỉnh sửa / Loại cạnh bản nháp. */
export function RealThreadDetail({ courseId, id }: { courseId: string; id: string }) {
  const { role } = useSession();
  const q = useThread(courseId, id);
  const similar = useSimilar(courseId, id);
  const staff = role === "teacher" || role === "ta";
  return (
    <Page width="full">
      <PageHeader title={q.data?.thread.title ?? "Thread"} actions={<ButtonLink href="/threads" variant="ghost">Tất cả câu hỏi</ButtonLink>} />
      <PageState query={q} loading={<Panel><Skeleton lines={5} /></Panel>}>
        {q.data && (
          <div className={r.detail}>
            <Panel>
              <div className={r.question} data-part="thread-question">
                <div className={r.head}>
                  <span className={r.who}>{q.data.thread.author.full_name}</span>
                  <span>{WHEN.format(new Date(q.data.thread.created_at))}</span>
                  {q.data.thread.week_no !== null && <span>Tuần {q.data.thread.week_no}</span>}
                  {q.data.thread.tags.map((t) => <span key={t}>#{t}</span>)}
                </div>
                <p className={r.body}>{q.data.thread.body}</p>
                {staff && q.data.thread.ai_state === "SKIPPED" && <p className={r.collapsed}>AI chưa trả lời: {SKIP_LABEL[q.data.thread.ai_skip_reason ?? ""] ?? "không đủ tin cậy"}</p>}
              </div>
            </Panel>
            <Panel>
              {q.data.posts.length > 0 && (
                <ul className={r.posts} data-part="thread-posts">
                  {q.data.posts.map((p) => <li key={p.id}>{p.kind === "AI" ? <AiPost courseId={courseId} threadId={id} p={p} staff={staff} /> : <HumanPost p={p} />}</li>)}
                </ul>
              )}
              <Reply courseId={courseId} threadId={id} canChat={role === "student"} />
            </Panel>
            {similar.data && similar.data.length > 0 && (
              <Panel>
                <h2 className="ep-item-title">Câu hỏi tương tự</h2>
                <ul className={r.similar}>
                  {similar.data.map((s) => <li key={s.id}><Link href={`/threads/${s.id}`}>{s.title}</Link></li>)}
                </ul>
              </Panel>
            )}
          </div>
        )}
      </PageState>
    </Page>
  );
}

function HumanPost({ p }: { p: ThreadPost }) {
  return (
    <div className={r.post}>
      <div className={r.head}>
        <span className={r.who}>{p.author?.full_name}</span>
        {p.author && p.author.role !== "STUDENT" && <span>{p.author.role === "TEACHER" ? "Giảng viên" : p.author.role === "TA" ? "Trợ giảng" : ""}</span>}
        <span>{WHEN.format(new Date(p.created_at))}</span>
      </div>
      <p className={r.body}>{p.body}</p>
    </div>
  );
}

function AiPost({ courseId, threadId, p, staff }: { courseId: string; threadId: string; p: ThreadPost; staff: boolean }) {
  const qc = useQueryClient();
  const [editing, setEditing] = useState(false);
  const [text, setText] = useState(p.body);
  const [error, setError] = useState<ApiError | null>(null);
  const [busy, setBusy] = useState(false);
  const state = p.verification_state ?? "PENDING";
  const label = ANSWER_LABEL[state];
  const ok = state === "VERIFIED" || state === "CORRECTED";
  const base = `/courses/${courseId}/posts/${p.id}`;

  async function act(fn: () => Promise<unknown>) {
    setBusy(true);
    setError(null);
    try {
      await fn();
      setEditing(false);
      await qc.invalidateQueries({ queryKey: threadKey(courseId, threadId) });
    } catch (e) {
      setError(e instanceof ApiError ? e : new ApiError({ status: 0, code: "NETWORK" }));
    } finally {
      setBusy(false);
    }
  }

  if (p.rejected) {
    return <p className={r.collapsed}>Đã loại một câu trả lời AI.</p>;
  }
  return (
    <div className={[r.post, r.ai, ok ? r.aiOk : ""].join(" ")} data-part="ai-post">
      <div className={r.head}>
        <span className={r.who}>AI</span>
        <span className={ok ? r.ok : r.muted}>{label?.text}</span>
        {staff && p.confidence !== undefined && <span>Độ tin cậy {confidenceLabel(p.confidence)}</span>}
      </div>
      {editing ? (
        <>
          <Field label="Bản sửa">{(fid) => <Textarea id={fid} rows={6} value={text} onChange={(e) => setText(e.target.value)} />}</Field>
          <div className={r.actions}>
            <Button variant="primary" disabled={busy || !text.trim()} onClick={() => void act(() => apiClient.put(`${base}/correct`, { body: text, version: p.version }))}>Lưu bản sửa</Button>
            <Button variant="ghost" onClick={() => { setEditing(false); setText(p.body); }}>Huỷ</Button>
          </div>
        </>
      ) : (
        <Markdown source={p.body} />
      )}
      {p.citations.length > 0 && (
        <CitationList items={p.citations.map((c) => ({ id: String(c.n), title: c.title, page: c.page_no ?? undefined, excerpt: <>{c.snippet}{" "}<Link href={`/library/${c.document_id}`}>Xem tài liệu</Link></> }))} />
      )}
      {staff && !editing && (
        <div className={r.actions}>
          {state !== "VERIFIED" && <Button disabled={busy} onClick={() => void act(() => apiClient.post(`${base}/verify`))}>Xác nhận</Button>}
          <Button variant="ghost" disabled={busy} onClick={() => { setText(p.body); setEditing(true); }}>Chỉnh sửa</Button>
          <OverflowMenu items={[{ label: "Loại", danger: true, onSelect: () => void act(() => apiClient.post(`${base}/reject`)) }]} />
        </div>
      )}
      {error && <InlineNotice tone="danger" compact>{error.code === "VERSION_CONFLICT" ? "Có người vừa sửa câu trả lời này. Tải lại để xem bản mới." : error.userMessage}</InlineNotice>}
    </div>
  );
}

function Reply({ courseId, threadId, canChat }: { courseId: string; threadId: string; canChat: boolean }) {
  const qc = useQueryClient();
  const { identity } = useSession();
  const draft = useAutosaveDraft(`thread:${courseId}:${threadId}:reply`, { userId: identity?.sub });
  const gate = usePostGate<ThreadPost>({
    courseId,
    path: `/courses/${courseId}/threads/${threadId}/posts`,
    body: draft.value,
    payload: { body: draft.value },
    canChat,
    onPosted: () => {
      draft.clear();
      void qc.invalidateQueries({ queryKey: threadKey(courseId, threadId) });
    },
    onSwitched: () => draft.clear(),
  });
  return (
    <form className={r.composer} onSubmit={(e) => { e.preventDefault(); if (draft.value.trim()) void gate.submit(false); }} data-part="reply-form">
      <Field label="Bình luận">{(fid) => <Textarea id={fid} rows={3} value={draft.value} onChange={(e) => draft.setValue(e.target.value)} />}</Field>
      {gate.check && !gate.check.allowed && <p className={r.noticeLine} role="status" data-part="pii-notice">Phát hiện {describeReasons(gate.check.reasons)}</p>}
      {gate.error && <InlineNotice tone="danger" compact action={<Button size="sm" onClick={() => void gate.submit(false)}>Gửi lại</Button>}>{gate.error.userMessage}</InlineNotice>}
      <div className={r.actions}>
        <Button type="submit" variant="primary" disabled={!draft.value.trim() || gate.pending}>Đăng</Button>
        {draft.status === "saved" && <StatusText>Đã lưu nháp</StatusText>}
      </div>
      {gate.dialog}
    </form>
  );
}
