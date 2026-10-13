"use client";

import { useQueryClient } from "@tanstack/react-query";
import { useState } from "react";
import { ApiError, apiClient } from "@/shared/data";
import { useUndoLine } from "@/shared/lib/useUndoLine";
import { Button, ConfirmIrreversible, Field, InlineNotice, Input, OverflowMenu, Select, Textarea, type MenuItem } from "@/shared/ui";
import { REPORT_REASONS, threadsKey, type Hidden } from "./threadsApi";
import { usePostGate } from "./usePostGate";
import r from "./RealThreads.module.css";

export type Target = { kind: "thread" | "post"; id: string; version: number };
const NOTE_MAX = 200;

const pathOf = (courseId: string, t: Target) => (t.kind === "thread" ? `/courses/${courseId}/threads/${t.id}` : `/courses/${courseId}/posts/${t.id}`);
const asApiError = (e: unknown) => (e instanceof ApiError ? e : new ApiError({ status: 0, code: "NETWORK" }));

/** Dòng một câu cho bài đang ẩn: Staff thấy lý do; chủ bài thấy lời trấn an (tự ẩn) hoặc lý do (ẩn tay). */
export function HiddenNotice({ hidden, staff }: { hidden: Hidden; staff: boolean }) {
  const text = staff ? `Đã ẩn · ${hidden.reason}` : hidden.auto ? "Bài của bạn đang được giảng viên xem xét" : `Bài của bạn đã bị ẩn: ${hidden.reason}`;
  return <InlineNotice tone="warning" compact>{text}</InlineNotice>;
}

/**
 * Menu `…` của một bài (câu hỏi hoặc bình luận): Báo cáo cho người khác; Sửa / Xoá cho chủ bài; Ẩn bài / Bỏ ẩn / Giữ bài / Ghim cho Staff.
 * Báo cáo xong chỉ có dòng tĩnh "Đã báo cáo · Hoàn tác" 5 giây (không toast, không hộp xác nhận).
 */
export function PostMenu({
  courseId,
  target,
  mine,
  staff,
  hidden,
  pinned,
  canEdit,
  replyCount = 0,
  onEdit,
  onDeleted,
}: {
  courseId: string;
  target: Target;
  mine: boolean;
  staff: boolean;
  hidden?: Hidden | null;
  pinned?: boolean;
  canEdit: boolean;
  replyCount?: number;
  onEdit: () => void;
  onDeleted?: () => void;
}) {
  const qc = useQueryClient();
  const undo = useUndoLine();
  const base = pathOf(courseId, target);
  const [dialog, setDialog] = useState<"report" | "hide" | "delete" | null>(null);
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState<ApiError | null>(null);
  const [reason, setReason] = useState("SPAM");
  const [text, setText] = useState("");

  const close = () => {
    setDialog(null);
    setError(null);
    setText("");
  };
  const refresh = () => qc.invalidateQueries({ queryKey: threadsKey(courseId) });

  async function run(fn: () => Promise<unknown>, done?: () => void) {
    setBusy(true);
    setError(null);
    try {
      await fn();
      close();
      await refresh();
      done?.();
    } catch (e) {
      setError(asApiError(e));
    } finally {
      setBusy(false);
    }
  }

  const report = async () => {
    setBusy(true);
    setError(null);
    try {
      await apiClient.post(`${base}/report`, { reason, note: reason === "OTHER" && text.trim() ? text.trim() : null });
      close();
      undo.push("Đã báo cáo", () => void apiClient.delete(`${base}/report`).then(refresh, () => undefined));
      await refresh();
    } catch (e) {
      const err = asApiError(e);
      if (err.code === "ALREADY_REPORTED") {
        close();
        undo.push("Bạn đã báo cáo bài này");
      } else setError(err);
    } finally {
      setBusy(false);
    }
  };

  const items: MenuItem[] = [];
  if (!mine) items.push({ label: "Báo cáo", onSelect: () => setDialog("report") });
  if (mine && canEdit) items.push({ label: "Sửa", onSelect: onEdit });
  if (mine) items.push({ label: "Xoá", danger: true, onSelect: () => setDialog("delete") });
  if (staff) {
    if (hidden) items.push({ label: "Bỏ ẩn", onSelect: () => void run(() => apiClient.post(`${base}/unhide`, { version: target.version })) });
    else items.push({ label: "Ẩn bài", onSelect: () => setDialog("hide") });
    if (hidden?.auto) items.push({ label: "Giữ bài", onSelect: () => void run(() => apiClient.post(`${base}/dismiss-reports`, { version: target.version })) });
    if (target.kind === "thread") items.push({ label: pinned ? "Bỏ ghim" : "Ghim", onSelect: () => void run(() => apiClient.post(`/courses/${courseId}/threads/${target.id}/${pinned ? "unpin" : "pin"}`)) });
  }
  if (items.length === 0) return null;

  return (
    <>
      <OverflowMenu items={items} label="Tuỳ chọn bài" />
      {undo.node}
      {!dialog && error && <InlineNotice tone="danger" compact>{error.userMessage}</InlineNotice>}

      {dialog === "report" && (
        <form className={`${r.composer} ${r.wide}`} data-part="report-form" onSubmit={(e) => { e.preventDefault(); void report(); }}>
          <Field label="Lý do báo cáo">
            {(id) => (
              <Select id={id} value={reason} onChange={(e) => setReason(e.target.value)}>
                {REPORT_REASONS.map((x) => <option key={x.value} value={x.value}>{x.label}</option>)}
              </Select>
            )}
          </Field>
          {reason === "OTHER" && <Field label="Ghi chú">{(id) => <Textarea id={id} rows={3} maxLength={NOTE_MAX} value={text} onChange={(e) => setText(e.target.value)} />}</Field>}
          {error && <InlineNotice tone="danger" compact>{error.userMessage}</InlineNotice>}
          <div className={r.actions}>
            <Button type="submit" variant="primary" loading={busy}>Gửi báo cáo</Button>
            <Button variant="ghost" onClick={close}>Để sau</Button>
          </div>
        </form>
      )}

      {dialog === "hide" && (
        <form className={`${r.composer} ${r.wide}`} data-part="hide-form" onSubmit={(e) => { e.preventDefault(); if (text.trim()) void run(() => apiClient.post(`${base}/hide`, { reason: text.trim(), version: target.version })); }}>
          <Field label="Lý do ẩn" required>{(id) => <Textarea id={id} rows={3} maxLength={NOTE_MAX} value={text} onChange={(e) => setText(e.target.value)} />}</Field>
          {error && <InlineNotice tone="danger" compact>{error.userMessage}</InlineNotice>}
          <div className={r.actions}>
            <Button type="submit" variant="primary" loading={busy} disabled={!text.trim()}>Ẩn bài</Button>
            <Button variant="ghost" onClick={close}>Để sau</Button>
          </div>
        </form>
      )}

      <ConfirmIrreversible
        open={dialog === "delete"}
        onClose={close}
        title="Xoá bài này"
        consequence={target.kind === "thread" ? `Câu hỏi và ${replyCount} bình luận sẽ bị xoá.` : "Bình luận sẽ bị xoá."}
        confirmLabel="Xoá"
        loading={busy}
        error={error?.userMessage}
        onConfirm={() => void run(() => apiClient.delete(base), onDeleted)}
      />
    </>
  );
}

