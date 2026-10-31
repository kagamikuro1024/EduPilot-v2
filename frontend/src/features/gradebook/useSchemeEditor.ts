"use client";

import { useQueryClient } from "@tanstack/react-query";
import { useCallback, useEffect, useRef, useState } from "react";
import { ApiError, fieldErrors } from "@/shared/data";
import { postImpact, putScheme, toWorking, schemesKey, schemeStatusKey, type Impact, type SchemeDoc, type Working } from "./gradeApi";

export type SaveState = "idle" | "saving" | "offline" | "error";

const RETRY_MS = 3000;
const retryable = (e: unknown) => !(e instanceof ApiError) || e.status === 0 || e.status === 408 || e.status === 429 || e.status >= 500;

/**
 * Bản nháp đang sửa: ô giữ chữ của chính nó, rời ô thì `commit` → lưu lạc quan (`PUT` tuần tự, mỗi lần MỘT yêu cầu, phần mới nhất thắng).
 * Mất mạng: giữ nguyên, tự gửi lại mỗi 3 s. `VERSION_CONFLICT`: nạp bản hiện tại + một dòng báo. Xem thử tác động chạy ngay sau mỗi lần `commit`.
 */
export function useSchemeEditor(course: string, server: SchemeDoc | null) {
  const qc = useQueryClient();
  const [working, setWorking] = useState<Working | null>(server ? toWorking(server) : null);
  const [doc, setDoc] = useState<SchemeDoc | null>(server);
  const [state, setState] = useState<SaveState>("idle");
  const [notice, setNotice] = useState<string | null>(null);
  const [errors, setErrors] = useState<Record<string, string>>({});
  const [impact, setImpact] = useState<Impact | null>(null);
  const ver = useRef(server?.version ?? 0);
  const pending = useRef<Working | null>(null);
  const busy = useRef(false);
  const seen = useRef(server ? `${server.id}:${server.version}` : "");

  // bản nháp đổi từ ngoài (trích xong, bản mới): nạp lại — trừ khi đang có phần chưa gửi
  useEffect(() => {
    const key = server ? `${server.id}:${server.version}` : "";
    if (key === seen.current) return;
    seen.current = key;
    if (pending.current || busy.current) return;
    setDoc(server);
    setWorking(server ? toWorking(server) : null);
    ver.current = server?.version ?? 0;
  }, [server]);

  const run = useCallback(async () => {
    if (busy.current) return;
    busy.current = true;
    try {
      while (pending.current) {
        const w = pending.current;
        pending.current = null;
        setState("saving");
        try {
          const out = await putScheme(course, ver.current, w);
          ver.current = out.version;
          seen.current = `${out.id}:${out.version}`;
          setDoc(out);
          setErrors({});
          setNotice(null);
          qc.setQueryData(schemesKey(course, false), (p: { confirmed: SchemeDoc | null; draft: SchemeDoc | null } | undefined) => (p ? { ...p, draft: out } : p));
          void qc.invalidateQueries({ queryKey: schemeStatusKey(course) });
          setState("idle");
        } catch (e) {
          if (e instanceof ApiError && e.code === "VERSION_CONFLICT") {
            const cur = (e.details as { current?: SchemeDoc } | undefined)?.current;
            if (cur) {
              ver.current = cur.version;
              seen.current = `${cur.id}:${cur.version}`;
              setDoc(cur);
              setWorking(toWorking(cur));
            }
            pending.current = null;
            setNotice("Có người vừa sửa công thức. Đã tải bản mới.");
            setState("idle");
          } else if (!retryable(e)) {
            setErrors(fieldErrors(e));
            setState("error");
          } else {
            pending.current ??= w;
            setState("offline");
            const wait = Promise.withResolvers<void>();
            setTimeout(wait.resolve, RETRY_MS);
            await wait.promise;
          }
        }
      }
    } finally {
      busy.current = false;
    }
  }, [course, qc]);

  const commit = useCallback(
    (next: Working) => {
      setWorking(next);
      pending.current = next;
      void run();
    },
    [run],
  );

  // xem thử: chạy lại sau mỗi lần commit (và lúc mở); lỗi xem thử không chặn việc sửa
  useEffect(() => {
    if (!working) return;
    const ctrl = new AbortController();
    postImpact(course, working, ctrl.signal).then(setImpact, () => {});
    return () => ctrl.abort();
  }, [course, working]);

  return { working, doc, state, notice, errors, impact, commit, dismissNotice: () => setNotice(null) };
}
