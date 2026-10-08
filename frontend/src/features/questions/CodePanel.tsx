"use client";

import { useQueryClient } from "@tanstack/react-query";
import { FileUp, Play, Plus, Sparkles, Trash2 } from "lucide-react";
import { useState } from "react";
import { ApiError, ApiErrorNotice, apiClient, fieldErrors, useCursorList, useJob } from "@/shared/data";
import { Button, Checkbox, DataTable, Field, InlineNotice, Input, Select, StatusText, Textarea, type Column } from "@/shared/ui";
import { qKey, qPath, VERDICT_LABEL, type CodeDetail, type QuestionDetail, type Testcase, type VerifyResult } from "./questionsApi";
import s from "./Questions.module.css";

const LANGS: Array<{ id: "c11" | "cpp17"; label: string }> = [{ id: "c11", label: "C (C11)" }, { id: "cpp17", label: "C++ (C++17)" }];

type Props = { courseId: string; q: QuestionDetail; canEditTests: boolean; locked: boolean; onChanged: (q?: QuestionDetail) => void };

/** Bài lập trình: giới hạn, mã khởi tạo, lời giải mẫu, bộ test (mẫu / ẩn), nhập zip, chạy lời giải mẫu, gợi ý đầu vào từ AI. */
export function CodePanel(props: Props) {
  const { courseId, q, locked } = props;
  const code = q.code as CodeDetail;
  return (
    <div className={s.codePanel}>
      {locked && <InlineNotice tone="warning" title={`Đang dùng trong bài thi ${q.used_in_exams.map((e) => e.title).join(", ")} — nhân bản để sửa`}>Cấu hình và test của câu này đang bị khoá.</InlineNotice>}
      <ConfigForm {...props} code={code} />
      <Tests {...props} code={code} />
      <Verify courseId={courseId} q={q} code={code} onChanged={props.onChanged} />
    </div>
  );
}

