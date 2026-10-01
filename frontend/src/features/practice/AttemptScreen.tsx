"use client";

import { Check, X } from "lucide-react";
import { useEffect, useState } from "react";
import { QUIZ_KEY, QUIZ_SEED, attemptById, type Question, type QuizState } from "@/mock/practice";
import { fmtTime } from "@/mock/core";
import { NOW } from "@/mock/core";
import { useDemoSlice } from "@/shared/state/demo";
import {
  Button,
  ButtonLink,
  ConfirmIrreversible,
  EmptyState,
  InlineNotice,
  Page,
  PageHeader,
  PageState,
  Section,
  Skeleton,
  StatusText,
  Textarea,
} from "@/shared/ui";
import s from "./Practice.module.css";

/** Làm bài: luyện theo chủ đề (có phản hồi ngay) hoặc bài tính điểm (có giờ) — DESIGN §14.19. */
export function AttemptScreen({ attemptId }: { attemptId: string }) {
  const attempt = attemptById(attemptId);
  if (!attempt) {
    return (
      <Page>
        <PageHeader title="Làm bài" back={{ href: "/practice", label: "Luyện đề" }} />
        <EmptyState title="Không tìm thấy lượt luyện này" action={<ButtonLink href="/practice" variant="primary">Về Luyện đề</ButtonLink>}>
          Lượt có thể đã kết thúc. Bạn có thể bắt đầu một lượt mới theo chủ đề.
        </EmptyState>
      </Page>
    );
  }
  if (attempt.mode === "quiz") return <QuizRun title={attempt.title} questions={attempt.questions} minutes={attempt.minutes ?? 20} />;
  if (attempt.mode === "review") return <ReviewRun title={attempt.title} questions={attempt.questions} answers={attempt.answers ?? []} />;
  return <TopicRun title={attempt.title} topic={attempt.topic} questions={attempt.questions} />;
}

// ---- luyện theo chủ đề ----------------------------------------------------------------------

function TopicRun({ title, topic, questions }: { title: string; topic: string; questions: Question[] }) {
  const [index, setIndex] = useState(0);
  const [picked, setPicked] = useState<number | null>(null);
  const [typed, setTyped] = useState("");
  const [checked, setChecked] = useState(false);
  const [right, setRight] = useState(0);
  const q = questions[index];
  const last = index === questions.length - 1;
  const correct = q.kind === "choice" ? picked === q.correct : (q.keywords ?? []).some((k) => typed.toLowerCase().includes(k));

  function check() {
    setChecked(true);
    if (correct) setRight((n) => n + 1);
  }

  function next() {
    setIndex((i) => i + 1);
    setPicked(null);
    setTyped("");
    setChecked(false);
  }

  return (
    <Page>
      <PageHeader
        title={title}
        back={{ href: "/practice", label: "Thoát về Luyện đề" }}
        meta={
          <>
            <span>
              Câu {index + 1}/{questions.length}
            </span>
            <span>Đúng {right}</span>
            <span>{topic}</span>
          </>
        }
      />
      <PageState loading={<Skeleton lines={6} />} empty={<EmptyState title="Lượt này chưa có câu hỏi">Chọn một chủ đề khác ở màn Luyện đề.</EmptyState>}>
        {index >= questions.length ? (
          <Section title="Xong lượt này">
            <p className={s.result}>
              Bạn đúng {right}/{questions.length} câu.
            </p>
            <div className={s.rowActions}>
              <ButtonLink href="/practice" variant="primary">
                Luyện chủ đề khác
              </ButtonLink>
              <ButtonLink href="/practice/history">Xem lịch sử luyện tập</ButtonLink>
            </div>
          </Section>
        ) : (
          <Section>
            <div className={s.progress} aria-hidden>
              <span className={s.progressFill} style={{ width: `${(index / questions.length) * 100}%` }} />
            </div>
            <p className={s.question}>{q.text}</p>

            {q.kind === "choice" ? (
              <ul className={s.options}>
                {(q.options ?? []).map((opt, i) => {
                  const state = !checked ? "" : i === q.correct ? s.optRight : i === picked ? s.optWrong : "";
                  return (
                    <li key={opt}>
                      <button
                        type="button"
                        className={[s.option, picked === i ? s.optPicked : "", state].join(" ")}
                        aria-pressed={picked === i}
                        disabled={checked}
                        onClick={() => setPicked(i)}
                      >
                        <span className={s.optMark} aria-hidden>
                          {String.fromCharCode(65 + i)}
                        </span>
                        {opt}
                        {checked && i === q.correct && <Check className={s.optIcon} aria-hidden />}
                        {checked && i === picked && i !== q.correct && <X className={s.optIcon} aria-hidden />}
                      </button>
                    </li>
                  );
                })}
              </ul>
            ) : (
              <Textarea
                value={typed}
                rows={3}
                aria-label="Câu trả lời ngắn"
                placeholder="Trả lời ngắn gọn trong 1–2 câu"
                disabled={checked}
                onChange={(e) => setTyped(e.target.value)}
              />
            )}

            {checked && (
              <div className={correct ? s.feedbackRight : s.feedbackWrong}>
                <p className={s.feedbackHead}>{correct ? "Đúng rồi" : "Chưa đúng"}</p>
                <p className={s.explain}>{q.explain}</p>
                <p className={s.cite}>
                  Nguồn: {q.source.title} · {q.source.locator}
                </p>
              </div>
            )}

            <div className={s.rowActions}>
              {checked ? (
                <Button variant="primary" onClick={next}>
                  {last ? "Xem kết quả" : "Câu tiếp theo"}
                </Button>
              ) : (
                <Button variant="primary" disabled={q.kind === "choice" ? picked === null : typed.trim().length < 3} onClick={check}>
                  Kiểm tra
                </Button>
              )}
            </div>
          </Section>
        )}
      </PageState>
    </Page>
  );
}

