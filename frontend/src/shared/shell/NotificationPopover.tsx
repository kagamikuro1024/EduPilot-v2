"use client";

import { Bell } from "lucide-react";
import Link from "next/link";
import { useEffect, useRef, useState } from "react";
import { Popover } from "@/shared/ui";
import s from "./AppShell.module.css";

export type NotificationItem = { id: string; title: string; context: string; when: string; href: string; read: boolean };

/**
 * Chuông thông báo — KHUNG nhận danh sách qua props (dữ liệu thật ở P4). Chấm đỏ (`bell-dot`) chỉ khi `unread > 0`; mở khung KHÔNG
 * xoá chấm (chỉ bấm vào từng thông báo mới `onRead`). Khi `unread` tăng, vùng `aria-live=polite` đọc "Có N thông báo mới" một lần.
 */
export function NotificationPopover({ items, unread, onRead, failed }: { items: NotificationItem[]; unread: number; onRead?: (id: string) => void; failed?: boolean }) {
  const prev = useRef(unread);
  const [announce, setAnnounce] = useState("");
  useEffect(() => {
    // Có thông báo MỚI = số chưa đọc tăng so với lần trước; đọc đúng một lần, không đọc lại khi hiển thị lại.
    if (unread > prev.current) setAnnounce(`Có ${unread - prev.current} thông báo mới`);
    else if (unread < prev.current) setAnnounce("");
    prev.current = unread;
  }, [unread]);

  return (
    <>
      <span className="ep-sr-only" aria-live="polite" data-part="bell-live">
        {announce}
      </span>
      <Popover
        width={340}
        label="Thông báo"
        trigger={(p) => (
          <button type="button" className={s.iconBtn} onClick={p.toggle} aria-expanded={p["aria-expanded"]} aria-haspopup="true" aria-label={unread ? `Thông báo, ${unread} chưa đọc` : "Thông báo"}>
            <Bell aria-hidden />
            {unread > 0 && <span className={s.unreadDot} data-part="bell-dot" aria-hidden />}
          </button>
        )}
      >
        {(close) => (
          <div className={s.notes} data-part="notifications">
            <p className={s.panelLabel}>Thông báo</p>
            {failed ? (
              <p className={s.noteEmpty}>Chưa tải được thông báo.</p>
            ) : items.length === 0 ? (
              <p className={s.noteEmpty}>Chưa có thông báo. Khi có việc cần bạn, nó sẽ hiện ở đây.</p>
            ) : (
              <ul>
                {items.map((n) => (
                  <li key={n.id}>
                    <Link
                      href={n.href}
                      className={s.note}
                      onClick={() => {
                        onRead?.(n.id);
                        close();
                      }}
                    >
                      <span className={[s.noteDot, n.read ? "" : s.noteUnread].join(" ")} aria-hidden />
                      <span>
                        <span className={s.noteTitle}>{n.title}</span>
                        <span className={s.noteMeta}>
                          {n.context} · {n.when}
                        </span>
                      </span>
                    </Link>
                  </li>
                ))}
              </ul>
            )}
          </div>
        )}
      </Popover>
    </>
  );
}
