"use client";

import { useState } from "react";
import { ApiError, apiClient } from "@/shared/data";
import { Button, Field, InlineNotice, StatusText, Textarea } from "@/shared/ui";
import type { KnowledgeState, StaffTicket } from "./inboxApi";
import s from "./RealInbox.module.css";

type Preview = { question: string; answer: string; personal: boolean };

function errorLine(e: ApiError): string {
  if (e.code === "PII_DETECTED") return "Còn tên, MSSV, email hoặc số điện thoại. Hãy xoá hoặc thay.";
  if (e.code === "KNOWLEDGE_PERSONAL") return "Câu hỏi riêng tư, không lưu được.";
  return e.userMessage;
}

/**
 * `Lưu thành tri thức` (ticket đã trả lời / đã đóng): xem trước đã che thông tin cá nhân, sửa, lưu. Đã lưu: "Đã lưu" + `Gỡ`;
 * chưa nạp xong (embedding NULL): "Chưa nạp vào tri thức" + `Thử lại` (lưu lại đúng bản đã lưu).
 */
export function KnowledgeBox({ t, state, onChanged }: { t: StaffTicket; state?: KnowledgeState; onChanged: () => void }) {
  const base = `/courses/${t.course.id}/tickets/${t.id}`;
  const [form, setForm] = useState<Preview | null>(null);
  const [busy, setBusy] = useState(false);
  const [err, setErr] = useState<ApiError | null>(null);

  async function run(fn: () => Promise<void>) {
    if (busy) return;
    setBusy(true);
    setErr(null);
    try {
      await fn();
    } catch (e) {
      setErr(e instanceof ApiError ? e : new ApiError({ status: 0, code: "NETWORK" }));
    } finally {
      setBusy(false);
    }
  }
  const open = () =>
    run(async () => {
      const p = (await apiClient.get<Preview>(`${base}/knowledge-preview`)).data;
      if (p.personal) throw new ApiError({ status: 422, code: "KNOWLEDGE_PERSONAL" });
      setForm(p);
    });
  const save = (q: string, a: string) =>
    run(async () => {
      await apiClient.post(`${base}/knowledge`, { question: q, answer: a });
      setForm(null);
      onChanged();
    });
  const remove = () =>
    run(async () => {
      await apiClient.delete(`${base}/knowledge`);
      onChanged();
    });

  return (
    <div className={s.reply} data-part="knowledge">
      {!form && !state?.saved && <Button variant="secondary" size="sm" disabled={busy} onClick={() => void open()}>Lưu thành tri thức</Button>}
      {!form && state?.saved && (
        <div className={s.claimRow}>
          {state.loaded ? <StatusText tone="green">Đã lưu</StatusText> : <StatusText tone="amber">Chưa nạp vào tri thức</StatusText>}
          {!state.loaded && <Button variant="secondary" size="sm" disabled={busy || !state.question} onClick={() => void save(state.question ?? "", state.answer ?? "")}>Thử lại</Button>}
          <Button variant="text" size="sm" disabled={busy} onClick={() => void remove()}>Gỡ</Button>
        </div>
      )}
      {form && (
        <div className={s.knowledgeForm}>
          <Field label="Câu hỏi">{(id) => <Textarea id={id} rows={2} value={form.question} onChange={(e) => setForm({ ...form, question: e.target.value })} />}</Field>
          <Field label="Trả lời">{(id) => <Textarea id={id} rows={4} value={form.answer} onChange={(e) => setForm({ ...form, answer: e.target.value })} />}</Field>
          <div className={s.claimRow}>
            <Button variant="primary" size="sm" disabled={busy || !form.question.trim() || !form.answer.trim()} onClick={() => void save(form.question, form.answer)}>Lưu</Button>
            <Button variant="text" size="sm" disabled={busy} onClick={() => setForm(null)}>Huỷ</Button>
          </div>
        </div>
      )}
      {err && <InlineNotice tone="danger" compact>{errorLine(err)}</InlineNotice>}
    </div>
  );
}
