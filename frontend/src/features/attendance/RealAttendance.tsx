"use client";

import { useQueryClient } from "@tanstack/react-query";
import { useRouter, useSearchParams } from "next/navigation";
import { useEffect, useMemo, useRef, useState } from "react";
import { AttendanceGrid, type AttendanceGridRow } from "@/shared/domain/AttendanceGrid";
import { useUndoLine } from "@/shared/lib/useUndoLine";
import { Button, ButtonLink, EmptyState, InlineNotice, Page, PageHeader, PageState, Panel, Select, Skeleton, StatusText, Toolbar } from "@/shared/ui";
import { useSchemeStatus, useSchemes } from "../gradebook/gradeApi";
import { formatWhen } from "@/shared/lib/datetime";
import { dayLabel, fmtDM, fmtHM } from "../calendar/calendarApi";
import { sessionsKey, useSessions, type SessionRow } from "../calendar/sessionsApi";
import { labelOf, useGrid, type Grid } from "./attendanceApi";
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
  const scheme = useSchemeStatus(courseId).data;
  const speakBonus = speakBonusOf(useSchemes(courseId, false).data);

  // nháy nền xanh nhạt ở hàng vừa lưu xong (hàng đợi về trạng thái đã lưu), không toast (US-UX91-04 AC7)
  const dirty = useRef(new Set<string>());
  const [flash, setFlash] = useState<Set<string>>(new Set());
  const rows: AttendanceGridRow[] = a.rows.map((r) => ({ saved: flash.has(r.id), id: r.id, name: r.name, code: r.code, status: r.status, speaks: r.speaks, conflict: r.conflict ? { by: r.conflict.current.marked_by.full_name, theirs: r.conflict.current.status } : null }));
  const byName = a.conflicts[0]?.current.marked_by.full_name;
  const q = a.qstate;

  function setMark(id: string, status: AttendanceGridRow["status"]) {
    const row = a.rows.find((r) => r.id === id);
    if (!row || row.status === status) return;
    dirty.current.add(id);
    a.setMark(id, status);
    undo.push(`Đã đánh ${labelOf(status)} · ${row.name}`, () => a.setMark(id, row.status));
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

  return (
    <div className={s.sheet}>
      <PageHeader
        title="Điểm danh"
        description={`buổi ${st.session_no} · ${formatWhen(st.starts_at)}–${fmtHM(st.ends_at)}${st.room ? ` · ${st.room}` : ""}`}
        actions={
          <span className={s.desktopOnly}>
            <Button variant={saved ? "secondary" : "primary"} disabled={saved || a.conflicts.length > 0} onClick={() => void complete()}>
              {saved ? "Đã lưu" : "Lưu điểm danh"}
            </Button>
          </span>
        }
      />
      <Panel>
        <Toolbar end={<span data-part="att-save-state">{state}</span>}>{picker}</Toolbar>
        {scheme && scheme.state !== "CONFIRMED" && (
          <InlineNotice tone="warning" compact action={<ButtonLink href={`/gradebook/scheme?course=${courseId}`} size="sm">Xem công thức</ButtonLink>}>
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
        <AttendanceGrid rows={rows} caption={`Điểm danh buổi ${st.session_no} lớp ${classCode}`} speakBonus={speakBonus} stickyHeight="calc(100dvh - 18rem)" onSet={setMark} onSpeak={speak} onResolve={a.resolve} />
        {undo.node}
      </Panel>
      <div className={s.saveBar} data-part="att-save-bar">
        <Button variant="primary" disabled={saved || a.conflicts.length > 0 || q.status === "offline"} onClick={() => void complete()}>
          {barState} · {present} có mặt · {absent} vắng
        </Button>
      </div>
    </div>
  );
}
