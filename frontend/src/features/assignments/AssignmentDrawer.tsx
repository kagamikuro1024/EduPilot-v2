"use client";

import { Plus, Sparkles, Trash2, X } from "lucide-react";
import { useRef, useState } from "react";
import { ApiError, fieldErrors, newIdempotencyKey, useAutosaveDraft } from "@/shared/data";
import { fmtLong, fromLocalInput, toLocalInput } from "@/features/calendar/calendarApi";
import { Button, ButtonLink, Checkbox, ConfirmIrreversible, Drawer, Field, IconButton, InlineNotice, Input, Skeleton, StatusText, Switch, Textarea } from "@/shared/ui";
import {
  CHANNELS, CHANNEL_LABEL, createAssignment, deleteAssignment, fromUnits, publishAssignment, putRubric, suggestRubric, sumUnits, toUnits, updateAssignment, useAssignment, vi,
  type Assignment, type AssignmentInput, type Channel, type RubricIn,
} from "./assignmentApi";
import { useAssignmentStats } from "@/features/grading/gradingApi";
import { GuidancePanel } from "./GuidancePanel";
import s from "./Assignments.module.css";

type Row = { id?: string; name: string; max: string; guidance: string };
type Form = {
  title: string; description: string; code: string; due: string; max: string;
  allowLate: boolean; penalty: string; lateDays: string; appeal: string;
  channels: Channel[]; teamsClass: string; teamsAssignment: string; answer: string; rubric: Row[];
};

const RETRY_MS = 3000;
const LOCKED_PUBLISHED = "Đã công bố, không đổi được.";
const LOCKED_RUBRIC = "Bài đã có kết quả chấm, không đổi tiêu chí nữa.";
const LOCKED_LATE = "Đã có điểm công bố, không đổi hạn hoặc quy tắc nộp muộn.";
const MIN_ANSWER = 50;

const blank = (): Form => ({
  title: "", description: "", code: "", due: "", max: "10",
  allowLate: false, penalty: "0", lateDays: "1", appeal: "7",
  channels: ["IN_APP"], teamsClass: "", teamsAssignment: "", answer: "", rubric: [{ name: "", max: "", guidance: "" }],
});

const fromDetail = (a: Assignment): Form => ({
  title: a.title,
  description: a.description,
  code: a.submission_code,
  due: toLocalInput(a.due_at),
  max: vi(a.max_score),
  allowLate: a.allow_late,
  penalty: vi(a.late_penalty_per_day),
  lateDays: String(a.late_max_days || 1),
  appeal: String(a.appeal_days),
  channels: a.channels,
  teamsClass: a.teams_class_id ?? "",
  teamsAssignment: a.teams_assignment_id ?? "",
  answer: a.reference_answer ?? "",
  rubric: a.rubric.length ? a.rubric.map((c) => ({ id: c.id, name: c.name, max: vi(c.max_score), guidance: c.guidance })) : [{ name: "", max: "", guidance: "" }],
});

const rubricOf = (f: Form): RubricIn[] => f.rubric.filter((r) => r.name.trim() !== "" || r.max.trim() !== "").map((r) => ({ id: r.id, name: r.name.trim(), max_score: r.max.replace(",", "."), guidance: r.guidance.trim() }));
const sameRubric = (f: Form, a: Assignment) => JSON.stringify(rubricOf(f).map((r) => [r.name, r.max_score, r.guidance])) === JSON.stringify(a.rubric.map((c) => [c.name, c.max_score, c.guidance]));
const retryable = (e: unknown) => !(e instanceof ApiError) || e.status === 0 || e.status === 408 || e.status === 429 || e.status >= 500;
const sleep = async (ms: number) => {
  const w = Promise.withResolvers<void>();
  setTimeout(w.resolve, ms);
  await w.promise;
};

function bodyOf(f: Form): AssignmentInput {
  const teams = f.channels.includes("TEAMS");
  return {
    title: f.title.trim(),
    description: f.description,
    due_at: f.due ? fromLocalInput(f.due) : "",
    max_score: f.max.replace(",", "."),
    allow_late: f.allowLate,
    late_penalty_per_day: f.allowLate ? (f.penalty || "0").replace(",", ".") : "0",
    late_max_days: f.allowLate ? Number(f.lateDays) || 0 : 0,
    appeal_days: Number(f.appeal) || 0,
    channels: f.channels,
    reference_answer: f.answer.trim() === "" ? null : f.answer,
    teams_class_id: teams ? f.teamsClass : null,
    teams_assignment_id: teams ? f.teamsAssignment : null,
  };
}

