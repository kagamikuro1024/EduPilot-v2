"use client";

import { Check, Circle, ListChecks } from "lucide-react";
import { useCallback, useEffect, useRef, useState } from "react";
import { ApiError } from "@/shared/data";
import { ExamTimer, Markdown } from "@/shared/domain";
import type { ExamClock } from "@/shared/lib/examClock";
import type { Answer } from "@/shared/lib/saveQueue";
import { Button, ConfirmIrreversible, Drawer, InlineNotice, StatusText } from "@/shared/ui";
import { CodeQuestion } from "./CodeQuestion";
import { useAnswers } from "./useAnswers";
import { saveAnswers, sleep, submitAttempt, takeover, type Running, type SubmitSummary, type TakeItem } from "./takeApi";
import { useWriter } from "./useWriter";
import s from "./Take.module.css";

const LETTERS = "ABCDEFGH";
const hhmmss = (ms: number) => new Date(ms + 7 * 3600_000).toISOString().slice(11, 19);

const answeredOf = (it: TakeItem, a: Answer | undefined, code: Record<string, boolean>): boolean => {
  if (it.type === "CODE") return code[it.item_id] ?? false;
  const v = a ?? (it.answer as Answer | null);
  if (!v) return false;
  return "value" in v ? typeof v.value === "boolean" : v.option_ids.length > 0;
};

/**
 * Màn làm trắc nghiệm (US-PE-05 AC10–AC12): MỖI LẦN MỘT CÂU ở < 720 px, danh sách câu bên cạnh ≥ 1100 px; thanh trên `Câu i/n` · đồng hồ · trạng thái lưu; thanh dưới
 * `Câu trước` · `Câu sau` · `Danh sách câu`. Lựa chọn ghi vào máy ngay (hàng đợi `SaveQueue`) rồi gửi dần; mất mạng vẫn làm tiếp; chỉ MỘT tab là người ghi.
 */
