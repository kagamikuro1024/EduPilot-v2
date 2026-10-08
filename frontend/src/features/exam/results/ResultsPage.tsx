"use client";

import { useQuery, useQueryClient } from "@tanstack/react-query";
import { useRouter, useSearchParams } from "next/navigation";
import { useEffect, useState } from "react";
import { ApiErrorNotice, apiClient, useCursorList } from "@/shared/data";
import { useClassCourse } from "@/features/members/classApi";
import { useSession } from "@/shared/session/session";
import { Button, ConfirmIrreversible, DataTable, EmptyState, Field, Input, InlineNotice, Page, PageHeader, SegmentedControl, Select, Skeleton, StatusText, Tabs, Textarea, type Column } from "@/shared/ui";
import { eKey, fmtClock, type ExamDetail } from "../examApi";
import { AppealsPanel } from "./AppealsPanel";
import { ResultDrawer } from "./ResultDrawer";
import { StatsPanel } from "./StatsPanel";
import { REASON_VI, resPath, saveBlob, STATUS_TONE, STATUS_VI, vnum, type Progress, type ResultsPage as Page_, type Row } from "./resultsApi";
import s from "./Results.module.css";

type Tab = "results" | "appeals" | "stats" | "similarity";
type Filter = "all" | "GRADED" | "GRADING" | "ABSENT";

/** `/exams/[id]/results` (Giảng viên và TA): bảng điểm của lớp, thống kê, phúc khảo; hàng mở Drawer chi tiết. `Tín hiệu` và `Nghi giống nhau` chỉ có với Giảng viên. */
export function ResultsPage({ id }: { id: string }) {
  const { role } = useSession();
  const cc = useClassCourse();
  const hint = useSearchParams().get("course");
  const course = hint ?? (cc.state === "ready" ? cc.course.id : null);
  const back = { href: "/exams", label: "Bài thi" };
  if (role !== "teacher" && role !== "ta")
    return (
      <Page width="wide">
        <PageHeader title="Kết quả bài thi" back={back} />
        <EmptyState title="Chỉ dành cho giảng viên và TA">Trang này dành cho giảng viên và TA của lớp.</EmptyState>
      </Page>
    );
  if (!course)
    return (
      <Page width="wide">
        <PageHeader title="Kết quả bài thi" back={back} />
        {cc.state === "loading" ? <Skeleton lines={6} /> : <EmptyState title="Chưa chọn lớp">Chọn một lớp ở thanh trên để xem kết quả.</EmptyState>}
      </Page>
    );
  return <Board course={course} exam={id} teacher={role === "teacher"} />;
}

function Board({ course, exam, teacher }: { course: string; exam: string; teacher: boolean }) {
  const router = useRouter();
  const qc = useQueryClient();
  const sp = useSearchParams();
  const [tab, setTab] = useState<Tab>(sp.get("tab") === "appeals" ? "appeals" : "results");
  const detail = useQuery({
    queryKey: eKey(course, "detail", exam),
    queryFn: async ({ signal }) => (await apiClient.get<ExamDetail>(`/courses/${course}/exams/${exam}`, { signal })).data,
    refetchInterval: (q) => (q.state.data && (q.state.data.status === "OPEN" || q.state.data.status === "CLOSED" || q.state.data.regrading) ? 15_000 : false),
  });
  const e = detail.data;
  const tabs: Array<{ value: Tab; label: string }> = [
    { value: "results", label: "Bảng điểm" },
    { value: "appeals", label: "Xem lại điểm" },
    { value: "stats", label: "Thống kê" },
    ...(teacher ? [{ value: "similarity" as const, label: "Nghi giống nhau" }] : []),
  ];
  function pick(t: Tab) {
    if (t === "similarity") router.push(`/exams/${exam}/similarity?course=${course}`);
    else setTab(t);
  }
  return (
    <Page width="wide">
      <PageHeader title={e ? `Kết quả · ${e.title}` : "Kết quả bài thi"} back={{ href: "/exams", label: "Bài thi" }} />
      {detail.isError && <ApiErrorNotice error={detail.error} onRetry={() => void detail.refetch()} />}
      {e && e.status !== "CLOSED" && e.status !== "PUBLISHED" && e.status !== "OPEN" && <InlineNotice>Bài thi chưa mở nên chưa có kết quả.</InlineNotice>}
      <Tabs label="Phần của kết quả" value={tab} onChange={pick} options={tabs} />
      {tab === "results" && <Table course={course} exam={exam} teacher={teacher} e={e} onChanged={() => void qc.invalidateQueries({ queryKey: eKey(course, "detail", exam) })} />}
      {tab === "appeals" && <AppealsPanel course={course} exam={exam} teacher={teacher} />}
      {tab === "stats" && <StatsPanel course={course} exam={exam} />}
    </Page>
  );
}

