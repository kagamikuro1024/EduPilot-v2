"use client";

import { useSearchParams } from "next/navigation";
import { useEffect, useRef, useState } from "react";
import { apiClient, ApiError } from "@/shared/data";
import { Button, ButtonLink, Field, Input, Skeleton } from "@/shared/ui";
import s from "@/app/login/login.module.css";

type State = "checking" | "ok" | "used" | "expired" | "invalid" | "error";

export function VerifyEmail() {
  const params = useSearchParams();
  const token = params.get("token");
  const [state, setState] = useState<State>(token ? "checking" : "invalid");
  const started = useRef(false);

  useEffect(() => {
    if (!token || started.current) return;
    started.current = true; // liên kết dùng MỘT lần: tuyệt đối không gửi hai lần (StrictMode, tải lại)
    // Xoá token khỏi thanh địa chỉ ngay: không để lọt vào lịch sử / Referer / ảnh chụp màn hình.
    window.history.replaceState(null, "", window.location.pathname);
    apiClient
      .post("/auth/verify-email", { token })
      .then(() => setState("ok"))
      .catch((err: unknown) => {
        if (err instanceof ApiError && err.code === "LINK_INVALID") {
          const reason = (err.details as { reason?: string } | undefined)?.reason;
          setState(reason === "used" ? "used" : reason === "expired" ? "expired" : "invalid");
        } else setState("error");
      });
  }, [token]);

  if (state === "checking") {
    return (
      <>
        <h1 className="ep-page-title">Đang xác minh email…</h1>
        <Skeleton lines={3} />
      </>
    );
  }
  if (state === "ok") {
    return (
      <>
        <h1 className="ep-page-title">Email đã được xác minh.</h1>
        <p>Bạn có thể đăng nhập và tham gia lớp.</p>
        <ButtonLink href="/login" variant="primary">Đăng nhập</ButtonLink>
      </>
    );
  }
  if (state === "used") {
    return (
      <>
        <h1 className="ep-page-title">Liên kết đã được dùng</h1>
        <p>Liên kết này đã được dùng. Nếu bạn đã xác minh, hãy đăng nhập.</p>
        <ButtonLink href="/login" variant="primary">Đăng nhập</ButtonLink>
      </>
    );
  }
  if (state === "error") {
    return (
      <>
        <h1 className="ep-page-title">Chưa xác minh được</h1>
        <p>Có lỗi xảy ra. Liên kết của bạn chưa bị dùng. Hãy mở lại liên kết trong thư sau ít phút.</p>
      </>
    );
  }
  return <Expired expired={state === "expired"} />;
}

function Expired({ expired }: { expired: boolean }) {
  const [email, setEmail] = useState("");
  const [note, setNote] = useState<string | null>(null);
  const [wait, setWait] = useState(0);
  const [busy, setBusy] = useState(false);

  useEffect(() => {
    if (wait <= 0) return;
    const t = setTimeout(() => setWait((w) => w - 1), 1000);
    return () => clearTimeout(t);
  }, [wait]);

  async function resend(e: React.FormEvent) {
    e.preventDefault();
    if (busy || wait > 0) return;
    setBusy(true);
    setNote(null);
    try {
      await apiClient.post("/auth/resend-verification", { email: email.trim() });
      setNote("Nếu email này cần xác minh, chúng tôi đã gửi lại thư.");
      setWait(60);
    } catch (err) {
      if (err instanceof ApiError && err.code === "RATE_LIMITED" && err.retryAfter) setWait(err.retryAfter);
      else setNote(err instanceof ApiError ? err.userMessage : "Có lỗi xảy ra. Hãy thử lại.");
    } finally {
      setBusy(false);
    }
  }

  return (
    <>
      <h1 className="ep-page-title">{expired ? "Liên kết đã hết hạn." : "Liên kết không dùng được"}</h1>
      <p>Nhập email để nhận thư xác minh mới.</p>
      <form onSubmit={resend} className={s.form} noValidate>
        <Field label="Email">
          {(id, d) => <Input id={id} aria-describedby={d} type="email" inputMode="email" autoComplete="username" autoCapitalize="none" spellCheck={false} required value={email} onChange={(e) => setEmail(e.target.value)} />}
        </Field>
        <Button type="submit" variant="primary" loading={busy} disabled={!email.trim() || wait > 0}>
          {wait > 0 ? `Gửi lại thư (${wait} giây)` : "Gửi lại thư"}
        </Button>
      </form>
      {note && <p role="status" className={s.hint}>{note}</p>}
    </>
  );
}
