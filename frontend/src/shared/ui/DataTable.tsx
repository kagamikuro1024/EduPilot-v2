"use client";

import { useRouter } from "next/navigation";
import type { ReactNode } from "react";
import s from "./DataTable.module.css";

export type Column<T> = {
  key: string;
  header: ReactNode;
  render: (row: T) => ReactNode;
  align?: "start" | "end" | "center";
  width?: string;
  /** cột cố định bên trái khi cuộn ngang (tên sinh viên…) */
  frozen?: boolean;
  /** ẩn ở màn < 720px */
  hideOnMobile?: boolean;
};

/**
 * Bảng so sánh: header dính, hàng 44–52px, không kẻ dọc (DESIGN.md §10.4).
 * `rowHref` làm cả hàng là liên kết; chọn nhiều qua `selection`.
 */
export function DataTable<T>({
  columns,
  rows,
  rowKey,
  rowHref,
  onRowClick,
  activeKey,
  caption,
  empty,
  selection,
  dense,
}: {
  columns: Column<T>[];
  rows: T[];
  rowKey: (row: T) => string;
  rowHref?: (row: T) => string;
  onRowClick?: (row: T) => void;
  activeKey?: string;
  caption: string;
  empty?: ReactNode;
  selection?: { selected: Set<string>; onChange: (next: Set<string>) => void };
  dense?: boolean;
}) {
  const router = useRouter();
  const allSelected = selection && rows.length > 0 && rows.every((r) => selection.selected.has(rowKey(r)));

  function toggle(key: string) {
    if (!selection) return;
    const next = new Set(selection.selected);
    if (next.has(key)) next.delete(key);
    else next.add(key);
    selection.onChange(next);
  }

  function activate(row: T) {
    if (onRowClick) onRowClick(row);
    else if (rowHref) router.push(rowHref(row));
  }

  const clickable = Boolean(onRowClick || rowHref);

  return (
    <div className={s.scroll}>
      <table className={[s.table, dense ? s.dense : ""].join(" ")}>
        <caption className="ep-sr-only">{caption}</caption>
        <thead>
          <tr>
            {selection && (
              <th className={s.checkCol} scope="col">
                <input
                  type="checkbox"
                  aria-label="Chọn tất cả"
                  checked={Boolean(allSelected)}
                  onChange={() => selection.onChange(allSelected ? new Set() : new Set(rows.map(rowKey)))}
                />
              </th>
            )}
            {columns.map((c) => (
              <th
                key={c.key}
                scope="col"
                style={{ width: c.width, textAlign: c.align === "end" ? "right" : c.align === "center" ? "center" : "left" }}
                className={[c.frozen ? s.frozen : "", c.hideOnMobile ? s.hideMobile : ""].join(" ")}
              >
                {c.header}
              </th>
            ))}
          </tr>
        </thead>
        <tbody>
          {rows.map((row) => {
            const key = rowKey(row);
            return (
              <tr
                key={key}
                className={[clickable ? s.clickable : "", key === activeKey || selection?.selected.has(key) ? s.active : ""].join(" ")}
                onClick={clickable ? () => activate(row) : undefined}
                onKeyDown={clickable ? (e) => e.key === "Enter" && activate(row) : undefined}
                tabIndex={clickable ? 0 : undefined}
              >
                {selection && (
                  <td className={s.checkCol} onClick={(e) => e.stopPropagation()}>
                    <input type="checkbox" aria-label="Chọn hàng" checked={selection.selected.has(key)} onChange={() => toggle(key)} />
                  </td>
                )}
                {columns.map((c) => (
                  <td
                    key={c.key}
                    style={{ textAlign: c.align === "end" ? "right" : c.align === "center" ? "center" : "left" }}
                    className={[c.frozen ? s.frozen : "", c.hideOnMobile ? s.hideMobile : ""].join(" ")}
                  >
                    {c.render(row)}
                  </td>
                ))}
              </tr>
            );
          })}
        </tbody>
      </table>
      {rows.length === 0 && empty && <div className={s.empty}>{empty}</div>}
    </div>
  );
}
