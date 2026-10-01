"use client";

import { X } from "lucide-react";
import { useEffect, useId, useRef, type ReactNode } from "react";
import s from "./Dialog.module.css";

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
}: {
  open: boolean;
  onClose: () => void;
  title: ReactNode;
  description?: ReactNode;
  children?: ReactNode;
  footer?: ReactNode;
  variant: "dialog" | "drawer";
  wide?: boolean;
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
      onClick={(e) => e.target === ref.current && onClose()}
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
          <button type="button" className={s.close} onClick={onClose} aria-label="Đóng">
            <X aria-hidden />
          </button>
        </header>
        {children && <div className={s.body}>{children}</div>}
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
