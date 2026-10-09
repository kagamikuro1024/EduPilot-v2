"use client";

import { AuthPanel } from "@/shared/shell/AuthShell";
import Link from "next/link";
import { useRouter, useSearchParams } from "next/navigation";
import { useEffect, useRef, useState } from "react";
import { acceptSession, apiClient, ApiError, type SessionPayload } from "@/shared/data";
import { useAuth } from "@/shared/session/AuthProvider";
import { safeNext } from "@/shared/session/safeNext";
import { Button, Field, Input, InlineNotice, PasswordInput } from "@/shared/ui";
import s from "./login.module.css";

const REVOKED_COPY: Record<string, string> = {
  password_changed: "Bạn đã bị đăng xuất vì mật khẩu của tài khoản vừa được đổi.",
  password_reset: "Bạn đã bị đăng xuất vì mật khẩu của tài khoản vừa được đổi.",
};
const REVOKED_DEFAULT = "Bạn đã bị đăng xuất. Hãy đăng nhập lại.";

const mmss = (sec: number) => `${String(Math.floor(sec / 60)).padStart(2, "0")}:${String(sec % 60).padStart(2, "0")}`;

export function LoginForm() {
  const router = useRouter();
  const params = useSearchParams();
  const auth = useAuth();
  const next = safeNext(params.get("next"));
  const revoked = params.get("revoked") ?? (auth.status === "revoked" ? (auth.reason ?? "1") : null);

  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [wait, setWait] = useState(0); // giây còn phải chờ (LOGIN_THROTTLED)
  const passwordRef = useRef<HTMLInputElement>(null);

  // Đã có phiên (mở /login khi đang đăng nhập): vào thẳng đích an toàn.
  useEffect(() => {
    if (auth.status === "authenticated") router.replace(next);
  }, [auth.status, next, router]);

  useEffect(() => {
    if (wait <= 0) return;
    const t = setTimeout(() => setWait((w) => w - 1), 1000);
    return () => clearTimeout(t);
  }, [wait]);

  async function submit(e: React.FormEvent) {
    e.preventDefault();
    if (busy || wait > 0) return;
    setBusy(true);
    setError(null);
    try {
      const { data } = await apiClient.post<SessionPayload>("/auth/login", { email: email.trim(), password });
      acceptSession(data);
      setPassword("");
      router.replace(next);
    } catch (err) {
      if (err instanceof ApiError && (err.code === "LOGIN_THROTTLED" || err.code === "RATE_LIMITED") && err.retryAfter) {
        setWait(Math.ceil(err.retryAfter)); // bị chờ / khoá / vượt giới hạn: cùng một câu, không gợi ý email có tồn tại; giữ nguyên chữ đã gõ
        setError(null);
      } else {
        setPassword("");
        setError(err instanceof ApiError ? err.userMessage : "Có lỗi xảy ra. Hãy thử lại.");
      }
      passwordRef.current?.focus();
    } finally {
      setBusy(false);
    }
  }

  return (
    <AuthPanel title="Đăng nhập EduPilot">
      {revoked && (
        <InlineNotice tone="warning" compact>
          {REVOKED_COPY[revoked] ?? REVOKED_DEFAULT}
        </InlineNotice>
      )}
      <form onSubmit={submit} className={s.form} noValidate>
        <Field label="Email">
          {(id, d) => (
            <Input id={id} aria-describedby={d} type="email" inputMode="email" autoComplete="username" autoCapitalize="none" spellCheck={false} required value={email} onChange={(e) => setEmail(e.target.value)} />
          )}
        </Field>
        <Field label="Mật khẩu">
          {(id, d) => (
            <PasswordInput id={id} ref={passwordRef} aria-describedby={d} autoComplete="current-password" required value={password} onChange={(e) => setPassword(e.target.value)} />
          )}
        </Field>
        {wait > 0 && (
          <p role="status" id="login-wait" className={s.problem}>
            Bạn đã thử quá nhiều lần. Thử lại sau {mmss(wait)}.
          </p>
        )}
        {error && (
          <p role="alert" className={s.problem}>
            {error}
          </p>
        )}
        <Button type="submit" variant="primary" loading={busy} disabled={wait > 0 || !email.trim() || !password} aria-describedby={wait > 0 ? "login-wait" : undefined}>
          Đăng nhập
        </Button>
      </form>
      <p className={s.links}>
        <Link href="/forgot-password">Quên mật khẩu?</Link>
        <Link href="/register">Chưa có tài khoản? Đăng ký</Link>
      </p>
    </AuthPanel>
  );
}
