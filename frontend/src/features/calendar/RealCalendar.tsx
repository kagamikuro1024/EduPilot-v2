"use client";

import { ChevronLeft, ChevronRight } from "lucide-react";
import { useEffect, useRef, useState, useSyncExternalStore } from "react";
import { ApiErrorNotice } from "@/shared/data/ApiErrorNotice";
import { ActionList, ActionRow, Button, ConfirmIrreversible, EmptyState, Field, IconButton, Input, OverflowMenu, Page, PageHeader, PageState, Panel, SegmentedControl, Select, Skeleton, StatusText, Toolbar, UndoLine } from "@/shared/ui";
import {
  DAY, STATE_LABEL, TYPE_LABEL, WD_SHORT, dayLabel, fmtDM, fmtHM, fmtLong, fromLocalInput, isSoonExam, labelToIso, toLocalInput, useCalendar, useEventMutations, useIcsState, useIssueIcs, weekStartLabel,
  type CalItem,
} from "./calendarApi";
import s from "./Calendar.module.css";

type View = "week" | "month" | "list";
const MOBILE = "(max-width: 719px)";
const VIEW_KEY = "ep.calendar.view";
const VIEW_EVENT = "ep:calendar-view";

/** Kiểu xem đã chọn nhớ ở localStorage; đổi thì báo cho chính tab này (sự kiện `storage` chỉ tới tab khác). */
function subscribeView(cb: () => void) {
  window.addEventListener("storage", cb);
  window.addEventListener(VIEW_EVENT, cb);
  return () => {
    window.removeEventListener("storage", cb);
    window.removeEventListener(VIEW_EVENT, cb);
  };
}

function subscribeNarrow(cb: () => void) {
  const mql = window.matchMedia(MOBILE);
  mql.addEventListener("change", cb);
  return () => mql.removeEventListener("change", cb);
}

