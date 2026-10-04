"use client";

import { useInfiniteQuery, useQueryClient, type QueryKey } from "@tanstack/react-query";
import { useEffect, useMemo, useRef } from "react";
import { apiClient } from "./apiClient";
import { ApiError } from "./ApiError";

export type CursorPage<T> = { items: T[]; next_cursor: string | null };

const DEFAULT_LIMIT = 30;
const MAX_LIMIT = 100;

/**
 * Danh sách phân trang con trỏ (PG 6.4): `limit` mặc định 30, kẹp ≤ 100; `next_cursor` là chuỗi mờ (không giải mã);
 * gộp các trang, loại trùng theo `id`; INVALID_CURSOR → quay về trang đầu MỘT lần rồi mới báo lỗi.
 */
export function useCursorList<T extends object>(
  key: QueryKey,
  path: string,
  opts: { limit?: number; query?: Record<string, string | number | boolean | undefined>; /** khoá duy nhất của một dòng; mặc định `id` */ idOf?: (row: T) => string } = {},
) {
  const idOf = opts.idOf ?? ((row: T) => (row as { id: string }).id);
  const limit = Math.min(MAX_LIMIT, Math.max(1, opts.limit ?? DEFAULT_LIMIT));
  const qc = useQueryClient();
  const reset = useRef(false);
  const q = useInfiniteQuery({
    queryKey: key,
    initialPageParam: null as string | null,
    queryFn: async ({ pageParam, signal }) => (await apiClient.get<CursorPage<T>>(path, { signal, query: { ...opts.query, limit, cursor: pageParam } })).data,
    getNextPageParam: (last) => last.next_cursor ?? undefined,
  });

  const err = q.error;
  useEffect(() => {
    if (err instanceof ApiError && err.code === "INVALID_CURSOR" && !reset.current) {
      reset.current = true;
      void qc.resetQueries({ queryKey: key });
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [err]);

  const items = useMemo(() => {
    const seen = new Set<string>();
    const out: T[] = [];
    for (const p of q.data?.pages ?? []) for (const it of p.items) { const k = idOf(it); if (!seen.has(k)) { seen.add(k); out.push(it); } }
    return out;
  // idOf là hàm thuần của một dòng: đổi danh tính hàm (inline ở nơi gọi) không đổi kết quả nên không đưa vào deps
  // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [q.data]);

  return { ...q, items, limit };
}