export function TakeRunning({ courseId, examId, initial, tab, clock, userId, prevWriter, startedHere, onFinished, onExpired }: {
  courseId: string;
  examId: string;
  initial: Running;
  tab: string;
  clock: ExamClock;
  userId: string;
  prevWriter: boolean;
  startedHere: boolean;
  onFinished: (summary?: SubmitSummary) => void;
  /** đồng hồ về 0: nhờ trang chủ hỏi máy chủ xem bài đã được nộp chưa */
  onExpired: () => void;
}) {
  const { attempt, exam, items } = initial;
  const deadlineMs = Date.parse(attempt.deadline_at);
  const [index, setIndex] = useState(0);
  const [listOpen, setListOpen] = useState(false);
  const [confirm, setConfirm] = useState(false);
  const [submitting, setSubmitting] = useState(false);
  const [submitErr, setSubmitErr] = useState<ApiError | null>(null);
  const [warn, setWarn] = useState<1 | 5 | null>(null);
  const [expired, setExpired] = useState(false);
  const submitKey = useRef<string | null>(null);
  // câu code: "đã làm" khi có nháp không rỗng hoặc đã nộp (do CodeQuestion báo); nháp máy chủ trả lúc mở lượt tính ngay từ đầu
  const [codeAnswered, setCodeAnswered] = useState<Record<string, boolean>>(() =>
    Object.fromEntries(items.filter((it) => it.type === "CODE").map((it) => [it.item_id, Object.values((it.code as { drafts?: Record<string, { source: string }> } | null)?.drafts ?? {}).some((d) => d.source.trim() !== "")])),
  );
  const onAnswered = useCallback((id: string, v: boolean) => setCodeAnswered((p) => (p[id] === v ? p : { ...p, [id]: v })), []);
  // hàm gửi NGAY bản nháp của từng câu code: gọi trước khi nộp cả bài thi
  const flushers = useRef(new Set<() => Promise<void>>());
  const registerFlush = useCallback((fn: () => Promise<void>) => {
    flushers.current.add(fn);
    return () => void flushers.current.delete(fn);
  }, []);
  const hasCode = items.some((it) => it.type === "CODE");

  // quyền ghi
  const writer = useWriter({
    examId, attemptId: attempt.id, tab, initiallyYou: startedHere, prevWriter,
    takeover: async (reload) => void (await takeover(clock, courseId, examId, attempt.id, tab, reload)),
  });
  const canWrite = writer.state === "you";

  // hàng đợi lưu
  const sender = useCallback(async (batch: Array<{ item_id: string; answer: Answer }>) => {
    try {
      await saveAnswers(clock, courseId, examId, attempt.id, tab, batch);
      return { ok: true as const };
    } catch (e) {
      if (e instanceof ApiError) {
        if ((e.code as string) === "ATTEMPT_OTHER_TAB") {
          writer.lose();
          return { ok: false as const, fatal: true };
        }
        if ((e.code as string) === "ATTEMPT_CLOSED" || (e.code as string) === "ATTEMPT_ALREADY_SUBMITTED") {
          onExpired();
          return { ok: false as const, fatal: true };
        }
        if (e.status >= 400 && e.status < 500 && e.status !== 429 && e.status !== 408) return { ok: false as const, fatal: true };
      }
      return { ok: false as const }; // mạng / 5xx / quá tải: giữ và thử lại
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [clock, courseId, examId, attempt.id, tab]);
  const { queue, state: qs, answers, choose } = useAnswers({ attemptId: attempt.id, userId, items, sender });

  // `stop()` của hàng đợi khi mất quyền ghi; làm tiếp khi giành lại
  useEffect(() => {
    if (writer.state === "other") queue.stop();
    if (writer.state === "you") queue.restart();
  }, [writer.state, queue]);

  // mạng, tab hiện lại, rời trang
  useEffect(() => {
    const on = () => queue.setOnline(true);
    const off = () => queue.setOnline(false);
    const vis = () => document.visibilityState === "visible" && queue.resume();
    const unload = (e: BeforeUnloadEvent) => {
      if (queue.state.pending > 0) e.preventDefault();
    };
    if (!navigator.onLine) queue.setOnline(false);
    window.addEventListener("online", on);
    window.addEventListener("offline", off);
    document.addEventListener("visibilitychange", vis);
    window.addEventListener("beforeunload", unload);
    return () => {
      window.removeEventListener("online", on);
      window.removeEventListener("offline", off);
      document.removeEventListener("visibilitychange", vis);
      window.removeEventListener("beforeunload", unload);
    };
  }, [queue]);

  const item = items[index];
  const answeredCount = items.filter((it) => answeredOf(it, answers[it.item_id], codeAnswered)).length;
  const go = useCallback((i: number) => setIndex(Math.max(0, Math.min(items.length - 1, i))), [items.length]);

  // bàn phím: ← → đổi câu; 1…8 / A…H chọn đáp án
  useEffect(() => {
    const onKey = (e: KeyboardEvent) => {
      const el = e.target as HTMLElement | null;
      if (el && (el.tagName === "INPUT" && (el as HTMLInputElement).type === "text" ? true : el.tagName === "TEXTAREA" || el.isContentEditable)) return;
      if (e.ctrlKey || e.metaKey || e.altKey || document.querySelector("dialog[open]")) return;
      if (e.key === "ArrowRight") return go(index + 1);
      if (e.key === "ArrowLeft") return go(index - 1);
      if (!canWrite) return;
      const k = e.key.length === 1 ? e.key.toUpperCase() : "";
      const pos = /[1-8]/.test(k) ? Number(k) - 1 : LETTERS.indexOf(k);
      if (pos >= 0 && pos < item.options.length) choose(item, item.options[pos].id);
    };
    window.addEventListener("keydown", onKey);
    return () => window.removeEventListener("keydown", onKey);
  }, [go, index, item, canWrite, choose]);

  async function submit() {
    setSubmitting(true);
    setSubmitErr(null);
    submitKey.current ??= crypto.randomUUID();
    const delays = [1000, 2000, 4000, 8000, 15000];
    for (let n = 0; ; n++) {
      try {
        await Promise.all([queue.flush(), ...[...flushers.current].map((f) => f())]); // bản cuối lên máy chủ trước khi nộp (chấm theo phần đã lưu)
        if (queue.state.pending > 0) throw new ApiError({ status: 0, code: "NETWORK" });
        const r = await submitAttempt(courseId, examId, attempt.id, tab, submitKey.current);
        queue.clear();
        try {
          localStorage.removeItem(`exam-writer:${examId}`);
        } catch {
          /* bỏ qua */
        }
        onFinished(r);
        return;
      } catch (e) {
        if (e instanceof ApiError && e.status !== 0 && e.status < 500 && e.status !== 429) {
          if ((e.code as string) === "ATTEMPT_ALREADY_SUBMITTED") return onFinished(); // đã nộp (tự nộp hết giờ / lần trước đã tới máy chủ)
          setSubmitErr(e);
          setSubmitting(false);
          return;
        }
        await sleep(delays[Math.min(n, delays.length - 1)]); // chập chờn: giữ "Đang nộp…" và gửi lại CÙNG khoá
      }
    }
  }

  const unanswered = items.length - answeredCount;
  const offline = qs.status === "offline";
  const status = offline
    ? "Mất mạng — bài vẫn được giữ trên máy bạn"
    : qs.pending > 0
      ? `Chưa lưu lên máy chủ (${qs.pending} thay đổi) — đang thử lại`
      : qs.savedAt
        ? `Đã lưu lúc ${hhmmss(qs.savedAt)}`
        : "Mọi thay đổi sẽ tự lưu";

  return (
    <div className={[s.take, item.type === "CODE" ? s.takeWide : ""].join(" ")} data-part="take">
      <header className={s.bar}>
        <span className={s.barQ}>Câu {index + 1}/{items.length}</span>
        <ExamTimer deadlineMs={deadlineMs} clock={clock} onWarn={setWarn} onExpire={() => { setExpired(true); if (!hasCode) onExpired(); }} />
        <span className={s.save} role="status" aria-live="off" data-part="save-status">{status}</span>
        <Button size="sm" onClick={() => setConfirm(true)} disabled={!canWrite || expired}>Nộp bài</Button>
      </header>

      <p className={s.warn} role="status" aria-live="polite">{warn === 5 ? "Còn 5 phút." : warn === 1 ? "Còn 1 phút." : ""}</p>
      {expired && !hasCode && <InlineNotice compact>Hết giờ — đang nộp bài của bạn…</InlineNotice>}
      {expired && hasCode && <InlineNotice compact action={<Button size="sm" onClick={onExpired}>Xem tóm tắt</Button>}>Hết giờ — bài của bạn đang được nộp. Phần bạn gõ sau giờ không được tính.</InlineNotice>}
      {offline && <InlineNotice tone="warning" compact>Mất mạng — bài vẫn được giữ trên máy bạn. Nếu hết giờ khi chưa có mạng, chỉ phần đã lưu lên máy chủ được tính.</InlineNotice>}
      {writer.state === "other" && (
        <InlineNotice tone="warning" compact title="Bài đang mở ở nơi khác" action={<Button size="sm" onClick={() => void writer.claim()}>Làm tiếp ở đây</Button>}>
          Tab này chỉ để xem.{qs.pending > 0 ? " Có thay đổi chưa lưu — chép ra nếu cần." : ""}
        </InlineNotice>
      )}

      <div className={s.layout}>
        <aside className={s.side} aria-label="Danh sách câu">
          <Navigator items={items} answers={answers} code={codeAnswered} index={index} onGo={go} />
        </aside>
        <section className={s.question} aria-label="Câu hỏi" data-part="question">
          {item.type === "CODE" ? (
            <CodeQuestion
              key={item.item_id}
              item={item}
              number={index + 1}
              courseId={courseId}
              examId={examId}
              attemptId={attempt.id}
              tab={tab}
              clock={clock}
              canWrite={canWrite}
              expired={expired}
              onFatal={(why) => (why === "other" ? writer.lose() : onExpired())}
              registerFlush={registerFlush}
              onAnswered={onAnswered}
            />
          ) : (
            <Question item={item} number={index + 1} mode={exam.multi_scoring} value={answers[item.item_id]} disabled={!canWrite || expired} onChoose={(id) => choose(item, id)} />
          )}
        </section>
      </div>

      <nav className={s.foot} aria-label="Chuyển câu">
        <Button onClick={() => go(index - 1)} disabled={index === 0}>Câu trước</Button>
        <Button className={s.listBtn} icon={<ListChecks aria-hidden />} onClick={() => setListOpen(true)}>Danh sách câu</Button>
        <Button onClick={() => go(index + 1)} disabled={index === items.length - 1}>Câu sau</Button>
      </nav>

      <Drawer open={listOpen} onClose={() => setListOpen(false)} title="Danh sách câu" description={`Đã làm ${answeredCount}/${items.length} câu.`}>
        <Navigator items={items} answers={answers} code={codeAnswered} index={index} onGo={(i) => { go(i); setListOpen(false); }} />
      </Drawer>

      <ConfirmIrreversible
        open={confirm}
        onClose={() => !submitting && setConfirm(false)}
        title="Nộp bài thi?"
        consequence={`Bạn đã trả lời ${answeredCount}/${items.length} câu. Còn ${unanswered} câu chưa trả lời. Sau khi nộp bạn không sửa được.`}
        confirmLabel={submitting ? "Đang nộp…" : "Nộp bài"}
        cancelLabel="Làm tiếp"
        loading={submitting}
        error={submitErr ? submitErr.userMessage : undefined}
        onConfirm={() => void submit()}
      />
    </div>
  );
}

function Navigator({ items, answers, code, index, onGo }: { items: TakeItem[]; answers: Record<string, Answer | undefined>; code: Record<string, boolean>; index: number; onGo: (i: number) => void }) {
  return (
    <ol className={s.nav}>
      {items.map((it, i) => {
        const done = answeredOf(it, answers[it.item_id], code);
        return (
          <li key={it.item_id}>
            <button type="button" className={[s.navBtn, i === index ? s.navCurrent : ""].join(" ")} aria-current={i === index ? "step" : undefined} onClick={() => onGo(i)}>
              <span>Câu {i + 1}</span>
              <StatusText tone={done ? "green" : "neutral"}>{done ? <Check aria-hidden /> : <Circle aria-hidden />}{done ? "Đã làm" : "Chưa làm"}</StatusText>
            </button>
          </li>
        );
      })}
    </ol>
  );
}

function Question({ item, number, mode, value, disabled, onChoose }: { item: TakeItem; number: number; mode: "PARTIAL" | "ALL_OR_NOTHING"; value: Answer | undefined; disabled: boolean; onChoose: (optionId: string) => void }) {
  const multi = item.type === "MCQ_MULTI";
  const chosen = new Set(value && "option_ids" in value ? value.option_ids : []);
  const tf = value && "value" in value ? value.value : undefined;
  return (
    <fieldset className={s.fieldset} disabled={disabled}>
      <legend className={s.legend}>
        <span className={s.qMeta}>Câu {number} · {item.points.replace(".", ",")} điểm</span>
        <Markdown source={item.stem} />
      </legend>
      {multi && <p className={s.hint}>{mode === "PARTIAL" ? "Chọn tất cả đáp án đúng. Chọn sai sẽ bị trừ vào điểm của câu này (không xuống dưới 0)." : "Chỉ được điểm khi chọn đủ và đúng."}</p>}
      {item.type === "TRUE_FALSE" ? (
        <div className={s.options} role="radiogroup" aria-label="Đúng hoặc sai">
          {[true, false].map((v) => (
            <label key={String(v)} className={[s.option, tf === v ? s.optionOn : ""].join(" ")}>
              <input type="radio" name={item.item_id} checked={tf === v} onChange={() => onChoose(v ? "true" : "false")} />
              <span className={s.letter} aria-hidden>{v ? "Đ" : "S"}</span>
              <span>{v ? "Đúng" : "Sai"}</span>
            </label>
          ))}
        </div>
      ) : (
        <div className={s.options} role={multi ? "group" : "radiogroup"} aria-label="Các đáp án">
          {item.options.map((o, i) => (
            <label key={o.id} className={[s.option, chosen.has(o.id) ? s.optionOn : ""].join(" ")}>
              <input type={multi ? "checkbox" : "radio"} name={item.item_id} checked={chosen.has(o.id)} onChange={() => onChoose(o.id)} />
              <span className={s.letter} aria-hidden>{LETTERS[i] ?? i + 1}</span>
              <span className={s.optionText}>{o.body}</span>
            </label>
          ))}
        </div>
      )}
    </fieldset>
  );
}
