"use client";

import type { ReactNode } from "react";
import { Button, EmptyState, InlineNotice, Select, Skeleton } from "@/shared/ui";
import s from "./LLMRouteTable.module.css";

export type RouteRow = { task: string; label: string; laneLabel: string; modelId: string };
export type ModelGroup = { provider: string; models: Array<{ id: string; label: string }> };

/**
 * Bảng "Mô hình theo tác vụ" (FEAT-llm-gateway US-P1-05 AC5): mỗi tác vụ một chọn mô hình chính, nhóm theo nhà cung cấp, kèm nhãn làn
 * đọc được. Chỉ trình bày: dữ liệu và việc ghi do màn gọi. Dưới 720 px thành danh sách xếp chồng (không cuộn ngang).
 * `renderExtra` đặt nội dung ngay dưới hàng (dòng Hoàn tác, ô nâng cao).
 */
export function LLMRouteTable({
  rows,
  groups,
  readOnly,
  disabled,
  loading,
  error,
  onRetry,
  emptyAction,
  openTask,
  onToggleAdvanced,
  onChange,
  renderExtra,
}: {
  rows: RouteRow[];
  groups: ModelGroup[];
  /** giảng viên: chỉ xem */
  readOnly?: boolean;
  disabled?: boolean;
  loading?: boolean;
  error?: ReactNode;
  onRetry?: () => void;
  /** nút duy nhất của trạng thái rỗng */
  emptyAction?: ReactNode;
  /** tác vụ đang mở "Cài đặt nâng cao" */
  openTask?: string | null;
  onToggleAdvanced?: (task: string) => void;
  onChange?: (task: string, modelId: string) => void;
  renderExtra?: (row: RouteRow) => ReactNode;
}) {
  if (loading) return <Skeleton lines={6} />;
  if (error)
    return (
      <InlineNotice tone="danger" title={error} action={onRetry && <Button size="sm" onClick={onRetry}>Thử lại</Button>}>
        Cấu hình hiện có không bị ảnh hưởng.
      </InlineNotice>
    );
  if (rows.length === 0 || groups.every((g) => g.models.length === 0))
    return (
      <EmptyState title="Chưa có mô hình nào để chọn" action={emptyAction}>
        Thêm nhà cung cấp có mô hình trò chuyện, rồi quay lại đây để chọn.
      </EmptyState>
    );
  const lock = disabled || readOnly;
  return (
    <ul className={s.list} aria-label="Mô hình theo tác vụ" data-part="route-table">
      {rows.map((r) => (
        <li key={r.task} className={s.row} data-part="route-row" data-task={r.task}>
          <div className={s.main}>
            <div className={s.name}>
              <span className={s.label}>{r.label}</span>
              <span className={s.lane}>{r.laneLabel}</span>
            </div>
            <Select aria-label={`Mô hình cho ${r.label}`} value={r.modelId} disabled={lock} className={s.select} onChange={(e) => onChange?.(r.task, e.target.value)}>
              {!r.modelId && <option value="">Chưa chọn — dùng cấu hình mặc định</option>}
              {/* tùy chọn phẳng "Nhà · mô hình" (optgroup làm khung đo của QC 1.5 báo cắt chữ) */}
              {groups.flatMap((g) =>
                g.models.map((m) => (
                  <option key={m.id} value={m.id}>
                    {g.provider} · {m.label}
                  </option>
                )),
              )}
            </Select>
            {onToggleAdvanced && (
              <Button variant="text" size="sm" aria-expanded={openTask === r.task} disabled={disabled} onClick={() => onToggleAdvanced(r.task)}>
                Cài đặt nâng cao
              </Button>
            )}
          </div>
          {renderExtra?.(r)}
        </li>
      ))}
    </ul>
  );
}
