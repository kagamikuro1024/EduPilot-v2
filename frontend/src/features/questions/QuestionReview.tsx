"use client";

import { useQuery, useQueryClient } from "@tanstack/react-query";
import { Check, Copy, Pencil, Send, X, Archive } from "lucide-react";
import { useState } from "react";
import { ApiError, ApiErrorNotice, apiClient } from "@/shared/data";
import { Markdown } from "@/shared/domain";
import { Button, Drawer, InlineNotice, OverflowMenu, PrivateMark, StatusText } from "@/shared/ui";
import { CodePanel } from "./CodePanel";
import { QuestionForm } from "./QuestionForm";
import { DIFF_LABEL, ORIGIN_LABEL, qKey, qPath, REVIEW_LABEL, REVIEW_TONE, TYPE_LABEL, type QuestionDetail } from "./questionsApi";
import s from "./Questions.module.css";

type Props = { courseId: string; id: string; userId?: string; canEditTests: boolean; onClose: () => void; onChanged: (note?: string) => void; onOpen: (id: string) => void };

/** Drawer rộng `QuestionReview` (DESIGN.md §14.17): nội dung, đáp án đúng, giải thích; bài code: đề, giới hạn, test, nhập zip, chạy lời giải mẫu. Hành động chính `Duyệt`; `Chỉnh sửa`; menu `Loại` / `Nhân bản` / `Lưu trữ`. */
export function QuestionReview({ courseId, id, userId, canEditTests, onClose, onChanged, onOpen }: Props) {
  const qc = useQueryClient();
  const key = qKey(courseId, id);
  const q = useQuery({ queryKey: key, queryFn: async ({ signal }) => (await apiClient.get<QuestionDetail>(`${qPath(courseId)}/${id}`, { signal })).data });
  const [editing, setEditing] = useState(false);
  const [err, setErr] = useState<unknown>(null);
  const [pending, setPending] = useState(false);
  const d = q.data;

  const used = (d?.used_in_exams ?? []).filter((e) => ["SCHEDULED", "OPEN", "CLOSED", "PUBLISHED"].includes(e.status));
  const locked = used.length > 0;

  async function refresh(note?: string) {
    await qc.invalidateQueries({ queryKey: ["questions", courseId] });
    onChanged(note);
  }
  async function review(decision: "REQUEST" | "APPROVE" | "REJECT", note: string) {
    if (!d) return;
    setPending(true);
    setErr(null);
    try {
      await apiClient.put(`${qPath(courseId)}/${d.id}/review`, { decision, version: d.version });
      await refresh(note);
    } catch (x) {
      setErr(x);
      void qc.invalidateQueries({ queryKey: key });
    } finally {
      setPending(false);
    }
  }
  async function archive() {
    if (!d) return;
    setErr(null);
    try {
      await apiClient.post(`${qPath(courseId)}/${d.id}/archive`, {});
      await refresh("Đã lưu trữ câu hỏi");
      onClose();
    } catch (x) {
      setErr(x);
    }
  }
  async function duplicate() {
    if (!d) return;
    setErr(null);
    try {
      const r = await apiClient.post<QuestionDetail>(`${qPath(courseId)}/${d.id}/duplicate`, {});
      await refresh("Đã nhân bản — bạn đang xem bản sao");
      onOpen(r.data.id);
    } catch (x) {
      setErr(x);
    }
  }

  const approvalProblems = err instanceof ApiError && err.status === 422 && Array.isArray(err.details) ? (err.details as Array<{ message?: string }>) : null;
  const canReview = d && d.archived_at === null && (d.review_status === "DRAFT" || d.review_status === "PENDING");

  return (
    <Drawer
      open
      wide
      onClose={onClose}
      title={d?.title ?? "Câu hỏi"}
      description={d ? `${TYPE_LABEL[d.type]} · ${d.topic} · ${DIFF_LABEL[d.difficulty]}` : undefined}
      loading={q.isPending}
    >
      {q.isError && <ApiErrorNotice error={q.error} showTechnical onRetry={() => void q.refetch()} context="Dữ liệu câu hỏi không bị mất." />}
      {d && (
        <div className={s.detail}>
          <p className={s.statusRow}>
            <StatusText tone={d.archived_at ? "neutral" : REVIEW_TONE[d.review_status]}>{d.archived_at ? "Đã lưu trữ" : REVIEW_LABEL[d.review_status]}</StatusText>
            <span className={s.meta}>{ORIGIN_LABEL[d.origin]}</span>
            {d.origin === "AI_DRAFT" && <PrivateMark>AI soạn nháp — chỉ giảng viên/TA thấy nhãn này</PrivateMark>}
          </p>
          {locked && <InlineNotice tone="warning" title={`Đang dùng trong bài thi ${used.map((e) => e.title).join(", ")} — nhân bản để sửa`}>Chỉ sửa được chủ đề, độ khó và giải thích.</InlineNotice>}
          {err !== null && approvalProblems && (
            <InlineNotice tone="warning" title="Chưa duyệt được">
              <ul className={s.zipErrors}>{approvalProblems.map((p, i) => <li key={i}>{p.message}</li>)}</ul>
            </InlineNotice>
          )}
          {err !== null && !approvalProblems && <ApiErrorNotice error={err} showTechnical />}

          {editing ? (
            <QuestionForm courseId={courseId} userId={userId} detail={d} onCancel={() => setEditing(false)} onSaved={async () => { setEditing(false); await refresh("Đã lưu thay đổi"); await q.refetch(); }} />
          ) : (
            <>
              <Markdown source={d.stem} />
              {d.options.length > 0 && (
                <ul className={s.options} aria-label="Đáp án">
                  {d.options.map((o) => {
                    const right = d.answer_key?.option_ids?.includes(o.id);
                    return (
                      <li key={o.id} className={[s.option, right ? s.optionRight : ""].join(" ")}>
                        <span className={s.optionMark}>{right ? "Đúng" : ""}</span>
                        <span>{o.body}{o.pinned_last ? " (ghim cuối)" : ""}</span>
                      </li>
                    );
                  })}
                </ul>
              )}
              {d.type === "TRUE_FALSE" && <p className={s.optionRight}>Đáp án đúng: <strong>{d.answer_key?.value ? "Đúng" : "Sai"}</strong></p>}
              {d.explanation && (
                <div>
                  <h3 className={s.h3}>Giải thích</h3>
                  <Markdown source={d.explanation} />
                </div>
              )}
              {d.type === "CODE" && d.code && <CodePanel courseId={courseId} q={d} canEditTests={canEditTests} locked={locked} onChanged={() => { void q.refetch(); void refresh(); }} />}
            </>
          )}

          {!editing && (
            <div className={s.detailActions}>
              {canReview && d.review_status === "PENDING" && (
                <Button variant="primary" icon={<Check aria-hidden />} loading={pending} onClick={() => void review("APPROVE", "Đã duyệt câu hỏi")}>Duyệt</Button>
              )}
              {canReview && d.review_status === "DRAFT" && (
                <>
                  <Button variant="primary" icon={<Check aria-hidden />} loading={pending} onClick={() => void review("APPROVE", "Đã duyệt câu hỏi")}>Duyệt</Button>
                  <Button icon={<Send aria-hidden />} loading={pending} onClick={() => void review("REQUEST", "Đã gửi duyệt")}>Gửi duyệt</Button>
                </>
              )}
              {d.archived_at === null && <Button icon={<Pencil aria-hidden />} onClick={() => setEditing(true)}>Chỉnh sửa</Button>}
              <OverflowMenu
                label="Thêm hành động với câu hỏi"
                items={[
                  ...(canReview ? [{ label: "Loại", icon: <X aria-hidden />, danger: true, onSelect: () => void review("REJECT", "Đã loại câu hỏi") }] : []),
                  { label: "Nhân bản", icon: <Copy aria-hidden />, onSelect: () => void duplicate() },
                  ...(d.archived_at === null ? [{ label: "Lưu trữ", icon: <Archive aria-hidden />, onSelect: () => void archive() }] : []),
                ]}
              />
            </div>
          )}
        </div>
      )}
    </Drawer>
  );
}
