"use client";

import { useQueryClient } from "@tanstack/react-query";
import { useRef, useState } from "react";
import { ApiError, newIdempotencyKey } from "@/shared/data";
import { checkGradeImportFile, uploadGradeImportFile } from "@/shared/data/uploadFile";
import { Button, Checkbox, DataTable, Dialog, Field, Input, Select } from "@/shared/ui";
import { gradebookKey, importEntries, publishItem, type Column, type ImportMapping, type ImportOut } from "./gradeApi";
import s from "./ImportDialog.module.css";

type Step = 1 | 2 | 3;
const colName = (i: number) => String.fromCharCode(65 + i);
const vi = (v: string | null) => (v === null ? "—" : v.replace(".", ","));

/**
 * Nhập điểm từ XLSX trong một Dialog ba bước: Chọn tệp & cột điểm → Ánh xạ cột → Xem trước (SRS FEAT-gradebook 4.6.4).
 * Xem trước không ghi; `Ghi` gửi lại đúng tệp (máy chủ đọc lại từ kho tệp) với `Idempotency-Key` cố định cho mỗi lần ghi.
 */
export function ImportDialog({ course, columns, onClose }: { course: string; columns: Column[]; onClose: () => void }) {
  const qc = useQueryClient();
  const items = columns.filter((c) => c.source === "MANUAL");
  const [step, setStep] = useState<Step>(1);
  const [file, setFile] = useState<File | null>(null);
  const [item, setItem] = useState(items[0]?.item_id ?? "");
  const [upload, setUpload] = useState("");
  const [headers, setHeaders] = useState<string[]>([]);
  const [map, setMap] = useState<ImportMapping>({ header_row: 1, student_code_col: "", score_col: "" });
  const [out, setOut] = useState<ImportOut | null>(null);
  const [publish, setPublish] = useState(false);
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const commitKey = useRef("");
  const picked = items.find((c) => c.item_id === item);

  const fail = (e: unknown, fallback: string) => setError(e instanceof ApiError ? e.userMessage : fallback);

  async function inspect(up: string, headerRow: number) {
    const r = await importEntries(course, { upload_id: up, item_id: item, mode: "preview", mapping: { header_row: headerRow } });
    setHeaders(r.headers);
    setMap((m) => ({ header_row: headerRow, student_code_col: m.student_code_col && headerRow === m.header_row ? m.student_code_col : r.suggested_mapping.student_code_col, score_col: m.score_col && headerRow === m.header_row ? m.score_col : r.suggested_mapping.score_col }));
  }

  async function next1() {
    if (!file) return;
    const bad = checkGradeImportFile(file);
    if (bad) return setError(bad);
    setBusy(true);
    setError(null);
    try {
      const id = upload || (await uploadGradeImportFile(course, file));
      setUpload(id);
      await inspect(id, map.header_row);
      setStep(2);
    } catch (e) {
      fail(e, "Chưa tải được tệp.");
    } finally {
      setBusy(false);
    }
  }

  async function reload() {
    setUpload("");
    await next1();
  }

  async function preview() {
    setBusy(true);
    setError(null);
    try {
      setOut(await importEntries(course, { upload_id: upload, item_id: item, mode: "preview", mapping: map }));
      setStep(3);
    } catch (e) {
      fail(e, "Chưa xem trước được.");
    } finally {
      setBusy(false);
    }
  }

  async function commit() {
    if (!out) return;
    setBusy(true);
    setError(null);
    commitKey.current ||= newIdempotencyKey();
    try {
      await importEntries(course, { upload_id: upload, item_id: item, mode: "commit", mapping: map, skip_errors: out.errors.length > 0 }, commitKey.current);
      if (publish) await publishItem(course, item, true);
      await qc.invalidateQueries({ queryKey: gradebookKey(course) });
      onClose();
    } catch (e) {
      commitKey.current = "";
      fail(e, "Chưa ghi được điểm.");
    } finally {
      setBusy(false);
    }
  }

  const errs = out?.errors ?? [];
  const primary =
    step === 1 ? (
      <Button variant="primary" disabled={!file || !item || busy} loading={busy} onClick={() => void next1()}>Tiếp tục</Button>
    ) : step === 2 ? (
      <Button variant="primary" disabled={!map.student_code_col || !map.score_col || map.student_code_col === map.score_col || busy} loading={busy} onClick={() => void preview()}>Xem trước</Button>
    ) : (
      <Button variant="primary" disabled={busy || !out || out.rows_ok === 0} loading={busy} onClick={() => void commit()}>
        {errs.length > 0 ? `Bỏ qua ${errs.length} dòng lỗi và ghi ${out?.rows_ok ?? 0} dòng` : `Ghi ${out?.rows_ok ?? 0} dòng`}
      </Button>
    );

  return (
    <Dialog
      open
      onClose={onClose}
      title="Nhập điểm từ XLSX"
      error={error && <span>{error}{step === 1 && file && upload === "" && <Button size="sm" variant="ghost" onClick={() => void reload()}>Tải lại</Button>}</span>}
      footer={
        <>
          {step > 1 && <Button variant="ghost" disabled={busy} onClick={() => setStep((step - 1) as Step)}>Quay lại</Button>}
          <Button variant="ghost" onClick={onClose}>Đóng</Button>
          {primary}
        </>
      }
    >
      <div className={s.body} data-step={step}>
        {step === 1 && (
          <>
            <Field label="Nhập vào">{(id) => (
              <Select id={id} value={item} onChange={(e) => setItem(e.target.value)}>
                {items.map((c) => <option key={c.item_id} value={c.item_id}>{c.name}</option>)}
              </Select>
            )}</Field>
            <Field label="Tệp XLSX">{(id) => (
              <input id={id} type="file" accept=".xlsx" data-part="import-input" className={s.file} onChange={(e) => { setFile(e.target.files?.[0] ?? null); setUpload(""); setError(null); }} />
            )}</Field>
            {file && <p className={s.fileName} data-part="import-file">{file.name}</p>}
          </>
        )}
        {step === 2 && (
          <>
            <Field label="Dòng tiêu đề">{(id) => (
              <Input id={id} type="number" min={1} max={100} value={map.header_row} className={s.narrow} onChange={(e) => setMap({ ...map, header_row: Math.max(1, Number(e.target.value) || 1) })} onBlur={() => void inspect(upload, map.header_row).catch((e) => fail(e, "Chưa đọc được tiêu đề."))} />
            )}</Field>
            {(["student_code_col", "score_col"] as const).map((k) => (
              <Field key={k} label={k === "student_code_col" ? "Cột MSSV" : "Cột chứa điểm"}>{(id) => (
                <Select id={id} value={map[k]} onChange={(e) => setMap({ ...map, [k]: e.target.value })}>
                  <option value="">Chọn cột</option>
                  {headers.map((h, i) => <option key={i} value={colName(i)}>{`${colName(i)} · ${h || "(trống)"}`}</option>)}
                </Select>
              )}</Field>
            ))}
          </>
        )}
        {step === 3 && out && (
          <>
            <p className={s.summary} data-part="import-summary">
              {`${out.rows_ok} dòng hợp lệ · ${out.overwrites} ghi đè · ${out.unchanged} không đổi · ${errs.length} lỗi`}
            </p>
            {errs.length > 0 && (
              <ul className={s.errors} data-part="import-errors">
                {errs.slice(0, 100).map((e) => <li key={e.row}>{`Dòng ${e.row}${e.student_code ? ` · ${e.student_code}` : ""} · ${e.message}`}</li>)}
                {errs.length > 100 && <li>{`…và ${errs.length - 100} dòng nữa`}</li>}
              </ul>
            )}
            {out.sample.length > 0 && (
              <div data-part="import-sample">
                <DataTable
                  dense
                  caption="Mẫu điểm sẽ ghi"
                  rows={out.sample}
                  rowKey={(r) => r.student.id}
                  columns={[
                    { key: "st", header: "Sinh viên", render: (r) => `${r.student.full_name} · ${r.student.student_code}` },
                    { key: "b", header: "Trước", align: "end", width: "80px", render: (r) => vi(r.before) },
                    { key: "a", header: "Sau", align: "end", width: "80px", render: (r) => vi(r.after) },
                  ]}
                />
              </div>
            )}
            <Checkbox label="Công bố cột cho sinh viên" checked={publish} onChange={(e) => setPublish(e.target.checked)} disabled={!picked} />
          </>
        )}
      </div>
    </Dialog>
  );
}