/** Lịch thật (DESIGN §14.16): tuần ≥ 720 px, danh sách < 720 px, có tháng; dữ liệu gộp lúc đọc từ API. */
export function RealCalendar({ courseId, staff }: { courseId: string; staff: boolean }) {
  const narrow = useSyncExternalStore(subscribeNarrow, () => window.matchMedia(MOBILE).matches, () => false);
  const stored = useSyncExternalStore(subscribeView, () => window.localStorage.getItem(VIEW_KEY), () => null);
  const view: View = stored === "week" || stored === "month" || stored === "list" ? stored : narrow ? "list" : "week";
  const [offset, setOffset] = useState(0);
  const [form, setForm] = useState<CalItem | "new" | null>(null);
  const [ics, setIcs] = useState(false);
  const [pending, setPending] = useState<CalItem | null>(null);
  const m = useEventMutations(courseId);

  const [now] = useState(() => Date.now());
  const today = dayLabel(now);
  const weekStart = weekStartLabel(today) + offset * 7 * DAY;
  const anchor = new Date(today);
  const monthFirst = Date.UTC(anchor.getUTCFullYear(), anchor.getUTCMonth() + offset, 1);
  const monthStart = weekStartLabel(monthFirst);
  const [from, to] = view === "week" ? [weekStart, weekStart + 7 * DAY] : view === "month" ? [monthStart, monthStart + 42 * DAY] : [today - DAY, today + 60 * DAY];
  const list = useCalendar(courseId, labelToIso(from), labelToIso(to));
  const items = list.items.filter((e) => e.id !== pending?.id);

  // Xoá: dòng biến ngay, "Hoàn tác" 5 giây, hết giờ mới gọi API; rời trang giữa chừng vẫn gọi.
  const cancelled = useRef(false); // "Hoàn tác" cũng gọi onDone ngay sau onUndo (cùng một lượt bấm) nên cần cờ
  const pendingRef = useRef<CalItem | null>(null);
  useEffect(() => { pendingRef.current = pending; });
  useEffect(() => () => { if (pendingRef.current) m.remove.mutate(pendingRef.current.id); }, []); // eslint-disable-line react-hooks/exhaustive-deps
  const commitDelete = () => {
    if (pending && !cancelled.current) m.remove.mutate(pending.id);
    cancelled.current = false;
    setPending(null);
  };

  const monthName = new Date(monthFirst);
  const title = view === "month" ? `Tháng ${monthName.getUTCMonth() + 1} năm ${monthName.getUTCFullYear()}` : view === "week" ? `Tuần ${fmtDM(weekStart)} – ${fmtDM(weekStart + 6 * DAY)}` : "Sắp tới";
  const edit = staff ? (e: CalItem) => setForm(e) : undefined;
  const del = staff ? (e: CalItem) => { if (pending) commitDelete(); setPending(e); } : undefined;

  return (
    <Page width="wide">
      <PageHeader
        title="Lịch"
        actions={
          <>
            <Button onClick={() => setIcs((v) => !v)} aria-expanded={ics}>Thêm vào lịch</Button>
            {staff && <Button variant="primary" onClick={() => setForm("new")}>Thêm sự kiện</Button>}
          </>
        }
      />
      {ics && <IcsPanel />}
      {form && (
        <EventForm
          key={form === "new" ? "new" : form.id}
          item={form === "new" ? null : form}
          error={m.create.error ?? m.update.error}
          busy={m.create.isPending || m.update.isPending}
          onCancel={() => setForm(null)}
          onSave={(b) => {
            const o = { onSuccess: () => setForm(null) };
            if (form === "new") m.create.mutate(b, o);
            else m.update.mutate({ ...b, id: form.id, version: form.version }, o);
          }}
        />
      )}
      <Toolbar
        end={
          view === "list" ? undefined : (
            <span className={s.nav}>
              <IconButton label="Khoảng trước" size="sm" onClick={() => setOffset((o) => o - 1)}><ChevronLeft aria-hidden /></IconButton>
              <Button size="sm" onClick={() => setOffset(0)}>Hôm nay</Button>
              <IconButton label="Khoảng sau" size="sm" onClick={() => setOffset((o) => o + 1)}><ChevronRight aria-hidden /></IconButton>
            </span>
          )
        }
      >
        <SegmentedControl
          label="Kiểu xem lịch"
          value={view}
          onChange={(v) => {
            setOffset(0);
            window.localStorage.setItem(VIEW_KEY, v);
            window.dispatchEvent(new Event(VIEW_EVENT));
          }}
          options={[{ value: "week", label: "Tuần" }, { value: "month", label: "Tháng" }, { value: "list", label: "Danh sách" }]}
        />
        <span className={s.range}>{title}</span>
      </Toolbar>
      {pending && <UndoLine message="Đã xoá sự kiện" onUndo={() => { cancelled.current = true; setPending(null); }} onDone={commitDelete} />}
      {m.remove.isError && <ApiErrorNotice error={m.remove.error} />}

      <PageState
        query={list}
        isEmpty={() => items.length === 0}
        loading={<Panel><Skeleton lines={8} /></Panel>}
        empty={
          <Panel>
            <EmptyState title="Chưa có gì trong lịch." action={staff ? <Button variant="primary" onClick={() => setForm("new")}>Thêm sự kiện</Button> : undefined} />
          </Panel>
        }
      >
        {view === "week" && <WeekView start={weekStart} today={today} items={items} now={now} onEdit={edit} onDelete={del} />}
        {view === "month" && <MonthView first={monthFirst} start={monthStart} today={today} items={items} now={now} />}
        {view === "list" && <ListView items={items.filter((e) => new Date(e.ends_at ?? e.starts_at).getTime() >= now - 3 * 3_600_000)} now={now} onEdit={edit} onDelete={del} />}
      </PageState>
    </Page>
  );
}

const kindClass = (e: CalItem, now: number) => (isSoonExam(e, now) ? s.exam : s.session);

function ItemMenu({ e, onEdit, onDelete }: { e: CalItem; onEdit?: (e: CalItem) => void; onDelete?: (e: CalItem) => void }) {
  if (!e.editable || !onEdit || !onDelete) return null;
  return <OverflowMenu label={`Thêm với ${e.title}`} items={[{ label: "Sửa", onSelect: () => onEdit(e) }, { label: "Xoá", onSelect: () => onDelete(e) }]} />;
}

