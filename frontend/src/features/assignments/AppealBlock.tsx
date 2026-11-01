"use client";

import { useQueryClient } from "@tanstack/react-query";
import { useRef, useState } from "react";
import { ApiError, newIdempotencyKey, useAutosaveDraft } from "@/shared/data";
import { Button, Checkbox, Field, InlineNotice, PanelSection, Textarea } from "@/shared/ui";
import { WD, dayLabel, fmtDM, fmtHM } from "@/features/calendar/calendarApi";
import { fileAppeal, mineKey, type MineOut } from "./assignmentApi";
import s from "./StudentAssignment.module.css";

const MIN = 10;
const MAX = 1000;
const at = (iso: string) => `${WD[new Date(dayLabel(iso)).getUTCDay()]}, ${fmtHM(iso)}`;
const closeDate = (iso: string) => fmtDM(dayLabel(iso));

/**
 * Yêu cầu xem lại điểm của sinh viên (US-P7-10 AC1, AC9): chọn tiêu chí theo TÊN + lý do 10–1.000 ký tự.
 * Lý do tự lưu vào máy (mất mạng không mất chữ) và gửi lại được; hạn do MÁY CHỦ quyết (`closes_at`).
 */
export function AppealBlock({ courseId, id, data, userId, now }: { courseId: string; id: string; data: MineOut; userId?: string; now: number }) {
  const qc = useQueryClient();
  const draft = useAutosaveDraft(`appeal:${id}`, { userId });
  const [open, setOpen] = useState(false);
  const [picked, setPicked] = useState<string[]>([]);
  const [sending, setSending] = useState(false);
  const [err, setErr] = useState<unknown>(null);
  const [sentAt, setSentAt] = useState<string | null>(null);
  const busy = useRef(false);
  const idem = useRef<string | null>(null);

  const r = data.result;
  if (!r) return null;
  const ap = r.appeal;
  const closed = !ap.closes_at || new Date(ap.closes_at).getTime() <= now;

  if (ap.state === "UPHELD" || ap.state === "ADJUSTED") {
    return (
      <PanelSection title="Yêu cầu xem lại">
        <p className={s.critHead} data-part="appeal-decision">
          <span>{ap.state === "UPHELD" ? "Giữ nguyên" : "Đã sửa"}</span>
        </p>
        {ap.response && <p className={s.critText} data-part="appeal-response">{ap.response}</p>}
      </PanelSection>
    );
  }

  if (ap.state === "OPEN" || sentAt) {
    return (
      <PanelSection title="Yêu cầu xem lại">
        <p className={s.status} data-part="appeal-sent">{`Đã gửi yêu cầu xem lại${sentAt ? ` · ${at(sentAt)}` : ""}`}</p>
      </PanelSection>
    );
  }

  if (closed) return null;

  const reason = draft.value;
  const tooShort = reason.trim().length < MIN;

  const send = () => {
    if (busy.current) return; // bấm đúp: một yêu cầu
    busy.current = true;
    setSending(true);
    setErr(null);
    idem.current ??= newIdempotencyKey();
    void (async () => {
      try {
        const out = await fileAppeal(courseId, id, picked, reason.trim(), idem.current!);
        idem.current = null;
        draft.clear();
        setSentAt(out.appeal.created_at);
        setOpen(false);
        await qc.invalidateQueries({ queryKey: mineKey(courseId, id) });
      } catch (e) {
        setErr(e); // giữ nguyên chữ đang gõ: bấm Gửi lại dùng cùng Idempotency-Key
      } finally {
        busy.current = false;
        setSending(false);
      }
    })();
  };

  return (
    <PanelSection title="Yêu cầu xem lại">
      {!open ? (
        <div className={s.actions}>
          <Button onClick={() => setOpen(true)} title={ap.closes_at ? `Hạn ${closeDate(ap.closes_at)}` : undefined}>
            Yêu cầu xem lại
          </Button>
        </div>
      ) : (
        <div data-part="appeal-form">
          <Field label="Tiêu chí muốn xem lại">
            {() => (
              <div className={s.checks}>
                {data.assignment.criteria.map((c) => (
                  <Checkbox
                    key={c.id}
                    label={c.name}
                    checked={picked.includes(c.id)}
                    onChange={(e) => setPicked((p) => (e.target.checked ? [...p, c.id] : p.filter((x) => x !== c.id)))}
                  />
                ))}
              </div>
            )}
          </Field>
          <Field label={`Lý do · ${reason.trim().length}/${MAX}`}>
            {(fid) => (
              <Textarea id={fid} rows={4} maxLength={MAX} value={reason} onChange={(e) => draft.setValue(e.target.value)} />
            )}
          </Field>
          {err != null && (
            <InlineNotice tone="danger" compact action={<Button size="sm" loading={sending} onClick={send}>Gửi lại</Button>}>
              {err instanceof ApiError ? err.userMessage : "Chưa gửi được."}
            </InlineNotice>
          )}
          <div className={s.actions}>
            <Button variant="primary" loading={sending} disabled={picked.length === 0 || tooShort} onClick={send}>
              Gửi
            </Button>
          </div>
        </div>
      )}
    </PanelSection>
  );
}