function ConfigForm({ courseId, q, code, locked, onChanged }: Props & { code: CodeDetail }) {
  const [langs, setLangs] = useState(code.languages);
  const [time, setTime] = useState(String(code.time_limit_ms));
  const [mem, setMem] = useState(String(code.memory_limit_mb));
  const [out, setOut] = useState(String(code.output_limit_kb));
  const [checker, setChecker] = useState(code.checker);
  const [eps, setEps] = useState(code.float_eps ?? "");
  const [starter, setStarter] = useState(code.starter_code);
  const [refLang, setRefLang] = useState(code.reference?.language ?? code.languages[0] ?? "cpp17");
  const [refSrc, setRefSrc] = useState(code.reference?.source ?? "");
  const [err, setErr] = useState<unknown>(null);
  const [pending, setPending] = useState(false);
  const fe = fieldErrors(err);

  async function save(e: React.FormEvent) {
    e.preventDefault();
    setPending(true);
    setErr(null);
    try {
      const r = await apiClient.put<QuestionDetail>(`${qPath(courseId)}/${q.id}/code`, {
        languages: langs,
        time_limit_ms: Number(time),
        memory_limit_mb: Number(mem),
        output_limit_kb: Number(out),
        checker,
        float_eps: checker === "FLOAT_EPS" ? eps : null,
        starter_code: Object.fromEntries(Object.entries(starter).filter(([l, v]) => langs.includes(l as "c11") && v !== "")),
        reference: refSrc.trim() === "" ? null : { language: refLang, source: refSrc },
        version: q.version,
      });
      onChanged(r.data);
    } catch (x) {
      setErr(x);
    } finally {
      setPending(false);
    }
  }
  const toggle = (id: "c11" | "cpp17", on: boolean) => setLangs((p) => (on ? [...new Set([...p, id])] : p.filter((x) => x !== id)));

  return (
    <form className={s.section} onSubmit={save} noValidate aria-label="Cấu hình bài lập trình">
      <h3 className={s.h3}>Cấu hình</h3>
      <fieldset className={s.options} disabled={locked}>
        <legend className={s.legend}>Ngôn ngữ cho phép</legend>
        <div className={s.formRow}>{LANGS.map((l) => <Checkbox key={l.id} label={l.label} checked={langs.includes(l.id)} onChange={(e) => toggle(l.id, e.target.checked)} />)}</div>
        {fe.languages && <p role="alert" className={s.fieldNote}>Chọn ít nhất một ngôn ngữ.</p>}
      </fieldset>
      <div className={s.formRow}>
        <Field label="Thời gian (ms)" error={fe.time_limit_ms} helper="100–10.000">{(id, by) => <Input id={id} aria-describedby={by} inputMode="numeric" disabled={locked} value={time} onChange={(e) => setTime(e.target.value)} invalid={Boolean(fe.time_limit_ms)} />}</Field>
        <Field label="Bộ nhớ (MiB)" error={fe.memory_limit_mb} helper="16–512">{(id, by) => <Input id={id} aria-describedby={by} inputMode="numeric" disabled={locked} value={mem} onChange={(e) => setMem(e.target.value)} invalid={Boolean(fe.memory_limit_mb)} />}</Field>
        <Field label="Đầu ra (KiB)" error={fe.output_limit_kb} helper="1–16.384">{(id, by) => <Input id={id} aria-describedby={by} inputMode="numeric" disabled={locked} value={out} onChange={(e) => setOut(e.target.value)} invalid={Boolean(fe.output_limit_kb)} />}</Field>
      </div>
      <div className={s.formRow}>
        <Field label="Cách so khớp đầu ra">
          {(id) => (
            <Select id={id} disabled={locked} value={checker} onChange={(e) => setChecker(e.target.value as CodeDetail["checker"])}>
              <option value="EXACT">Khớp từng dòng (bỏ khoảng trắng cuối dòng)</option>
              <option value="TOKENS">Khớp từng từ</option>
              <option value="FLOAT_EPS">Số thực có sai số</option>
            </Select>
          )}
        </Field>
        {checker === "FLOAT_EPS" && (
          <Field label="Sai số" error={fe.float_eps} helper="Lớn hơn 0, tối đa 0,1">{(id, by) => <Input id={id} aria-describedby={by} inputMode="decimal" disabled={locked} value={eps} onChange={(e) => setEps(e.target.value)} invalid={Boolean(fe.float_eps)} />}</Field>
        )}
      </div>
      {langs.map((l) => (
        <Field key={l} label={`Mã khởi tạo — ${l === "c11" ? "C" : "C++"}`} helper="Sinh viên thấy sẵn trong ô soạn mã (tối đa 16 KiB).">
          {(id) => <Textarea id={id} rows={4} disabled={locked} value={starter[l] ?? ""} onChange={(e) => setStarter((p) => ({ ...p, [l]: e.target.value }))} />}
        </Field>
      ))}
      <Field label="Lời giải mẫu" helper="Chỉ giảng viên / TA thấy. Dùng để kiểm bộ test và sinh đầu ra mong đợi." error={fe["reference.source"]}>
        {(id, by) => (
          <div className={s.refBox}>
            <Select aria-label="Ngôn ngữ lời giải mẫu" disabled={locked} value={refLang} onChange={(e) => setRefLang(e.target.value)}>{langs.map((l) => <option key={l} value={l}>{l === "c11" ? "C" : "C++"}</option>)}</Select>
            <Textarea id={id} aria-describedby={by} rows={6} disabled={locked} value={refSrc} onChange={(e) => setRefSrc(e.target.value)} />
          </div>
        )}
      </Field>
      {err !== null && Object.keys(fe).length === 0 && <ApiErrorNotice error={err} showTechnical />}
      {!locked && <div className={s.formActions}><Button type="submit" variant="primary" loading={pending}>Lưu cấu hình</Button></div>}
    </form>
  );
}

