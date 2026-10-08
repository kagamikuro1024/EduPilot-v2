"use client";

import { useCallback, useEffect, useRef, useState } from "react";
import { ApiError } from "@/shared/data";
import type { ExamClock } from "@/shared/lib/examClock";
import { saveDraft, type Lang } from "./codeApi";
import { getMine, isRunning } from "./takeApi";

const DEBOUNCE_MS = 2000;
const BACKOFF_MS = [1000, 2000, 4000, 8000, 15000];
const LANGS: Lang[] = ["c11", "cpp17"];

type Slot = { source: string; rev: number; dirty: boolean; editedAt: number };
type Persisted = { source: string; base_rev: number; dirty: boolean; savedAt: number };
export type DraftStatus = "saved" | "pending" | "offline" | "conflict" | "stopped";
export type Conflict = { lang: Lang; currentRev: number; updatedAt: string | null };

const keyOf = (attempt: string, item: string, lang: Lang) => `exam-code:${attempt}:${item}:${lang}`;

function readStored(key: string): Persisted | null {
  try {
    const v = JSON.parse(localStorage.getItem(key) ?? "null") as Persisted | null;
    return v && typeof v.source === "string" ? v : null;
  } catch {
    return null;
  }
}
function writeStored(key: string, v: Persisted) {
  try {
    localStorage.setItem(key, JSON.stringify(v));
  } catch {
    /* kho đầy: giữ trong bộ nhớ */
  }
}

/**
 * Bản nháp mã theo ngôn ngữ (US-PE-06 AC4–AC5): gõ → ghi vào máy NGAY (`exam-code:<lượt>:<câu>:<ngôn ngữ>`) → gửi sau 2 s ngừng gõ với `base_rev`;
 * mất mạng / lỗi máy chủ → giữ và thử lại 1-2-4-8-15 s; `DRAFT_CONFLICT` → KHÔNG ghi đè, dừng tự lưu và để người dùng chọn bản nào (chữ đang gõ giữ nguyên).
 */
