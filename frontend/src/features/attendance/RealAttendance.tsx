"use client";

import { useQueryClient } from "@tanstack/react-query";
import { useRouter, useSearchParams } from "next/navigation";
import { useMemo } from "react";
import { AttendanceGrid, type AttendanceGridRow } from "@/shared/domain/AttendanceGrid";
import { useUndoLine } from "@/shared/lib/useUndoLine";
import { Button, ButtonLink, EmptyState, InlineNotice, Page, PageHeader, PageState, Panel, Select, Skeleton, StatusText, Toolbar } from "@/shared/ui";
import { WD, dayLabel, fmtDM, fmtHM } from "../calendar/calendarApi";
import { sessionsKey, useSessions, type SessionRow } from "../calendar/sessionsApi";
import { labelOf, useGrid, type Grid } from "./attendanceApi";
import { useAttendance } from "./useAttendance";
import s from "./RealAttendance.module.css";

const SPEAK_LABEL = "+ phát biểu · +0,25";
const dateLabel = (iso: string) => `${WD[new Date(dayLabel(iso)).getUTCDay()]}, ${fmtDM(dayLabel(iso))}`;

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

  if (list.isPending) return <Page width="wide"><PageHeader title="Điểm danh" /><Panel><Skeleton lines={10} /></Panel></Page>;
  if (list.isError) return <Page width="wide"><PageHeader title="Điểm danh" /><PageState query={{ isPending: false, isError: true, error: list.error, data: null, refetch: list.refetch }}>{null}</PageState></Page>;
  if (!current) {
    return (
      <Page width="wide">
        <PageHeader title="Điểm danh" />
        <Panel>
          <EmptyState title="Lớp chưa có lịch buổi học." action={<ButtonLink href="/calendar" variant="primary">Tạo lịch buổi học</ButtonLink>} />
        </Panel>
      </Page>
    );
  }
  return (
    <Page width="wide">
      <PageState
        query={{ isPending: grid.isPending, isError: grid.isError, error: grid.error, data: grid.data, refetch: grid.refetch }}
        loading={<><PageHeader title="Điểm danh" /><Panel><Skeleton lines={10} /></Panel></>}
        showTechnical
      >
        {grid.data && <Sheet key={current.id} courseId={courseId} userId={userId} classCode={classCode} grid={grid.data} picker={picker} />}
      </PageState>
    </Page>
  );
}

/** Một buổi đã tải. `key` là mã buổi nên đổi buổi dựng lại hàng đợi riêng; phần đang chờ của buổi cũ vẫn được gửi (xem `useAttendance`). */
function Sheet({ courseId, userId, classCode, grid, picker }: { courseId: string; userId: string; classCode: string; grid: Grid; picker: React.ReactNode }) {
  const qc = useQueryClient();
  const a = useAttendance({ course: courseId, userId, grid });
  const undo = useUndoLine();
  const st = grid.session;

  const rows: AttendanceGridRow[] = a.rows.map((r) => ({ id: r.id, name: r.name, code: r.code, status: r.status, speaks: r.speaks, conflict: r.conflict ? { by: r.conflict.current.marked_by.full_name, theirs: r.conflict.current.status } : null }));
  const byName = a.conflicts[0]?.current.marked_by.full_name;
  const q = a.qstate;

  function setMark(id: string, status: AttendanceGridRow["status"]) {
    const row = a.rows.find((r) => r.id === id);
    if (!row || row.status === status) return;
    a.setMark(id, status);
    undo.push(`Đã đánh ${labelOf(status)} · ${row.name}`, () => a.setMark(id, row.status));
  }
  function speak(id: string) {
    const row = a.rows.find((r) => r.id === id);
    if (!row) return;
    const ev = a.speak(id);
    undo.push(`Đã ghi phát biểu · ${row.name}`, () => void a.undoSpeak(id, ev));
  }
  async function complete() {
    await a.complete();
    void qc.invalidateQueries({ queryKey: sessionsKey(courseId) });
    undo.clear();
  }

  const saved = !a.changedSinceDone && a.doneAt != null && q.pending === 0;
  const state =
    a.conflicts.length > 0 || a.fatal ? <StatusText tone="amber">Chưa lưu</StatusText>
    : q.status === "offline" ? <StatusText tone="amber">Chờ mạng · {q.pending} thay đổi</StatusText>
    : q.status === "pending" ? <StatusText tone="neutral">Đang lưu…</StatusText>
    : q.savedAt ? <StatusText tone="green">Đã lưu {fmtHM(q.savedAt)}</StatusText>
    : <StatusText tone="neutral">Chưa có thay đổi</StatusText>;

  return (
    <>
      <PageHeader
        title="Điểm danh"
        description={`${classCode} · buổi ${st.session_no} · ${dateLabel(st.starts_at)} ${fmtHM(st.starts_at)}–${fmtHM(st.ends_at)}${st.room ? ` · ${st.room}` : ""}`}
        actions={
          <Button variant={saved ? "secondary" : "primary"} disabled={saved || a.conflicts.length > 0} onClick={() => void complete()}>
            {saved ? "Đã lưu" : "Lưu điểm danh"}
          </Button>
        }
      />
      <Panel>
        <Toolbar end={<span data-part="att-save-state">{state}</span>}>{picker}</Toolbar>
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
        <AttendanceGrid rows={rows} caption={`Điểm danh buổi ${st.session_no} lớp ${classCode}`} speakLabel={SPEAK_LABEL} onSet={setMark} onSpeak={speak} onResolve={a.resolve} />
        {undo.node}
      </Panel>
    </>
  );
}
