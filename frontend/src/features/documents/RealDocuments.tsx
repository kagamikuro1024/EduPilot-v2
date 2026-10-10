"use client";

import { useQueryClient } from "@tanstack/react-query";
import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import { ApiError, apiClient, checkDocumentFile, newIdempotencyKey, uploadDocumentFile, useJob } from "@/shared/data";
import { useUndoLine } from "@/shared/lib/useUndoLine";
import {
  Button, ConfirmIrreversible, DataTable, Drawer, EmptyState, Field, InlineNotice, Input, OverflowMenu, Page, PageHeader, PageState, Panel, Select, Skeleton, StatusText, Switch, Textarea, type Column,
} from "@/shared/ui";
import { STATUS_LABEL, TYPE_LABEL, docsKey, useChunks, useDocList, useDocStats, type DocRow, type DocType } from "./documentsApi";
import r from "./RealDocuments.module.css";

const WHEN = new Intl.DateTimeFormat("vi-VN", { day: "2-digit", month: "2-digit", hour: "2-digit", minute: "2-digit" });
const MAX_PARALLEL = 3;
const TYPES = Object.keys(TYPE_LABEL) as DocType[];
const titleOf = (f: File) => f.name.replace(/\.[^.]+$/, "").slice(0, 200);

type Upload = { key: string; file: File; type: DocType; idem: string; state: "queued" | "uploading" | "processing" | "error"; jobId?: string; error?: string };

