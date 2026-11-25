"use client";

import { useEffect, useState } from "react";
import { CalendarPlus, Download, Pin, Sparkles } from "lucide-react";
import { ApiError, useJob } from "@/shared/data";
import { ApiErrorNotice } from "@/shared/data/ApiErrorNotice";
import { Button, EmptyState, type HeaderAction, InlineNotice, Page, PageHeader, PageState, Panel, Section, SegmentedControl, Skeleton, StatusText } from "@/shared/ui";
import { downloadMarkdown, type Latest, type Period, type Topic, useGenerate, useLatest, useRefreshLatest } from "./insightsApi";
import { PinThreadForm, ReviewForm } from "./TopicForms";
import s from "./insights.module.css";

const TOP = 8; // SRS 4.1: hiển thị tối đa 8 chủ đề đầu, còn lại gập «Chủ đề khác»

const dmy = (iso: string) => new Date(iso).toLocaleDateString("vi-VN", { timeZone: "Asia/Ho_Chi_Minh" });
const hm = (iso: string) => new Date(iso).toLocaleTimeString("vi-VN", { hour: "2-digit", minute: "2-digit", timeZone: "Asia/Ho_Chi_Minh" });
const stageOf = (pct: number) => (pct < 25 ? "Đang gom câu hỏi của lớp…" : pct < 70 ? "Đang nhóm câu hỏi theo chủ đề…" : "Đang đặt tên và soạn gợi ý…");

export function KnowledgeGap({ courseId }: { courseId: string }) {
  const latest = useLatest(courseId);
  const gen = useGenerate(courseId);
  const refresh = useRefreshLatest(courseId);
  const [jobId, setJobId] = useState<string | null>(null);
  const [reusedAt, setReusedAt] = useState<string | null>(null);
  const [period, setPeriod] = useState<Period>(14);
  const [open, setOpen] = useState<{ kind: "pin" | "review"; rank: number } | null>(null);
  const job = useJob(jobId);

  const running = gen.isPending || (jobId !== null && job.status !== "SUCCEEDED" && job.status !== "FAILED");
  const failed = jobId !== null && job.status === "FAILED";
  useEffect(() => {
    if (job.status === "SUCCEEDED") void refresh();
  }, [job.status]); // eslint-disable-line react-hooks/exhaustive-deps

  async function start() {
    setReusedAt(null);
    setJobId(null);
    try {
      const r = await gen.mutateAsync(period);
      if (r.reused) setReusedAt(latest.data ? dmy(latest.data.report.data_watermark) : dmy(new Date().toISOString()));
      else setJobId(r.job_id ?? null);
    } catch {
      /* gen.error hiện tại chỗ */
    }
  }

  const data = latest.data ?? null;
  const create = { label: "Tạo báo cáo mới", icon: <Sparkles aria-hidden />, busy: running, onClick: () => void start() };
  const more: HeaderAction[] | undefined = data
    ? [
        { label: "Xuất Markdown", icon: <Download aria-hidden />, onClick: () => void downloadMarkdown(courseId, data.report.id) },
        { label: "Xuất PDF", onClick: () => window.print() },
      ]
    : undefined;

  return (
    <Page kind="read">
      <PageHeader
        title="Lỗ hổng kiến thức"
        primary={data?.has_new_data ? create : undefined}
        secondary={data && !data.has_new_data ? [create] : undefined}
        more={more}
        meta={data ? [`Báo cáo ${dmy(data.report.created_at)}, ${hm(data.report.created_at)}`, `${data.report.payload.counts.questions} câu hỏi · ${data.report.payload.counts.students} sinh viên`] : undefined}
      />

      <div className={s.period}>
      <SegmentedControl
        label="Kỳ báo cáo"
        value={String(period) as `${Period}`}
        onChange={(v) => setPeriod(Number(v) as Period)}
        options={[{ value: "7", label: "7 ngày" }, { value: "14", label: "14 ngày" }, { value: "30", label: "30 ngày" }, { value: "120", label: "Toàn kỳ", title: "120 ngày gần nhất" }]}
      />
      </div>

      {running && (
        <Section title="Đang tạo báo cáo">
          <p className="ep-meta" role="status" aria-live="polite">{stageOf(job.progress)}</p>
        </Section>
      )}
      {failed && (
        <InlineNotice tone="danger" title="Chưa tạo được báo cáo." action={<Button size="sm" onClick={() => void start()}>Thử lại</Button>} />
      )}
      {gen.error instanceof ApiError && (gen.error.code === "NOT_ENOUGH_DATA" ? <InlineNotice tone="danger" title="Chưa đủ câu hỏi để gom (cần ≥ 20)." /> : <ApiErrorNotice error={gen.error} />)}
      {reusedAt && <p className="ep-meta" role="status">Chưa có câu hỏi mới từ ngày {reusedAt}</p>}

      <PageState
        query={latest}
        isEmpty={(d) => d === null}
        loading={<Panel><div className={s.loading}><Skeleton lines={2} /><Skeleton lines={5} /><Skeleton lines={5} /></div></Panel>}
        empty={running ? null : <Panel><EmptyState title="Chưa có báo cáo" action={<Button icon={<Sparkles aria-hidden />} onClick={() => void start()} disabled={running}>Tạo báo cáo mới</Button>} /></Panel>}
        error={{ problem: "Không tải được báo cáo lỗ hổng kiến thức." }}
        showTechnical
      >
        {data && <Body data={data} courseId={courseId} open={open} setOpen={setOpen} />}
      </PageState>

    </Page>
  );
}