function progressLine(p: Progress, status: string | undefined): string {
  if (status === "OPEN") return `Đang làm ${p.in_progress} · Đã nộp ${p.grading + p.graded} · Chưa bắt đầu ${p.not_started}`;
  if (status === "CLOSED") return `Đang chấm ${p.graded}/${p.graded + p.grading}${p.absent ? ` · Vắng ${p.absent}` : ""}`;
  return `Đã chấm ${p.graded}${p.absent ? ` · Vắng ${p.absent}` : ""}`;
}

function Table({ course, exam, teacher, e, onChanged }: { course: string; exam: string; teacher: boolean; e: ExamDetail | undefined; onChanged: () => void }) {
  const qc = useQueryClient();
  const [filter, setFilter] = useState<Filter>("all");
  const [q, setQ] = useState("");
  const [open, setOpen] = useState<string | null>(null);
  const [confirm, setConfirm] = useState(false);
  const [regrade, setRegrade] = useState(false);
  const key = ["exam-results", course, exam, filter, q];
  const list = useCursorList<Row>(key, `${resPath(course, exam)}/results`, { query: { status: filter === "all" ? undefined : filter, q: q.trim() || undefined }, idOf: (r) => r.student.id, limit: 100 });
  const first = list.data?.pages[0] as Page_ | undefined;
  const open_ = e?.status === "OPEN";
  useEffect(() => {
    if (!open_) return;
    const t = setInterval(() => void qc.invalidateQueries({ queryKey: ["exam-results", course, exam] }), 15_000); // bài đang mở: tiến độ làm mới 15 s
    return () => clearInterval(t);
  }, [open_, qc, course, exam]);

  const [hold, setHold] = useState<{ pending: boolean; error: unknown }>({ pending: false, error: null });
  async function setPublishHold(v: boolean) {
    if (!e) return;
    setHold({ pending: true, error: null });
    try {
      await apiClient.put(`${resPath(course, exam)}/publish-hold`, { hold: v, version: e.version });
      setHold({ pending: false, error: null });
      setConfirm(false);
      onChanged();
      void qc.invalidateQueries({ queryKey: ["exam-results", course, exam] });
      void qc.invalidateQueries({ queryKey: ["today"] });
    } catch (err) {
      setHold({ pending: false, error: err });
    }
  }
  const [exporting, setExporting] = useState<unknown>(null);
  async function exportCsv() {
    setExporting(null);
    try {
      const r = await apiClient.get<Blob>(`${resPath(course, exam)}/results.csv`, { blob: true });
      saveBlob(r.data, "ket-qua-bai-thi.csv");
    } catch (err) {
      setExporting(err);
    }
  }

  const cols: Column<Row>[] = [
    { key: "name", header: "Họ tên", frozen: true, primary: true, width: "220px", render: (r) => r.student.full_name },
    { key: "code", header: "MSSV", width: "120px", render: (r) => r.student.student_code || "—" },
    { key: "status", header: "Trạng thái", width: "130px", render: (r) => <StatusText tone={STATUS_TONE[r.status]}>{STATUS_VI[r.status]}</StatusText> },
    { key: "score", header: "Điểm", align: "end", width: "130px", render: (r) => (r.score ? <span>{vnum(r.score)}{r.adjusted ? " (đã sửa)" : ""}</span> : "—") },
    { key: "at", header: "Nộp lúc", width: "150px", render: (r) => (r.submitted_at ? `${fmtClock(r.submitted_at)}${r.submit_reason && r.submit_reason !== "MANUAL" ? ` · ${REASON_VI[r.submit_reason] ?? ""}` : ""}` : "—") },
    ...(teacher
      ? [{ key: "signal", header: "Tín hiệu", width: "190px", render: (r: Row) => signal(r) } satisfies Column<Row>]
      : []),
  ];

  const graded = first?.progress.graded ?? 0;
  const primary = teacher && e?.status === "CLOSED" && e.publish_hold;
  return (
    <>
      <div className={s.bar}>
        {first ? <p className={s.progress} data-part="results-progress">{progressLine(first.progress, e?.status)}</p> : <span />}
        <span className={s.grow} />
        <SegmentedControl label="Lọc theo trạng thái" value={filter} onChange={setFilter} options={[{ value: "all", label: "Tất cả" }, { value: "GRADED", label: "Đã chấm" }, { value: "GRADING", label: "Đang chấm" }, { value: "ABSENT", label: "Vắng" }]} />
        {primary && <Button variant="primary" onClick={() => setConfirm(true)}>Công bố điểm</Button>}
        <Button onClick={() => void exportCsv()}>Xuất CSV</Button>
        {teacher && e && (e.status === "CLOSED" || e.status === "PUBLISHED") && e.kind !== "MCQ" && <Button onClick={() => setRegrade(true)}>Chấm lại</Button>}
        {teacher && e?.status === "CLOSED" && !e.publish_hold && <Button onClick={() => void setPublishHold(true)} loading={hold.pending}>Hoãn công bố</Button>}
      </div>
      {e?.regrading && <InlineNotice compact>Đang chấm lại — điểm cũ giữ nguyên tới khi tính xong.</InlineNotice>}
      {exporting != null && <ApiErrorNotice error={exporting} />}
      {hold.error != null && !confirm && <ApiErrorNotice error={hold.error} />}
      <div className={s.bar}>
        <Field label="Tìm theo tên hoặc MSSV">{(id, by) => <Input id={id} aria-describedby={by} value={q} onChange={(ev) => setQ(ev.target.value)} maxLength={100} />}</Field>
      </div>
      {regrade && <RegradePanel course={course} exam={exam} onClose={() => setRegrade(false)} onDone={() => { setRegrade(false); onChanged(); }} />}
      {list.isPending ? (
        <Skeleton lines={8} />
      ) : list.isError ? (
        <ApiErrorNotice error={list.error} onRetry={() => void list.refetch()} />
      ) : (
        <div className={s.tableWrap}>
          <DataTable
            caption="Bảng điểm của lớp"
            columns={cols}
            rows={list.items}
            rowKey={(r) => r.student.id}
            onRowClick={(r) => r.attempt_id && setOpen(r.attempt_id)}
            activeKey={list.items.find((r) => r.attempt_id === open)?.student.id}
            virtual={{ height: 520 }}
            pagination={{ nextCursor: list.hasNextPage ? (list.data?.pages.at(-1)?.next_cursor ?? null) : null, onLoadMore: () => void list.fetchNextPage(), loading: list.isFetchingNextPage }}
            empty={<EmptyState title="Chưa có dòng nào">Không có sinh viên nào khớp bộ lọc.</EmptyState>}
          />
        </div>
      )}
      {open && e && <ResultDrawer key={open} course={course} exam={exam} attempt={open} teacher={teacher} e={e} onClose={() => setOpen(null)} onChanged={() => { void qc.invalidateQueries({ queryKey: ["exam-results", course, exam] }); onChanged(); }} />}
      <ConfirmIrreversible
        open={confirm}
        onClose={() => setConfirm(false)}
        onConfirm={() => void setPublishHold(false)}
        title="Công bố điểm?"
        consequence={`Công bố điểm cho ${graded} sinh viên. Họ sẽ thấy điểm${e?.reveal_answers ? ", đáp án" : ""} và có thể gửi yêu cầu xem lại trong ${e?.appeal_days ?? 0} ngày. Không thu hồi được.`}
        confirmLabel="Công bố"
        loading={hold.pending}
        error={confirm && hold.error ? "Chưa công bố được, thử lại." : undefined}
      />
    </>
  );
}