function Line({ e, now, onEdit, onDelete }: { e: CalItem; now: number; onEdit?: (e: CalItem) => void; onDelete?: (e: CalItem) => void }) {
  const inner = (
    <>
      <span className={s.eventTime}>{fmtHM(e.starts_at)}</span>
      <span className={s.eventTitle} data-part="cal-event">{e.title}</span>
      {e.personal_state && <span className={s.eventNote}>{STATE_LABEL[e.personal_state]}</span>}
    </>
  );
  return (
    <span className={[s.event, kindClass(e, now)].join(" ")} data-kind={e.source}>
      {e.href ? <a href={e.href} className={s.eventLink}>{inner}</a> : inner}
      <ItemMenu e={e} onEdit={onEdit} onDelete={onDelete} />
    </span>
  );
}

function WeekView({ start, today, items, now, onEdit, onDelete }: { start: number; today: number; items: CalItem[]; now: number; onEdit?: (e: CalItem) => void; onDelete?: (e: CalItem) => void }) {
  return (
    <Panel>
      <div className={s.week}>
        {Array.from({ length: 7 }, (_, i) => start + i * DAY).map((d, i) => {
          const day = items.filter((e) => dayLabel(e.starts_at) === d);
          return (
            <section key={d} className={[s.day, d === today ? s.today : ""].join(" ")}>
              <h3 className={s.dayHead}>
                <span className={s.dayName}>{WD_SHORT[i]}</span>
                <span className={s.dayNum}>{fmtDM(d)}</span>
              </h3>
              {day.length === 0 ? <p className={s.dayEmpty}>—</p> : day.map((e) => <Line key={e.id} e={e} now={now} onEdit={onEdit} onDelete={onDelete} />)}
            </section>
          );
        })}
      </div>
    </Panel>
  );
}

function MonthView({ first, start, today, items, now }: { first: number; start: number; today: number; items: CalItem[]; now: number }) {
  const month = new Date(first).getUTCMonth();
  return (
    <Panel>
      <div className={s.month}>
        {WD_SHORT.map((d) => <p key={d} className={s.monthHead}>{d}</p>)}
        {Array.from({ length: 42 }, (_, i) => start + i * DAY).map((d) => {
          const day = items.filter((e) => dayLabel(e.starts_at) === d);
          return (
            <div key={d} className={[s.cell, new Date(d).getUTCMonth() !== month ? s.otherMonth : "", d === today ? s.today : ""].join(" ")}>
              <span className={s.cellNum}>{new Date(d).getUTCDate()}</span>
              {day.slice(0, 2).map((e) => <span key={e.id} className={[s.chip, kindClass(e, now)].join(" ")} aria-label={e.title}>{e.title}</span>)}
              {day.length > 2 && <span className={s.more}>+{day.length - 2} việc khác</span>}
            </div>
          );
        })}
      </div>
    </Panel>
  );
}

function ListView({ items, now, onEdit, onDelete }: { items: CalItem[]; now: number; onEdit?: (e: CalItem) => void; onDelete?: (e: CalItem) => void }) {
  if (items.length === 0) return <Panel><EmptyState title="Không còn sự kiện nào sắp tới." /></Panel>;
  return (
    <Panel>
      <ActionList label="Sự kiện sắp tới">
        {items.map((e) => {
          const running = new Date(e.starts_at).getTime() <= now && new Date(e.ends_at ?? e.starts_at).getTime() >= now;
          return (
            <ActionRow
              key={e.id}
              tone={isSoonExam(e, now) ? "red" : "neutral"}
              href={e.href ?? undefined}
              title={e.title}
              context={`${fmtLong(e.starts_at)}${e.ends_at ? `–${fmtHM(e.ends_at)}` : ""}${e.location ? ` · ${e.location}` : ""}`}
              meta={<>{TYPE_LABEL[e.type]}{running && <> · <StatusText tone="green">Đang diễn ra</StatusText></>}{e.personal_state ? ` · ${STATE_LABEL[e.personal_state]}` : ""}</>}
              action={<ItemMenu e={e} onEdit={onEdit} onDelete={onDelete} />}
            />
          );
        })}
      </ActionList>
    </Panel>
  );
}

