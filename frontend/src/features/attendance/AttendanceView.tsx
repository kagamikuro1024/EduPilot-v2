"use client";

import { useState } from "react";
import { at, fmtLongDate, fmtTime, type Student } from "@/mock/core";
import { absentSessions, rosterOf, sessionsOf } from "@/mock/roster";
import {
  ATTENDANCE_SEED,
  CURRENT_SESSION,
  KEYS,
  MEMBERS_SEED,
  SCHEMES_SEED,
  type AttendanceState,
  type Mark,
  type MembersState,
  type SchemesState,
  type SessionAttendance,
} from "@/mock/state";
import { useUndoLine } from "@/shared/lib/useUndoLine";
import { useSession } from "@/shared/session/session";
import { useDemoSlice } from "@/shared/state/demo";
import {
  Button,
  ButtonLink,
  DataTable,
  EmptyState,
  InlineNotice,
  Kbd,
  Page,
  PageHeader,
  PageState,
  Select,
  Skeleton,
  StatusText,
  Switch,
  Toolbar,
  useRouteState,
  type Column,
} from "@/shared/ui";
import s from "./AttendanceView.module.css";

const MARKS: Array<{ value: Mark; label: string; key: string; verb: string }> = [
  { value: "present", label: "Có mặt", key: "1", verb: "Đã đánh có mặt" },
  { value: "late", label: "Muộn", key: "2", verb: "Đã đánh muộn" },
  { value: "excused", label: "Vắng phép", key: "3", verb: "Đã đánh vắng phép" },
  { value: "absent", label: "Vắng", key: "4", verb: "Đã đánh vắng" },
];

