"use client";

import { useQueryClient } from "@tanstack/react-query";
import { useRef, useState, type DragEvent } from "react";
import { ApiError, apiClient, useIdempotentMutation } from "@/shared/data";
import { Button, Checkbox, type Column, DataTable, InlineNotice, Panel, PanelSection } from "@/shared/ui";
import s from "./RosterImport.module.css";
import { TODAY_KEY } from "@/features/today/todayApi";
import { classKey } from "./classApi";

const MAX_BYTES = 2 * 1024 * 1024;
const FIELD: Record<string, string> = { email: "Email", full_name: "Họ và tên", student_code: "MSSV" };

export type RosterReport = {
  total: number;
  created_users: number;
  linked_existing: number;
  already_member: number;
  pending_unverified: number;
  skipped_removed: number;
  errors: { row: number; field: string; code: string; message: string }[];
  dry_run: boolean;
};
type Line = { row: number; field: string; message: string };

/** Số dòng sẽ ghi: tài khoản mới + nối vào tài khoản có sẵn + chờ xác minh email. Dòng đã có trong lớp không đổi gì. */
const writable = (r: RosterReport) => r.created_users + r.linked_existing + r.pending_unverified;

/** Lý do tệp bị từ chối cả tệp (422 / 413): ưu tiên câu máy chủ trả (nêu tên cột thiếu), không lộ chi tiết nội bộ. */
function fileMessage(e: unknown): string {
  if (e instanceof ApiError) {
    if (e.status === 413 || e.code === "PAYLOAD_TOO_LARGE") return "Tệp quá lớn. Tối đa 2 MB.";
    if (e.code === "VALIDATION_FAILED" && Array.isArray(e.details)) {
      const first = (e.details as { message?: string }[])[0];
      if (first?.message) return first.message;
    }
    if (e.code === "COURSE_ARCHIVED") return "Lớp này đã được lưu trữ.";
    if (e.code === "FORBIDDEN") return "Chỉ giảng viên của lớp nhập được danh sách.";
    return e.userMessage;
  }
  return "Chưa nhập được. Hãy thử lại.";
}

const columns: Column<Line>[] = [{ key: "detail", header: "Dòng lỗi", render: (l) => `Dòng ${l.row} · ${FIELD[l.field] ?? l.field} · "${l.message}"` }];

