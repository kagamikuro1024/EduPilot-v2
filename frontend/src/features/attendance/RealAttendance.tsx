"use client";

import { useQueryClient } from "@tanstack/react-query";
import { useRouter, useSearchParams } from "next/navigation";
import { useEffect, useMemo, useRef, useState } from "react";
import { AttendanceGrid, type AttendanceGridRow } from "@/shared/domain/AttendanceGrid";
import { useUndoLine } from "@/shared/lib/useUndoLine";
import { ActionList, ActionRow, Button, ButtonLink, ConfirmIrreversible, Drawer, EmptyState, FilterChips, InlineNotice, Page, PageHeader, PageState, Panel, Select, Skeleton, StatusText, Toolbar, type HeaderAction } from "@/shared/ui";
import { ApiError } from "@/shared/data";
import { useSchemeStatus, useSchemes } from "../gradebook/gradeApi";
import { formatWhen } from "@/shared/lib/datetime";
import { dayLabel, fmtDM, fmtHM } from "../calendar/calendarApi";
import { sessionsKey, useSessions, type SessionRow } from "../calendar/sessionsApi";
import { gridKey, labelOf, useGrid, type Grid } from "./attendanceApi";
import { CheckinWindowLine } from "./CheckinWindowLine";
import { isOpen, useCheckinActions, useCheckins, useCoursePolicy, useWindows, windowsKey, type CheckinRow, type CheckinWindow } from "./checkinApi";
import { useAttendance } from "./useAttendance";
import s from "./RealAttendance.module.css";

/** Điểm cộng mỗi lần phát biểu đọc từ công thức lớp (luật BONUS `per_speak`), dấu phẩy thập phân; chưa có công thức thì không có giá trị. */
function speakBonusOf(v: ReturnType<typeof useSchemes>["data"]): string | undefined {
  const rule = (v?.confirmed ?? v?.draft)?.rules.find((r) => r.kind === "BONUS" && r.applicable);
  const per = rule?.params.per_speak;
  return typeof per === "string" && per !== "" ? per.replace(".", ",") : undefined;
}

/** Buổi mặc định: đang diễn ra, hoặc buổi đã qua gần nhất; lớp chưa có buổi nào đã qua thì buổi sắp tới sớm nhất. */
function defaultSession(list: SessionRow[], now = Date.now()): SessionRow | undefined {
  const t = (x: string) => new Date(x).getTime();
  const current = list.find((x) => t(x.starts_at) <= now && now <= t(x.ends_at));
  if (current) return current;
  const past = list.filter((x) => t(x.ends_at) < now).sort((a, b) => t(b.ends_at) - t(a.ends_at))[0];
  return past ?? [...list].sort((a, b) => t(a.starts_at) - t(b.starts_at))[0];
}

