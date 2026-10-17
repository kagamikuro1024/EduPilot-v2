"use client";

import { X } from "lucide-react";
import { useEffect, useRef, useState } from "react";
import { ApiError, ApiErrorNotice, fieldErrors } from "@/shared/data";
import { ActionList, ActionRow, Button, DataTable, Drawer, EmptyState, Field, FilterChips, IconButton, Input, OverflowMenu, Panel, Skeleton, StatusText, UndoLine, type Column } from "@/shared/ui";
import { WD, dayLabel, fmtDM, fmtHM, fromLocalInput, toLocalInput } from "./calendarApi";
import { useGenerateSessions, usePreviewSessions, useSessionMutations, useSessions, type GenerateInput, type PlannedSession, type Preview, type SessionRow } from "./sessionsApi";
import s from "./Calendar.module.css";

const DAYS = [
  { value: "1", label: "T2" }, { value: "2", label: "T3" }, { value: "3", label: "T4" }, { value: "4", label: "T5" },
  { value: "5", label: "T6" }, { value: "6", label: "T7" }, { value: "7", label: "CN" },
];

const dateLabel = (iso: string) => `${WD[new Date(dayLabel(iso)).getUTCDay()]}, ${fmtDM(dayLabel(iso))}`;
const ymdLabel = (ymd: string) => { const [y, m, d] = ymd.split("-"); return `${d}/${m}/${y}`; };

const COLUMNS: Column<PlannedSession>[] = [
  { key: "no", header: "Buổi", render: (r) => r.session_no, width: "72px", primary: true },
  { key: "day", header: "Ngày", render: (r) => dateLabel(r.starts_at) },
  { key: "time", header: "Giờ", render: (r) => `${fmtHM(r.starts_at)}–${fmtHM(r.ends_at)}` },
];

/** Ngăn kéo "Lịch buổi học" (Staff, FEAT-crm-360 7.1): tạo hàng loạt có xem trước; sửa / xoá từng buổi. */
export function SessionsDrawer({ courseId, open, onClose }: { courseId: string; open: boolean; onClose: () => void }) {
  return (
    <Drawer open={open} onClose={onClose} title="Lịch buổi học" wide>
      <div className={s.sessions} data-part="sessions-drawer">
        {open && <GenerateForm courseId={courseId} />}
        {open && <SessionList courseId={courseId} />}
      </div>
    </Drawer>
  );
}

