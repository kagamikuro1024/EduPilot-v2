"use client";

import { useCallback, useEffect, useState } from "react";
import { SaveQueue, type Answer, type QueueState, type Sender } from "@/shared/lib/saveQueue";
import type { TakeItem } from "./takeApi";

/** Khoá localStorage của hàng đợi: `ep:draft:<người>:exam:<attempt_id>` (cùng tiền tố nháp của `useAutosaveDraft`; SaveQueue ghi `savedAt` để bộ dọn nháp cũ không xoá nhầm). */
const keyOf = (userId: string, attemptId: string) => `ep:draft:${userId}:exam:${attemptId}`;

function browserStore(key: string) {
  return {
    read: () => {
      try {
        return localStorage.getItem(key);
      } catch {
        return null;
      }
    },
    write: (v: string) => {
      try {
        localStorage.setItem(key, v);
      } catch {
        /* kho đầy: giữ trong bộ nhớ */
      }
    },
    remove: () => {
      try {
        localStorage.removeItem(key);
      } catch {
        /* bỏ qua */
      }
    },
  };
}

/** Câu trả lời đang hiển thị = bản của máy chủ, đè bởi phần còn giữ ở máy (chỉ MỘT nơi ghi nên bản ở máy luôn mới hơn). */
export function useAnswers({ attemptId, userId, items, sender }: { attemptId: string; userId: string; items: TakeItem[]; sender: Sender }) {
  const [queue] = useState(
    () => new SaveQueue(browserStore(keyOf(userId, attemptId)), async () => ({ ok: false as const }), { set: (fn, ms) => window.setTimeout(fn, ms), clear: (h) => window.clearTimeout(h as number) }, () => Date.now()),
  );
  useEffect(() => queue.setSender(sender), [queue, sender]); // hàm gửi thật; lần gửi đầu chỉ xảy ra sau ≥ 0 ms nên đã kịp đặt
  const [answers, setAnswers] = useState<Record<string, Answer | undefined>>(() => {
    const base: Record<string, Answer | undefined> = {};
    for (const it of items) if (it.answer) base[it.item_id] = it.answer as Answer;
    return { ...base, ...queue.pending() };
  });
  const [state, setState] = useState<QueueState>(queue.state);
  useEffect(() => queue.subscribe(setState), [queue]);
  // gửi phần còn giữ ở máy từ phiên trước (tải lại / sập trình duyệt)
  useEffect(() => {
    if (Object.keys(queue.pending()).length > 0) queue.resume();
  }, [queue]);

  const choose = useCallback(
    (item: TakeItem, optionId: string) => {
      let next: Answer;
      if (item.type === "TRUE_FALSE") next = { value: optionId === "true" };
      else {
        const cur = answers[item.item_id];
        const set = new Set(cur && "option_ids" in cur ? cur.option_ids : []);
        if (item.type === "MCQ_SINGLE") {
          set.clear();
          set.add(optionId);
        } else if (set.has(optionId)) set.delete(optionId);
        else set.add(optionId);
        next = { option_ids: [...set] };
      }
      setAnswers((prev) => ({ ...prev, [item.item_id]: next }));
      queue.set(item.item_id, next);
    },
    [answers, queue],
  );
  return { queue, state, answers, choose };
}