export function RealAttendance({ courseId, userId, classCode }: { courseId: string; userId: string; classCode: string }) {
  const router = useRouter();
  const list = useSessions(courseId);
  const picked = useSearchParams().get("session");
  // gần nhất trước
  const sessions = useMemo(() => [...list.items].sort((a, b) => new Date(b.starts_at).getTime() - new Date(a.starts_at).getTime()), [list.items]);
  const current = sessions.find((x) => x.id === picked) ?? defaultSession(sessions);
  const grid = useGrid(courseId, current?.id);
  // điểm danh bằng mã (10.8): chỉ khi lớp bật; chưa biết chính sách / cửa sổ thì chưa dựng lưới (ô trống hay «Có mặt» phụ thuộc vào đó)
  const policy = useCoursePolicy(courseId);
  const selfCheckin = policy.data?.self_checkin === true;
  const windows = useWindows(courseId, current?.id, selfCheckin, policy.data?.rotate_seconds ?? 10);
  // chốt «đã biết» theo từng buổi: tải lại nền (mạng về, hết hạn) KHÔNG được đưa lưới về khung xương, vì Sheet dựng lại sẽ gửi hàng đợi hai lần
  const knownNow = !policy.isPending && !(selfCheckin && windows.isPending);
  const [knownFor, setKnownFor] = useState<string | null>(null);
  if (knownNow && current && knownFor !== current.id) setKnownFor(current.id);
  const checkinReady = knownNow || (current !== undefined && knownFor === current.id);
  const windowsFailed = (policy.isError && !policy.data) || (selfCheckin && windows.isError && !windows.data);

  const picker = current && (
    <label className={s.picker}>
      <span className="ep-label">Buổi</span>
      <Select data-part="att-session-select" value={current.id} onChange={(e) => router.replace(`/attendance?session=${e.target.value}`)}>
        {sessions.map((x) => (
          <option key={x.id} value={x.id}>
            {`Buổi ${x.session_no} · ${fmtDM(dayLabel(x.starts_at))} · ${x.attendance?.done_at ? "Đã điểm danh" : "Chưa"}`}
          </option>
        ))}
      </Select>
    </label>
  );

  // MỘT PageHeader cho mọi trạng thái (TL 14.6): nút cần dữ liệu hiện nhưng khoá; thanh dính đáy khoá nằm CUỐI trang để không chen giữa đầu trang và thân
  const noData = <PageHeader title="Điểm danh" primary={{ label: "Lưu điểm danh", disabled: true, docked: true }} />;
  const noDock = (
    <div className={s.saveBar} data-part="att-save-bar" data-ep-header-dock>
      <Button variant="primary" disabled>Lưu điểm danh</Button>
    </div>
  );
  if (list.isPending) return <Page kind="work">{noData}<Panel><Skeleton lines={10} /></Panel>{noDock}</Page>;
  if (list.isError) return <Page kind="work">{noData}<PageState query={{ isPending: false, isError: true, error: list.error, data: null, refetch: list.refetch }}>{null}</PageState>{noDock}</Page>;
  if (!current) {
    return (
      <Page kind="work">
        {noData}
        <Panel>
          <EmptyState title="Lớp chưa có lịch buổi học" action={<ButtonLink href="/calendar">Tạo lịch buổi học</ButtonLink>} />
        </Panel>
        {noDock}
      </Page>
    );
  }
  return (
    <Page kind="work">
      <PageState
        query={{ isPending: grid.isPending || !checkinReady, isError: grid.isError || windowsFailed, error: grid.error ?? policy.error ?? windows.error, data: grid.data && !windowsFailed ? grid.data : null, refetch: () => void Promise.all([grid.refetch(), policy.refetch(), windows.refetch()]) }}
        header={noData}
        loading={<>{noData}<Panel><Skeleton lines={10} /></Panel></>}
        showTechnical
      >
        {grid.data && checkinReady && (
          <Sheet
            key={`${current.id}:${grid.dataUpdatedAt}:${selfCheckin && (windows.data?.length ?? 0) > 0}`}
            courseId={courseId}
            userId={userId}
            classCode={classCode}
            grid={grid.data}
            picker={picker}
            selfCheckin={selfCheckin}
            rotate={policy.data?.rotate_seconds ?? 10}
            windows={selfCheckin ? (windows.data ?? []) : []}
          />
        )}
      </PageState>
      {!grid.data && noDock}
    </Page>
  );
}

