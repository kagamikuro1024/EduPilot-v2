"use client";

import { useQueryClient } from "@tanstack/react-query";
import { FileText, Upload } from "lucide-react";
import { useCallback, useEffect, useMemo, useRef, useState, useSyncExternalStore } from "react";
import { ApiError, newIdempotencyKey } from "@/shared/data";
import { Button, ButtonLink, ConfirmIrreversible, Drawer, EmptyState, InlineNotice, Page, PageHeader, PageState, Panel, PanelSection, Skeleton, StatusText } from "@/shared/ui";
import { useSession } from "@/shared/session/session";
import { CitationPane } from "./CitationPane";
import { BLOCKS, Blocks, unclearOf, type BlockId } from "./SchemeBlocks";
import { confirmScheme, schemeStatusKey, toWorking, useSchemes, type Citation, type Impact, type SchemeDiff, type SchemeDoc } from "./gradeApi";
import { useSchemeEditor } from "./useSchemeEditor";
import { usePolicyUpload, type PolicyUpload } from "./usePolicyUpload";
import s from "./GradeScheme.module.css";

const vi = (v: unknown) => String(v ?? "").replace(".", ",");
const WIDE = "(min-width: 1024px)";
const REASON = { ITEM_REMOVED: "cột bị bỏ", MAX_CHANGED: "đổi thang", SOURCE_CHANGED: "đổi nguồn" } as const;

function useWide() {
  return useSyncExternalStore(
    (cb) => {
      const m = window.matchMedia(WIDE);
      m.addEventListener("change", cb);
      return () => m.removeEventListener("change", cb);
    },
    () => window.matchMedia(WIDE).matches,
    () => false,
  );
}

const at = (iso?: string) => (iso ? new Date(iso).toLocaleString("vi-VN", { day: "2-digit", month: "2-digit", hour: "2-digit", minute: "2-digit" }) : "");

/** Công thức điểm: chỉ có thật, Giảng viên / TA của lớp (TA chỉ đọc). */
export function GradeScheme() {
  const { role, realCourseId, realPending, realCourses } = useSession();
  if (realPending || !realCourseId || realCourseId === "all" || (role !== "teacher" && role !== "ta")) return null;
  const code = realCourses?.find((c) => c.id === realCourseId)?.class_code ?? "";
  return <RealGradeScheme courseId={realCourseId} classCode={code} teacher={role === "teacher"} />;
}

