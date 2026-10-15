"use client";

import { useQueryClient } from "@tanstack/react-query";
import { useEffect, useRef, useState } from "react";
import { ApiError, apiClient, useAutosaveDraft } from "@/shared/data";
import { ApiErrorNotice } from "@/shared/data/ApiErrorNotice";
import { useCursorList } from "@/shared/data/useCursorList";
import { Button, EmptyState, InlineNotice, PrivateMark, Section, StatusText, Textarea } from "@/shared/ui";
import { draftText, type AiDraft, type Observation } from "./studentsApi";
import s from "./StudentProfile.module.css";

const when = (iso: string) => new Date(iso).toLocaleString("vi-VN", { day: "2-digit", month: "2-digit", hour: "2-digit", minute: "2-digit", timeZone: "Asia/Ho_Chi_Minh" });

/**
 * Tab `Ghi chú` (US-P5-05 AC3, AC5, AC6, AC9): ghi chú riêng của nhân viên + `Soạn nhận xét tổng hợp` bằng AI.
 * Ô nhập tự lưu nháp ở máy (`obs:<uid>`, `aisum:<uid>`): lỗi mạng không làm mất chữ; sửa đồng thời → 409 và hiện bản của người kia.
 */
export function ProfileNotes({ courseId, studentId, userId, focusSignal }: { courseId: string; studentId: string; userId: string; focusSignal: number }) {
  const base = `/courses/${courseId}/students/${studentId}/observations`;
  const qc = useQueryClient();
  const key = ["student-observations", courseId, studentId];
  const list = useCursorList<Observation>(key, base, { limit: 50 });
  const note = useAutosaveDraft(`obs:${studentId}`, { userId });
  const ai = useAutosaveDraft(`aisum:${studentId}`, { userId });
  const [aiOriginal, setAiOriginal] = useState<string | null>(null); // bản AI nguyên bản của lần soạn này (để đánh dấu "đã sửa")
  const [busy, setBusy] = useState<"note" | "ai" | "draft" | null>(null);
  const [err, setErr] = useState<unknown>(null);
  const [aiErr, setAiErr] = useState<unknown>(null);
  const area = useRef<HTMLTextAreaElement>(null);
  const [editing, setEditing] = useState<{ id: string; version: number; text: string } | null>(null);
  const [conflict, setConflict] = useState<Observation | null>(null);

  useEffect(() => {
    if (focusSignal > 0) area.current?.focus();
  }, [focusSignal]);

  const refresh = () => qc.invalidateQueries({ queryKey: key });

  async function saveNote() {
    setBusy("note");
    setErr(null);
    try {
      await apiClient.post(base, { body: note.value.trim(), tags: [] });
      note.clear();
      void refresh();
    } catch (e) {
      setErr(e); // nháp vẫn ở trong ô
    } finally {
      setBusy(null);
    }
  }

  async function compose() {
    setBusy("draft");
    setAiErr(null);
    try {
      const d = (await apiClient.post<{ draft: AiDraft }>(`/courses/${courseId}/students/${studentId}/ai-summary`)).data.draft;
      const text = draftText(d);
      setAiOriginal(text);
      ai.setValue(text);
    } catch (e) {
      setAiErr(e);
    } finally {
      setBusy(null);
    }
  }

  async function saveAi() {
    setBusy("ai");
    setAiErr(null);
    try {
      const body = ai.value.trim();
      await apiClient.post(base, { kind: "AI_SUMMARY", body, tags: [], ai_edited: aiOriginal !== null && body !== aiOriginal.trim() });
      ai.clear();
      setAiOriginal(null);
      void refresh();
    } catch (e) {
      setAiErr(e);
    } finally {
      setBusy(null);
    }
  }

  async function saveEdit() {
    if (!editing) return;
    setBusy("note");
    setErr(null);
    try {
      await apiClient.put(`${base}/${editing.id}`, { body: editing.text.trim(), tags: [], version: editing.version });
      setEditing(null);
      setConflict(null);
      void refresh();
    } catch (e) {
      if (e instanceof ApiError && e.code === "VERSION_CONFLICT") {
        const cur = (e.details as { current?: Observation } | undefined)?.current;
        if (cur) setConflict(cur);
      } else setErr(e);
    } finally {
      setBusy(null);
    }
  }

  async function remove(id: string) {
    try {
      await apiClient.delete(`${base}/${id}`);
      void refresh();
    } catch (e) {
      setErr(e);
    }
  }

  const aiDown = aiErr instanceof ApiError && (aiErr.status === 503 || aiErr.status === 504 || aiErr.status === 0);
  const noData = aiErr instanceof ApiError && aiErr.code === "NOT_ENOUGH_DATA";
  const drafting = ai.value !== "";

  return (
    <Section panel title="Ghi chú" action={<PrivateMark />}>
      <div className={s.compose}>
        <Textarea ref={area} rows={3} value={note.value} aria-label="Ghi chú mới" placeholder="Điều bạn quan sát được và việc đã làm" onChange={(e) => note.setValue(e.target.value)} />
        {err instanceof ApiError && <ApiErrorNotice error={err} onRetry={() => void saveNote()} title="Chưa lưu được ghi chú. Chữ của bạn vẫn còn trong ô." />}
        <div className={s.composeBar}>
          <Button variant="primary" loading={busy === "note" && !editing} disabled={note.value.trim() === ""} onClick={() => void saveNote()}>Lưu ghi chú</Button>
          {!drafting && <Button loading={busy === "draft"} onClick={() => void compose()}>Soạn nhận xét tổng hợp</Button>}
        </div>
      </div>

      {noData && <InlineNotice tone="info" compact>Sinh viên này chưa có dữ liệu để nhận xét.</InlineNotice>}
      {aiDown && (
        <InlineNotice tone="warning" compact action={<Button size="sm" onClick={() => void compose()}>Thử lại</Button>}>
          Chưa soạn được nhận xét. Thử lại.
        </InlineNotice>
      )}
      {aiErr instanceof ApiError && !aiDown && !noData && <ApiErrorNotice error={aiErr} title="Chưa soạn được nhận xét." />}

      {drafting && (
        <div className={s.compose} data-part="ai-draft">
          <p className="ep-meta"><StatusText tone="blue">AI</StatusText> Nhận xét tổng hợp</p>
          <Textarea rows={8} value={ai.value} aria-label="Nhận xét tổng hợp (bản nháp AI)" onChange={(e) => ai.setValue(e.target.value)} />
          <div className={s.composeBar}>
            <Button variant="primary" loading={busy === "ai"} disabled={ai.value.trim() === ""} onClick={() => void saveAi()}>Lưu nhận xét</Button>
            <Button variant="text" onClick={() => { ai.clear(); setAiOriginal(null); }}>Bỏ bản nháp</Button>
          </div>
        </div>
      )}

      {list.isError && list.items.length === 0 ? (
        <ApiErrorNotice error={list.error} showTechnical onRetry={() => void list.refetch()} title="Không tải được ghi chú." />
      ) : list.items.length === 0 && !list.isPending ? (
        <EmptyState title="Chưa có ghi chú nào." />
      ) : (
        <ul className={s.notes}>
          {list.items.map((n) => (
            <li key={n.id} data-part="observation">
              {n.kind === "AI_SUMMARY" && <p className="ep-meta"><StatusText tone="blue">AI</StatusText> {n.ai_edited ? "Nhận xét tổng hợp · Đã sửa bởi giảng viên" : "Nhận xét tổng hợp · AI"}</p>}
              {editing?.id === n.id ? (
                <>
                  <Textarea rows={4} value={editing.text} aria-label="Sửa ghi chú" onChange={(e) => setEditing({ ...editing, text: e.target.value })} />
                  {conflict && (
                    <InlineNotice tone="warning" compact action={<Button size="sm" onClick={() => { setEditing({ id: n.id, version: conflict.version, text: editing.text }); setConflict(null); }}>Giữ bản của tôi</Button>}>
                      {conflict.author.full_name} vừa sửa ghi chú này: “{conflict.body}”
                    </InlineNotice>
                  )}
                  <div className={s.composeBar}>
                    <Button variant="primary" loading={busy === "note"} onClick={() => void saveEdit()}>Lưu</Button>
                    <Button variant="text" onClick={() => { setEditing(null); setConflict(null); }}>Để sau</Button>
                  </div>
                </>
              ) : (
                <>
                  <p className={s.noteBody}>{n.body}</p>
                  <p className="ep-meta">
                    {n.author.full_name} · {when(n.created_at)}
                    {n.author.id === userId && (
                      <>
                        {" · "}
                        <Button variant="text" size="sm" onClick={() => setEditing({ id: n.id, version: n.version, text: n.body })}>Sửa</Button>
                        <Button variant="text" size="sm" onClick={() => void remove(n.id)}>Xoá</Button>
                      </>
                    )}
                  </p>
                </>
              )}
            </li>
          ))}
        </ul>
      )}
    </Section>
  );
}
