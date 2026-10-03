"use client";

import { useParams, useRouter } from "next/navigation";
import { useEffect, useRef, useState } from "react";
import { ApiError, acceptSession, apiClient, fieldErrors, type SessionPayload } from "@/shared/data";
import { Button, Field, InlineNotice, PasswordInput, Skeleton } from "@/shared/ui";
import s from "@/app/login/login.module.css";

type Preview = { full_name?: string; role?: string };
type State = "checking" | "form" | "invalid";

const ROLE_VN: Record<string, string> = { TEACHER: "Giảng viên", TA: "Trợ giảng" };

/** Nhận lời mời: xem trước liên kết khi mở, người được mời tự đặt mật khẩu (hai ô) rồi vào thẳng `/`. Token chỉ nằm trong bộ nhớ trang. */
export function InviteAccept() {
  const params = useParams<{ token: string }>();
  const router = useRouter();
  const token = useRef<string | null>(params.token ?? null);
  const started = useRef(false);
  const [state, setState] = useState<State>(params.token ? "checking" : "invalid");
  const [who, setWho] = useState<Preview>({});
  const [pw, setPw] = useState("");
  const [again, setAgain] = useState("");
  const [errors, setErrors] = useState<Record<string, string>>({});
  const [problem, setProblem] = useState<string | null>(null);
  const [busy, setBusy] = useState(false);

  useEffect(() => {
    if (!token.current || started.current) return;
    started.current = true;
    window.history.replaceState(null, "", "/invite/%C2%B7"); // thay token bằng dấu giữ chỗ "·": không lọt vào lịch sử, Referer, ảnh chụp
    apiClient
      .post<Preview>("/auth/tokens/preview", { kind: "INVITE", token: token.current })
      .then(({ data }) => {
        setWho(data);
        setState("form");
      })
      .catch((err: unknown) => {
        if (!(err instanceof ApiError && err.code === "LINK_INVALID")) setProblem(err instanceof ApiError ? err.userMessage : "Có lỗi xảy ra. Hãy thử lại.");
        setState("invalid");
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
      const { data } = await apiClient.post<SessionPayload>("/auth/accept-invite", { token: token.current, password: pw });
      token.current = null;
      acceptSession(data);
      router.replace("/");
    } catch (err) {
      if (err instanceof ApiError && err.code === "LINK_INVALID") setState("invalid");
      else {
        const fe = fieldErrors(err);
        if (fe.password) setErrors({ password: fe.password });
        else setProblem(err instanceof ApiError ? err.userMessage : "Có lỗi xảy ra. Hãy thử lại.");
      }
    } finally {
      setBusy(false);
    }
  }

  if (state === "checking") {
    return (
      <>
        <h1 className="ep-page-title">Đang kiểm tra lời mời…</h1>
        <Skeleton lines={3} />
      </>
    );
  }
  if (state === "invalid") {
    return (
      <>
        <h1 className="ep-page-title">Lời mời không dùng được</h1>
        <p>{problem ?? "Lời mời đã hết hạn hoặc đã được dùng. Hãy nhờ quản trị viên gửi lại."}</p>
      </>
    );
  }
  return (
    <>
      <h1 className="ep-page-title">
        Chào {who.full_name}, bạn được mời làm {ROLE_VN[who.role ?? ""] ?? "thành viên"} trên EduPilot.
      </h1>
      {problem && <InlineNotice tone="danger" compact>{problem}</InlineNotice>}
      <form onSubmit={submit} className={s.form} noValidate>
        <Field label="Mật khẩu" helper="Ít nhất 10 ký tự, không phải mật khẩu phổ biến." error={errors.password}>
          {(id, d) => <PasswordInput id={id} aria-describedby={d} invalid={!!errors.password} autoComplete="new-password" required value={pw} onChange={(e) => setPw(e.target.value)} />}
        </Field>
        <Field label="Nhập lại mật khẩu" error={errors.again}>
          {(id, d) => <PasswordInput id={id} aria-describedby={d} invalid={!!errors.again} autoComplete="new-password" required value={again} onChange={(e) => setAgain(e.target.value)} />}
        </Field>
        <Button type="submit" variant="primary" loading={busy} disabled={!pw || !again}>
          Đặt mật khẩu và vào
        </Button>
      </form>
    </>
  );
}
