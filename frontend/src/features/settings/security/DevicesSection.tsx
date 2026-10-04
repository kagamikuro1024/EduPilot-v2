"use client";

import { useQueryClient } from "@tanstack/react-query";
import { useState } from "react";
import { ApiError } from "@/shared/data";
import { ApiErrorNotice } from "@/shared/data/ApiErrorNotice";
import { Button, ConfirmIrreversible, OverflowMenu, PageState, Section, UndoLine } from "@/shared/ui";
import { SESSIONS_KEY, useRevokeOthers, useRevokeSession, useSessions, type DeviceSession } from "./api";
import s from "./security.module.css";

const TIME = new Intl.DateTimeFormat("vi-VN", { hour: "2-digit", minute: "2-digit", hour12: false, timeZone: "Asia/Ho_Chi_Minh" });
const DAY = new Intl.DateTimeFormat("vi-VN", { day: "2-digit", month: "2-digit", timeZone: "Asia/Ho_Chi_Minh" });

/** "Lần cuối 09:20" (hôm nay) hoặc "Lần cuối 09:20 · 03/10". */
export function lastSeen(iso: string, now = new Date()): string {
  const t = new Date(iso);
  const same = DAY.format(t) === DAY.format(now);
  return `Lần cuối ${TIME.format(t)}${same ? "" : ` · ${DAY.format(t)}`}`;
}

export function DevicesSection() {
  const qc = useQueryClient();
  const q = useSessions();
  const revoke = useRevokeSession();
  const others = useRevokeOthers();
  const [gone, setGone] = useState<string[]>([]); // hàng vừa đăng xuất: hiện dòng tĩnh thay cho hàng
  const [confirm, setConfirm] = useState(false);
  const [failed, setFailed] = useState<unknown>(null);

  const items = q.data ?? [];
  const otherList = items.filter((d) => !d.current && !gone.includes(d.id));

  function signOut(d: DeviceSession) {
    setFailed(null);
    setGone((g) => [...g, d.id]);
    revoke.mutate(d.id, {
      onError: (err) => {
        setGone((g) => g.filter((x) => x !== d.id));
        setFailed(err);
      },
    });
  }

  return (
    <Section title="Thiết bị đang đăng nhập">
      <PageState query={q}>
        {failed != null && <ApiErrorNotice error={failed} />}
        <ul className={s.list}>
          {items.map((d) =>
            gone.includes(d.id) ? (
              <li key={d.id} className={s.row}>
                <UndoLine message="Đã đăng xuất thiết bị này" onDone={() => {
                    setGone((g) => g.filter((x) => x !== d.id));
                    void qc.invalidateQueries({ queryKey: SESSIONS_KEY });
                  }}
                />
              </li>
            ) : (
              <li key={d.id} className={s.row}>
                <div className={s.who}>
                  <span className={s.name}>
                    {d.device_label || "Thiết bị không rõ"}
                    {d.current && <span className={s.here}> · Thiết bị này</span>}
                  </span>
                  <span className={s.meta}>
                    {d.ip_masked || "IP không rõ"} · {lastSeen(d.last_used_at)}
                  </span>
                </div>
                {!d.current && <OverflowMenu label={`Thao tác cho ${d.device_label || "thiết bị"}`} items={[{ label: "Đăng xuất thiết bị này", onSelect: () => signOut(d) }]} />}
              </li>
            ),
          )}
        </ul>
        {otherList.length === 0 ? (
          <p className={s.empty}>Không có thiết bị nào khác.</p>
        ) : (
          <Button variant="text" onClick={() => setConfirm(true)}>
            Đăng xuất mọi thiết bị khác
          </Button>
        )}
        <ConfirmIrreversible
          open={confirm}
          onClose={() => !others.isPending && setConfirm(false)}
          onConfirm={() => others.mutate(undefined, { onSuccess: () => setConfirm(false) })}
          title="Đăng xuất mọi thiết bị khác?"
          consequence={`Đăng xuất ${otherList.length} thiết bị khác. Họ sẽ phải đăng nhập lại.`}
          confirmLabel="Đăng xuất"
          loading={others.isPending}
          error={others.isError ? (others.error instanceof ApiError ? others.error.userMessage : "Chưa đăng xuất được. Hãy thử lại.") : undefined}
        />
      </PageState>
    </Section>
  );
}
