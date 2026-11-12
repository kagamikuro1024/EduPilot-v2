"use client";

import { useRef, useState } from "react";
import { ApiError, ApiErrorNotice } from "@/shared/data";
import { Markdown } from "@/shared/domain";
import { Button, Page, PageHeader, Section } from "@/shared/ui";
import { EssayFeedbackBlock, ExplainMore } from "./Extras";
import { QuestionInput, filled } from "./QuestionInput";
import { answerKeyText, newIdempotencyKey, patchIndex, putAnswer, submitAttempt, type Ans, type PAttempt, type PFeedback, type PItem } from "./practiceApi";
import s from "./Practice.module.css";

/** Luyện theo chủ đề (US-P9-06): mỗi lần một câu, trả lời xong có phản hồi ngay; không đồng hồ. */
export function TopicRun({ courseId, initial, tab, onDone }: { courseId: string; initial: PAttempt; tab: string; onDone: (a: PAttempt) => void }) {
  const [items, setItems] = useState<PItem[]>(initial.items);
  const firstOpen = items.findIndex((i) => !i.answer);
  const [index, setIndex] = useState(() => (firstOpen < 0 ? items.length - 1 : firstOpen)); // lượt dở: tới câu chưa trả lời đầu tiên
  const [draft, setDraft] = useState<Ans | undefined>();
  const [busy, setBusy] = useState(false);
  const [err, setErr] = useState<ApiError | null>(null);
  const submitKey = useRef<string | null>(null);
  const item = items[index];
  const answered = item.answer !== null;
  const fb: PFeedback | null = item.feedback;
  const rightCount = items.filter((i) => i.feedback?.correct === true).length;
  const last = index === items.length - 1;
  const essay = item.question.type === "ESSAY";
  const allDone = items.every((i) => i.answer);

  async function check() {
    if (!draft) return;
    setBusy(true);
    setErr(null);
    try {
      const r = await putAnswer(courseId, initial.id, item.id, tab, { answer: draft });
      setItems((prev) => prev.map((x) => (x.id === item.id ? { ...x, answer: { answer: draft as Record<string, unknown>, version: r.version, locked: r.locked }, feedback: r.feedback } : x)));
    } catch (e) {
      setErr(e instanceof ApiError ? e : new ApiError({ status: 0, code: "NETWORK" }));
    } finally {
      setBusy(false);
    }
  }

  function next() {
    const i = Math.min(items.length - 1, index + 1);
    setIndex(i);
    setDraft(undefined);
    setErr(null);
    void patchIndex(courseId, initial.id, tab, i).catch(() => undefined); // vị trí chỉ để quay lại đúng câu; mất cũng không sao
  }

  async function finish() {
    setBusy(true);
    setErr(null);
    submitKey.current ??= newIdempotencyKey();
    try {
      onDone(await submitAttempt(courseId, initial.id, submitKey.current));
    } catch (e) {
      setErr(e instanceof ApiError ? e : new ApiError({ status: 0, code: "NETWORK" }));
      setBusy(false);
    }
  }

  return (
    <Page>
      <PageHeader
        title={initial.topic ?? "Luyện tập"}
        back={{ href: "/practice", label: "Thoát về Luyện đề" }}
        meta={<><span>Câu {index + 1}/{items.length}</span><span>Đúng {rightCount}</span></>}
      />
      <Section panel>
        <div className={s.runBody}>
        <div className={s.progress} aria-hidden><span className={s.progressFill} style={{ "--p": index / items.length } as React.CSSProperties} /></div>
        <div className={s.question}><Markdown source={item.question.stem} /></div>
        <QuestionInput q={item.question} value={answered ? (item.answer!.answer as Ans) : draft} onChange={setDraft} disabled={answered || busy} fb={fb} />

        {answered && !essay && fb && (
          <div className={fb.correct ? s.feedbackRight : s.feedbackWrong}>
            <p className={s.feedbackHead}>{fb.correct ? "Đúng rồi" : "Chưa đúng"}</p>
            {!fb.correct && <p className={s.reviewLine}>Đáp án đúng: {answerKeyText(item)}</p>}
            {fb.explanation && <p className={s.explain}>{fb.explanation}</p>}
            {fb.citations.length > 0 && <p className={s.cite}>Nguồn: {fb.citations.map((c) => (c.page ? `trang ${c.page}` : "tài liệu")).join(", ")}</p>}
            <ExplainMore key={item.id} course={courseId} attempt={initial.id} item={item.id} />
          </div>
        )}
        {answered && essay && fb && (
          <div className={s.modelBox}>
            <h3>Đáp án mẫu</h3>
            <p>{(fb.key.model_answer as string | undefined) ?? ""}</p>
            <h3>Rubric</h3>
            <p>{(fb.key.rubric as string | undefined) ?? ""}</p>
            <EssayFeedbackBlock key={item.id} course={courseId} attempt={initial.id} item={item.id} />
          </div>
        )}

        {err && <ApiErrorNotice error={err} />}
        <div className={s.rowActions}>
          {!answered ? (
            <Button variant="primary" loading={busy} disabled={!filled(item.question, draft)} onClick={() => void check()}>{essay ? "Trả lời" : "Kiểm tra"}</Button>
          ) : last || allDone ? (
            <Button variant="primary" loading={busy} onClick={() => void finish()}>Xem kết quả</Button>
          ) : (
            <Button variant="primary" onClick={next}>Câu tiếp theo</Button>
          )}
        </div>
        </div>
      </Section>
    </Page>
  );
}
