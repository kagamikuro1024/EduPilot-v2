"use client";

import { X } from "lucide-react";
import { useEffect, useId, useRef, type ReactNode } from "react";
import { InlineNotice } from "./Feedback";
import s from "./Dialog.module.css";

/** Bẫy Tab trong hộp: Tab ở phần tử cuối quay về đầu, Shift+Tab ở đầu nhảy về cuối (không thoát ra giao diện trình duyệt). */
function trapTab(e: React.KeyboardEvent<HTMLDialogElement>) {
  if (e.key !== "Tab") return;
  const items = Array.from(e.currentTarget.querySelectorAll<HTMLElement>('a[href], button:not(:disabled), input:not(:disabled), textarea:not(:disabled), select:not(:disabled), [tabindex]:not([tabindex="-1"])')).filter((x) => x.offsetParent !== null);
  if (items.length === 0) {
    e.preventDefault();
    return;
  }
  const first = items[0];
  const last = items[items.length - 1];
  const active = document.activeElement;
  if (e.shiftKey && (active === first || active === e.currentTarget)) {
    e.preventDefault();
    last.focus();
  } else if (!e.shiftKey && active === last) {
    e.preventDefault();
    first.focus();
  }
}

/**
 * Lớp phủ dùng <dialog> gốc của trình duyệt: bẫy focus, Esc để đóng, trả focus về chỗ cũ.
 * `variant="dialog"` CHỈ cho việc cần bảo vệ (DESIGN.md §10.11); `drawer` cho xem xét theo bối cảnh (§10.10).
 */
function Overlay({
  open,
  onClose,
  title,
  description,
  children,
  footer,
  variant,
  wide,
  dismissible = true,
  loading,
  error,
}: {
  open: boolean;
  onClose: () => void;
  title: ReactNode;
  description?: ReactNode;
  children?: ReactNode;
  footer?: ReactNode;
  variant: "dialog" | "drawer";
  wide?: boolean;
  /** false (đang xử lý việc không đảo ngược): Esc, bấm nền và nút Đóng đều không đóng được */
  dismissible?: boolean;
  /** nội dung đang chờ: khung xương + aria-busy */
  loading?: boolean;
  /** lỗi của thao tác trong hộp: role=alert, giữ nguyên hộp */
  error?: ReactNode;
}) {
  const ref = useRef<HTMLDialogElement>(null);
  const titleId = useId();

  useEffect(() => {
    const d = ref.current;
    if (!d) return;
    if (open && !d.open) d.showModal();
    if (!open && d.open) d.close();
  }, [open]);

  return (
    <dialog
      ref={ref}
      className={[s.overlay, variant === "drawer" ? s.drawer : s.dialog, wide ? s.wide : ""].join(" ")}
      onClose={onClose}
      onCancel={(e) => !dismissible && e.preventDefault()}
      onKeyDown={trapTab}
      onClick={(e) => dismissible && e.target === ref.current && onClose()}
      role="dialog"
      aria-labelledby={titleId}
    >
      <div className={s.frame}>
        <header className={s.head}>
          <div className={s.headText}>
            <h2 id={titleId} className="ep-section-title">
              {title}
            </h2>
            {description && <p className={s.desc}>{description}</p>}
          </div>
          <button type="button" className={s.close} onClick={onClose} aria-label="Đóng" disabled={!dismissible}>
            <X aria-hidden />
          </button>
        </header>
        {(children || loading || error) && (
          <div className={s.body} aria-busy={loading || undefined}>
            {error && (
              <InlineNotice tone="danger" compact>
                {error}
              </InlineNotice>
            )}
            {loading ? (
              <div className={s.skeleton}>
                <span />
                <span />
                <span />
              </div>
            ) : (
              children
            )}
          </div>
        )}
        {footer && <footer className={s.foot}>{footer}</footer>}
      </div>
    </dialog>
  );
}

type OverlayProps = Omit<Parameters<typeof Overlay>[0], "variant">;

export function Dialog(props: OverlayProps) {
  return <Overlay {...props} variant="dialog" />;
}

export function Drawer(props: OverlayProps) {
  return <Overlay {...props} variant="drawer" />;
}
