"use client";

import { useQueryClient } from "@tanstack/react-query";
import { useRef, useState } from "react";
import { ApiError, newIdempotencyKey } from "@/shared/data";
import { useCursorList } from "@/shared/data/useCursorList";
import { checkFormsFile, uploadFormsFile } from "@/shared/data/uploadFile";
import { Button, DataTable, Dialog, Field, Input, Select } from "@/shared/ui";
import { qKey, qPath, type QuestionRow } from "@/features/questions/questionsApi";
import { gradebookKey, type Column } from "./gradeApi";
import { importForms, previewForms, type FormsMapping, type FormsPreview, type FormsQuestionMap, type FormsRequest, type FormsResult } from "./formsApi";
import s from "./ImportDialog.module.css";

type Step = 1 | 2 | 3;
type Source = "BANK" | "MANUAL";
type Row = { col: string; question_id: string; accepted: string };
const vi = (v: string) => v.replace(".", ",");
const REASON = { NOT_FOUND: "Không có trong lớp", MISMATCH: "Email và MSSV lệch nhau" } as const;

/**
 * Nhập kết quả Microsoft / Google Forms trong một Dialog ba bước: Chọn tệp & cột điểm → Ánh xạ cột → Xem trước điểm (SRS FEAT-practice 4.5, chỉ TEACHER).
 * Xem trước không ghi; `Ghi` gửi lại đúng ánh xạ (máy chủ đọc lại tệp từ kho) với `Idempotency-Key` cố định cho mỗi lần ghi. Không công bố, không gọi AI.
 */
