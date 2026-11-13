import Link from "next/link";
import { useId, type ComponentProps, type ReactNode } from "react";
import s from "./Button.module.css";

export type ButtonVariant = "primary" | "secondary" | "ghost" | "text";
type Common = { variant?: ButtonVariant; size?: "md" | "sm"; icon?: ReactNode; iconEnd?: ReactNode };

function classes(variant: ButtonVariant, size: "md" | "sm", extra?: string) {
  return [s.btn, s[variant], size === "sm" ? s.sm : "", extra ?? ""].join(" ");
}

/**
 * Nút hành động. Mỗi vùng làm việc chỉ một `primary` (DESIGN.md §12).
 * `disabledReason` (FEAT-uiux-91 US-03): nút `disabled` KÈM lý do thì dùng `aria-disabled` thay thuộc tính gốc — vẫn focus được bằng bàn phím,
 * `onClick` bị chặn, lý do hiện khi rê chuột / focus / chạm (đọc được qua `aria-describedby`). Không có `disabled` thì bỏ qua lý do.
 * `hint` (US-UX91-01 AC5): một câu ≤ 12 từ về hệ quả / cú pháp của một nút đang dùng được; cùng cách hiện với lý do khoá, nút vẫn bấm bình thường.
 */
export function Button({ variant = "secondary", size = "md", icon, iconEnd, loading, className, children, disabledReason, hint, ...rest }: Common & ComponentProps<"button"> & { loading?: boolean; disabledReason?: string; hint?: string }) {
  const reasonId = useId();
  const blocked = !!disabledReason && !!rest.disabled && !loading;
  const tip = blocked ? disabledReason : hint;
  const inner = (
    <>
      {loading ? <span className={s.spinner} aria-hidden /> : icon}
      {children}
      {iconEnd}
    </>
  );
  if (!tip) {
    return (
      <button type="button" data-variant={variant} className={classes(variant, size, className)} aria-busy={loading || undefined} {...rest} disabled={rest.disabled || loading}>
        {inner}
      </button>
    );
  }
  return (
    <span className={s.reasonWrap}>
      {blocked ? (
        <button {...rest} onClick={undefined} onSubmit={undefined} disabled={undefined} type="button" data-variant={variant} className={classes(variant, size, className)} aria-disabled="true" aria-describedby={reasonId}>
          {inner}
        </button>
      ) : (
        <button type="button" data-variant={variant} className={classes(variant, size, className)} aria-busy={loading || undefined} aria-describedby={reasonId} {...rest} disabled={rest.disabled || loading}>
          {inner}
        </button>
      )}
      <span id={reasonId} role="tooltip" className={s.reason}>{tip}</span>
    </span>
  );
}

export function ButtonLink({ variant = "secondary", size = "md", icon, iconEnd, className, children, ...rest }: Common & ComponentProps<typeof Link>) {
  return (
    <Link data-variant={variant} className={classes(variant, size, className)} {...rest}>
      {icon}
      {children}
      {iconEnd}
    </Link>
  );
}

/** Nút chỉ có icon — bắt buộc có nhãn cho trình đọc màn hình. */
export function IconButton({ label, children, className, size = "md", ...rest }: ComponentProps<"button"> & { label: string; size?: "md" | "sm" }) {
  return (
    <button type="button" aria-label={label} title={label} className={[s.iconBtn, size === "sm" ? s.iconSm : "", className ?? ""].join(" ")} {...rest}>
      {children}
    </button>
  );
}
