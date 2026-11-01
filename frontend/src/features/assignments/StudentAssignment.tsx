"use client";

import { useQueryClient } from "@tanstack/react-query";
import { useEffect, useRef, useState } from "react";
import { ApiError, ApiErrorNotice, newIdempotencyKey } from "@/shared/data";
import { checkSubmissionFile, uploadSubmissionFile } from "@/shared/data/uploadFile";
import { Markdown } from "@/shared/domain";
import { useSession } from "@/shared/session/session";
import { Button, DefinitionList, InlineNotice, Page, PageHeader, PageState, Panel, PanelSection, Skeleton } from "@/shared/ui";
import { WD, dayLabel, fmtDM, fmtHM, fmtLong } from "@/features/calendar/calendarApi";
import { AppealBlock } from "./AppealBlock";
import { completeSubmission, dueLabel, mineKey, useMine, vi, type MineOut, type MineResult } from "./assignmentApi";
import s from "./StudentAssignment.module.css";

// Vé tải lên còn hạn 15 phút: CHỈ siêu dữ liệu (không tệp, không token) để lần mở sau hoàn tất được (AC6).
const RESUME_MS = 15 * 60 * 1000;
type Resume = { upload_id: string; filename: string; size: number; ts: number };
const resumeKey = (course: string, id: string) => `ep:submit:${course}:${id}`;
function readResume(key: string): Resume | null {
  try {
    const r = JSON.parse(localStorage.getItem(key) ?? "") as Resume;
    return r?.upload_id && Date.now() - r.ts < RESUME_MS ? r : null;
  } catch {
    return null;
  }
}

/** "08:10 Thứ Sáu, 23/10" — dấu thời gian của MÁY CHỦ, giờ Việt Nam. */
const at = (iso: string) => `${fmtHM(iso)} ${WD[new Date(dayLabel(iso)).getUTCDay()]}, ${fmtDM(dayLabel(iso))}`;

/** Bài tập của sinh viên: đề bài, tiêu chí, nộp bài (US-P7-03). Không có mục sidebar: vào từ "Hôm nay", Lịch, `/me`. */
export function StudentAssignment({ courseId, id }: { courseId: string; id: string }) {
  const q = useMine(courseId, id);
  const [now] = useState(() => Date.now());
  const a = q.data?.assignment;
  return (
    <Page>
      <PageHeader
        title={a?.title ?? "Bài tập"}
        meta={
          a ? (
            <>
              <span>{fmtLong(a.due_at)}</span>
              <span data-part="due-left">{dueLabel(a.due_at, now)}</span>
            </>
          ) : undefined
        }
      />
      <Panel>
        <PageState query={{ isPending: q.isPending, isError: q.isError, error: q.error, data: q.data, refetch: q.refetch }} loading={<Skeleton lines={8} />}>
          {q.data && <Mine courseId={courseId} id={id} data={q.data} now={now} />}
        </PageState>
      </Panel>
    </Page>
  );
}

/** Kết quả ĐÃ CÔNG BỐ (US-P7-09 AC6): chỉ thứ sinh viên được thấy — không cờ, không điểm AI, không chữ "nháp". */
function Result({ r }: { r: MineResult }) {
  return (
    <PanelSection title="Kết quả">
      <div data-part="result">
        <p className={s.total} data-part="result-total">{`${vi(r.total)} / ${vi(r.max)}`}</p>
        {r.criteria.map((c) => (
          <div key={c.name} className={s.crit} data-part="result-criterion">
            <p className={s.critHead}>
              <span>{c.name}</span>
              <span className={s.critScore}>{`${vi(c.score)} / ${vi(c.max)}`}</span>
            </p>
            {c.comment && <p className={s.critText}>{c.comment}</p>}
            {c.evidence.map((q) => (
              <blockquote key={q} className={s.quote} data-part="result-evidence">
                {q}
              </blockquote>
            ))}
          </div>
        ))}
        {r.late && <p className={s.critText} data-part="result-late">{`Trừ nộp muộn: −${vi(r.late.penalty)} điểm`}</p>}
        {r.overall_feedback && <p className={s.critText} data-part="result-overall">{r.overall_feedback}</p>}
        <p className={s.status}>{r.label}</p>
      </div>
    </PanelSection>
  );
}

