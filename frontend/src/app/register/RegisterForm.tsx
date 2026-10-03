"use client";

import Link from "next/link";
import { useEffect, useState } from "react";
import { apiClient, ApiError, fieldErrors } from "@/shared/data";
import { Button, Field, Input, InlineNotice, PasswordInput } from "@/shared/ui";
import s from "@/app/login/login.module.css";

const RESEND_SECONDS = 60;
const SENT = "Nếu email này dùng được, chúng tôi đã gửi thư xác nhận.";

export function RegisterForm() {
  const [fullName, setFullName] = useState("");
  const [email, setEmail] = useState("");
  const [code, setCode] = useState("");
  const [password, setPassword] = useState("");
  const [busy, setBusy] = useState(false);
  const [errors, setErrors] = useState<Record<string, string>>({});
  const [problem, setProblem] = useState<string | null>(null);
  const [sent, setSent] = useState(false);
  const [wait, setWait] = useState(0);
  const [resendNote, setResendNote] = useState<string | null>(null);

  useEffect(() => {
    if (wait <= 0) return;
    const t = setTimeout(() => setWait((w) => w - 1), 1000);
    return () => clearTimeout(t);
  }, [wait]);

  async function submit(e: React.FormEvent) {
    e.preventDefault();
    if (busy) return;
    setBusy(true);
    setErrors({});
    setProblem(null);
    try {
      await apiClient.post("/auth/register", { email: email.trim(), password, full_name: fullName.trim(), ...(code.trim() ? { student_code: code.trim() } : {}) });
      setSent(true);
      setWait(RESEND_SECONDS);
      setPassword("");
    } catch (err) {
      const fe = fieldErrors(err);
      if (Object.keys(fe).length) setErrors(fe);
      else setProblem(err instanceof ApiError ? err.userMessage : "Có lỗi xảy ra. Hãy thử lại.");
    } finally {
      setBusy(false);
    }
  }

  async function resend() {
    if (wait > 0) return;
    setResendNote(null);
    try {
      await apiClient.post("/auth/resend-verification", { email: email.trim() });
      setResendNote("Đã gửi lại thư.");
      setWait(RESEND_SECONDS);
    } catch (err) {
      if (err instanceof ApiError && err.code === "RATE_LIMITED" && err.retryAfter) setWait(err.retryAfter);
      else setResendNote(err instanceof ApiError ? err.userMessage : "Có lỗi xảy ra. Hãy thử lại.");
    }
  }

  if (sent) {
    return (
      <>
        <h1 className="ep-page-title">Kiểm tra email của bạn</h1>
        <p>{SENT}</p>
        <p className={s.hint}>Thư có thể nằm trong mục thư rác. Liên kết dùng một lần và có hiệu lực 24 giờ.</p>
        <Button variant="secondary" onClick={resend} disabled={wait > 0}>
          {wait > 0 ? `Gửi lại thư (${wait} giây)` : "Gửi lại thư"}
        </Button>
        {resendNote && <p role="status" className={s.hint}>{resendNote}</p>}
        <p className={s.links}>
          <Link href="/login">Đã xác minh? Đăng nhập</Link>
        </p>
      </>
    );
  }

  return (
    <>
      <h1 className="ep-page-title">Tạo tài khoản EduPilot</h1>
      {problem && <InlineNotice tone="danger" compact>{problem}</InlineNotice>}
      <form onSubmit={submit} className={s.form} noValidate>
        <Field label="Họ và tên" error={errors.full_name}>
          {(id, d) => <Input id={id} aria-describedby={d} invalid={!!errors.full_name} autoComplete="name" required value={fullName} onChange={(e) => setFullName(e.target.value)} />}
        </Field>
        <Field label="Email" error={errors.email}>
          {(id, d) => (
            <Input id={id} aria-describedby={d} invalid={!!errors.email} type="email" inputMode="email" autoComplete="username" autoCapitalize="none" spellCheck={false} required value={email} onChange={(e) => setEmail(e.target.value)} />
          )}
        </Field>
        <Field label="Mã số sinh viên (không bắt buộc)" helper="Chỉ để giảng viên đối chiếu; không dùng để vào lớp." error={errors.student_code}>
          {(id, d) => <Input id={id} aria-describedby={d} invalid={!!errors.student_code} autoComplete="off" autoCapitalize="characters" spellCheck={false} value={code} onChange={(e) => setCode(e.target.value)} />}
        </Field>
        <Field label="Mật khẩu" helper="Ít nhất 10 ký tự, không phải mật khẩu phổ biến." error={errors.password}>
          {(id, d) => <PasswordInput id={id} aria-describedby={d} invalid={!!errors.password} autoComplete="new-password" required value={password} onChange={(e) => setPassword(e.target.value)} />}
        </Field>
        <Button type="submit" variant="primary" loading={busy} disabled={!fullName.trim() || !email.trim() || !password}>
          Tạo tài khoản
        </Button>
      </form>
      <p className={s.links}>
        <Link href="/login">Đã có tài khoản? Đăng nhập</Link>
      </p>
    </>
  );
}
