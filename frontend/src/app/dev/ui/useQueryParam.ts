"use client";

import { useSyncExternalStore } from "react";

const noop = () => () => {};

/**
 * Đọc một tham số truy vấn ở client mà KHÔNG dùng `useSearchParams` (hàm đó buộc cả trang vào Suspense ⇒ máy chủ không vẽ gì).
 * Máy chủ / lần hydrate đầu thấy `null`; ngay sau đó React dựng lại với giá trị thật. Dùng cho công cụ dev (US-PU-06: `/dev/ui` vẽ từ máy chủ).
 */
export function useQueryParam(name: string): string | null {
  return useSyncExternalStore(
    noop,
    () => new URLSearchParams(window.location.search).get(name),
    () => null,
  );
}