function EventForm({ item, error, busy, onCancel, onSave }: { item: CalItem | null; error: unknown; busy: boolean; onCancel: () => void; onSave: (b: { type: "EXAM" | "OTHER"; title: string; starts_at: string; ends_at: string | null; location: string | null }) => void }) {
  const [type, setType] = useState<"EXAM" | "OTHER">(item?.type === "EXAM" ? "EXAM" : "OTHER");
  const [title, setTitle] = useState(item?.title ?? "");
  const [starts, setStarts] = useState(item ? toLocalInput(item.starts_at) : "");
  const [ends, setEnds] = useState(item?.ends_at ? toLocalInput(item.ends_at) : "");
  const [where, setWhere] = useState(item?.location ?? "");
  const ok = title.trim() !== "" && starts !== "";
  return (
    <Panel>
      <form
        className={s.form}
        onSubmit={(e) => {
          e.preventDefault();
          if (ok) onSave({ type, title: title.trim(), starts_at: fromLocalInput(starts), ends_at: ends ? fromLocalInput(ends) : null, location: where.trim() || null });
        }}
      >
        <Field label="Loại">{(id) => <Select id={id} value={type} onChange={(e) => setType(e.target.value as "EXAM" | "OTHER")}><option value="OTHER">Sự kiện khác</option><option value="EXAM">Bài thi</option></Select>}</Field>
        <Field label="Tên sự kiện">{(id) => <Input id={id} value={title} maxLength={120} onChange={(e) => setTitle(e.target.value)} autoFocus />}</Field>
        <Field label="Bắt đầu">{(id) => <Input id={id} type="datetime-local" value={starts} onChange={(e) => setStarts(e.target.value)} />}</Field>
        <Field label="Kết thúc">{(id) => <Input id={id} type="datetime-local" value={ends} onChange={(e) => setEnds(e.target.value)} />}</Field>
        <Field label="Địa điểm">{(id) => <Input id={id} value={where} maxLength={80} onChange={(e) => setWhere(e.target.value)} />}</Field>
        <div className={s.formActions}>
          <Button type="submit" variant="primary" disabled={!ok || busy}>Lưu</Button>
          <Button type="button" onClick={onCancel}>Huỷ</Button>
        </div>
        {error != null && <ApiErrorNotice error={error} />}
      </form>
    </Panel>
  );
}

/** "Thêm vào lịch": URL ICS chỉ hiện một lần ngay sau khi tạo; có rồi mà mất thì đặt lại. */
function IcsPanel() {
  const state = useIcsState(true);
  const issue = useIssueIcs();
  const [url, setUrl] = useState<string | null>(null);
  const [confirm, setConfirm] = useState(false);
  const [copied, setCopied] = useState(false);
  const exists = state.data?.exists ?? false;
  const make = () => issue.mutate(undefined, { onSuccess: (u) => { setUrl(u); setConfirm(false); } });
  return (
    <Panel>
      <div className={s.ics} data-part="ics-panel">
        {issue.isError && <ApiErrorNotice error={issue.error} />}
        {url ? (
          <>
            <Field label="Liên kết lịch">{(id) => <Input id={id} readOnly value={url} onFocus={(e) => e.currentTarget.select()} />}</Field>
            <Button onClick={() => void navigator.clipboard.writeText(url).then(() => setCopied(true))}>Sao chép</Button>
            {copied && <UndoLine message="Đã sao chép" onDone={() => setCopied(false)} />}
          </>
        ) : exists ? (
          <StatusText>Mất liên kết thì đặt lại</StatusText>
        ) : (
          <Button variant="primary" onClick={make} disabled={issue.isPending || state.isPending}>Tạo liên kết</Button>
        )}
        {(url || exists) && <Button onClick={() => setConfirm(true)}>Đặt lại liên kết</Button>}
      </div>
      <ConfirmIrreversible
        open={confirm}
        onClose={() => setConfirm(false)}
        onConfirm={make}
        title="Đặt lại liên kết lịch?"
        consequence="Liên kết cũ ngừng hoạt động; lịch đã đăng ký ở Google, Outlook hay iPhone sẽ không cập nhật nữa."
        confirmLabel="Đặt lại"
        loading={issue.isPending}
      />
    </Panel>
  );
}
