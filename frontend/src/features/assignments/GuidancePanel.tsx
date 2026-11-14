"use client";

import { useQueryClient } from "@tanstack/react-query";
import { useEffect, useRef, useState } from "react";
import { ApiError, newIdempotencyKey, useAutosaveDraft, useJob, useUndoableAction } from "@/shared/data";
import { Button, ConfirmIrreversible, InlineNotice, OverflowMenu, Skeleton, StatusText, Textarea } from "@/shared/ui";
import type { Criterion } from "./assignmentApi";
import {
  DIRECTION_TEXT, applyGuidance, decideGuidance, guidanceKey, reflectGuidance, regradeStale, useGuidance, useGuidanceStats,
  type GuidanceItem,
} from "./guidanceApi";
import s from "./Guidance.module.css";

const fmt = (d: string | null) => (d === null ? "—" : d.replace(".", ","));
const pct = (d: string | null) => (d === null ? "—" : `${Math.round(Number(d) * 100)}%`);

function stateOf(i: GuidanceItem): { text: string; tone: "amber" | "green" | "neutral" | "blue" } {
  if (i.status === "DRAFT") return { text: "Chờ duyệt", tone: "amber" };
  if (i.status === "APPROVED" && i.approved_in_version === null) return { text: "Chờ áp dụng", tone: "blue" };
  if (i.status === "APPROVED" && i.pending_retire) return { text: "Chờ nghỉ", tone: "amber" };
  if (i.status === "APPROVED") return { text: `Đang dùng · v${i.approved_in_version}`, tone: "green" };
  return { text: i.status === "REJECTED" ? "Đã bỏ" : "Đã nghỉ", tone: "neutral" };
}

type Act = "APPROVE" | "REJECT" | "RETIRE";
const DONE_LABEL: Record<Act, string> = { APPROVE: "Đã duyệt lưu ý", REJECT: "Đã bỏ lưu ý", RETIRE: "Đã cho nghỉ lưu ý" };

