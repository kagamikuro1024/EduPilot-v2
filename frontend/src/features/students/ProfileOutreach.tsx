"use client";

import { useState } from "react";
import { ApiError, apiClient, useIdempotentMutation } from "@/shared/data";
import { ApiErrorNotice } from "@/shared/data/ApiErrorNotice";
import { Button, InlineNotice, Panel, Textarea, UndoLine } from "@/shared/ui";
import s from "./StudentProfile.module.css";

/**
 * `Nhắn sinh viên` (US-P5-05 AC7): sửa tại chỗ, không hộp thoại. Gửi một lần = một tin vào chat riêng của sinh viên; gửi lại sau lỗi dùng LẠI khoá Idempotency (không thêm tin).
 * Thân tin do giảng viên gõ; hệ thống không thêm chữ nào về rủi ro.
 */
export function ProfileOutreach({ courseId, studentId, name, onClose }: { courseId: string; studentId: string; name: string; onClose: () => void }) {
  const [text, setText] = useState("");
  const [err, setErr] = useState<unknown>(null);
  const [sent, setSent] = useState(false);
  const send = useIdempotentMutation((body: string, key: string) => apiClient.post(`/courses/${courseId}/students/${studentId}/outreach`, { body }, { idempotencyKey: key }));

  async function go(retry: boolean) {
    setErr(null);
    try {
      await (retry ? send.retry() : send.mutate(text.trim()));
      setText("");
      setSent(true);
    } catch (e) {
      setErr(e); // chữ đang gõ ở lại trong ô
    }
  }

  if (sent) return <UndoLine message={`Đã gửi tin tới ${name}`} onDone={onClose} />;
  const e = err instanceof ApiError ? err : null;
  return (
    <Panel>
      <div className={s.compose} data-part="outreach">
        <Textarea rows={3} autoFocus value={text} maxLength={2000} aria-label={`Tin nhắn tới ${name}`} placeholder="Tin nhắn riêng — sinh viên đọc ở Chat của mình." onChange={(ev) => setText(ev.target.value)} />
        {e?.code === "PII_IN_ANSWER" ? (
          <InlineNotice tone="warning" compact>Tin có thông tin của sinh viên khác. Bỏ họ tên, MSSV hoặc email đó rồi gửi lại.</InlineNotice>
        ) : e ? (
          <ApiErrorNotice error={e} onRetry={e.status === 0 || e.status >= 500 ? () => void go(true) : undefined} title="Chưa gửi được tin." />
        ) : null}
        <div className={s.composeBar}>
          <Button variant="primary" loading={send.pending} disabled={text.trim() === ""} onClick={() => void go(false)}>Gửi</Button>
          <Button variant="text" onClick={onClose}>Để sau</Button>
        </div>
      </div>
    </Panel>
  );
}