function RealGradeScheme({ courseId, teacher }: { courseId: string; classCode: string; teacher: boolean }) {
  const qc = useQueryClient();
  const base = useSchemes(courseId, false);
  const [compare, setCompare] = useState(false);
  const diffQ = useSchemes(courseId, true, compare);
  const picker = useRef<HTMLInputElement>(null);
  const refetch = useCallback(() => void base.refetch(), [base]);
  const upload = usePolicyUpload(courseId, refetch);
  const draft = base.data?.draft ?? null;
  const confirmed = base.data?.confirmed ?? null;

  const header = (
    <PageHeader
      title="Công thức điểm"
      back={{ href: "/gradebook", label: "Sổ điểm" }}
      meta={draft ? <StatusText tone="amber">Bản nháp</StatusText> : confirmed ? <span data-part="confirmed-line"><StatusText tone="green">{`Đã xác nhận · ${at(confirmed.confirmed_at)}`}</StatusText></span> : undefined}
      actions={
        teacher && (
          <>
            <input ref={picker} type="file" hidden accept=".pdf,.docx,.pptx" data-part="policy-input" onChange={(e) => { const f = e.target.files?.[0]; e.target.value = ""; if (f) void upload.start(f); }} />
            <Button variant="secondary" icon={<Upload aria-hidden />} disabled={upload.state.phase === "uploading" || upload.state.phase === "reading" || upload.state.phase === "extracting"} onClick={() => picker.current?.click()}>
              Tải quy chế
            </Button>
          </>
        )
      }
    />
  );

  return (
    <Page>
      <PageState
        query={{ isPending: base.isPending, isError: base.isError, error: base.error, data: base.data, refetch: base.refetch }}
        loading={<><PageHeader title="Công thức điểm" back={{ href: "/gradebook", label: "Sổ điểm" }} /><Panel><Skeleton lines={10} /></Panel></>}
        showTechnical
      >
        {header}
        <UploadProgress up={upload} />
        {!draft && !confirmed ? (
          <Panel>
            <EmptyState
              title="Lớp chưa có quy chế."
              icon={<FileText aria-hidden />}
              action={teacher ? <Button variant="primary" icon={<Upload aria-hidden />} onClick={() => picker.current?.click()}>Tải quy chế</Button> : <ButtonLink href="/gradebook">Về sổ điểm</ButtonLink>}
            />
          </Panel>
        ) : draft ? (
          <Review
            key={draft.id}
            courseId={courseId}
            draft={draft}
            confirmed={confirmed}
            editable={teacher}
            compare={compare}
            diff={diffQ.data?.diff}
            onCompare={setCompare}
            onConfirmed={() => {
              void qc.invalidateQueries({ queryKey: ["grade-schemes", courseId] });
              void qc.invalidateQueries({ queryKey: schemeStatusKey(courseId) });
              void qc.invalidateQueries({ queryKey: ["today"] });
            }}
          />
        ) : (
          confirmed && <ReadOnly courseId={courseId} doc={confirmed} />
        )}
      </PageState>
    </Page>
  );
}

function UploadProgress({ up }: { up: PolicyUpload }) {
  const { state, percent } = up;
  if (state.phase === "idle") return null;
  if (state.phase === "error") {
    return (
      <InlineNotice
        tone="danger"
        action={
          <>
            {state.canReplace && <Button size="sm" onClick={up.replace}>Thay bản nháp</Button>}
            <Button size="sm" onClick={up.retry}>{state.docId ? "Thử lại" : "Tải lại"}</Button>
            <Button size="sm" variant="ghost" onClick={up.dismiss}>Bỏ qua</Button>
          </>
        }
      >
        {state.file ? `${state.file.name}: ${state.error}` : state.error}
      </InlineNotice>
    );
  }
  const label = state.phase === "uploading" ? "Đang tải" : state.phase === "reading" ? "Đang đọc" : "Đang trích";
  return (
    <div className={s.progress} data-part="policy-progress" role="status">
      <span className={s.bar}><span className={s.barFill} style={{ width: `${percent}%` }} /></span>
      <span className={s.progressText}>{`${label} · ${state.file?.name ?? ""}`}</span>
    </div>
  );
}

function ReadOnly({ courseId, doc }: { courseId: string; doc: SchemeDoc }) {
  const wide = useWide();
  const [sel, setSel] = useState<{ block: BlockId; cite: Citation } | null>(null);
  const w = useMemo(() => toWorking(doc), [doc]);
  return (
    <Panel>
      <div className={s.review}>
        <div className={s.main}>
          <Blocks courseId={courseId} doc={doc} working={w} editable={false} errors={{}} selected={sel?.block ?? null} onSelect={(block, cite) => cite && setSel({ block, cite })} onChange={() => undefined} />
        </div>
        {wide ? <aside className={s.aside}><CitationPane course={courseId} docId={doc.source_document_id} title={doc.source_title} cite={sel?.cite ?? null} /></aside> : null}
      </div>
      {!wide && sel && <CiteDrawer open onClose={() => setSel(null)} courseId={courseId} doc={doc} cite={sel.cite} />}
    </Panel>
  );
}

