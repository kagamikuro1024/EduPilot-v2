"use client";

import { useRef, useState } from "react";
import { ApiError, apiClient, newIdempotencyKey, useAutosaveDraft, useUndoableAction } from "@/shared/data";
import { Button, InlineNotice, Textarea } from "@/shared/ui";
import { ticketLine } from "./TicketAction";
import type { StudentTicket } from "./chatApi";
import s from "./ChatScreen.module.css";

const MAX = 2000;

/**
 * Dưới tin của giảng viên: `Hỏi tiếp` / `Đã rõ` (ANSWERED), "Đã đóng" + `Hỏi tiếp` (CLOSED), dòng trạng thái khi đã mở lại (OPEN / CLAIMED).
 * `Đã rõ` đảo ngược được: lạc quan, "Đã đóng · Hoàn tác" 5 giây, không hộp thoại. Chữ `Hỏi tiếp` tự lưu nháp; gửi lại dùng cùng mã tin.
 */
export function StaffReplyActions({ ticket, userId, onChanged, locked }: { ticket: StudentTicket; userId?: string; onChanged: () => void; locked: boolean }) {
  const [closed, setClosed] = useState(false);
  const [asking, setAsking] = useState(false);
  const [busy, setBusy] = useState(false);
  const [err, setErr] = useState<ApiError | null>(null);
  const clientId = useRef<string | null>(null);
  const draft = useAutosaveDraft(`followup:${ticket.id}`, { userId });

  const confirm = useUndoableAction<null>({
    apply: () => setClosed(true),
    rollback: () => setClosed(false),
    commit: async () => {
      await apiClient.post(`/chat/tickets/${ticket.id}/confirm`);
      onChanged();
    },
    compensate: async () => {
      await apiClient.post(`/chat/tickets/${ticket.id}/confirm/undo`);
      onChanged();
    },
    label: () => "Đã đóng",
  });

  const status = closed ? "CLOSED" : ticket.status;

  async function send() {
    const content = draft.value.trim();
    if (!content || busy) return;
    clientId.current ??= newIdempotencyKey();
    setBusy(true);
    setErr(null);
    try {
      await apiClient.post(`/chat/tickets/${ticket.id}/followup`, { content, client_msg_id: clientId.current });
      clientId.current = null;
      draft.clear();
      setAsking(false);
      setClosed(false);
      onChanged();
    } catch (e) {
      setErr(e instanceof ApiError ? e : new ApiError({ status: 0, code: "NETWORK" }));
    } finally {
      setBusy(false);
    }
  }

  if (status === "OPEN" || status === "CLAIMED") {
    return (
      <span className={s.who} data-part="ticket-status" role="status">
        {ticketLine({ ...ticket, status })}
      </span>
    );
  }
  const expired = err?.code === "TICKET_REOPEN_EXPIRED";
  return (
    <div className={s.staffActions} data-part="staff-actions">
      {!asking && (
        <>
          {status === "CLOSED" && <span className={s.who}>Đã đóng</span>}
          {!expired && <Button size="sm" disabled={locked} onClick={() => setAsking(true)}>Hỏi tiếp</Button>}
          {status === "ANSWERED" && <Button size="sm" variant="secondary" disabled={locked} onClick={() => void confirm.run(null)}>Đã rõ</Button>}
          {confirm.node}
        </>
      )}
      {asking && (
        <div className={s.followup}>
          <Textarea
            aria-label="Hỏi tiếp"
            rows={3}
            maxLength={MAX}
            value={draft.value}
            onChange={(e) => draft.setValue(e.target.value)}
            onKeyDown={(e) => {
              if (e.key === "Enter" && (e.metaKey || e.ctrlKey)) {
                e.preventDefault();
                void send();
              }
            }}
          />
          <div className={s.followupRow}>
            <Button size="sm" variant="primary" disabled={busy || !draft.value.trim()} onClick={() => void send()}>Gửi</Button>
            <Button size="sm" variant="text" disabled={busy} onClick={() => setAsking(false)}>Huỷ</Button>
          </div>
        </div>
      )}
      {err && (
        <InlineNotice tone="danger" compact action={expired ? undefined : <Button size="sm" onClick={() => void send()}>Thử lại</Button>}>
          {err.userMessage}
        </InlineNotice>
      )}
    </div>
  );
}
