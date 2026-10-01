"use client";

import Link from "next/link";
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
  /** ẩn ở màn < 720px (cả chế độ bảng cuộn lẫn chế độ danh sách) */
  hideOnMobile?: boolean;
  /** nhãn dòng phụ ở chế độ danh sách; mặc định dùng `header` nếu là chữ */
  mobileLabel?: string;
  /** cột chính của chế độ danh sách (in đậm, dòng đầu); mặc định cột `frozen` hoặc cột đầu tiên */
  primary?: boolean;
  /** móc đo dev: `data-part` đặt lên ô tiêu đề của cột (vd. `col-qt`, `col-status`) */
  part?: string;
};

/**
 * Bảng so sánh: header dính, hàng 44–52px, không kẻ dọc (DESIGN.md §10.4).
 * `rowHref` làm cả hàng là liên kết; chọn nhiều qua `selection`.
 *
 * Dưới 720px (`mobile`):
 *  - "list" (mặc định): mỗi hàng thành mục danh sách 2–3 dòng — cột chính đậm, các cột còn lại là dòng phụ "Nhãn: giá trị".
 *    `mobileRow` thay hẳn nội dung mục (dùng khi hàng có điều khiển riêng như điểm danh).
 *  - "scroll": giữ bảng, cuộn ngang trong vùng `data-scroll-x`, cột `frozen` dính trái (sổ điểm — cần so cột).
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
  mobile = "list",
  mobileRow,
  scrollHint = "Vuốt ngang để xem thêm",
  rowAttrs,
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
  mobile?: "list" | "scroll";
  mobileRow?: (row: T) => ReactNode;
  /** dòng gợi ý cuộn ngang, chỉ hiện dưới 720px ở chế độ `scroll` (03-AC7) */
  scrollHint?: ReactNode;
  /** móc đo dev trên `<tr>` (vd. `{ "data-part": "student-row", "data-student-id": id }`) — chỉ ở hàng bảng để không đếm đôi với chế độ danh sách */
  rowAttrs?: (row: T) => Record<string, string>;
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
    <div className={s.root}>
      {mobile === "scroll" && scrollHint && <p className={s.scrollHint}>{scrollHint}</p>}
      <div className={[s.scroll, mobile === "list" ? s.tableOnly : ""].join(" ")} data-scroll-x={mobile === "scroll" ? "" : undefined}>
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
                data-part={c.part}
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
                {...rowAttrs?.(row)}
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
                    data-part={c.part}
                  >
                    {c.render(row)}
                  </td>
                ))}
              </tr>
            );
          })}
        </tbody>
      </table>
      </div>
      {mobile === "list" && (
        <ul className={s.list} aria-label={caption}>
          {rows.map((row) => {
            const key = rowKey(row);
            const main = columns.find((c) => c.primary) ?? columns.find((c) => c.frozen) ?? columns[0];
            const rest = columns.filter((c) => c !== main && !c.hideOnMobile);
            const href = rowHref?.(row);
            const inner = mobileRow ? (
              mobileRow(row)
            ) : (
              <>
                <div className={s.itemMain}>{main.render(row)}</div>
                <dl className={s.itemMeta}>
                  {rest.map((c) => {
                    const label = c.mobileLabel ?? (typeof c.header === "string" ? c.header : null);
                    return (
                      <div key={c.key} className={s.itemField}>
                        {label && <dt>{label}</dt>}
                        <dd>{c.render(row)}</dd>
                      </div>
                    );
                  })}
                </dl>
              </>
            );
            return (
              <li key={key} className={[s.item, selection ? s.itemSelectable : "", key === activeKey ? s.itemActive : ""].join(" ")}>
                {selection && (
                  <label className={s.itemCheck}>
                    <input type="checkbox" aria-label="Chọn hàng" checked={selection.selected.has(key)} onChange={() => toggle(key)} />
                  </label>
                )}
                {href ? (
                  <Link href={href} className={s.itemLink}>
                    {inner}
                  </Link>
                ) : onRowClick ? (
                  <div className={s.itemLink} role="button" tabIndex={0} onClick={() => onRowClick(row)} onKeyDown={(e) => e.key === "Enter" && onRowClick(row)}>
                    {inner}
                  </div>
                ) : (
                  <div className={s.itemStatic}>{inner}</div>
                )}
              </li>
            );
          })}
        </ul>
      )}
      {rows.length === 0 && empty && <div className={s.empty}>{empty}</div>}
    </div>
  );
}
