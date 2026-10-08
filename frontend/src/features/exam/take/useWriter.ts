"use client";

import { useCallback, useEffect, useRef, useState } from "react";

export type WriterState = "checking" | "you" | "other";

type Msg = { type: "ping" | "pong" | "taken"; tab: string };
const stored = (examId: string) => `exam-writer:${examId}`;

/** Tab id lần cuối làm người ghi TRONG TRÌNH DUYỆT NÀY (nhớ qua tải lại; không dùng để quyết định gì ở máy chủ). */
export function readStoredTab(examId: string): string | null {
  try {
    return (JSON.parse(localStorage.getItem(stored(examId)) ?? "null") as { tab?: string } | null)?.tab ?? null;
  } catch {
    return null;
  }
}

/**
 * Một nơi được ghi (US-PE-05 AC9, SRS 4.3.5). Tab id đổi mỗi lần tải trang nên quyền ghi sau khi tải lại được giữ bằng `BroadcastChannel('exam-tab:<attempt_id>')` + `localStorage`:
 * lúc tải, gửi `ping` và đợi 300 ms. (1) Có tab khác của cùng trình duyệt trả lời → CHỈ ĐỌC (ca nhân bản tab / `window.open`: không bao giờ thành người ghi thứ hai).
 * (2) Không ai trả lời và máy chủ nói người ghi cũ chính là id đã nhớ (`prevWriter`) → writer cũ là trang vừa bị tải lại → tự `takeover({reload:true})`.
 * (3) Còn lại → chỉ đọc + `Làm tiếp ở đây`.
 */
export function useWriter({ examId, attemptId, tab, initiallyYou, prevWriter, takeover, onLostWrite }: {
  examId: string;
  attemptId: string;
  tab: string;
  /** vừa bấm Bắt đầu làm bài trong chính tab này: máy chủ đã đặt tab này làm người ghi */
  initiallyYou: boolean;
  prevWriter: boolean;
  takeover: (reload: boolean) => Promise<void>;
  /** tab khác trong trình duyệt vừa nhận quyền ghi */
  onLostWrite?: () => void;
}) {
  const [state, setState] = useState<WriterState>(initiallyYou ? "you" : "checking");
  const stateRef = useRef<WriterState>(state);
  const chan = useRef<BroadcastChannel | null>(null);
  const take = useRef(takeover);
  const lost = useRef(onLostWrite);
  useEffect(() => {
    take.current = takeover;
    lost.current = onLostWrite;
  });

  const remember = useCallback(() => {
    try {
      localStorage.setItem(stored(examId), JSON.stringify({ tab, attempt: attemptId }));
    } catch {
      /* kho đầy: bỏ qua */
    }
  }, [examId, tab, attemptId]);
  const set = useCallback((s: WriterState) => {
    stateRef.current = s;
    setState(s);
    if (s === "you") remember();
  }, [remember]);

  useEffect(() => {
    const ch = typeof BroadcastChannel === "undefined" ? null : new BroadcastChannel(`exam-tab:${attemptId}`);
    chan.current = ch;
    let answered = false;
    if (ch) {
      ch.onmessage = (e: MessageEvent<Msg>) => {
        const m = e.data;
        if (m.tab === tab) return;
        if (m.type === "ping" && stateRef.current === "you") ch.postMessage({ type: "pong", tab } satisfies Msg);
        if (m.type === "pong") answered = true;
        if (m.type === "taken" && stateRef.current === "you") {
          set("other");
          lost.current?.();
        }
      };
    }
    if (initiallyYou) {
      remember();
      return () => ch?.close();
    }
    ch?.postMessage({ type: "ping", tab } satisfies Msg);
    const t = window.setTimeout(() => {
      if (answered) return set("other");
      if (prevWriter) {
        take.current(true).then(() => set("you"), () => set("other"));
        return;
      }
      set("other");
    }, ch ? 300 : 0);
    return () => {
      window.clearTimeout(t);
      ch?.close();
    };
    // chỉ chạy khi lượt đổi: các giá trị khác là hằng của lần tải trang
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [attemptId]);

  /** `Làm tiếp ở đây`: giành quyền ghi cho tab này, báo các tab khác của trình duyệt. */
  const claim = useCallback(async () => {
    await take.current(false);
    set("you");
    chan.current?.postMessage({ type: "taken", tab } satisfies Msg);
  }, [set, tab]);

  const lose = useCallback(() => set("other"), [set]);
  return { state, claim, lose };
}