export function useCodeDraft({ courseId, examId, attemptId, itemId, tab, clock, canWrite, initial, starter, onFatal }: {
  courseId: string;
  examId: string;
  attemptId: string;
  itemId: string;
  tab: string;
  clock: ExamClock;
  canWrite: boolean;
  /** bản nháp máy chủ đã trả lúc mở lượt */
  initial: Partial<Record<Lang, { source: string; rev: number }>>;
  starter: Partial<Record<Lang, string>>;
  onFatal: (why: "other" | "closed") => void;
}) {
  const [slots, setSlots] = useState<Record<Lang, Slot>>(() => {
    const out = {} as Record<Lang, Slot>;
    for (const l of LANGS) {
      const srv = initial[l];
      const loc = readStored(keyOf(attemptId, itemId, l));
      if (loc && loc.dirty) out[l] = { source: loc.source, rev: loc.base_rev, dirty: true, editedAt: loc.savedAt }; // phần còn giữ ở máy: mới hơn bản máy chủ
      else if (srv) out[l] = { source: srv.source, rev: srv.rev, dirty: false, editedAt: 0 };
      else out[l] = { source: starter[l] ?? "", rev: 0, dirty: false, editedAt: 0 };
    }
    return out;
  });
  const [status, setStatus] = useState<DraftStatus>("saved");
  const [savedAt, setSavedAt] = useState<number | null>(null);
  const [conflict, setConflict] = useState<Conflict | null>(null);
  const ref = useRef(slots);
  const timer = useRef<number | null>(null);
  const fails = useRef(0);
  const busy = useRef<Promise<void> | null>(null);
  const again = useRef<() => void>(() => undefined); // thử lại sau lỗi: trỏ tới `pump` mới nhất
  const stopped = useRef(!canWrite);
  const online = useRef(true);
  const fatal = useRef(onFatal);
  useEffect(() => {
    fatal.current = onFatal;
  });

  const commit = useCallback((next: Record<Lang, Slot>) => {
    ref.current = next;
    setSlots(next);
  }, []);

  const persist = useCallback(
    (l: Lang, s: Slot) => writeStored(keyOf(attemptId, itemId, l), { source: s.source, base_rev: s.rev, dirty: s.dirty, savedAt: Date.now() }),
    [attemptId, itemId],
  );

  const clearTimer = () => {
    if (timer.current !== null) window.clearTimeout(timer.current);
    timer.current = null;
  };

  /** Gửi mọi ngôn ngữ còn dirty, lần lượt. Trả khi hết hoặc gặp lỗi. */
  const pump = useCallback(async () => {
    if (busy.current) return busy.current;
    const run = (async () => {
      // bản gõ gần nhất gửi SAU CÙNG: `updated_at` của máy chủ nhờ đó chỉ đúng ngôn ngữ người dùng đang làm (AC4: tải lại mở ngôn ngữ của bản lưu gần nhất)
      for (const l of [...LANGS].sort((a, b) => ref.current[a].editedAt - ref.current[b].editedAt)) {
        const s = ref.current[l];
        if (!s.dirty || stopped.current) continue;
        const sentSource = s.source;
        try {
          const r = await saveDraft(clock, courseId, examId, attemptId, itemId, tab, { language: l, source: sentSource, base_rev: s.rev });
          const cur = ref.current[l];
          const still = cur.source !== sentSource; // gõ thêm trong lúc gửi: vẫn còn dirty
          const nxt = { ...cur, rev: r.rev, dirty: still };
          commit({ ...ref.current, [l]: nxt });
          persist(l, nxt);
          fails.current = 0;
          setSavedAt(Date.parse(r.saved_at));
          setStatus("saved");
        } catch (e) {
          if (e instanceof ApiError) {
            const code = e.code as string;
            if (code === "DRAFT_CONFLICT") {
              const d = (e.details ?? {}) as { current_rev?: number; updated_at?: string };
              stopped.current = true;
              setConflict({ lang: l, currentRev: d.current_rev ?? 0, updatedAt: d.updated_at ?? null });
              setStatus("conflict");
              return;
            }
            if (code === "ATTEMPT_OTHER_TAB") {
              stopped.current = true;
              setStatus("stopped");
              fatal.current("other");
              return;
            }
            if (code === "ATTEMPT_CLOSED") {
              stopped.current = true;
              setStatus("stopped");
              fatal.current("closed");
              return;
            }
            if (e.status >= 400 && e.status < 500 && e.status !== 429 && e.status !== 408) {
              stopped.current = true;
              setStatus("stopped");
              return;
            }
          }
          fails.current += 1;
          setStatus(online.current ? "pending" : "offline");
          timer.current = window.setTimeout(() => again.current(), BACKOFF_MS[Math.min(fails.current - 1, BACKOFF_MS.length - 1)]);
          return;
        }
      }
    })();
    busy.current = run.finally(() => {
      busy.current = null;
    });
    return busy.current;
  }, [clock, courseId, examId, attemptId, itemId, tab, commit, persist]);

  useEffect(() => {
    again.current = () => void pump();
  }, [pump]);

  const schedule = useCallback(
    (ms: number) => {
      clearTimer();
      timer.current = window.setTimeout(() => void pump(), ms);
    },
    [pump],
  );

  useEffect(() => {
    stopped.current = !canWrite;
    if (canWrite && !conflict && LANGS.some((l) => ref.current[l].dirty)) schedule(0); // gửi lại phần còn giữ ở máy (tải lại / giành lại quyền ghi)
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [canWrite]);

  useEffect(() => {
    const on = () => {
      online.current = true;
      if (!stopped.current && LANGS.some((l) => ref.current[l].dirty)) schedule(0);
    };
    const off = () => {
      online.current = false;
      if (LANGS.some((l) => ref.current[l].dirty)) setStatus("offline");
    };
    if (!navigator.onLine) off();
    window.addEventListener("online", on);
    window.addEventListener("offline", off);
    return () => {
      window.removeEventListener("online", on);
      window.removeEventListener("offline", off);
      clearTimer();
    };
  }, [schedule]);

  const setSource = useCallback(
    (l: Lang, v: string) => {
      const s = { ...ref.current[l], source: v, dirty: true, editedAt: Date.now() };
      commit({ ...ref.current, [l]: s });
      persist(l, s);
      setStatus(online.current ? "pending" : "offline");
      if (!stopped.current) schedule(DEBOUNCE_MS);
    },
    [commit, persist, schedule],
  );

  /** Gửi NGAY (không chờ debounce): hết giờ / trước khi nộp bài thi. Trả khi không còn gì chờ hoặc đã thử một lần. */
  const flush = useCallback(async () => {
    clearTimer();
    if (busy.current) await busy.current;
    await pump();
  }, [pump]);

  /** Xung đột → `Dùng bản trên máy này`: gửi lại với `base_rev` = bản máy chủ hiện tại (ghi đè CÓ CHỦ ĐÍCH). */
  const keepMine = useCallback(() => {
    if (!conflict) return;
    const s = { ...ref.current[conflict.lang], rev: conflict.currentRev, dirty: true, editedAt: Date.now() };
    commit({ ...ref.current, [conflict.lang]: s });
    persist(conflict.lang, s);
    stopped.current = !canWrite;
    setConflict(null);
    setStatus("pending");
    schedule(0);
  }, [conflict, canWrite, commit, persist, schedule]);

  /** Xung đột → `Dùng bản đã lưu`: nạp bản trên máy chủ (đọc lại lượt), bỏ phần trên máy. */
  const takeServer = useCallback(async () => {
    if (!conflict) return;
    const m = await getMine(clock, courseId, examId, tab);
    const it = isRunning(m) ? m.items.find((x) => x.item_id === itemId) : undefined;
    const d = (it?.code as { drafts?: Partial<Record<Lang, { source: string; rev: number }>> } | null | undefined)?.drafts?.[conflict.lang];
    const s = { source: d?.source ?? "", rev: d?.rev ?? conflict.currentRev, dirty: false, editedAt: 0 };
    commit({ ...ref.current, [conflict.lang]: s });
    persist(conflict.lang, s);
    stopped.current = !canWrite;
    setConflict(null);
    setStatus("saved");
  }, [conflict, canWrite, clock, courseId, examId, itemId, tab, commit, persist]);

  /** Nạp mã vào ô soạn (ví dụ `Dùng lại mã này` từ một lần nộp). */
  const replace = setSource;

  const dirtyCount = LANGS.filter((l) => slots[l].dirty).length;
  const nonEmpty = LANGS.some((l) => slots[l].source.trim() !== "");
  return { slots, setSource, replace, status, savedAt, conflict, keepMine, takeServer, flush, dirtyCount, nonEmpty };
}