/** Một buổi đã tải. `key` là mã buổi nên đổi buổi dựng lại hàng đợi riêng; phần đang chờ của buổi cũ vẫn được gửi (xem `useAttendance`). */
function Sheet({ courseId, userId, classCode, grid, picker, selfCheckin, rotate, windows }: { courseId: string; userId: string; classCode: string; grid: Grid; picker: React.ReactNode; selfCheckin: boolean; rotate: number; windows: CheckinWindow[] }) {
  const qc = useQueryClient();
  const router = useRouter();
  const hasWindow = selfCheckin && windows.length > 0;
  const a = useAttendance({ course: courseId, userId, grid, blank: hasWindow });
  const undo = useUndoLine();
  const st = grid.session;
  const scheme = useSchemeStatus(courseId).data;
  const speakBonus = speakBonusOf(useSchemes(courseId, false).data);

  // nháy nền xanh nhạt ở hàng vừa lưu xong (hàng đợi về trạng thái đã lưu), không toast (US-UX91-04 AC7)
  const dirty = useRef(new Set<string>());
  const [flash, setFlash] = useState<Set<string>>(new Set());
  const rows: AttendanceGridRow[] = a.rows.map((r) => ({ saved: flash.has(r.id), id: r.id, name: r.name, code: r.code, status: r.status, speaks: r.speaks, conflict: r.conflict ? { by: r.conflict.current.marked_by.full_name, theirs: r.conflict.current.status } : null }));
  const byName = a.conflicts[0]?.current.marked_by.full_name;
  const q = a.qstate;

  function setMark(id: string, status: NonNullable<AttendanceGridRow["status"]>) {
    const row = a.rows.find((r) => r.id === id);
    if (!row || row.status === status) return;
    dirty.current.add(id);
    a.setMark(id, status);
    const before = row.status;
    undo.push(`Đã đánh ${labelOf(status)} · ${row.name}`, () => (before === null ? a.unmark(id) : a.setMark(id, before)));
  }
  function speak(id: string) {
    const row = a.rows.find((r) => r.id === id);
    if (!row) return;
    dirty.current.add(id);
    const ev = a.speak(id);
    undo.push(`Đã ghi phát biểu · ${row.name}`, () => void a.undoSpeak(id, ev));
  }
  async function complete() {
    await a.complete();
    void qc.invalidateQueries({ queryKey: sessionsKey(courseId) });
    undo.clear();
  }

  // ---- điểm danh bằng mã (10.8) ----
  const open = windows.find((w) => isOpen(w)) ?? null;
  const latest = windows.reduce<CheckinWindow | null>((m, w) => (!m || w.round > m.round ? w : m), null);
  const acts = useCheckinActions(courseId, st.id);
  const checkins = useCheckins(courseId, st.id, hasWindow, open ? rotate * 1000 : false);
  const filter: "all" | "review" = useSearchParams().get("filter") === "review" ? "review" : "all";
  const [actionError, setActionError] = useState<string | null>(null);
  const [closeQueued, setCloseQueued] = useState(false);
  const [editingHeadcount, setEditingHeadcount] = useState(false);
  const [spotOpen, setSpotOpen] = useState(false);
  const [absentOpen, setAbsentOpen] = useState(false);
  const displayHref = `/attendance/display?course=${courseId}&session=${st.id}`;
  const refreshGrid = () => qc.invalidateQueries({ queryKey: gridKey(courseId, st.id) });

  const perStudent = useMemo(() => {
    type Entry = { at: string; flags: Map<string, string>; rows: CheckinRow[]; unverified: boolean; code: boolean };
    const m = new Map<string, Entry>();
    for (const r of checkins.data ?? []) {
      const e: Entry = m.get(r.student.id) ?? { at: r.created_at, flags: new Map(), rows: [], unverified: false, code: false };
      if (r.created_at < e.at) e.at = r.created_at;
      if (r.channel === "CODE") e.code = true;
      if (!r.cleared) {
        if (r.assurance === 1) e.unverified = true;
        for (const f of r.flags) e.flags.set(f.code, f.label);
        if (r.flags.length > 0) e.rows.push(r);
      }
      m.set(r.student.id, e);
    }
    return m;
  }, [checkins.data]);

  const needsReview = (id: string) => (perStudent.get(id)?.flags.size ?? 0) > 0;
  const reviewCount = hasWindow ? rows.filter((r) => needsReview(r.id)).length : 0;
  const gridRows: AttendanceGridRow[] = (hasWindow && filter === "review" ? rows.filter((r) => needsReview(r.id)) : rows).map((r) => {
    const e = hasWindow ? perStudent.get(r.id) : undefined;
    if (!e) return r;
    const chips: Array<{ label: string; tone: "warn" | "neutral" }> = [...e.flags.values()].map((label) => ({ label, tone: "warn" as const }));
    if (e.unverified) chips.push({ label: "Chưa xác minh", tone: "warn" });
    if (e.code) chips.push({ label: "Nhập mã", tone: "neutral" });
    return { ...r, checkin: { at: fmtHM(e.at), chips, review: e.flags.size > 0 ? () => void review(r.id) : undefined } };
  });

  // cửa sổ vừa đóng (người khác đóng / hết giờ / sweep): bản ghi có mặt - muộn do máy chủ ghi lúc đóng, tải lại lưới
  const wasOpen = useRef(false);
  const openId = open?.id ?? null;
  useEffect(() => {
    if (wasOpen.current && openId === null) void refreshGrid();
    wasOpen.current = openId !== null;
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [openId]);

  async function openWindow() {
    setActionError(null);
    try {
      await acts.open.mutateAsync();
      router.push(displayHref);
    } catch (e) {
      if (e instanceof ApiError && e.code === "WINDOW_ALREADY_OPEN") return void router.push(displayHref);
      setActionError(e instanceof ApiError ? e.userMessage : "Chưa mở được. Thử lại.");
    }
  }
  async function closeWindow() {
    if (!open) return;
    setActionError(null);
    try {
      await acts.close.mutateAsync(open.id);
      setCloseQueued(false);
      void refreshGrid();
    } catch (e) {
      if (e instanceof ApiError && e.status >= 400 && e.status < 500 && e.status !== 408 && e.status !== 429) {
        setCloseQueued(false);
        setActionError(e.userMessage);
        void qc.invalidateQueries({ queryKey: windowsKey(courseId, st.id) });
      } else setCloseQueued(true); // offline / 5xx: «Chưa đóng», thử lại tự động
    }
  }
  const retryClose = useRef(closeWindow);
  useEffect(() => {
    retryClose.current = closeWindow;
  });
  useEffect(() => {
    if (!closeQueued) return;
    const run = () => void retryClose.current();
    const t = setInterval(run, 5000);
    window.addEventListener("online", run);
    return () => {
      clearInterval(t);
      window.removeEventListener("online", run);
    };
  }, [closeQueued]);

  async function confirmAbsent() {
    try {
      await acts.absent.mutateAsync();
      a.clearUnresolved();
      setAbsentOpen(false);
      void refreshGrid();
    } catch {
      /* lỗi hiện trong hộp xác nhận */
    }
  }
  async function review(id: string) {
    const e = perStudent.get(id);
    if (!e) return;
    await Promise.all(e.rows.map((r) => acts.clear.mutateAsync(r.id).catch(() => undefined)));
  }
  function openSpot() {
    if (!latest) return;
    setSpotOpen(true);
    acts.spot.mutate(latest.id);
  }

  const qSavedAt = q.savedAt;
  useEffect(() => {
    if (!qSavedAt || q.pending > 0 || dirty.current.size === 0) return;
    setFlash(new Set(dirty.current));
    dirty.current.clear();
    const t = setTimeout(() => setFlash(new Set()), 400);
    return () => clearTimeout(t);
  }, [qSavedAt, q.pending]);

  const saved = !a.changedSinceDone && a.doneAt != null && q.pending === 0;
  const state =
    a.conflicts.length > 0 || a.fatal ? <StatusText tone="amber">Chưa lưu</StatusText>
    : q.status === "offline" ? <StatusText tone="amber">Chờ mạng · {q.pending} thay đổi</StatusText>
    : q.status === "pending" ? <StatusText tone="neutral">Đang lưu…</StatusText>
    : q.savedAt ? <StatusText tone="green">Đã lưu {fmtHM(q.savedAt)}</StatusText>
    : <StatusText tone="neutral">Chưa có thay đổi</StatusText>;

  const present = rows.filter((r) => r.status === "PRESENT" || r.status === "LATE").length;
  const absent = rows.length - present;
  const barState = q.status === "offline" ? "Chờ mạng" : saved ? "Đã lưu" : "Lưu điểm danh";
  const locked = saved || a.conflicts.length > 0 || q.status === "offline";
  const moreActions: HeaderAction[] = [];
  if (hasWindow && !open && windows.length < 3) moreActions.push({ label: `Mở vòng ${(latest?.round ?? 0) + 1}`, onClick: () => void openWindow() });
  if (hasWindow) {
    moreActions.push({ label: "Nhập số đầu người", onClick: () => setEditingHeadcount(true) });
    moreActions.push({ label: "Gọi ngẫu nhiên 5 người", onClick: openSpot });
  }
  const unchecked = rows.filter((r) => r.status === null).length;
  const absentCount = a.unresolved ?? unchecked;

  return (
    <div className={s.sheet}>
      <PageHeader
        title="Điểm danh"
        meta={[`buổi ${st.session_no} · ${formatWhen(st.starts_at)}–${fmtHM(st.ends_at)}${st.room ? ` · ${st.room}` : ""}`]}
        primary={{ label: saved ? "Đã lưu" : "Lưu điểm danh", disabled: locked, onClick: () => void complete(), docked: true }}
        secondary={[selfCheckin && !open && windows.length < 3 ? { label: "Mở điểm danh", onClick: () => void openWindow(), busy: acts.open.isPending } : undefined]}
        more={moreActions}
      />
      <Panel>
        {a.unresolved !== null && (
          <InlineNotice tone="warning" compact action={<Button size="sm" onClick={() => setAbsentOpen(true)}>Đánh vắng {a.unresolved} SV chưa điểm danh</Button>}>
            Còn {a.unresolved} sinh viên chưa điểm danh
          </InlineNotice>
        )}
        <Toolbar end={<span data-part="att-save-state">{state}</span>}>
          {picker}
          {hasWindow && (
            <FilterChips
              label="Lọc điểm danh"
              value={[filter]}
              onChange={(v) => {
                const next = v.find((x) => x !== filter) ?? filter;
                router.replace(`/attendance?session=${st.id}${next === "review" ? "&filter=review" : ""}`);
              }}
              options={[
                { value: "all", label: "Tất cả" },
                { value: "review", label: `Cần xem (${reviewCount})` },
              ]}
            />
          )}
        </Toolbar>
        {hasWindow && (
          <CheckinWindowLine
            open={open}
            latest={latest}
            checkedIn={perStudent.size}
            active={rows.length}
            summary={{ present: rows.filter((r) => r.status === "PRESENT").length, late: rows.filter((r) => r.status === "LATE").length, unchecked }}
            closeQueued={closeQueued}
            closing={acts.close.isPending}
            displayHref={displayHref}
            editingHeadcount={editingHeadcount}
            onHeadcount={(v) => latest && acts.headcount.mutate({ wid: (open ?? latest).id, headcount: v })}
            onClose={() => void closeWindow()}
            onAbsent={() => setAbsentOpen(true)}
            onExpire={() => {
              void qc.invalidateQueries({ queryKey: windowsKey(courseId, st.id) });
              void refreshGrid();
            }}
          />
        )}
        {actionError && <InlineNotice tone="warning" compact>{actionError}</InlineNotice>}
        {scheme && scheme.state !== "CONFIRMED" && (
          <InlineNotice tone="warning" compact action={<span className={s.warnAction}><ButtonLink href={`/gradebook/scheme?course=${courseId}`} size="sm" aria-label="Xem công thức"><span className={s.longText}>Xem công thức</span><span className={s.shortText}>Xem</span></ButtonLink></span>}>
            <span className={s.longText}>Công thức điểm chưa xác nhận</span>
            <span className={s.shortText}>Công thức chưa xác nhận</span>
          </InlineNotice>
        )}
        {a.conflicts.length > 0 && (
          <InlineNotice tone="warning" compact>
            {byName} vừa đổi {a.conflicts.length} dòng
          </InlineNotice>
        )}
        {a.fatal && <InlineNotice tone="danger" compact>{a.fatal}</InlineNotice>}
        {q.retryInMs != null && q.status !== "stopped" && q.status !== "offline" && (
          <InlineNotice tone="danger" compact action={<Button size="sm" onClick={a.retry}>Thử lại</Button>}>
            Chưa lưu được. Thử lại
          </InlineNotice>
        )}
        {a.completed && (
          <InlineNotice tone="success" compact>
            Đã lưu điểm danh · {a.completed.total} sinh viên · vắng {a.completed.absent}
          </InlineNotice>
        )}
        <AttendanceGrid rows={gridRows} caption={`Điểm danh buổi ${st.session_no} lớp ${classCode}`} speakBonus={speakBonus} stickyHeight="calc(100dvh - 18rem)" onSet={setMark} onSpeak={speak} onResolve={a.resolve} />
        {undo.node}
      </Panel>
      <div className={s.saveBar} data-part="att-save-bar" data-ep-header-dock>
        <Button variant="primary" disabled={locked} onClick={() => void complete()}>
          {barState} · {present} có mặt · {absent} vắng
        </Button>
      </div>
      <ConfirmIrreversible
        open={absentOpen}
        onClose={() => setAbsentOpen(false)}
        onConfirm={() => void confirmAbsent()}
        title={`Đánh vắng ${absentCount} sinh viên?`}
        consequence={`${absentCount} sinh viên chưa điểm danh thành Vắng.`}
        confirmLabel="Đánh vắng"
        loading={acts.absent.isPending}
        error={acts.absent.error ? (acts.absent.error instanceof ApiError ? acts.absent.error.userMessage : "Chưa đánh vắng được. Thử lại.") : undefined}
      />
      <Drawer open={spotOpen} onClose={() => setSpotOpen(false)} title="Gọi ngẫu nhiên">
        <ActionList label="Gọi ngẫu nhiên" loading={acts.spot.isPending ? 5 : undefined}>
          {(acts.spot.data ?? []).map((x) => (
            <ActionRow key={x.student_id} tone="neutral" title={x.full_name} context={x.student_code} meta={x.reason === "FLAGGED" ? "Có cờ" : "Ngẫu nhiên"} />
          ))}
        </ActionList>
      </Drawer>
    </div>
  );
}
