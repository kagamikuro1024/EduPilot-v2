"use client";

import { useRef } from "react";
import { type ChatSession } from "@/mock/chat";
import { agoLabel } from "@/mock/derive";
import { ActionList, ActionRow, Button, Dialog, EmptyState } from "@/shared/ui";
import s from "./ChatScreen.module.css";

export const sessionMeta = (c: ChatSession, now: number) => `${agoLabel(c.at, now)} · ${c.msgs} tin`;

/**
 * Bảng phiên trước ở < 1100 px: trượt từ đáy (Dialog ở ≤ 719 px neo đáy), liệt kê phiên + `Phiên mới`;
 * đóng bằng vuốt xuống hoặc Esc (01-AC22). Dùng `Dialog` vì shared/ui chưa có biến thể neo đáy riêng.
 */
export function SessionSheet({
  open,
  onClose,
  sessions,
  now,
  openSession,
  onPick,
  onNew,
}: {
  open: boolean;
  onClose: () => void;
  sessions: ChatSession[];
  now: number;
  openSession: string | null;
  onPick: (id: string) => void;
  onNew: () => void;
}) {
  const startY = useRef<number | null>(null);
  return (
    <Dialog
      open={open}
      onClose={onClose}
      title="Phiên trước"
      description={`${sessions.length} phiên của bạn`}
      footer={
        <Button variant="primary" onClick={onNew}>
          Phiên mới
        </Button>
      }
    >
      <div
        className={s.sheet}
        onTouchStart={(e) => {
          startY.current = e.touches[0].clientY;
        }}
        onTouchMove={(e) => {
          if (startY.current !== null && e.touches[0].clientY - startY.current > 60) {
            startY.current = null;
            onClose();
          }
        }}
        onTouchEnd={() => {
          startY.current = null;
        }}
      >
        <p className={s.sheetHint} aria-hidden>
          Vuốt xuống để đóng
        </p>
        {sessions.length === 0 ? (
          <EmptyState title="Chưa có phiên nào">Phiên chat của bạn sẽ hiện ở đây sau lần hỏi đầu tiên.</EmptyState>
        ) : (
          <ActionList label="Phiên trước">
            {sessions.map((c) => (
              <ActionRow
                key={c.id}
                title={c.title}
                context={sessionMeta(c, now)}
                meta={openSession === c.id ? "Đang xem" : undefined}
                onSelect={() => onPick(c.id)}
              />
            ))}
          </ActionList>
        )}
      </div>
    </Dialog>
  );
}
