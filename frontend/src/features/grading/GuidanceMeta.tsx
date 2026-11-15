"use client";

import { useQueryClient } from "@tanstack/react-query";
import { useRef, useState } from "react";
import { ApiError, newIdempotencyKey } from "@/shared/data";
import { Button, ConfirmIrreversible, Popover, StatusText } from "@/shared/ui";
import { gradeOne, type ResultView } from "./gradingApi";

/**
 * Dòng phụ đề của màn duyệt cho «Lưu ý chấm» (10.6): bản đã ghim cho KẾT QUẢ NÀY, danh sách lưu ý theo tiêu chí,
 * chip «Bản cũ» + `Chấm lại` khi nháp chưa duyệt chấm theo bản cũ. Chỉ có khi lớp bật tính năng (view.guidance).
 */
export function GuidanceMeta({ courseId, view }: { courseId: string; view: ResultView }) {
  const g = view.guidance;
  const qc = useQueryClient();
  const [confirm, setConfirm] = useState(false);
  const [state, setState] = useState<{ loading: boolean; error?: string }>({ loading: false });
  const idem = useRef<string | null>(null);
  if (!g || (g.version === 0 && !g.stale)) return null;
  const count = Object.values(g.items_by_criterion).reduce((n, l) => n + l.length, 0);
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
      {g.version > 0 && (
        <Popover
          label="Lưu ý chấm đã dùng"
          align="start"
          trigger={(p) => (
            <Button size="sm" variant="ghost" data-part="guidance-meta" onClick={p.toggle} aria-expanded={p["aria-expanded"]} aria-haspopup="true">{`bản v${g.version} · ${count} lưu ý`}</Button>
          )}
        >
          <div data-part="guidance-popover">
            {view.criteria
              .filter((c) => (g.items_by_criterion[c.id] ?? []).length > 0)
              .map((c) => (
                <div key={c.id}>
                  <strong>{c.name}</strong>
                  {g.items_by_criterion[c.id]!.map((t, i) => (
                    <p key={i}>{t}</p>
                  ))}
                </div>
              ))}
          </div>
        </Popover>
      )}
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
