"use client";

import { useQueryClient, type InfiniteData } from "@tanstack/react-query";
import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import { ApiError } from "@/shared/data";
import { GradeCalculation } from "@/shared/domain";
import { Button, ButtonLink, type Column as TableColumn, DataTable, Drawer, EmptyState, InlineNotice, OverflowMenu, type HeaderAction, Page, PageHeader, PageState, Panel, Skeleton, StatusText } from "@/shared/ui";
import { useSession } from "@/shared/session/session";
import {
  exportXlsx, gradebookKey, publishItem, putEntry, useExplain, useGradebook, useHistory, useSchemeStatus, useSnapshots,
  type Column, type GEntry, type GradebookPage, type GRow, type Incomplete,
} from "./gradeApi";
import { FinalizeDialog, SnapshotHistory, UnlockDialog, stamp } from "./FinalizeGrades";
import { FormsImportDialog } from "./FormsImportDialog";
import { ImportDialog } from "./ImportDialog";
import f from "./FinalizeGrades.module.css";
import s from "./Gradebook.module.css";
import { formatWhen } from "@/shared/lib/datetime";

const vi = (v: string | undefined) => (v === undefined || v === "" ? "" : v.replace(".", ","));
const SCORE_RE = /^\d{1,3}([.,]\d{1,2})?$/;
const TOO_MANY = /^\d{1,3}[.,]\d{3,}$/;
const RETRY_MS = 3000;
const OFFICIAL_NOTE = "Điểm chính thức nằm ở hệ thống quản lý đào tạo của trường.";

type CellState = { kind: "offline" | "error"; message?: string };
type Conflict = { rowId: string; col: string; name: string; mine: string; theirs: { score: string; version: number; updated_by?: { name: string }; updated_at?: string }; at: { top: number; left: number } };

/** Trạng thái dòng: chỉ nói điều cần chú ý. "Đủ điểm" để trống (US-UX91-16 AC4). */
const statusText = (r: GRow) => {
  if (r.locked) return `Đã chốt v${r.locked.version}`;
  if (!r.result) return "";
  if (r.result.status === "INCOMPLETE") return "Thiếu điểm";
  if (r.result.status === "INVALID") return "Chưa tính được";
  return r.result.verdict.startsWith("LIET") ? "Liệt" : "";
};

/** Hiển thị điểm KHÔNG làm tròn: đệm số 0 bên phải tới số chữ số lẻ lớn nhất của cột (tối thiểu 2), dấu phẩy (US-UX91-16 AC1). */
const decimalsOf = (values: Array<string | undefined>) => Math.max(2, ...values.map((v) => (v && v.includes(".") ? v.split(".")[1].length : 0)));
const padScore = (v: string | undefined, d: number) => {
  if (!v) return "—";
  const [i, f = ""] = v.split(".");
  return `${i},${f.padEnd(d, "0")}`;
};

/** Chiều cao vùng cuộn của bảng = khung nhìn còn lại sau phần đầu trang (một vùng cuộn, US-UX91-16 AC3). */
function useTableHeight(offset: number) {
  const [h, setH] = useState(560);
  useEffect(() => {
    const on = () => setH(Math.max(320, window.innerHeight - offset));
    on();
    window.addEventListener("resize", on);
    return () => window.removeEventListener("resize", on);
  }, [offset]);
  return h;
}

const ago = (iso?: string) => {
  if (!iso) return "";
  const m = Math.max(0, Math.round((Date.now() - new Date(iso).getTime()) / 60000));
  return m < 1 ? "vừa xong" : m < 60 ? `${m} phút trước` : `${Math.round(m / 60)} giờ trước`;
};

/** Sổ điểm: chỉ có thật, Giảng viên / TA của lớp (TA chỉ đọc). */
export function Gradebook() {
  const { role, realCourseId, realPending } = useSession();
  if (realPending || !realCourseId || realCourseId === "all" || (role !== "teacher" && role !== "ta")) {
    return (
      <Page kind="work">
        <PageHeader title="Sổ điểm" />
        <Panel>{realPending ? <Skeleton lines={10} /> : <EmptyState title="Chọn một lớp để xem" />}</Panel>
      </Page>
    );
  }
  return <RealGradebook courseId={realCourseId} teacher={role === "teacher"} />;
}

