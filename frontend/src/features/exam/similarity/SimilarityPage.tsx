"use client";

import { useQueryClient } from "@tanstack/react-query";
import { useSearchParams } from "next/navigation";
import { useEffect, useState } from "react";
import { ApiErrorNotice, apiClient, useCursorList, useJob } from "@/shared/data";
import { useClassCourse } from "@/features/members/classApi";
import { useSession } from "@/shared/session/session";
import { Button, type Column, DataTable, EmptyState, Field, InlineNotice, Page, PageHeader, Panel, PanelSection, SegmentedControl, Skeleton, StatusText, Textarea } from "@/shared/ui";
import { fmtWhen } from "../examApi";
import { percent, simPath, STATE_LABEL, type Pair, type PairDetail } from "./similarityApi";
import s from "./Similarity.module.css";

/** Câu cố định (US-PE-07 AC9): độ giống là gợi ý, quyết định là của Giảng viên. */
const FIXED_NOTE = "Độ giống chỉ là gợi ý — nhiều bài đúng cùng một cách làm tự nhiên giống nhau. Quyết định là của thầy/cô.";

/** `/exams/[id]/similarity` (chỉ Giảng viên): cặp bài code nghi giống nhau, hai mã cạnh nhau có tô phần khớp, đánh dấu Đã xem / Cần trao đổi. Không có hành động trừ điểm. */
export function SimilarityPage({ id }: { id: string }) {
  const { role } = useSession();
  const cc = useClassCourse();
  const hint = useSearchParams().get("course");
  const course = hint ?? (cc.state === "ready" ? cc.course.id : null);
  const back = { href: "/exams", label: "Bài thi" };
  if (role !== "teacher")
    return (
      <Page width="wide">
        <PageHeader title="Nghi giống nhau" back={back} />
        <Panel><EmptyState title="Chỉ dành cho giảng viên">Trang này dành cho giảng viên của lớp.</EmptyState></Panel>
      </Page>
    );
  if (!course)
    return (
      <Page width="wide">
        <PageHeader title="Nghi giống nhau" back={back} />
        {cc.state === "loading" ? <Panel><Skeleton lines={6} /></Panel> : <Panel><EmptyState title="Chưa chọn lớp">Chọn một lớp ở thanh trên để xem các cặp bài.</EmptyState></Panel>}
      </Page>
    );
  return <Board course={course} exam={id} />;
}

function Board({ course, exam }: { course: string; exam: string }) {
  const qc = useQueryClient();
  const [flagged, setFlagged] = useState<"flagged" | "all">("flagged");
  const [open, setOpen] = useState<string | null>(null);
  const key = ["exam-similarity", course, exam, flagged];
  const list = useCursorList<Pair>(key, simPath(course, exam), { query: { flagged: flagged === "flagged" ? true : undefined } });
  const [jobId, setJobId] = useState<string | null>(null);
  const [runErr, setRunErr] = useState<unknown>(null);
  const job = useJob(jobId);
  const running = jobId !== null && job.status !== "SUCCEEDED" && job.status !== "FAILED";
  useEffect(() => {
    if (job.status === "SUCCEEDED") {
      void qc.invalidateQueries({ queryKey: ["exam-similarity", course, exam] });
    }
  }, [job.status, qc, course, exam]);

  async function rerun() {
    setRunErr(null);
    try {
      const r = await apiClient.post<{ job_id: string }>(`${simPath(course, exam)}/run`);
      setJobId(r.data.job_id);
    } catch (e) {
      setRunErr(e);
    }
  }

  const cols: Column<Pair>[] = [
    { key: "who", header: "Hai sinh viên", primary: true, render: (p) => <span>{p.a.name} · {p.b.name}</span> },
    { key: "problem", header: "Bài", render: (p) => p.problem_title },
    { key: "score", header: "Độ giống", align: "end", render: (p) => percent(p.score) },
    { key: "flag", header: "Gợi ý", render: (p) => (p.flagged ? <StatusText tone="amber">Nên xem</StatusText> : <StatusText tone="neutral">Thấp hơn mức nền</StatusText>) },
    { key: "state", header: "Trạng thái", render: (p) => <StatusText tone={p.review_state === "NEW" ? "blue" : "green"}>{STATE_LABEL[p.review_state]}</StatusText> },
  ];

  return (
    <Page width="wide">
      <PageHeader title="Nghi giống nhau" back={{ href: "/exams", label: "Bài thi" }} />
      <Panel>
      <p className={s.note} data-part="similarity-note">{FIXED_NOTE}</p>
      <div className={s.bar}>
        <SegmentedControl label="Lọc cặp" value={flagged} onChange={setFlagged} options={[{ value: "flagged", label: "Nên xem" }, { value: "all", label: "Tất cả" }]} />
        <span className={s.grow} />
        <Button onClick={() => void rerun()} loading={running}>{running ? "Đang so sánh…" : "Chạy lại so sánh"}</Button>
      </div>
      {runErr != null && <ApiErrorNotice error={runErr} />}
      {job.status === "FAILED" && <InlineNotice tone="danger" compact>Chưa so sánh được: {job.error ?? "thử lại sau."}</InlineNotice>}
      {list.isPending ? (
        <Skeleton lines={6} />
      ) : list.isError ? (
        <ApiErrorNotice error={list.error} onRetry={() => void list.refetch()} />
      ) : (
        <>
          <DataTable
            caption="Các cặp bài code nghi giống nhau"
            columns={cols}
            rows={list.items}
            rowKey={(p) => p.id}
            onRowClick={(p) => setOpen(p.id)}
            activeKey={open ?? undefined}
            empty={<EmptyState title="Chưa có cặp nào">Chưa có cặp bài nào cần xem. Bài thi cần đã đóng và có câu lập trình; bấm Chạy lại so sánh nếu vừa đóng.</EmptyState>}
          />
          {list.hasNextPage && <div><Button onClick={() => void list.fetchNextPage()} loading={list.isFetchingNextPage}>Xem thêm</Button></div>}
        </>
      )}
      {open && <PairView key={open} course={course} exam={exam} id={open} onReviewed={() => void qc.invalidateQueries({ queryKey: ["exam-similarity", course, exam] })} />}
      </Panel>
    </Page>
  );
}