function CiteDrawer({ open, onClose, courseId, doc, cite }: { open: boolean; onClose: () => void; courseId: string; doc: SchemeDoc; cite: Citation | null }) {
  return (
    <Drawer open={open} onClose={onClose} title={doc.source_title ?? "Quy chế"}>
      <CitationPane course={courseId} docId={doc.source_document_id} title={undefined} cite={cite} />
    </Drawer>
  );
}

function firstCitation(doc: SchemeDoc): { block: BlockId; cite: Citation } | null {
  for (const b of BLOCKS) {
    const c = b.id === "components" ? doc.components.flatMap((x) => x.citations)[0] : doc.rules.filter((r) => r.kind === b.id).flatMap((r) => r.citations)[0];
    if (c) return { block: b.id, cite: c };
  }
  return null;
}

function lockReason(doc: SchemeDoc, saving: string): string | null {
  if (saving === "offline" || saving === "error") return "Chưa lưu";
  if (saving === "saving") return "Đang lưu";
  const un = doc.unclear_items;
  if (un.length === 0) return null;
  const names = BLOCKS.filter((b) => unclearOf(un, b.id).length > 0).map((b) => b.short);
  return `Còn ${un.length} mục chưa rõ: ${names.join(", ")}`;
}

function Review({ courseId, draft, confirmed, editable, compare, diff, onCompare, onConfirmed }: { courseId: string; draft: SchemeDoc; confirmed: SchemeDoc | null; editable: boolean; compare: boolean; diff?: SchemeDiff; onCompare: (v: boolean) => void; onConfirmed: () => void }) {
  const ed = useSchemeEditor(courseId, draft);
  const wide = useWide();
  const doc = ed.doc ?? draft;
  const [sel, setSel] = useState<{ block: BlockId; cite: Citation } | null>(null);
  const [asking, setAsking] = useState(false);
  const [busy, setBusy] = useState(false);
  const [err, setErr] = useState<string | null>(null);
  const key = useRef("");

  // mở sẵn trích dẫn đầu tiên để cột phải không trống
  const first = firstCitation(doc);
  const shown = sel ?? first;
  const reason = lockReason(doc, ed.state);
  const locked = Boolean(reason) || !ed.impact;

  useEffect(() => {
    if (asking) key.current = newIdempotencyKey();
  }, [asking]);

  async function doConfirm() {
    setBusy(true);
    setErr(null);
    try {
      await confirmScheme(courseId, doc.version, key.current);
      setAsking(false);
      onConfirmed();
    } catch (e) {
      key.current = newIdempotencyKey();
      if (e instanceof ApiError && (e.code === "VERSION_CONFLICT" || e.status === 422)) {
        onConfirmed();
        setErr(e.code === "VERSION_CONFLICT" ? "Có người vừa sửa công thức. Đã tải bản mới." : "Còn mục chưa rõ.");
      } else setErr(e instanceof ApiError ? e.userMessage : "Chưa xác nhận được.");
    } finally {
      setBusy(false);
    }
  }

  const imp = ed.impact;
  const dropped = imp?.entries_dropped.reduce((n, d) => n + d.count, 0) ?? 0;
  const consequence = imp ? (
    <>
      {`Áp cho ${imp.students_total} sinh viên · điểm học phần của ${imp.changed_vs_confirmed} sinh viên sẽ đổi so với công thức hiện tại · ${imp.entries_carried} điểm đã nhập được giữ · ${dropped} điểm đã nhập bị bỏ`}
      {imp.entries_dropped.length > 0 && ` (${imp.entries_dropped.map((d) => `${d.item_key}: ${d.count}, ${REASON[d.reason]}`).join("; ")})`}
    </>
  ) : null;

  const saving = ed.state === "saving" ? "Đang lưu…" : ed.state === "offline" ? "Chưa lưu" : ed.state === "error" ? "Chưa lưu" : null;

  return (
    <Panel>
      {ed.notice && <InlineNotice tone="warning" compact action={<Button size="sm" variant="ghost" onClick={ed.dismissNotice}>Đã rõ</Button>}>{ed.notice}</InlineNotice>}
      <div className={s.bar2}>
        {saving && <span data-part="save-state"><StatusText tone="amber">{saving}</StatusText></span>}
        {confirmed && <Button size="sm" variant="ghost" aria-pressed={compare} onClick={() => onCompare(!compare)}>{compare ? "Bản nháp" : "So sánh"}</Button>}
        {editable && (
          <span className={s.lock}>
            <Button variant="primary" aria-disabled={locked || undefined} className={locked ? s.lockBtn : undefined} aria-describedby={locked ? "confirm-why" : undefined} onClick={() => (locked ? undefined : setAsking(true))}>
              Xác nhận công thức
            </Button>
            {locked && <span id="confirm-why" role="tooltip" className={s.lockWhy}>{reason ?? "Đang tính"}</span>}
          </span>
        )}
      </div>
      <div className={s.review}>
        <div className={s.main}>
          {compare && diff ? (
            <DiffView diff={diff} />
          ) : (
            ed.working && <Blocks courseId={courseId} doc={doc} working={ed.working} editable={editable} errors={ed.errors} selected={shown?.block ?? null} onSelect={(block, cite) => cite && setSel({ block, cite })} onChange={ed.commit} />
          )}
          {!compare && <Preview impact={imp} />}
        </div>
        {wide ? <aside className={s.aside}><CitationPane course={courseId} docId={doc.source_document_id} title={doc.source_title} cite={shown?.cite ?? null} /></aside> : null}
      </div>
      {!wide && sel && <CiteDrawer open onClose={() => setSel(null)} courseId={courseId} doc={doc} cite={sel.cite} />}
      {editable && (
        <ConfirmIrreversible
          open={asking}
          onClose={() => setAsking(false)}
          onConfirm={() => void doConfirm()}
          title="Xác nhận công thức"
          consequence={consequence}
          confirmLabel="Xác nhận công thức"
          disabledReason={imp ? undefined : "Đang tính"}
          loading={busy}
          error={err ?? undefined}
        />
      )}
    </Panel>
  );
}

