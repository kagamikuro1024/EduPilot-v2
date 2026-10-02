"use client";

import { Check, X } from "lucide-react";
import { useEffect, useMemo, useState } from "react";
import {
  HISTORY_KEY,
  HISTORY_SEED,
  quizKey,
  QUIZ_SEED,
  RUN_KEY,
  RUN_SEED,
  attemptById,
  isCorrect,
  newRun,
  questionsOf,
  scoreOf,
  type Answer,
  type Attempt,
  type Question,
  type QuizState,
  type Run,
  type RunRecord,
} from "@/mock/practice";
import { fmtTime } from "@/mock/core";
import { NOW } from "@/mock/core";
import { agoLabel } from "@/mock/derive";
import { simNowMs, useSimNow } from "@/shared/state/clock";
import { useSession } from "@/shared/session/session";
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
  const { studentId } = useSession();
  const [history] = useDemoSlice<RunRecord[]>(`${HISTORY_KEY}.${studentId}`, HISTORY_SEED);
  const attempt = attemptById(attemptId);
  const record = history.find((r) => r.id === attemptId);

  if (record) {
    return <ReviewRun title={`${record.topic} · ${record.qids.length} câu`} questions={questionsOf(record.qids)} answers={record.answers} />;
  }
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
  return <TopicRun attempt={attempt} />;
}

// ---- luyện theo chủ đề ----------------------------------------------------------------------

const HINT_ID = "practice-check-hint";

function TopicRun({ attempt }: { attempt: Attempt }) {
  const { studentId } = useSession();
  const [stored, setRun] = useDemoSlice<Run | null>(`${RUN_KEY}.${studentId}`, RUN_SEED);
  const [, setHistory] = useDemoSlice<RunRecord[]>(`${HISTORY_KEY}.${studentId}`, HISTORY_SEED);
  const now = useSimNow();
  const [bootMs] = useState(() => simNowMs());
  const [picked, setPicked] = useState<number | null>(null);
  const [typed, setTyped] = useState("");
  /** chỉ số câu vừa bấm `Kiểm tra` (đang xem phản hồi), `null` = đang làm câu chưa trả lời đầu tiên. */
  const [checked, setChecked] = useState<number | null>(null);

  const run = useMemo(() => (stored && stored.attemptId === attempt.id ? stored : newRun(attempt, bootMs)), [stored, attempt, bootMs]);
  const questions = useMemo(() => questionsOf(run.qids), [run.qids]);
  const open = run.answers.findIndex((a) => a === null);
  const index = checked ?? (open < 0 ? questions.length : open);
  const q: Question | undefined = questions[index];
  const right = questions.filter((item, i) => isCorrect(item, run.answers[i])).length;

  function check() {
    if (!q) return;
    const value: Answer = q.kind === "choice" ? picked : typed.trim();
    setRun({ ...run, answers: run.answers.map((a, k) => (k === index ? value : a)) });
    setChecked(index);
  }

  function next() {
    setChecked(null);
    setPicked(null);
    setTyped("");
  }

  /** Ghi lượt vào lịch sử ngay lúc bấm `Xem kết quả`, rồi chuyển màn kết quả cùng route. */
  function finish() {
    const ms = simNowMs();
    setHistory((prev) => [{ id: `run-${ms}`, topic: run.topic, qids: run.qids, answers: run.answers, score: scoreOf(run.qids, run.answers), atMs: ms }, ...prev]);
    setRun({ ...run, finishedMs: ms });
    next();
  }

  /** `Ôn lại n câu sai`: lượt mới chỉ gồm các câu còn sai. */
  function reviewWrong(ids: string[]) {
    setRun(newRun(attempt, simNowMs(), ids, `Ôn lại câu sai: ${attempt.topic}`));
    next();
  }

  if (run.finishedMs) {
    const wrong = questions.map((item, i) => ({ item, i })).filter(({ item, i }) => !isCorrect(item, run.answers[i]));
    const total = questions.length;
    return (
      <Page>
        <PageHeader
          title={wrong.length === 0 ? `Bạn đúng cả ${total} câu` : `Bạn đúng ${total - wrong.length}/${total} câu`}
          back={{ href: "/practice", label: "Luyện đề" }}
          description={`${run.topic} · ${total} câu · ${agoLabel(run.finishedMs, now)}`}
        />
        {wrong.length > 0 && (
          <Section title={`Câu cần ôn (${wrong.length})`}>
            <ol className={s.review}>
              {wrong.map(({ item, i }) => (
                <li key={item.id} className={s.reviewItem}>
                  <div className={s.reviewHead}>
                    <span className={s.reviewNo}>Câu {i + 1}</span>
                    <StatusText tone="red">Sai</StatusText>
                  </div>
                  <p className={s.question}>{item.text}</p>
                  <p className={s.reviewLine}>Bạn chọn: {answerText(item, run.answers[i])}</p>
                  <p className={s.reviewLine}>Đáp án đúng: {correctText(item)}</p>
                  <p className={s.explain}>{item.explain}</p>
                  <p className={s.cite}>
                    {item.source.title} · {item.source.locator}
                  </p>
                  <div className={s.rowActions}>
                    <ButtonLink href={item.source.href} size="sm">
                      Nguồn tham khảo
                    </ButtonLink>
                  </div>
                </li>
              ))}
            </ol>
          </Section>
        )}
        <Section>
          <div className={s.rowActions}>
            {wrong.length > 0 ? (
              <>
                <Button variant="primary" onClick={() => reviewWrong(wrong.map(({ item }) => item.id))}>
                  Ôn lại {wrong.length} câu sai
                </Button>
                <ButtonLink href="/practice">Về Luyện đề</ButtonLink>
              </>
            ) : (
              <>
                <ButtonLink href="/practice" variant="primary">
                  Luyện chủ đề khác
                </ButtonLink>
                <ButtonLink href="/practice/history">Xem lịch sử luyện tập</ButtonLink>
              </>
            )}
          </div>
        </Section>
      </Page>
    );
  }

  const missing = !q ? false : q.kind === "choice" ? picked === null : typed.trim().length < 3;
  const last = index === questions.length - 1;

  return (
    <Page>
      <PageHeader
        title={run.title}
        back={{ href: "/practice", label: "Thoát về Luyện đề" }}
        meta={
          <>
            <span>
              Câu {Math.min(index + 1, questions.length)}/{questions.length}
            </span>
            <span>Đúng {right}</span>
            <span>{run.topic}</span>
          </>
        }
      />
      <PageState loading={<Skeleton lines={6} />} empty={<EmptyState title="Lượt này chưa có câu hỏi">Chọn một chủ đề khác ở màn Luyện đề.</EmptyState>}>
        {!q ? (
          <Section title="Bạn đã trả lời hết các câu">
            <p className={s.result}>Xem kết quả để biết mình cần ôn lại câu nào.</p>
            <div className={s.rowActions}>
              <Button variant="primary" onClick={finish}>
                Xem kết quả
              </Button>
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
                  const state = checked === null ? "" : i === q.correct ? s.optRight : i === picked ? s.optWrong : "";
                  return (
                    <li key={opt}>
                      <button
                        type="button"
                        className={[s.option, picked === i ? s.optPicked : "", state].join(" ")}
                        aria-pressed={picked === i}
                        disabled={checked !== null}
                        onClick={() => setPicked(i)}
                      >
                        <span className={s.optMark} aria-hidden>
                          {String.fromCharCode(65 + i)}
                        </span>
                        {opt}
                        {checked !== null && i === q.correct && <Check className={s.optIcon} aria-hidden />}
                        {checked !== null && i === picked && i !== q.correct && <X className={s.optIcon} aria-hidden />}
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
                disabled={checked !== null}
                onChange={(e) => setTyped(e.target.value)}
              />
            )}

            {checked !== null && (
              <div className={isCorrect(q, run.answers[index]) ? s.feedbackRight : s.feedbackWrong}>
                <p className={s.feedbackHead}>{isCorrect(q, run.answers[index]) ? "Đúng rồi" : "Chưa đúng"}</p>
                {!isCorrect(q, run.answers[index]) && <p className={s.reviewLine}>Đáp án đúng: {correctText(q)}</p>}
                <p className={s.explain}>{q.explain}</p>
                <p className={s.cite}>
                  Nguồn: {q.source.title} · {q.source.locator}
                </p>
              </div>
            )}

            <div className={s.rowActions}>
              {checked !== null ? (
                <Button variant="primary" onClick={last ? finish : next}>
                  {last ? "Xem kết quả" : "Câu tiếp theo"}
                </Button>
              ) : (
                <>
                  <Button variant="primary" disabled={missing} aria-describedby={missing ? HINT_ID : undefined} onClick={check}>
                    Kiểm tra
                  </Button>
                  {missing && (
                    <p id={HINT_ID} className={s.hint}>
                      {q.kind === "choice" ? "Chọn một đáp án để kiểm tra" : "Nhập câu trả lời để kiểm tra"}
                    </p>
                  )}
                </>
              )}
            </div>
          </Section>
        )}
      </PageState>
    </Page>
  );
}

