"use client";

import { useState, type KeyboardEvent } from "react";
import { Plus } from "lucide-react";
import { DataTable, Kbd, type Column } from "@/shared/ui";
import s from "./AttendanceGrid.module.css";

export type AttendanceStatus = "PRESENT" | "LATE" | "EXCUSED" | "ABSENT";
export const ATTENDANCE_OPTIONS: Array<{ value: AttendanceStatus; label: string; key: string }> = [
  { value: "PRESENT", label: "Có mặt", key: "1" },
  { value: "LATE", label: "Muộn", key: "2" },
  { value: "EXCUSED", label: "Vắng phép", key: "3" },
  { value: "ABSENT", label: "Vắng", key: "4" },
];
/** "+2 · +0,25×2" (số lần · điểm cộng mỗi lần × số lần); chưa biết điểm cộng thì chỉ "+2". */
const speakText = (n: number, bonus?: string) => (bonus ? `+${n} · +${bonus}×${n}` : `+${n}`);
const labelOf = (v: AttendanceStatus) => ATTENDANCE_OPTIONS.find((o) => o.value === v)?.label ?? v;

export type AttendanceGridRow = {
  id: string;
  name: string;
  code: string;
  status: AttendanceStatus;
  speaks: number;
  /** vừa lưu xong: nháy nền xanh nhạt tại ô tên (US-UX91-04 AC7) */
  saved?: boolean;
  /** có khi người khác vừa đổi dòng này (409): hiện lựa chọn giữ / lấy ngay trên dòng */
  conflict?: { by: string; theirs: AttendanceStatus } | null;
};

/**
 * Lưới điểm danh (DESIGN §14.8): máy tính = bảng radio, phím `1`–`4` đặt trạng thái cho hàng đang chọn rồi xuống hàng kế (`Shift+1…4` giữ hàng), `↑`/`↓` đổi hàng, `P` ghi phát biểu;
 * điện thoại = danh sách hàng chỉ có chữ trạng thái, bấm hàng mở bốn lựa chọn ≥ 44 px ngay dưới hàng (đổi một người ≤ 2 chạm).
 */
