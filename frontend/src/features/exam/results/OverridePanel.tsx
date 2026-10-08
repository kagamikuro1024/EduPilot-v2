"use client";

import { useState } from "react";
import { ApiErrorNotice, apiClient } from "@/shared/data";
import { Button, ConfirmIrreversible, Field, InlineNotice, Select, Textarea } from "@/shared/ui";
import { resPath, type ResultItem } from "./resultsApi";
import s from "./Results.module.css";

/** Đổi đáp án đúng / huỷ một câu TRẮC NGHIỆM sau khi bài đóng (Giảng viên): mọi sinh viên được tính lại. Câu code dùng `Chấm lại`. */
export function OverridePanel({ course, exam, items, onDone }: { course: string; exam: string; items: ResultItem[]; onDone: () => void }) {
  const mcq = items.filter((i) => i.type !== "CODE");
  const [item, setItem] = useState(mcq[0]?.item_id ?? "");
  const [mode, setMode] = useState<"void" | "key">("void");
  const [key, setKey] = useState<string[]>([]);
  const [reason, setReason] = useState("");
  const [ask, setAsk] = useState(false);
  const [busy, setBusy] = useState(false);
  const [err, setErr] = useState<unknown>(null);
  const [done, setDone] = useState<string | null>(null);
  if (mcq.length === 0) return null;
  const cur = mcq.find((i) => i.item_id === item) ?? mcq[0];
  const isTF = cur.type === "TRUE_FALSE";
  const single = cur.type !== "MCQ_MULTI";
  async function go() {
    setBusy(true);
    setErr(null);
    try {
      const body = mode === "void" ? { void: true, reason: reason.trim() } : { answer_key: isTF ? { value: key[0] === "true" } : { option_ids: key }, reason: reason.trim() };
      const r = await apiClient.put<{ recomputed: number; job_id: string | null }>(`${resPath(course, exam)}/items/${cur.item_id}/override`, body);
      setDone(r.data.job_id ? "Đang tính lại điểm ở nền." : `Đã tính lại điểm của ${r.data.recomputed} sinh viên.`);
      setAsk(false);
      setReason("");
      onDone();
    } catch (e) {
      setErr(e);
    } finally {
      setBusy(false);
    }
  }
  const ready = reason.trim() !== "" && (mode === "void" || key.length > 0);
  return (
    <details className={s.sub} data-part="override-panel">
      <summary className={s.subHead}>Đổi đáp án hoặc huỷ một câu trắc nghiệm</summary>
      <div className={s.form}>
        <Field label="Câu">{(id, by) => <Select id={id} aria-describedby={by} value={cur.item_id} onChange={(ev) => { setItem(ev.target.value); setKey([]); }}>{mcq.map((i) => <option key={i.item_id} value={i.item_id}>Câu {i.position}</option>)}</Select>}</Field>
        <Field label="Cách điều chỉnh">{(id, by) => <Select id={id} aria-describedby={by} value={mode} onChange={(ev) => setMode(ev.target.value as "void" | "key")}><option value="void">Huỷ câu — mọi sinh viên được trọn điểm câu này</option><option value="key">Đổi đáp án đúng</option></Select>}</Field>
        {mode === "key" && (
          <Field label={single ? "Đáp án đúng mới" : "Các đáp án đúng mới"}>
            {(id, by) =>
              isTF ? (
                <Select id={id} aria-describedby={by} value={key[0] ?? ""} onChange={(ev) => setKey(ev.target.value ? [ev.target.value] : [])}><option value="">Chọn…</option><option value="true">Đúng</option><option value="false">Sai</option></Select>
              ) : single ? (
                <Select id={id} aria-describedby={by} value={key[0] ?? ""} onChange={(ev) => setKey(ev.target.value ? [ev.target.value] : [])}><option value="">Chọn…</option>{cur.options.map((o) => <option key={o.id} value={o.id}>{o.body}</option>)}</Select>
              ) : (
                <Select id={id} aria-describedby={by} multiple value={key} onChange={(ev) => setKey(Array.from(ev.target.selectedOptions, (o) => o.value))}>{cur.options.map((o) => <option key={o.id} value={o.id}>{o.body}</option>)}</Select>
              )
            }
          </Field>
        )}
        <Field label="Lý do (bắt buộc, tối đa 500 ký tự)">{(id, by) => <Textarea id={id} aria-describedby={by} rows={2} maxLength={500} value={reason} onChange={(ev) => setReason(ev.target.value)} />}</Field>
        {done && <InlineNotice tone="success" compact>{done}</InlineNotice>}
        {err != null && !ask && <ApiErrorNotice error={err} />}
        <div><Button onClick={() => setAsk(true)} disabled={!ready}>Tính lại điểm</Button></div>
      </div>
      <ConfirmIrreversible
        open={ask}
        onClose={() => setAsk(false)}
        onConfirm={() => void go()}
        title="Tính lại điểm cả lớp?"
        consequence={mode === "void" ? `Câu ${cur.position} bị huỷ: mọi sinh viên được trọn điểm câu này và điểm cả lớp được tính lại.` : `Đổi đáp án đúng của câu ${cur.position}: điểm của mọi sinh viên được tính lại.`}
        confirmLabel="Tính lại"
        loading={busy}
        error={err ? "Chưa tính lại được, thử lại." : undefined}
      />
    </details>
  );
}