function GenerateForm({ courseId }: { courseId: string }) {
  const [days, setDays] = useState<string[]>([]);
  const [start, setStart] = useState("");
  const [end, setEnd] = useState("");
  const [room, setRoom] = useState("");
  const [from, setFrom] = useState("");
  const [to, setTo] = useState("");
  const [skip, setSkip] = useState<string[]>([]);
  const [skipInput, setSkipInput] = useState("");
  const [preview, setPreview] = useState<Preview | null>(null);
  const [done, setDone] = useState<number | null>(null);
  const [failed, setFailed] = useState(false);
  const previewM = usePreviewSessions(courseId);
  const gen = useGenerateSessions(courseId);

  const body: GenerateInput = { weekdays: days.map(Number), start_time: start, end_time: end, room: room.trim() || null, from, to, exclude_dates: skip };
  const ready = days.length > 0 && start !== "" && end !== "" && from !== "" && to !== "";
  // Mọi chỉnh sửa làm xem trước cũ hết hiệu lực: không bao giờ tạo thứ khác với thứ người dùng đã xem.
  const edit = <T,>(set: (v: T) => void) => (v: T) => { set(v); setPreview(null); setDone(null); setFailed(false); previewM.reset(); };
  const errs = fieldErrors(previewM.error ?? gen.error);
  const other = [previewM.error, gen.error].find((e) => e instanceof ApiError && !(e.code === "VALIDATION_FAILED" && Object.keys(fieldErrors(e)).length > 0));

  const wouldN = preview?.would_create.length ?? 0;
  const excluded = preview?.skipped.filter((k) => k.reason === "EXCLUDED").length ?? 0;
  const dup = preview?.skipped.filter((k) => k.reason === "DUPLICATE").length ?? 0;

  const runPreview = () => {
    setDone(null);
    setFailed(false);
    previewM.mutate(body, { onSuccess: setPreview });
  };
  const create = async () => {
    try {
      // sau lỗi mạng: thử lại CÙNG khoá (không tạo đôi); lần bấm mới: khoá mới
      const r = await (failed ? gen.retry() : gen.mutate(body));
      if (r) { setDone(r.data.created); setPreview(null); setFailed(false); }
    } catch {
      setFailed(true);
    }
  };

  return (
    <form
      className={s.sessionForm}
      onSubmit={(e) => {
        e.preventDefault();
        if (ready) runPreview();
      }}
    >
      <Field label="Thứ" error={errs.weekdays}>
        {() => <FilterChips label="Thứ trong tuần" value={days} onChange={edit(setDays)} options={DAYS} />}
      </Field>
      <div className={s.formPair}>
        <Field label="Bắt đầu" error={errs.start_time}>{(id) => <Input id={id} type="time" value={start} onChange={(e) => edit(setStart)(e.target.value)} invalid={Boolean(errs.start_time)} />}</Field>
        <Field label="Kết thúc" error={errs.end_time}>{(id) => <Input id={id} type="time" value={end} onChange={(e) => edit(setEnd)(e.target.value)} invalid={Boolean(errs.end_time)} />}</Field>
      </div>
      <Field label="Phòng" error={errs.room}>{(id) => <Input id={id} value={room} maxLength={40} placeholder="P.302" onChange={(e) => edit(setRoom)(e.target.value)} invalid={Boolean(errs.room)} />}</Field>
      <div className={s.formPair}>
        <Field label="Từ ngày" error={errs.from}>{(id) => <Input id={id} type="date" value={from} onChange={(e) => edit(setFrom)(e.target.value)} invalid={Boolean(errs.from)} />}</Field>
        <Field label="Đến ngày" error={errs.to}>{(id) => <Input id={id} type="date" value={to} onChange={(e) => edit(setTo)(e.target.value)} invalid={Boolean(errs.to)} />}</Field>
      </div>
      <Field label="Ngày nghỉ" error={errs.exclude_dates}>
        {(id) => (
          <div className={s.skipRow}>
            <Input id={id} type="date" value={skipInput} onChange={(e) => setSkipInput(e.target.value)} />
            <Button
              type="button"
              disabled={skipInput === "" || skip.includes(skipInput)}
              onClick={() => { edit(setSkip)([...skip, skipInput].sort()); setSkipInput(""); }}
            >
              Thêm
            </Button>
          </div>
        )}
      </Field>
      {skip.length > 0 && (
        <ul className={s.skipList} aria-label="Ngày nghỉ đã thêm">
          {skip.map((d) => (
            <li key={d}>
              {ymdLabel(d)}
              <IconButton label={`Bỏ ngày nghỉ ${ymdLabel(d)}`} size="sm" onClick={() => edit(setSkip)(skip.filter((x) => x !== d))}><X aria-hidden /></IconButton>
            </li>
          ))}
        </ul>
      )}
      {other != null && <ApiErrorNotice error={other} />}

      {preview && (
        <div className={s.preview} data-part="sessions-preview">
          <p className={s.previewSum} role="status">
            Sẽ tạo {wouldN} buổi{excluded > 0 ? ` · bỏ ${excluded} ngày nghỉ` : ""}{dup > 0 ? ` · ${dup} buổi đã có` : ""}
          </p>
          {wouldN > 0 && <DataTable caption="Các buổi sẽ tạo" columns={COLUMNS} rows={preview.would_create} rowKey={(r) => String(r.session_no)} dense />}
        </div>
      )}
      {done !== null && <StatusText tone="green">{done > 0 ? `Đã tạo ${done} buổi` : "Không có buổi mới"}</StatusText>}

      <div className={s.formActions}>
        <Button type="submit" disabled={!ready || previewM.isPending}>Xem trước</Button>
        <Button type="button" variant="primary" disabled={wouldN === 0 || gen.pending} onClick={() => void create()}>{wouldN > 0 ? `Tạo ${wouldN} buổi` : "Tạo buổi"}</Button>
      </div>
    </form>
  );
}

