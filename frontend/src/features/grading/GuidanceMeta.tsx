"use client";

import { useQueryClient } from "@tanstack/react-query";
import { useRef, useState } from "react";
import { ApiError, newIdempotencyKey } from "@/shared/data";
import { Button, ConfirmIrreversible, Popover, StatusText } from "@/shared/ui";
import { gradeOne, type ResultView } from "./gradingApi";

/** Phụ đề `meta` trên nhãn một tiêu chí: «bản v1 · 3 lần sửa», bấm mở lưu ý áp dụng cho tiêu chí đó (US-GD-06 AC3). Không có lưu ý → không hiện. */
export function CriterionGuidance({ view, criterionId }: { view: ResultView; criterionId: string }) {
  const g = view.guidance;
  const items = g?.items_by_criterion[criterionId] ?? [];
  if (!g || g.version === 0 || items.length === 0) return null;
  return (
    <Popover
      label="Lưu ý chấm đã dùng"
      align="start"
      trigger={(p) => (
        <Button size="sm" variant="ghost" data-part="guidance-meta" onClick={p.toggle} aria-expanded={p["aria-expanded"]} aria-haspopup="true">{`bản v${g.version} · ${g.support_by_criterion?.[criterionId] ?? 0} lần sửa`}</Button>
      )}
    >
      <div data-part="guidance-popover">
        {items.map((t, i) => (
          <p key={i}>{t}</p>
        ))}
      </div>
    </Popover>
  );
}

/**
 * Chip «Bản cũ» + `Chấm lại` trên đầu màn duyệt cho «Lưu ý chấm» (10.6) khi nháp chưa duyệt chấm theo bản cũ. Chỉ có khi lớp bật tính năng (view.guidance).
 */
export function GuidanceMeta({ courseId, view }: { courseId: string; view: ResultView }) {
  const g = view.guidance;
  const qc = useQueryClient();
  const [confirm, setConfirm] = useState(false);
  const [state, setState] = useState<{ loading: boolean; error?: string }>({ loading: false });
  const idem = useRef<string | null>(null);
  if (!g || (g.version === 0 && !g.stale)) return null;
  const edited = view.criteria.some((c) => c.edited);

  async function regrade() {
    setState({ loading: true });
    idem.current ??= newIdempotencyKey();
    try {
      await gradeOne(courseId, view.submission.id, idem.current);
      idem.current = null;
      setState({ loading: false });
      setConfirm(false);
      void qc.invalidateQueries({ queryKey: ["grading-result", courseId] });
      void qc.invalidateQueries({ queryKey: ["grading-queue", courseId] });
    } catch (e) {
      setState({ loading: false, error: e instanceof ApiError ? e.userMessage : "Chưa chấm lại được. Hãy thử lại." });
    }
  }

  return (
    <>
      {g.stale && view.result.status === "DRAFT" && (
        <>
          <StatusText tone="amber" chip>Bản cũ</StatusText>
          <Button size="sm" onClick={() => { setState({ loading: false }); setConfirm(true); }}>Chấm lại</Button>
        </>
      )}
      <ConfirmIrreversible
        open={confirm}
        onClose={() => setConfirm(false)}
        onConfirm={() => void regrade()}
        title="Chấm lại bài này"
        consequence={edited ? "Điểm và nhận xét bạn đã sửa trên bài này sẽ không còn. AI chấm lại theo lưu ý mới nhất." : "AI chấm lại theo lưu ý mới nhất."}
        confirmLabel="Chấm lại"
        loading={state.loading}
        error={state.error}
      />
    </>
  );
}
