"use client";

import Link from "next/link";
import { useEffect, useState } from "react";
import { apiClient, ApiError, fieldErrors } from "@/shared/data";
import { Button, Field, InlineNotice, Input } from "@/shared/ui";
import s from "@/app/login/login.module.css";

const RESEND_SECONDS = 60;
const SENT = "Nếu email này có tài khoản, chúng tôi đã gửi hướng dẫn đặt lại mật khẩu.";

export function ForgotForm() {
  const [email, setEmail] = useState("");
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState<string | undefined>();
  const [problem, setProblem] = useState<string | null>(null);
  const [sent, setSent] = useState(false);
  const [wait, setWait] = useState(0);

  useEffect(() => {
    if (wait <= 0) return;
    const t = setTimeout(() => setWait((w) => w - 1), 1000);
    return () => clearTimeout(t);
  }, [wait]);

  async function submit(e: React.FormEvent) {
    e.preventDefault();
    if (busy || wait > 0) return;
    setBusy(true);
    setError(undefined);
    setProblem(null);
    try {
      await apiClient.post("/auth/forgot-password", { email: email.trim() });
      setSent(true);
      setWait(RESEND_SECONDS);
    } catch (err) {
      if (err instanceof ApiError && err.code === "RATE_LIMITED") {
        setSent(true); // không lộ gì: vẫn là màn "đã gửi"; chỉ khoá nút theo thời gian máy chủ báo
        setWait(err.retryAfter ?? RESEND_SECONDS);
      } else {
        const fe = fieldErrors(err);
        if (fe.email) setError(fe.email);
        else setProblem(err instanceof ApiError ? err.userMessage : "Có lỗi xảy ra. Hãy thử lại.");
      }
    } finally {
      setBusy(false);
    }
  }

  return (
    <>
      <h1 className="ep-page-title">Quên mật khẩu</h1>
      {sent ? (
        <p role="status">{SENT}</p>
      ) : (
        <p>Nhập email đã đăng ký. Chúng tôi sẽ gửi hướng dẫn đặt lại mật khẩu.</p>
      )}
      {problem && <InlineNotice tone="danger" compact>{problem}</InlineNotice>}
      <form onSubmit={submit} className={s.form} noValidate>
        <Field label="Email" error={error}>
          {(id, d) => (
            <Input id={id} aria-describedby={d} invalid={!!error} type="email" inputMode="email" autoComplete="username" autoCapitalize="none" spellCheck={false} required value={email} onChange={(e) => setEmail(e.target.value)} />
          )}
        </Field>
        <Button type="submit" variant="primary" loading={busy} disabled={!email.trim() || wait > 0}>
          {wait > 0 ? `Gửi lại hướng dẫn (${wait} giây)` : sent ? "Gửi lại hướng dẫn" : "Gửi hướng dẫn"}
        </Button>
      </form>
      <p className={s.links}>
        <Link href="/login">Quay lại đăng nhập</Link>
      </p>
    </>
  );
}
