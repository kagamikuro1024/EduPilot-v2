"use client";

import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import { ApiError, ApiErrorNotice, useSSE } from "@/shared/data";
import { Markdown } from "@/shared/domain";
import type { ExamClock } from "@/shared/lib/examClock";
import { useMinWidth } from "@/shared/lib/useMinWidth";
import { Button, InlineNotice, Select, Tabs } from "@/shared/ui";
import { CodeEditor } from "./CodeEditor";
import { getRun, getSubmission, listSubmissions, runCode, submitCode, type Lang, type RunView, type SubmissionView } from "./codeApi";
import { RunResult, SubmissionList, visible } from "./CodeResults";
import type { TakeItem } from "./takeApi";
import { useCodeDraft } from "./useCodeDraft";
import s from "./CodeQuestion.module.css";

const POLL_MS = 2000;
const hhmmss = (ms: number) => new Date(ms + 7 * 3600_000).toISOString().slice(11, 19);
const LANG_LABEL: Record<Lang, string> = { c11: "C (C11)", cpp17: "C++ (C++17)" };

type CodeItem = {
  languages: Lang[];
  time_limit_ms: number;
  memory_limit_mb: number;
  starter_code: Partial<Record<Lang, string>>;
  samples: Array<{ name: string; input: string; expected: string }>;
  language: Lang | null;
  drafts: Partial<Record<Lang, { source: string; rev: number; saved_at: string }>>;
};

type Props = {
  item: TakeItem;
  number: number;
  courseId: string;
  examId: string;
  attemptId: string;
  tab: string;
  clock: ExamClock;
  canWrite: boolean;
  expired: boolean;
  onFatal: (why: "other" | "closed") => void;
  /** đăng ký hàm gửi NGAY bản nháp (trước khi nộp bài thi); trả hàm huỷ đăng ký */
  registerFlush: (fn: () => Promise<void>) => () => void;
  onAnswered: (itemId: string, answered: boolean) => void;
};

/** Một câu lập trình trong lượt làm (US-PE-06). < 1024 px: chỉ đề + dải báo cần màn hình rộng, KHÔNG có ô soạn mã. */
export function CodeQuestion(p: Props) {
  const wide = useMinWidth(1024);
  const code = p.item.code as unknown as CodeItem;
  return (
    <div className={[s.cols, wide ? "" : s.narrow].join(" ")} data-part="code-question">
      <section className={s.left} aria-label="Đề bài">
        <p className={s.meta}>Câu {p.number} · {p.item.points.replace(".", ",")} điểm · {code.time_limit_ms} ms · {code.memory_limit_mb} MB</p>
        <Markdown source={p.item.stem} />
        {code.samples.length > 0 && (
          <ul className={s.samples} aria-label="Test mẫu">
            {code.samples.map((x) => (
              <li key={x.name} className={s.panel}>
                <p className={s.sampleTitle}>{x.name}</p>
                <pre className={s.sampleBlock}>{`Vào:\n${visible(x.input)}\nRa:\n${visible(x.expected)}`}</pre>
              </li>
            ))}
          </ul>
        )}
      </section>
      {wide ? (
        <WideEditor {...p} code={code} />
      ) : (
        <InlineNotice tone="warning" compact>
          Bài lập trình cần màn hình rộng hơn (từ 1.024 px). Hãy mở bài thi này trên máy tính — bài của bạn vẫn là một lượt duy nhất và tự đồng bộ.
        </InlineNotice>
      )}
    </div>
  );
}

