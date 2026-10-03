"use client";

import { Eye, EyeOff } from "lucide-react";
import { useId, useState, type ComponentProps, type ReactNode } from "react";
import s from "./Field.module.css";

/** Nhãn ở trên, gợi ý / lỗi ở dưới (DESIGN.md §10.9). Truyền render-prop để nhận id. */
export function Field({
  label,
  helper,
  error,
  required,
  children,
  className,
}: {
  label: ReactNode;
  helper?: ReactNode;
  error?: ReactNode;
  required?: boolean;
  children: (id: string, describedBy?: string) => ReactNode;
  className?: string;
}) {
  const id = useId();
  const noteId = helper || error ? `${id}-note` : undefined;
  return (
    <div className={[s.field, className ?? ""].join(" ")}>
      <label htmlFor={id} className={s.label}>
        {label}
        {required && <span className={s.req}> · bắt buộc</span>}
      </label>
      {children(id, noteId)}
      {(error || helper) && (
        <p id={noteId} className={error ? s.error : s.helper}>
          {error ?? helper}
        </p>
      )}
    </div>
  );
}

export function Input({ className, invalid, ...rest }: ComponentProps<"input"> & { invalid?: boolean }) {
  return <input className={[s.control, className ?? ""].join(" ")} aria-invalid={invalid || undefined} {...rest} />;
}

/** Ô mật khẩu có nút hiện / ẩn (vùng chạm 44 px). Mặc định ẩn. */
export function PasswordInput({ className, invalid, ...rest }: Omit<ComponentProps<"input">, "type"> & { invalid?: boolean }) {
  const [shown, setShown] = useState(false);
  return (
    <span className={s.pw}>
      <input className={[s.control, className ?? ""].join(" ")} aria-invalid={invalid || undefined} {...rest} type={shown ? "text" : "password"} />
      <button type="button" className={s.pwToggle} onClick={() => setShown((v) => !v)} aria-pressed={shown} aria-label={shown ? "Ẩn mật khẩu" : "Hiện mật khẩu"}>
        {shown ? <EyeOff aria-hidden /> : <Eye aria-hidden />}
      </button>
    </span>
  );
}

export function Textarea({ className, invalid, ...rest }: ComponentProps<"textarea"> & { invalid?: boolean }) {
  return <textarea className={[s.control, s.textarea, className ?? ""].join(" ")} aria-invalid={invalid || undefined} {...rest} />;
}

export function Select({ className, children, ...rest }: ComponentProps<"select">) {
  return (
    <select className={[s.control, s.select, className ?? ""].join(" ")} {...rest}>
      {children}
    </select>
  );
}

export function Checkbox({ label, className, ...rest }: ComponentProps<"input"> & { label: ReactNode }) {
  return (
    <label className={[s.check, className ?? ""].join(" ")}>
      <input type="checkbox" {...rest} />
      <span>{label}</span>
    </label>
  );
}

/** Công tắc bật/tắt cho cài đặt có hiệu lực ngay. */
export function Switch({
  checked,
  onChange,
  label,
  description,
  disabled,
}: {
  checked: boolean;
  onChange: (next: boolean) => void;
  label: ReactNode;
  description?: ReactNode;
  disabled?: boolean;
}) {
  const id = useId();
  return (
    <div className={s.switchRow}>
      <div>
        <label htmlFor={id} className={s.label}>
          {label}
        </label>
        {description && <p className={s.helper}>{description}</p>}
      </div>
      <button
        id={id}
        type="button"
        role="switch"
        aria-checked={checked}
        disabled={disabled}
        className={s.switch}
        onClick={() => onChange(!checked)}
      >
        <span className={s.knob} />
      </button>
    </div>
  );
}
