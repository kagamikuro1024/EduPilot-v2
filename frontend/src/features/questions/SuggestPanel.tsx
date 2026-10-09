"use client";

import { useQueryClient } from "@tanstack/react-query";
import { Sparkles } from "lucide-react";
import { useState } from "react";
import { ApiErrorNotice, apiClient, fieldErrors, useJob } from "@/shared/data";
import { Button, Field, InlineNotice, Input, Select, Textarea } from "@/shared/ui";
import { DIFF_LABEL, qPath, type Difficulty } from "./questionsApi";
import s from "./Questions.module.css";

/** `Gợi ý từ AI` (D57): AI chỉ soạn NHÁP câu trắc nghiệm; câu vào ngân hàng ở trạng thái "Chờ duyệt", chỉ vào bài thi sau khi giảng viên / TA duyệt. */
export function SuggestPanel({ courseId, onDone, onCancel }: { courseId: string; onDone: (created: number) => void; onCancel: () => void }) {
  const qc = useQueryClient();
  const [topic, setTopic] = useState("");
  const [difficulty, setDifficulty] = useState<Difficulty>("MEDIUM");
  const [count, setCount] = useState("5");
  const [source, setSource] = useState("");
  const [jobId, setJobId] = useState<string | null>(null);
  const [err, setErr] = useState<unknown>(null);
  const job = useJob(jobId);
  const fe = fieldErrors(err);
  const running = jobId !== null && job.status !== "SUCCEEDED" && job.status !== "FAILED";
  const [finished, setFinished] = useState(false);

  if (job.status === "SUCCEEDED" && !finished) {
    setFinished(true);
    const res = job.result as { created?: string[] } | undefined;
    void qc.invalidateQueries({ queryKey: ["questions", courseId] });
    onDone(res?.created?.length ?? 0);
  }

  async function start(e: React.FormEvent) {
    e.preventDefault();
    setErr(null);
    setFinished(false);
    try {
      const r = await apiClient.post<{ job_id: string }>(`${qPath(courseId)}/suggest`, { kind: "MCQ", topic, difficulty, count: Number(count), ...(source.trim() ? { source_text: source } : {}) });
      setJobId(r.data.job_id);
    } catch (x) {
      setErr(x);
    }
  }
  return (
    <form className={s.gen} onSubmit={start} noValidate aria-label="Gợi ý câu hỏi từ AI">
      <div className={s.genRow}>
        <Field label="Chủ đề" required error={fe.topic} className={s.genField}>{(id, by) => <Input id={id} aria-describedby={by} invalid={Boolean(fe.topic)} value={topic} maxLength={80} onChange={(e) => setTopic(e.target.value)} />}</Field>
        <Field label="Độ khó">{(id) => <Select id={id} value={difficulty} onChange={(e) => setDifficulty(e.target.value as Difficulty)}>{(Object.keys(DIFF_LABEL) as Difficulty[]).map((k) => <option key={k} value={k}>{DIFF_LABEL[k]}</option>)}</Select>}</Field>
        <Field label="Số câu" error={fe.count} helper="1–10">{(id, by) => <Input id={id} aria-describedby={by} inputMode="numeric" value={count} onChange={(e) => setCount(e.target.value)} invalid={Boolean(fe.count)} />}</Field>
      </div>
      <Field label="Văn bản nguồn (tuỳ chọn)" helper="Dán đoạn tài liệu để AI chỉ dựa vào đó (tối đa 6.000 ký tự). Không dán tên hay thông tin của sinh viên." error={fe.source_text}>
        {(id, by) => <Textarea id={id} aria-describedby={by} rows={4} value={source} onChange={(e) => setSource(e.target.value)} />}
      </Field>
      {err !== null && Object.keys(fe).length === 0 && <ApiErrorNotice error={err} showTechnical />}
      {running && (
        <div role="status">
          <div className={s.bar}><span className={s.barFill} style={{ width: `${job.progress}%` }} /></div>
          <p className={s.genText}>AI đang soạn nháp… {job.progress}%</p>
        </div>
      )}
      {job.status === "FAILED" && <InlineNotice tone="danger" title={job.error ?? "Chưa soạn được câu hỏi, thử lại sau."} />}
      <div className={s.formActions}>
        <Button type="submit" variant="primary" icon={<Sparkles aria-hidden />} loading={running}>Soạn nháp</Button>
        <Button type="button" variant="text" onClick={onCancel}>Đóng</Button>
      </div>
    </form>
  );
}
