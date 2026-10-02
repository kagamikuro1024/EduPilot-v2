"use client";

import { Send, Square } from "lucide-react";
import { useEffect, useRef, type ReactNode } from "react";
import s from "./Composer.module.css";

/**
 * Vùng soạn chat / thread (DESIGN.md §10.12): tối đa 820px, một nút gửi chính,
 * công cụ phụ ở trái, thông báo (vd. bảo vệ thông tin cá nhân) ngay phía trên.
 * Enter gửi, Shift+Enter xuống dòng. Không bao giờ xoá chữ khi gửi lỗi.
 */
export function Composer({
  value,
  onChange,
  onSubmit,
  placeholder,
  notice,
  tools,
  busy,
  onStop,
  submitLabel = "Gửi",
  disabled,
  hint,
  label = "Nội dung",
}: {
  value: string;
  onChange: (v: string) => void;
  onSubmit: () => void;
  placeholder?: string;
  notice?: ReactNode;
  tools?: ReactNode;
  busy?: boolean;
  onStop?: () => void;
  submitLabel?: string;
  disabled?: boolean;
  /** Dòng chữ cạnh nút gửi khi nút bị khoá — nói còn thiếu gì (FR-X19). */
  hint?: ReactNode;
  label?: string;
}) {
  const ref = useRef<HTMLTextAreaElement>(null);

  useEffect(() => {
    const el = ref.current;
    if (!el) return;
    el.style.height = "auto";
    el.style.height = `${Math.min(el.scrollHeight, 220)}px`;
  }, [value]);

  const canSend = value.trim().length > 0 && !busy && !disabled;

  return (
    <div className={s.wrap}>
      {notice && <div className={s.notice}>{notice}</div>}
      <form
        className={s.composer}
        onSubmit={(e) => {
          e.preventDefault();
          if (canSend) onSubmit();
        }}
      >
        <textarea
          id="ep-composer"
          aria-label={label}
          ref={ref}
          rows={1}
          className={s.input}
          value={value}
          placeholder={placeholder}
          disabled={disabled}
          onChange={(e) => onChange(e.target.value)}
          onKeyDown={(e) => {
            if (e.key === "Enter" && !e.shiftKey && !e.nativeEvent.isComposing) {
              e.preventDefault();
              if (canSend) onSubmit();
            }
          }}
        />
        <div className={s.bar}>
          <div className={s.tools}>{tools}</div>
          {busy && onStop ? (
            <button type="button" className={s.stop} onClick={onStop}>
              <Square aria-hidden />
              Dừng
            </button>
          ) : (
            <div className={s.sendGroup}>
              {!canSend && hint && (
                <p id="ep-composer-hint" className={s.hint}>
                  {hint}
                </p>
              )}
              <button type="submit" className={s.send} disabled={!canSend} aria-describedby={!canSend && hint ? "ep-composer-hint" : undefined}>
                <Send aria-hidden />
                {submitLabel}
              </button>
            </div>
          )}
        </div>
      </form>
    </div>
  );
}
