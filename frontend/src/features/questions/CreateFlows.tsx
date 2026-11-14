"use client";

import { useQueryClient } from "@tanstack/react-query";
import { useState } from "react";
import { ApiErrorNotice, apiClient, fieldErrors, useIdempotentMutation, useJob } from "@/shared/data";
import { useDocList } from "@/features/documents/documentsApi";
import { progressStyle } from "@/shared/lib/progress";
import { Button, Checkbox, Field, InlineNotice, Input, Select } from "@/shared/ui";
import { DIFF_LABEL, qPath, type Difficulty } from "./questionsApi";
import s from "./Questions.module.css";

export type JobOutcome = { created: string[]; skipped: Array<{ number: number; page: number; reason: string }> };
const REASON: Record<string, string> = {
  NO_ANSWER: "Không thấy đáp án",
  NO_ANSWER_MATCH: "Đáp án không khớp số câu",
  DUPLICATE: "Đã có trong ngân hàng",
  INVALID: "Câu chưa đủ hoặc sai định dạng",
  CITATION_REQUIRED: "Thiếu nguồn trích",
  TYPE_NOT_ALLOWED: "Loại câu không được chọn",
  EXTRA: "Vượt số câu yêu cầu",
};

/** Tiến độ + kết quả một việc nền (trích / sinh): tiến độ ngay trong luồng, không modal; "bỏ qua N câu" mở ra lý do từng câu. */
function JobView({ jobId, onDone, label }: { jobId: string; onDone: (n: number) => void; label: string }) {
  const qc = useQueryClient();
  const job = useJob(jobId);
  const [done, setDone] = useState(false);
  const running = job.status !== "SUCCEEDED" && job.status !== "FAILED";
  const out = job.result as JobOutcome | undefined;
  if (job.status === "SUCCEEDED" && !done) {
    setDone(true);
    void qc.invalidateQueries({ queryKey: ["questions"] });
    onDone(out?.created?.length ?? 0);
  }
  return (
    <div role="status">
      {running && (
        <>
          <div className={s.bar}><span className={s.barFill} style={progressStyle(job.progress)} /></div>
          <p className={s.genText}>{label} {job.progress}%</p>
        </>
      )}
      {job.status === "FAILED" && <InlineNotice tone="danger" title={job.error ?? "Chưa chạy được. Thử lại."} />}
      {out && (
        <>
          <p className={s.genText}>Đã tạo {out.created.length} câu chờ duyệt{out.skipped.length > 0 ? ` · bỏ qua ${out.skipped.length} câu` : ""}</p>
          {out.skipped.length > 0 && (
            <details className={s.skipped}>
              <summary>Xem lý do</summary>
              <ul>{out.skipped.map((k, i) => <li key={i}>{k.number > 0 ? `Câu ${k.number}${k.page ? ` (trang ${k.page})` : ""}` : "Một câu"}: {REASON[k.reason] ?? k.reason}</li>)}</ul>
            </details>
          )}
        </>
      )}
    </div>
  );
}

/** `Trích từ đề`: chọn đề cũ (EXAM_PAPER) + tuỳ chọn tài liệu đáp án; việc chạy nền, câu vào ngân hàng ở trạng thái Chờ duyệt. */
export function ExtractPanel({ courseId, onDone, onCancel }: { courseId: string; onDone: (n: number) => void; onCancel: () => void }) {
  const papers = useDocList(courseId, { type: "EXAM_PAPER", status: "READY", q: "" });
  const keys = useDocList(courseId, { type: "ANSWER_KEY", status: "READY", q: "" });
  const [paper, setPaper] = useState("");
  const [key, setKey] = useState("");
  const [jobId, setJobId] = useState<string | null>(null);
  const run = useIdempotentMutation<{ document_id: string; answer_key_document_id: string | null }, { job_id: string }>((b, k) => apiClient.post(`${qPath(courseId)}/extract`, b, { idempotencyKey: k }));
  const fe = fieldErrors(run.error);
  async function start(e: React.FormEvent) {
    e.preventDefault();
    try {
      setJobId((await run.mutate({ document_id: paper, answer_key_document_id: key || null })).data.job_id);
    } catch {
      /* lỗi hiện dưới form */
    }
  }
  return (
    <form className={s.gen} onSubmit={start} noValidate aria-label="Trích câu hỏi từ đề cũ">
      <div className={s.genRow}>
        <Field label="Đề cũ" required error={fe.document_id} className={s.genField}>
          {(id) => (
            <Select id={id} value={paper} onChange={(e) => setPaper(e.target.value)}>
              <option value="">Chọn đề</option>
              {papers.items.map((d) => <option key={d.id} value={d.id}>{d.title}</option>)}
            </Select>
          )}
        </Field>
        <Field label="Đáp án" className={s.genField}>
          {(id) => (
            <Select id={id} value={key} onChange={(e) => setKey(e.target.value)}>
              <option value="">Không có</option>
              {keys.items.map((d) => <option key={d.id} value={d.id}>{d.title}</option>)}
            </Select>
          )}
        </Field>
      </div>
      {papers.isSuccess && papers.items.length === 0 && <InlineNotice compact>Chưa có đề nào sẵn sàng. Tải đề lên ở Tài liệu với loại Đề tham khảo.</InlineNotice>}
      {run.error && Object.keys(fe).length === 0 && <ApiErrorNotice error={run.error} onRetry={() => void run.retry().then((r) => r && setJobId(r.data.job_id), () => undefined)} />}
      {jobId && <JobView jobId={jobId} label="Đang trích câu hỏi…" onDone={onDone} />}
      <div className={s.formActions}>
        <Button type="submit" variant="primary" loading={run.pending} disabled={!paper || (jobId !== null)}>Trích câu hỏi</Button>
        <Button type="button" variant="text" onClick={onCancel}>Đóng</Button>
      </div>
    </form>
  );
}

