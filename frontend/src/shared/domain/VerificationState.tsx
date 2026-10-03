"use client";

import { Clock } from "lucide-react";
import type { ReactNode } from "react";
import s from "./VerificationState.module.css";

export type Verification = "pending" | "verified" | "edited" | "awaiting";

/**
 * Trạng thái xác nhận của một câu trả lời AI (UX.md quy tắc 7, DESIGN.md §13). Bốn lời cố định:
 * "Chờ xác nhận" · "Đã được giảng viên xác nhận · <tên>" · "Đã được giảng viên sửa & xác nhận" · "Đang chờ giảng viên".
 * Đã xác nhận: chấm xanh + đường kẻ xanh 1 px cạnh nội dung, KHÔNG tô nền. Nút duyệt chỉ hiện khi `canReview` (TA / GV / Admin)
 * và nằm ngay cạnh nháp; sinh viên không bao giờ thấy.
 */
export function VerificationState({
  status,
  verifier,
  canReview,
  disabled,
  onConfirm,
  onEdit,
  onReject,
  onViewOriginal,
  children,
}: {
  status: Verification;
  verifier?: string;
  canReview?: boolean;
  /** khoá ba nút duyệt (đang lưu…) */
  disabled?: boolean;
  onConfirm?: () => void;
  onEdit?: () => void;
  onReject?: () => void;
  onViewOriginal?: () => void;
  children: ReactNode;
}) {
  const done = status === "verified" || status === "edited";
  const label =
    status === "pending" ? "Chờ xác nhận" : status === "awaiting" ? "Đang chờ giảng viên" : status === "edited" ? "Đã được giảng viên sửa & xác nhận" : `Đã được giảng viên xác nhận${verifier ? ` · ${verifier}` : ""}`;
  return (
    <div className={s.root} data-status={status}>
      <p className={[s.label, done ? s.green : s.amber].join(" ")}>
        {status === "pending" ? <Clock className={s.icon} aria-hidden /> : <span className={s.dot} aria-hidden />}
        <span>{label}</span>
        {status === "edited" && (
          <button type="button" className={s.link} onClick={onViewOriginal}>
            Xem câu trả lời AI gốc
          </button>
        )}
      </p>
      <div className={[s.body, done ? s.verifiedBody : "", status === "pending" ? s.draft : ""].join(" ")} data-part="verified-block">
        {children}
      </div>
      {canReview && status === "pending" && (
        <div className={s.actions}>
          <button type="button" className={s.act} disabled={disabled} onClick={onConfirm}>
            Xác nhận
          </button>
          <button type="button" className={s.act} disabled={disabled} onClick={onEdit}>
            Chỉnh sửa
          </button>
          <button type="button" className={s.act} disabled={disabled} onClick={onReject}>
            Loại khỏi tri thức
          </button>
        </div>
      )}
    </div>
  );
}
