"use client";

import { useQueryClient } from "@tanstack/react-query";
import { useCallback, useEffect, useRef, useState } from "react";
import { ApiError, apiClient, newIdempotencyKey, useJob } from "@/shared/data";
import { checkDocumentFile, uploadDocumentFile } from "@/shared/data/uploadFile";
import { schemeStatusKey } from "./gradeApi";

export type Phase = "idle" | "uploading" | "reading" | "extracting" | "error";
export type UploadState = { phase: Phase; file?: File; docId?: string; ingestJob?: string; extractJob?: string; error?: string; canReplace?: boolean };

export interface PolicyUpload {
  state: UploadState;
  percent: number;
  start: (file: File) => Promise<void>;
  /** `Tải lại` (hỏng trước khi có tài liệu) hoặc `Thử lại` (đã có tài liệu: chỉ trích lại) */
  retry: () => void;
  replace: () => void;
  dismiss: () => void;
}

/**
 * `Tải quy chế`: tải tệp (URL ký sẵn) → đọc / lập chỉ mục (việc nền) → trích công thức (việc nền). Tiến độ lấy từ `useJob` (SSE `job.progress`, rơi về thăm dò).
 * Lỗi giữ tên tệp và cho `Tải lại` / `Thử lại`; bản nháp cũ không bị đụng cho tới khi việc trích xong.
 */
export function usePolicyUpload(course: string, onDone: () => void): PolicyUpload {
  const qc = useQueryClient();
  const [st, setSt] = useState<UploadState>({ phase: "idle" });
  const idem = useRef({ upload: "", extract: "" });
  const ingest = useJob(st.phase === "reading" ? st.ingestJob : undefined);
  const extract = useJob(st.phase === "extracting" ? st.extractJob : undefined);

  const runExtract = useCallback(
    async (docId: string, replace: boolean) => {
      try {
        const r = await apiClient.post<{ job_id: string }>(`/courses/${course}/grade-scheme/extract`, { document_id: docId, replace_draft: replace }, { idempotencyKey: idem.current.extract });
        setSt((p) => ({ ...p, phase: "extracting", extractJob: r.data.job_id, error: undefined, canReplace: false }));
      } catch (e) {
        idem.current.extract = newIdempotencyKey();
        const draftExists = e instanceof ApiError && e.code === ("DRAFT_EXISTS" as string);
        setSt((p) => ({ ...p, phase: "error", error: draftExists ? "Bản nháp đang có chỉnh sửa." : e instanceof ApiError ? e.userMessage : "Chưa trích được.", canReplace: draftExists }));
      }
    },
    [course],
  );

  const start = useCallback(
    async (file: File) => {
      const bad = checkDocumentFile(file);
      if (bad) return setSt({ phase: "error", file, error: bad });
      idem.current = { upload: newIdempotencyKey(), extract: newIdempotencyKey() };
      setSt({ phase: "uploading", file });
      try {
        const id = await uploadDocumentFile(course, file);
        const r = await apiClient.post<{ job_id: string; document: { id: string } }>(`/courses/${course}/uploads/complete`, { upload_id: id, title: file.name.replace(/\.[^.]+$/, ""), type: "COURSE_POLICY" }, { idempotencyKey: idem.current.upload });
        setSt({ phase: "reading", file, docId: r.data.document.id, ingestJob: r.data.job_id });
      } catch (e) {
        setSt({ phase: "error", file, error: e instanceof ApiError ? e.userMessage : "Chưa tải được." });
      }
    },
    [course],
  );

  useEffect(() => {
    if (st.phase !== "reading" || !st.docId) return;
    if (ingest.status === "SUCCEEDED") void runExtract(st.docId, false);
    else if (ingest.status === "FAILED") setSt((p) => ({ ...p, phase: "error", error: ingest.error || "Chưa đọc được tệp." }));
  }, [ingest.status, ingest.error, st.phase, st.docId, runExtract]);

  useEffect(() => {
    if (st.phase !== "extracting") return;
    if (extract.status === "SUCCEEDED") {
      setSt({ phase: "idle" });
      void qc.invalidateQueries({ queryKey: ["grade-schemes", course] });
      void qc.invalidateQueries({ queryKey: schemeStatusKey(course) });
      onDone();
    } else if (extract.status === "FAILED") {
      idem.current.extract = newIdempotencyKey();
      setSt((p) => ({ ...p, phase: "error", error: extract.error || "Chưa trích được." }));
    }
  }, [extract.status, extract.error, st.phase, qc, course, onDone]);

  const percent = st.phase === "uploading" ? 5 : st.phase === "reading" ? Math.round(5 + ingest.progress * 0.45) : st.phase === "extracting" ? Math.round(50 + extract.progress * 0.5) : 0;
  return {
    state: st,
    percent,
    start,
    retry: () => (st.docId ? void runExtract(st.docId, false) : st.file ? void start(st.file) : undefined),
    replace: () => (st.docId ? void runExtract(st.docId, true) : undefined),
    dismiss: () => setSt({ phase: "idle" }),
  };
}
