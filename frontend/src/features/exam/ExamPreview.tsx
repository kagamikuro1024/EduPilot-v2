"use client";

import { RefreshCw } from "lucide-react";
import { useState } from "react";
import { useQuery } from "@tanstack/react-query";
import { ApiErrorNotice, apiClient } from "@/shared/data";
import { Markdown } from "@/shared/domain";
import { Button, InlineNotice, Skeleton } from "@/shared/ui";
import { fmtClock, ePath, eKey, type Preview, type PreviewItem } from "./examApi";
import s from "./Exam.module.css";

/**
 * Tab `Xem trước`: đúng DTO và hàm dựng của lượt làm sinh viên, hạt giống ngẫu nhiên mỗi lần xem (US-PE-04 AC3). Không tạo lượt làm, không lưu gì;
 * không có đáp án đúng. `Xáo lại` đổi hạt giống để thấy thứ tự khác.
 */
export function ExamPreview({ courseId, examId, version }: { courseId: string; examId: string; version: number }) {
  const [seed, setSeed] = useState(0);
  const q = useQuery({
    queryKey: eKey(courseId, examId, "preview", version, seed),
    queryFn: async ({ signal }) => (await apiClient.get<Preview>(`${ePath(courseId)}/${examId}/preview`, { signal })).data,
    staleTime: Infinity,
    gcTime: 0,
  });
  if (q.isPending) return <Skeleton lines={6} />;
  if (q.isError) return <ApiErrorNotice error={q.error} onRetry={() => void q.refetch()} showTechnical />;
  const { exam, items } = q.data;
  return (
    <div className={s.preview}>
      <InlineNotice compact>Chỉ xem: bài như sinh viên sẽ thấy. Không có lượt làm nào được tạo và đáp án không hiện.</InlineNotice>
      <div className={s.previewHead}>
        <h2 className={s.previewTitle}>{exam.title}</h2>
        <p className={s.meta}>{exam.duration_minutes ? `${exam.duration_minutes} phút` : "Chưa đặt thời lượng"}{exam.closes_at ? ` · đóng lúc ${fmtClock(exam.closes_at)}` : ""}</p>
        {exam.instructions && <Markdown source={exam.instructions} />}
        <Button size="sm" icon={<RefreshCw aria-hidden />} onClick={() => setSeed((n) => n + 1)} loading={q.isFetching}>Xáo lại</Button>
      </div>
      {items.length === 0 ? (
        <InlineNotice>Bài chưa có câu hỏi nào.</InlineNotice>
      ) : (
        <ol className={s.pvItems} aria-label="Câu hỏi như sinh viên thấy">
          {items.map((it) => <PreviewCard key={it.item_id} it={it} />)}
        </ol>
      )}
    </div>
  );
}

function PreviewCard({ it }: { it: PreviewItem }) {
  return (
    <li className={s.pvItem}>
      <p className={s.pvMeta}>Câu {it.position} · {it.points.replace(".", ",")} điểm{it.type === "MCQ_MULTI" ? " · chọn tất cả đáp án đúng" : ""}</p>
      <Markdown source={it.stem} />
      {it.type === "TRUE_FALSE" && <p className={s.meta}>Đúng / Sai</p>}
      {it.options.length > 0 && (
        <ol className={s.pvOptions} type="A">
          {it.options.map((o) => <li key={o.id} className={s.pvOption}>{o.body}</li>)}
        </ol>
      )}
      {it.code && (
        <div className={s.pvCode}>
          <p className={s.meta}>C hoặc C++ · {it.code.time_limit_ms} ms · {it.code.memory_limit_mb} MiB</p>
          {Object.entries(it.code.starter_code).map(([lang, src]) => (
            <pre key={lang} className={s.pre} aria-label={`Mã khởi tạo ${lang}`}>{src}</pre>
          ))}
          {it.code.samples.map((sm) => (
            <dl key={sm.name} className={s.sample}>
              <dt>Ví dụ {sm.name} — vào</dt>
              <dd><pre className={s.pre}>{sm.input}</pre></dd>
              <dt>Ra</dt>
              <dd><pre className={s.pre}>{sm.expected}</pre></dd>
            </dl>
          ))}
        </div>
      )}
    </li>
  );
}
