"use client";

import { useEffect, useState } from "react";
import { apiClient } from "./apiClient";
import { ApiError } from "./ApiError";
import { sseManager } from "./sse";
import { useSSEStatus } from "./useSSE";

export type JobStatus = "QUEUED" | "RUNNING" | "SUCCEEDED" | "FAILED";
type Job = { id: string; status: JobStatus; progress: number; result?: unknown; error?: { code: string; message: string } };
export type JobState = { status: JobStatus | "UNKNOWN"; progress: number; result?: unknown; error?: string };

const POLL_MS = 2000;
const done = (s: string) => s === "SUCCEEDED" || s === "FAILED";

/**
 * Theo dõi một việc dài (PG 202 `{job_id}`): tải `GET /jobs/{id}` TRƯỚC (tránh đua), rồi nghe SSE `job.progress`;
 * khi SSE `degraded` / chưa mở thì thăm dò mỗi 2 s; dừng khi SUCCEEDED / FAILED. `progress` không bao giờ giảm.
 */
export function useJob(jobId: string | null | undefined): JobState {
  const [state, setState] = useState<JobState>({ status: "UNKNOWN", progress: 0 });
  const sse = useSSEStatus();

  const apply = (next: { status: string; progress?: number; result?: unknown; error?: { code: string; message: string } }) =>
    setState((prev) => {
      if (done(prev.status)) return prev;
      const progress = Math.max(prev.progress, next.progress ?? 0);
      return { status: next.status as JobStatus, progress: done(next.status) && next.status === "SUCCEEDED" ? 100 : progress, result: next.result ?? prev.result, error: next.error ? next.error.message || next.error.code : prev.error };
    });

  // 1) tải trạng thái hiện tại (đổi việc ⇒ đặt lại trạng thái)
  useEffect(() => {
    // eslint-disable-next-line react-hooks/set-state-in-effect
    setState({ status: "UNKNOWN", progress: 0 });
    if (!jobId) return;
    const ctrl = new AbortController();
    apiClient.get<Job>(`/jobs/${jobId}`, { signal: ctrl.signal }).then(
      (r) => apply(r.data),
      () => {},
    );
    return () => ctrl.abort();
  }, [jobId]);

  // 2) nghe SSE
  useEffect(() => {
    if (!jobId) return;
    return sseManager.on("job.progress", (e) => {
      try {
        const d = JSON.parse(e.data) as { job_id: string; status: string; progress: number; result?: unknown };
        if (d.job_id === jobId) apply(d);
      } catch { /* khung hỏng: bỏ qua */ }
    });
  }, [jobId]);

  // 3) thăm dò khi SSE không mở
  const live = sse === "open";
  useEffect(() => {
    if (!jobId || live || done(state.status)) return;
    const ctrl = new AbortController();
    const t = setInterval(() => {
      apiClient.get<Job>(`/jobs/${jobId}`, { signal: ctrl.signal }).then(
        (r) => apply(r.data),
        (e) => { if (e instanceof ApiError && e.status === 404) clearInterval(t); },
      );
    }, POLL_MS);
    return () => {
      clearInterval(t);
      ctrl.abort();
    };
  }, [jobId, live, state.status]);

  return state;
}