function RealGradebook({ courseId, teacher }: { courseId: string; teacher: boolean }) {
  const qc = useQueryClient();
  const q = useGradebook(courseId);
  const status = useSchemeStatus(courseId).data;
  const [drafts, setDrafts] = useState<ReadonlyMap<string, string>>(new Map());
  const [cells, setCells] = useState<Record<string, CellState>>({});
  const [pending, setPending] = useState<ReadonlySet<string>>(new Set());
  const [conflict, setConflict] = useState<Conflict | null>(null);
  const [explainOf, setExplainOf] = useState<GRow | null>(null);
  const [historyOf, setHistoryOf] = useState<GRow | null>(null);
  const [warnOpen, setWarnOpen] = useState(false);
  const [onlyMissing, setOnlyMissing] = useState(false);
  const [savedCells, setSavedCells] = useState<ReadonlySet<string>>(new Set());
  const savedTimer = useRef<ReturnType<typeof setTimeout> | undefined>(undefined);
  const flashCell = useCallback((key: string) => {
    setSavedCells(new Set([key]));
    clearTimeout(savedTimer.current);
    savedTimer.current = setTimeout(() => setSavedCells(new Set()), 400);
  }, []);
  const [importing, setImporting] = useState(false);
  const [formsImporting, setFormsImporting] = useState(false);
  const [finalizing, setFinalizing] = useState(false);
  const [unlocking, setUnlocking] = useState(false);
  const [history, setHistory] = useState(false);
  const [incomplete, setIncomplete] = useState<Incomplete | null>(null);
  const [focusReq, setFocusReq] = useState<{ rowKey: string; col: string; n: number } | undefined>();
  const latest = useRef(new Map<string, string>());
  const busy = useRef(new Map<string, string>()); // khoá ô → chữ đang gửi

  const first = q.data?.pages[0];
  const columns = useMemo<Column[]>(() => first?.columns ?? [], [first]);
  const rows = useMemo(() => {
    const seen = new Set<string>();
    const out: GRow[] = [];
    for (const p of q.data?.pages ?? []) for (const r of p.rows) if (!seen.has(r.student.id)) { seen.add(r.student.id); out.push(r); }
    return out;
  }, [q.data]);
  const locked = first?.state.status === "LOCKED";
  const snaps = useSnapshots(courseId, locked).data;
  // danh sách thiếu điểm cần tên sinh viên: nạp hết các trang (≤ 10 trang cho 1.000 sinh viên)
  useEffect(() => {
    if (incomplete && q.hasNextPage && !q.isFetchingNextPage) void q.fetchNextPage();
  }, [incomplete, q]);
  const canEdit = teacher && !locked;

  const patchRow = useCallback(
    (id: string, fn: (r: GRow) => GRow) =>
      qc.setQueryData<InfiniteData<GradebookPage>>(gradebookKey(courseId), (old) => (old ? { ...old, pages: old.pages.map((p) => ({ ...p, rows: p.rows.map((r) => (r.student.id === id ? fn(r) : r)) })) } : old)),
    [qc, courseId],
  );
  const setDraft = useCallback((key: string, text: string | null) => {
    setDrafts((d) => {
      const n = new Map(d);
      if (text === null) n.delete(key);
      else n.set(key, text);
      return n;
    });
    if (text !== null) setCells((c) => { if (!(key in c)) return c; const n = { ...c }; delete n[key]; return n; });
  }, []);
  const mark = useCallback((id: string, on: boolean) => setPending((p) => { const n = new Set(p); if (on) n.add(id); else n.delete(id); return n; }), []);

  const send = useCallback(
    async (row: GRow, colKey: string, text: string, version: number | null) => {
      const key = `${row.student.id}:${colKey}`;
      const body = { item_id: colKey, student_id: row.student.id, score: text.trim() === "" ? null : text.trim().replace(",", "."), version };
      busy.current.set(key, text);
      mark(row.student.id, true);
      try {
        const out = await putEntry(courseId, body);
        patchRow(row.student.id, (r) => {
          const entries = { ...r.entries };
          if (out.entry) entries[colKey] = { score: out.entry.score, version: out.entry.version, source: out.entry.source };
          else delete entries[colKey];
          return { ...r, entries, result: out.row_result, bonus: out.bonus, penalty: out.penalty };
        });
        setCells((c) => { const n = { ...c }; delete n[key]; return n; });
        flashCell(key);
        setDrafts((d) => { if (d.get(key) !== text) return d; const n = new Map(d); n.delete(key); return n; }); // chữ gõ tiếp trong lúc chờ không bị xoá
        const again = latest.current.get(key);
        latest.current.delete(key);
        busy.current.delete(key);
        mark(row.student.id, false);
        if (again !== undefined) void send(row, colKey, again, out.entry?.version ?? null);
        return;
      } catch (e) {
        busy.current.delete(key);
        mark(row.student.id, false);
        if (e instanceof ApiError && e.code === "VERSION_CONFLICT") {
          const cur = (e.details as { current?: { score: string | null; version: number; updated_by?: { name: string }; updated_at?: string } } | undefined)?.current;
          const el = document.querySelector<HTMLElement>(`input[data-cell="${CSS.escape(key)}"]`)?.getBoundingClientRect();
          setDraft(key, text);
          setConflict({ rowId: row.student.id, col: colKey, name: row.student.full_name, mine: text, theirs: { score: cur?.score ?? "", version: cur?.version ?? 0, updated_by: cur?.updated_by, updated_at: cur?.updated_at }, at: { top: (el?.bottom ?? 120) + 4, left: el?.left ?? 120 } });
        } else if (e instanceof ApiError && (e.code === "SCORE_OUT_OF_RANGE" || e.code === "SCORE_PRECISION" || e.status === 422)) {
          setDraft(key, text);
          setCells((c) => ({ ...c, [key]: { kind: "error", message: e.code === "SCORE_PRECISION" ? "Điểm có quá 2 chữ số thập phân" : e.code === "SCORE_OUT_OF_RANGE" ? "Điểm ngoài thang" : "Điểm chưa hợp lệ" } }));
        } else if (e instanceof ApiError && e.code === "GRADEBOOK_LOCKED") {
          void q.refetch();
        } else {
          // mất mạng / lỗi tạm: giữ chữ đã gõ, xếp hàng, tự gửi lại
          setDraft(key, text);
          setCells((c) => ({ ...c, [key]: { kind: "offline" } }));
          const wait = Promise.withResolvers<void>();
          setTimeout(wait.resolve, RETRY_MS);
          await wait.promise;
          const nowRow = qc.getQueryData<InfiniteData<GradebookPage>>(gradebookKey(courseId))?.pages.flatMap((p) => p.rows).find((r) => r.student.id === row.student.id) ?? row;
          void send(nowRow, colKey, latest.current.get(key) ?? text, nowRow.entries[colKey]?.version ?? null);
          latest.current.delete(key);
        }
      }
    },
    [courseId, patchRow, setDraft, mark, qc, q, flashCell],
  );

  const commit = useCallback(
    (row: GRow, colKey: string, text: string) => {
      const key = `${row.student.id}:${colKey}`;
      const col = columns.find((c) => c.item_id === colKey);
      const cur = row.entries[colKey];
      if (vi(cur?.score) === text.trim() || (cur === undefined && text.trim() === "")) return setDraft(key, null);
      const t = text.trim();
      if (t !== "" && !SCORE_RE.test(t)) {
        setCells((c) => ({ ...c, [key]: { kind: "error", message: TOO_MANY.test(t) ? "Điểm có quá 2 chữ số thập phân" : "Nhập một số" } }));
        return;
      }
      if (t !== "" && col && Number(t.replace(",", ".")) > Number(col.max_score)) {
        setCells((c) => ({ ...c, [key]: { kind: "error", message: `Điểm ngoài thang 0–${vi(col.max_score)}` } }));
        return;
      }
      if (busy.current.has(key)) {
        if (busy.current.get(key) !== t) latest.current.set(key, t);
        return;
      }
      setCells((c) => { const n = { ...c }; delete n[key]; return n; });
      void send(row, colKey, t, cur?.version ?? null);
    },
    [columns, send, setDraft],
  );

  const resolve = useCallback(
    (keepMine: boolean) => {
      const c = conflict;
      if (!c) return;
      const row = rows.find((r) => r.student.id === c.rowId);
      setConflict(null);
      if (!row) return;
      const key = `${c.rowId}:${c.col}`;
      if (keepMine) void send(row, c.col, c.mine, c.theirs.version || null);
      else {
        patchRow(c.rowId, (r) => ({ ...r, entries: { ...r.entries, [c.col]: { score: c.theirs.score, version: c.theirs.version, source: "MANUAL" } } }));
        setDraft(key, null);
        void q.refetch();
      }
    },
    [conflict, rows, send, patchRow, setDraft, q],
  );

  async function publish(col: Column, on: boolean) {
    patchColumn(col.item_id, on);
    try {
      await publishItem(courseId, col.item_id, on);
    } catch {
      patchColumn(col.item_id, !on);
    }
  }
  function patchColumn(item: string, on: boolean) {
    qc.setQueryData<InfiniteData<GradebookPage>>(gradebookKey(courseId), (old) => (old ? { ...old, pages: old.pages.map((p, i) => (i === 0 ? { ...p, columns: p.columns.map((c) => (c.item_id === item ? { ...c, published: on } : c)) } : p)) } : old));
  }

  const dec = {
    items: Object.fromEntries(columns.map((c) => [c.item_id, decimalsOf(rows.map((r) => r.entries[c.item_id]?.score))])),
    bonus: decimalsOf(rows.map((r) => r.bonus)),
    penalty: decimalsOf(rows.map((r) => r.penalty)),
    qt: decimalsOf(rows.map((r) => r.result?.qt)),
    hp: decimalsOf(rows.map((r) => r.result?.hp)),
  };
  const missing = rows.filter((r) => statusText(r) === "Thiếu điểm");
  const shownRows = onlyMissing ? missing : rows;
  const showStatus = rows.some((r) => statusText(r) !== "" || r.drift);
  const tableHeight = useTableHeight(300);
  const hasPublished = columns.some((c) => c.source === "MANUAL" && c.published);

  const itemCols: TableColumn<GRow>[] = columns.map((c) => ({
    key: c.item_id,
    align: "end",
    width: "104px",
    part: `col-${c.key}`,
    header: (
      <span className={s.colHead} title={c.source === "ASSIGNMENT" ? "Điểm từ bài tập" : c.source === "EXAM" ? "Điểm từ bài thi" : undefined}>
        {c.source === "MANUAL" && c.published && <i className={s.pubDot} role="img" aria-label="Đã công bố" />}
        <span className={s.colName}>{c.name}</span>
        {teacher && c.source === "MANUAL" && !locked && (
          <span className={s.colMenu}>
            <OverflowMenu label={`Cột ${c.name}`} items={[c.published ? { label: "Gỡ công bố", onSelect: () => void publish(c, false) } : { label: "Công bố cho sinh viên", onSelect: () => void publish(c, true) }]} />
          </span>
        )}
      </span>
    ),
    cell: c.source === "MANUAL" ? {
      editable: () => canEdit,
      value: (r: GRow) => vi(r.entries[c.item_id]?.score),
      display: (r: GRow) => (r.entries[c.item_id]?.score ? padScore(r.entries[c.item_id]?.score, dec.items[c.item_id]) : ""),
      label: (r: GRow) => `${c.name} — ${r.student.full_name}`,
      invalid: (r: GRow) => cells[`${r.student.id}:${c.item_id}`]?.kind === "error",
      status: (r: GRow) => {
        const st = cells[`${r.student.id}:${c.item_id}`];
        return st?.kind === "offline" ? <span className={s.cellOff}>Chưa lưu</span> : st?.kind === "error" ? <span className={s.cellErr} role="alert">{st.message}</span> : null;
      },
    } : undefined,
    render: (r: GRow) => {
      const e: GEntry | undefined = r.entries[c.item_id];
      const shown = padScore(e?.score, dec.items[c.item_id]);
      if (c.source === "EXAM") return <span title="Điểm lấy từ bài thi, sửa ở bài thi">{shown}</span>;
      if (c.source === "ASSIGNMENT") return <span title="Điểm lấy từ bài tập, sửa ở màn chấm">{shown}</span>;
      return <span>{shown}</span>;
    },
  }));

  const calc = (r: GRow, v: string | undefined, d: number) => (pending.has(r.student.id) ? "…" : padScore(v, d));
  const tableCols: TableColumn<GRow>[] = [
    { key: "student", header: "Sinh viên", frozen: true, width: "220px", render: (r) => <span className={s.who}><span>{r.student.full_name}</span><span className={s.code}>{r.student.student_code}</span></span> },
    ...itemCols,
    { key: "bonus", header: "Điểm cộng", align: "end", width: "88px", render: (r) => calc(r, r.bonus, dec.bonus) },
    { key: "penalty", header: "Điểm trừ", align: "end", width: "88px", render: (r) => calc(r, r.penalty, dec.penalty) },
    { key: "qt", header: "Quá trình", align: "end", width: "92px", part: "col-qt", render: (r) => calc(r, r.result?.qt, dec.qt) },
    { key: "hp", header: "Điểm học phần", align: "end", width: "112px", part: "col-hp", render: (r) => calc(r, r.result?.hp, dec.hp) },
    { key: "letter", header: "Điểm chữ", align: "center", width: "88px", part: "col-letter", render: (r) => (pending.has(r.student.id) ? "…" : r.result?.letter ?? "—") },
    ...(showStatus ? [{ key: "status", header: "Trạng thái", width: "130px", part: "col-status", render: (r: GRow) => {
      const t = pending.has(r.student.id) ? "…" : statusText(r);
      return <span data-part="row-status">{t === "Thiếu điểm" && <i className={s.missDot} aria-hidden />}{t}{r.drift && <span className={s.drift} data-part="row-drift">Khác bản chốt</span>}</span>;
    } }] : []),
    {
      key: "menu", header: <span className="ep-sr-only">Hành động</span>, width: "56px", align: "end",
      render: (r) => <OverflowMenu label={`Hành động — ${r.student.full_name}`} items={[{ label: "Giải trình", onSelect: () => setExplainOf(r) }, { label: "Lịch sử", onSelect: () => setHistoryOf(r) }]} />,
    },
  ];

  const hasScheme = first?.scheme != null;
  async function download() {
    const blob = await exportXlsx(courseId);
    const a = document.createElement("a");
    a.href = URL.createObjectURL(blob);
    a.download = "so-diem.xlsx";
    a.click();
    URL.revokeObjectURL(a.href);
  }
  const banner = status && status.state !== "CONFIRMED" && (
    <InlineNotice tone="warning" compact action={teacher ? <ButtonLink size="sm" href="/gradebook/scheme">{status.state === "NO_POLICY" ? "Tải quy chế" : "Xác nhận công thức"}</ButtonLink> : <ButtonLink size="sm" href="/gradebook/scheme">Xem công thức</ButtonLink>}>
      Công thức điểm chưa xác nhận
    </InlineNotice>
  );
  const warnLabel = first?.warnings?.map((w) => (w === "PENALTY_NOT_IN_SCHEME" ? "Có điểm trừ ghi tay không được tính theo công thức" : "Có điểm cộng ghi tay không được tính theo công thức")) ?? [];

  const moreItems: HeaderAction[] =
    hasScheme && (teacher || (first?.state.snapshot_version ?? 0) > 0)
      ? [
          ...(teacher && !locked ? [{ label: "Nhập từ XLSX", onClick: () => setImporting(true) }, { label: "Nhập từ Forms", onClick: () => setFormsImporting(true) }] : []),
          ...(teacher ? [{ label: "Xuất XLSX", onClick: () => void download() }] : []),
          ...((first?.state.snapshot_version ?? 0) > 0 ? [{ label: "Lịch sử chốt", onClick: () => setHistory(true) }] : []),
          ...(teacher && locked ? [{ label: "Mở khoá", onClick: () => setUnlocking(true) }] : []),
        ]
      : [];
  const chotDiem: HeaderAction | undefined =
    teacher && status?.state === "CONFIRMED" && !locked
      ? { label: "Chốt điểm", onClick: () => setFinalizing(true) }
      : teacher && status && status.state !== "CONFIRMED"
        ? { label: "Chốt điểm", disabled: true, reason: status.finalize_blocked_reason ?? "Chưa có công thức điểm đã xác nhận" }
        : undefined;

  return (
    <Page kind="work">
      <PageState
        query={{ isPending: q.isPending, isError: q.isError, error: q.error, data: q.data, refetch: q.refetch }}
        loading={<><PageHeader title="Sổ điểm" /><Panel><Skeleton lines={10} /></Panel></>}
        showTechnical
      >
        <PageHeader
          title="Sổ điểm"
          primary={chotDiem}
          secondary={[{ label: "Xem công thức", href: "/gradebook/scheme" }]}
          more={moreItems}
          meta={[
            locked && <StatusText key="m0" tone="green">{`Đã chốt v${first?.state.snapshot_version}${snaps?.[0] ? ` · ${stamp(snaps[0].locked_at)}` : ""}`}</StatusText>,
            missing.length > 0 && (
              <button key="m1" type="button" className={s.missLink} aria-pressed={onlyMissing} data-part="missing-filter" onClick={() => setOnlyMissing((v) => !v)}>
                {`Thiếu điểm: ${missing.length} sinh viên`}
              </button>
            ),
          ]}
        />
        <Panel>
          {banner}
          {incomplete && (
            <InlineNotice tone="warning" compact action={<Button size="sm" variant="ghost" onClick={() => setIncomplete(null)}>Ẩn</Button>}>
              {`Còn ${incomplete.total} sinh viên chưa đủ điểm`}
              <ul className={f.missing} data-part="incomplete-list">
                {incomplete.students.map((m) => {
                  const who = rows.find((r) => r.student.id === m.id)?.student.full_name ?? "Sinh viên";
                  const col = columns.find((c) => m.missing.includes(c.component_key) && c.source === "MANUAL") ?? columns.find((c) => m.missing.includes(c.component_key));
                  return (
                    <li key={m.id}>
                      <button type="button" className={f.link} onClick={() => col && setFocusReq({ rowKey: m.id, col: col.item_id, n: Date.now() })}>{`${who} — thiếu ${col?.name ?? m.missing.join(", ")}`}</button>
                    </li>
                  );
                })}
              </ul>
            </InlineNotice>
          )}
          {warnLabel.map((w, i) => (
            <InlineNotice key={w} tone="warning" compact action={<Button size="sm" variant="ghost" onClick={() => setWarnOpen((x) => !x)}>{warnOpen ? "Ẩn" : "Xem"}</Button>}>
              {w}
              {warnOpen && <span className={s.warnList}>{(first?.warning_students?.[first.warnings![i]] ?? []).map((x) => x.full_name).join(", ")}</span>}
            </InlineNotice>
          ))}
          {!hasScheme ? (
            <EmptyState title="Chưa có công thức điểm" action={teacher ? <ButtonLink href="/gradebook/scheme">Thiết lập công thức</ButtonLink> : undefined} />
          ) : rows.length === 0 ? (
            <EmptyState title="Lớp chưa có sinh viên" />
          ) : (
            <DataTable
              columns={tableCols}
              rows={shownRows}
              rowKey={(r) => r.student.id}
              caption="Bảng điểm của lớp"
              virtual={{ height: tableHeight }}
              cellNav={{ drafts, onDraft: setDraft, onCommit: (row, col, text) => commit(row, col, text), focus: focusReq, saved: savedCells }}
              pagination={{ nextCursor: q.hasNextPage ? (q.data?.pages.at(-1)?.next_cursor ?? null) : null, onLoadMore: () => void q.fetchNextPage(), loading: q.isFetchingNextPage }}
              rowAttrs={(r) => ({ "data-part": "grade-row", "data-student-id": r.student.id })}
            />
          )}
          {hasPublished && rows.length > 0 && <p className={s.note} data-part="published-legend"><i className={s.pubDot} aria-hidden /> đã công bố</p>}
          {hasScheme && rows.length > 0 && <p className={s.note} data-part="official-note">{OFFICIAL_NOTE}</p>}
        </Panel>
      </PageState>
      {conflict && (
        <div className={s.conflict} style={{ top: conflict.at.top, left: conflict.at.left }} role="alertdialog" aria-label="Điểm vừa được người khác nhập" data-part="cell-conflict">
          <p className={s.conflictText}>{`${conflict.theirs.updated_by?.name ?? "Người khác"} vừa nhập ${vi(conflict.theirs.score) || "—"} · ${ago(conflict.theirs.updated_at)}`}</p>
          <div className={s.conflictActions}>
            <Button size="sm" variant="primary" onClick={() => resolve(true)}>Giữ của tôi</Button>
            <Button size="sm" onClick={() => resolve(false)}>Dùng của họ</Button>
          </div>
        </div>
      )}
      {finalizing && <FinalizeDialog course={courseId} onClose={() => setFinalizing(false)} onIncomplete={setIncomplete} />}
      {unlocking && <UnlockDialog course={courseId} version={first?.state.snapshot_version ?? 0} students={snaps?.[0]?.students ?? rows.length} onClose={() => setUnlocking(false)} />}
      {history && <SnapshotHistory course={courseId} onClose={() => setHistory(false)} />}
      {importing && <ImportDialog course={courseId} columns={columns} onClose={() => setImporting(false)} />}
      {formsImporting && <FormsImportDialog course={courseId} columns={columns} onClose={() => setFormsImporting(false)} />}
      <ExplainDrawer courseId={courseId} row={explainOf} onClose={() => setExplainOf(null)} />
      <HistoryDrawer courseId={courseId} row={historyOf} onClose={() => setHistoryOf(null)} />
    </Page>
  );
}

