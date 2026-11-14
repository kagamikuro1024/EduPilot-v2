"use client";

import { Page, PageHeader, Section, ActionList, ActionRow, BarList, Button, ButtonLink, InlineNotice, StatusText } from "@/shared/ui";
import { Markdown } from "@/shared/domain";
import { EssayFeedbackBlock, ExplainMore } from "./Extras";
import { answerKeyText, givenText, type PAttempt, type PItem } from "./practiceApi";
import { } from "@/features/exam/examApi";
import s from "./Practice.module.css";

const sc = (v: string) => v.replace(".", ",");

/** Kết quả một lượt đã đóng (TOPIC hoặc MOCK): `đúng/tổng` là kết quả chính, điểm thang 10 chỉ là phụ (không vào sổ điểm, không xếp hạng). */
export function PracticeResult({ courseId, attempt, onAgain }: { courseId: string; attempt: PAttempt; onAgain?: () => void }) {
  const r = attempt.result;
  const wrong = attempt.items.filter((i) => i.feedback?.correct === false);
  const essays = attempt.items.filter((i) => i.question.type === "ESSAY" && i.answer);
  const total = r?.total ?? attempt.items.length;
  const right = r?.correct ?? 0;
  const title = total > 0 && right === total ? `Bạn đúng cả ${total} câu` : `Bạn đúng ${right}/${total} câu`;
  const name = attempt.mode === "MOCK" ? "Thi thử" : attempt.topic;
  return (
    <Page kind="read">
      <PageHeader
        title={title}
        back={{ href: "/practice", label: "Luyện đề" }}
      />
      {r?.expired && <InlineNotice tone="warning" compact>Hết giờ — bài được nộp theo các câu đã lưu.</InlineNotice>}
      {attempt.notes.length > 0 && <InlineNotice compact>{attempt.notes.join(" ")}</InlineNotice>}

      {r && r.topics.length > 1 && (
        <Section title="Theo chủ đề">
          <div className={s.weak}>
            <BarList items={r.topics.map((t) => ({ label: `${t.topic} · ${t.correct}/${t.total}`, value: Math.round((t.correct / t.total) * 100), tone: t.correct / t.total < 0.5 ? "red" : "ink" }))} max={100} format={(v) => `${v}% đúng`} />
          </div>
        </Section>
      )}

      {wrong.length > 0 && (
        <Section title={`Câu cần ôn (${wrong.length})`}>
          <ol className={s.review}>
            {wrong.map((it) => (
              <Review key={it.id} courseId={courseId} attempt={attempt.id} it={it} no={it.position + 1} />
            ))}
          </ol>
        </Section>
      )}

      {essays.length > 0 && (
        <Section title="Tự luận">
          <ol className={s.review}>
            {essays.map((it) => (
              <li key={it.id} className={s.reviewItem}>
                <div className={s.reviewHead}><span className={s.reviewNo}>Câu {it.position + 1}</span></div>
                <div className={s.question}><Markdown source={it.question.stem} /></div>
                <p className={s.reviewLine}>Bài của bạn: {givenText(it)}</p>
                <div className={s.modelBox}>
                  <h3>Đáp án mẫu</h3>
                  <p>{(it.feedback?.key.model_answer as string | undefined) ?? ""}</p>
                  <h3>Rubric</h3>
                  <p>{(it.feedback?.key.rubric as string | undefined) ?? ""}</p>
                </div>
                <EssayFeedbackBlock course={courseId} attempt={attempt.id} item={it.id} />
              </li>
            ))}
          </ol>
        </Section>
      )}

      <Section>
        <div className={s.rowActions}>
          {r?.weakest_topic && wrong.length > 0 ? <ButtonLink href={`/practice?topic=${encodeURIComponent(r.weakest_topic)}`} variant="primary">Luyện {r.weakest_topic}</ButtonLink> : <ButtonLink href="/practice" variant="primary">Luyện đề</ButtonLink>}
          <ButtonLink href="/practice/history">Lịch sử luyện tập</ButtonLink>
          {onAgain && <Button onClick={onAgain}>Làm lại</Button>}
        </div>
      </Section>
    </Page>
  );
}

function Review({ courseId, attempt, it, no }: { courseId: string; attempt: string; it: PItem; no: number }) {
  const fb = it.feedback;
  return (
    <li className={s.reviewItem}>
      <div className={s.reviewHead}>
        <span className={s.reviewNo}>Câu {no}</span>
        <StatusText tone="red">Sai</StatusText>
      </div>
      <div className={s.question}><Markdown source={it.question.stem} /></div>
      <p className={s.reviewLine}>Bạn chọn: {givenText(it)}</p>
      <p className={s.reviewLine}>Đáp án đúng: {answerKeyText(it)}</p>
      {fb?.explanation && <p className={s.explain}>{fb.explanation}</p>}
      {fb && fb.citations.length > 0 && (
        <ActionList label="Nguồn">
          {fb.citations.map((c, i) => (c.document_id ? <ActionRow key={i} href={`/library/${c.document_id}`} title="Nguồn tham khảo" context={c.page ? `Trang ${c.page}` : undefined} /> : null))}
        </ActionList>
      )}
      <ExplainMore course={courseId} attempt={attempt} item={it.id} />
    </li>
  );
}
