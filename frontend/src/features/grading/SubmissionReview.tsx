"use client";

import { useQueryClient } from "@tanstack/react-query";
import { useRouter, useSearchParams } from "next/navigation";
import { Check, ChevronLeft, ChevronRight, FileSearch, Quote } from "lucide-react";
import { useCallback, useEffect, useLayoutEffect, useMemo, useRef, useState } from "react";
import { ApiError, useAutosaveDraft, useUndoableAction } from "@/shared/data";
import { Button, ButtonLink, Field, InlineNotice, Input, Kbd, Page, PageHeader, PageState, Panel, Skeleton, StatusText, Textarea } from "@/shared/ui";
import { useSession } from "@/shared/session/session";
import { fromUnits, toUnits, vi } from "@/features/assignments/assignmentApi";
import { fmtLong } from "@/features/calendar/calendarApi";
import { submissionFileUrl } from "./submissionApi";
import {
  FLAG_TEXT, FLAG_TODO, editResult, resultKey, reviewResult, useQueue, useResult,
  type Criterion, type QueueStatus, type ResultView,
} from "./gradingApi";
import { AppealPanel } from "./AppealPanel";
import { GuidanceMeta } from "./GuidanceMeta";
import s from "./SubmissionReview.module.css";

type Draft = { overall: string; comments: Record<string, string> };

/** Đoạn văn kèm vị trí trong `extracted_text` (offset UTF-16 như máy chủ). */
function paragraphs(text: string) {
  const out: Array<{ start: number; end: number; text: string }> = [];
  const re = /\n{2,}/g;
  let i = 0;
  let m: RegExpExecArray | null;
  while ((m = re.exec(text)) !== null) {
    out.push({ start: i, end: m.index, text: text.slice(i, m.index) });
    i = m.index + m[0].length;
  }
  out.push({ start: i, end: text.length, text: text.slice(i) });
  return out.filter((p) => p.text.trim() !== "");
}

type Mark = { start: number; end: number; kind: "injection" | "evidence" };

/** Cắt đoạn thành khúc theo các vùng tô (giao nhau thì ưu tiên vùng bằng chứng). */
function segments(p: { start: number; end: number; text: string }, marks: Mark[]) {
  const inside = marks.filter((m) => m.end > p.start && m.start < p.end);
  const points = new Set<number>([p.start, p.end]);
  for (const m of inside) {
    points.add(Math.max(p.start, m.start));
    points.add(Math.min(p.end, m.end));
  }
  const sorted = [...points].sort((a, b) => a - b);
  const out: Array<{ text: string; kind: Mark["kind"] | null }> = [];
  for (let i = 0; i < sorted.length - 1; i++) {
    const [a, b] = [sorted[i], sorted[i + 1]];
    const hit = inside.find((m) => m.start <= a && m.end >= b);
    out.push({ text: p.text.slice(a - p.start, b - p.start), kind: hit?.kind ?? null });
  }
  return out;
}

export function SubmissionReview({ submissionId }: { submissionId: string }) {
  const { realCourseId, realPending, role } = useSession();
  if (realPending || !realCourseId || realCourseId === "all" || (role !== "teacher" && role !== "ta")) return null;
  return <Review key={`${realCourseId}:${submissionId}`} courseId={realCourseId} submissionId={submissionId} />;
}

function Review({ courseId, submissionId }: { courseId: string; submissionId: string }) {
  const q = useResult(courseId, submissionId);
  const view = q.data ?? null;
  return (
    <Page width="full">
      <PageHeader
        title={view ? view.assignment.title : "Duyệt bài"}
        back={{ href: "/grading", label: "Hàng chờ chấm" }}
        meta={
          view ? (
            <>
              <span>{`Phiên bản ${view.submission.version}`}</span>
              <span>
                {`Nộp ${fmtLong(view.submission.received_at)}`}
                {view.submission.late_days > 0 ? ` · muộn ${view.submission.late_days} ngày` : ""}
              </span>
              <StatusText tone={view.result.status === "PUBLISHED" ? "green" : view.result.status === "REVIEWED" ? "blue" : "amber"}>
                {view.result.status === "PUBLISHED" ? "Đã công bố" : view.result.status === "REVIEWED" ? "Đã duyệt" : "Chưa duyệt"}
              </StatusText>
              <GuidanceMeta courseId={courseId} view={view} />
            </>
          ) : undefined
        }
      />
      {view ? (
        <Body courseId={courseId} submissionId={submissionId} view={view} />
      ) : (
        <Panel>
          <PageState query={{ isPending: q.isPending, isError: q.isError, error: q.error, data: q.data, refetch: q.refetch }} loading={<Skeleton lines={14} />} showTechnical>
            {null}
          </PageState>
        </Panel>
      )}
    </Page>
  );
}

