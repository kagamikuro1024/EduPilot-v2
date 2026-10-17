"use client";

import { useEffect, useRef, useState } from "react";
import { ApiError, apiClient } from "@/shared/data";
import { timeAgo } from "@/shared/lib/timeAgo";
import { Button, InlineNotice } from "@/shared/ui";
import type { ChatMessage, StudentTicket } from "./chatApi";
import s from "./ChatScreen.module.css";

/** Trần ticket mở của một sinh viên ở một lớp (máy chủ giữ trần thật: `TICKET_OPEN_CAP`; đây chỉ để khoá nút sớm). */
const OPEN_CAP = 5;

function waited(iso: string, now = new Date()): string {
  const min = Math.max(0, Math.floor((now.getTime() - new Date(iso).getTime()) / 60_000));
  if (min < 60) return `${min} phút`;
  if (min < 48 * 60) return `${Math.floor(min / 60)} giờ`;
  return `${Math.floor(min / (24 * 60))} ngày`;
}

/** Giờ thật, cập nhật mỗi phút: "đã chờ N phút" tự đổi khi để trang mở (chữ cùng độ dài nên không nhảy layout). */
function useMinute(): Date {
  const [now, setNow] = useState(() => new Date());
  useEffect(() => {
    const t = window.setInterval(() => setNow(new Date()), 60_000);
    return () => window.clearInterval(t);
  }, []);
  return now;
}

/** Ba trạng thái của sinh viên (SRS FEAT-escalation-notify 7.2): chỉ lời, không số, không thuật ngữ kỹ thuật. */
export function ticketLine(t: StudentTicket, now = new Date()): string {
  switch (t.status) {
    case "OPEN": {
      const age = now.getTime() - new Date(t.created_at).getTime();
      return age < 60_000 ? "Đang chờ giảng viên · vừa gửi" : `Đang chờ giảng viên · đã gửi ${timeAgo(t.created_at, now)}`;
    }
    case "CLAIMED":
      return `Giảng viên đã nhận · đã chờ ${waited(t.created_at, now)}`;
    case "ANSWERED":
      return "Giảng viên đã trả lời";
    default:
      return "Đã đóng";
  }
}

/**
 * Nút `Nhờ giảng viên hỗ trợ` dưới câu trả lời của AI, hoặc dòng trạng thái khi đã có ticket (tự mở hoặc đã bấm).
 * Lạc quan: bấm là hiện dòng trạng thái ngay; bấm đúp chỉ phát một request; mất mạng giữ "Đang gửi…" và gửi lại khi có mạng (máy chủ khử trùng theo tin).
 */
export function TicketAction({ m, openTickets, onChanged, locked }: { m: ChatMessage; openTickets: number; onChanged: () => void; locked: boolean }) {
  const [local, setLocal] = useState<StudentTicket | null>(null);
  const [phase, setPhase] = useState<"idle" | "sending" | "offline" | "error">("idle");
  const flight = useRef(false);
  const now = useMinute();

  const ticket = m.ticket ?? local;

  async function send() {
    if (flight.current) return;
    flight.current = true;
    setLocal((cur) => cur ?? { id: "", status: "OPEN", created_at: new Date().toISOString() });
    if (typeof navigator !== "undefined" && !navigator.onLine) {
      setPhase("offline"); // không có mạng: không gửi, hiện "Đang gửi…" và gửi khi có mạng
      flight.current = false;
      return;
    }
    setPhase("sending");
    try {
      await apiClient.post(`/chat/messages/${m.id}/escalate`);
      setPhase("idle");
      onChanged();
    } catch (e) {
      const err = e instanceof ApiError ? e : new ApiError({ status: 0, code: "NETWORK" });
      if (err.code === "NETWORK") {
        setPhase("offline");
      } else {
        setLocal(null);
        setPhase("error");
        if (err.code === "TICKET_LIMIT_REACHED") onChanged();
      }
    } finally {
      flight.current = false;
    }
  }

  // mất mạng khi bấm: tự gửi lại khi có mạng (cùng message_id ⇒ máy chủ chỉ tạo một ticket)
  useEffect(() => {
    if (phase !== "offline") return;
    const retry = () => void send();
    window.addEventListener("online", retry);
    return () => window.removeEventListener("online", retry);
  }, [phase]); // eslint-disable-line react-hooks/exhaustive-deps

  if (ticket) {
    return (
      <span className={s.who} data-part="ticket-status" role="status">
        {phase === "offline" ? "Đang gửi…" : ticketLine(ticket, now)}
      </span>
    );
  }
  if (!m.can_escalate) return null;
  const full = openTickets >= OPEN_CAP;
  return (
    <>
      <Button
        size="sm"
        data-part="ask-staff"
        disabled={full || locked}
        title={full ? `Bạn đang có ${OPEN_CAP} câu hỏi chờ giảng viên` : "Giảng viên sẽ đọc câu hỏi này và câu trả lời của AI"}
        onClick={() => void send()}
      >
        Nhờ giảng viên hỗ trợ
      </Button>
      {phase === "error" && (
        <InlineNotice tone="danger" compact action={<Button size="sm" onClick={() => void send()}>Thử lại</Button>}>
          Chưa gửi được. Thử lại
        </InlineNotice>
      )}
    </>
  );
}
