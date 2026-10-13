"use client";

import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import { ApiError, apiClient } from "@/shared/data";
import { SaveQueue, type QueueState, type Sender } from "@/shared/lib/saveQueue";
import { notifyUnsaved, registerUnsaved } from "@/shared/lib/unsavedWork";
import { queueKey, type Conflict, type Grid, type PutResult, type SpeakCreated, type Status } from "./attendanceApi";

/** Việc chờ gửi: đổi trạng thái (khoá = mã sinh viên) hoặc một lần phát biểu (khoá = `speak:<client_event_id>`). */
type Item = { kind: "mark"; status: Status } | { kind: "speak"; student_id: string };
type Base = Record<string, { status: Status; version: number | null }>;

function browserStore(key: string) {
  return {
    read: () => {
      try {
        return localStorage.getItem(key);
      } catch {
        return null;
      }
    },
    write: (v: string) => {
      try {
        localStorage.setItem(key, v);
      } catch {
        /* kho đầy: giữ trong bộ nhớ */
      }
    },
    remove: () => {
      try {
        localStorage.removeItem(key);
      } catch {
        /* bỏ qua */
      }
    },
  };
}

/** `details.conflicts` của 409 VERSION_CONFLICT (hợp đồng: openapi `AttendanceConflict`). */
function conflictsOf(e: ApiError): Conflict[] {
  const d = e.details;
  if (d && typeof d === "object" && "conflicts" in d && Array.isArray(d.conflicts)) return d.conflicts as Conflict[];
  return [];
}

export type Row = { id: string; name: string; code: string; status: Status; speaks: number; conflict: Conflict | null };

/**
 * Trạng thái điểm danh của một buổi (FEAT-crm-360 AC8, AC13): lựa chọn ghi NGAY vào kho ở máy (`SaveQueue`, khoá có người dùng), gửi `PUT` một lần cho cả lô sau 2 s yên lặng
 * và khi có mạng lại; phát biểu gửi ngay (mất mạng thì chờ trong cùng hàng đợi, `client_event_id` khử trùng). Xung đột 409 dừng hàng đợi, giữ chữ, chờ người dùng chọn từng dòng.
 */
