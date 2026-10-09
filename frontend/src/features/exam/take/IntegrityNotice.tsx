"use client";

import { useEffect, useRef } from "react";
import s from "./Take.module.css";

/** Câu minh bạch (US-PE-07 AC6) — đúng chữ ở màn bắt đầu và ở dải cố định lúc làm bài. */
export const INTEGRITY_TEXT =
  "Trong giờ làm bài, chat AI tạm khoá. Hệ thống ghi lại số lần bạn rời trang hoặc dán nội dung để giảng viên xem khi cần; đây không phải giám thị và không tự trừ điểm của bạn.";

/**
 * Câu nói rõ với sinh viên kèm "Tìm hiểu thêm" (ghi gì / không ghi gì). `onSeen` báo MỘT lần khi câu đã hiện trong khung nhìn — `Bắt đầu làm bài` chỉ bấm được sau đó
 * (không bắt tích ô). Không có IntersectionObserver (môi trường cũ) → coi như đã thấy.
 */
export function IntegrityNotice({ onSeen, compact }: { onSeen?: () => void; compact?: boolean }) {
  const ref = useRef<HTMLParagraphElement>(null);
  const cb = useRef(onSeen);
  useEffect(() => {
    cb.current = onSeen;
  });
  useEffect(() => {
    const el = ref.current;
    if (!el || !cb.current) return;
    if (typeof IntersectionObserver === "undefined") return void cb.current();
    const io = new IntersectionObserver((es) => {
      if (es.some((e) => e.isIntersecting)) {
        cb.current?.();
        io.disconnect();
      }
    });
    io.observe(el);
    return () => io.disconnect();
  }, []);
  return (
    <div className={compact ? s.integrityCompact : s.integrity} data-part="integrity-notice">
      <p ref={ref} className={s.integrityText}>{INTEGRITY_TEXT}</p>
      {!compact && (
        <details className={s.integrityMore}>
          <summary>Tìm hiểu thêm</summary>
          <p>Hệ thống chỉ ghi loại sự kiện (rời trang, quay lại, dán, mất mạng), thời điểm và độ dài đoạn dán. Không ghi nội dung bạn dán, không dùng camera, không ghi màn hình.</p>
        </details>
      )}
    </div>
  );
}