/** Nhập danh sách lớp từ CSV / XLSX: chọn tệp → xem trước (dry_run) → nhập. Tệp được giữ khi mất mạng; "Gửi lại" dùng cùng Idempotency-Key. */
export function RosterImport({ courseId }: { courseId: string }) {
  const qc = useQueryClient();
  const input = useRef<HTMLInputElement>(null);
  const [file, setFile] = useState<File | null>(null);
  const [invites, setInvites] = useState(true);
  const [over, setOver] = useState(false);
  const [problem, setProblem] = useState<string | null>(null);
  const [preview, setPreview] = useState<RosterReport | null>(null);
  const [busy, setBusy] = useState(false);
  const [done, setDone] = useState<RosterReport | null>(null);

  const form = (f: File) => {
    const fd = new FormData();
    fd.append("file", f);
    fd.append("send_invites", String(invites));
    return fd;
  };
  const doImport = useIdempotentMutation<File, RosterReport>((f, key) => apiClient.post<RosterReport>(`/courses/${courseId}/roster/import`, form(f), { idempotencyKey: key }));

  function pick(f: File | undefined) {
    setPreview(null);
    setDone(null);
    if (!f) return;
    if (!/\.(csv|xlsx)$/i.test(f.name)) {
      setFile(null);
      setProblem("Tệp phải là CSV hoặc XLSX.");
      return;
    }
    if (f.size > MAX_BYTES) {
      setFile(null);
      setProblem("Tệp quá lớn. Tối đa 2 MB.");
      return;
    }
    setProblem(null);
    setFile(f);
  }
  function drop(e: DragEvent) {
    e.preventDefault();
    setOver(false);
    pick(e.dataTransfer.files[0]);
  }

  async function runPreview() {
    if (!file) return;
    setBusy(true);
    setProblem(null);
    try {
      setPreview((await apiClient.post<RosterReport>(`/courses/${courseId}/roster/import`, form(file), { query: { dry_run: true } })).data);
    } catch (e) {
      setProblem(fileMessage(e));
    } finally {
      setBusy(false);
    }
  }
  async function run(retry: boolean) {
    if (!file) return;
    setProblem(null);
    try {
      const out = await (retry ? doImport.retry() : doImport.mutate(file));
      if (out) {
        setDone(out.data);
        setPreview(null);
        setFile(null);
        await qc.invalidateQueries({ queryKey: classKey(courseId) });
        void qc.invalidateQueries({ queryKey: TODAY_KEY });
      }
    } catch (e) {
      if (!(e instanceof ApiError) || e.code !== "NETWORK") setProblem(fileMessage(e));
    }
  }
  const offline = doImport.error?.code === "NETWORK";

  const n = preview ? writable(preview) : 0;
  const lines: Line[] = (preview ?? done)?.errors ?? [];
  const notices = (
    <>
      {offline && (
        <InlineNotice tone="warning" compact action={<Button size="sm" onClick={() => void run(true)} disabled={doImport.pending}>Gửi lại</Button>}>
          Mất kết nối. Tệp vẫn được giữ; gửi lại sẽ không nhập trùng.
        </InlineNotice>
      )}
      {problem && preview && <InlineNotice tone="danger" compact>{problem}</InlineNotice>}
    </>
  );
  const actions =
    file && !done ? (
      <div className={s.actions}>
        {!preview ? (
          <Button variant="primary" onClick={() => void runPreview()} disabled={busy}>Xem trước</Button>
        ) : (
          <Button variant="primary" onClick={() => void run(false)} disabled={n === 0 || doImport.pending}>{`Nhập ${n} sinh viên`}</Button>
        )}
      </div>
    ) : null;
  return (
    <Panel>
      <PanelSection title="Chọn tệp">
        <div className={s.stack}>
          <div
            className={s.drop}
            data-over={over}
            data-invalid={Boolean(problem) && !preview}
            onDragOver={(e) => {
              e.preventDefault();
              setOver(true);
            }}
            onDragLeave={() => setOver(false)}
            onDrop={drop}
          >
          <p className={s.hint}>Thả tệp CSV hoặc XLSX vào đây. Tệp cần ba cột: Email, Họ và tên, MSSV (dòng đầu là tiêu đề, tối đa 500 dòng). Tệp không được lưu lại.</p>
          <input ref={input} hidden type="file" accept=".csv,.xlsx,text/csv,application/vnd.openxmlformats-officedocument.spreadsheetml.sheet" aria-label="Tệp danh sách lớp" onChange={(e) => { pick(e.target.files?.[0]); e.target.value = ""; }} />
          <Button variant="secondary" onClick={() => input.current?.click()}>Chọn tệp</Button>
          {file && <p className={s.file}>{file.name}</p>}
          {problem && !preview && <p className={s.error} role="alert">{problem}</p>}
          </div>

          <Checkbox checked={invites} onChange={(e) => setInvites(e.target.checked)} label="Gửi thư mời cho sinh viên chưa có tài khoản" />
          {!preview && notices}
          {!preview && actions}
        </div>
      </PanelSection>

      {preview && (
        <PanelSection title="Xem trước">
          <div className={s.stack}>
            <p className={s.summary}>
                {preview.total} dòng · {n} sẽ được nhập ({preview.created_users} tài khoản mới, {preview.linked_existing} nối vào tài khoản có sẵn, {preview.pending_unverified} chờ xác minh email) · {preview.already_member + preview.skipped_removed} đã có trong lớp · {preview.errors.length} lỗi
            </p>
            {lines.length > 0 && <DataTable caption="Dòng lỗi" columns={columns} rows={lines} rowKey={(l) => `${l.row}-${l.field}`} dense />}
            {notices}
            {actions}
          </div>
        </PanelSection>
      )}

      {done && (
        <PanelSection title="Kết quả">
          <div className={s.stack}>
            <p className={s.result} role="status">Đã nhập {writable(done)} sinh viên{done.errors.length > 0 ? ` · ${done.errors.length} dòng lỗi chưa nhập` : ""}.</p>
            {lines.length > 0 && <DataTable caption="Dòng lỗi" columns={columns} rows={lines} rowKey={(l) => `${l.row}-${l.field}`} dense />}
          </div>
        </PanelSection>
      )}
    </Panel>
  );
}
