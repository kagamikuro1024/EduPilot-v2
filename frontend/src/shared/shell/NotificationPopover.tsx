"use client";

import { Bell } from "lucide-react";
import Link from "next/link";
import { useEffect, useLayoutEffect, useRef, useState } from "react";
import { Popover, StatusText } from "@/shared/ui";
import s from "./AppShell.module.css";

export type NotificationItem = { id: string; title: string; context: string; when: string; href: string; read: boolean };

/**
 * Chuông thông báo — KHUNG nhận danh sách qua props (dữ liệu thật ở P4). Chấm đỏ (`bell-dot`) chỉ khi `unread > 0`; mở khung KHÔNG
 * xoá chấm (chỉ bấm vào từng thông báo mới `onRead`). Khi `unread` tăng, vùng `aria-live=polite` đọc "Có N thông báo mới" một lần.
 */
export function NotificationPopover({ items, unread, onRead, failed, stale }: { items: NotificationItem[]; unread: number; onRead?: (id: string) => void; failed?: boolean; stale?: boolean }) {
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
            {unread > 0 && <span className={s.unreadDot} data-part="bell-dot" aria-hidden>{unread > 9 ? "9+" : unread}</span>}
          </button>
        )}
      >
        {(close) => (
          <div className={s.notes} data-part="notifications">
            <p className={s.panelLabel}>Thông báo</p>
            {stale && <StatusText>Chưa cập nhật được</StatusText>}
            {failed ? (
              <p className={s.noteEmpty}>Chưa tải được thông báo.</p>
            ) : items.length === 0 ? (
              <p className={s.noteEmpty}>Chưa có thông báo. Khi có việc cần bạn, nó sẽ hiện ở đây.</p>
            ) : (
              <NoteList items={items} onRead={onRead} close={close} />
            )}
          </div>
        )}
      </Popover>
    </>
  );
}

/**
 * Danh sách có thể cuộn. Thông báo mới chèn lên ĐẦU lúc khung đang mở không được đẩy mục đang đọc đi chỗ khác:
 * khung chốt chiều cao lúc mở; khi mục đầu đổi, bù đúng độ lệch của mục đầu cũ vào `scrollTop` (mục mới nằm trên mép nhìn, mục đang đọc đứng yên).
 */
function NoteList({ items, onRead, close }: { items: NotificationItem[]; onRead?: (id: string) => void; close: () => void }) {
  const ref = useRef<HTMLUListElement>(null);
  const last = useRef<{ first: string | undefined; top: number }>({ first: items[0]?.id, top: 0 });
  useLayoutEffect(() => {
    const el = ref.current;
    if (!el) return;
    if (!el.style.height) el.style.height = `${el.offsetHeight}px`; // chiều cao của khung chốt lúc mở: mục mới làm nội dung dài ra (cuộn được) chứ không đẩy khung / mục đang đọc
    const first = items[0]?.id;
    const anchor = (id: string | undefined) => el.querySelector<HTMLElement>(`[data-note-id="${id}"]`);
    if (last.current.first && first !== last.current.first) {
      const old = anchor(last.current.first); // mục đầu CŨ: giữ nguyên vị trí trên màn hình
      if (old) el.scrollTop += old.offsetTop - last.current.top; // offsetTop: không bị animation mở khung làm lệch phép đo
    }
    last.current = { first, top: anchor(first)?.offsetTop ?? 0 };
  }, [items]);
  return (
    <ul ref={ref} className={s.noteList} style={{ overflowAnchor: "none", position: "relative" }}>
      {items.map((n) => (
        <li key={n.id} data-note-id={n.id}>
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
  );
}
