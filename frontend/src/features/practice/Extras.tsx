"use client";

import { useRef, useState } from "react";
import { ApiError, ApiErrorNotice } from "@/shared/data";
import { Button, InlineNotice } from "@/shared/ui";
import { essayFeedback, explain, newIdempotencyKey, type EssayFeedback } from "./practiceApi";
import s from "./Practice.module.css";

/** `Giải thích thêm`: một lời gọi AI, chữ hiện dần; lỗi chỉ ảnh hưởng phần giải thích (đúng / sai còn nguyên). */
export function ExplainMore({ course, attempt, item }: { course: string; attempt: string; item: string }) {
  const [text, setText] = useState("");
  const [state, setState] = useState<"idle" | "running" | "done" | "error">("idle");
  const abort = useRef<AbortController | null>(null);
  async function run() {
    abort.current?.abort();
    const ctrl = new AbortController();
    abort.current = ctrl;
    setText("");
    setState("running");
    try {
      await explain(course, attempt, item, ctrl.signal, (t) => setText((p) => p + t));
      setState("done");
    } catch (e) {
      if (!ctrl.signal.aborted) setState("error");
      void e;
    }
  }
  return (
    <div className={s.explainBox}>
      {state !== "running" && state !== "done" && <div><Button size="sm" onClick={() => void run()}>Giải thích thêm</Button></div>}
      {text && <p className={s.stream} aria-live="polite">{text}</p>}
      {state === "error" && <InlineNotice tone="warning" compact action={<Button size="sm" onClick={() => void run()}>Thử lại</Button>}>Chưa giải thích được.</InlineNotice>}
    </div>
  );
}

/** `Nhận xét` cho câu tự luận: gợi ý theo từng tiêu chí, không điểm. */
export function EssayFeedbackBlock({ course, attempt, item }: { course: string; attempt: string; item: string }) {
  const key = useRef<string | null>(null);
  const [res, setRes] = useState<EssayFeedback | null>(null);
  const [err, setErr] = useState<ApiError | null>(null);
  const [busy, setBusy] = useState(false);
  async function run() {
    key.current ??= newIdempotencyKey();
    setBusy(true);
    setErr(null);
    try {
      setRes(await essayFeedback(course, attempt, item, key.current));
      key.current = null;
    } catch (e) {
      setErr(e instanceof ApiError ? e : new ApiError({ status: 0, code: "NETWORK" }));
    } finally {
      setBusy(false);
    }
  }
  return (
    <div className={s.explainBox}>
      {!res && <div><Button size="sm" loading={busy} onClick={() => void run()}>Nhận xét</Button></div>}
      {err && (err.status === 429 ? <InlineNotice tone="warning" compact>Bạn đã xin nhận xét quá nhiều lần. Thử lại sau.</InlineNotice> : err.status === 503 ? <InlineNotice tone="warning" compact action={<Button size="sm" onClick={() => void run()}>Thử lại</Button>}>Chưa nhận xét được.</InlineNotice> : <ApiErrorNotice error={err} onRetry={() => void run()} />)}
      {res && (
        <>
          <p className={s.tag}>Gợi ý, giảng viên không xem</p>
          {res.injected && <InlineNotice tone="warning" compact>Bài có câu giống lệnh cho hệ thống; nhận xét chỉ dựa vào nội dung bài.</InlineNotice>}
          <p className={s.explain}>{res.overall}</p>
          <ul className={s.criteria}>
            {res.criteria.map((k) => (
              <li key={k.name} className={s.criterion}>
                <p className={s.criterionName}>{k.name}</p>
                <p className={s.explain}>{k.comment}</p>
                {k.evidence.map((q) => <p key={q} className={s.quote}>“{q}”</p>)}
              </li>
            ))}
          </ul>
        </>
      )}
    </div>
  );
}
