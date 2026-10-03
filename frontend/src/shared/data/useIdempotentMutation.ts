"use client";

import { useCallback, useRef, useState } from "react";
import { ApiError } from "./ApiError";
import { newIdempotencyKey, type ApiResult } from "./apiClient";

/**
 * Một khoá Idempotency cho MỘT ý định: `retry()` sau lỗi dùng LẠI khoá cũ (không tạo bản ghi thứ hai);
 * `mutate()` mới sau thành công sinh khoá mới. `result.replayed` (Idempotent-Replayed) không hiện gì cho người dùng.
 */
export function useIdempotentMutation<V, R>(fn: (vars: V, key: string) => Promise<ApiResult<R>>) {
  const key = useRef<string | null>(null);
  const last = useRef<V | null>(null);
  const [state, setState] = useState<{ pending: boolean; error?: ApiError; result?: { data: R; replayed: boolean } }>({ pending: false });

  const run = useCallback(
    async (vars: V) => {
      key.current ??= newIdempotencyKey();
      last.current = vars;
      setState({ pending: true });
      try {
        const r = await fn(vars, key.current);
        key.current = null; // thành công: ý định đã xong
        const out = { data: r.data, replayed: Boolean(r.replayed) };
        setState({ pending: false, result: out });
        return out;
      } catch (e) {
        setState({ pending: false, error: e instanceof ApiError ? e : new ApiError({ status: 0, code: "NETWORK" }) });
        throw e;
      }
    },
    [fn],
  );

  const mutate = useCallback(
    (vars: V) => {
      key.current = null; // bấm gửi MỚI ⇒ ý định mới
      return run(vars);
    },
    [run],
  );
  const retry = useCallback(() => (last.current === null ? Promise.resolve(undefined) : run(last.current)), [run]);

  return { mutate, retry, ...state, key: () => key.current };
}
