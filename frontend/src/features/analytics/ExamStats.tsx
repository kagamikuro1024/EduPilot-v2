"use client";

import Link from "next/link";
import { useState } from "react";
import { useCursorList } from "@/shared/data";
import { useLatest } from "@/features/insights/insightsApi";
import { Button, type Column, DataTable, EmptyState, Field, Input, PageState, Panel, Section, SegmentedControl, Select, Skeleton, TrendChart } from "@/shared/ui";
import { type CompareRow, type Declining, type HardItem, num, type Overview, path, pct, useCompare, useOverview, useTrend } from "./examStatsApi";
import s from "./analytics.module.css";

type Mode = "exam" | "week";
const fold = (t: string) => t.normalize("NFD").replace(/\p{M}/gu, "").replace(/đ/g, "d").toLowerCase().trim();

/** Tab «Điểm thi»: thống kê bài thi PE đã công bố của MỘT lớp (giảng viên / TA). Số liệu tính lại từ điểm chính thức mỗi lần mở, không cache. */
export function ExamStats({ courseId, teacher }: { courseId: string; teacher: boolean }) {
  const trend = useTrend(courseId);
  const [mode, setMode] = useState<Mode>("exam");
  const [examId, setExamId] = useState("");
  const [week, setWeek] = useState("");
  const scope = mode === "exam" ? { examId: examId || undefined } : { week: week || undefined };
  const overview = useOverview(courseId, mode === "week" && !week ? { week: undefined } : scope);

  const exams = trend.data ?? [];
  return (
    <PageState query={trend} isEmpty={(d) => (d as unknown[]).length === 0} loading={<Panel><Skeleton lines={6} /></Panel>} empty={<Panel><EmptyState title="Chưa có bài thi công bố" /></Panel>} error={{ problem: "Không tải được thống kê điểm thi." }} showTechnical>
      <div className={s.blocks}>
        <div className={s.pick}>
          <SegmentedControl label="Phạm vi" value={mode} onChange={setMode} options={[{ value: "exam", label: "Theo bài" }, { value: "week", label: "Theo tuần" }]} />
          {mode === "exam" ? (
            <Field label="Bài thi">
              {(id) => (
                <Select id={id} value={examId} onChange={(e) => setExamId(e.target.value)}>
                  <option value="">Mới nhất</option>
                  {[...exams].reverse().map((e) => <option key={e.exam_id} value={e.exam_id}>{e.title}</option>)}
                </Select>
              )}
            </Field>
          ) : (
            <Field label="Tuần">{(id) => <Input id={id} type="week" value={week} onChange={(e) => setWeek(e.target.value)} />}</Field>
          )}
        </div>

        <Section title="Điểm thi">
          <PageState query={overview} isEmpty={(d) => (d as Overview).exams.length === 0} loading={<Skeleton lines={5} />} empty={<EmptyState title="Chưa có bài thi" />} error={{ problem: "Không tải được điểm thi." }} showTechnical>
            {overview.data && <OverviewBody o={overview.data} />}
          </PageState>
        </Section>

        {exams.length > 1 && (
          <Section title="Xu hướng qua các bài thi">
            <TrendChart
              points={exams.map((e) => ({ x: e.title, full: `${e.title} · ${e.submissions} bài nộp`, y: Number(e.mean ?? 0) }))}
              label="Điểm trung bình theo bài thi"
              format={(v) => num(String(v))}
            />
          </Section>
        )}

        <HardItems courseId={courseId} examId={mode === "exam" ? examId : ""} />
        <DecliningList courseId={courseId} />
        {teacher && <CompareTable />}
      </div>
    </PageState>
  );
}

function OverviewBody({ o }: { o: Overview }) {
  const top = Math.max(1, ...(o.distribution ?? []).map((b) => b.count));
  return (
    <>
      <p className={s.lead}>
        {`Trung bình ${num(o.mean)} · trung vị ${num(o.median)} · đạt ${pct(o.pass_rate)} · ${o.submissions} bài nộp`}
      </p>
      {o.distribution && (
        <figure className={s.hist} aria-label="Phân bố điểm">
          <div className={s.histBars}>
            {o.distribution.map((b) => (
              <div key={b.from} className={s.histCol} title={`${num(b.from)}–${num(b.to)}: ${b.count} bài`}>
                <span className={s.histValue}>{b.count}</span>
                <span className={s.histBar} style={{ height: `${(b.count / top) * 100}%` }} />
                <span className={s.histLabel}>{num(b.from)}</span>
              </div>
            ))}
          </div>
        </figure>
      )}
    </>
  );
}

