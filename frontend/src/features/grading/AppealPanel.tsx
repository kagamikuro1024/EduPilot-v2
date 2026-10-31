"use client";

import { useQueryClient } from "@tanstack/react-query";
import { useRef, useState } from "react";
import { ApiError, newIdempotencyKey } from "@/shared/data";
import { Button, ConfirmIrreversible, Field, InlineNotice, Input, Textarea } from "@/shared/ui";
import { fmtLong } from "@/features/calendar/calendarApi";
import { vi } from "@/features/assignments/assignmentApi";
import { decideAppeal, resultKey, type Appeal, type ResultView } from "./gradingApi";
import s from "./SubmissionReview.module.css";

const MIN = 10;
const MAX = 2000;

/**
 * Khối "Yêu cầu xem lại" ở đầu cột phải màn duyệt (US-P7-10 AC3): tiêu chí sinh viên chọn, lý do,
 * điểm đã công bố và điểm AI nháp gốc để đối chiếu. Chỉ TEACHER quyết; TA chỉ đọc.
 */
export function AppealPanel({ courseId, submissionId, view, appeal, teacher }: { courseId: string; submissionId: string; view: ResultView; appeal: Appeal; teacher: boolean }) {
  const qc = useQueryClient();
  const [mode, setMode] = useState<"UPHELD" | "ADJUSTED" | null>(null);
  const [response, setResponse] = useState("");
  const [scores, setScores] = useState<Record<string, string>>({});
  const [confirming, setConfirming] = useState(false);
  const [err, setErr] = useState<string | null>(null);
  const [busy, setBusy] = useState(false);
  const running = useRef(false);
  const idem = useRef<string | null>(null);

  const decided = appeal.status !== "OPEN";
  const changed = view.criteria.filter((c) => scores[c.id] !== undefined && scores[c.id].replace(",", ".") !== (c.score ?? ""));

  const decide = () => {
    if (running.current) return; // bấm đúp: một quyết định
    running.current = true;
    setBusy(true);
    setErr(null);
    idem.current ??= newIdempotencyKey();
    void (async () => {
      try {
        await decideAppeal(
          courseId,
          appeal.id,
          {
            decision: mode === "ADJUSTED" ? "ADJUSTED" : "UPHELD",
            response: response.trim(),
            version: appeal.version,
            ...(mode === "ADJUSTED" ? { criteria_scores: changed.map((c) => ({ id: c.id, score: scores[c.id].replace(",", ".") })) } : {}),
          },
          idem.current!,
        );
        idem.current = null;
        setConfirming(false);
        await qc.invalidateQueries({ queryKey: resultKey(courseId, submissionId) });
      } catch (e) {
        setErr(e instanceof ApiError ? e.userMessage : "Chưa gửi được quyết định.");
      } finally {
        running.current = false;
        setBusy(false);
      }
    })();
  };

  return (
    <section className={s.appeal} data-part="appeal" aria-label="Yêu cầu xem lại">
      <p className={s.critHead}>
        <span className={s.critName}>Yêu cầu xem lại</span>
        <span className={s.aiTag}>{fmtLong(appeal.created_at)}</span>
      </p>
      <p className={s.aiTag} data-part="appeal-who">{`${appeal.student.full_name} · ${appeal.student.student_code}`}</p>
      <p className={s.aiTag} data-part="appeal-criteria">{appeal.criteria.map((c) => c.name).join(" · ")}</p>
      <p className={s.footText} data-part="appeal-reason">{appeal.reason}</p>
      <p className={s.aiTag} data-part="appeal-scores">
        {`Điểm đã công bố ${vi(appeal.score_before)}${appeal.ai_total_before_late ? ` · AI chấm nháp ${vi(appeal.ai_total_before_late)}` : ""}`}
      </p>

      {decided ? (
        <>
          <p className={s.critHead} data-part="appeal-decision">
            <span>{appeal.status === "UPHELD" ? "Giữ nguyên" : "Đã sửa"}</span>
            {appeal.score_after && <span>{vi(appeal.score_after)}</span>}
          </p>
          {appeal.response && <p className={s.footText} data-part="appeal-response">{appeal.response}</p>}
        </>
      ) : teacher ? (
        <>
          <div className={s.appealActions}>
            <Button size="sm" variant={mode === "UPHELD" ? "primary" : "secondary"} onClick={() => setMode("UPHELD")}>
              Giữ nguyên điểm
            </Button>
            <Button size="sm" variant={mode === "ADJUSTED" ? "primary" : "secondary"} onClick={() => setMode("ADJUSTED")}>
              Sửa điểm
            </Button>
          </div>
          {mode === "ADJUSTED" &&
            view.criteria.map((c) => (
              <Field key={c.id} label={`${c.name} · tối đa ${vi(c.max)}`}>
                {(id) => (
                  <Input
                    id={id}
                    className={s.scoreInput}
                    inputMode="decimal"
                    data-part="appeal-score"
                    data-criterion={c.id}
                    value={scores[c.id] ?? (c.score ? vi(c.score) : "")}
                    onChange={(e) => setScores((p) => ({ ...p, [c.id]: e.target.value }))}
                  />
                )}
              </Field>
            ))}
          {mode && (
            <Field label={`Phản hồi · ${response.trim().length}/${MAX}`}>
              {(id) => <Textarea id={id} rows={3} maxLength={MAX} data-part="appeal-response-input" value={response} onChange={(e) => setResponse(e.target.value)} />}
            </Field>
          )}
          {err && <InlineNotice tone="danger" compact>{err}</InlineNotice>}
          {mode && (
            <div className={s.appealActions}>
              <Button
                variant="primary"
                disabled={response.trim().length < MIN || (mode === "ADJUSTED" && changed.length === 0)}
                onClick={() => setConfirming(true)}
              >
                Quyết định
              </Button>
            </div>
          )}
          <ConfirmIrreversible
            open={confirming}
            onClose={() => setConfirming(false)}
            onConfirm={decide}
            title="Quyết định phúc khảo"
            consequence={
              mode === "ADJUSTED"
                ? `Sửa ${changed.length} tiêu chí · sinh viên nhận thư phản hồi · không quyết lại được`
                : "Giữ nguyên điểm · sinh viên nhận thư phản hồi · không quyết lại được"
            }
            confirmLabel="Quyết định"
            loading={busy}
          />
        </>
      ) : null}
    </section>
  );
}
