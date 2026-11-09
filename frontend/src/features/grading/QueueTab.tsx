"use client";

import { useQueryClient } from "@tanstack/react-query";
import Link from "next/link";
import { useRouter } from "next/navigation";
import { ArrowRight, ListChecks, Upload } from "lucide-react";
import { useEffect, useRef, useState, useSyncExternalStore } from "react";
import { ApiError, newIdempotencyKey, useJob } from "@/shared/data";
import { checkSubmissionBundle, uploadSubmissionBundle } from "@/shared/data/uploadFile";
import { Button, type Column, ConfirmIrreversible, DataTable, EmptyState, Field, FilterChips, InlineNotice, PageState, Panel, Select, Skeleton, StatusText, Toolbar } from "@/shared/ui";
import { useAssignments, vi } from "@/features/assignments/assignmentApi";
import { useSchemes } from "@/features/gradebook/gradeApi";
import { fmtLong } from "@/features/calendar/calendarApi";
import { SOURCE_LABEL } from "./submissionApi";
import { FLAG_TEXT, gradeAll, importSubmissions, publishResults, reviewStateText, syncFromTeams, useQueue, type QueueFilter, type QueueRow, type QueueStatus, type TeamsSyncResult } from "./gradingApi";
import s from "./Grading.module.css";

const STATUS_OPTIONS: Array<{ value: QueueStatus; label: string }> = [
  { value: "DRAFT", label: "Chưa duyệt" },
  { value: "REVIEWED", label: "Đã duyệt" },
  { value: "PUBLISHED", label: "Đã công bố" },
  { value: "FAILED", label: "Chấm lỗi" },
  { value: "RUNNING", label: "Đang chấm" },
  { value: "NONE", label: "Chưa chấm" },
];

const reviewTone = (text: string) => (text === "Đã công bố" ? "green" : text === "Đã duyệt" ? "blue" : text === "Chấm lỗi" ? "red" : text === "Đang chấm" ? "neutral" : "amber") as "green" | "blue" | "red" | "neutral" | "amber";

/** Lý do một bài Teams không lấy được (mã của máy chủ → lời thường). */
const TEAMS_FAIL_TEXT: Record<string, string> = {
  FILE_DOWNLOAD_FAILED: "Không tải được tệp",
  TEAMS_RESPONSE_INVALID: "Teams trả dữ liệu không đúng",
  TEAMS_AUTH_FAILED: "Không đăng nhập được vào Teams",
  FILE_TYPE_NOT_ALLOWED: "Loại tệp không nhận",
  FILE_TOO_LARGE: "Tệp quá lớn",
  FILE_EMPTY: "Tệp trống",
};

/** Liên kết sang màn duyệt mang theo bộ lọc đang xem: màn đó dùng lại đúng hàng chờ để đi bài kế. */
export const reviewHref = (id: string, f: QueueFilter) => {
  const p = new URLSearchParams();
  if (f.assignment_id) p.set("assignment", f.assignment_id);
  if (f.status) p.set("status", f.status);
  if (f.needs_review) p.set("needs", "1");
  const q = p.toString();
  return q ? `/grading/${id}?${q}` : `/grading/${id}`;
};

const NARROW = "(max-width: 719px)";
const subscribeNarrow = (cb: () => void) => {
  const m = window.matchMedia(NARROW);
  m.addEventListener("change", cb);
  return () => m.removeEventListener("change", cb);
};

