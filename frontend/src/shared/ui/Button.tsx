import Link from "next/link";
import type { ComponentProps, ReactNode } from "react";
import s from "./Button.module.css";

export type ButtonVariant = "primary" | "secondary" | "ghost" | "text";
type Common = { variant?: ButtonVariant; size?: "md" | "sm"; icon?: ReactNode; iconEnd?: ReactNode };

function classes(variant: ButtonVariant, size: "md" | "sm", extra?: string) {
  return [s.btn, s[variant], size === "sm" ? s.sm : "", extra ?? ""].join(" ");
}

/** Nút hành động. Mỗi vùng làm việc chỉ một `primary` (DESIGN.md §12). */
export function Button({ variant = "secondary", size = "md", icon, iconEnd, loading, className, children, ...rest }: Common & ComponentProps<"button"> & { loading?: boolean }) {
  return (
    <button type="button" className={classes(variant, size, className)} aria-busy={loading || undefined} {...rest} disabled={rest.disabled || loading}>
      {loading ? <span className={s.spinner} aria-hidden /> : icon}
      {children}
      {iconEnd}
    </button>
  );
}

export function ButtonLink({ variant = "secondary", size = "md", icon, iconEnd, className, children, ...rest }: Common & ComponentProps<typeof Link>) {
  return (
    <Link className={classes(variant, size, className)} {...rest}>
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