function SessionList({ courseId }: { courseId: string }) {
  const list = useSessions(courseId);
  const m = useSessionMutations(courseId);
  const [editing, setEditing] = useState<string | null>(null);
  const [pending, setPending] = useState<SessionRow | null>(null);
  const cancelled = useRef(false);
  const pendingRef = useRef<SessionRow | null>(null);
  useEffect(() => { pendingRef.current = pending; });
  useEffect(() => () => { if (pendingRef.current) m.remove.mutate(pendingRef.current.id); }, []); // eslint-disable-line react-hooks/exhaustive-deps
  const commitDelete = () => {
    if (pending && !cancelled.current) m.remove.mutate(pending.id);
    cancelled.current = false;
    setPending(null);
  };
  const items = list.items.filter((x) => x.id !== pending?.id);

  if (list.isPending) return <Skeleton lines={4} />;
  if (list.isError) return <ApiErrorNotice error={list.error} onRetry={() => void list.refetch()} />;
  return (
    <div className={s.sessionList}>
      {pending && <UndoLine message={`Đã xoá buổi ${pending.session_no}`} onUndo={() => { cancelled.current = true; setPending(null); }} onDone={commitDelete} />}
      {m.remove.isError && <ApiErrorNotice error={m.remove.error} />}
      {items.length === 0 ? (
        <EmptyState title="Lớp chưa có buổi học." />
      ) : (
        <ActionList label="Buổi đã có">
          {items.map((x) =>
            editing === x.id ? (
              <li key={x.id} className={s.editRow}>
                <EditRow row={x} busy={m.update.isPending} error={m.update.error} onCancel={() => { setEditing(null); m.update.reset(); }} onSave={(b) => m.update.mutate({ ...b, id: x.id, version: x.version }, { onSuccess: () => setEditing(null) })} />
              </li>
            ) : (
              <ActionRow
                key={x.id}
                tone="neutral"
                title={`Buổi ${x.session_no} · ${dateLabel(x.starts_at)}`}
                context={`${fmtHM(x.starts_at)}–${fmtHM(x.ends_at)}${x.room ? ` · ${x.room}` : ""}${x.topic ? ` · ${x.topic}` : ""}`}
                action={
                  <OverflowMenu
                    label={`Thêm với buổi ${x.session_no}`}
                    items={[
                      { label: "Sửa", onSelect: () => { m.update.reset(); setEditing(x.id); } },
                      { label: "Xoá", danger: true, disabled: x.attendance?.has_records, hint: x.attendance?.has_records ? "Đã có điểm danh" : undefined, onSelect: () => { if (pending) commitDelete(); setPending(x); } },
                    ]}
                  />
                }
              />
            ),
          )}
        </ActionList>
      )}
    </div>
  );
}

function EditRow({ row, busy, error, onCancel, onSave }: { row: SessionRow; busy: boolean; error: unknown; onCancel: () => void; onSave: (b: { starts_at: string; ends_at: string; room: string | null; topic: string | null }) => void }) {
  const [starts, setStarts] = useState(toLocalInput(row.starts_at));
  const [ends, setEnds] = useState(toLocalInput(row.ends_at));
  const [room, setRoom] = useState(row.room ?? "");
  const [topic, setTopic] = useState(row.topic ?? "");
  const errs = fieldErrors(error);
  const ok = starts !== "" && ends !== "";
  return (
    <Panel>
      <form
        className={s.sessionForm}
        onSubmit={(e) => {
          e.preventDefault();
          if (ok) onSave({ starts_at: fromLocalInput(starts), ends_at: fromLocalInput(ends), room: room.trim() || null, topic: topic.trim() || null });
        }}
      >
        <div className={s.formPair}>
          <Field label="Bắt đầu" error={errs.starts_at}>{(id) => <Input id={id} type="datetime-local" value={starts} onChange={(e) => setStarts(e.target.value)} />}</Field>
          <Field label="Kết thúc" error={errs.ends_at}>{(id) => <Input id={id} type="datetime-local" value={ends} onChange={(e) => setEnds(e.target.value)} invalid={Boolean(errs.ends_at)} />}</Field>
        </div>
        <div className={s.formPair}>
          <Field label="Phòng" error={errs.room}>{(id) => <Input id={id} value={room} maxLength={40} placeholder="P.302" onChange={(e) => setRoom(e.target.value)} />}</Field>
          <Field label="Chủ đề" error={errs.topic}>{(id) => <Input id={id} value={topic} maxLength={200} placeholder="Học bù" onChange={(e) => setTopic(e.target.value)} />}</Field>
        </div>
        {error != null && Object.keys(errs).length === 0 && <ApiErrorNotice error={error} />}
        <div className={s.formActions}>
          <Button type="submit" variant="primary" disabled={!ok || busy}>Lưu</Button>
          <Button type="button" onClick={onCancel}>Huỷ</Button>
        </div>
      </form>
    </Panel>
  );
}