export function useAttendance({ course, userId, grid }: { course: string; userId: string; grid: Grid }) {
  const session = grid.session.id;
  const [base, setBaseState] = useState<Base>(() => Object.fromEntries(grid.students.map((s) => [s.id, { status: s.record?.status ?? "PRESENT", version: s.record?.version ?? null }])));
  const baseRef = useRef(base);
  const setBase = useCallback((fn: (b: Base) => Base) => {
    baseRef.current = fn(baseRef.current);
    setBaseState(baseRef.current);
  }, []);
  const [delta, setDelta] = useState<Record<string, number>>({});
  const [conflicts, setConflicts] = useState<Conflict[]>([]);
  const [fatal, setFatal] = useState<string | null>(null);
  const [doneAt, setDoneAt] = useState<string | null>(grid.session.attendance_done_at);
  const [changedSinceDone, setChangedSinceDone] = useState(false);
  const [completed, setCompleted] = useState<{ total: number; absent: number } | null>(null);
  const completeRef = useRef(false);
  const speakIds = useRef(new Map<string, string>());

  const sender: Sender<Item> = async (items) => {
    const marks = items.flatMap((i) => (i.answer.kind === "mark" ? [{ id: i.item_id, status: i.answer.status }] : []));
    const speaks = items.flatMap((i) => (i.answer.kind === "speak" ? [{ eventId: i.item_id.slice("speak:".length), student: i.answer.student_id }] : []));
    if (marks.length > 0 || completeRef.current) {
      try {
        const complete = completeRef.current;
        // `Lưu điểm danh` ghi CẢ LƯỚI trong một lời gọi (hàng không đổi so với máy chủ bị bỏ qua, không xung đột); tự lưu chỉ gửi hàng đã đổi
        const overlay = new Map(marks.map((m) => [m.id, m.status]));
        const ids = complete ? grid.students.map((s) => s.id) : marks.map((m) => m.id);
        const res = await apiClient.put<PutResult>(`/courses/${course}/sessions/${session}/attendance`, {
          records: ids.map((id) => ({ student_id: id, status: overlay.get(id) ?? baseRef.current[id]?.status ?? "PRESENT", ...(baseRef.current[id]?.version != null ? { version: baseRef.current[id].version } : {}) })),
          complete,
        });
        setBase((b) => {
          const next = { ...b };
          for (const r of res.data.records) next[r.student_id] = { status: r.status, version: r.version };
          return next;
        });
        if (complete) {
          completeRef.current = false;
          setDoneAt(res.data.done_at);
        }
        setFatal(null);
      } catch (e) {
        if (e instanceof ApiError && e.code === "VERSION_CONFLICT") {
          const list = conflictsOf(e);
          if (list.length) {
            setConflicts(list);
            return { ok: false, fatal: true };
          }
        }
        if (e instanceof ApiError && e.status >= 400 && e.status < 500 && e.status !== 408 && e.status !== 429 && e.status !== 401) {
          completeRef.current = false;
          setFatal(e.userMessage);
          return { ok: false, fatal: true };
        }
        return { ok: false };
      }
    }
    for (const sp of speaks) {
      try {
        const res = await apiClient.post<SpeakCreated>(`/courses/${course}/participation`, { student_id: sp.student, session_id: session, type: "SPEAK", client_event_id: sp.eventId });
        speakIds.current.set(sp.eventId, res.data.id);
      } catch (e) {
        if (e instanceof ApiError && (e.status === 422 || e.status === 404)) continue; // sinh viên không còn trong lớp: bỏ lần ghi này, không kẹt hàng đợi
        return { ok: false };
      }
    }
    return { ok: true };
  };
  const [queue] = useState(
    () =>
      new SaveQueue<Item>(
        browserStore(queueKey(userId, course, session)),
        async () => ({ ok: false as const }), // hàm gửi thật đặt ở effect bên dưới; lần gửi đầu chỉ xảy ra sau ≥ 0 ms
        { set: (fn, ms) => window.setTimeout(fn, ms), clear: (h) => window.clearTimeout(h as number) },
        () => Date.now(),
      ),
  );
  useEffect(() => queue.setSender(sender)); // mỗi lần dựng lại: hàm gửi luôn đọc phiên bản / cờ mới nhất qua ref
  const [qstate, setQState] = useState<QueueState>(queue.state);
  useEffect(
    () =>
      queue.subscribe((s) => {
        setQState(s);
        notifyUnsaved();
      }),
    [queue],
  );
  useEffect(() => registerUnsaved(`attendance:${session}`, () => queue.state.pending), [queue, session]);

  // còn phần chưa gửi từ lần trước (tải lại / sập trình duyệt): gửi tiếp; rời màn / đổi buổi: gửi ngay phần đang chờ
  useEffect(() => {
    if (queue.state.pending > 0) queue.resume();
    const on = () => queue.setOnline(true);
    const off = () => queue.setOnline(false);
    const vis = () => document.visibilityState === "visible" && queue.resume();
    window.addEventListener("online", on);
    window.addEventListener("offline", off);
    document.addEventListener("visibilitychange", vis);
    if (!navigator.onLine) queue.setOnline(false);
    return () => {
      window.removeEventListener("online", on);
      window.removeEventListener("offline", off);
      document.removeEventListener("visibilitychange", vis);
      if (queue.state.pending > 0) queue.resume();
    };
  }, [queue]);

  const pending = queue.pending();
  const statusOf = (id: string): Status => {
    const p = pending[id];
    return p?.kind === "mark" ? p.status : (base[id]?.status ?? "PRESENT");
  };
  /** Trạng thái hiện hành đọc thẳng từ hàng đợi + bản máy chủ (dùng trong sự kiện, không phụ thuộc lần dựng gần nhất). */
  const currentStatus = useCallback(
    (id: string): Status => {
      const p = queue.pending()[id];
      return p?.kind === "mark" ? p.status : (baseRef.current[id]?.status ?? "PRESENT");
    },
    [queue],
  );

  const rows: Row[] = useMemo(
    () =>
      grid.students.map((s) => ({
        id: s.id,
        name: s.full_name,
        code: s.student_code ?? "",
        status: statusOf(s.id),
        speaks: Math.max(0, s.speak_count + (delta[s.id] ?? 0)),
        conflict: conflicts.find((c) => c.student_id === s.id) ?? null,
      })),
    // eslint-disable-next-line react-hooks/exhaustive-deps
    [grid.students, base, delta, conflicts, qstate],
  );

  const setMark = useCallback(
    (id: string, status: Status) => {
      if (currentStatus(id) === status) return;
      queue.set(id, { kind: "mark", status });
      setChangedSinceDone(true);
      setCompleted(null);
    },
    [queue, currentStatus],
  );

  /** Ghi một lần phát biểu: đếm tăng ngay, gửi ngay (không chờ 2 s). Trả mã sự kiện để hoàn tác. */
  const speak = useCallback(
    (studentId: string): string => {
      const eventId = crypto.randomUUID();
      queue.set(`speak:${eventId}`, { kind: "speak", student_id: studentId });
      setDelta((d) => ({ ...d, [studentId]: (d[studentId] ?? 0) + 1 }));
      queue.resume();
      return eventId;
    },
    [queue],
  );

  /** Hoàn tác phát biểu: chưa gửi thì bỏ khỏi hàng đợi, đã gửi thì `DELETE` (gửi lại cùng `client_event_id` trả đúng sự kiện cũ nếu chưa biết mã). */
  const undoSpeak = useCallback(
    async (studentId: string, eventId: string) => {
      setDelta((d) => ({ ...d, [studentId]: (d[studentId] ?? 0) - 1 }));
      const key = `speak:${eventId}`;
      if (key in queue.pending()) {
        queue.discard(key);
        return;
      }
      try {
        let id = speakIds.current.get(eventId);
        if (!id) id = (await apiClient.post<SpeakCreated>(`/courses/${course}/participation`, { student_id: studentId, session_id: session, type: "SPEAK", client_event_id: eventId })).data.id;
        await apiClient.delete(`/courses/${course}/participation/${id}`);
      } catch {
        setDelta((d) => ({ ...d, [studentId]: (d[studentId] ?? 0) + 1 })); // không hoàn tác được: đếm trả lại
      }
    },
    [queue, course, session],
  );

  /** Chọn một dòng xung đột: giữ của mình (gửi lại với phiên bản mới) hoặc lấy của người kia (bỏ thay đổi của mình ở dòng này). */
  const resolve = useCallback(
    (studentId: string, choice: "mine" | "theirs") => {
      const c = conflicts.find((x) => x.student_id === studentId);
      if (!c) return;
      setBase((b) => ({ ...b, [studentId]: { status: c.current.status, version: c.current.version } }));
      if (choice === "theirs") queue.discard(studentId);
      const left = conflicts.filter((x) => x.student_id !== studentId);
      setConflicts(left);
      if (left.length === 0) queue.restart();
    },
    [conflicts, queue, setBase],
  );

  /** `Lưu điểm danh`: gửi ngay mọi thay đổi đang chờ cùng `complete:true`. */
  const complete = useCallback(async () => {
    completeRef.current = true;
    if (queue.state.pending > 0) await queue.flush();
    if (completeRef.current && queue.state.pending === 0 && queue.state.status !== "stopped") {
      const r = await sender([]);
      if (!r.ok) return;
    }
    if (!completeRef.current) {
      setCompleted({ total: grid.students.length, absent: grid.students.filter((s) => currentStatus(s.id) === "ABSENT").length });
      setChangedSinceDone(false);
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [queue, grid.students, currentStatus]);

  return { rows, qstate, conflicts, fatal, doneAt, changedSinceDone, completed, setMark, speak, undoSpeak, resolve, complete, retry: () => queue.resume() };
}
