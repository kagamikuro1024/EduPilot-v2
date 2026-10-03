"use client";

import Link from "next/link";
import { useRouter } from "next/navigation";
import { lazy, Suspense, useRef, type CSSProperties, type ReactNode } from "react";
import { useScrollRow } from "@/shared/lib/useScrollRow";
import s from "./DataTable.module.css";

type VirtualProps<T> = {
  columns: Column<T>[];
  rows: T[];
  rowKey: (row: T) => string;
  onRowClick?: (row: T) => void;
  activeKey?: string;
  caption: string;
  height: number;
  pagination?: Pagination;
};
// Nạp lười (react-virtual không vào gói của route không dùng bảng dài). lazy làm mất tham số kiểu generic → ép lại chữ ký.
const VirtualTable = lazy(() => import("./DataTableVirtual")) as unknown as <T>(props: VirtualProps<T>) => ReactNode;

/** Phân trang con trỏ cho bảng ảo hoá: `onLoadMore(nextCursor)` gọi đúng MỘT lần cho mỗi con trỏ; `nextCursor: null` = hết. */
export type Pagination = {
  nextCursor: string | null;
  onLoadMore: (cursor: string) => void;
  loading?: boolean;
  /** lỗi tải trang kế: hiện ở đáy kèm `Thử lại`, KHÔNG xoá các dòng đã có */
  error?: ReactNode;
};

export type Column<T> = {
  key: string;
  header: ReactNode;
  render: (row: T) => ReactNode;
  align?: "start" | "end" | "center";
  width?: string;
  /** cột cố định bên trái khi cuộn ngang (tên sinh viên…) */
  frozen?: boolean;
  /** bề rộng ở màn < 720px (chế độ bảng cuộn); mặc định = `width`. Không cột nào bị ẩn: cột thừa cuộn ngang hoặc thành dòng phụ. */
  mobileWidth?: string;
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
  virtual,
  pagination,
  loading,
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
  /** danh sách dài: ảo hoá dòng trong khung cao `height` px (chỉ dựng dòng nhìn thấy); bỏ chế độ danh sách ở < 720px */
  virtual?: { height: number };
  pagination?: Pagination;
  /** đang tải: giữ tiêu đề cột thật + từng ấy dòng khung xương cao đúng bằng dòng thật (CLS ≈ 0), aria-busy */
  loading?: number;
}) {
  const router = useRouter();
  const scrollRef = useRef<HTMLDivElement>(null);
  // bảng rộng hơn khung (ví dụ 720 px có thanh bên) cuộn ngang được và nói rõ: data-scroll-x + mép mờ
  useScrollRow(scrollRef, columns.length + rows.length);
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

  if (virtual) {
    return (
      <div className={s.root}>
        {/* khung chờ cao ĐÚNG bằng khung thật nên nạp xong không dịch bố cục (CLS) */}
        <Suspense fallback={<div aria-busy="true" className={s.vscroll} style={{ "--vh": `${virtual.height}px` } as CSSProperties} />}>
          <VirtualTable columns={columns} rows={rows} rowKey={rowKey} onRowClick={onRowClick} activeKey={activeKey} caption={caption} height={virtual.height} pagination={pagination} />
        </Suspense>
        {rows.length === 0 && empty && <div className={s.empty}>{empty}</div>}
      </div>
    );
  }

  return (
    <div className={s.root}>
      {mobile === "scroll" && scrollHint && <p className={s.scrollHint}>{scrollHint}</p>}
      <div className={[s.scroll, mobile === "list" ? s.tableOnly : ""].join(" ")} ref={scrollRef} aria-busy={loading ? true : undefined}>
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
                style={{ "--w": c.width, "--wm": c.mobileWidth ?? c.width, textAlign: c.align === "end" ? "right" : c.align === "center" ? "center" : "left" } as CSSProperties}
                className={[s.th, c.frozen ? s.frozen : ""].join(" ")}
                data-part={c.part}
              >
                {c.header}
              </th>
            ))}
          </tr>
        </thead>
        <tbody>
          {loading ? (
            Array.from({ length: loading }, (_, i) => (
              <tr key={`sk-${i}`} aria-hidden>
                {selection && <td className={s.checkCol} />}
                {columns.map((c) => (
                  <td key={c.key}>
                    <span className={s.skCell} />
                  </td>
                ))}
              </tr>
            ))
          ) : rows.map((row) => {
            const key = rowKey(row);
            return (
              <tr
                key={key}
                className={[clickable ? s.clickable : "", key === activeKey || selection?.selected.has(key) ? s.active : ""].join(" ")}
                aria-current={key === activeKey ? "true" : undefined}
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
                    className={c.frozen ? s.frozen : undefined}
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
      {mobile === "list" && !loading && (
        <ul className={s.list} aria-label={caption}>
          {rows.map((row) => {
            const key = rowKey(row);
            const main = columns.find((c) => c.primary) ?? columns.find((c) => c.frozen) ?? columns[0];
            const rest = columns.filter((c) => c !== main);
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
              <li key={key} aria-current={key === activeKey ? "true" : undefined} className={[s.item, selection ? s.itemSelectable : "", key === activeKey ? s.itemActive : ""].join(" ")}>
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