function Tests({ courseId, q, code, canEditTests, locked, onChanged }: Props & { code: CodeDetail }) {
  const qc = useQueryClient();
  const key = qKey(courseId, q.id, "tests", code.tests_version);
  const list = useCursorList<Testcase>(key, `${qPath(courseId)}/${q.id}/testcases`, { limit: 100 });
  const [adding, setAdding] = useState(false);
  const [err, setErr] = useState<unknown>(null);
  const [zipMsg, setZipMsg] = useState<{ ok: boolean; text: string; errors: Array<{ file: string; code: string; message: string }> } | null>(null);
  const [selected, setSelected] = useState<Set<string>>(new Set());
  const [suggestJob, setSuggestJob] = useState<string | null>(null);
  const job = useJob(suggestJob);
  const refresh = async () => {
    await qc.invalidateQueries({ queryKey: qKey(courseId, q.id) });
    onChanged();
  };
  const readOnly = locked && !canEditTests;

  async function remove(t: Testcase) {
    setErr(null);
    try {
      await apiClient.delete(`${qPath(courseId)}/${q.id}/testcases/${t.id}`);
      await refresh();
    } catch (x) {
      setErr(x);
    }
  }
  async function approve() {
    setErr(null);
    try {
      await apiClient.post(`${qPath(courseId)}/${q.id}/testcases/approve`, { ids: [...selected] });
      setSelected(new Set());
      await refresh();
    } catch (x) {
      setErr(x);
    }
  }
  async function importZip(file: File, dry: boolean, mode: "append" | "replace") {
    setErr(null);
    setZipMsg(null);
    const fd = new FormData();
    fd.append("file", file);
    try {
      const r = await apiClient.post<{ would_create: number; created: number; errors: Array<{ file: string; code: string; message: string }> }>(`${qPath(courseId)}/${q.id}/testcases/import?dry_run=${dry}&mode=${mode}`, fd);
      if (r.data.errors.length > 0) setZipMsg({ ok: false, text: "Tệp zip có lỗi — chưa ghi gì.", errors: r.data.errors });
      else if (dry) setZipMsg({ ok: true, text: `Kiểm tra xong: sẽ thêm ${r.data.would_create} test.`, errors: [] });
      else {
        setZipMsg({ ok: true, text: `Đã thêm ${r.data.created} test.`, errors: [] });
        await refresh();
      }
    } catch (x) {
      if (x instanceof ApiError && x.status === 422 && Array.isArray(x.details)) {
        setZipMsg({ ok: false, text: "Tệp zip có lỗi — chưa ghi gì.", errors: (x.details as Array<{ field: string; code: string; message: string }>).map((d) => ({ file: d.field, code: d.code, message: d.message })) });
      } else setErr(x);
    }
  }
  async function suggest() {
    setErr(null);
    try {
      const r = await apiClient.post<{ job_id: string }>(`${qPath(courseId)}/suggest`, { kind: "CODE_TESTS", question_id: q.id, count: 5 });
      setSuggestJob(r.data.job_id);
    } catch (x) {
      setErr(x);
    }
  }
  if (job.status === "SUCCEEDED" && suggestJob) {
    queueMicrotask(() => {
      setSuggestJob(null);
      void refresh();
    });
  }

  const columns: Column<Testcase>[] = [
    { key: "name", header: "Test", render: (t) => <span>{t.name}</span> },
    { key: "kind", header: "Loại", width: "90px", render: (t) => (t.is_sample ? "Mẫu" : "Ẩn") },
    { key: "weight", header: "Trọng số", width: "100px", align: "end", render: (t) => t.weight },
    { key: "src", header: "Nguồn", width: "120px", render: (t) => (t.source === "AI_DRAFT" ? "AI gợi ý" : t.source === "IMPORT" ? "Nhập zip" : "Soạn tay") },
    {
      key: "status",
      header: "Trạng thái",
      width: "130px",
      render: (t) => (t.approved ? <StatusText tone="green">Đã duyệt</StatusText> : <StatusText tone="amber">Chờ duyệt</StatusText>),
    },
    {
      key: "act",
      header: <span className={s.srOnly}>Hành động</span>,
      width: "56px",
      align: "end",
      render: (t) => (readOnly ? null : <Button size="sm" variant="text" aria-label={`Xoá test ${t.name}`} icon={<Trash2 aria-hidden />} onClick={() => void remove(t)} />),
    },
  ];
  const pendingAi = list.items.filter((t) => !t.approved);

  return (
    <section className={s.section} aria-label="Bộ test">
      <h3 className={s.h3}>Bộ test · v{code.tests_version}</h3>
      <p className={s.meta}>{code.tests.samples} mẫu · {code.tests.hidden} ẩn · tổng trọng số {code.tests.total_weight}</p>
      {err !== null && <ApiErrorNotice error={err} showTechnical />}
      <DataTable
        caption="Bộ test của bài lập trình"
        columns={columns}
        rows={list.items}
        rowKey={(t) => t.id}
        mobile="scroll"
        selection={pendingAi.length > 0 ? { selected, onChange: setSelected } : undefined}
        empty={<InlineNotice compact>Chưa có test nào. Thêm test, nhập từ zip hoặc nhờ AI gợi ý đầu vào.</InlineNotice>}
      />
      {!readOnly && (
        <div className={s.formActions}>
          <Button size="sm" icon={<Plus aria-hidden />} onClick={() => setAdding((a) => !a)}>Thêm test</Button>
          <ZipPicker onPick={importZip} />
          <Button size="sm" icon={<Sparkles aria-hidden />} onClick={() => void suggest()} disabled={!code.reference || suggestJob !== null}>Gợi ý đầu vào từ AI</Button>
          {pendingAi.length > 0 && <Button size="sm" variant="primary" disabled={selected.size === 0} onClick={() => void approve()}>Duyệt {selected.size > 0 ? `${selected.size} test đã chọn` : "test AI đã chọn"}</Button>}
        </div>
      )}
      {suggestJob && <p className={s.meta} role="status">AI đang đề xuất đầu vào… {job.progress}% — đầu ra do lời giải mẫu sinh, test chỉ được chấm sau khi bạn duyệt.</p>}
      {job.status === "FAILED" && <InlineNotice tone="danger" title={job.error ?? "Chưa gợi ý được."} />}
      {adding && !readOnly && <AddTest courseId={courseId} q={q} onDone={async () => { setAdding(false); await refresh(); }} />}
      {zipMsg && (
        <InlineNotice tone={zipMsg.ok ? "success" : "danger"} title={zipMsg.text}>
          {zipMsg.errors.length > 0 && <ul className={s.zipErrors}>{zipMsg.errors.map((e, i) => <li key={i}><code>{e.file}</code> — {e.message}</li>)}</ul>}
        </InlineNotice>
      )}
    </section>
  );
}