function ExplainDrawer({ courseId, row, onClose }: { courseId: string; row: GRow | null; onClose: () => void }) {
  const q = useExplain(courseId, row?.student.id ?? null);
  if (!row) return null;
  return (
    <Drawer open onClose={onClose} title={`Giải trình · ${row.student.full_name}`}>
      {q.isPending ? <Skeleton lines={6} /> : q.isError || !q.data ? <p>Chưa tải được giải trình.</p> : <GradeCalculation lines={q.data.lines} caption={`Giải trình điểm của ${row.student.full_name}`} />}
      {q.data?.lines.some((l) => l.value === "Còn bài chưa khớp") && (
        <ButtonLink size="sm" href="/grading?tab=unmatched">
          Xem bài chưa khớp
        </ButtonLink>
      )}
    </Drawer>
  );
}

const when = (iso: string) => formatWhen(iso);

function HistoryDrawer({ courseId, row, onClose }: { courseId: string; row: GRow | null; onClose: () => void }) {
  const q = useHistory(courseId, row?.student.id ?? null);
  if (!row) return null;
  const items = q.data?.pages.flatMap((p) => p.items) ?? [];
  return (
    <Drawer open onClose={onClose} title={`Lịch sử · ${row.student.full_name}`}>
      {q.isPending ? (
        <Skeleton lines={5} />
      ) : items.length === 0 ? (
        <p>Chưa có thay đổi.</p>
      ) : (
        <>
          <ul className={s.history} data-part="history-list">
            {items.map((h, i) => (
              <li key={i}>
                <span>{`${when(h.at)} · ${h.by}`}</span>
                <span>{`${h.item}: ${h.from ? vi(h.from) : "—"} → ${h.to ? vi(h.to) : "—"}`}</span>
              </li>
            ))}
          </ul>
          {q.hasNextPage && <Button size="sm" variant="ghost" onClick={() => void q.fetchNextPage()}>Xem thêm</Button>}
        </>
      )}
    </Drawer>
  );
}