function HardItems({ courseId, examId }: { courseId: string; examId: string }) {
  const list = useCursorList<HardItem & { id: string }>(["analytics", "hard", courseId, examId], path.hard(courseId), { limit: 10, staleTime: 0, idOf: (r) => r.item_id, query: { exam_id: examId || undefined } });
  const report = useLatest(courseId);
  const topics = report.data?.report.payload.topics ?? [];
  const linked = (t: string) => {
    const k = fold(t);
    return k !== "" && topics.some((x) => fold(x.name).includes(k) || k.includes(fold(x.name)));
  };
  const cols: Column<HardItem & { id: string }>[] = [
    { key: "title", header: "Câu", primary: true, frozen: true, render: (r) => r.title },
    { key: "type", header: "Loại", render: (r) => (r.type === "CODE" ? "Bài code" : "Trắc nghiệm") },
    { key: "topic", header: "Chủ đề", render: (r) => (r.topic ? <>{r.topic}{linked(r.topic) && <> · <Link href="/insights" className="ep-link">Xem ở Báo cáo lỗ hổng</Link></>}</> : "—") },
    { key: "wrong", header: "Sai", align: "end", render: (r) => <span className="ep-num">{pct(r.wrong_rate)}</span> },
  ];
  return (
    <Section title="Câu sai nhiều nhất">
      <PageState query={{ isPending: list.isPending, isError: list.isError, error: list.error, data: list.items, refetch: list.refetch }} isEmpty={() => list.items.length === 0} empty={<EmptyState title="Chưa có câu nào" />} showTechnical>
        <DataTable caption="Câu sai nhiều nhất" columns={cols} rows={list.items} rowKey={(r) => r.item_id} dense mobile="scroll" />
        {list.hasNextPage && <Button onClick={() => void list.fetchNextPage()} loading={list.isFetchingNextPage}>Tải thêm</Button>}
      </PageState>
    </Section>
  );
}

function DecliningList({ courseId }: { courseId: string }) {
  const list = useCursorList<Declining & { id: string }>(["analytics", "declining", courseId], path.declining(courseId), { limit: 20, staleTime: 0, idOf: (r) => r.student_id });
  const cols: Column<Declining & { id: string }>[] = [
    { key: "name", header: "Sinh viên", primary: true, frozen: true, render: (r) => <Link href={`/students/${r.student_id}`} className="ep-link">{r.name}</Link> },
    { key: "code", header: "Mã số", render: (r) => r.student_code ?? "—" },
    { key: "scores", header: "3 bài gần nhất", render: (r) => <span className="ep-num">{r.scores.map((x) => num(x.score)).join(" → ")}</span> },
  ];
  return (
    <Section title="Điểm thi đang giảm">
      <PageState query={{ isPending: list.isPending, isError: list.isError, error: list.error, data: list.items, refetch: list.refetch }} isEmpty={() => list.items.length === 0} empty={<EmptyState title="Không có sinh viên nào" />} showTechnical>
        <DataTable caption="Sinh viên có điểm giảm ở 3 bài thi liên tiếp" columns={cols} rows={list.items} rowKey={(r) => r.student_id} dense mobile="scroll" />
        {list.hasNextPage && <Button onClick={() => void list.fetchNextPage()} loading={list.isFetchingNextPage}>Tải thêm</Button>}
      </PageState>
    </Section>
  );
}

function CompareTable() {
  const q = useCompare(true);
  if (!q.data || q.data.length < 2) return null; // chỉ giảng viên phụ trách ≥ 2 lớp
  const cols: Column<CompareRow>[] = [
    { key: "cls", header: "Lớp", primary: true, frozen: true, render: (r) => `${r.class_code} · ${r.name}` },
    { key: "exams", header: "Bài thi", align: "end", render: (r) => <span className="ep-num">{r.exams}</span> },
    { key: "mean", header: "Trung bình", align: "end", render: (r) => (r.enough ? num(r.mean) : "Chưa đủ") },
    { key: "median", header: "Trung vị", align: "end", render: (r) => (r.enough ? num(r.median) : "Chưa đủ") },
    { key: "pass", header: "Đạt", align: "end", render: (r) => (r.enough ? pct(r.pass_rate) : "Chưa đủ") },
    { key: "n", header: "Bài nộp", align: "end", render: (r) => <span className="ep-num">{r.submissions}</span> },
  ];
  return (
    <Section title="So sánh các lớp">
      <DataTable caption="So sánh điểm thi giữa các lớp bạn dạy" columns={cols} rows={q.data} rowKey={(r) => r.course_id} dense mobile="scroll" />
    </Section>
  );
}
