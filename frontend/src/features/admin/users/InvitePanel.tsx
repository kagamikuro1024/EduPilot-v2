"use client";

import { useState } from "react";
import { ApiError, apiClient, fieldErrors, newIdempotencyKey } from "@/shared/data";
import { useIdempotentMutation } from "@/shared/data/useIdempotentMutation";
import { Button, Field, InlineNotice, Input, Section, Select } from "@/shared/ui";
import s from "../admin.module.css";
import type { AdminUser } from "./api";

/** Mời giảng viên / trợ giảng: khung mở dần tại chỗ (không hộp thoại). Gửi lại sau lỗi dùng CÙNG khoá Idempotency-Key. */
export function InvitePanel({ onSent, onClose }: { onSent: (u: AdminUser) => void; onClose: () => void }) {
  const [email, setEmail] = useState("");
  const [name, setName] = useState("");
  const [role, setRole] = useState<"TEACHER" | "TA">("TEACHER");
  const m = useIdempotentMutation((v: { email: string; full_name: string; role: string }, key) => apiClient.post<AdminUser>("/admin/users", v, { idempotencyKey: key || newIdempotencyKey() }));

  const err = m.error;
  const fe = err ? fieldErrors(err) : {};
  const dup = err instanceof ApiError && err.code === "CONFLICT" && (err.details as { field?: string } | undefined)?.field === "email";
  const general = err && !dup && Object.keys(fe).length === 0 ? err.userMessage : null;

  async function submit(e: React.FormEvent) {
    e.preventDefault();
    if (m.pending) return;
    try {
      const { data } = await m.mutate({ email: email.trim(), full_name: name.trim(), role });
      onSent(data);
    } catch {
      /* lỗi hiện tại ô / dòng thông báo; chữ đã gõ giữ nguyên */
    }
  }

  return (
    <Section title="Mời giảng viên" description="Người được mời nhận một liên kết đặt mật khẩu, dùng trong 72 giờ. Quản trị viên không biết và không đặt mật khẩu của ai.">
      <form onSubmit={submit} className={s.form} noValidate>
        <Field label="Email" error={dup ? "Email này đã có tài khoản." : fe.email}>
          {(id, d) => (
            <Input id={id} aria-describedby={d} invalid={dup || !!fe.email} type="email" inputMode="email" autoComplete="off" autoCapitalize="none" spellCheck={false} required value={email} onChange={(e) => setEmail(e.target.value)} />
          )}
        </Field>
        <Field label="Họ và tên" error={fe.full_name}>
          {(id, d) => <Input id={id} aria-describedby={d} invalid={!!fe.full_name} autoComplete="off" required value={name} onChange={(e) => setName(e.target.value)} />}
        </Field>
        <Field label="Vai" error={fe.role}>
          {(id, d) => (
            <Select id={id} aria-describedby={d} value={role} onChange={(e) => setRole(e.target.value as "TEACHER" | "TA")}>
              <option value="TEACHER">Giảng viên</option>
              <option value="TA">Trợ giảng</option>
            </Select>
          )}
        </Field>
        {general && (
          <InlineNotice tone="danger" compact action={<Button size="sm" onClick={() => void m.retry().then((r) => r && onSent(r.data), () => undefined)}>Gửi lại</Button>}>
            {general}
          </InlineNotice>
        )}
        <div className={s.formActions}>
          <Button type="submit" variant="primary" loading={m.pending} disabled={!email.trim() || !name.trim()}>
            Gửi lời mời
          </Button>
          <Button variant="ghost" onClick={onClose} disabled={m.pending}>
            Huỷ
          </Button>
        </div>
      </form>
    </Section>
  );
}