/**
 * Tạo / sửa MỘT bài tập (US-P7-02): trường theo AC2, tiêu chí theo AC3, `Gợi ý tiêu chí` (AC4), `Công bố bài` (AC5), xoá (AC7).
 * TA mở chỉ để xem (trừ `Gợi ý tiêu chí`, không lưu). Chữ đã gõ tự lưu vào máy mỗi 2 giây và không bao giờ mất (AC9).
 */
/** Một dòng số của bài ĐÃ CÔNG BỐ (US-P7-09 AC5) — không phải thẻ KPI. */
function Stats({ courseId, id }: { courseId: string; id: string }) {
  const q = useAssignmentStats(courseId, id, true);
  if (!q.data) return null;
  const d = q.data;
  return <p className={s.link} data-part="assignment-stats">{`Đã nộp ${d.submitted} · Đã chấm ${d.graded} · Đã duyệt ${d.reviewed} · Đã công bố ${d.published}`}</p>;
}

export function AssignmentDrawer({ courseId, teacher, id, userId, onClose, onSaved, onDeleted }: {
  courseId: string;
  teacher: boolean;
  /** "new" = tạo mới */
  id: string;
  userId?: string;
  onClose: () => void;
  onSaved: (a: Assignment) => void;
  onDeleted: () => void;
}) {
  const creating = id === "new";
  const q = useAssignment(courseId, creating ? null : id);
  const detail = q.data ?? null;
  const draft = useAutosaveDraft(`assignment:${courseId}:${id}`, { userId });
  const [form, setForm] = useState<Form | null>(creating ? blank() : null);
  const [restored, setRestored] = useState(false);
  const [err, setErr] = useState<unknown>(null);
  const [conflict, setConflict] = useState<Assignment | null>(null);
  const [phase, setPhase] = useState<"idle" | "saving" | "queued">("idle");
  const [confirm, setConfirm] = useState<"publish" | "delete" | null>(null);
  const [init, setInit] = useState<"pending" | "ready">("pending"); // giá trị đầu đã chốt (bản nháp trong máy, hoặc bản của máy chủ)
  const busy = useRef(false);
  const idem = useRef<string | null>(null);
  const stage = useRef<{ server: Assignment | null; fieldsDone: boolean }>({ server: null, fieldsDone: false });

  // `useAutosaveDraft` đọc kho của trình duyệt sau khi gắn kết: bản nháp tới ở lượt dựng sau, và luôn thắng bản của máy chủ.
  if (init === "pending" && draft.value) {
    try {
      setForm(JSON.parse(draft.value) as Form);
      setRestored(true);
      setInit("ready");
    } catch {
      setInit("ready"); // nháp hỏng: giữ bản đang có
    }
  } else if (init === "pending" && form === null && detail) {
    setForm(fromDetail(detail));
    setInit("ready");
  }

  const f = form ?? (detail ? fromDetail(detail) : blank());
  const fe = fieldErrors(err);
  const apiErr = err instanceof ApiError ? err : null;
  const published = Boolean(detail?.published_at);
  const ro = !teacher;
  const rubricLocked = ro || Boolean(detail?.rubric_locked);
  const lateLocked = ro || Boolean(detail?.late_rule_locked);
  const total = sumUnits(rubricOf(f).map((r) => r.max_score));
  const maxUnits = toUnits(f.max);
  const answerOk = f.answer.trim().length >= MIN_ANSWER;

  function patch(next: Partial<Form>) {
    setInit("ready");
    setForm((prev) => {
      const merged = { ...(prev ?? f), ...next };
      draft.setValue(JSON.stringify(merged));
      return merged;
    });
  }
  const setRow = (i: number, p: Partial<Row>) => patch({ rubric: f.rubric.map((r, k) => (k === i ? { ...r, ...p } : r)) });

  async function run(job: () => Promise<void>) {
    if (busy.current) return; // bấm đúp: đúng MỘT lần gửi có hiệu lực
    busy.current = true;
    setErr(null);
    setConflict(null);
    try {
      for (;;) {
        setPhase("saving");
        try {
          await job();
          setPhase("idle");
          return;
        } catch (e) {
          if (e instanceof ApiError && e.code === "VERSION_CONFLICT") {
            const cur = (e.details as { current?: Assignment } | undefined)?.current ?? null;
            setConflict(cur);
            setPhase("idle");
            return;
          }
          if (!retryable(e)) {
            setErr(e);
            setPhase("idle");
            return;
          }
          setPhase("queued"); // mất mạng / máy chủ bận: giữ nguyên chữ, tự gửi lại
          await sleep(RETRY_MS);
        }
      }
    } finally {
      busy.current = false;
    }
  }

  function save(version?: number) {
    stage.current = { server: null, fieldsDone: false };
    void run(async () => {
      const body = bodyOf(f);
      if (creating) {
        idem.current ??= newIdempotencyKey();
        const out = await createAssignment(courseId, { ...body, submission_code: f.code.trim().toUpperCase(), rubric: rubricOf(f) }, idem.current);
        idem.current = null;
        draft.clear();
        onSaved(out);
        return;
      }
      let cur = stage.current.server ?? detail!;
      if (!stage.current.fieldsDone) {
        cur = await updateAssignment(courseId, id, { ...body, submission_code: f.code.trim().toUpperCase(), version: version ?? cur.version });
        stage.current = { server: cur, fieldsDone: true };
      }
      if (!sameRubric(f, cur)) {
        cur = await putRubric(courseId, id, cur.version, rubricOf(f));
        stage.current = { server: cur, fieldsDone: true };
      }
      draft.clear();
      onSaved(cur);
    });
  }

  function publish() {
    void run(async () => {
      idem.current ??= newIdempotencyKey();
      const out = await publishAssignment(courseId, id, idem.current);
      idem.current = null;
      setConfirm(null);
      onSaved(out.assignment);
    });
  }

  function remove() {
    void run(async () => {
      await deleteAssignment(courseId, id);
      draft.clear();
      setConfirm(null);
      onDeleted();
    });
  }

  function suggest() {
    void run(async () => {
      const out = await suggestRubric(courseId, id, newIdempotencyKey());
      patch({ rubric: out.criteria.slice(0, 10).map((c) => ({ name: c.name, max: vi(c.max_score), guidance: c.guidance })) });
    });
  }

  const title = creating ? "Bài tập mới" : (detail?.title ?? "Bài tập");
  const suggestWhy = creating ? "Lưu bài trước khi gợi ý tiêu chí." : !answerOk ? `Cần đáp án chuẩn từ ${MIN_ANSWER} ký tự.` : undefined;

  return (
    <>
      <Drawer
        open
        wide
        onClose={onClose}
        title={title}
        error={apiErr && Object.keys(fe).length === 0 ? apiErr.userMessage : undefined}
        footer={
          teacher ? (
            <>
              {!creating && (
                <Button variant="ghost" icon={<Trash2 aria-hidden />} onClick={() => setConfirm("delete")}>
                  Xoá
                </Button>
              )}
              {!creating && !published && (
                <Button variant="secondary" onClick={() => setConfirm("publish")} disabled={!detail}>
                  Công bố bài
                </Button>
              )}
              <Button variant="primary" loading={phase === "saving"} onClick={() => save()} disabled={!creating && !detail}>
                {creating ? "Tạo" : "Lưu"}
              </Button>
            </>
          ) : undefined
        }
      >
        {!creating && q.isPending ? (
          <Skeleton lines={8} />
        ) : (
          <div className={s.form}>
            {restored && <InlineNotice compact>Đã khôi phục phần bạn đang soạn dở.</InlineNotice>}
            {phase === "queued" && (
              <InlineNotice tone="warning" compact>
                Mất mạng. Sẽ gửi lại khi có mạng.
              </InlineNotice>
            )}
            {conflict && (
              <InlineNotice
                tone="warning"
                title={`Bài tập vừa được ${"người khác"} sửa`}
                action={
                  <>
                    <Button size="sm" onClick={() => save(conflict.version)}>
                      Giữ của tôi
                    </Button>
                    <Button size="sm" variant="ghost" onClick={() => { setForm(fromDetail(conflict)); setConflict(null); draft.clear(); }}>
                      Dùng của họ
                    </Button>
                  </>
                }
              />
            )}

            <Field label="Tiêu đề" required error={fe.title}>
              {(fid, by) => <Input id={fid} aria-describedby={by} invalid={Boolean(fe.title)} value={f.title} maxLength={120} disabled={ro} onChange={(e) => patch({ title: e.target.value })} />}
            </Field>
            <Field label="Mô tả" error={fe.description}>
              {(fid, by) => <Textarea id={fid} aria-describedby={by} rows={4} value={f.description} maxLength={8000} disabled={ro} onChange={(e) => patch({ description: e.target.value })} />}
            </Field>

            <div className={s.row}>
              <Field label={<span title={published ? LOCKED_PUBLISHED : undefined}>Mã nộp thư</span>} error={fe.submission_code}>
                {(fid, by) => (
                  <Input id={fid} aria-describedby={by} invalid={Boolean(fe.submission_code)} value={f.code} placeholder="BT03" maxLength={12} disabled={ro || published}
                    onChange={(e) => patch({ code: e.target.value.toUpperCase() })} />
                )}
              </Field>
              <Field label={<span title={published ? LOCKED_PUBLISHED : undefined}>Điểm tối đa</span>} error={fe.max_score ?? (apiErr?.code === "SCORE_STEP" ? apiErr.userMessage : undefined)}>
                {(fid, by) => (
                  <Input id={fid} aria-describedby={by} inputMode="decimal" placeholder="10" value={f.max} disabled={ro || published} onChange={(e) => patch({ max: e.target.value })} />
                )}
              </Field>
            </div>

            <div className={s.row}>
              <Field label={<span title={lateLocked && !ro ? LOCKED_LATE : undefined}>Hạn nộp</span>} required error={fe.due_at}>
                {(fid, by) => <Input id={fid} aria-describedby={by} type="datetime-local" value={f.due} disabled={lateLocked} onChange={(e) => patch({ due: e.target.value })} />}
              </Field>
              <Field label="Hạn phúc khảo" error={fe.appeal_days}>
                {(fid, by) => <Input id={fid} aria-describedby={by} type="number" min={0} max={30} value={f.appeal} disabled={ro} onChange={(e) => patch({ appeal: e.target.value })} />}
              </Field>
            </div>

            <Switch label="Cho nộp muộn" checked={f.allowLate} disabled={lateLocked} title={lateLocked && !ro ? LOCKED_LATE : undefined} onChange={(v) => patch({ allowLate: v })} />
            {f.allowLate && (
              <div className={s.row}>
                <Field label="Trừ mỗi ngày" error={fe.late_penalty_per_day}>
                  {(fid, by) => <Input id={fid} aria-describedby={by} type="number" step="0.05" min={0} value={f.penalty} disabled={lateLocked} onChange={(e) => patch({ penalty: e.target.value })} />}
                </Field>
                <Field label="Tối đa số ngày" error={fe.late_max_days}>
                  {(fid, by) => <Input id={fid} aria-describedby={by} type="number" min={1} max={14} value={f.lateDays} disabled={lateLocked} onChange={(e) => patch({ lateDays: e.target.value })} />}
                </Field>
              </div>
            )}

            <Field label="Kênh nhận bài" error={fe.channels ?? fe.teams_class_id}>
              {() => (
                <div className={s.checks}>
                  {CHANNELS.map((c) => (
                    <Checkbox key={c} label={CHANNEL_LABEL[c]} checked={f.channels.includes(c)} disabled={ro}
                      onChange={(e) => patch({ channels: e.target.checked ? [...f.channels, c] : f.channels.filter((x) => x !== c) })} />
                  ))}
                </div>
              )}
            </Field>
            {f.channels.includes("TEAMS") && (
              <div className={s.row}>
                <Field label="Mã lớp Teams">
                  {(fid) => <Input id={fid} value={f.teamsClass} disabled={ro} onChange={(e) => patch({ teamsClass: e.target.value })} />}
                </Field>
                <Field label="Mã bài tập Teams">
                  {(fid) => <Input id={fid} value={f.teamsAssignment} disabled={ro} onChange={(e) => patch({ teamsAssignment: e.target.value })} />}
                </Field>
              </div>
            )}

            <Field label="Đáp án chuẩn" error={fe.reference_answer}>
              {(fid, by) => <Textarea id={fid} aria-describedby={by} rows={4} value={f.answer} maxLength={20000} disabled={ro} onChange={(e) => patch({ answer: e.target.value })} />}
            </Field>

            <div className={s.rubric} data-part="rubric">
              <div className={s.rubricHead}>
                <h3 className="ep-item-title" title={detail?.rubric_locked ? LOCKED_RUBRIC : undefined}>Tiêu chí chấm</h3>
                <Button size="sm" icon={<Sparkles aria-hidden />} disabled={Boolean(suggestWhy)} title={suggestWhy} loading={phase === "saving"} onClick={suggest}>
                  Gợi ý tiêu chí
                </Button>
              </div>
              {fe.rubric && <InlineNotice tone="danger" compact>{fe.rubric}</InlineNotice>}
              {apiErr?.code === "RUBRIC_INVALID" && (
                <InlineNotice tone="danger" compact>
                  {`${apiErr.userMessage} Tổng hiện tại ${vi(String((apiErr.details as { sum?: string } | undefined)?.sum ?? ""))}.`}
                </InlineNotice>
              )}
              {f.rubric.map((r, i) => (
                <div key={i} className={s.criterion}>
                  <Input aria-label={`Tên tiêu chí ${i + 1}`} value={r.name} maxLength={120} disabled={rubricLocked} onChange={(e) => setRow(i, { name: e.target.value })} />
                  <Input aria-label={`Điểm tiêu chí ${i + 1}`} inputMode="decimal" className={s.score} value={r.max} disabled={rubricLocked} onChange={(e) => setRow(i, { max: e.target.value })} />
                  <Input aria-label={`Hướng dẫn chấm tiêu chí ${i + 1}`} value={r.guidance} maxLength={1000} disabled={rubricLocked} onChange={(e) => setRow(i, { guidance: e.target.value })} />
                  {!rubricLocked && f.rubric.length > 1 && (
                    <IconButton label={`Bỏ tiêu chí ${i + 1}`} onClick={() => patch({ rubric: f.rubric.filter((_, k) => k !== i) })}>
                      <X aria-hidden />
                    </IconButton>
                  )}
                </div>
              ))}
              <div className={s.rubricFoot}>
                {!rubricLocked && f.rubric.length < 10 && (
                  <Button size="sm" variant="ghost" icon={<Plus aria-hidden />} onClick={() => patch({ rubric: [...f.rubric, { name: "", max: "", guidance: "" }] })}>
                    Thêm tiêu chí
                  </Button>
                )}
                <p className={s.total} data-part="rubric-total">
                  Tổng tiêu chí
                  <strong>
                    {total === null ? "—" : vi(fromUnits(total))} / {f.max}
                  </strong>
                  {total !== null && maxUnits !== null && total !== maxUnits && <StatusText tone="red">Chưa khớp</StatusText>}
                </p>
              </div>
            </div>

            {published && <Stats courseId={courseId} id={id!} />}

            {!creating && detail?.type === "ESSAY" && <GuidancePanel courseId={courseId} aid={id} teacher={teacher} userId={userId} rubric={detail.rubric} />}

            {!creating && (
              <p className={s.link} data-part="grade-link">
                {detail?.grade_link ? `Tính vào điểm: ${detail.grade_link.component_name} · ${detail.grade_link.component_key}` : "Chưa tính vào điểm"}
                <ButtonLink href="/gradebook/scheme" size="sm" variant="ghost">
                  Mở công thức
                </ButtonLink>
              </p>
            )}
          </div>
        )}
      </Drawer>

      <ConfirmIrreversible
        open={confirm !== null}
        onClose={() => setConfirm(null)}
        onConfirm={confirm === "delete" ? remove : publish}
        title={confirm === "delete" ? "Xoá bài tập" : "Công bố bài"}
        consequence={
          confirm === "delete"
            ? `Xoá “${detail?.title ?? ""}”. Không khôi phục được.`
            : `Công bố cho ${detail?.counts?.total ?? 0} sinh viên · hạn ${detail ? fmtLong(detail.due_at) : ""}`
        }
        confirmLabel={confirm === "delete" ? "Xoá" : "Công bố bài"}
        loading={phase !== "idle"}
        error={apiErr ? apiErr.userMessage : undefined}
      />
    </>
  );
}
