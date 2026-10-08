"use client";

import { useEffect, useState } from "react";

/**
 * Dựng danh sách dài theo đợt: trả số dòng được phép dựng, bắt đầu bằng `first` rồi thêm `step` dòng mỗi lượt rảnh của luồng chính.
 * Một lượt dựng 40+ dòng nhiều ô trong MỘT tác vụ chặn luồng chính > 50 ms (TBT, US-PU-06); chia đợt giữ mỗi tác vụ ngắn.
 * Dòng dựng sau nằm DƯỚI dòng đã có nên không dịch bố cục (CLS). `total` đổi (đủ ít) thì tự co / giãn.
 */
export function useProgressiveCount(total: number, first = 8, step = 8): number {
  const [n, setN] = useState(first);
  useEffect(() => {
    if (n >= total) return;
    const id = window.setTimeout(() => setN((c) => c + step), 16);
    return () => window.clearTimeout(id);
  }, [n, total, step]);
  return Math.min(n, total);
}