/** Tab `Hàng chờ` của `/grading` (US-P7-08 AC1): bảng ảo hoá, lọc mặc định `Cần xem kỹ` + `Chưa duyệt`. */
export function QueueTab({ courseId, teacher }: { courseId: string; teacher: boolean }) {
  const qc = useQueryClient();
  const router = useRouter();
  const assignments = useAssignments(courseId);
  const [assignmentId, setAssignmentId] = useState("");
  const [status, setStatus] = useState<QueueStatus>("DRAFT");
  const [flagOnly, setFlagOnly] = useState(true);
  const [notice, setNotice] = useState<string | null>(null);
  const [busy, setBusy] = useState(false);
  const picker = useRef<HTMLInputElement>(null);
  const running = useRef(false);
  const [picked, setPicked] = useState<Set<string>>(new Set());
  const [job, setJob] = useState<{ id: string; queued: number } | null>(null);
  const [confirming, setConfirming] = useState(false);
  const publishIdem = useRef<string | null>(null);
  const relaxed = useRef(false);
  const [teamsJob, setTeamsJob] = useState<string | null>(null);
  const [teamsWait, setTeamsWait] = useState(0);

  const filter: QueueFilter = { assignment_id: assignmentId || undefined, status, needs_review: flagOnly };
  const narrow = useSyncExternalStore(subscribeNarrow, () => window.matchMedia(NARROW).matches, () => false);
  const list = useQueue(courseId, filter);
  const rows = list.items;

  // Lọc mặc định có `Cần xem kỹ`; nếu không còn việc chờ nào thì bỏ lọc cờ MỘT lần (AC1).
  useEffect(() => {
    if (!relaxed.current && flagOnly && !list.isPending && !list.isError && rows.length === 0) {
      relaxed.current = true;
      setFlagOnly(false);
    }
  }, [flagOnly, list.isPending, list.isError, rows.length]);

  // Tiến độ của CẢ lô do máy chủ báo (job.progress); `rows.length` chỉ là số dòng đã tải nên không dùng làm tổng.
  const jobState = useJob(job?.id ?? null);
  const batch = (jobState.result ?? null) as { done?: number; failed?: number; total?: number } | null;
  const batchDone = (batch?.done ?? 0) + (batch?.failed ?? 0);
  const batchTotal = batch?.total ?? job?.queued ?? 0;
  const batchRunning = Boolean(job) && jobState.status !== "SUCCEEDED" && jobState.status !== "FAILED";
  const waiting = rows.some((r) => r.result && (r.result.run_state === "QUEUED" || r.result.run_state === "RUNNING"));
  useEffect(() => {
    if (!waiting && !batchRunning) return;
    const t = setInterval(() => void list.refetch(), 5000);
    return () => clearInterval(t);
  }, [waiting, batchRunning, list]);

  // Lấy bài từ Teams (US-P7-11 AC4): chỉ cho bài có kênh TEAMS; tiến độ và danh sách không lấy được đọc từ cùng một việc.
  const teamsState = useJob(teamsJob);
  const teams = (teamsState.result ?? null) as TeamsSyncResult | null;
  const teamsRunning = Boolean(teamsJob) && teamsState.status !== "SUCCEEDED" && teamsState.status !== "FAILED";
  const teamsChannel = assignments.items.find((a) => a.id === assignmentId)?.channels.includes("TEAMS") ?? false;
  const teamsFailed = teamsState.status === "SUCCEEDED" ? teams?.failed ?? [] : [];

  const teamsSync = () =>
    void run(async () => {
      setTeamsWait(0);
      setTeamsJob(null);
      try {
        const out = await syncFromTeams(courseId, assignmentId, newIdempotencyKey());
        setTeamsJob(out.job_id);
      } catch (e) {
        if (e instanceof ApiError && e.status === 429) {
          setTeamsWait(e.retryAfter ?? 30);
          setNotice(e.userMessage);
          return;
        }
        throw e;
      }
    });
  useEffect(() => {
    if (teamsState.status !== "SUCCEEDED") return;
    void qc.invalidateQueries({ queryKey: ["grading-queue", courseId] });
  }, [teamsState.status, qc, courseId]);

  const refresh = () => void qc.invalidateQueries({ queryKey: ["grading-queue", courseId] });

  function fail(e: unknown) {
    setNotice(e instanceof ApiError ? e.userMessage : "Chưa thực hiện được. Hãy thử lại.");
  }

  async function run(job: () => Promise<void>) {
    if (running.current) return; // bấm đúp: một yêu cầu
    running.current = true;
    setBusy(true);
    setNotice(null);
    try {
      await job();
    } catch (e) {
      fail(e);
    } finally {
      running.current = false;
      setBusy(false);
    }
  }

  const gradeAllNow = () =>
    void run(async () => {
      const out = await gradeAll(courseId, assignmentId, newIdempotencyKey());
      if (out.job_id && out.queued > 0) setJob({ id: out.job_id, queued: out.queued });
      setNotice(out.queued > 0 ? `Đã xếp hàng chấm ${out.queued} bài.` : "Không có bài nào cần chấm.");
      refresh();
    });

  const upload = (file: File) => {
    const bad = checkSubmissionBundle(file);
    if (bad) {
      setNotice(bad);
      return;
    }
    void run(async () => {
      const uploadId = await uploadSubmissionBundle(courseId, file);
      const out = await importSubmissions(courseId, assignmentId, uploadId, newIdempotencyKey());
      setNotice(out.job_id ? "Đang nhận bài từ tệp ZIP." : `Đã nhận ${out.created ?? 0} bài · khớp ${out.matched ?? 0} · chưa khớp ${out.unmatched ?? 0}.`);
      refresh();
    });
  };

  const needAssignment = assignmentId === "" ? "Chọn một bài tập trước." : undefined;
  const noneToGrade = !needAssignment && !list.isPending && !list.isError && rows.length === 0 ? "Chưa có bài nộp." : undefined; // hàng chờ rỗng: không có gì để chấm (US-UX91-03 AC2)

  // ---- công bố hàng loạt (US-P7-09 AC1): chỉ TEACHER, chỉ hàng ĐÃ DUYỆT ---------------------------------------------------
  const reviewable = rows.filter((r) => r.result?.status === "REVIEWED");
  const chosen = reviewable.filter((r) => picked.has(r.id));
  const schemes = useSchemes(courseId, false, teacher && chosen.length > 0);
  const assignmentIds = new Set(chosen.map((r) => r.assignment.id));
  const scheme = schemes.data?.confirmed ?? schemes.data?.draft ?? null;
  const flagText = (r: QueueRow) => (r.result?.flags ?? []).map((f) => FLAG_TEXT[f] ?? f).join(" · ");
  // một bài tập (đang chọn, hoặc lớp chỉ có một): bỏ cột "Bài tập", tên bài hiện một lần trên bảng (US-UX91-14 AC1)
  const oneAssignment = assignmentId !== "" || assignments.items.length === 1;
  const tableTitle = oneAssignment ? (assignments.items.find((a) => a.id === assignmentId) ?? assignments.items[0])?.title : undefined;
  const column =
    assignmentIds.size === 1
      ? scheme?.components.flatMap((c) => c.items).find((i) => i.assignment_id && assignmentIds.has(i.assignment_id))?.name
      : undefined;
  const students = new Set(chosen.map((r) => r.student?.id).filter(Boolean)).size;
  const consequence = `Công bố ${chosen.length} bài · ${students} sinh viên nhận thư · ${column ? `điểm vào sổ điểm cột ${column}` : "không tính vào sổ điểm"}`;

  const publishNow = () =>
    void run(async () => {
      publishIdem.current ??= newIdempotencyKey();
      const versions: Record<string, number> = {};
      for (const r of chosen) if (r.result) versions[r.result.id] = r.result.version;
      try {
        await publishResults(courseId, { result_ids: chosen.map((r) => r.result!.id), versions }, publishIdem.current);
      } catch (e) {
        if (e instanceof ApiError && (e.code === "NOT_REVIEWED" || e.code === "VERSION_CONFLICT" || e.code === "LATE_RECOMPUTED" || e.code === "ALREADY_PUBLISHED")) {
          const ids = (e.details as { ids?: string[] } | undefined)?.ids ?? [];
          setConfirming(false);
          setPicked(new Set());
          publishIdem.current = null;
          refresh();
          setNotice(`${e.userMessage}${ids.length > 0 ? ` ${ids.length} bài cần xem lại.` : ""}`);
          return;
        }
        throw e;
      }
      publishIdem.current = null;
      setConfirming(false);
      setPicked(new Set());
      refresh();
    });

  const allPicked = reviewable.length > 0 && reviewable.every((r) => picked.has(r.id));
  const pickCol: Column<QueueRow> = {
    key: "pick",
    header: (
      <input
        type="checkbox"
        aria-label="Chọn tất cả đã duyệt"
        checked={allPicked}
        onChange={() => setPicked(allPicked ? new Set() : new Set(reviewable.map((r) => r.id)))}
      />
    ),
    width: "44px",
    frozen: true,
    render: (r) =>
      r.result?.status === "REVIEWED" ? (
        <input
          type="checkbox"
          aria-label={`Chọn bài của ${r.student?.full_name ?? "sinh viên"}`}
          checked={picked.has(r.id)}
          onClick={(ev) => ev.stopPropagation()}
          onChange={() => setPicked((p) => { const n = new Set(p); if (n.has(r.id)) n.delete(r.id); else n.add(r.id); return n; })}
        />
      ) : null,
  };

  const columns: Column<QueueRow>[] = [
    {
      key: "student",
      header: "Sinh viên",
      frozen: true,
      width: "220px",
      primary: true,
      render: (r) => (
        <span className={s.who}>
          <span className={s.name}>{r.student?.full_name ?? "Chưa khớp"}</span>
          <span className={s.code}>{r.student ? `${r.student.student_code}${r.student.inactive ? " · không còn trong lớp" : ""}` : ""}</span>
          {flagText(r) && <span className={s.flag} data-part="flag-reason">{flagText(r)}</span>}
        </span>
      ),
    },
    ...(oneAssignment ? [] : [{ key: "assignment", header: "Bài tập", width: "200px", render: (r: QueueRow) => <span className={s.clip} title={r.assignment.title}>{r.assignment.title}</span> }]),
    { key: "source", header: "Nguồn", width: "130px", render: (r) => SOURCE_LABEL[r.source] ?? r.source },
    {
      key: "score",
      header: "Điểm",
      align: "end",
      width: "120px",
      render: (r) => (
        <span className={s.score}>
          {r.result?.total ? vi(r.result.total) : "—"}
          {r.result && r.result.status !== "DRAFT" && <span className={s.sub}> đã duyệt</span>}
        </span>
      ),
    },
    {
      key: "state",
      header: "Trạng thái duyệt",
      width: "150px",
      part: "col-state",
      render: (r) => {
        const text = reviewStateText(r.result);
        return <StatusText tone={reviewTone(text)}>{text}</StatusText>;
      },
    },
    {
      key: "at",
      header: "Thời gian nộp",
      width: "180px",
      render: (r) => (
        <span className={s.sub}>
          {fmtLong(r.received_at)}
          {r.late_days > 0 ? ` · muộn ${r.late_days} ngày` : ""}
        </span>
      ),
    },
    {
      key: "open",
      header: <span className="ep-sr-only">Duyệt</span>,
      align: "end",
      width: "110px",
      render: (r) => (
        <Link href={reviewHref(r.id, filter)} className={s.open} onClick={(e) => e.stopPropagation()} aria-label={`Duyệt bài của ${r.student?.full_name ?? "sinh viên"}`}>
          Duyệt <ArrowRight aria-hidden />
        </Link>
      ),
    },
  ];

  return (
    <Panel>
        <Toolbar
          alignEnd
          end={
            <>
              <Button icon={<Upload aria-hidden />} disabled={Boolean(needAssignment) || busy} disabledReason={needAssignment} onClick={() => picker.current?.click()}>
                Tải bài lên
              </Button>
              <Button variant="primary" loading={busy} disabled={Boolean(needAssignment || noneToGrade)} disabledReason={needAssignment ?? noneToGrade} onClick={gradeAllNow}>
                Chấm tất cả
              </Button>
              {teacher && chosen.length > 0 && (
                <Button variant="primary" onClick={() => setConfirming(true)}>
                  Công bố
                </Button>
              )}
              {teamsChannel && (
                <Button
                  disabled={busy || teamsRunning || teamsWait > 0}
                  title={teamsWait > 0 ? `Chờ ${teamsWait} giây nữa` : undefined}
                  onClick={teamsSync}
                >
                  Đồng bộ từ Teams
                </Button>
              )}
            </>
          }
        >
          <Field label="Bài tập" className={s.filterField}>
            {(id) => (
              <Select id={id} value={assignmentId} onChange={(e) => setAssignmentId(e.target.value)}>
                <option value="">Tất cả bài tập</option>
                {assignments.items.map((a) => (
                  <option key={a.id} value={a.id}>
                    {a.title}
                  </option>
                ))}
              </Select>
            )}
          </Field>
          <Field label="Trạng thái" className={s.filterField}>
            {(id) => (
              <Select id={id} value={status} onChange={(e) => setStatus(e.target.value as QueueStatus)}>
                {STATUS_OPTIONS.map((o) => (
                  <option key={o.value} value={o.value}>
                    {o.label}
                  </option>
                ))}
              </Select>
            )}
          </Field>
          <FilterChips
            label="Lọc hàng chờ chấm"
            value={flagOnly ? ["flag"] : []}
            onChange={(next) => {
              relaxed.current = true;
              setFlagOnly(next.includes("flag"));
            }}
            options={[{ value: "flag", label: "Cần xem kỹ" }]}
          />
        </Toolbar>

        <input
          ref={picker}
          type="file"
          className={s.picker}
          accept=".zip,.pdf,.docx,.txt,.md"
          aria-label="Chọn tệp bài nộp để tải lên"
          onChange={(e) => {
            const f = e.target.files?.[0];
            e.target.value = "";
            if (f) upload(f);
          }}
        />
        {notice && (
          <InlineNotice tone="info" compact action={<Button size="sm" onClick={() => setNotice(null)}>Đã hiểu</Button>}>
            {notice}
          </InlineNotice>
        )}
        {teamsRunning && (
          <p className={s.progressText} data-part="teams-progress">
            {teams?.total ? `Đã lấy ${teams.fetched}/${teams.total} bài` : "Đang lấy bài từ Teams"}
          </p>
        )}
        {teamsState.status === "SUCCEEDED" && teams && (
          <p className={s.progressText} data-part="teams-result">
            {`Đã lấy ${teams.fetched} bài · tạo ${teams.created ?? 0} · trùng ${teams.deduped ?? 0} · chưa khớp ${teams.unmatched ?? 0}`}
          </p>
        )}
        {teamsState.status === "FAILED" && (
          <InlineNotice tone="warning" compact action={<Button size="sm" onClick={teamsSync}>Thử lại</Button>}>
            <span data-part="teams-error">{teamsState.error || "Chưa lấy được bài từ Teams."}</span>
          </InlineNotice>
        )}
        {teamsFailed.length > 0 && (
          <div data-part="teams-failed">
            <p className={s.progressText}>Không lấy được</p>
            <ul className={s.failedList}>
              {teamsFailed.map((f) => (
                <li key={f.graph_submission_id}>{`${f.graph_submission_id} — ${TEAMS_FAIL_TEXT[f.code] ?? "Chưa lấy được"}`}</li>
              ))}
            </ul>
            <Button size="sm" onClick={teamsSync}>
              Thử lại
            </Button>
          </div>
        )}
    <PageState
      query={{ isPending: list.isPending, isError: list.isError, error: list.error, data: rows, refetch: list.refetch }}
      isEmpty={(d) => (d as QueueRow[]).length === 0}
      loading={<Skeleton lines={10} />}
      empty={
        <EmptyState
          title="Chưa có bài nộp."
          icon={<ListChecks aria-hidden />}
          action={teacher ? <Button icon={<Upload aria-hidden />} disabled={Boolean(needAssignment) || busy} disabledReason={needAssignment} onClick={() => picker.current?.click()}>Tải bài lên</Button> : undefined}
        >
          Sinh viên nộp trong ứng dụng, hoặc thầy/cô tải ZIP lên.
        </EmptyState>
      }
      showTechnical
    >
      <>
        {batchRunning && batchTotal > 0 && (
          <p className={s.progressText} data-part="grading-progress">{`Đang chấm ${batchDone}/${batchTotal} bài`}</p>
        )}

        {tableTitle && <h3 className="ep-item-title" data-part="queue-title">{tableTitle}</h3>}
        <DataTable
          caption="Hàng chờ chấm bài"
          columns={teacher && reviewable.length > 0 ? [pickCol, ...columns] : columns}
          rows={rows}
          rowKey={(r) => r.id}
          onRowClick={(r) => router.push(reviewHref(r.id, filter))}
          rowHref={(r) => reviewHref(r.id, filter)} /* ở danh sách di động cả hàng là MỘT liên kết (không liên kết lồng liên kết) */
          rowAttrs={(r) => ({ "data-part": "queue-row", "data-submission-id": r.id })}
          virtual={narrow ? undefined : { height: 560 }}
          mobileRow={(r) => (
            <div className={s.mItem} data-part="queue-row" data-submission-id={r.id}>
              <span className={s.name}>{r.student?.full_name ?? "Chưa khớp"}</span>
              <span className={s.code}>{r.student ? r.student.student_code : ""}</span>
              {flagText(r) && <span className={s.flag}>{flagText(r)}</span>}
              <span className={s.mMeta}>
                <span className={s.score}>{r.result?.total ? vi(r.result.total) : "—"}</span>
                <StatusText tone={reviewTone(reviewStateText(r.result))}>{reviewStateText(r.result)}</StatusText>
              </span>
              <span className={s.mOpen}>Duyệt <ArrowRight aria-hidden /></span>
            </div>
          )}
          pagination={{
            nextCursor: list.data?.pages[list.data.pages.length - 1]?.next_cursor ?? null,
            loading: list.isFetchingNextPage,
            onLoadMore: () => void list.fetchNextPage(),
          }}
        />
      </>
    </PageState>
        <ConfirmIrreversible
          open={confirming}
          onClose={() => setConfirming(false)}
          onConfirm={publishNow}
          title="Công bố điểm"
          consequence={consequence}
          confirmLabel="Công bố"
          loading={busy}
        />
    </Panel>
  );
}
