"use client";

import { useEffect, useState } from "react";
import { Button, ButtonLink, Field, Input, StatusText } from "@/shared/ui";
import type { CheckinWindow } from "./checkinApi";
import s from "./RealAttendance.module.css";

const mmss = (ms: number) => {
  const t = Math.max(0, Math.floor(ms / 1000));
  return `${String(Math.floor(t / 60)).padStart(2, "0")}:${String(t % 60).padStart(2, "0")}`;
};

/** Đếm ngược riêng một nhịp 1 s để lưới không dựng lại mỗi giây; hết giờ gọi `onExpire` đúng một lần. */
function Countdown({ closesAt, onExpire }: { closesAt: string; onExpire: () => void }) {
  const [now, setNow] = useState(() => Date.now());
  const end = Date.parse(closesAt);
  const left = end - now;
  useEffect(() => {
    if (left <= 0) {
      onExpire();
      return;
    }
    const t = setTimeout(() => setNow(Date.now()), 1000);
    return () => clearTimeout(t);
  }, [left, onExpire]);
  return <span data-part="window-left">còn {mmss(left)}</span>;
}

/**
 * Dòng cửa sổ điểm danh (DESIGN §14.8, bảng «Window state»): một dòng yên lặng ngay dưới thanh công cụ, không toast, không panel thứ hai.
 * Đang mở · Chưa đóng (đóng lúc offline, tự thử lại) · Đã đóng (số ghi từ lưới) · số đầu người.
 */
export function CheckinWindowLine({
  open,
  latest,
  checkedIn,
  active,
  summary,
  closeQueued,
  closing,
  displayHref,
  editingHeadcount,
  onHeadcount,
  onClose,
  onAbsent,
  absentInNotice,
  onExpire,
}: {
  open: CheckinWindow | null;
  latest: CheckinWindow | null;
  checkedIn: number;
  active: number;
  summary: { present: number; late: number; unchecked: number };
  closeQueued: boolean;
  closing: boolean;
  displayHref: string;
  editingHeadcount: boolean;
  onHeadcount: (v: number | null) => void;
  onClose: () => void;
  onAbsent: () => void;
  /** Cảnh báo CHECKIN_UNRESOLVED đang hiện và đã có nút «Đánh vắng…»: dòng này không lặp lại (một việc một nút, §23) */
  absentInNotice: boolean;
  onExpire: () => void;
}) {
  const [hc, setHc] = useState<string>(() => (latest?.headcount != null ? String(latest.headcount) : ""));
  useEffect(() => {
    if (!editingHeadcount) return;
    const v = hc.trim() === "" ? null : Math.min(2000, Math.max(0, Math.round(Number(hc) || 0)));
    if (v === (latest?.headcount ?? null)) return;
    const t = setTimeout(() => onHeadcount(v), 600);
    return () => clearTimeout(t);
  }, [hc, editingHeadcount, latest?.headcount, onHeadcount]);

  if (!latest) return null;
  const tail = editingHeadcount ? (
    <Field label="Số đầu người">
      {(id, d) => <Input id={id} aria-describedby={d} type="number" inputMode="numeric" min={0} max={2000} value={hc} onChange={(e) => setHc(e.target.value)} />}
    </Field>
  ) : latest.headcount != null ? (
    <>
      <span>· {latest.headcount} người trong lớp</span>
      {latest.over_headcount && <span data-part="over-headcount"><StatusText tone="amber">Số điểm danh nhiều hơn số người đếm</StatusText></span>}
    </>
  ) : null;

  if (open) {
    return (
      <div className={s.windowLine} data-part="window-line" data-state="open">
        <span>Đang mở</span>
        {open.round >= 2 && <span>· Vòng {open.round}</span>}
        <span data-part="window-count">· {checkedIn}/{active}</span>
        <span>· <Countdown closesAt={open.closes_at} onExpire={onExpire} /></span>
        {tail}
        {closeQueued ? (
          <span data-part="close-queued"><StatusText tone="amber">Chưa đóng</StatusText></span>
        ) : (
          <span className={s.lineButtons}>
            <ButtonLink href={displayHref} size="sm">Mở màn chiếu</ButtonLink>
            <Button size="sm" onClick={onClose} disabled={closing}>Đóng</Button>
          </span>
        )}
      </div>
    );
  }
  return (
    <div className={s.windowLine} data-part="window-line" data-state="closed">
      {closeQueued ? (
        <span data-part="close-queued"><StatusText tone="amber">Chưa đóng</StatusText></span>
      ) : (
        <span data-part="window-summary">
          Đã ghi {summary.present} có mặt · {summary.late} muộn · {summary.unchecked} chưa điểm danh
        </span>
      )}
      {tail}
      {summary.unchecked > 0 && !closeQueued && !absentInNotice && (
        <span className={s.lineButtons}>
          <Button size="sm" onClick={onAbsent}>Đánh vắng {summary.unchecked} SV chưa điểm danh</Button>
        </span>
      )}
    </div>
  );
}
