"use client";

import { useState } from "react";
import { ApiError, fieldErrors } from "@/shared/data";
import { Button, Field, InlineNotice, PasswordInput, Section } from "@/shared/ui";
import { useChangePassword } from "./api";
import s from "./security.module.css";

export function PasswordSection() {
  const change = useChangePassword();
  const [cur, setCur] = useState("");
  const [next, setNext] = useState("");
  const [again, setAgain] = useState("");
  const [errors, setErrors] = useState<Record<string, string>>({});
  const [problem, setProblem] = useState<string | null>(null);
  const [done, setDone] = useState(false);

  function submit(e: React.FormEvent) {
    e.preventDefault();
    if (change.isPending) return;
    setDone(false);
    setProblem(null);
    if (next !== again) {
      setErrors({ again: "Hai mật khẩu chưa giống nhau." });
      return;
    }
    setErrors({});
    change.mutate(
      { current_password: cur, new_password: next },
      {
        onSuccess: () => {
          setCur("");
          setNext("");
          setAgain("");
          setDone(true);
        },
        onError: (err) => {
          const fe = fieldErrors(err);
          if (Object.keys(fe).length) setErrors({ ...(fe.current_password ? { cur: fe.current_password } : {}), ...(fe.new_password ? { next: fe.new_password } : {}) });
          else setProblem(err instanceof ApiError ? err.userMessage : "Có lỗi xảy ra. Hãy thử lại.");
        },
      },
    );
  }

  return (
    <Section title="Mật khẩu" description="Đổi mật khẩu sẽ đăng xuất mọi thiết bị khác; thiết bị này vẫn đăng nhập.">
      <form onSubmit={submit} className={s.form} noValidate>
        {problem && <InlineNotice tone="danger" compact>{problem}</InlineNotice>}
        <Field label="Mật khẩu hiện tại" error={errors.cur}>
          {(id, d) => <PasswordInput id={id} aria-describedby={d} invalid={!!errors.cur} autoComplete="current-password" required value={cur} onChange={(e) => setCur(e.target.value)} />}
        </Field>
        <Field label="Mật khẩu mới" helper="Ít nhất 10 ký tự, không phải mật khẩu phổ biến." error={errors.next}>
          {(id, d) => <PasswordInput id={id} aria-describedby={d} invalid={!!errors.next} autoComplete="new-password" required value={next} onChange={(e) => setNext(e.target.value)} />}
        </Field>
        <Field label="Nhập lại mật khẩu mới" error={errors.again}>
          {(id, d) => <PasswordInput id={id} aria-describedby={d} invalid={!!errors.again} autoComplete="new-password" required value={again} onChange={(e) => setAgain(e.target.value)} />}
        </Field>
        <div className={s.actions}>
          <Button type="submit" variant="primary" loading={change.isPending} disabled={!cur || !next || !again}>
            Đổi mật khẩu
          </Button>
          {done && <p role="status" className={s.done}>Đã đổi mật khẩu. Các thiết bị khác đã bị đăng xuất.</p>}
        </div>
      </form>
    </Section>
  );
}
