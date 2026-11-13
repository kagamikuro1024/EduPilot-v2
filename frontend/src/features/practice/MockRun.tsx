"use client";

import { Check, Circle, ListChecks } from "lucide-react";
import { useCallback, useEffect, useRef, useState } from "react";
import { ApiError } from "@/shared/data";
import { ExamTimer, Markdown } from "@/shared/domain";
import type { ExamClock } from "@/shared/lib/examClock";
import { SaveQueue, type QueueState, type Sender } from "@/shared/lib/saveQueue";
import { Button, ConfirmIrreversible, Drawer, InlineNotice, Panel, StatusText } from "@/shared/ui";
import { QuestionInput, filled } from "./QuestionInput";
import { newIdempotencyKey, putAnswer, submitAttempt, type Ans, type PAttempt, type PItem } from "./practiceApi";
import s from "./Practice.module.css";

type Qd = Ans & { ts?: number }; // `ts`: lúc người làm bài đổi đáp án (ms) — gửi thành client_ts để dung sai ghi ngoại tuyến (Q-P9-5)
const hhmmss = (ms: number) => new Date(ms + 7 * 3600_000).toISOString().slice(11, 19);
const sleep = (ms: number) => new Promise<void>((r) => setTimeout(r, ms));
const storeOf = (key: string) => ({
  read: () => { try { return localStorage.getItem(key); } catch { return null; } },
  write: (v: string) => { try { localStorage.setItem(key, v); } catch { /* kho đầy: giữ trong bộ nhớ */ } },
  remove: () => { try { localStorage.removeItem(key); } catch { /* bỏ qua */ } },
});

/**
 * Thi thử (US-P9-07): đồng hồ theo máy chủ, tự lưu từng câu qua hàng đợi cục bộ (mất mạng vẫn làm tiếp), MỘT tab ghi (`X-Practice-Tab`), nộp có xác nhận.
 * Không phản hồi đúng / sai và không `Giải thích thêm` cho tới khi nộp.
 */