function Body({ courseId, submissionId, view }: { courseId: string; submissionId: string; view: ResultView }) {
  const qc = useQueryClient();
  const router = useRouter();
  const params = useSearchParams();
  const { identity } = useSession();
  const rid = view.result.id;

  // hàng chờ theo đúng bộ lọc đang xem (liên kết từ tab mang sẵn) → đi bài kế bằng phím
  const queue = useQueue(courseId, {
    assignment_id: params.get("assignment") || undefined,
    status: (params.get("status") as QueueStatus | null) ?? "DRAFT",
    needs_review: params.get("needs") === "1",
  });
  const order = queue.items.map((r) => r.id);
  const at = order.indexOf(submissionId);
  // N = số bài đã duyệt, M = tổng cố định của hàng chờ lúc mở: mang theo trên URL để không mất khi sang bài kế (US-UX91-15 AC3, P-91-10)
  const done = Number(params.get("n")) || 0;
  const total = Number(params.get("m")) || order.length;
  const [footErr, setFootErr] = useState<string | null>(null);
  const panelRef = useRef<HTMLDivElement>(null);
  // cột phải cao vừa phần khung nhìn còn lại sau đầu trang: thanh dưới luôn thấy khi mở bài (AC3)
  useLayoutEffect(() => {
    const el = panelRef.current;
    if (!el) return;
    const fit = () => {
      const wide = window.matchMedia("(min-width: 1100px)").matches;
      el.style.maxHeight = wide ? `${Math.max(320, window.innerHeight - (el.getBoundingClientRect().top + window.scrollY) - 56)}px` : "";
    };
    fit();
    window.addEventListener("resize", fit);
    return () => window.removeEventListener("resize", fit);
  }, []);

  const draft = useAutosaveDraft(`grading:${rid}`, { userId: identity?.sub });
  const restored = useMemo<Draft | null>(() => {
    try {
      return draft.value ? (JSON.parse(draft.value) as Draft) : null;
    } catch {
      return null;
    }
  }, [draft.value]);

  const [comments, setComments] = useState<Record<string, string>>({});
  const [overall, setOverall] = useState<string | null>(null);
  const [scores, setScores] = useState<Record<string, string>>({});
  const [lateText, setLateText] = useState<string | null>(null);
  const [fieldErr, setFieldErr] = useState<{ id: string; message: string } | null>(null);
  const [notice, setNotice] = useState<string | null>(null);
  /** 409: bản hiện tại của người kia + điểm TÔI đã gõ cho tiêu chí đang lưu (giữ nguyên trong ô cho tới khi chọn). */
  const [conflict, setConflict] = useState<{ theirs: ResultView; id: string; mine: string } | null>(null);
  const [undone, setUndone] = useState(false);
  const [activeEv, setActiveEv] = useState<{ start: number; end: number } | null>(null);
  const [fading, setFading] = useState(false);
  const [tab, setTab] = useState<"doc" | "grade">("grade");
  const [savedKey, setSavedKey] = useState<string | null>(null); // ô vừa lưu xong: nháy nền xanh nhạt 200 ms (US-UX91-04 AC7)
  const flashTimer = useRef<ReturnType<typeof setTimeout> | undefined>(undefined);
  const evTimers = useRef<Array<ReturnType<typeof setTimeout>>>([]);
  const flashSaved = (key: string) => {
    setSavedKey(key);
    clearTimeout(flashTimer.current);
    flashTimer.current = setTimeout(() => setSavedKey(null), 400);
  };
  const [activeCrit, setActiveCrit] = useState<string | null>(null);
  const [busy, setBusy] = useState(false);
  const running = useRef(false);
  const docRef = useRef<HTMLDivElement>(null);

  const commentOf = (c: Criterion) => comments[c.id] ?? restored?.comments?.[c.id] ?? c.comment;
  const overallText = overall ?? restored?.overall ?? view.result.overall_feedback;
  const scoreOf = (c: Criterion) => scores[c.id] ?? (c.score === null ? "" : vi(c.score));
  const lateValue = lateText ?? vi(view.late.penalty);

  const saveDraft = (next: Partial<Draft>) => {
    const base: Draft = { overall: overallText, comments: Object.fromEntries(view.criteria.map((c) => [c.id, commentOf(c)])) };
    draft.setValue(JSON.stringify({ ...base, ...next, comments: { ...base.comments, ...next.comments } }));
  };

  const put = useCallback(
    async (body: Parameters<typeof editResult>[2]) => {
      const out = await editResult(courseId, rid, body);
      qc.setQueryData(resultKey(courseId, submissionId), out);
      return out;
    },
    [courseId, rid, qc, submissionId],
  );

  function handle(e: unknown, critId?: string, mine?: string) {
    if (e instanceof ApiError && e.code === "VERSION_CONFLICT") {
      const theirs = (e.details as { current?: ResultView } | undefined)?.current ?? e.conflict?.current as ResultView | undefined;
      if (theirs && critId && mine !== undefined) setConflict({ theirs, id: critId, mine });
      else void qc.invalidateQueries({ queryKey: resultKey(courseId, submissionId) });
      return;
    }
    if (e instanceof ApiError && critId && (e.code === "SCORE_STEP" || e.code === "SCORE_OUT_OF_RANGE" || e.code === "SCORE_PRECISION")) {
      setFieldErr({ id: critId, message: e.userMessage });
      return;
    }
    setNotice(e instanceof ApiError ? e.userMessage : "Chưa lưu được. Hãy thử lại.");
  }

  // Sửa điểm: lưu lạc quan + dòng "Đã sửa điểm · Hoàn tác" 5 giây (hoàn tác = ghi lại giá trị cũ).
  const undo = useUndoableAction<{ id: string; before: string; after: string }>({
    apply: ({ id, after }) => setScores((p) => ({ ...p, [id]: after })),
    rollback: ({ id, before }) => setScores((p) => ({ ...p, [id]: before })),
    commit: async ({ id, after }) => {
      setFieldErr(null);
      try {
        await put({ version: view.result.version, criteria: [{ id, score: after.replace(",", ".") }] });
        flashSaved(`score:${id}`);
      } catch (e) {
        handle(e, id, after);
        throw e;
      }
    },
    handled: (e) => e instanceof ApiError && e.code === "VERSION_CONFLICT",
    compensate: async ({ id, before }) => {
      const cur = qc.getQueryData<ResultView>(resultKey(courseId, submissionId)) ?? view;
      await put({ version: cur.result.version, criteria: [{ id, score: before.replace(",", ".") }] });
      setUndone(true);
      setTimeout(() => setUndone(false), 5000);
    },
    label: () => "Đã sửa điểm",
    itemKey: ({ id }) => id,
  });

  function commitScore(c: Criterion) {
    const next = scoreOf(c).trim();
    const was = c.score === null ? "" : vi(c.score);
    if (next === was || next === "") return;
    undo.run({ id: c.id, before: was, after: next });
  }

  async function run(job: () => Promise<void>) {
    if (running.current) return;
    running.current = true;
    setBusy(true);
    try {
      await job();
    } catch (e) {
      handle(e);
    } finally {
      running.current = false;
      setBusy(false);
    }
  }

  const dirtyComments = view.criteria.filter((c) => commentOf(c) !== c.comment).map((c) => ({ id: c.id, comment: commentOf(c) }));
  const overallDirty = overallText !== view.result.overall_feedback;

  const goto = (i: number) => {
    const id = order[i];
    if (id) router.push(`/grading/${id}?${params.toString()}`);
  };

  const lockedReason = view.submission.superseded
    ? "Sinh viên vừa nộp lại — xem phiên bản mới"
    : view.result.run_state === "QUEUED" || view.result.run_state === "RUNNING"
      ? "Đang chấm — chưa duyệt được"
      : undefined;

  const approve = () =>
    void run(async () => {
      try {
        let cur = qc.getQueryData<ResultView>(resultKey(courseId, submissionId)) ?? view;
        if (dirtyComments.length > 0 || overallDirty) {
          cur = await put({ version: cur.result.version, criteria: dirtyComments, overall_feedback: overallText });
        }
        await reviewResult(courseId, rid, cur.result.version);
      } catch (e) {
        if (e instanceof ApiError && e.code === "VERSION_CONFLICT") throw e;
        setFootErr(e instanceof ApiError ? e.userMessage : "Chưa duyệt được.");
        return;
      }
      setFootErr(null);
      draft.clear();
      void qc.invalidateQueries({ queryKey: ["grading-queue", courseId] });
      const next = new URLSearchParams(params);
      next.set("n", String(done + 1));
      next.set("m", String(total));
      if (at >= 0 && at + 1 < order.length) router.push(`/grading/${order[at + 1]}?${next}`);
      else {
        router.replace(`/grading/${submissionId}?${next}`);
        void qc.invalidateQueries({ queryKey: resultKey(courseId, submissionId) });
      }
    });

  // Phím tắt (không chạy khi đang gõ trong ô): A duyệt rồi sang bài kế, ←/→ đổi bài, ↑/↓ đổi tiêu chí, Esc hoàn tác ô.
  useEffect(() => {
    const onKey = (e: KeyboardEvent) => {
      const el = e.target as HTMLElement | null;
      const typing = el && (el.tagName === "INPUT" || el.tagName === "TEXTAREA" || el.isContentEditable);
      if (e.key === "Escape" && typing) {
        (el as HTMLInputElement).blur();
        return;
      }
      if (typing || e.metaKey || e.ctrlKey || e.altKey) return;
      const i = view.criteria.findIndex((c) => c.id === activeCrit);
      if (e.key === "a" || e.key === "A") {
        if (!lockedReason) approve();
      } else if (e.key === "ArrowRight") goto(at + 1);
      else if (e.key === "ArrowLeft") goto(at - 1);
      else if (e.key === "ArrowDown") setActiveCrit(view.criteria[Math.min(view.criteria.length - 1, i + 1)]?.id ?? null);
      else if (e.key === "ArrowUp") setActiveCrit(view.criteria[Math.max(0, i - 1)]?.id ?? null);
      else return;
      e.preventDefault();
    };
    window.addEventListener("keydown", onKey);
    return () => window.removeEventListener("keydown", onKey);
  });

  const paras = useMemo(() => paragraphs(view.submission.extracted_text), [view.submission.extracted_text]);
  const marks: Mark[] = [
    ...view.submission.flagged_ranges.map((r) => ({ start: r.start, end: r.end, kind: "injection" as const })),
    ...(activeEv ? [{ start: activeEv.start, end: activeEv.end, kind: "evidence" as const }] : []),
  ];

  function showEvidence(ev: { start: number; end: number }) {
    // tô nền giữ ≈ 1 s (trạng thái), rồi mờ đi trong 200 ms, rồi gỡ (US-UX91-15 AC2)
    evTimers.current.forEach(clearTimeout);
    setFading(false);
    setActiveEv(ev);
    evTimers.current = [setTimeout(() => setFading(true), 1000), setTimeout(() => setActiveEv(null), 1200)];
    const idx = paras.findIndex((p) => ev.start >= p.start && ev.start < p.end);
    docRef.current?.querySelector(`[data-para="${idx}"]`)?.scrollIntoView({ block: "center", behavior: "auto" });
  }

  function paraClick(i: number) {
    const p = paras[i];
    const c = view.criteria.find((x) => x.evidence.some((ev) => ev.start < p.end && ev.end > p.start));
    if (c) setActiveCrit(c.id);
  }

  const openFile = () =>
    void run(async () => {
      const url = await submissionFileUrl(courseId, view.submission.id);
      window.open(url, "_blank", "noopener,noreferrer");
    });

  const otherFlags = view.flags.filter((f) => f !== "NEEDS_REVIEW");

  return (
    <div className={s.reviewGrid} data-tab={tab}>
      <div className={s.tabs} role="tablist" aria-label="Chế độ xem">
        <button type="button" role="tab" aria-selected={tab === "doc"} onClick={() => setTab("doc")}>Bài làm</button>
        <button type="button" role="tab" aria-selected={tab === "grade"} onClick={() => setTab("grade")}>Chấm</button>
      </div>
      <div className={s.docCol}>
      <Panel>
        <div className={s.doc} ref={docRef} data-part="submission-text">
          <div className={s.docMeta}>
            <span>{view.submission.file_name}</span>
            <span>{`Phiên bản ${view.submission.version}`}</span>
            <Button size="sm" variant="text" icon={<FileSearch aria-hidden />} onClick={openFile}>
              Mở tệp gốc
            </Button>
          </div>
          {paras.map((p, i) => (
            <p
              key={p.start}
              data-para={i}
              className={s.para}
              onClick={() => paraClick(i)}
            >
              {segments(p, marks).map((seg, k) =>
                seg.kind === null ? (
                  <span key={k}>{seg.text}</span>
                ) : (
                  <mark key={k} className={seg.kind === "injection" ? s.injected : s.evidenceMark} data-fading={seg.kind === "evidence" && fading ? "" : undefined} data-part={seg.kind === "injection" ? "injection" : "evidence-hit"}>
                    {seg.text}
                  </mark>
                ),
              )}
            </p>
          ))}
        </div>
      </Panel>
      </div>

      <div className={s.gradeCol}>
      <Panel>
        <div className={s.panel} ref={panelRef}>
          {view.appeal && <AppealPanel courseId={courseId} submissionId={submissionId} view={view} appeal={view.appeal} teacher={identity?.role === "teacher"} />}
          <div className={s.total} data-part="total">
            <span className={s.totalLabel}>Tổng điểm</span>
            <span className={s.totalValue}>{view.result.total ? vi(view.result.total) : "—"}</span>
          </div>
          <div className={s.lateRow}>
            <Field label={`Trừ nộp muộn${view.late.days > 0 ? ` · ${view.late.days} ngày` : ""}`} error={fieldErr?.id === "__late" ? fieldErr.message : undefined}>
              {(id) => (
                <Input
                  id={id}
                  className={s.scoreInput}
                  inputMode="decimal"
                  value={lateValue}
                  onChange={(e) => setLateText(e.target.value)}
                  onBlur={() =>
                    void run(async () => {
                      if (lateValue === vi(view.late.penalty)) return;
                      setFieldErr(null);
                      try {
                        await put({ version: view.result.version, late_penalty: lateValue.replace(",", ".") });
                      } catch (e) {
                        handle(e, "__late");
                      }
                    })
                  }
                />
              )}
            </Field>
          </div>

          {lockedReason && (
            <span data-part="locked">
            <InlineNotice tone="warning" compact>
              {view.submission.superseded ? (
                <>
                  {lockedReason} · <a href={`/grading/${view.submission.id}`}>Mở bản mới</a>
                </>
              ) : (
                lockedReason
              )}
            </InlineNotice>
            </span>
          )}
          {view.result.run_state === "FAILED" && (
            <InlineNotice tone="warning" compact>
              Chấm lỗi: nhập điểm tay cho mọi tiêu chí rồi duyệt.
            </InlineNotice>
          )}
          {otherFlags.map((f) => (
            <InlineNotice key={f} tone="warning" compact title={FLAG_TEXT[f] ?? f}>
              {FLAG_TODO[f] ?? ""}
            </InlineNotice>
          ))}
          {view.criteria.some((c) => c.evidence.length > 0) && <p className={s.evNote}>Bấm một dòng trích để xem trong bài.</p>}
          {restored && <InlineNotice compact>Đã khôi phục nhận xét bạn đang soạn dở.</InlineNotice>}
          {conflict && (
            <InlineNotice
              tone="warning"
              title={`Người khác vừa nhập ${vi(conflict.theirs.criteria.find((x) => x.id === conflict.id)?.score ?? "") || "—"}`}
              action={
                <>
                  <Button size="sm" variant="primary" onClick={() => void run(async () => { await put({ version: conflict.theirs.result.version, criteria: [{ id: conflict.id, score: conflict.mine.replace(",", ".") }] }); setConflict(null); })}>
                    Giữ của tôi
                  </Button>
                  <Button size="sm" onClick={() => { qc.setQueryData(resultKey(courseId, submissionId), conflict.theirs); setScores((p) => { const n = { ...p }; delete n[conflict.id]; return n; }); setConflict(null); }}>
                    Dùng của họ
                  </Button>
                </>
              }
            >
              {`Bạn đã nhập ${conflict.mine}.`}
            </InlineNotice>
          )}
          {notice && (
            <InlineNotice tone="danger" compact action={<Button size="sm" onClick={() => setNotice(null)}>Đã hiểu</Button>}>
              {notice}
            </InlineNotice>
          )}
          {undo.node}
          {undone && (
            <p className={s.sub} role="status">
              Đã hoàn tác
            </p>
          )}

          {view.criteria.map((c) => {
            const pd = view.pass_detail?.[c.id];
            const u1 = pd?.pass1 ? toUnits(pd.pass1) : null;
            const u2 = pd?.pass2 ? toUnits(pd.pass2) : null;
            const gap = u1 !== null && u2 !== null ? Math.abs(u1 - u2) : 0;
            return (
              <section key={c.id} className={s.crit} data-part="criterion" data-criterion={c.id} data-active={activeCrit === c.id || undefined}>
                <div className={s.critHead}>
                  <span className={s.critName}>{c.name}</span>
                  <span className={s.critScore}>
                      <span className={s.scoreBox}>
                        <Input
                          aria-label={`Điểm ${c.name}`}
                          className={[s.scoreInput, savedKey === `score:${c.id}` ? "ep-saved ep-motion-safe" : ""].join(" ")}
                          inputMode="decimal"
                          data-part="score-input"
                          value={scoreOf(c)}
                          onChange={(e) => setScores((p) => ({ ...p, [c.id]: e.target.value }))}
                          onBlur={() => commitScore(c)}
                          onKeyDown={(e) => {
                            if (e.key === "Enter") (e.target as HTMLInputElement).blur();
                            if (e.key === "Escape") setScores((p) => ({ ...p, [c.id]: c.score === null ? "" : vi(c.score) }));
                          }}
                        />
                        <span className={s.stepMax}>{` / ${vi(c.max)}`}</span>
                      </span>
                  </span>
                </div>
                {fieldErr?.id === c.id && <p className={s.fieldErr} role="alert">{fieldErr.message}</p>}
                {gap > 100 && (
                  <span data-part="disagree">
                    <InlineNotice tone="warning" compact>
                      {`Lệch ${vi(fromUnits(gap))} điểm so với lượt 2 (${vi(pd!.pass1!)} và ${vi(pd!.pass2!)})`}
                    </InlineNotice>
                  </span>
                )}
                {c.edited && <p className={s.aiTag} data-part="edited">Đã sửa bởi giảng viên</p>}
                {c.evidence.map((ev, i) => (
                  <button key={i} type="button" className={s.evidence} data-part="evidence" onClick={() => showEvidence(ev)}>
                    <Quote aria-hidden />
                    <span>{`“${ev.quote}”`}</span>
                  </button>
                ))}
                <Field label="Nhận xét cho sinh viên">
                  {(id) => (
                    <Textarea
                      id={id}
                      rows={2}
                      data-part="criterion-comment"
                      className={savedKey === `comment:${c.id}` ? "ep-saved ep-motion-safe" : undefined}
                      value={commentOf(c)}
                      onChange={(e) => {
                        setComments((p) => ({ ...p, [c.id]: e.target.value }));
                        saveDraft({ comments: { [c.id]: e.target.value } });
                      }}
                      onBlur={() => {
                        if (commentOf(c) !== c.comment) void run(() => put({ version: view.result.version, criteria: [{ id: c.id, comment: commentOf(c) }] }).then(() => flashSaved(`comment:${c.id}`)));
                      }}
                    />
                  )}
                </Field>
              </section>
            );
          })}

          <Field label="Nhận xét chung">
            {(id) => (
              <Textarea
                id={id}
                rows={3}
                data-part="overall-comment"
                value={overallText}
                onChange={(e) => {
                  setOverall(e.target.value);
                  saveDraft({ overall: e.target.value });
                }}
              />
            )}
          </Field>

          <div className={s.foot} data-part="review-bar">
            {footErr && (
              <div className={s.footErr}>
                <InlineNotice tone="danger" compact action={<Button size="sm" onClick={approve}>Thử lại</Button>}>
                  {footErr}
                </InlineNotice>
              </div>
            )}
            <p className={s.footText} data-part="review-progress">{`Đã duyệt ${done}/${total}`}</p>
            <span className={s.footActions}>
              <Button variant="ghost" size="sm" icon={<ChevronLeft aria-hidden />} aria-label="Bài trước" disabled={at <= 0} onClick={() => goto(at - 1)} />
              <Button variant="ghost" size="sm" icon={<ChevronRight aria-hidden />} aria-label="Bài sau" disabled={at < 0 || at + 1 >= order.length} onClick={() => goto(at + 1)} />
              <Button variant="primary" icon={<Check aria-hidden />} loading={busy} disabled={Boolean(lockedReason) || view.result.status === "PUBLISHED"} title={lockedReason} onClick={approve}>
                {view.result.status === "REVIEWED" ? "Đã duyệt" : "Duyệt bài"}
              </Button>
              <Kbd>A</Kbd>
            </span>
          </div>
          <ButtonLink href="/grading" variant="ghost" size="sm">
            Về hàng chờ chấm
          </ButtonLink>
        </div>
      </Panel>
      </div>
    </div>
  );
}