/** Sửa tại chỗ (chủ bài). Cùng tường lửa PII như lúc đăng: 422 → hộp hai lối `Chuyển sang chat riêng` / `Ẩn thông tin rồi lưu`. */
export function EditForm({
  courseId,
  target,
  title: title0,
  body: body0,
  canChat,
  onDone,
  onCancel,
}: {
  courseId: string;
  target: Target;
  title?: string;
  body: string;
  canChat: boolean;
  onDone: () => void;
  onCancel: () => void;
}) {
  const qc = useQueryClient();
  const [title, setTitle] = useState(title0 ?? "");
  const [body, setBody] = useState(body0);
  const isThread = target.kind === "thread";
  const gate = usePostGate<unknown>({
    courseId,
    path: pathOf(courseId, target),
    method: "PUT",
    title: isThread ? title : undefined,
    body,
    payload: isThread ? { title, body, version: target.version } : { body, version: target.version },
    canChat,
    redactLabel: "Ẩn thông tin rồi lưu",
    onPosted: () => {
      void qc.invalidateQueries({ queryKey: threadsKey(courseId) });
      onDone();
    },
    onSwitched: onCancel,
  });
  const empty = !body.trim() || (isThread && !title.trim());
  return (
    <form onSubmit={(e) => { e.preventDefault(); if (!empty) void gate.submit(false); }} className={r.composer} data-part="edit-form">
      {isThread && <Field label="Tiêu đề" required>{(id) => <Input id={id} value={title} maxLength={200} onChange={(e) => setTitle(e.target.value)} />}</Field>}
      <Field label="Nội dung" required>{(id) => <Textarea id={id} rows={4} value={body} onChange={(e) => setBody(e.target.value)} />}</Field>
      {gate.error && <InlineNotice tone="danger" compact>{gate.error.userMessage}</InlineNotice>}
      <div className={r.actions}>
        <Button type="submit" variant="primary" disabled={empty || gate.pending}>Lưu</Button>
        <Button variant="ghost" onClick={onCancel}>Huỷ</Button>
      </div>
      {gate.dialog}
    </form>
  );
}