/** Màn làm việc dày đặc, ưu tiên bàn phím: 30 sinh viên dưới 60 giây (DESIGN §14.8). */
export function AttendanceView() {
  const { course } = useSession();
  const [attendance, setAttendance] = useDemoSlice<AttendanceState>(KEYS.attendance, ATTENDANCE_SEED);
  const [members] = useDemoSlice<MembersState>(KEYS.members, MEMBERS_SEED);
  const [schemes] = useDemoSlice<SchemesState>(KEYS.schemes, SCHEMES_SEED);
  const [n, setN] = useState(CURRENT_SESSION);
  const [cursor, setCursor] = useState(0);
  const [offline, setOffline] = useState(false);
  const [queued, setQueued] = useState<SessionAttendance | null>(null);
  const [syncs, setSyncs] = useState(0);
  const undo = useUndoLine();
  const routeState = useRouteState();

  const roster = rosterOf(course.id, members);
  const sessions = sessionsOf(course.id);
  const session = sessions.find((x) => x.n === n);
  const stored = attendance[course.id]?.[n];

  const base: SessionAttendance = stored ?? {
    marks: Object.fromEntries(roster.map((st) => [st.id, session?.state === "recorded" && absentSessions(st).includes(n) ? "absent" : "present"])) as Record<string, Mark>,
    speaks: {},
    finalized: session?.state === "recorded",
  };
  const view = queued ?? base;
  const readOnly = session?.state === "recorded";

  function write(next: SessionAttendance) {
    if (offline) {
      setQueued(next);
      return;
    }
    setQueued(null);
    setSyncs((k) => k + 1);
    setAttendance((prev) => ({ ...prev, [course.id]: { ...prev[course.id], [n]: next } }));
  }

  function setMark(st: Student, mark: Mark) {
    if (readOnly) return;
    const before = view.marks[st.id] ?? "present";
    if (before === mark) return;
    write({ ...view, marks: { ...view.marks, [st.id]: mark } });
    undo.push(`${MARKS.find((m) => m.value === mark)?.verb} ${st.name}`, () => write({ ...view, marks: { ...view.marks, [st.id]: before } }));
  }

  function addSpeak(st: Student) {
    if (readOnly) return;
    const before = view.speaks[st.id] ?? 0;
    write({ ...view, speaks: { ...view.speaks, [st.id]: before + 1 } });
    undo.push(`Đã ghi phát biểu cho ${st.name} · +0,25`, () => write({ ...view, speaks: { ...view.speaks, [st.id]: before } }));
  }

  function toggleOffline(next: boolean) {
    setOffline(next);
    if (!next && queued) {
      setQueued(null);
      setSyncs((k) => k + 1);
      const flush = queued;
      setAttendance((prev) => ({ ...prev, [course.id]: { ...prev[course.id], [n]: flush } }));
    }
  }

  function onKeyDown(e: React.KeyboardEvent) {
    const st = roster[cursor];
    if (!st) return;
    if (e.key === "ArrowDown" || e.key === "ArrowUp") {
      e.preventDefault();
      setCursor((i) => Math.min(roster.length - 1, Math.max(0, i + (e.key === "ArrowDown" ? 1 : -1))));
      return;
    }
    const m = MARKS.find((x) => x.key === e.key);
    if (m) {
      e.preventDefault();
      setMark(st, m.value);
      return;
    }
    if (e.key.toLowerCase() === "p") {
      e.preventDefault();
      addSpeak(st);
    }
  }

  const count = (m: Mark) => roster.filter((st) => (view.marks[st.id] ?? "present") === m).length;
  // chỉ nêu loại có người: "27 có mặt, 1 muộn, 2 vắng"
  const summary = MARKS.map((m) => (count(m.value) > 0 ? `${count(m.value)} ${m.label.toLowerCase()}` : null))
    .filter(Boolean)
    .join(", ");

  const columns: Column<Student>[] = [
    {
      key: "name",
      header: "Sinh viên",
      frozen: true,
      width: "30%",
      render: (st) => (
        <span className={s.name}>
          <span className="ep-item-title">{st.name}</span>
          <span className="ep-meta">{st.code}</span>
        </span>
      ),
    },
    ...MARKS.map<Column<Student>>((m) => ({
      key: m.value,
      header: (
        <>
          {m.label} <Kbd>{m.key}</Kbd>
        </>
      ),
      align: "center",
      width: "13%",
      render: (st) => (
        <label className={s.cell}>
          <span className="ep-sr-only">{`${m.label} — ${st.name}`}</span>
          <input
            type="radio"
            name={`mark-${st.id}`}
            className={s.radio}
            disabled={readOnly}
            checked={(view.marks[st.id] ?? "present") === m.value}
            onChange={() => setMark(st, m.value)}
            onFocus={() => setCursor(roster.indexOf(st))}
          />
        </label>
      ),
    })),
    {
      key: "speak",
      header: (
        <>
          Phát biểu <Kbd>P</Kbd>
        </>
      ),
      align: "end",
      render: (st) => (
        <button type="button" className={s.speak} disabled={readOnly} onClick={() => addSpeak(st)}>
          {view.speaks[st.id] ? `${view.speaks[st.id]} lần · +${(view.speaks[st.id] * 0.25).toFixed(2).replace(".", ",")}` : "+ Phát biểu"}
        </button>
      ),
    },
  ];

  return (
    <Page width="wide">
      <PageHeader
        title="Điểm danh"
        description={session ? `${course.label} · buổi ${n} · ${fmtLongDate(session.date)} 09:00–11:30 · ${course.room}` : course.label}
        meta={
          session && (
            <>
              <span>Mặc định mọi sinh viên có mặt — chỉ đánh người vắng, muộn.</span>
              <span className={s.keyHint}>
                <Kbd>↑</Kbd> <Kbd>↓</Kbd> chọn hàng · <Kbd>1</Kbd>–<Kbd>4</Kbd> đánh dấu · <Kbd>P</Kbd> ghi phát biểu
              </span>
            </>
          )
        }
        actions={
          session?.state === "current" && (
            <Button
              variant="primary"
              onClick={() => {
                write({ ...view, finalized: true });
                undo.clear();
              }}
            >
              Lưu điểm danh
            </Button>
          )
        }
      />
      <PageState
        state={routeState}
        loading={<Skeleton lines={10} />}
        empty={<EmptyState title="Lớp này chưa có lịch buổi học">Tạo lịch buổi học ở bước thiết lập lớp, rồi quay lại điểm danh.</EmptyState>}
        error={{ problem: "Không lưu được điểm danh lên máy chủ.", recovery: "Các ô bạn đã đánh vẫn nằm trên máy và sẽ được gửi lại. Thử lại, hoặc tiếp tục đánh rồi lưu sau." }}
      >
        {sessions.length === 0 ? (
          <EmptyState title={`Lớp ${course.code} chưa có lịch buổi học`} action={<ButtonLink href="/">Mở việc thiết lập lớp</ButtonLink>}>
            Buổi học được tạo ở bước “Thiết lập lớp mới”. Sau khi có lịch, màn điểm danh sẽ mở đúng buổi đang diễn ra.
          </EmptyState>
        ) : (
          <>
            <Toolbar
              end={
                <>
                  <Switch label="Giả lập mất mạng" checked={offline} onChange={toggleOffline} />
                  {offline && queued ? (
                    <StatusText tone="amber">Đang chờ mạng · {changeCount(queued, base)} thay đổi</StatusText>
                  ) : syncs > 0 ? (
                    <StatusText tone="green">Đã lưu {fmtTime(at(syncs))}</StatusText>
                  ) : (
                    <StatusText tone="neutral">Chưa có thay đổi</StatusText>
                  )}
                </>
              }
            >
              <label className={s.picker}>
                <span className="ep-label">Buổi</span>
                <Select value={n} onChange={(e) => { setN(Number(e.target.value)); setQueued(null); }}>
                  {sessions.map((x) => (
                    <option key={x.n} value={x.n}>
                      {x.label}
                      {x.state === "current" ? " · đang diễn ra" : x.state === "future" ? " · chưa diễn ra" : ""}
                    </option>
                  ))}
                </Select>
              </label>
              <span className="ep-meta">{summary}</span>
            </Toolbar>

            {schemes[course.id]?.status !== "confirmed" && (
              <InlineNotice tone="info" compact>
                Lớp chưa xác nhận công thức điểm nên điểm cộng phát biểu chưa được tính — việc điểm danh vẫn ghi bình thường.
              </InlineNotice>
            )}
            {session?.state === "future" && (
              <EmptyState title="Buổi này chưa diễn ra">Buổi {n} bắt đầu lúc 09:00 ngày {session ? fmtLongDate(session.date) : ""}. Chọn buổi khác để xem hoặc sửa điểm danh.</EmptyState>
            )}
            {readOnly && (
              <InlineNotice tone="info" compact>
                Buổi {n} đã chốt điểm danh — chỉ xem. Cần sửa thì mở lại buổi ở sổ điểm.
              </InlineNotice>
            )}
            {view.finalized && session?.state === "current" && (
              <InlineNotice tone="success">Đã hoàn tất buổi {n} · {summary}</InlineNotice>
            )}

            {session?.state !== "future" && (
              <div className={s.grid} tabIndex={0} onKeyDown={onKeyDown} aria-label="Bảng điểm danh — phím mũi tên chọn hàng, phím 1 đến 4 đánh dấu, phím P ghi phát biểu">
                <DataTable
                  caption={`Điểm danh buổi ${n} lớp ${course.code}`}
                  columns={columns}
                  rows={roster}
                  rowKey={(st) => st.id}
                  activeKey={roster[cursor]?.id}
                  mobileRow={(st) => (
                    <div className={s.mRow}>
                      <span className={s.name}>
                        <span className="ep-item-title">{st.name}</span>
                        <span className="ep-meta">{st.code}</span>
                      </span>
                      <div className={s.mMarks} role="radiogroup" aria-label={`Điểm danh ${st.name}`}>
                        {MARKS.map((m) => (
                          <label key={m.value} className={s.mMark}>
                            <input
                              type="radio"
                              name={`m-mark-${st.id}`}
                              className={s.mRadio}
                              disabled={readOnly}
                              checked={(view.marks[st.id] ?? "present") === m.value}
                              onChange={() => setMark(st, m.value)}
                              onFocus={() => setCursor(roster.indexOf(st))}
                            />
                            <span>{m.label}</span>
                          </label>
                        ))}
                      </div>
                      <button type="button" className={s.speak} disabled={readOnly} onClick={() => addSpeak(st)}>
                        {view.speaks[st.id] ? `${view.speaks[st.id]} lần phát biểu · +${(view.speaks[st.id] * 0.25).toFixed(2).replace(".", ",")}` : "+ Phát biểu"}
                      </button>
                    </div>
                  )}
                />
              </div>
            )}
            {undo.node}
          </>
        )}
      </PageState>
    </Page>
  );
}

/** Số ô đã đổi so với bản đã lưu — con số trong "Đang chờ mạng · n thay đổi". */
function changeCount(next: SessionAttendance, saved: SessionAttendance): number {
  const marks = Object.keys(next.marks).filter((id) => next.marks[id] !== saved.marks[id]).length;
  const speaks = Object.keys(next.speaks).filter((id) => (next.speaks[id] ?? 0) !== (saved.speaks[id] ?? 0)).length;
  return marks + speaks;
}
