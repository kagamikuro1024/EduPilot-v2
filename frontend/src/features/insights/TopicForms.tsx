"use client";

import Link from "next/link";
import { useState } from "react";
import { ApiError, apiClient, useAutosaveDraft } from "@/shared/data";
import { ApiErrorNotice } from "@/shared/data/ApiErrorNotice";
import { useEventMutations, fromLocalInput } from "@/features/calendar/calendarApi";
import { Button, Field, Input, Textarea } from "@/shared/ui";
import type { Topic } from "./insightsApi";
import s from "./insights.module.css";

type Created = { thread: { id: string } };

/** Nháp thread ghim (mở tại chỗ dưới chủ đề): tiêu đề + nội dung từ khuyến nghị; giảng viên sửa rồi bấm `Đăng`. Không chứa câu hỏi mẫu. */
export function PinThreadForm({ courseId, topic, onClose }: { courseId: string; topic: Topic; onClose: () => void }) {
  const [title, setTitle] = useState(`Chủ đề đang vướng: ${topic.name}`);
  const draft = useAutosaveDraft(`insight-thread:${courseId}:${topic.rank}`);
  const body = draft.value || (topic.recommendation ? `Nhiều bạn đang hỏi về “${topic.name}”. ${topic.recommendation}` : `Nhiều bạn đang hỏi về “${topic.name}”.`);
  const [pending, setPending] = useState(false);
  const [error, setError] = useState<unknown>(null);
  const [threadId, setThreadId] = useState<string | null>(null); // đã đăng; còn thiếu bước ghim
  const [done, setDone] = useState(false);

  async function post() {
    setPending(true);
    setError(null);
    try {
      let id = threadId;
      if (!id) {
        const r = await apiClient.post<Created>(`/courses/${courseId}/threads`, { title: title.trim(), body: body.trim(), tags: [], week_no: topic.week });
        id = r.data.thread.id;
        setThreadId(id);
      }
      await apiClient.post(`/courses/${courseId}/threads/${id}/pin`);
      draft.clear();
      setDone(true);
    } catch (e) {
      setError(e instanceof ApiError ? e : new ApiError({ status: 0, code: "NETWORK" }));
    } finally {
      setPending(false);
    }
  }

  if (done) {
    return (
      <div className={s.inline} role="group" aria-label="Tạo thread ghim">
        <p>Đã đăng và ghim thread.</p>
        <div className={s.inlineActions}>
          <Link href="/threads" className="ep-link">Xem ở Hỏi đáp công khai</Link>
          <Button size="sm" onClick={onClose}>Xong</Button>
        </div>
      </div>
    );
  }
  return (
    <div className={s.inline} role="group" aria-label="Tạo thread ghim">
      <Field label="Tiêu đề">{(id) => <Input id={id} value={title} maxLength={200} onChange={(e) => setTitle(e.target.value)} />}</Field>
      <Field label="Nội dung">{(id) => <Textarea id={id} rows={6} value={body} onChange={(e) => draft.setValue(e.target.value)} />}</Field>
      {error != null && <ApiErrorNotice error={error} />}
      <div className={s.inlineActions}>
        <Button size="sm" variant="primary" loading={pending} disabled={pending || !title.trim() || !body.trim()} onClick={() => void post()}>{threadId ? "Ghim lại" : "Đăng"}</Button>
        <Button size="sm" onClick={onClose} disabled={pending}>Huỷ</Button>
      </div>
    </div>
  );
}

/** Buổi ôn tập: sự kiện lịch loại OTHER của lớp; ngày giờ do giảng viên chọn. */
export function ReviewForm({ courseId, topic, onClose }: { courseId: string; topic: Topic; onClose: () => void }) {
  const [title, setTitle] = useState(`Ôn tập: ${topic.name}`);
  const [starts, setStarts] = useState("");
  const [ends, setEnds] = useState("");
  const [where, setWhere] = useState("");
  const [done, setDone] = useState(false);
  const { create } = useEventMutations(courseId);
  const ok = title.trim() !== "" && starts !== "";

  if (done) {
    return (
      <div className={s.inline} role="group" aria-label="Tạo buổi ôn tập">
        <p>Đã thêm buổi ôn tập vào lịch của lớp.</p>
        <div className={s.inlineActions}>
          <Link href="/calendar" className="ep-link">Xem ở Lịch</Link>
          <Button size="sm" onClick={onClose}>Xong</Button>
        </div>
      </div>
    );
  }
  return (
    <div className={s.inline} role="group" aria-label="Tạo buổi ôn tập">
      <Field label="Tên buổi">{(id) => <Input id={id} value={title} maxLength={120} onChange={(e) => setTitle(e.target.value)} />}</Field>
      <Field label="Bắt đầu">{(id) => <Input id={id} type="datetime-local" value={starts} onChange={(e) => setStarts(e.target.value)} />}</Field>
      <Field label="Kết thúc">{(id) => <Input id={id} type="datetime-local" value={ends} onChange={(e) => setEnds(e.target.value)} />}</Field>
      <Field label="Địa điểm">{(id) => <Input id={id} value={where} maxLength={80} onChange={(e) => setWhere(e.target.value)} />}</Field>
      {create.error != null && <ApiErrorNotice error={create.error} />}
      <div className={s.inlineActions}>
        <Button
          size="sm"
          variant="primary"
          loading={create.isPending}
          disabled={!ok || create.isPending}
          onClick={() => create.mutate({ type: "OTHER", title: title.trim(), starts_at: fromLocalInput(starts), ends_at: ends ? fromLocalInput(ends) : null, location: where.trim() || null }, { onSuccess: () => setDone(true) })}
        >
          Lưu
        </Button>
        <Button size="sm" onClick={onClose} disabled={create.isPending}>Huỷ</Button>
      </div>
    </div>
  );
}