function signal(r: Row) {
  const f = r.flags;
  if (!f) return "—";
  const parts = [f.tab_hidden > 0 ? `Rời tab ${f.tab_hidden}` : "", f.similarity > 0 ? `Giống nhau ${f.similarity}` : ""].filter(Boolean);
  return parts.length ? parts.join(" · ") : "—";
}

/** Mở dần tại chỗ phía trên bảng: phạm vi + lý do + `Chấm lại`. */
function RegradePanel({ course, exam, onClose, onDone }: { course: string; exam: string; onClose: () => void; onDone: () => void }) {
  const [scope, setScope] = useState<"all" | "errors">("errors");
  const [reason, setReason] = useState("");
  const [busy, setBusy] = useState(false);
  const [err, setErr] = useState<unknown>(null);
  async function go() {
    setBusy(true);
    setErr(null);
    try {
      await apiClient.post(`${resPath(course, exam)}/regrade`, { scope, reason: reason.trim() });
      onDone();
    } catch (e) {
      setErr(e);
    } finally {
      setBusy(false);
    }
  }
  return (
    <section className={s.form} aria-label="Chấm lại bài code" data-part="regrade-panel">
      <p className={s.muted}>Chỉ chấm lại các bài chưa chấm bằng bộ test hiện hành. Điểm cũ giữ nguyên tới khi chấm xong; điểm đổi thì sinh viên được báo nếu bài đã công bố.</p>
      <Field label="Phạm vi">{(id, by) => <Select id={id} aria-describedby={by} value={scope} onChange={(ev) => setScope(ev.target.value as "all" | "errors")}><option value="errors">Chỉ bài chấm lỗi</option><option value="all">Mọi bài nộp</option></Select>}</Field>
      <Field label="Lý do (bắt buộc, tối đa 500 ký tự)">{(id, by) => <Textarea id={id} aria-describedby={by} rows={2} maxLength={500} value={reason} onChange={(ev) => setReason(ev.target.value)} />}</Field>
      {err != null && <ApiErrorNotice error={err} />}
      <div className={s.bar}>
        <Button variant="primary" onClick={() => void go()} loading={busy} disabled={reason.trim() === ""}>Chấm lại</Button>
        <Button onClick={onClose} disabled={busy}>Để sau</Button>
      </div>
    </section>
  );
}