/** Chữ hiển thị cho đáp án người học đã đưa ra. */
function answerText(q: Question, given: Answer): string {
  if (typeof given === "number") return q.options?.[given] ?? "Không trả lời";
  const typed = (given ?? "").trim();
  return typed.length > 0 ? typed : "Không trả lời";
}

/** Chữ hiển thị cho đáp án đúng: lựa chọn đúng, hoặc đáp án mẫu của câu trả lời ngắn. */
function correctText(q: Question): string {
  return q.kind === "choice" ? (q.options?.[q.correct ?? 0] ?? "—") : (q.answer ?? q.explain);
}

// ---- bài tính điểm (QUIZ01) -------------------------------------------------------------------

function QuizRun({ title, questions, minutes }: { title: string; questions: Question[]; minutes: number }) {
  const { studentId } = useSession();
  const [quiz, setQuiz] = useDemoSlice<QuizState>(quizKey(studentId), QUIZ_SEED);
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

function ReviewRun({ title, questions, answers }: { title: string; questions: Question[]; answers: Answer[] }) {
  const right = questions.filter((q, i) => isCorrect(q, answers[i])).length;
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
            const ok = isCorrect(q, answers[i]);
            return (
              <li key={q.id} className={s.reviewItem}>
                <div className={s.reviewHead}>
                  <span className={s.reviewNo}>Câu {i + 1}</span>
                  <StatusText tone={ok ? "green" : "red"}>{ok ? "Đúng" : "Sai"}</StatusText>
                </div>
                <p className={s.question}>{q.text}</p>
                <p className={s.reviewLine}>Bạn chọn: {answerText(q, answers[i] ?? null)}</p>
                {!ok && <p className={s.reviewLine}>Đáp án đúng: {correctText(q)}</p>}
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