function ZipPicker({ onPick }: { onPick: (f: File, dry: boolean, mode: "append" | "replace") => void }) {
  const [file, setFile] = useState<File | null>(null);
  const [mode, setMode] = useState<"append" | "replace">("append");
  return (
    <span className={s.zip}>
      <label className={s.fileLabel}>
        <FileUp aria-hidden /> <span>{file ? file.name : "Chọn tệp zip"}</span>
        <input type="file" accept=".zip,application/zip" className={s.srOnly} aria-label="Tệp zip chứa test" onChange={(e) => setFile(e.target.files?.[0] ?? null)} />
      </label>
      <Select aria-label="Cách nhập" value={mode} onChange={(e) => setMode(e.target.value as "append" | "replace")}>
        <option value="append">Thêm vào cuối</option>
        <option value="replace">Thay toàn bộ</option>
      </Select>
      <Button size="sm" disabled={!file} onClick={() => file && onPick(file, true, mode)}>Kiểm tra zip</Button>
      <Button size="sm" disabled={!file} onClick={() => file && onPick(file, false, mode)}>Nhập zip</Button>
    </span>
  );
}

function AddTest({ courseId, q, onDone }: { courseId: string; q: QuestionDetail; onDone: () => void }) {
  const [name, setName] = useState("");
  const [input, setInput] = useState("");
  const [expected, setExpected] = useState("");
  const [sample, setSample] = useState(false);
  const [weight, setWeight] = useState("1");
  const [err, setErr] = useState<unknown>(null);
  const fe = fieldErrors(err);
  async function save(e: React.FormEvent) {
    e.preventDefault();
    setErr(null);
    try {
      await apiClient.post(`${qPath(courseId)}/${q.id}/testcases`, { name, input, expected, is_sample: sample, weight: Number(weight) });
      onDone();
    } catch (x) {
      setErr(x);
    }
  }
  return (
    <form className={s.addTest} onSubmit={save} noValidate aria-label="Thêm test">
      <div className={s.formRow}>
        <Field label="Tên test" required error={fe.name}>{(id, by) => <Input id={id} aria-describedby={by} value={name} maxLength={60} onChange={(e) => setName(e.target.value)} invalid={Boolean(fe.name)} />}</Field>
        <Field label="Trọng số" error={fe.weight}>{(id, by) => <Input id={id} aria-describedby={by} inputMode="numeric" value={weight} onChange={(e) => setWeight(e.target.value)} invalid={Boolean(fe.weight)} />}</Field>
        <Checkbox label="Test mẫu (sinh viên thấy)" checked={sample} onChange={(e) => setSample(e.target.checked)} />
      </div>
      <Field label="Đầu vào" error={fe.input}>{(id, by) => <Textarea id={id} aria-describedby={by} rows={3} value={input} onChange={(e) => setInput(e.target.value)} />}</Field>
      <Field label="Đầu ra mong đợi" error={fe.expected}>{(id, by) => <Textarea id={id} aria-describedby={by} rows={3} value={expected} onChange={(e) => setExpected(e.target.value)} />}</Field>
      {err !== null && Object.keys(fe).length === 0 && <ApiErrorNotice error={err} showTechnical />}
      <Button type="submit" variant="primary" size="sm">Thêm test</Button>
    </form>
  );
}

