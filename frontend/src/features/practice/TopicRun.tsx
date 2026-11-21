"use client";

import { Check } from "lucide-react";
import { useEffect, useRef, useState } from "react";
import { ApiError, ApiErrorNotice } from "@/shared/data";
import { Markdown } from "@/shared/domain";
import { useMinWidth } from "@/shared/lib/useMinWidth";
import { Button, InlineNotice, Page, PageHeader, Section, Split } from "@/shared/ui";
import { EssayFeedbackBlock, ExplainMore } from "./Extras";
import { QuestionInput, filled } from "./QuestionInput";
import { answerKeyText, newIdempotencyKey, patchIndex, putAnswer, submitAttempt, type Ans, type PAttempt, type PFeedback, type PItem } from "./practiceApi";
import s from "./Practice.module.css";

const draftStore = (key: string) => ({
  read: (): Ans | undefined => { try { const v = localStorage.getItem(key); return v ? (JSON.parse(v) as Ans) : undefined; } catch { return undefined; } },
  write: (v: Ans | undefined) => { try { if (v) localStorage.setItem(key, JSON.stringify(v)); else localStorage.removeItem(key); } catch { /* kho đầy: giữ trong bộ nhớ */ } },
});

/** Luyện theo chủ đề (US-P9-06): mỗi lần một câu, trả lời xong có phản hồi ngay; không đồng hồ. Lựa chọn chưa gửi giữ trên máy (AC10); mất mạng → "Chưa gửi được" rồi tự gửi lại (AC4). */
export function TopicRun({ courseId, initial, tab, userId, onDone }: { courseId: string; initial: PAttempt; tab: string; userId: string; onDone: (a: PAttempt) => void }) {
  const [items, setItems] = useState<PItem[]>(initial.items);
  const firstOpen = items.findIndex((i) => !i.answer);
  const [index, setIndex] = useState(() => (firstOpen < 0 ? items.length - 1 : firstOpen)); // lượt dở: tới câu chưa trả lời đầu tiên
  const item = items[index];
  const store = (id: string) => draftStore(`ep:draft:${userId}:practice:${initial.id}:${id}`);
  const [draft, setDraft] = useState<Ans | undefined>(() => (item.answer ? undefined : store(item.id).read()));
  const [busy, setBusy] = useState(false);
  const [unsent, setUnsent] = useState(false); // mất mạng khi gửi: chờ tự gửi lại
  const [err, setErr] = useState<ApiError | null>(null);
  const submitKey = useRef<string | null>(null);
  const wide = useMinWidth(1024);
  const answered = item.answer !== null;
  const fb: PFeedback | null = item.feedback;
  const rightCount = items.filter((i) => i.feedback?.correct === true).length;
  const last = index === items.length - 1;
  const essay = item.question.type === "ESSAY";
  const allDone = items.every((i) => i.answer);

  function choose(v: Ans) {
    setDraft(v);
    store(item.id).write(v);
  }

  async function check() {
    if (!draft) return;
    setBusy(true);
    setErr(null);
    try {
      const r = await putAnswer(courseId, initial.id, item.id, tab, { answer: draft });
      setItems((prev) => prev.map((x) => (x.id === item.id ? { ...x, answer: { answer: draft as Record<string, unknown>, version: r.version, locked: r.locked }, feedback: r.feedback } : x)));
      setUnsent(false);
      store(item.id).write(undefined);
    } catch (e) {
      const ae = e instanceof ApiError ? e : new ApiError({ status: 0, code: "NETWORK" });
      if (ae.code === "NETWORK") setUnsent(true);
      else setErr(ae);
    } finally {
      setBusy(false);
    }
  }

  // Chưa gửi được: gửi lại khi có mạng (sự kiện `online`) và mỗi 5 s phòng sự kiện không tới; hàm gửi luôn là bản mới nhất.
  const retry = useRef(check);
  useEffect(() => { retry.current = check; });
  useEffect(() => {
    if (!unsent) return;
    const go = () => { if (navigator.onLine) void retry.current(); };
    window.addEventListener("online", go);
    const t = window.setInterval(go, 5000);
    return () => { window.removeEventListener("online", go); window.clearInterval(t); };
  }, [unsent]);

  function goTo(i: number) {
    setIndex(i);
    setDraft(items[i].answer ? undefined : store(items[i].id).read());
    setUnsent(false);
    setErr(null);
  }

  function next() {
    const i = Math.min(items.length - 1, index + 1);
    goTo(i);
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

  const nav = (
    <ol className={s.topicNav} aria-label="Danh sách câu">
      {items.map((it, i) => (
        <li key={it.id}>
          <button type="button" className={[s.topicNavBtn, i === index ? s.navCurrent : ""].join(" ")} aria-current={i === index ? "step" : undefined} aria-label={`Câu ${i + 1}${it.answer ? ", đã làm" : ""}`} disabled={i > (firstOpen < 0 ? items.length : firstOpen)} onClick={() => goTo(i)}>
            {it.answer ? <Check aria-hidden /> : i + 1}
          </button>
        </li>
      ))}
    </ol>
  );

  return (
    <Page kind="read">
      <PageHeader
        title={initial.topic ?? "Luyện tập"}
        back={{ href: "/practice", label: "Thoát về Luyện đề" }}
        meta={[<span key="m0">Câu {index + 1}/{items.length}</span>, <span key="m1">Đúng {rightCount}</span>]}
      />
      <Section>
        <Split ratio="main-aside" aside={wide ? <aside aria-label="Danh sách câu">{nav}</aside> : null} main={
        <div className={s.runBody}>
        <div className={s.progress} aria-hidden><span className={s.progressFill} style={{ "--p": index / items.length } as React.CSSProperties} /></div>
        <div className={s.question}><Markdown source={item.question.stem} /></div>
        <QuestionInput q={item.question} value={answered ? (item.answer!.answer as Ans) : draft} onChange={choose} disabled={answered || busy} fb={fb} />

        {answered && !essay && fb && (
          <div className={fb.correct ? s.feedbackRight : s.feedbackWrong}>
            <p className={s.feedbackHead}>{fb.correct ? "Đúng rồi" : "Chưa đúng"}</p>
            {!fb.correct && <p className={s.reviewLine}>Đáp án đúng: {answerKeyText(item)}</p>}
            {fb.explanation && <p className={s.explain}>{fb.explanation}</p>}
            {fb.citations.length > 0 && <p className={s.cite}>Nguồn trong tài liệu{fb.citations.some((c) => c.page) ? ` · trang ${fb.citations.flatMap((c) => (c.page ? [c.page] : [])).join(", ")}` : ""}</p>}
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

        {unsent && <InlineNotice tone="warning" compact>Chưa gửi được. Bài vẫn giữ trên máy bạn.</InlineNotice>}
        {err && <ApiErrorNotice error={err} />}
        <div className={`${s.rowActions} ${s.actionBar}`}>
          {!answered ? (
            <Button variant="primary" loading={busy} disabled={!filled(item.question, draft)} onClick={() => void check()}>{essay ? "Trả lời" : "Kiểm tra"}</Button>
          ) : last || allDone ? (
            <Button variant="primary" loading={busy} onClick={() => void finish()}>Xem kết quả</Button>
          ) : (
            <Button variant="primary" onClick={next}>Câu tiếp theo</Button>
          )}
        </div>
        </div>
        } />
      </Section>
    </Page>
  );
}
