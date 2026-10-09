"use client";

import { useQuery } from "@tanstack/react-query";
import { ApiErrorNotice, apiClient } from "@/shared/data";
import { EmptyState, Skeleton } from "@/shared/ui";
import { BarList } from "@/shared/ui";
import { pct, resPath, vnum, type Stats } from "./resultsApi";
import s from "./Results.module.css";

/** Tab "Thống kê": phân bố điểm, trung bình / trung vị, câu trắc nghiệm sai nhiều và bài code. Tính từ các lượt đã chấm; không có tên sinh viên. */
export function StatsPanel({ course, exam }: { course: string; exam: string }) {
  const q = useQuery({ queryKey: ["exam-stats", course, exam], queryFn: async ({ signal }) => (await apiClient.get<Stats>(`${resPath(course, exam)}/stats`, { signal })).data });
  if (q.isPending) return <Skeleton lines={6} />;
  if (q.isError) return <ApiErrorNotice error={q.error} onRetry={() => void q.refetch()} />;
  const d = q.data;
  const total = d.distribution.reduce((n, b) => n + b.count, 0);
  if (total === 0) return <EmptyState title="Chưa có điểm">Thống kê hiện khi có lượt đã chấm.</EmptyState>;
  return (
    <div className={s.statGrid} data-part="stats">
      <section className={s.sub} aria-label="Phân bố điểm">
        <p className={s.subHead}>Phân bố điểm ({total} bài)</p>
        <BarList items={d.distribution.map((b) => ({ label: `${vnum(b.from)}–${vnum(b.to)}`, value: b.count }))} />
        <p className={s.muted}>Trung bình <span className={s.big}>{vnum(d.mean)}</span> · Trung vị <span className={s.big}>{vnum(d.median)}</span></p>
      </section>
      <section className={s.sub} aria-label="Câu sai nhiều">
        <p className={s.subHead}>Câu trắc nghiệm đúng ít nhất</p>
        {d.hardest.length === 0 ? <p className={s.muted}>Không có câu trắc nghiệm.</p> : <BarList max={1} format={(v) => pct(String(v))} items={d.hardest.map((h) => ({ label: h.title, value: Number(h.correct_rate), tone: "red" }))} />}
        {d.code.length > 0 && (
          <>
            <p className={s.subHead}>Bài lập trình</p>
            <ul className={s.opts}>
              {d.code.map((c) => <li key={c.item_id} className={s.opt}><span>{c.title}</span><span className={s.tag}>trung bình {pct(c.mean_ratio)} · lỗi biên dịch {pct(c.ce_rate)}</span></li>)}
            </ul>
          </>
        )}
      </section>
    </div>
  );
}