function Verify({ courseId, q, code, onChanged }: { courseId: string; q: QuestionDetail; code: CodeDetail; onChanged: (q?: QuestionDetail) => void }) {
  const qc = useQueryClient();
  const [jobId, setJobId] = useState<string | null>(null);
  const [err, setErr] = useState<unknown>(null);
  const job = useJob(jobId);
  const res = job.status === "SUCCEEDED" ? (job.result as VerifyResult | undefined) : undefined;
  const verified = code.reference_verified_version === code.tests_version;
  const [done, setDone] = useState<string | null>(null);
  if (job.status === "SUCCEEDED" && jobId && done !== jobId) {
    setDone(jobId);
    void qc.invalidateQueries({ queryKey: qKey(courseId, q.id) }).then(() => onChanged());
  }
  async function run() {
    setErr(null);
    setJobId(null);
    try {
      const r = await apiClient.post<{ job_id: string }>(`${qPath(courseId)}/${q.id}/reference/verify`, {});
      setJobId(r.data.job_id);
    } catch (x) {
      setErr(x);
    }
  }
  const running = jobId !== null && job.status !== "SUCCEEDED" && job.status !== "FAILED";
  return (
    <section className={s.section} aria-label="Lời giải mẫu">
      <h3 className={s.h3}>Chạy lời giải mẫu</h3>
      <p className={s.meta}>
        {verified ? <StatusText tone="green">Đã kiểm với bộ test v{code.tests_version}</StatusText> : <StatusText tone="amber">Chưa kiểm với bộ test v{code.tests_version}</StatusText>}
      </p>
      <div className={s.formActions}>
        <Button icon={<Play aria-hidden />} onClick={() => void run()} loading={running} disabled={!code.reference || code.tests.total === 0}>Chạy lời giải mẫu</Button>
        {!code.reference && <span className={s.meta}>Nhập lời giải mẫu và lưu cấu hình trước.</span>}
      </div>
      {running && <p className={s.meta} role="status">Đang chạy… {job.progress}%</p>}
      {err !== null && <ApiErrorNotice error={err} showTechnical />}
      {job.status === "FAILED" && <InlineNotice tone="danger" title={job.error ?? "Chưa chạy được lời giải mẫu, thử lại sau."} />}
      {res && (
        <div>
          <InlineNotice tone={res.ok ? "success" : "warning"} title={res.ok ? "Mọi test đều đúng — lời giải mẫu đã được kiểm." : res.message || (res.compile_ok ? "Có test lệch so với đầu ra mong đợi." : "Lời giải mẫu không biên dịch được.")} />
          <DataTable
            caption="Kết quả từng test"
            columns={[
              { key: "n", header: "Test", render: (t: VerifyResult["per_test"][number]) => t.name },
              { key: "v", header: "Kết quả", width: "180px", render: (t: VerifyResult["per_test"][number]) => VERDICT_LABEL[t.verdict] ?? t.verdict },
              { key: "t", header: "Thời gian", width: "110px", align: "end" as const, render: (t: VerifyResult["per_test"][number]) => `${t.time_ms} ms` },
            ]}
            rows={res.per_test}
            rowKey={(t) => t.test_id}
            mobile="scroll"
          />
        </div>
      )}
    </section>
  );
}
