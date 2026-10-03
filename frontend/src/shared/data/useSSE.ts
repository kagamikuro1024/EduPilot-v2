"use client";

import { useQueryClient, type QueryKey } from "@tanstack/react-query";
import { useEffect, useRef, useSyncExternalStore } from "react";
import { sseManager, type SSEEvent, type SSEStatus } from "./sse";

export type { SSEEvent, SSEStatus };

/** Trạng thái kết nối SSE của tab: connecting → open → reconnecting → degraded → closed. */
export function useSSEStatus(): SSEStatus {
  return useSyncExternalStore(sseManager.subscribeStatus, () => sseManager.status, () => "closed" as SSEStatus);
}

/**
 * Nghe một loại sự kiện SSE qua kết nối duy nhất của tab. `onResync` mặc định vô hiệu hoá các query trong `invalidateKeys`
 * (khoá đăng ký SSE); huỷ gắn kết cuối cùng của cả tab ⇒ abort kết nối.
 */
export function useSSE(type: string, handler: (e: SSEEvent) => void, opts: { invalidateKeys?: QueryKey[]; onResync?: () => void } = {}) {
  const qc = useQueryClient();
  const h = useRef(handler);
  const keys = useRef(opts.invalidateKeys);
  const resync = useRef(opts.onResync);
  useEffect(() => {
    h.current = handler;
    keys.current = opts.invalidateKeys;
    resync.current = opts.onResync;
  });
  useEffect(
    () =>
      sseManager.on(
        type,
        (e) => h.current(e),
        () => {
          if (resync.current) resync.current();
          else keys.current?.forEach((k) => void qc.invalidateQueries({ queryKey: k }));
        },
      ),
    [type, qc],
  );
  return useSSEStatus();
}