export function FormsImportDialog({ course, columns, onClose }: { course: string; columns: Column[]; onClose: () => void }) {
  const qc = useQueryClient();
  const items = columns.filter((c) => c.source === "MANUAL");
  const [step, setStep] = useState<Step>(1);
  const [file, setFile] = useState<File | null>(null);
  const [item, setItem] = useState(items[0]?.item_id ?? "");
  const [upload, setUpload] = useState("");
  const [sheet, setSheet] = useState<FormsPreview | null>(null);
  const [headerRow, setHeaderRow] = useState(1);
  const [ids, setIds] = useState({ email_col: "", code_col: "", time_col: "" });
  const [source, setSource] = useState<Source>("BANK");
  const [rows, setRows] = useState<Row[]>([{ col: "", question_id: "", accepted: "" }]);
  const [out, setOut] = useState<FormsResult | null>(null);
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const commitKey = useRef("");
  const bank = useCursorList<QuestionRow>(qKey(course, "forms-picker"), qPath(course), { limit: 100, query: { review_status: "APPROVED" } });
  const pickable = bank.items.filter((q) => q.type !== "ESSAY" && q.type !== "CODE");
  const cols = sheet?.columns ?? [];

  const fail = (e: unknown, fallback: string) => setError(e instanceof ApiError ? e.userMessage : fallback);
  const colOption = (c: { col: string; header: string }) => <option key={c.col} value={c.col}>{`${c.col} · ${c.header || "(trống)"}`}</option>;

  const request = (): FormsRequest => {
    const questions: FormsQuestionMap[] = rows.filter((r) => r.col).map((r) => (source === "BANK" ? { col: r.col, question_id: r.question_id || null } : { col: r.col, accepted: r.accepted.split("|").map((a) => a.trim()).filter(Boolean) }));
    const mapping: FormsMapping = { header_row: headerRow, ...ids, questions };
    return { upload_id: upload, item_id: item, answer_source: source, mapping };
  };
  const ready = (ids.email_col || ids.code_col) && rows.some((r) => r.col) && rows.filter((r) => r.col).every((r) => (source === "BANK" ? r.question_id : r.accepted.trim()));

  async function next1() {
    if (!file) return;
    const bad = checkFormsFile(file);
    if (bad) return setError(bad);
    setBusy(true);
    setError(null);
    try {
      const id = upload || (await uploadFormsFile(course, file));
      setUpload(id);
      const r = await previewForms(course, { upload_id: id });
      setSheet(r);
      setIds({ email_col: r.suggested_mapping.email_col, code_col: r.suggested_mapping.code_col, time_col: r.suggested_mapping.time_col });
      setHeaderRow(r.suggested_mapping.header_row);
      setStep(2);
    } catch (e) {
      fail(e, "Chưa tải được tệp.");
    } finally {
      setBusy(false);
    }
  }

  async function preview() {
    setBusy(true);
    setError(null);
    try {
      const r = await previewForms(course, request());
      setOut(r.result ?? null);
      setStep(3);
    } catch (e) {
      fail(e, "Chưa xem trước được.");
    } finally {
      setBusy(false);
    }
  }

  async function commit() {
    setBusy(true);
    setError(null);
    commitKey.current ||= newIdempotencyKey();
    try {
      await importForms(course, request(), commitKey.current);
      await qc.invalidateQueries({ queryKey: gradebookKey(course) });
      onClose();
    } catch (e) {
      commitKey.current = "";
      fail(e, "Chưa ghi được điểm.");
    } finally {
      setBusy(false);
    }
  }

  const setRow = (i: number, patch: Partial<Row>) => setRows(rows.map((r, k) => (k === i ? { ...r, ...patch } : r)));
  const used = new Set([ids.email_col, ids.code_col, ids.time_col]);
  const primary =
    step === 1 ? (
      <Button variant="primary" disabled={!file || !item || busy} loading={busy} onClick={() => void next1()}>Tiếp tục</Button>
    ) : step === 2 ? (
      <Button variant="primary" disabled={!ready || busy} loading={busy} onClick={() => void preview()}>Xem trước</Button>
    ) : (
      <Button variant="primary" disabled={busy || !out || out.matched === 0} loading={busy} onClick={() => void commit()}>{`Ghi ${out?.matched ?? 0} điểm`}</Button>
    );

  return (
    <Dialog
      open
      onClose={onClose}
      title="Nhập kết quả từ Forms"
      error={error}
      footer={
        <>
          {step > 1 && <Button variant="ghost" disabled={busy} onClick={() => setStep((step - 1) as Step)}>Quay lại</Button>}
          <Button variant="ghost" onClick={onClose}>Đóng</Button>
          {primary}
        </>
      }
    >
      <div className={s.body} data-step={step} data-part="forms-import">
        {step === 1 && (
          <>
            <Field label="Nhập vào">{(id) => (
              <Select id={id} value={item} onChange={(e) => setItem(e.target.value)}>
                {items.map((c) => <option key={c.item_id} value={c.item_id}>{c.name}</option>)}
              </Select>
            )}</Field>
            <Field label="Tệp XLSX hoặc CSV">{(id) => (
              <input id={id} type="file" accept=".xlsx,.csv" data-part="forms-input" className={s.file} onChange={(e) => { setFile(e.target.files?.[0] ?? null); setUpload(""); setError(null); }} />
            )}</Field>
            {file && <p className={s.fileName}>{file.name}</p>}
          </>
        )}
        {step === 2 && sheet && (
          <>
            <Field label="Dòng tiêu đề">{(id) => (
              <Input id={id} type="number" min={1} max={100} value={headerRow} className={s.narrow} onChange={(e) => setHeaderRow(Math.max(1, Number(e.target.value) || 1))} />
            )}</Field>
            <Field label="Cột Email">{(id) => (
              <Select id={id} value={ids.email_col} onChange={(e) => setIds({ ...ids, email_col: e.target.value })}><option value="">Không dùng</option>{cols.map(colOption)}</Select>
            )}</Field>
            <Field label="Cột MSSV">{(id) => (
              <Select id={id} value={ids.code_col} onChange={(e) => setIds({ ...ids, code_col: e.target.value })}><option value="">Không dùng</option>{cols.map(colOption)}</Select>
            )}</Field>
            <Field label="Cột thời gian nộp">{(id) => (
              <Select id={id} value={ids.time_col} onChange={(e) => setIds({ ...ids, time_col: e.target.value })}><option value="">Không dùng</option>{cols.map(colOption)}</Select>
            )}</Field>
            <Field label="Đáp án đúng lấy từ">{(id) => (
              <Select id={id} value={source} onChange={(e) => setSource(e.target.value as Source)}>
                <option value="BANK">Ngân hàng câu hỏi</option>
                <option value="MANUAL">Tự nhập</option>
              </Select>
            )}</Field>
            {rows.map((r, i) => (
              <div key={i} className={s.pair} data-part="forms-question">
                <Select aria-label={`Cột câu ${i + 1}`} value={r.col} onChange={(e) => setRow(i, { col: e.target.value })}>
                  <option value="">Cột câu {i + 1}</option>
                  {cols.filter((c) => !used.has(c.col)).map(colOption)}
                </Select>
                {source === "BANK" ? (
                  <Select aria-label={`Câu hỏi ${i + 1}`} value={r.question_id} onChange={(e) => setRow(i, { question_id: e.target.value })}>
                    <option value="">Chọn câu</option>
                    {pickable.map((q) => <option key={q.id} value={q.id}>{q.title}</option>)}
                  </Select>
                ) : (
                  <Input aria-label={`Đáp án câu ${i + 1}`} value={r.accepted} placeholder="Đáp án; nhiều đáp án cách nhau bằng |" onChange={(e) => setRow(i, { accepted: e.target.value })} />
                )}
              </div>
            ))}
            <div className={s.pair}>
              <Button size="sm" variant="ghost" disabled={rows.length >= 100} onClick={() => setRows([...rows, { col: "", question_id: "", accepted: "" }])}>Thêm câu</Button>
              <Button size="sm" variant="ghost" onClick={() => setRows(cols.filter((c) => !used.has(c.col)).map((c) => ({ col: c.col, question_id: "", accepted: "" })))}>Lấy hết cột còn lại</Button>
            </div>
          </>
        )}
        {step === 3 && out && <FormsResultView out={out} />}
      </div>
    </Dialog>
  );
}