function ItemRow({ courseId, aid, item, teacher, userId, onDecided, onError }: {
  courseId: string; aid: string; item: GuidanceItem; teacher: boolean; userId?: string; onDecided: (id: string) => void; onError: (e: unknown) => void;
}) {
  const draft = useAutosaveDraft(`guidance:${aid}:${item.id}`, { userId });
  const [editing, setEditing] = useState(false);
  const [rejected, setRejected] = useState(false);
  // lạc quan: trạng thái hiện ngay, hết hiệu lực khi máy chủ trả bản mới của mục (version đổi)
  const [local, setLocal] = useState<{ act: Act; v: number } | null>(null);
  const ver = useRef(item.version); // version mới nhất sau lần ghi gần nhất (để Hoàn tác đúng bản)
  const view: GuidanceItem = local && local.v === item.version
    ? { ...item, status: local.act === "REJECT" ? "REJECTED" : "APPROVED", pending_retire: local.act === "RETIRE", approved_in_version: item.approved_in_version }
    : item;
  const st = stateOf(view);
  const text = draft.value || item.text;
  const isDraft = view.status === "DRAFT";
  const active = view.status === "APPROVED" && view.approved_in_version !== null && !view.pending_retire;

  const undo = useUndoableAction<Act>({
    apply: (act) => setLocal({ act, v: item.version }),
    rollback: () => setLocal(null),
    commit: async (act) => {
      try {
        const out = await decideGuidance(courseId, aid, item.id, { decision: act, version: item.version, ...(act === "APPROVE" && text !== item.text ? { text } : {}) });
        ver.current = out.version;
        draft.clear();
        setEditing(false);
        onDecided(item.id);
      } catch (e) {
        setLocal(null);
        if (e instanceof ApiError && e.code === "GUIDANCE_TEXT_REJECTED") setRejected(true);
        else if (e instanceof ApiError && (e.code === "VERSION_CONFLICT" || e.code === "ITEM_NOT_DRAFT")) onError(e); // mục đã đổi: báo tại panel, tải lại
        throw e;
      }
    },
    compensate: async () => {
      try {
        await decideGuidance(courseId, aid, item.id, { decision: "RESET", version: ver.current });
      } catch (e) {
        if (e instanceof ApiError && e.code === "ITEM_NOT_RESETTABLE") onError(e); // Áp dụng chạy trước ở nơi khác: báo tại panel và tải lại
        throw e;
      }
      onDecided(item.id);
    },
    handled: (e) => e instanceof ApiError && (e.code === "GUIDANCE_TEXT_REJECTED" || e.code === "VERSION_CONFLICT" || e.code === "ITEM_NOT_DRAFT"), // đã báo tại chỗ
    label: (act) => DONE_LABEL[act],
  });
  const busy = undo.pending;
  // «Đặt lại» còn tới khi Áp dụng: chờ áp dụng / đã bỏ / chờ nghỉ (SRS 4.2.9)
  const resettable = (view.status === "APPROVED" && (view.approved_in_version === null || view.pending_retire)) || view.status === "REJECTED";
  async function reset() {
    try {
      await decideGuidance(courseId, aid, item.id, { decision: "RESET", version: local ? ver.current : item.version });
      setLocal(null);
      onDecided(item.id);
    } catch (e) {
      onError(e);
      onDecided(item.id); // tải lại hàng
    }
  }
  const go = (act: Act) => {
    if (busy) return;
    setRejected(false);
    void undo.run(act);
  };

  return (
    <div className={s.item} data-part="guidance-item" data-status={view.status}>
      <div className={s.itemMain}>
        {editing ? (
          <Textarea aria-label="Chữ lưu ý" value={text} maxLength={400} rows={2} invalid={rejected} onChange={(e) => draft.setValue(e.target.value)} />
        ) : (
          <p className={s.text}>{view.text}</p>
        )}
        {teacher && (
          <div className={s.actions}>
            {isDraft && (
              <>
                <Button size="sm" variant="primary" loading={busy} onClick={() => go("APPROVE")}>Duyệt</Button>
                <Button size="sm" variant="ghost" disabled={busy} onClick={() => go("REJECT")}>Bỏ</Button>
                <Button size="sm" variant="ghost" onClick={() => setEditing((v) => !v)}>{editing ? "Giữ chữ gốc" : "Sửa"}</Button>
              </>
            )}
            {active && <Button size="sm" variant="ghost" disabled={busy} onClick={() => go("RETIRE")}>Cho nghỉ</Button>}
            {resettable && <OverflowMenu label="Thêm hành động của lưu ý" items={[{ label: "Đặt lại", onSelect: () => void reset() }]} />}
          </div>
        )}
      </div>
      {rejected && <InlineNotice tone="danger" compact>Chữ này không dùng được. Bỏ phần nhắc điểm tổng, phạt, làm tròn hoặc tên người.</InlineNotice>}
      <div className={s.meta}>
        <StatusText tone={st.tone}>{st.text}</StatusText>
        <span>{DIRECTION_TEXT[view.direction]}</span>
        <span>{view.support} bài đổi điểm</span>
        {view.needs_recheck && <StatusText tone="amber">Cần xem lại</StatusText>}
      </div>
      {undo.node}
    </div>
  );
}

const DROP_WHY: Record<string, string> = {
  keyword: "nhắc tổng điểm, trừ điểm hoặc phạt", pii: "chứa tên hoặc mã", support: "ít bài ủng hộ", case: "dẫn bài không khớp", direction: "sai chiều", criterion: "tiêu chí lạ", overflow: "quá số lượng",
};

/** P-106-12: «N lưu ý bị loại · <tiêu chí>: <lý do>» từ kết quả việc rút lưu ý (chỉ số đếm theo tiêu chí; mục trùng lưu ý đã có không tính). */
function droppedLine(result: unknown, names: Map<string, string>): string | null {
  const by = (result as { dropped?: { by_criterion?: Record<string, Record<string, number>> } } | undefined)?.dropped?.by_criterion;
  if (!by) return null;
  let n = 0;
  const parts: string[] = [];
  for (const [cid, why] of Object.entries(by)) {
    for (const [reason, c] of Object.entries(why)) {
      if (reason === "duplicate") continue;
      n += c;
      parts.push(`${names.get(cid) ?? cid}: ${DROP_WHY[reason] ?? reason}`);
    }
  }
  return n > 0 ? `${n} lưu ý bị loại · ${parts.join("; ")}` : null;
}

