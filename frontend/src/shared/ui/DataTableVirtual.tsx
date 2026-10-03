"use client";

import { useVirtualizer } from "@tanstack/react-virtual";
import { useEffect, useRef, useState, type CSSProperties, type KeyboardEvent, type ReactNode } from "react";
import type { Column, Pagination } from "./DataTable";
import { InlineNotice } from "./Feedback";
import s from "./DataTable.module.css";

const ROW_H = 48; // 44–52 px (US-PU-02 AC5)
const LOAD_AHEAD = 6; // còn ≤ 6 dòng tới đáy thì xin trang kế

/**
 * Bảng ảo hoá cho danh sách dài (1.000+ dòng): chỉ dựng dòng nhìn thấy (+ đệm), tiêu đề dính, bàn phím ↑ ↓ Home End,
 * Enter kích hoạt, Esc thoát; phân trang con trỏ qua `pagination` (mỗi con trỏ xin đúng một lần).
 * Nạp lười từ DataTable (react-virtual không vào gói của route không dùng bảng dài).
 */
export default function VirtualTable<T>({
  columns,
  rows,
  rowKey,
  onRowClick,
  activeKey,
  caption,
  height,
  pagination,
}: {
  columns: Column<T>[];
  rows: T[];
  rowKey: (row: T) => string;
  onRowClick?: (row: T) => void;
  activeKey?: string;
  caption: string;
  height: number;
  pagination?: Pagination;
}) {
  const parentRef = useRef<HTMLDivElement>(null);
  const [cursorIdx, setCursorIdx] = useState(-1);
  const requested = useRef<string | null>(null);
  const footer = Boolean(pagination && (pagination.nextCursor !== null || pagination.error));
  const count = rows.length;
  // eslint-disable-next-line react-hooks/incompatible-library -- useVirtualizer trả hàm không memo được; bảng này không dùng React Compiler
  const v = useVirtualizer({ count, getScrollElement: () => parentRef.current, estimateSize: () => ROW_H, overscan: 8 });
  const items = v.getVirtualItems();
  const padTop = items.length ? items[0].start : 0;
  const padBottom = items.length ? v.getTotalSize() - items[items.length - 1].end : 0;
  const last = items.length ? items[items.length - 1].index : -1;

  // Xin trang kế đúng một lần cho mỗi con trỏ (cuộn nhanh không gọi trùng).
  const next = pagination?.nextCursor ?? null;
  useEffect(() => {
    if (!pagination || next === null || pagination.loading || pagination.error) return;
    if (last >= count - LOAD_AHEAD && requested.current !== next) {
      requested.current = next;
      pagination.onLoadMore(next);
    }
  }, [last, count, next, pagination]);

  const activeIdx = activeKey !== undefined ? rows.findIndex((r) => rowKey(r) === activeKey) : cursorIdx;

  function onKeyDown(e: KeyboardEvent<HTMLDivElement>) {
    if (e.target !== e.currentTarget) return;
    const move = (i: number) => {
      const n = Math.max(0, Math.min(count - 1, i));
      setCursorIdx(n);
      v.scrollToIndex(n, { align: "auto" });
      e.preventDefault();
    };
    if (e.key === "ArrowDown") move(activeIdx + 1);
    else if (e.key === "ArrowUp") move(activeIdx < 0 ? 0 : activeIdx - 1);
    else if (e.key === "Home") move(0);
    else if (e.key === "End") move(count - 1);
    else if (e.key === "Enter" && activeIdx >= 0 && onRowClick) {
      onRowClick(rows[activeIdx]);
      e.preventDefault();
    } else if (e.key === "Escape") {
      setCursorIdx(-1);
      e.currentTarget.blur();
    }
  }

  const span = columns.length;
  const spacer = (h: number, key: string) =>
    h > 0 ? (
      <tr key={key} aria-hidden style={{ height: h }}>
        <td colSpan={span} className={s.spacer} />
      </tr>
    ) : null;

  return (
    <div
      ref={parentRef}
      className={s.vscroll}
      style={{ "--vh": `${height}px` } as CSSProperties}
      tabIndex={0}
      role="region"
      aria-label={caption}
      aria-activedescendant={activeIdx >= 0 && rows[activeIdx] ? `vt-${rowKey(rows[activeIdx])}` : undefined}
      onKeyDown={onKeyDown}
      data-part="virtual-scroll"
    >
      <table className={[s.table, s.virtual].join(" ")}>
        <caption className="ep-sr-only">{caption}</caption>
        <thead>
          <tr>
            {columns.map((c) => (
              <th key={c.key} scope="col" style={{ "--w": c.width, textAlign: c.align === "end" ? "right" : c.align === "center" ? "center" : "left" } as CSSProperties} className={s.th}>
                {c.header}
              </th>
            ))}
          </tr>
        </thead>
        <tbody>
          {spacer(padTop, "pt")}
          {items.map((it) => {
            const row = rows[it.index];
            const key = rowKey(row);
            const on = it.index === activeIdx;
            return (
              <tr
                key={key}
                id={`vt-${key}`}
                data-index={it.index}
                className={[onRowClick ? s.clickable : "", on ? s.active : ""].join(" ")}
                aria-current={on ? "true" : undefined}
                onClick={onRowClick ? () => { setCursorIdx(it.index); onRowClick(row); } : undefined}
              >
                {columns.map((c) => (
                  <td key={c.key} style={{ textAlign: c.align === "end" ? "right" : c.align === "center" ? "center" : "left" }}>
                    {c.render(row)}
                  </td>
                ))}
              </tr>
            );
          })}
          {spacer(padBottom, "pb")}
          {footer && (
            <tr data-part="load-more" aria-busy={pagination?.loading || undefined}>
              <td colSpan={span} className={s.footCell}>
                {pagination?.error ? (
                  <InlineNotice
                    tone="danger"
                    compact
                    action={
                      <button type="button" className={s.retry} onClick={() => next !== null && pagination.onLoadMore(next)}>
                        Thử lại
                      </button>
                    }
                  >
                    {pagination.error as ReactNode}
                  </InlineNotice>
                ) : (
                  <span className={pagination?.loading ? s.skBar : s.skIdle} aria-label={pagination?.loading ? "Đang tải thêm" : undefined} />
                )}
              </td>
            </tr>
          )}
        </tbody>
      </table>
    </div>
  );
}