const GEN_TYPES: Array<[string, string]> = [["MCQ_SINGLE", "Một đáp án"], ["MCQ_MULTI", "Nhiều đáp án"], ["TRUE_FALSE", "Đúng–sai"]];

/** `Sinh từ tài liệu`: câu nháp có trích dẫn tới tài liệu nguồn; chỉ vào bài luyện sau khi duyệt. */
export function GeneratePanel({ courseId, onDone, onCancel }: { courseId: string; onDone: (n: number) => void; onCancel: () => void }) {
  const lectures = useDocList(courseId, { type: "LECTURE", status: "READY", q: "" });
  const [doc, setDoc] = useState("");
  const [topic, setTopic] = useState("");
  const [difficulty, setDifficulty] = useState<Difficulty>("MEDIUM");
  const [count, setCount] = useState("5");
  const [types, setTypes] = useState<string[]>(["MCQ_SINGLE"]);
  const [jobId, setJobId] = useState<string | null>(null);
  const run = useIdempotentMutation<Record<string, unknown>, { job_id: string }>((b, k) => apiClient.post(`${qPath(courseId)}/generate`, b, { idempotencyKey: k }));
  const fe = fieldErrors(run.error);
  const body = { document_ids: [doc], topic: topic.trim(), difficulty, count: Number(count), types };
  async function start(e: React.FormEvent) {
    e.preventDefault();
    try {
      setJobId((await run.mutate(body)).data.job_id);
    } catch {
      /* lỗi hiện dưới form */
    }
  }
  return (
    <form className={s.gen} onSubmit={start} noValidate aria-label="Sinh câu hỏi từ tài liệu">
      <div className={s.genRow}>
        <Field label="Tài liệu" required error={fe.document_ids} className={s.genField}>
          {(id) => (
            <Select id={id} value={doc} onChange={(e) => setDoc(e.target.value)}>
              <option value="">Chọn tài liệu</option>
              {lectures.items.map((d) => <option key={d.id} value={d.id}>{d.title}</option>)}
            </Select>
          )}
        </Field>
        <Field label="Chủ đề" required error={fe.topic} className={s.genField}>{(id, by) => <Input id={id} aria-describedby={by} invalid={Boolean(fe.topic)} value={topic} maxLength={80} onChange={(e) => setTopic(e.target.value)} />}</Field>
      </div>
      <div className={s.genRow}>
        <Field label="Độ khó">{(id) => <Select id={id} value={difficulty} onChange={(e) => setDifficulty(e.target.value as Difficulty)}>{(Object.keys(DIFF_LABEL) as Difficulty[]).map((k) => <option key={k} value={k}>{DIFF_LABEL[k]}</option>)}</Select>}</Field>
        <Field label="Số câu" error={fe.count}>{(id, by) => <Input id={id} aria-describedby={by} inputMode="numeric" placeholder="1–10" value={count} invalid={Boolean(fe.count)} onChange={(e) => setCount(e.target.value)} />}</Field>
        <fieldset className={s.genTypes}>
          <legend className={s.legend}>Loại câu</legend>
          {GEN_TYPES.map(([v, label]) => (
            <Checkbox key={v} label={label} checked={types.includes(v)} onChange={(e) => setTypes((p) => (e.target.checked ? [...p, v] : p.filter((x) => x !== v)))} />
          ))}
        </fieldset>
      </div>
      {run.error && Object.keys(fe).length === 0 && <ApiErrorNotice error={run.error} onRetry={() => void run.retry().then((r) => r && setJobId(r.data.job_id), () => undefined)} />}
      {jobId && <JobView jobId={jobId} label="Đang soạn câu hỏi…" onDone={onDone} />}
      <div className={s.formActions}>
        <Button type="submit" variant="primary" loading={run.pending} disabled={!doc || topic.trim() === "" || types.length === 0 || jobId !== null}>Sinh câu hỏi</Button>
        <Button type="button" variant="text" onClick={onCancel}>Đóng</Button>
      </div>
    </form>
  );
}
