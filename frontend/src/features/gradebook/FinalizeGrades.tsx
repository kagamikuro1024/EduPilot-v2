"use client";

import { useQueryClient } from "@tanstack/react-query";
import { useRef, useState } from "react";
import { ApiError, newIdempotencyKey } from "@/shared/data";
import { Button, ConfirmIrreversible, DataTable, Drawer, EmptyState, Field, Skeleton, Textarea } from "@/shared/ui";
import s from "./FinalizeGrades.module.css";
import { finalizeGradebook, gradebookKey, schemeStatusKey, snapshotsKey, unlockGradebook, usePreviewSummary, useSnapshotRows, useSnapshots, type Incomplete } from "./gradeApi";

const LETTERS = ["A+", "A", "B+", "B", "C+", "C", "D+", "D", "F"];
const vi = (v: string) => v.replace(".", ",");
export const stamp = (iso: string) => new Date(iso).toLocaleString("vi-VN", { day: "2-digit", month: "2-digit", hour: "2-digit", minute: "2-digit" });

/** "A+ 3, A 3, B+ 3, …": chỉ những chữ có sinh viên, theo thứ tự thang. */
function distribution(d: Record<string, number>) {
  const keys = [...LETTERS.filter((k) => d[k]), ...Object.keys(d).filter((k) => !LETTERS.includes(k) && d[k]).sort()];
  return keys.map((k) => `${k} ${d[k]}`).join(", ");
}

/**
 * `Chốt điểm`: hộp xác nhận nêu hậu quả bằng số (số sinh viên, phân bố chữ, số sinh viên liệt) — số lấy từ xem trước của máy chủ, không tính ở đây.
 * 409 `GRADES_INCOMPLETE` đóng hộp và trả danh sách thiếu cho `onIncomplete`; lỗi khác hiện trong hộp với `Thử lại` (cùng Idempotency-Key cho đến khi đổi ý định).
 */
export function FinalizeDialog({ course, onClose, onIncomplete }: { course: string; onClose: () => void; onIncomplete: (i: Incomplete) => void }) {
  const qc = useQueryClient();
  const sum = usePreviewSummary(course, true);
  const key = useRef(newIdempotencyKey());
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState<string | undefined>();
  const d = sum.data;
  const consequence = d
    ? [`Chốt ${d.students} sinh viên`, d.students > 0 && distribution(d.distribution) ? `phân bố: ${distribution(d.distribution)}` : "", `${d.liet} sinh viên liệt`, d.incomplete > 0 ? `${d.incomplete} sinh viên chưa đủ điểm` : ""].filter(Boolean).join(" · ")
    : "Đang tính…";

  async function go() {
    setBusy(true);
    setError(undefined);
    try {
      await finalizeGradebook(course, key.current);
      await Promise.all([qc.invalidateQueries({ queryKey: gradebookKey(course) }), qc.invalidateQueries({ queryKey: snapshotsKey(course) }), qc.invalidateQueries({ queryKey: schemeStatusKey(course) })]);
      onClose();
    } catch (e) {
      if (e instanceof ApiError && e.code === "GRADES_INCOMPLETE") {
        const det = e.details as Incomplete | undefined;
        onIncomplete({ total: det?.total ?? 0, students: det?.students ?? [] });
        onClose();
      } else {
        key.current = e instanceof ApiError && e.status < 500 ? newIdempotencyKey() : key.current;
        setError(e instanceof ApiError ? e.userMessage : "Chưa chốt được.");
      }
    } finally {
      setBusy(false);
    }
  }

  return <ConfirmIrreversible open onClose={onClose} onConfirm={() => void go()} title="Chốt điểm" consequence={consequence} confirmLabel="Chốt điểm" disabledReason={d ? undefined : "Đang tính…"} loading={busy} error={error} />;
}

