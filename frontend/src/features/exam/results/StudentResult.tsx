"use client";

import { useQuery } from "@tanstack/react-query";
import { useState } from "react";
import { ApiErrorNotice, apiClient, useIdempotentMutation } from "@/shared/data";
import { Button, Field, InlineNotice, PanelSection, Skeleton, Textarea } from "@/shared/ui";
import { fmtWhen } from "../examApi";
import { ResultItems } from "./ResultItems";
import { resPath, vnum, type StudentResult } from "./resultsApi";
import s from "./Results.module.css";

/** Kết quả của sinh viên SAU công bố (`/exams/[id]/take`): điểm, từng câu (đáp án của bạn / đáp án đúng / giải thích nếu bài cho xem), test mẫu, "Test ẩn: đạt 3 trên 4", mã của bạn; `Gửi yêu cầu xem lại` khi còn hạn. */
export function StudentResultView({ course, exam, attempt }: { course: string; exam: string; attempt: string }) {
  const [asking, setAsking] = useState(false);
  const q = useQuery({ queryKey: ["exam-my-result", course, exam, attempt], queryFn: async ({ signal }) => (await apiClient.get<StudentResult>(`${resPath(course, exam)}/attempts/${attempt}/result`, { signal })).data });
  const [nowMs] = useState(() => Date.now());
  const res = q.data;
  if (q.isError && !res) return <ApiErrorNotice error={q.error} onRetry={() => void q.refetch()} />;
  if (!res) return <Skeleton lines={6} />;
  const e = res.exam;
  const canAppeal = res.appeal.status === null && e.appeal_open_until !== null && Date.parse(e.appeal_open_until) >= nowMs;
  return (
    <div className={s.drawer} data-part="student-result">
      <PanelSection tone="strong">
        <section aria-label="Điểm">
          <p className={s.big} data-part="final-score">{vnum(res.score)} / {vnum(e.max_score)}</p>
          {res.score_adjusted && <p className={s.muted}>Điểm đã được giảng viên điều chỉnh.</p>}
        </section>
      </PanelSection>
      {!asking && <Appeal res={res} canAppeal={canAppeal} onAsk={() => setAsking(true)} />}
      <ResultItems items={res.items} />
      {asking && <AppealForm course={course} exam={exam} attempt={attempt} onClose={() => setAsking(false)} onSent={() => { setAsking(false); void q.refetch(); }} />}
    </div>
  );
}

function Appeal({ res, canAppeal, onAsk }: { res: StudentResult; canAppeal: boolean; onAsk: () => void }) {
  const a = res.appeal;
  if (a.status === "OPEN") return <InlineNotice>Yêu cầu xem lại của bạn đã gửi, giảng viên sẽ trả lời.</InlineNotice>;
  if (a.status) return <InlineNotice tone="success" title={a.status === "ADJUSTED" ? "Giảng viên đã sửa điểm của bạn." : "Giảng viên giữ nguyên điểm."}>{a.response}</InlineNotice>;
  if (!canAppeal) return null;
  return (
    <div className={s.bar}>
      <Button onClick={onAsk}>Gửi yêu cầu xem lại</Button>
      {res.exam.appeal_open_until && <span className={s.muted}>Còn hạn đến {fmtWhen(res.exam.appeal_open_until)}.</span>}
    </div>
  );
}

/** Mở dần tại chỗ (không hộp thoại): ô lý do + `Gửi yêu cầu`. Mỗi bài chỉ một yêu cầu. */
function AppealForm({ course, exam, attempt, onClose, onSent }: { course: string; exam: string; attempt: string; onClose: () => void; onSent: () => void }) {
  const [reason, setReason] = useState("");
  const send = useIdempotentMutation<string, unknown>((r, key) => apiClient.post(`${resPath(course, exam)}/attempts/${attempt}/appeal`, { reason: r }, { idempotencyKey: key }));
  return (
    <section className={s.form} aria-label="Gửi yêu cầu xem lại" data-part="appeal-form">
      <p className={s.muted}>Mỗi bài chỉ gửi được một yêu cầu. Giảng viên sẽ xem lại và trả lời.</p>
      <Field label="Bạn muốn giảng viên xem lại điều gì? (tối đa 1.000 ký tự)">{(id, by) => <Textarea id={id} aria-describedby={by} rows={4} maxLength={1000} value={reason} onChange={(ev) => setReason(ev.target.value)} />}</Field>
      {send.error && <ApiErrorNotice error={send.error} />}
      <div className={s.bar}>
        <Button variant="primary" loading={send.pending} disabled={reason.trim() === ""} onClick={() => void send.mutate(reason.trim()).then(onSent, () => undefined)}>Gửi yêu cầu</Button>
        <Button onClick={onClose} disabled={send.pending}>Để sau</Button>
      </div>
    </section>
  );
}
