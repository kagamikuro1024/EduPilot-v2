"use client";

import Link from "next/link";
import { useSearchParams } from "next/navigation";
import { useEffect, useRef, useState } from "react";
import { apiClient, ApiError, fieldErrors } from "@/shared/data";
import { Button, ButtonLink, Field, InlineNotice, PasswordInput, Skeleton } from "@/shared/ui";
import s from "@/app/login/login.module.css";

type State = "checking" | "form" | "done" | "invalid";

export function ResetPassword() {
  const params = useSearchParams();
  const initial = params.get("token");
  const token = useRef<string | null>(initial);
  const started = useRef(false);
  const [state, setState] = useState<State>(initial ? "checking" : "invalid");
  const [pw, setPw] = useState("");
  const [again, setAgain] = useState("");
  const [errors, setErrors] = useState<Record<string, string>>({});
  const [problem, setProblem] = useState<string | null>(null);
  const [busy, setBusy] = useState(false);

  useEffect(() => {
    if (!token.current || started.current) return;
    started.current = true;
    window.history.replaceState(null, "", window.location.pathname); // token chỉ còn trong bộ nhớ trang
    apiClient
      .post("/auth/tokens/preview", { kind: "RESET_PASSWORD", token: token.current })
      .then(() => setState("form"))
      .catch((err: unknown) => {
        if (err instanceof ApiError && err.code === "LINK_INVALID") setState("invalid");
        else {
          setProblem(err instanceof ApiError ? err.userMessage : "Có lỗi xảy ra. Hãy thử lại.");
          setState("invalid");
        }
      });
  }, []);

  async function submit(e: React.FormEvent) {
    e.preventDefault();
    if (busy) return;
    if (pw !== again) {
      setErrors({ again: "Hai mật khẩu chưa giống nhau." });
      return;
    }
    setBusy(true);
    setErrors({});
    setProblem(null);
    try {
      await apiClient.post("/auth/reset-password", { token: token.current, new_password: pw });
      token.current = null;
      setPw("");
      setAgain("");
      setState("done");
    } catch (err) {
      if (err instanceof ApiError && err.code === "LINK_INVALID") setState("invalid");
      else {
        const fe = fieldErrors(err);
        if (Object.keys(fe).length) setErrors({ ...(fe.new_password ? { password: fe.new_password } : {}) });
        else setProblem(err instanceof ApiError ? err.userMessage : "Có lỗi xảy ra. Hãy thử lại.");
      }
    } finally {
      setBusy(false);
    }
  }

  if (state === "checking") {
    return (
      <>
        <h1 className="ep-page-title">Đang kiểm tra liên kết…</h1>
        <Skeleton lines={3} />
      </>
    );
  }
  if (state === "done") {
    return (
      <>
        <h1 className="ep-page-title">Mật khẩu đã được đổi. Hãy đăng nhập lại.</h1>
        <ButtonLink href="/login" variant="primary">Đăng nhập</ButtonLink>
      </>
    );
  }
  if (state === "invalid") {
    return (
      <>
        <h1 className="ep-page-title">Liên kết không dùng được</h1>
        <p>{problem ?? "Liên kết đã hết hạn hoặc đã được dùng."}</p>
        <ButtonLink href="/forgot-password" variant="primary">Yêu cầu liên kết mới</ButtonLink>
      </>
    );
  }
  return (
    <>
      <h1 className="ep-page-title">Đặt mật khẩu mới</h1>
      {problem && <InlineNotice tone="danger" compact>{problem}</InlineNotice>}
      <form onSubmit={submit} className={s.form} noValidate>
        <Field label="Mật khẩu mới" helper="Ít nhất 10 ký tự, không phải mật khẩu phổ biến." error={errors.password}>
          {(id, d) => <PasswordInput id={id} aria-describedby={d} invalid={!!errors.password} autoComplete="new-password" required value={pw} onChange={(e) => setPw(e.target.value)} />}
        </Field>
        <Field label="Nhập lại mật khẩu" error={errors.again}>
          {(id, d) => <PasswordInput id={id} aria-describedby={d} invalid={!!errors.again} autoComplete="new-password" required value={again} onChange={(e) => setAgain(e.target.value)} />}
        </Field>
        <Button type="submit" variant="primary" loading={busy} disabled={!pw || !again}>
          Đổi mật khẩu
        </Button>
      </form>
      <p className={s.links}>
        <Link href="/login">Quay lại đăng nhập</Link>
      </p>
    </>
  );
}