function Mine({ courseId, id, data, now }: { courseId: string; id: string; data: MineOut; now: number }) {
  const { identity } = useSession();
  const userId = identity?.sub;
  const qc = useQueryClient();
  const a = data.assignment;
  const key = resumeKey(courseId, id);
  const picker = useRef<HTMLInputElement>(null);
  const busy = useRef(false);
  const idem = useRef<string | null>(null);
  const [file, setFile] = useState<File | null>(null);
  const [fileError, setFileError] = useState<string | null>(null);
  const [sending, setSending] = useState(false);
  const [failed, setFailed] = useState(false);
  const [err, setErr] = useState<unknown>(null);
  const [receipt, setReceipt] = useState<string | null>(null);
  const [resume, setResume] = useState<Resume | null>(null);
  const [over, setOver] = useState(false);

  // kho của trình duyệt chỉ đọc được sau khi gắn kết (máy chủ dựng HTML không có nó)
  /* eslint-disable react-hooks/set-state-in-effect */
  useEffect(() => {
    setResume(readResume(key));
  }, [key]);
  /* eslint-enable react-hooks/set-state-in-effect */

  const overdue = new Date(a.due_at).getTime() < now;
  const sub = data.submission;
  const lateTag = sub && sub.late_days > 0 ? ` · Nộp muộn ${sub.late_days} ngày` : "";
  const status = receipt
    ? `Đã nhận bài lúc ${at(receipt)}`
    : data.state === "NOT_SUBMITTED" || !sub
      ? "Chưa nộp"
      : data.state === "GRADING"
        ? `Đang chấm${lateTag}`
        : data.state === "GRADED"
          ? "Đã có điểm"
          : `Đã nộp ${fmtLong(sub.received_at)}${lateTag}`;

  function choose(list: FileList | null) {
    const files = Array.from(list ?? []);
    setErr(null);
    setFailed(false);
    if (files.length === 0) return;
    if (files.length > 1) {
      setFile(null);
      setFileError("Chỉ nộp một tệp");
      return;
    }
    const bad = checkSubmissionFile(files[0]); // kiểm ở máy: sai thì KHÔNG có yêu cầu presign
    setFileError(bad);
    setFile(bad ? null : files[0]);
  }

  async function run(job: () => Promise<void>) {
    if (busy.current) return; // bấm đúp: đúng một lần nộp có hiệu lực
    busy.current = true;
    setErr(null);
    setFailed(false);
    setSending(true);
    try {
      await job();
    } catch (e) {
      const ae = e instanceof ApiError ? e : new ApiError({ status: 0, code: "NETWORK" });
      if (ae.code === "UPLOAD_NOT_FOUND") {
        localStorage.removeItem(key);
        setResume(null);
      }
      if (ae.code === "NETWORK" || ae.code === "UPLOAD_INCOMPLETE" || ae.code === "UPLOAD_NOT_FOUND" || ae.code === "BAD_GATEWAY" || ae.status >= 500) setFailed(true);
      else setErr(ae);
    } finally {
      busy.current = false;
      setSending(false);
    }
  }

  async function complete(uploadId: string) {
    idem.current ??= newIdempotencyKey();
    const out = await completeSubmission(courseId, id, uploadId, idem.current);
    idem.current = null;
    try {
      localStorage.removeItem(key);
    } catch { /* kho bị chặn */ }
    setResume(null);
    setFile(null);
    setReceipt(out.submission.received_at);
    await qc.invalidateQueries({ queryKey: mineKey(courseId, id) });
  }

  const send = () => {
    if (!file) return;
    const f = file;
    void run(async () => {
      const uploadId = await uploadSubmissionFile(courseId, id, f);
      try {
        localStorage.setItem(key, JSON.stringify({ upload_id: uploadId, filename: f.name, size: f.size, ts: Date.now() } satisfies Resume));
      } catch { /* kho bị chặn: vẫn nộp tiếp */ }
      await complete(uploadId);
    });
  };

  return (
    <>
      {data.result && <Result r={data.result} />}
      {data.result && <AppealBlock courseId={courseId} id={id} data={data} userId={userId} now={now} />}
      <PanelSection title="Đề bài">
        <Markdown source={a.description} />
      </PanelSection>
      <PanelSection title="Tiêu chí chấm">
        <DefinitionList items={a.criteria.map((c) => ({ term: c.name, value: `${vi(c.max_score)} điểm` }))} />
      </PanelSection>
      <PanelSection title="Bài nộp của bạn">
        {resume && (
          <InlineNotice
            tone="warning"
            compact
            action={
              <Button size="sm" loading={sending} onClick={() => void run(() => complete(resume.upload_id))}>
                Hoàn tất
              </Button>
            }
          >
            {`Chưa hoàn tất nộp · ${resume.filename}`}
          </InlineNotice>
        )}
        {overdue && sub && (
          <InlineNotice tone="warning" compact>
            Nộp lại sau hạn sẽ tính nộp muộn
          </InlineNotice>
        )}
        <div
          className={[s.drop, over ? s.over : ""].join(" ")}
          data-part="dropzone"
          role="group"
          aria-label="Vùng nộp bài: kéo thả tệp hoặc chọn tệp"
          onDragOver={(e) => {
            e.preventDefault();
            setOver(true);
          }}
          onDragLeave={() => setOver(false)}
          onDrop={(e) => {
            e.preventDefault();
            setOver(false);
            choose(e.dataTransfer.files);
          }}
        >
          <input
            ref={picker}
            type="file"
            className={s.picker}
            accept=".pdf,.docx,.txt,.md"
            aria-label="Chọn tệp bài nộp"
            onChange={(e) => {
              choose(e.target.files);
              e.target.value = "";
            }}
          />
          <Button onClick={() => picker.current?.click()}>Chọn tệp</Button>
          <p className={s.pick} data-part="picked">{file ? file.name : "Kéo thả tệp vào đây"}</p>
        </div>
        {fileError && (
          <p className={s.error} role="alert" data-part="file-error">
            {fileError}
          </p>
        )}
        {failed && (
          <InlineNotice
            tone="danger"
            compact
            action={
              <Button size="sm" loading={sending} onClick={file ? send : () => picker.current?.click()}>
                Thử lại
              </Button>
            }
          >
            Chưa gửi được
          </InlineNotice>
        )}
        {err != null && <ApiErrorNotice error={err} />}
        <div className={s.actions}>
          <Button variant="primary" className={s.submit} disabled={!file} loading={sending} onClick={send}>
            {sub ? "Nộp lại" : "Nộp bài"}
          </Button>
        </div>
        <p className={s.status} role="status" aria-live="polite" data-part="submit-status">
          {status}
        </p>
      </PanelSection>
    </>
  );
}