function FormsResultView({ out }: { out: FormsResult }) {
  const mismatch = out.items.reduce((n, r) => n + r.mismatch, 0);
  return (
    <>
      <p className={s.summary} data-part="forms-summary">{`${out.matched} khớp · ${out.unmatched} chưa khớp${out.duplicates > 0 ? ` · ${out.duplicates} dòng trùng` : ""}${mismatch > 0 ? ` · ${mismatch} ô không khớp lựa chọn` : ""}`}</p>
      {out.unmatched_rows.length > 0 && (
        <ul className={s.errors} data-part="forms-unmatched">
          {out.unmatched_rows.slice(0, 100).map((u) => <li key={u.row}>{`Dòng ${u.row} · ${u.identifier} · ${REASON[u.reason]}`}</li>)}
          {out.unmatched_rows.length > 100 && <li>{`…và ${out.unmatched_rows.length - 100} dòng nữa`}</li>}
        </ul>
      )}
      {out.items.length > 0 && (
        <div data-part="forms-sample">
          <DataTable
            dense
            caption="Điểm sẽ ghi"
            rows={out.items.slice(0, 50)}
            rowKey={(r) => r.student.id}
            columns={[
              { key: "st", header: "Sinh viên", render: (r) => `${r.student.full_name} · ${r.student.student_code}` },
              { key: "c", header: "Đúng", align: "end", width: "80px", render: (r) => `${r.correct}/${r.total}` },
              { key: "a", header: "Điểm", align: "end", width: "80px", render: (r) => vi(r.score) },
            ]}
          />
        </div>
      )}
    </>
  );
}