// ---- bài tính điểm (QUIZ01) -------------------------------------------------------------------

function QuizRun({ title, questions, minutes }: { title: string; questions: Question[]; minutes: number }) {
  const [quiz, setQuiz] = useDemoSlice<QuizState>(QUIZ_KEY, QUIZ_SEED);
  const [index, setIndex] = useState(0);
  const [left, setLeft] = useState(minutes * 60);
  const [endAt] = useState(() => Date.now() + minutes * 60_000);
  const [confirm, setConfirm] = useState(false);
  const [timedOut, setTimedOut] = useState(false);
  const done = quiz.status === "submitted";

  useEffect(() => {
    if (quiz.status === "idle") setQuiz({ status: "doing", answers: {} });
  }, [quiz.status, setQuiz]);

  function submit(auto: boolean) {
    setQuiz((prev) => ({ status: "submitted", answers: prev.answers, score: questions.filter((q) => prev.answers[q.id] === q.correct).length }));
    setTimedOut(auto);
  }

  // Hết giờ thì tự nộp ngay trong nhịp đếm (không đặt setState trong thân effect).
  useEffect(() => {
    if (done) return;
    const t = window.setInterval(() => {
      setLeft((v) => Math.max(0, v - 1));
      if (Date.now() >= endAt) {
        window.clearInterval(t);
        submit(true);
      }
    }, 1000);
    return () => window.clearInterval(t);
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [done, endAt]);

  const q = questions[index];
  const answered = Object.keys(quiz.answers).length;
  const mm = Math.floor(left / 60).toString().padStart(2, "0");
  const ss = (left % 60).toString().padStart(2, "0");

  if (done) {
    return (
      <Page>
        <PageHeader title={title} back={{ href: "/practice", label: "Luyện đề" }} />
        <Section title="Đã nộp bài">
          {timedOut && (
            <InlineNotice tone="warning" title="Hết giờ — bài đã được nộp tự động">
              Bài nộp lúc {fmtTime(NOW)}. Các câu chưa trả lời được tính là bỏ trống.
            </InlineNotice>
          )}
          <p className={s.result}>
            Bạn đúng {quiz.score}/{questions.length} câu. Điểm chính thức do giảng viên công bố sau khi bài đóng.
          </p>
          <div className={s.rowActions}>
            <ButtonLink href="/" variant="primary">
              Về Hôm nay
            </ButtonLink>
            <ButtonLink href="/practice/at-symmetric">Luyện lại chủ đề này</ButtonLink>
          </div>
        </Section>
      </Page>
    );
  }

  return (
    <Page>
      <PageHeader
        title={title}
        back={{ href: "/practice", label: "Thoát (bài vẫn giữ nguyên)" }}
        meta={
          <>
            <span>
              Câu {index + 1}/{questions.length}
            </span>
            <span>Đã trả lời {answered}</span>
            <StatusText tone={left < 120 ? "red" : "neutral"}>
              Còn {mm}:{ss}
            </StatusText>
          </>
        }
      />
      <PageState loading={<Skeleton lines={6} />} empty={<EmptyState title="Bài này chưa có câu hỏi">Liên hệ giảng viên nếu bạn thấy màn hình này.</EmptyState>}>
        <Section>
          <InlineNotice compact>Trong lúc làm bài, Chat riêng chỉ trả lời câu hỏi thủ tục. Bài không chấm từng câu cho tới khi bạn nộp.</InlineNotice>
          <div className={s.progress} aria-hidden>
            <span className={s.progressFill} style={{ width: `${(answered / questions.length) * 100}%` }} />
          </div>
          <p className={s.question}>{q.text}</p>
          <ul className={s.options}>
            {(q.options ?? []).map((opt, i) => (
              <li key={opt}>
                <button
                  type="button"
                  className={[s.option, quiz.answers[q.id] === i ? s.optPicked : ""].join(" ")}
                  aria-pressed={quiz.answers[q.id] === i}
                  onClick={() => setQuiz({ ...quiz, status: "doing", answers: { ...quiz.answers, [q.id]: i } })}
                >
                  <span className={s.optMark} aria-hidden>
                    {String.fromCharCode(65 + i)}
                  </span>
                  {opt}
                </button>
              </li>
            ))}
          </ul>
          <div className={s.rowActions}>
            <Button disabled={index === 0} onClick={() => setIndex((i) => i - 1)}>
              Câu trước
            </Button>
            {index < questions.length - 1 ? (
              <Button onClick={() => setIndex((i) => i + 1)}>Câu sau</Button>
            ) : (
              <span className={s.quizEnd}>Câu cuối</span>
            )}
            <Button variant="primary" onClick={() => setConfirm(true)}>
              Nộp bài
            </Button>
          </div>
        </Section>
      </PageState>
      <ConfirmIrreversible
        open={confirm}
        onClose={() => setConfirm(false)}
        onConfirm={() => submit(false)}
        title="Nộp bài kiểm tra?"
        consequence={`Bạn đã trả lời ${answered}/${questions.length} câu. Bài chỉ nộp được một lần, nộp xong không sửa được.`}
        confirmLabel="Nộp bài"
        cancelLabel="Làm tiếp"
      />
    </Page>
  );
}

// ---- xem lại lượt đã làm ----------------------------------------------------------------------

function ReviewRun({ title, questions, answers }: { title: string; questions: Question[]; answers: number[] }) {
  const right = questions.filter((q, i) => answers[i] === q.correct).length;
  return (
    <Page>
      <PageHeader
        title={title}
        back={{ href: "/practice/history", label: "Lịch sử luyện tập" }}
        description={`Xem lại lượt đã làm · đúng ${right}/${questions.length} câu`}
        actions={
          <ButtonLink href="/practice/at-symmetric" variant="primary">
            Luyện lại chủ đề này
          </ButtonLink>
        }
      />
      <PageState loading={<Skeleton lines={8} />} empty={<EmptyState title="Lượt này không còn dữ liệu">Chọn một lượt khác trong lịch sử luyện tập.</EmptyState>}>
        <ol className={s.review}>
          {questions.map((q, i) => {
            const ok = answers[i] === q.correct;
            return (
              <li key={q.id} className={s.reviewItem}>
                <div className={s.reviewHead}>
                  <span className={s.reviewNo}>Câu {i + 1}</span>
                  <StatusText tone={ok ? "green" : "red"}>{ok ? "Đúng" : "Sai"}</StatusText>
                </div>
                <p className={s.question}>{q.text}</p>
                <p className={s.reviewLine}>Bạn chọn: {q.options?.[answers[i]] ?? "Không trả lời"}</p>
                {!ok && <p className={s.reviewLine}>Đáp án đúng: {q.options?.[q.correct ?? 0]}</p>}
                <p className={s.explain}>{q.explain}</p>
                <p className={s.cite}>
                  Nguồn: {q.source.title} · {q.source.locator}
                </p>
              </li>
            );
          })}
        </ol>
      </PageState>
    </Page>
  );
}