const grade = (g: { hp: string; letter: string }) => `${vi(g.hp)} (${g.letter})`;

function Preview({ impact }: { impact: Impact | null }) {
  if (!impact) return null;
  return (
    <PanelSection title="Xem thử">
      {impact.samples.length === 0 ? (
        <p className={s.value} data-part="impact-empty">Chưa đủ điểm để xem thử</p>
      ) : (
        <ul className={s.samples} data-part="impact-samples">
          {impact.samples.map((x) => (
            <li key={x.student.id}>
              <span>{x.student.full_name}</span>
              <span>{x.before ? `${grade(x.before)} → ${grade(x.after)}` : grade(x.after)}</span>
            </li>
          ))}
        </ul>
      )}
    </PanelSection>
  );
}

function DiffView({ diff }: { diff: SchemeDiff }) {
  const rows: string[] = [
    ...diff.components_added.map((k) => `Thêm thành phần ${k}`),
    ...diff.components_removed.map((k) => `Bớt thành phần ${k}`),
    ...diff.weight_changed.map((c) => `${c.key}: ${vi(c.from)}% → ${vi(c.to)}%`),
    ...diff.rules_changed.map((r) => `${r.kind}: ${JSON.stringify(r.from)} → ${JSON.stringify(r.to)}`),
  ];
  return (
    <PanelSection title="So với bản đang dùng">
      {rows.length === 0 ? <p className={s.value}>Không khác</p> : <ul className={s.samples} data-part="diff-rows">{rows.map((r) => <li key={r}><span>{r}</span></li>)}</ul>}
      {diff.weight_sum_from !== diff.weight_sum_to && <p className={s.value}>{`Tổng trọng số ${vi(diff.weight_sum_from)}% → ${vi(diff.weight_sum_to)}%`}</p>}
    </PanelSection>
  );
}
