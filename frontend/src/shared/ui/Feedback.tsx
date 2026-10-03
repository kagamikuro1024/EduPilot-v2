"use client";

import { CircleAlert, CircleCheck, Info, ShieldCheck, TriangleAlert } from "lucide-react";
import { useEffect, type ReactNode } from "react";
import s from "./Feedback.module.css";

type NoticeTone = "info" | "warning" | "danger" | "success" | "privacy";

const NOTICE_ICON = {
  info: Info,
  warning: TriangleAlert,
  danger: CircleAlert,
  success: CircleCheck,
  privacy: ShieldCheck,
} as const;

/** Thông báo mảnh tại chỗ, thay modal (DESIGN.md §10.5). */
export function InlineNotice({ tone = "info", title, children, action, compact, technical }: { tone?: NoticeTone; title?: ReactNode; children?: ReactNode; action?: ReactNode; compact?: boolean; /** "Chi tiết kỹ thuật" gấp sẵn — chỉ truyền cho TA / GV / Admin (mã lỗi, trace_id) */ technical?: ReactNode }) {
  const Icon = NOTICE_ICON[tone];
  return (
    <div className={[s.notice, s[`n_${tone}`], compact ? s.compact : ""].join(" ")} role={tone === "danger" ? "alert" : "status"}>
      <Icon className={s.noticeIcon} aria-hidden />
      <div className={s.noticeBody}>
        {title && <p className={s.noticeTitle}>{title}</p>}
        {children && <div className={s.noticeText}>{children}</div>}
        {technical && (
          <details className={s.tech}>
            <summary>Chi tiết kỹ thuật</summary>
            <div>{technical}</div>
          </details>
        )}
      </div>
      {action && <div className={s.noticeAction}>{action}</div>}
    </div>
  );
}

export type StatusTone = "red" | "amber" | "green" | "blue" | "neutral";

/** Trạng thái bằng chữ + chấm nhỏ; không dùng màu làm tín hiệu duy nhất (DESIGN.md §10.6). */
export function StatusText({ tone = "neutral", children, chip }: { tone?: StatusTone; children: ReactNode; chip?: boolean }) {
  return (
    <span className={[s.status, chip ? s.chip : "", s[`t_${tone}`]].join(" ")}>
      <span className={s.statusDot} aria-hidden />
      {children}
    </span>
  );
}

/** Trạng thái rỗng dạy việc kế tiếp (DESIGN.md §15). */
export function EmptyState({ title, children, action, icon }: { title: ReactNode; children?: ReactNode; action?: ReactNode; icon?: ReactNode }) {
  return (
    <div className={s.empty}>
      {icon && <div className={s.emptyIcon}>{icon}</div>}
      <p className="ep-item-title">{title}</p>
      {children && <p className={s.emptyText}>{children}</p>}
      {action && <div className={s.emptyAction}>{action}</div>}
    </div>
  );
}

/** Khung chờ có hình giống nội dung thật (không spinner giữa màn). */
export function Skeleton({ lines = 3, className }: { lines?: number; className?: string }) {
  return (
    <div className={[s.skeleton, className ?? ""].join(" ")} role="group" aria-busy="true" aria-label="Đang tải">
      {Array.from({ length: lines }, (_, i) => (
        <span key={i} className={s.skLine} style={{ width: `${92 - ((i * 17) % 40)}%` }} />
      ))}
    </div>
  );
}

/**
 * Xác nhận tĩnh lặng gắn tại chỗ vừa thao tác: "Đã đánh vắng · Hoàn tác", tự biến sau 5 s
 * (INTEGRATION.md mục 2 #4). Không toast "Thành công".
 */
export function UndoLine({ message, onUndo, onDone }: { message: ReactNode; onUndo?: () => void; onDone: () => void }) {
  useEffect(() => {
    const t = window.setTimeout(onDone, 5000);
    return () => window.clearTimeout(t);
  }, [onDone]);
  return (
    <p className={s.undo} role="status">
      <CircleCheck aria-hidden />
      <span>{message}</span>
      {onUndo && (
        <button
          type="button"
          className={s.undoBtn}
          onClick={() => {
            onUndo();
            onDone();
          }}
        >
          Hoàn tác
        </button>
      )}
    </p>
  );
}

export function Kbd({ children }: { children: ReactNode }) {
  return <kbd className={s.kbd}>{children}</kbd>;
}

/** Nhãn riêng tư cho nội dung chỉ một số vai trò thấy. */
export function PrivateMark({ children = "Chỉ giảng viên/TA thấy" }: { children?: ReactNode }) {
  return (
    <span className={s.private}>
      <ShieldCheck aria-hidden />
      {children}
    </span>
  );
}