export function MockRun({ courseId, initial, tab, clock, userId, onDone, onTakeOver }: {
  courseId: string;
  initial: PAttempt;
  tab: string;
  clock: ExamClock;
  userId: string;
  onDone: (a: PAttempt) => void;
  /** `Làm tiếp ở tab này`: tải lại lượt (máy chủ chuyển quyền ghi sang tab này) */
  onTakeOver: () => void;
}) {
  const items = initial.items;
  const deadlineMs = Date.parse(initial.deadline_at ?? "");
  const [index, setIndex] = useState(Math.min(initial.current_index, items.length - 1));
  const [listOpen, setListOpen] = useState(false);
  const [confirm, setConfirm] = useState(false);
  const [submitting, setSubmitting] = useState(false);
  const [submitErr, setSubmitErr] = useState<ApiError | null>(null);
  const [warn, setWarn] = useState<1 | 5 | null>(null);
  const [expired, setExpired] = useState(false);
  const [other, setOther] = useState(false); // tab khác đã giành quyền ghi
  const versions = useRef<Record<string, number>>(Object.fromEntries(items.filter((i) => i.answer).map((i) => [i.id, i.answer!.version])));
  const submitKey = useRef<string | null>(null);
  const closedByServer = useRef(false); // máy chủ đã đóng lượt (ATTEMPT_CLOSED): phần còn chờ trong hàng đợi không gửi được nữa, nộp lấy kết quả theo phần đã lưu

  const sender = useCallback<Sender<Qd>>(async (batch) => {
    for (const b of batch) {
      const { ts, ...answer } = b.answer;
      try {
        const r = await putAnswer(courseId, initial.id, b.item_id, tab, { answer: answer as Ans, version: versions.current[b.item_id] ?? 0, client_ts: ts ? new Date(ts).toISOString() : undefined });
        versions.current[b.item_id] = r.version;
      } catch (e) {
        if (!(e instanceof ApiError)) return { ok: false as const };
        if (e.code === "TAB_TAKEN_OVER") { setOther(true); return { ok: false as const, fatal: true }; }
        if (e.code === "ATTEMPT_CLOSED") { closedByServer.current = true; setExpired(true); return { ok: false as const, fatal: true }; }
        if (e.code === "VERSION_CONFLICT") {
          const v = (e.details as { version?: number } | undefined)?.version;
          if (typeof v === "number") versions.current[b.item_id] = v;
          return { ok: false as const }; // thử lại với phiên bản mới
        }
        if (e.status >= 400 && e.status < 500 && e.status !== 429 && e.status !== 408) return { ok: false as const, fatal: true };
        return { ok: false as const };
      }
    }
    return { ok: true as const };
  }, [courseId, initial.id, tab]);

  const [queue] = useState(() => new SaveQueue<Qd>(storeOf(`ep:draft:${userId}:practice:${initial.id}`), async () => ({ ok: false as const }), { set: (fn, ms) => window.setTimeout(fn, ms), clear: (h) => window.clearTimeout(h as number) }, () => Date.now()));
  useEffect(() => queue.setSender(sender), [queue, sender]);
  const [qs, setQs] = useState<QueueState>(queue.state);
  useEffect(() => queue.subscribe(setQs), [queue]);
  const [answers, setAnswers] = useState<Record<string, Qd | undefined>>(() => ({
    ...Object.fromEntries(items.filter((i) => i.answer).map((i) => [i.id, i.answer!.answer as Ans])),
    ...queue.pending(),
  }));
  useEffect(() => { if (Object.keys(queue.pending()).length > 0) queue.resume(); }, [queue]);
  useEffect(() => { if (other) queue.stop(); }, [other, queue]);

  useEffect(() => {
    const on = () => queue.setOnline(true);
    const off = () => queue.setOnline(false);
    const vis = () => document.visibilityState === "visible" && queue.resume();
    const unload = (e: BeforeUnloadEvent) => { if (queue.state.pending > 0) e.preventDefault(); };
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
  const canWrite = !other && !expired;
  const done = (it: PItem) => filled(it.question, answers[it.id]);
  const answeredCount = items.filter(done).length;
  const go = useCallback((i: number) => setIndex(Math.max(0, Math.min(items.length - 1, i))), [items.length]);

  function choose(it: PItem, v: Ans) {
    const next: Qd = { ...v, ts: Date.now() };
    setAnswers((p) => ({ ...p, [it.id]: next }));
    queue.set(it.id, next);
  }

  async function submit() {
    setSubmitting(true);
    setSubmitErr(null);
    submitKey.current ??= newIdempotencyKey();
    const delays = [1000, 2000, 4000, 8000, 15000];
    for (let n = 0; ; n++) {
      try {
        if (!other && !closedByServer.current) await queue.flush(); // bản cuối lên máy chủ trước khi nộp (chấm theo phần đã lưu)
        if (!other && !closedByServer.current && queue.state.pending > 0) throw new ApiError({ status: 0, code: "NETWORK" });
        const r = await submitAttempt(courseId, initial.id, submitKey.current);
        queue.clear();
        onDone(r);
        return;
      } catch (e) {
        if (e instanceof ApiError && e.status !== 0 && e.status < 500 && e.status !== 429) {
          setSubmitErr(e);
          setSubmitting(false);
          return;
        }
        await sleep(delays[Math.min(n, delays.length - 1)]); // chập chờn: giữ "Đang nộp…", gửi lại CÙNG khoá
      }
    }
  }

  // Hết giờ: nộp ngay (máy chủ vẫn nhận lựa chọn ghi ngoại tuyến trong dung sai; worker đóng lượt nếu tab này không nộp được).
  const autoSubmitted = useRef(false);
  useEffect(() => {
    if (expired && !autoSubmitted.current && !other) {
      autoSubmitted.current = true;
      void submit();
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [expired]);

  const offline = qs.status === "offline";
  const status = offline ? "Chưa gửi được" + (qs.pending ? ` ${qs.pending} câu` : "") : qs.pending > 0 ? "Đang lưu…" : qs.savedAt ? `Đã lưu lúc ${hhmmss(qs.savedAt)}` : "Tự lưu";

  return (
    <div className={s.take} data-part="practice-mock">
      <header className={s.bar}>
        <span className={s.barQ}>Câu {index + 1}/{items.length}</span>
        <ExamTimer deadlineMs={deadlineMs} clock={clock} onWarn={setWarn} onExpire={() => setExpired(true)} />
        <span className={s.save} role="status" aria-live="off">{status}</span>
        <Button size="sm" onClick={() => setConfirm(true)} disabled={!canWrite || submitting}>Nộp bài</Button>
      </header>
      <p className={s.warnLine} role="status" aria-live="polite">{warn === 5 ? "Còn 5 phút." : warn === 1 ? "Còn 1 phút." : ""}</p>
      {expired && <InlineNotice compact>Hết giờ — đang nộp bài của bạn.</InlineNotice>}
      {offline && !expired && <InlineNotice tone="warning" compact>Chưa gửi được. Bài vẫn giữ trên máy bạn.</InlineNotice>}
      {other && (
        <InlineNotice tone="warning" compact title="Bài đang mở ở tab khác" action={<Button size="sm" onClick={onTakeOver}>Làm tiếp ở tab này</Button>}>
          Tab này chỉ để xem.
        </InlineNotice>
      )}

      <div className={s.layout}>
        <aside className={s.side} aria-label="Danh sách câu">
          <Panel><Navigator items={items} done={done} index={index} onGo={go} /></Panel>
        </aside>
        <section className={s.mockQuestion} aria-label="Câu hỏi">
          <Panel>
            <span className={s.qMeta}>Câu {index + 1}</span>
            <div className={s.question}><Markdown source={item.question.stem} /></div>
            <QuestionInput q={item.question} value={answers[item.id]} onChange={(v) => choose(item, v)} disabled={!canWrite} />
          </Panel>
        </section>
      </div>

      <nav className={s.foot} aria-label="Chuyển câu">
        <Button onClick={() => go(index - 1)} disabled={index === 0}>Câu trước</Button>
        <Button className={s.listBtn} icon={<ListChecks aria-hidden />} onClick={() => setListOpen(true)}>Danh sách câu</Button>
        <Button onClick={() => go(index + 1)} disabled={index === items.length - 1}>Câu sau</Button>
      </nav>

      <Drawer open={listOpen} onClose={() => setListOpen(false)} title="Danh sách câu" description={`Đã làm ${answeredCount}/${items.length} câu.`}>
        <Navigator items={items} done={done} index={index} onGo={(i) => { go(i); setListOpen(false); }} />
      </Drawer>

      <ConfirmIrreversible
        open={confirm}
        onClose={() => !submitting && setConfirm(false)}
        title="Nộp bài thi thử?"
        consequence={`Bạn đã trả lời ${answeredCount}/${items.length} câu. Nộp xong không sửa được.`}
        confirmLabel={submitting ? "Đang nộp…" : "Nộp bài"}
        cancelLabel="Làm tiếp"
        loading={submitting}
        error={submitErr ? submitErr.userMessage : undefined}
        onConfirm={() => void submit()}
      />
    </div>
  );
}

function Navigator({ items, done, index, onGo }: { items: PItem[]; done: (it: PItem) => boolean; index: number; onGo: (i: number) => void }) {
  return (
    <ol className={s.nav}>
      {items.map((it, i) => (
        <li key={it.id}>
          <button type="button" className={[s.navBtn, i === index ? s.navCurrent : ""].join(" ")} aria-current={i === index ? "step" : undefined} onClick={() => onGo(i)}>
            <span>Câu {i + 1}</span>
            <StatusText tone={done(it) ? "green" : "neutral"}>{done(it) ? <Check aria-hidden /> : <Circle aria-hidden />}{done(it) ? "Đã làm" : "Chưa làm"}</StatusText>
          </button>
        </li>
      ))}
    </ol>
  );
}