/**
 * «Lưu ý chấm» của MỘT bài tập tự luận (FEAT-grading-guidance US-GD-03…06): rút từ lần giảng viên sửa → duyệt → Áp dụng → Chấm lại.
 * Chỉ hiện khi lớp bật tính năng (overview.enabled). TA chỉ xem. Không bao giờ hiện cho sinh viên.
 */
export function GuidancePanel({ courseId, aid, teacher, userId, rubric }: { courseId: string; aid: string; teacher: boolean; userId?: string; rubric: Criterion[] }) {
  const qc = useQueryClient();
  const q = useGuidance(courseId, aid);
  const d = q.data;
  const stats = useGuidanceStats(courseId, aid, Boolean(d?.enabled));
  const [err, setErr] = useState<unknown>(null);
  const [reflectJob, setReflectJob] = useState<string | null>(null);
  const [regradeJob, setRegradeJob] = useState<string | null>(null);
  const [note, setNote] = useState<string | null>(null);
  const [busy, setBusy] = useState<"reflect" | "apply" | "regrade" | null>(null);
  const [confirmApply, setConfirmApply] = useState(false);
  const rkey = useRef<string | null>(null);
  const job = useJob(reflectJob);
  const regrade = useJob(regradeJob);

  const refresh = () => {
    void qc.invalidateQueries({ queryKey: guidanceKey(aid) });
    void qc.invalidateQueries({ queryKey: ["guidance-stats", aid] });
    void qc.invalidateQueries({ queryKey: ["grading-queue", courseId] });
  };
  useEffect(() => {
    if (job.status === "SUCCEEDED" || job.status === "FAILED") {
      void qc.invalidateQueries({ queryKey: guidanceKey(aid) });
    }
  }, [job.status, qc, aid]);
  useEffect(() => {
    if (regrade.status === "SUCCEEDED" || regrade.status === "FAILED") {
      void qc.invalidateQueries({ queryKey: guidanceKey(aid) });
      void qc.invalidateQueries({ queryKey: ["grading-queue", courseId] });
    }
  }, [regrade.status, qc, aid, courseId]);

  if (q.isPending) return <Skeleton lines={2} />;
  if (!d || !d.enabled) return null;

  const names = new Map(rubric.map((c) => [c.id, c.name]));
  const groups = rubric.map((c) => ({ id: c.id, name: c.name, items: d.items.filter((i) => i.criterion_id === c.id && (i.status === "DRAFT" || i.status === "APPROVED" || i.status === "REJECTED")) })).filter((g) => g.items.length > 0);
  const adding = d.items.filter((i) => i.status === "APPROVED" && i.approved_in_version === null).length;
  const retiring = d.items.filter((i) => i.status === "APPROVED" && i.pending_retire).length;
  const waiting = d.items.filter((i) => i.status === "APPROVED" && (i.approved_in_version === null || i.pending_retire)).length;
  const reflecting = job.status === "QUEUED" || job.status === "RUNNING" || (reflectJob !== null && job.status === "UNKNOWN");
  const regrading = regrade.status === "QUEUED" || regrade.status === "RUNNING";
  const rows = (stats.data?.items ?? []).filter((r) => r.mae_criterion !== null);

  async function reflect() {
    setErr(null);
    setNote(null);
    setBusy("reflect");
    rkey.current ??= newIdempotencyKey();
    try {
      const r = await reflectGuidance(courseId, aid, rkey.current);
      rkey.current = null;
      setReflectJob(r.job_id);
    } catch (e) {
      setErr(e);
    } finally {
      setBusy(null);
    }
  }
  async function apply() {
    setErr(null);
    setNote(null);
    setBusy("apply");
    try {
      const r = await applyGuidance(courseId, aid, d!.current_version, newIdempotencyKey());
      setNote(`Đã áp dụng bản v${r.version}`);
      setConfirmApply(false);
      refresh();
    } catch (e) {
      setConfirmApply(false);
      setErr(e);
      refresh();
    } finally {
      setBusy(null);
    }
  }
  async function doRegrade() {
    setErr(null);
    setNote(null);
    setBusy("regrade");
    try {
      const r = await regradeStale(courseId, aid, newIdempotencyKey());
      if (r.job_id) setRegradeJob(r.job_id);
      else setNote("Không có bài nào cần chấm lại");
      refresh();
    } catch (e) {
      setErr(e);
    } finally {
      setBusy(null);
    }
  }

  const apiErr = err instanceof ApiError ? err : null;

  return (
    <div className={s.wrap} data-part="guidance-panel">
      <div className={s.head}>
        <h3 className={s.title}>
          Lưu ý chấm
          <StatusText tone="neutral">{d.current_version === 0 ? "Chưa có bản" : `Bản v${d.current_version}`}</StatusText>
        </h3>
        {teacher && (
          <div className={s.actions}>
            <Button
              size="sm"
              loading={busy === "reflect" || reflecting}
              disabled={!d.reflect.eligible}
              disabledReason={`Mới có ${d.reflect.results} bài đã duyệt, ${d.reflect.edits} lần sửa`}
              onClick={() => void reflect()}
            >
              Rút lưu ý
            </Button>
            <Button size="sm" variant="primary" loading={busy === "apply"} disabled={waiting === 0} disabledReason="Chưa có lưu ý nào chờ áp dụng" onClick={() => setConfirmApply(true)}>
              {waiting > 0 ? `Áp dụng (${waiting})` : "Áp dụng"}
            </Button>
          </div>
        )}
      </div>

      {apiErr && <InlineNotice tone="danger" compact>{apiErr.userMessage}</InlineNotice>}
      {job.status === "FAILED" && <InlineNotice tone="warning" compact>{job.error ?? "Chưa rút được lưu ý."}</InlineNotice>}
      {job.status === "SUCCEEDED" && d.counts.draft === 0 && <InlineNotice tone="info" compact>Không có lưu ý mới đủ chắc.</InlineNotice>}
      {job.status === "SUCCEEDED" && droppedLine(job.result, names) && <p role="status" className={s.stats} data-part="guidance-dropped">{droppedLine(job.result, names)}</p>}
      {note && <p role="status" className={s.stats}>{note}</p>}

      {d.stale.count + d.stale.editing > 0 && (
        <InlineNotice
          tone="warning"
          compact
          action={
            d.stale.count > 0 ? (
              <Button size="sm" loading={busy === "regrade" || regrading} onClick={() => void doRegrade()}>Chấm lại</Button>
            ) : undefined
          }
        >
          {[d.stale.count > 0 && `${d.stale.count} bài chấm theo bản cũ`, d.stale.editing > 0 && `${d.stale.editing} đang sửa dở`].filter(Boolean).join(" · ")}
        </InlineNotice>
      )}

      <ConfirmIrreversible
        open={confirmApply}
        onClose={() => setConfirmApply(false)}
        onConfirm={() => void apply()}
        title={`Áp dụng bản v${d.current_version + 1}?`}
        consequence={`Thêm ${adding} lưu ý, cho nghỉ ${retiring}. Bài chấm sau này dùng bản mới; bài đã chấm giữ nguyên.`}
        confirmLabel="Áp dụng"
        loading={busy === "apply"}
      />

      {groups.map((g) => (
        <div key={g.id} className={s.group}>
          <p className={s.crit}>{names.get(g.id)}</p>
          {g.items.map((i) => (
            <ItemRow key={i.id} courseId={courseId} aid={aid} item={i} teacher={teacher} userId={userId} onDecided={refresh} onError={setErr} />
          ))}
        </div>
      ))}

      {rows.map((r) => (
        <p key={r.version} className={s.stats} data-part="guidance-stats">
          {`Bản v${r.version}${r.version === d.current_version ? " (đang dùng)" : ""} · ${r.n} bài đã duyệt · lệch tiêu chí ${fmt(r.mae_criterion)} · lệch tổng ${fmt(r.mae_total)} · đổi điểm ${pct(r.edited_rate)}`}
        </p>
      ))}
      {d.current_version > 0 && !rows.some((r) => r.version === d.current_version) && (
        <p className={s.stats} data-part="guidance-stats">{`Bản v${d.current_version} (đang dùng) · chưa đủ bài đã duyệt để đo`}</p>
      )}
      {(stats.data?.regression_cases ?? 0) > 0 && <p className={s.stats} data-part="guidance-cases">{`Ca kiểm thử ${stats.data?.regression_cases}`}</p>}
    </div>
  );
}