function WideEditor({ item, number, courseId, examId, attemptId, tab, clock, canWrite, expired, onFatal, registerFlush, onAnswered, code }: Props & { code: CodeItem }) {
  const [lang, setLang] = useState<Lang>(code.language ?? code.languages[0]);
  const draft = useCodeDraft({
    courseId, examId, attemptId, itemId: item.item_id, tab, clock, canWrite, onFatal,
    initial: Object.fromEntries(Object.entries(code.drafts ?? {}).map(([l, d]) => [l, { source: d!.source, rev: d!.rev }])) as Partial<Record<Lang, { source: string; rev: number }>>,
    starter: code.starter_code,
  });
  const source = draft.slots[lang].source;
  const locked = !canWrite || expired;

  // gửi ngay bản nháp khi hết giờ (AC12) và khi người làm bài nộp cả bài thi
  const flushRef = useRef(draft.flush);
  useEffect(() => {
    flushRef.current = draft.flush;
  });
  useEffect(() => registerFlush(() => flushRef.current()), [registerFlush]);
  const flushedAtExpiry = useRef(false);
  useEffect(() => {
    if (expired && !flushedAtExpiry.current) {
      flushedAtExpiry.current = true;
      void flushRef.current();
    }
  }, [expired]);

  // ---- chạy thử
  const [run, setRun] = useState<RunView | null>(null);
  const [runBusy, setRunBusy] = useState(false);
  const [runErr, setRunErr] = useState<unknown>(null);
  const runKey = useRef<string | null>(null);
  const runId = useRef<string | null>(null);
  async function doRun() {
    setRunErr(null);
    setRunBusy(true);
    runKey.current ??= crypto.randomUUID();
    try {
      const r = await runCode(courseId, examId, attemptId, item.item_id, tab, runKey.current, { language: lang, source });
      runKey.current = null;
      runId.current = r.run_id;
      setRun({ id: r.run_id, status: "QUEUED", language: lang, created_at: new Date().toISOString(), compile_ok: null, samples: [] });
    } catch (e) {
      if (e instanceof ApiError && e.status !== 0 && e.status < 500) runKey.current = null; // lỗi rõ ràng: lần sau là ý định mới
      setRunErr(e);
    } finally {
      setRunBusy(false);
    }
  }
  const pullRun = useCallback(async () => {
    const id = runId.current;
    if (!id) return;
    try {
      const v = await getRun(courseId, examId, attemptId, id);
      setRun((prev) => (prev && prev.id === id ? v : prev)); // kết quả theo id: không lặp, không ghi đè lần chạy mới hơn
    } catch {
      /* mạng chập chờn: lần thăm dò sau */
    }
  }, [courseId, examId, attemptId]);
  const runPending = run !== null && (run.status === "QUEUED" || run.status === "RUNNING");
  useEffect(() => {
    if (!runPending) return;
    const t = window.setInterval(() => void pullRun(), POLL_MS);
    return () => window.clearInterval(t);
  }, [runPending, pullRun]);
  useSSE("exam.run", (e) => {
    try {
      if ((JSON.parse(e.data) as { run_id?: string }).run_id === runId.current) void pullRun();
    } catch {
      /* sự kiện hỏng: bỏ */
    }
  });

  // ---- nộp lời giải + lịch sử
  const [tabSel, setTabSel] = useState<"run" | "subs">("run");
  const [subs, setSubs] = useState<SubmissionView[]>([]);
  const [next, setNext] = useState<string | null>(null);
  const [moreBusy, setMoreBusy] = useState(false);
  const [subBusy, setSubBusy] = useState(false);
  const [subErr, setSubErr] = useState<unknown>(null);
  const subKey = useRef<string | null>(null);
  const loadSubs = useCallback(async () => {
    try {
      const r = await listSubmissions(courseId, examId, attemptId, item.item_id);
      setSubs(r.items);
      setNext(r.next_cursor);
    } catch {
      /* lần sau */
    }
  }, [courseId, examId, attemptId, item.item_id]);
  useEffect(() => {
    let alive = true;
    listSubmissions(courseId, examId, attemptId, item.item_id).then(
      (r) => {
        if (!alive) return;
        setSubs(r.items);
        setNext(r.next_cursor);
      },
      () => undefined,
    );
    return () => {
      alive = false;
    };
  }, [courseId, examId, attemptId, item.item_id]);
  async function doSubmit() {
    setSubErr(null);
    setSubBusy(true);
    subKey.current ??= crypto.randomUUID();
    try {
      await draft.flush();
      await submitCode(courseId, examId, attemptId, item.item_id, tab, subKey.current, { language: lang, source });
      subKey.current = null;
      setTabSel("subs");
      await loadSubs();
    } catch (e) {
      if (e instanceof ApiError && e.status !== 0 && e.status < 500) subKey.current = null;
      setSubErr(e);
    } finally {
      setSubBusy(false);
    }
  }
  const pendingIds = useMemo(() => subs.filter((x) => x.status === "QUEUED" || x.status === "RUNNING").map((x) => x.id), [subs]);
  const pullSubs = useCallback(async () => {
    const got = await Promise.all(pendingIds.map((id) => getSubmission(courseId, examId, attemptId, id).catch(() => null)));
    setSubs((prev) => prev.map((x) => got.find((g) => g?.id === x.id) ?? x)); // thay đúng dòng theo id: không lặp
  }, [pendingIds, courseId, examId, attemptId]);
  useEffect(() => {
    if (pendingIds.length === 0) return;
    const t = window.setInterval(() => void pullSubs(), POLL_MS);
    return () => window.clearInterval(t);
  }, [pendingIds, pullSubs]);
  useSSE("exam.submission", () => void loadSubs()); // trạng thái đổi → đọc lại danh sách (còn lại dùng thăm dò 2 s khi SSE đứt)

  // ---- đếm "đã làm"
  const answered = draft.nonEmpty || subs.length > 0;
  useEffect(() => onAnswered(item.item_id, answered), [answered, item.item_id, onAnswered]);

  const status =
    draft.status === "offline" ? "Mất mạng — mã vẫn được giữ trên máy bạn"
    : draft.status === "pending" ? "Có thay đổi chưa lưu lên máy chủ — đang thử lại"
    : draft.status === "conflict" ? "Chưa lưu: bản trên máy chủ mới hơn"
    : draft.savedAt ? `Bản nháp đã lưu lúc ${hhmmss(draft.savedAt)}` : "Bản nháp tự lưu khi bạn ngừng gõ";

  return (
    <section className={s.right} aria-label="Soạn mã" data-part="code-wide">
      {expired && <InlineNotice tone="warning" compact>Hết giờ — phần bạn gõ sau giờ không được tính. Mã vẫn còn trong ô để bạn chép ra.</InlineNotice>}
      {draft.conflict && (
        <InlineNotice
          tone="warning"
          compact
          title={`Bản trên máy chủ mới hơn${draft.conflict.updatedAt ? ` (lưu lúc ${hhmmss(Date.parse(draft.conflict.updatedAt))})` : ""}.`}
          action={
            <>
              <Button size="sm" onClick={draft.keepMine}>Dùng bản trên máy này</Button>
              <Button size="sm" onClick={() => void draft.takeServer()}>Dùng bản đã lưu</Button>
            </>
          }
        >
          Chữ bạn đang gõ vẫn được giữ nguyên. Chọn bản muốn dùng.
        </InlineNotice>
      )}
      <div className={s.bar}>
        {code.languages.length > 1 ? (
          <>
            <label htmlFor={`lang-${item.item_id}`} className={s.fieldLabel}>Ngôn ngữ</label>
            <Select id={`lang-${item.item_id}`} value={lang} onChange={(e) => setLang(e.target.value as Lang)} disabled={locked}>
              {code.languages.map((l) => <option key={l} value={l}>{LANG_LABEL[l]}</option>)}
            </Select>
          </>
        ) : (
          <span className={s.fieldLabel}>{LANG_LABEL[code.languages[0]]}</span>
        )}
        <span className={s.grow} />
        <span className={s.status} role="status" aria-live="off" data-part="draft-status">{status}</span>
      </div>
      <CodeEditor value={source} onChange={(v) => draft.setSource(lang, v)} readOnly={locked} label={`Mã nguồn bài ${number}`} />
      <div className={s.actions}>
        {code.samples.length > 0 && <Button onClick={() => void doRun()} loading={runBusy} disabled={locked || source.trim() === ""}>Chạy thử</Button>}
        <Button variant="primary" onClick={() => void doSubmit()} loading={subBusy} disabled={locked || source.trim() === ""}>Nộp lời giải</Button>
      </div>
      {runErr != null && tabSel === "run" && <ApiErrorNotice error={runErr} />}
      {subErr != null && <ApiErrorNotice error={subErr} />}
      <Tabs
        label="Kết quả"
        value={tabSel}
        onChange={setTabSel}
        options={[
          { value: "run", label: "Kết quả chạy thử" },
          { value: "subs", label: "Lần nộp", count: subs.length || undefined },
        ]}
      />
      {tabSel === "run" ? (
        <RunResult run={run} />
      ) : (
        <SubmissionList
          items={subs}
          hasMore={next !== null}
          loadingMore={moreBusy}
          onMore={async () => {
            if (!next) return;
            setMoreBusy(true);
            try {
              const r = await listSubmissions(courseId, examId, attemptId, item.item_id, next);
              setSubs((p) => [...p, ...r.items.filter((x) => !p.some((y) => y.id === x.id))]);
              setNext(r.next_cursor);
            } finally {
              setMoreBusy(false);
            }
          }}
          onView={(id) => getSubmission(courseId, examId, attemptId, id).catch(() => null)}
          onReuse={(x) => {
            const l = (x.language as Lang) in draft.slots ? (x.language as Lang) : lang;
            setLang(l);
            draft.replace(l, x.source ?? "");
            setTabSel("run");
          }}
        />
      )}
    </section>
  );
}