/** Quản lý tài liệu thật của Staff (DESIGN §14.14): vùng thả ngay trên bảng, dải thống kê, bảng sửa tại chỗ (cờ / loại), đoạn trong Drawer, xoá có xác nhận. */
export function RealDocuments({ courseId, canDelete }: { courseId: string; canDelete: boolean }) {
  const qc = useQueryClient();
  const [filter, setFilter] = useState({ type: "", status: "", q: "" });
  const list = useDocList(courseId, filter);
  const stats = useDocStats(courseId);
  const [uploads, setUploads] = useState<Upload[]>([]);
  const [rejects, setRejects] = useState<string[]>([]);
  const [over, setOver] = useState(false);
  const [pendingType, setPendingType] = useState<DocType>("LECTURE");
  const [patched, setPatched] = useState<Record<string, Partial<DocRow>>>({});
  const [failed, setFailed] = useState<string | null>(null);
  const [open, setOpen] = useState<DocRow | null>(null);
  const [del, setDel] = useState<DocRow | null>(null);
  const picker = useRef<HTMLInputElement>(null);
  const undo = useUndoLine();
  const refresh = useCallback(() => void qc.invalidateQueries({ queryKey: docsKey(courseId) }), [qc, courseId]);

  const rows = useMemo(() => list.items.map((d) => ({ ...d, ...patched[d.id] })), [list.items, patched]);

  // ---- tải lên: tối đa 3 tệp song song --------------------------------------------------------------------------------------------------------
  const set = (key: string, p: Partial<Upload>) => setUploads((u) => u.map((x) => (x.key === key ? { ...x, ...p } : x)));
  const run = useCallback(
    async (u: Upload) => {
      set(u.key, { state: "uploading", error: undefined });
      try {
        const id = await uploadDocumentFile(courseId, u.file);
        const res = await apiClient.post<{ job_id: string }>(`/courses/${courseId}/uploads/complete`, { upload_id: id, title: titleOf(u.file), type: u.type }, { idempotencyKey: u.idem });
        set(u.key, { state: "processing", jobId: res.data.job_id });
        refresh();
      } catch (e) {
        set(u.key, { state: "error", error: e instanceof ApiError ? e.userMessage : "Chưa tải được." });
      }
    },
    [courseId, refresh],
  );
  useEffect(() => {
    const active = uploads.filter((u) => u.state === "uploading").length;
    const next = uploads.filter((u) => u.state === "queued").slice(0, Math.max(0, MAX_PARALLEL - active));
    next.forEach((u) => void run(u));
  }, [uploads, run]);

  function add(files: FileList | File[], type: DocType) {
    const bad: string[] = [];
    const ok: Upload[] = [];
    for (const f of Array.from(files)) {
      const msg = checkDocumentFile(f);
      if (msg) bad.push(msg);
      else ok.push({ key: `${f.name}:${f.size}:${Math.random()}`, file: f, type, idem: newIdempotencyKey(), state: "queued" });
    }
    setRejects(bad);
    if (ok.length) setUploads((u) => [...u, ...ok]);
  }
  const pick = (type: DocType) => {
    setPendingType(type);
    picker.current?.click();
  };

  // ---- sửa tại chỗ: lạc quan + Hoàn tác ------------------------------------------------------------------------------------------------------
  async function patch(d: DocRow, change: Partial<Pick<DocRow, "use_for_rag" | "visible_to_students" | "type" | "week_no" | "title">>, label: string) {
    const before = Object.fromEntries(Object.keys(change).map((k) => [k, d[k as keyof DocRow]])) as Partial<DocRow>;
    setFailed(null);
    setPatched((p) => ({ ...p, [d.id]: { ...p[d.id], ...change } }));
    try {
      const res = await apiClient.patch<DocRow>(`/courses/${courseId}/documents/${d.id}`, { ...change, version: d.version });
      setPatched((p) => ({ ...p, [d.id]: { ...p[d.id], ...res.data } }));
      undo.push(`Đã đổi ${label}`, () => void patch({ ...d, ...res.data }, before as typeof change, label));
      refresh();
    } catch (e) {
      setPatched((p) => ({ ...p, [d.id]: { ...p[d.id], ...before } }));
      setFailed(e instanceof ApiError && e.code === "VERSION_CONFLICT" ? "Có người vừa đổi tài liệu này. Tải lại để xem bản mới." : "Chưa lưu được. Thử lại.");
    }
  }

  const action = async (d: DocRow, what: "retry" | "reindex") => {
    try {
      await apiClient.post(`/courses/${courseId}/documents/${d.id}/${what}`);
      refresh();
    } catch (e) {
      setFailed(e instanceof ApiError ? e.userMessage : "Chưa làm được. Thử lại.");
    }
  };

  const cols: Column<DocRow>[] = [
    {
      key: "title", header: "Tên", primary: true, width: "28%",
      render: (d) => (
        <div>
          <button type="button" className={r.name} onClick={() => setOpen(d)}>{d.title}</button>
          {d.shared_from && <div className={r.meta}>Chia sẻ từ {d.shared_from}</div>}
          {d.type === "ANSWER_KEY" && (
            <div className={r.keyNote}>
              <span>Không hiển thị cho sinh viên</span>
              <span>Không dùng cho AI của sinh viên</span>
            </div>
          )}
        </div>
      ),
    },
    {
      key: "type", header: "Loại", width: "13%",
      render: (d) =>
        d.shared_from ? TYPE_LABEL[d.type] : (
          <Select aria-label={`Loại của ${d.title}`} value={d.type} onChange={(e) => void patch(d, { type: e.target.value as DocType }, "loại")}>
            {TYPES.map((t) => <option key={t} value={t}>{TYPE_LABEL[t]}</option>)}
          </Select>
        ),
    },
    { key: "week", header: "Tuần", width: "8%", render: (d) => (d.week_no ? `Tuần ${d.week_no}` : "—") },
    {
      key: "ai", header: "Dùng cho AI", width: "11%",
      render: (d) => <Switch checked={d.use_for_rag} disabled={Boolean(d.shared_from)} onChange={(v) => void patch(d, { use_for_rag: v }, "cờ Dùng cho AI")} label={<span className="ep-sr-only">Dùng cho AI: {d.title}</span>} />,
    },
    {
      key: "vis", header: "Hiện cho sinh viên", width: "13%",
      render: (d) => <Switch checked={d.visible_to_students} disabled={Boolean(d.shared_from) || d.type === "ANSWER_KEY"} onChange={(v) => void patch(d, { visible_to_students: v }, "cờ Hiện cho sinh viên")} label={<span className="ep-sr-only">Hiện cho sinh viên: {d.title}</span>} />,
    },
    {
      key: "status", header: "Trạng thái", width: "12%",
      render: (d) => (
        <StatusText tone={d.status === "READY" ? "green" : d.status === "FAILED" ? "red" : "amber"}>
          {STATUS_LABEL[d.status]}{d.status === "FAILED" && d.error ? ` — ${d.error}` : ""}
        </StatusText>
      ),
    },
    { key: "updated", header: "Cập nhật", width: "11%", render: (d) => WHEN.format(new Date(d.updated_at)) },
    {
      key: "more", header: <span className="ep-sr-only">Thao tác</span>, width: "4%", align: "end",
      render: (d) => {
        const items = [
          ...(d.status === "FAILED" && !d.shared_from ? [{ label: "Thử lại", onSelect: () => void action(d, "retry") }] : []),
          ...(d.status === "READY" && !d.shared_from ? [{ label: "Lập chỉ mục lại", onSelect: () => void action(d, "reindex") }] : []),
          ...(canDelete && !d.shared_from ? [{ label: "Xoá", danger: true, onSelect: () => setDel(d) }] : []),
        ];
        return items.length ? <OverflowMenu items={items} label={`Thao tác với ${d.title}`} /> : null;
      },
    },
  ];

  const strip = stats.data
    ? [`${stats.data.total} tài liệu`, (stats.data.by_status.QUEUED ?? 0) + (stats.data.by_status.PROCESSING ?? 0) > 0 ? `${(stats.data.by_status.QUEUED ?? 0) + (stats.data.by_status.PROCESSING ?? 0)} đang xử lý` : "", (stats.data.by_status.FAILED ?? 0) > 0 ? `${stats.data.by_status.FAILED} lỗi` : ""].filter(Boolean).join(" · ")
    : "";
  const filtered = filter.type !== "" || filter.status !== "" || filter.q !== "";

  return (
    <Page width="full">
      <PageHeader title="Tài liệu" actions={<Button variant="primary" onClick={() => pick("LECTURE")}>Tải tài liệu lên</Button>} />
      <input ref={picker} type="file" hidden multiple accept=".pdf,.docx,.pptx" data-part="file-input" onChange={(e) => { if (e.target.files) add(e.target.files, pendingType); e.target.value = ""; }} />
      <Panel>
        <div className={r.stack}>
          {stats.data && !stats.data.has_course_policy && (
            <InlineNotice tone="info" compact action={<Button size="sm" onClick={() => pick("COURSE_POLICY")}>Tải quy chế môn học</Button>}>Lớp chưa có quy chế môn học.</InlineNotice>
          )}
          {strip && <p className={r.strip} data-part="doc-stats">{strip}</p>}
          <div
            className={[r.drop, over ? r.dropOver : ""].join(" ")}
            role="button"
            tabIndex={0}
            data-part="dropzone"
            onClick={() => pick("LECTURE")}
            onKeyDown={(e) => { if (e.key === "Enter" || e.key === " ") { e.preventDefault(); pick("LECTURE"); } }}
            onDragOver={(e) => { e.preventDefault(); setOver(true); }}
            onDragLeave={() => setOver(false)}
            onDrop={(e) => { e.preventDefault(); setOver(false); add(e.dataTransfer.files, "LECTURE"); }}
          >
            <span>Kéo thả hoặc chọn tệp</span>
          </div>
          {rejects.map((m) => <InlineNotice key={m} tone="danger" compact>{m}</InlineNotice>)}
          {uploads.length > 0 && (
            <ul className={r.uploads} data-part="uploads">
              {uploads.map((u) => (
                <UploadRow key={u.key} u={u} onRetry={() => { set(u.key, { state: "queued" }); }} onDone={() => { setUploads((x) => x.filter((y) => y.key !== u.key)); refresh(); }} />
              ))}
            </ul>
          )}
          <div className={r.filters} role="search">
            <Field label="Tìm theo tên">{(id) => <Input id={id} value={filter.q} onChange={(e) => setFilter({ ...filter, q: e.target.value })} placeholder="Ví dụ: quy chế" />}</Field>
            <Field label="Loại">
              {(id) => (
                <Select id={id} value={filter.type} onChange={(e) => setFilter({ ...filter, type: e.target.value })}>
                  <option value="">Mọi loại</option>
                  {TYPES.map((t) => <option key={t} value={t}>{TYPE_LABEL[t]}</option>)}
                </Select>
              )}
            </Field>
            <Field label="Trạng thái">
              {(id) => (
                <Select id={id} value={filter.status} onChange={(e) => setFilter({ ...filter, status: e.target.value })}>
                  <option value="">Mọi trạng thái</option>
                  <option value="READY">Sẵn sàng</option>
                  <option value="PROCESSING">Đang xử lý</option>
                  <option value="FAILED">Lỗi</option>
                </Select>
              )}
            </Field>
          </div>
          {failed && <InlineNotice tone="danger" compact>{failed}</InlineNotice>}
          {undo.node}
          <PageState
            query={list}
            isEmpty={() => rows.length === 0}
            loading={<Skeleton lines={5} />}
            empty={<EmptyState title={filtered ? "Không có tài liệu nào khớp." : "Lớp chưa có tài liệu."}>{!filtered && <Button variant="primary" onClick={() => pick("LECTURE")}>Tải tài liệu lên</Button>}</EmptyState>}
          >
            <DataTable columns={cols} rows={rows} rowKey={(d) => d.id} caption="Tài liệu của lớp" rowAttrs={(d) => ({ "data-part": "doc-row", "data-doc-id": d.id })} />
            {list.hasNextPage && <Button onClick={() => void list.fetchNextPage()} disabled={list.isFetchingNextPage}>Xem thêm</Button>}
          </PageState>
        </div>
      </Panel>
      <Drawer open={open !== null} onClose={() => setOpen(null)} title={open?.title ?? ""}>
        {open && <ChunkList courseId={courseId} doc={open} readOnly={Boolean(open.shared_from)} onSaved={refresh} />}
      </Drawer>
      <DeleteDoc courseId={courseId} doc={del} onClose={() => setDel(null)} onDone={() => { setDel(null); refresh(); }} />
    </Page>
  );
}