function Code({ source, hits }: { source: string; hits: number[] }) {
  const set = new Set(hits);
  return (
    <pre className={s.code}>
      {source.split("\n").map((ln, i) => (
        <div key={i} className={[s.line, set.has(i + 1) ? s.hit : ""].join(" ")} data-hit={set.has(i + 1) ? "1" : undefined}>
          <span className={s.num} aria-hidden>{i + 1}</span>
          <span>{ln}</span>
        </div>
      ))}
    </pre>
  );
}

function PairView({ course, exam, id, onReviewed }: { course: string; exam: string; id: string; onReviewed: () => void }) {
  const qc = useQueryClient();
  const [d, setD] = useState<PairDetail | null>(null);
  const [err, setErr] = useState<unknown>(null);
  const [note, setNote] = useState("");
  const [busy, setBusy] = useState(false);
  useEffect(() => {
    let alive = true;
    apiClient.get<PairDetail>(`${simPath(course, exam)}/${id}`).then(
      (r) => {
        if (!alive) return;
        setD(r.data);
        setNote(r.data.pair.note ?? "");
      },
      (e) => alive && setErr(e),
    );
    return () => {
      alive = false;
    };
  }, [course, exam, id]);

  async function review(state: "CLEARED" | "FOLLOW_UP") {
    setBusy(true);
    setErr(null);
    try {
      const r = await apiClient.put<Pair>(`${simPath(course, exam)}/${id}/review`, { state, note: note.trim() === "" ? null : note.trim() });
      setD((p) => (p ? { ...p, pair: r.data } : p));
      onReviewed();
      void qc.invalidateQueries({ queryKey: ["today"] });
    } catch (e) {
      setErr(e);
    } finally {
      setBusy(false);
    }
  }

  if (err && !d) return <ApiErrorNotice error={err} />;
  if (!d) return <Skeleton lines={8} />;
  const p = d.pair;
  return (
    <section className={s.head} aria-label="Hai mã cạnh nhau" data-part="pair-view">
      <p className={s.title}>{p.problem_title} · {percent(p.score)} · {p.shared_fingerprints} đoạn trùng</p>
      <p className={s.meta}>{STATE_LABEL[p.review_state]}{p.reviewed_at ? ` · ${fmtWhen(p.reviewed_at)}` : ""}</p>
      <div className={s.side}>
        <div><PanelSection tone="strong"><p className={s.title}>{p.a.name}</p><Code source={d.a.source} hits={d.a.match_lines} /></PanelSection></div>
        <div><PanelSection tone="strong"><p className={s.title}>{p.b.name}</p><Code source={d.b.source} hits={d.b.match_lines} /></PanelSection></div>
      </div>
      <div className={s.actions}>
        <Field label="Ghi chú (tối đa 500 ký tự)">{(id, by) => <Textarea id={id} aria-describedby={by} rows={2} maxLength={500} value={note} onChange={(e) => setNote(e.target.value)} />}</Field>
        {err != null && <ApiErrorNotice error={err} />}
        <div className={s.bar}>
          <Button onClick={() => void review("CLEARED")} loading={busy}>Đã xem — không có vấn đề</Button>
          <Button onClick={() => void review("FOLLOW_UP")} loading={busy}>Cần trao đổi</Button>
        </div>
      </div>
    </section>
  );
}