export function AttendanceGrid({
  rows,
  caption,
  readOnly,
  speakBonus,
  stickyHeight,
  onSet,
  onSpeak,
  onResolve,
}: {
  rows: AttendanceGridRow[];
  caption: string;
  readOnly?: boolean;
  /** điểm cộng mỗi lần phát biểu đọc từ công thức lớp, vd. "0,25"; chưa có công thức thì bỏ trống */
  speakBonus?: string;
  /** bảng máy tính cao tối đa, tiêu đề cột và cột tên dính (DESIGN §14.8) */
  stickyHeight?: string;
  onSet: (id: string, status: AttendanceStatus) => void;
  onSpeak: (id: string) => void;
  onResolve?: (id: string, choice: "mine" | "theirs") => void;
}) {
  const [cursor, setCursor] = useState(0);
  const [open, setOpen] = useState<string | null>(null);

  function onKeyDown(e: KeyboardEvent) {
    const row = rows[cursor];
    if (!row || readOnly) return;
    if (e.key === "ArrowDown" || e.key === "ArrowUp") {
      e.preventDefault();
      setCursor((i) => Math.min(rows.length - 1, Math.max(0, i + (e.key === "ArrowDown" ? 1 : -1))));
      return;
    }
    const digit = /^Digit([1-4])$/.exec(e.code)?.[1] ?? (/^[1-4]$/.test(e.key) ? e.key : undefined);
    const opt = ATTENDANCE_OPTIONS.find((o) => o.key === digit);
    if (opt && !e.ctrlKey && !e.metaKey && !e.altKey) {
      e.preventDefault();
      onSet(row.id, opt.value);
      if (!e.shiftKey) setCursor((i) => Math.min(rows.length - 1, i + 1));
      return;
    }
    if (e.key.toLowerCase() === "p" && !e.ctrlKey && !e.metaKey && !e.altKey) {
      e.preventDefault();
      onSpeak(row.id);
    }
  }

  const resolve = (r: AttendanceGridRow) =>
    r.conflict && onResolve ? (
      <span className={s.choose} role="group" aria-label={`Xung đột ${r.name}`}>
        <span>{r.conflict.by} đã đánh {labelOf(r.conflict.theirs)}</span>
        <button type="button" onClick={() => onResolve(r.id, "mine")}>Giữ của tôi</button>
        <button type="button" onClick={() => onResolve(r.id, "theirs")}>Lấy của {r.conflict.by}</button>
      </span>
    ) : null;

  const columns: Column<AttendanceGridRow>[] = [
    {
      key: "name",
      header: "Sinh viên",
      frozen: true,
      width: "30%",
      render: (r) => (
        <span className={[s.name, r.saved ? "ep-saved ep-motion-safe" : ""].join(" ")}>
          <span className="ep-item-title">{r.name}</span>
          <span className="ep-meta">
            {r.code}
            {r.status !== "PRESENT" && <span className={s.statusText} data-status={r.status}> · {labelOf(r.status)}</span>}
          </span>
          {resolve(r)}
        </span>
      ),
    },
    ...ATTENDANCE_OPTIONS.map<Column<AttendanceGridRow>>((o) => ({
      key: o.value,
      header: (
        <>
          {o.label} <Kbd>{o.key}</Kbd>
        </>
      ),
      align: "center",
      width: "12%",
      render: (r) => (
        <label className={s.cell}>
          <span className="ep-sr-only">{`${o.label} — ${r.name}`}</span>
          <input
            type="radio"
            name={`mark-${r.id}`}
            className={s.radio}
            data-status={o.value}
            disabled={readOnly}
            checked={r.status === o.value}
            onChange={() => onSet(r.id, o.value)}
            onFocus={() => setCursor(rows.findIndex((x) => x.id === r.id))}
          />
        </label>
      ),
    })),
    {
      key: "speak",
      header: (
        <>
          Phát biểu <Kbd>P</Kbd>
        </>
      ),
      align: "end",
      render: (r) => (
        <span className={s.speakCell}>
          {r.speaks > 0 && <span className={s.count} data-part="speak-count">{speakText(r.speaks, speakBonus)}</span>}
          <button type="button" className={s.speak} disabled={readOnly} onClick={() => onSpeak(r.id)} aria-label={`Ghi phát biểu · ${r.name}`} title="Ghi phát biểu (P)">
            <Plus aria-hidden />
          </button>
        </span>
      ),
    },
  ];

  return (
    <div className={s.grid} tabIndex={0} onKeyDown={onKeyDown} aria-label="Bảng điểm danh — phím mũi tên chọn hàng, phím 1 đến 4 đánh dấu, phím P ghi phát biểu" data-part="attendance-grid">
      <DataTable
        caption={caption}
        columns={columns}
        rows={rows}
        stickyHeight={stickyHeight}
        rowKey={(r) => r.id}
        activeKey={rows[cursor]?.id}
        rowAttrs={(r) => ({ "data-part": "student-row", "data-student-id": r.id })}
        mobileRow={(r) => {
          const expanded = open === r.id;
          const tone = r.status === "ABSENT" ? s.mStatusAbsent : r.status === "PRESENT" ? "" : s.mStatusOther;
          return (
            <div className={s.mRow} data-part="att-row" data-student-id={r.id}>
              <div className={s.mTop}>
                <button type="button" className={s.mHead} aria-expanded={expanded} onClick={() => setOpen(expanded ? null : r.id)} disabled={readOnly}>
                  <span className={[s.name, r.saved ? "ep-saved ep-motion-safe" : ""].join(" ")}>
                    <span className="ep-item-title">{r.name}</span>
                    <span className="ep-meta">{r.code}{r.speaks > 0 ? ` · ${speakText(r.speaks, speakBonus)}` : ""}</span>
                  </span>
                  <span className={`${s.mStatus} ${tone}`} data-part="att-status">{labelOf(r.status)}</span>
                </button>
                <button type="button" className={s.mPlus} disabled={readOnly} onClick={() => onSpeak(r.id)} aria-label={`Ghi phát biểu · ${r.name}`}>
                  <Plus aria-hidden />
                </button>
              </div>
              {resolve(r)}
              {expanded && !readOnly && (
                <div className={s.mPanel}>
                  <div className={s.mMarks} role="radiogroup" aria-label={`Điểm danh ${r.name}`}>
                    {ATTENDANCE_OPTIONS.map((o) => (
                      <button
                        key={o.value}
                        type="button"
                        role="radio"
                        aria-checked={r.status === o.value}
                        className={s.mMark}
                        onClick={() => {
                          onSet(r.id, o.value);
                          setOpen(null);
                        }}
                      >
                        {o.label}
                      </button>
                    ))}
                  </div>
                </div>
              )}
            </div>
          );
        }}
      />
    </div>
  );
}
