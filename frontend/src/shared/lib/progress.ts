import type { CSSProperties } from "react";

/**
 * Thanh tiến độ không đổi bố cục (FEAT-uiux-91 US-04 AC4): phần lấp rộng 100 % rồi `scaleX(var(--p))` từ mép trái,
 * nên chuyển động chỉ là `transform`. Giá trị ngoài 0–100 (hoặc NaN) kẹp về 0–100.
 */
export const progressStyle = (percent: number): CSSProperties => ({ ["--p" as string]: Number.isFinite(percent) ? Math.min(1, Math.max(0, percent / 100)) : 0 });
