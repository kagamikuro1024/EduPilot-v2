"use client";

import { useEffect, useRef, useState, type ReactNode } from "react";
import { Button, InlineNotice, UndoLine } from "@/shared/ui";
import { ApiError } from "./ApiError";

type Cfg<A> = {
  /** áp thay đổi lạc quan lên UI */
  apply: (arg: A) => void;
  /** đưa UI về trạng thái trước */
  rollback: (arg: A) => void;
  /** ghi lên máy chủ */
  commit: (arg: A) => Promise<unknown>;
  /** thao tác bù ghi lên máy chủ */
  compensate: (arg: A) => Promise<unknown>;
  label: (arg: A) => ReactNode;
  itemKey?: (arg: A) => string;
};

/**
 * Thao tác đảo ngược được (UX.md quy tắc 3): cập nhật LẠC QUAN ngay, hiện dòng tĩnh "Đã … · Hoàn tác" tại chỗ, tự biến sau 5 s.
 * `commit` lỗi ⇒ `rollback` UI + InlineNotice có Thử lại; `compensate` GHI NGAY (đóng tab không mất).
 * `Hoàn tác` bị bỏ qua nếu mục đã bị thay đổi mới hơn (khoá `itemKey`).
 */
export function useUndoableAction<A>(cfg: Cfg<A>) {
  const [line, setLine] = useState<{ arg: A; label: ReactNode; rev: number } | null>(null);
  const [error, setError] = useState<{ message: string; retry: () => void } | null>(null);
  const [pending, setPending] = useState(false);
  const revs = useRef(new Map<string, number>());
  const cfgRef = useRef(cfg);
  useEffect(() => {
    cfgRef.current = cfg;
  });
  const api = useRef({
    run: async (arg: A): Promise<void> => {
      const c = cfgRef.current;
      const k = c.itemKey?.(arg) ?? "_";
      const rev = (revs.current.get(k) ?? 0) + 1;
      revs.current.set(k, rev);
      setError(null);
      c.apply(arg); // lạc quan: ≤ 100 ms
      setLine({ arg, label: c.label(arg), rev });
      setPending(true);
      try {
        await c.commit(arg);
      } catch (e) {
        c.rollback(arg);
        setLine(null);
        setError({ message: e instanceof ApiError ? e.userMessage : "Chưa lưu được. Thử lại.", retry: () => void api.current.run(arg) });
      } finally {
        setPending(false);
      }
    },
    undo: async (arg: A, rev: number): Promise<void> => {
      const c = cfgRef.current;
      const k = c.itemKey?.(arg) ?? "_";
      if (revs.current.get(k) !== rev) return; // đã bị ghi đè bởi thay đổi mới hơn
      c.rollback(arg);
      setLine(null);
      try {
        await c.compensate(arg);
      } catch (e) {
        c.apply(arg);
        setError({ message: e instanceof ApiError ? e.userMessage : "Chưa hoàn tác được. Thử lại.", retry: () => void api.current.undo(arg, rev) });
      }
    },
  });

  /** Dòng "Đã … · Hoàn tác" và lỗi — đặt NGAY tại nơi vừa thao tác. */
  const node = (
    <>
      {line && (
        <span data-part="undo-line">
          <UndoLine message={line.label} onUndo={() => void api.current.undo(line.arg, line.rev)} onDone={() => setLine(null)} />
        </span>
      )}
      {error && (
        <InlineNotice tone="danger" compact action={<Button size="sm" onClick={error.retry}>Thử lại</Button>}>
          {error.message}
        </InlineNotice>
      )}
    </>
  );

  return { run: (arg: A) => api.current.run(arg), pending, node };
}