/** `Mở khoá`: nêu hậu quả bằng số, lý do bắt buộc 10–500 ký tự (khoá nút khi chưa đủ). Bản chốt cũ giữ nguyên. */
export function UnlockDialog({ course, version, students, onClose }: { course: string; version: number; students: number; onClose: () => void }) {
  const qc = useQueryClient();
  const [reason, setReason] = useState("");
  const key = useRef(newIdempotencyKey());
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState<string | undefined>();
  const n = [...reason.trim()].length;
  const ok = n >= 10 && n <= 500;
  const consequence = `Mở khoá bản chốt v${version} của ${students} sinh viên. Bản v${version} vẫn được giữ nguyên.`;

  async function go() {
    setBusy(true);
    setError(undefined);
    try {
      await unlockGradebook(course, reason.trim(), key.current);
      await Promise.all([qc.invalidateQueries({ queryKey: gradebookKey(course) }), qc.invalidateQueries({ queryKey: schemeStatusKey(course) })]);
      onClose();
    } catch (e) {
      key.current = e instanceof ApiError && e.status < 500 ? newIdempotencyKey() : key.current;
      setError(e instanceof ApiError ? e.userMessage : "Chưa mở khoá được.");
    } finally {
      setBusy(false);
    }
  }

  return (
    <ConfirmIrreversible open onClose={onClose} onConfirm={() => void go()} title="Mở khoá" consequence={consequence} confirmLabel="Mở khoá" disabledReason={ok ? undefined : `${consequence} Nhập lý do 10–500 ký tự.`} loading={busy} error={error}>
      <Field label="Lý do" error={n > 500 ? "Tối đa 500 ký tự." : undefined}>{(id, by) => <Textarea id={id} aria-describedby={by} rows={3} value={reason} onChange={(e) => setReason(e.target.value)} placeholder="Phúc khảo sinh viên A: sửa điểm giữa kỳ" />}</Field>
    </ConfirmIrreversible>
  );
}

/** `Lịch sử chốt`: các phiên bản đã chốt (mới nhất trước); chọn một phiên bản để xem lại điểm đã chốt (chỉ đọc). */
export function SnapshotHistory({ course, onClose }: { course: string; onClose: () => void }) {
  const list = useSnapshots(course, true);
  const [version, setVersion] = useState<number | null>(null);
  const rows = useSnapshotRows(course, version);
  const items = rows.data?.pages.flatMap((p) => p.items) ?? [];
  return (
    <Drawer open onClose={onClose} title="Lịch sử chốt">
      {list.isPending ? (
        <Skeleton lines={4} />
      ) : (list.data ?? []).length === 0 ? (
        <EmptyState title="Chưa chốt lần nào." />
      ) : (
        <div data-part="snapshot-history">
          <ul className={s.versions}>
            {(list.data ?? []).map((v) => (
              <li key={v.version}>
                <Button size="sm" variant={version === v.version ? "primary" : "secondary"} aria-pressed={version === v.version} onClick={() => setVersion(v.version)}>{`Bản chốt v${v.version} · ${stamp(v.locked_at)} · ${v.students} sinh viên`}</Button>
                {v.unlock_reason && <p className={s.why}>{v.unlock_reason}</p>}
              </li>
            ))}
          </ul>
          {version !== null && (
            <DataTable
              dense
              caption={`Điểm đã chốt v${version}`}
              loading={rows.isPending ? 5 : undefined}
              rows={items}
              rowKey={(r) => r.student.id}
              columns={[
                { key: "st", header: "Sinh viên", render: (r) => `${r.student.full_name} · ${r.student.student_code}` },
                { key: "g", header: "Điểm", align: "end", width: "72px", render: (r) => vi(r.numeric_grade) },
                { key: "l", header: "Chữ", align: "center", width: "64px", render: (r) => r.letter },
                { key: "f", header: "Ghi chú", render: (r) => (r.failed_reason ? "Liệt" : "") },
              ]}
              pagination={{ nextCursor: rows.hasNextPage ? (rows.data?.pages.at(-1)?.next_cursor ?? null) : null, onLoadMore: () => void rows.fetchNextPage(), loading: rows.isFetchingNextPage }}
            />
          )}
        </div>
      )}
    </Drawer>
  );
}