function UploadRow({ u, onRetry, onDone }: { u: Upload; onRetry: () => void; onDone: () => void }) {
  const job = useJob(u.jobId);
  useEffect(() => {
    if (u.state === "processing" && (job.status === "SUCCEEDED" || job.status === "FAILED")) onDone();
  }, [u.state, job.status, onDone]);
  const label = u.state === "error" ? "Lỗi" : u.state === "processing" ? `Đang xử lý ${job.progress} %` : u.state === "uploading" ? "Đang tải" : "Chờ tải";
  return (
    <li className={r.upload} data-part="upload-row" data-state={u.state}>
      <span className={r.name}>{u.file.name}</span>
      <span className={r.actions}>
        <StatusText tone={u.state === "error" ? "red" : "amber"}>{label}</StatusText>
        {u.state === "error" && <span className={r.meta}>{u.error}</span>}
        {u.state === "error" && <Button size="sm" onClick={onRetry}>Tải lại</Button>}
      </span>
    </li>
  );
}

function ChunkList({ courseId, doc, readOnly, onSaved }: { courseId: string; doc: DocRow; readOnly: boolean; onSaved: () => void }) {
  const q = useChunks(courseId, doc.id);
  const [edit, setEdit] = useState<{ id: string; text: string } | null>(null);
  const [err, setErr] = useState<string | null>(null);
  const [busy, setBusy] = useState(false);
  async function save() {
    if (!edit) return;
    setBusy(true);
    setErr(null);
    try {
      await apiClient.patch(`/courses/${courseId}/documents/${doc.id}/chunks/${edit.id}`, { text: edit.text });
      setEdit(null);
      await q.refetch();
      onSaved();
    } catch (e) {
      setErr(e instanceof ApiError && e.status === 422 ? "Đoạn dài 1–4.000 ký tự." : "Chưa lưu được. Thử lại.");
    } finally {
      setBusy(false);
    }
  }
  return (
    <PageState query={q} isEmpty={() => q.items.length === 0} loading={<Skeleton lines={4} />} empty={<EmptyState title="Tài liệu chưa có đoạn nào." />}>
      <div data-part="chunks">
        {q.items.map((c) => (
          <div key={c.id} className={r.chunk}>
            <div className={r.chunkHead}><span>Đoạn {c.ord + 1}</span>{c.page_no !== null && <span>Trang {c.page_no}</span>}</div>
            {edit?.id === c.id ? (
              <>
                <Field label="Nội dung đoạn">{(id) => <Textarea id={id} rows={6} value={edit.text} onChange={(e) => setEdit({ id: c.id, text: e.target.value })} />}</Field>
                {err && <InlineNotice tone="danger" compact>{err}</InlineNotice>}
                <div className={r.actions}>
                  <Button variant="primary" disabled={busy || !edit.text.trim() || edit.text.length > 4000} onClick={() => void save()}>Lưu đoạn</Button>
                  <Button variant="ghost" onClick={() => { setEdit(null); setErr(null); }}>Huỷ</Button>
                </div>
              </>
            ) : (
              <>
                <p className={r.chunkText}>{c.text}</p>
                {!readOnly && <div className={r.actions}><Button size="sm" variant="ghost" onClick={() => setEdit({ id: c.id, text: c.text })}>Sửa đoạn</Button></div>}
              </>
            )}
          </div>
        ))}
        {q.hasNextPage && <Button onClick={() => void q.fetchNextPage()} disabled={q.isFetchingNextPage}>Xem thêm</Button>}
      </div>
    </PageState>
  );
}

