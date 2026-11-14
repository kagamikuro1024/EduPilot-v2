"use client";

import { useQueryClient } from "@tanstack/react-query";
import { useEffect, useRef, useState } from "react";
import { ApiError, newIdempotencyKey, useAutosaveDraft, useJob } from "@/shared/data";
import { Button, InlineNotice, Skeleton, StatusText, Textarea } from "@/shared/ui";
import type { Criterion } from "./assignmentApi";
import {
  DIRECTION_TEXT, applyGuidance, decideGuidance, guidanceKey, reflectGuidance, regradeStale, useGuidance, useGuidanceStats,
  type GuidanceItem,
} from "./guidanceApi";
import s from "./Guidance.module.css";

const fmt = (d: string | null) => (d === null ? "—" : d.replace(".", ","));
const pct = (d: string | null) => (d === null ? "—" : `${Math.round(Number(d) * 100)}%`);

function stateOf(i: GuidanceItem): { text: string; tone: "amber" | "green" | "neutral" | "blue" } {
  if (i.status === "DRAFT") return { text: "Nháp", tone: "amber" };
  if (i.status === "APPROVED" && i.approved_in_version === null) return { text: "Chờ áp dụng", tone: "blue" };
  if (i.status === "APPROVED" && i.pending_retire) return { text: "Chờ nghỉ", tone: "amber" };
  if (i.status === "APPROVED") return { text: `Đang dùng · v${i.approved_in_version}`, tone: "green" };
  return { text: i.status === "REJECTED" ? "Đã bỏ" : "Đã nghỉ", tone: "neutral" };
}

function ItemRow({ courseId, aid, item, teacher, userId, onDecided, onError }: {
  courseId: string; aid: string; item: GuidanceItem; teacher: boolean; userId?: string; onDecided: () => void; onError: (e: unknown) => void;
}) {
  const draft = useAutosaveDraft(`guidance:${aid}:${item.id}`, { userId });
  const [editing, setEditing] = useState(false);
  const [busy, setBusy] = useState(false);
  const [rejected, setRejected] = useState(false);
  const st = stateOf(item);
  const text = draft.value || item.text;
  const isDraft = item.status === "DRAFT";
  const active = item.status === "APPROVED" && item.approved_in_version !== null && !item.pending_retire;

  async function decide(decision: "APPROVE" | "REJECT" | "RETIRE") {
    if (busy) return;
    setBusy(true);
    setRejected(false);
    try {
      await decideGuidance(courseId, aid, item.id, { decision, version: item.version, ...(decision === "APPROVE" && text !== item.text ? { text } : {}) });
      draft.clear();
      setEditing(false);
      onDecided();
    } catch (e) {
      if (e instanceof ApiError && e.code === "GUIDANCE_TEXT_REJECTED") setRejected(true);
      else onError(e);
    } finally {
      setBusy(false);
    }
  }

  return (
    <div className={s.item} data-part="guidance-item" data-status={item.status}>
      <div className={s.itemMain}>
        {editing ? (
          <Textarea aria-label="Chữ lưu ý" value={text} maxLength={400} rows={2} invalid={rejected} onChange={(e) => draft.setValue(e.target.value)} />
        ) : (
          <p className={s.text}>{item.text}</p>
        )}
        {teacher && (
          <div className={s.actions}>
            {isDraft && (
              <>
                <Button size="sm" variant="primary" loading={busy} onClick={() => void decide("APPROVE")}>Duyệt</Button>
                <Button size="sm" variant="ghost" disabled={busy} onClick={() => void decide("REJECT")}>Bỏ</Button>
                <Button size="sm" variant="ghost" onClick={() => setEditing((v) => !v)}>{editing ? "Giữ chữ gốc" : "Sửa"}</Button>
              </>
            )}
            {active && <Button size="sm" variant="ghost" loading={busy} onClick={() => void decide("RETIRE")}>Cho nghỉ</Button>}
          </div>
        )}
      </div>
      {rejected && <InlineNotice tone="danger" compact>Chữ này không dùng được. Bỏ phần nhắc điểm tổng, phạt, làm tròn hoặc tên người.</InlineNotice>}
      <div className={s.meta}>
        <StatusText tone={st.tone}>{st.text}</StatusText>
        <span>{DIRECTION_TEXT[item.direction]}</span>
        <span>{item.support} bài</span>
        {item.needs_recheck && <StatusText tone="amber">Cần xem lại</StatusText>}
      </div>
    </div>
  );
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
  const groups = rubric.map((c) => ({ id: c.id, name: c.name, items: d.items.filter((i) => i.criterion_id === c.id && (i.status === "DRAFT" || i.status === "APPROVED")) })).filter((g) => g.items.length > 0);
  const waiting = d.items.filter((i) => i.status === "APPROVED" && (i.approved_in_version === null || i.pending_retire)).length;
  const reflecting = job.status === "QUEUED" || job.status === "RUNNING" || (reflectJob !== null && job.status === "UNKNOWN");
  const regrading = regrade.status === "QUEUED" || regrade.status === "RUNNING";
  const rows = (stats.data ?? []).filter((r) => r.mae_criterion !== null);

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
      refresh();
    } catch (e) {
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
            <Button size="sm" variant="primary" loading={busy === "apply"} disabled={waiting === 0} disabledReason="Chưa có lưu ý nào chờ áp dụng" onClick={() => void apply()}>
              {waiting > 0 ? `Áp dụng (${waiting})` : "Áp dụng"}
            </Button>
          </div>
        )}
      </div>

      {apiErr && <InlineNotice tone="danger" compact>{apiErr.userMessage}</InlineNotice>}
      {job.status === "FAILED" && <InlineNotice tone="warning" compact>{job.error ?? "Chưa rút được lưu ý."}</InlineNotice>}
      {job.status === "SUCCEEDED" && d.counts.draft === 0 && <InlineNotice tone="info" compact>Không có lưu ý mới đủ chắc.</InlineNotice>}
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

      {groups.map((g) => (
        <div key={g.id} className={s.group}>
          <p className={s.crit}>{names.get(g.id)}</p>
          {g.items.map((i) => (
            <ItemRow key={`${i.id}:${i.version}`} courseId={courseId} aid={aid} item={i} teacher={teacher} userId={userId} onDecided={refresh} onError={setErr} />
          ))}
        </div>
      ))}

      {rows.map((r) => (
        <p key={r.version} className={s.stats} data-part="guidance-stats">
          {`v${r.version} · ${r.n} bài · lệch tiêu chí ${fmt(r.mae_criterion)} · lệch tổng ${fmt(r.mae_total)} · sửa ${pct(r.edited_rate)}`}
        </p>
      ))}
    </div>
  );
}