type Open = { kind: "pin" | "review"; rank: number } | null;
function Body({ data, courseId, open, setOpen }: { data: Latest; courseId: string; open: Open; setOpen: (o: Open) => void }) {
  const { topics, uncovered } = data.report.payload;
  const top = topics.slice(0, TOP);
  const rest = topics.slice(TOP);
  return (
    <div className={s.blocks}>
      <Section title="Chủ đề nên dạy lại">
        <ol className={s.topics}>
          {top.map((t) => <TopicItem key={t.rank} t={t} courseId={courseId} open={open?.rank === t.rank ? open.kind : null} setOpen={setOpen} />)}
        </ol>
        {rest.length > 0 && (
          <details className={s.samples}>
            <summary>Chủ đề khác ({rest.length})</summary>
            <ol className={s.topics}>
              {rest.map((t) => <TopicItem key={t.rank} t={t} courseId={courseId} open={open?.rank === t.rank ? open.kind : null} setOpen={setOpen} />)}
            </ol>
          </details>
        )}
      </Section>

      {uncovered.length > 0 && (
        <Section title="Tài liệu chưa đề cập">
          <ul className={s.gaps}>
            {uncovered.map((g) => (
              <li key={g.text} className={s.gap}>
                <p className="ep-item-title">{g.text}</p>
                <span className="ep-meta">{g.count} lần</span>
              </li>
            ))}
          </ul>
        </Section>
      )}

      {data.previous && (
        <Section title={`So với báo cáo ngày ${dmy(data.previous.created_at)}`}>
          <ul className={s.compare}>
            {data.previous.new_topics.map((n) => <li key={`n-${n}`}>Chủ đề mới: {n}</li>)}
            {(data.previous.moved ?? []).map((m) => <li key={`m-${m.name}`}>{m.to < m.from ? "Tăng hạng" : "Giảm hạng"}: {m.name} (hạng {m.from} → {m.to})</li>)}
            {data.previous.new_topics.length === 0 && (data.previous.moved ?? []).length === 0 && <li>Không đổi</li>}
          </ul>
        </Section>
      )}
    </div>
  );
}

function TopicItem({ t, courseId, open, setOpen }: { t: Topic; courseId: string; open: "pin" | "review" | null; setOpen: (o: Open) => void }) {
  const where = [t.chapter, t.week !== null ? `Tuần ${t.week}` : null].filter(Boolean).join(" · ");
  return (
    <li className={s.topic} id={`topic-${t.rank}`}>
      <div className={s.topicHead}>
        <span className={s.rank} aria-hidden>{t.rank}</span>
        <div className={s.topicTitle}>
          <h3 className="ep-item-title">{t.name}</h3>
          {(where || t.few_asked || t.named_by === "fallback") && (
            <p className="ep-meta">
              {where}
              {t.few_asked && <>{where && " · "}<StatusText tone="neutral">Ít người hỏi</StatusText></>}
              {t.named_by === "fallback" && <>{(where || t.few_asked) && " · "}Chưa đặt được tên chủ đề</>}
            </p>
          )}
        </div>
      </div>

      <p className={s.why}>{t.why}</p>

      {t.examples.length > 0 && (
        <details className={s.samples}>
          <summary>Câu hỏi mẫu ({t.examples.length})</summary>
          <ul>{t.examples.map((q) => <li key={q}>“{q}”</li>)}</ul>
        </details>
      )}

      {t.recommendation && <p className={s.action}><span className={s.actionLabel}>Nên làm</span> {t.recommendation}</p>}

      <div className={s.topicActions}>
        <Button size="sm" icon={<Pin aria-hidden />} onClick={() => setOpen({ kind: "pin", rank: t.rank })} aria-expanded={open === "pin"}>Tạo thread ghim</Button>
        <Button size="sm" icon={<CalendarPlus aria-hidden />} onClick={() => setOpen({ kind: "review", rank: t.rank })} aria-expanded={open === "review"}>Tạo buổi ôn tập</Button>
      </div>
      {open === "pin" && <PinThreadForm courseId={courseId} topic={t} onClose={() => setOpen(null)} />}
      {open === "review" && <ReviewForm courseId={courseId} topic={t} onClose={() => setOpen(null)} />}
    </li>
  );
}