function DeleteDoc({ courseId, doc, onClose, onDone }: { courseId: string; doc: DocRow | null; onClose: () => void; onDone: () => void }) {
  const [impact, setImpact] = useState<{ chunks: number; courses: number } | null>(null);
  const [busy, setBusy] = useState(false);
  const [err, setErr] = useState<string | null>(null);
  useEffect(() => {
    setImpact(null); // eslint-disable-line react-hooks/set-state-in-effect -- đổi tài liệu ⇒ tính lại số liệu
    setErr(null);
    if (!doc) return;
    const c = new AbortController();
    apiClient.get<{ chunks: number; courses: number }>(`/courses/${courseId}/documents/${doc.id}/impact`, { signal: c.signal }).then((x) => setImpact(x.data), () => undefined);
    return () => c.abort();
  }, [doc, courseId]);
  return (
    <ConfirmIrreversible
      open={doc !== null}
      onClose={onClose}
      title={doc ? `Xoá ${doc.title}?` : ""}
      consequence={impact ? `${impact.chunks} đoạn và ${impact.courses} lớp đang dùng tài liệu này sẽ mất nó.` : "Tài liệu và các đoạn của nó sẽ mất."}
      confirmLabel="Xoá"
      loading={busy}
      error={err}
      onConfirm={() => {
        if (!doc) return;
        setBusy(true);
        setErr(null);
        apiClient.delete(`/courses/${courseId}/documents/${doc.id}`).then(onDone, (e) => setErr(e instanceof ApiError ? e.userMessage : "Chưa xoá được. Thử lại.")).finally(() => setBusy(false));
      }}
    />
  );
}
