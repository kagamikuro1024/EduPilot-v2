"use client";

import { useVirtualizer } from "@tanstack/react-virtual";
import { useEffect, useLayoutEffect, useRef, useState, type CSSProperties, type KeyboardEvent, type ReactNode } from "react";
import type { CellNav, Column, Pagination } from "./DataTable";
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
  cellNav,
  rowAttrs,
  pagination,
}: {
  columns: Column<T>[];
  rows: T[];
  rowKey: (row: T) => string;
  onRowClick?: (row: T) => void;
  activeKey?: string;
  caption: string;
  height: number;
  cellNav?: CellNav<T>;
  rowAttrs?: (row: T) => Record<string, string>;
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

  // Di chuyển ô nhập (Enter / Tab): dòng đích có thể đang ngoài cửa sổ ảo → cuộn tới rồi lấy focus sau khi dựng.
  const pendingFocus = useRef<{ idx: number; col: string } | null>(null);
  const skipBlur = useRef(false); // Esc hoàn tác: rời ô sau đó không được lưu lại nháp vừa bỏ
  const [, bump] = useState(0);
  const [focused, setFocused] = useState<string | null>(null); // ô đang focus hiện giá trị gốc, ô khác hiện `display`
  useLayoutEffect(() => {
    if (focused) parentRef.current?.querySelector<HTMLInputElement>(`input[data-cell="${CSS.escape(focused)}"]`)?.select();
  }, [focused]);
  useEffect(() => {
    const p = pendingFocus.current;
    if (!p || !cellNav) return;
    const row = rows[p.idx];
    if (!row) { pendingFocus.current = null; return; }
    const el = parentRef.current?.querySelector<HTMLInputElement>(`input[data-cell="${CSS.escape(`${rowKey(row)}:${p.col}`)}"]`);
    if (el) { pendingFocus.current = null; el.focus(); el.select(); }
  });
  const editableCols = columns.filter((c) => c.cell);
  function focusCell(idx: number, col: string) {
    pendingFocus.current = { idx, col };
    v.scrollToIndex(idx, { align: "auto" });
    bump((n) => n + 1);
  }
  const wanted = cellNav?.focus;
  const doneFocus = useRef(0);
  useEffect(() => {
    if (!wanted || doneFocus.current === wanted.n) return;
    const idx = rows.findIndex((r) => rowKey(r) === wanted.rowKey);
    if (idx < 0) return; // dòng chưa nạp: chạy lại khi `rows` đổi
    doneFocus.current = wanted.n;
    pendingFocus.current = { idx, col: wanted.col };
    v.scrollToIndex(idx, { align: "center" });
    bump((n) => n + 1);
  }, [wanted, rows, rowKey, v]);
  function cellKey(e: KeyboardEvent<HTMLInputElement>, row: T, rowIdx: number, col: Column<T>) {
    if (!cellNav) return;
    const text = e.currentTarget.value;
    const next = (dir: 1 | -1) => {
      const at = editableCols.findIndex((c) => c.key === col.key);
      for (let i = at + dir; i >= 0 && i < editableCols.length; i += dir) {
        const c = editableCols[i];
        if (c.cell?.editable(row)) return c.key;
      }
      return null;
    };
    if (e.key === "Enter") {
      e.preventDefault();
      cellNav.onCommit(row, col.key, text, "down");
      if (rowIdx + 1 < count) focusCell(rowIdx + 1, col.key);
    } else if (e.key === "Tab" && !e.shiftKey) {
      const to = next(1);
      cellNav.onCommit(row, col.key, text, "right");
      if (to) { e.preventDefault(); focusCell(rowIdx, to); }
    } else if (e.key === "Tab" && e.shiftKey) {
      const to = next(-1);
      cellNav.onCommit(row, col.key, text, "stay");
      if (to) { e.preventDefault(); focusCell(rowIdx, to); }
    } else if (e.key === "Escape") {
      e.preventDefault();
      cellNav.onDraft(`${rowKey(row)}:${col.key}`, null);
      skipBlur.current = true;
      e.currentTarget.blur();
    } else if (e.key === "ArrowDown" || e.key === "ArrowUp") {
      e.preventDefault();
      const to = rowIdx + (e.key === "ArrowDown" ? 1 : -1);
      if (to >= 0 && to < count) focusCell(to, col.key);
    }
  }

  const activeIdx = activeKey !== undefined ? rows.findIndex((r) => rowKey(r) === activeKey) : cursorIdx;
  // aria-activedescendant chỉ được trỏ tới dòng ĐANG có trong DOM: dòng ngoài cửa sổ ảo thì bỏ thuộc tính (scrollToIndex dựng dòng đó ngay sau phím).
  const activeRendered = activeIdx >= 0 && items.some((it) => it.index === activeIdx);

  function onKeyDown(e: KeyboardEvent<HTMLTableElement>) {
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
    // Vùng cuộn là div thường; phần tử lấy focus là chính <table role="grid"> (grid cho phép aria-activedescendant và sở hữu các dòng của nó).
    <div ref={parentRef} className={s.vscroll} style={{ "--vh": `${height}px` } as CSSProperties} data-part="virtual-scroll">
      <table
        className={[s.table, s.virtual].join(" ")}
        role="grid"
        tabIndex={0}
        aria-rowcount={count + 1 + (footer ? 1 : 0)}
        aria-activedescendant={activeRendered ? `vt-${rowKey(rows[activeIdx])}` : undefined}
        onKeyDown={onKeyDown}
      >
        <caption className="ep-sr-only">{caption}</caption>
        <thead>
          <tr aria-rowindex={1}>
            {columns.map((c) => (
              <th key={c.key} scope="col" style={{ "--w": c.width, textAlign: c.align === "end" ? "right" : c.align === "center" ? "center" : "left" } as CSSProperties} className={[s.th, c.frozen ? s.frozen : ""].join(" ")}>
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
                aria-rowindex={it.index + 2}
                className={[onRowClick ? s.clickable : "", on ? s.active : ""].join(" ")}
                aria-current={on ? "true" : undefined}
                {...rowAttrs?.(row)}
                onClick={onRowClick ? () => { setCursorIdx(it.index); onRowClick(row); } : undefined}
              >
                {columns.map((c) => {
                  const edit = cellNav && c.cell?.editable(row);
                  const ck = `${key}:${c.key}`;
                  return (
                    <td key={c.key} className={c.frozen ? s.frozen : undefined} style={{ textAlign: c.align === "end" ? "right" : c.align === "center" ? "center" : "left" }}>
                      {edit && c.cell ? (
                        <span className={s.cellBox}>
                          <input
                            className={[s.cellInput, cellNav.saved?.has(ck) ? "ep-saved ep-motion-safe" : ""].join(" ")}
                            data-cell={ck}
                            inputMode="decimal"
                            aria-label={c.cell.label(row)}
                            aria-invalid={c.cell.invalid?.(row) ? true : undefined}
                            value={cellNav.drafts.get(ck) ?? (focused === ck || !c.cell.display ? c.cell.value(row) : c.cell.display(row))}
                            onChange={(e) => cellNav.onDraft(ck, e.target.value)}
                            onFocus={() => setFocused(ck)}
                            onBlur={(e) => {
                              setFocused(null);
                              if (skipBlur.current) {
                                skipBlur.current = false;
                                return;
                              }
                              if (cellNav.drafts.has(ck)) cellNav.onCommit(row, c.key, e.currentTarget.value, "stay");
                            }}
                            onKeyDown={(e) => cellKey(e, row, it.index, c)}
                          />
                          {c.cell.status?.(row)}
                        </span>
                      ) : (
                        c.render(row)
                      )}
                    </td>
                  );
                })}
              </tr>
            );
          })}
          {spacer(padBottom, "pb")}
          {footer && (
            <tr data-part="load-more" aria-rowindex={count + 2} aria-busy={pagination?.loading || undefined}>
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
