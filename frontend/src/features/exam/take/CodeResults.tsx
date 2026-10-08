"use client";

import { Check, Clock, X } from "lucide-react";
import { useState } from "react";
import { Button, StatusText, type StatusTone } from "@/shared/ui";
import type { RunView, SampleResult, SubmissionView, Verdict } from "./codeApi";
import s from "./CodeResults.module.css";

/** Nhãn tiếng Việt của kết quả (AC7); không bao giờ hiện mã trạng thái trần. */
export const VERDICT_LABEL: Record<Verdict, string> = {
  AC: "Đúng",
  WA: "Sai kết quả",
  TLE: "Quá thời gian",
  MLE: "Quá bộ nhớ",
  RE: "Lỗi khi chạy",
  OLE: "In ra quá nhiều",
  CE: "Lỗi biên dịch",
  IE: "Hệ thống chưa chạy được",
};

const verdictTone = (v: Verdict): StatusTone => (v === "AC" ? "green" : v === "IE" ? "neutral" : "red");

/** Hiện khoảng trắng và xuống dòng bằng ký hiệu (AC7): `·` dấu cách, `→` tab, `↵` xuống dòng. */
export function visible(v: string): string {
  return v.replace(/ /g, "·").replace(/\t/g, "→").replace(/\r?\n/g, "↵\n");
}

const IE_TEXT = "Hệ thống chưa chấm được bài của bạn. Bài đã được lưu và giảng viên sẽ xử lý.";
const pending = (st: string) => st === "QUEUED" || st === "RUNNING";
const COMPILE_LOG_MAX = 8192;

function VerdictIcon({ v }: { v: Verdict }) {
  return v === "AC" ? <Check aria-hidden /> : v === "TLE" || v === "MLE" ? <Clock aria-hidden /> : <X aria-hidden />;
}

function Block({ title, text }: { title: string; text: string }) {
  return (
    <div className={s.block}>
      <p className={s.blockTitle}>{title}</p>
      <pre className={s.pre}>{visible(text)}</pre>
    </div>
  );
}

function SampleRow({ r, detail }: { r: SampleResult; detail: boolean }) {
  return (
    <li className={s.sample}>
      <div className={s.sampleHead}>
        <span className={s.sampleName}>{r.name}</span>
        <StatusText tone={verdictTone(r.verdict)}>
          <VerdictIcon v={r.verdict} /> {VERDICT_LABEL[r.verdict]}
        </StatusText>
        <span className={s.meta}>{r.time_ms} ms · {Math.round(r.memory_kb / 1024 * 10) / 10} MB</span>
      </div>
      {detail && r.verdict === "WA" && (r.input !== undefined || r.expected !== undefined) && (
        <div className={s.blocks}>
          <Block title="Đầu vào" text={r.input ?? ""} />
          <Block title="Kết quả mong đợi" text={r.expected ?? ""} />
          <Block title="Kết quả của bạn" text={r.got ?? ""} />
        </div>
      )}
    </li>
  );
}

/** Kết quả một lần `Chạy thử` (AC7): mỗi test mẫu một hàng; `CE` hiện nhật ký biên dịch; `IE` nói rõ là lỗi phía hệ thống. */
export function RunResult({ run }: { run: RunView | null }) {
  if (!run) return <p className={s.empty}>Bấm Chạy thử để thử mã của bạn với các test mẫu. Chạy thử không ảnh hưởng điểm.</p>;
  if (pending(run.status)) return <p className={s.empty} role="status">Đang chạy thử…</p>;
  if (run.status === "ERROR") return <p className={s.empty}>{VERDICT_LABEL.IE}. Mã của bạn vẫn được lưu — thử lại sau ít phút.</p>;
  if (run.compile_ok === false)
    return (
      <div className={s.result}>
        <StatusText tone="red"><X aria-hidden /> {VERDICT_LABEL.CE}</StatusText>
        <pre className={s.pre}>{(run.compile_log ?? "").slice(0, COMPILE_LOG_MAX)}</pre>
      </div>
    );
  const ok = run.samples.filter((x) => x.verdict === "AC").length;
  return (
    <div className={s.result}>
      <p className={s.summary}>Đúng {ok}/{run.samples.length} test mẫu.</p>
      <ul className={s.samples}>{run.samples.map((x) => <SampleRow key={x.name} r={x} detail />)}</ul>
    </div>
  );
}

const hhmmss = (iso: string) => new Date(Date.parse(iso) + 7 * 3600_000).toISOString().slice(11, 19);

/** Dòng trạng thái của một lần nộp (AC10). */
export function submissionStatus(x: SubmissionView): string {
  if (pending(x.status)) return "Đang chấm";
  if (x.status === "SUPERSEDED") return "Đã được thay bằng lần nộp mới hơn";
  if (x.status === "ERROR") return IE_TEXT;
  if (x.compile_ok === false) return "Lỗi biên dịch";
  return `Biên dịch được · ${x.samples.filter((t) => t.verdict === "AC").length}/${x.samples.length} test mẫu đúng`;
}

/** Lịch sử nộp của chính mình ở một câu: mới nhất trước, nhãn `Lần nộp tính điểm`, xem lại mã + `Dùng lại mã này`. */
export function SubmissionList({ items, hasMore, onMore, onReuse, onView, loadingMore }: {
  items: SubmissionView[];
  hasMore: boolean;
  onMore: () => void;
  onReuse: (x: SubmissionView) => void;
  /** nạp mã của một lần nộp (danh sách không kèm mã) */
  onView: (id: string) => Promise<SubmissionView | null>;
  loadingMore: boolean;
}) {
  const [open, setOpen] = useState<string | null>(null);
  const [full, setFull] = useState<Record<string, SubmissionView>>({});
  if (items.length === 0) return <p className={s.empty}>Bạn chưa nộp lần nào. Lần nộp cuối cùng của mỗi bài là lần được tính điểm.</p>;
  return (
    <div className={s.result}>
      <ul className={s.samples}>
        {items.map((x) => {
          const v = full[x.id];
          return (
            <li key={x.id} className={s.sample} data-part="submission">
              <div className={s.sampleHead}>
                <span className={s.sampleName}>{hhmmss(x.created_at)} · {x.language === "c11" ? "C" : "C++"}</span>
                {x.is_final && <StatusText tone="blue" chip>Lần nộp tính điểm</StatusText>}
                <span className={s.meta}>{submissionStatus(x)}</span>
                <Button
                  size="sm"
                  variant="ghost"
                  onClick={async () => {
                    if (open === x.id) return setOpen(null);
                    setOpen(x.id);
                    if (!full[x.id]) {
                      const got = await onView(x.id);
                      if (got) setFull((p) => ({ ...p, [x.id]: got }));
                    }
                  }}
                >
                  {open === x.id ? "Ẩn mã" : "Xem mã"}
                </Button>
              </div>
              {open === x.id && v && (
                <div className={s.blocks}>
                  <pre className={s.pre}>{v.source}</pre>
                  <div><Button size="sm" onClick={() => onReuse(v)}>Dùng lại mã này</Button></div>
                  {v.compile_ok === false && <pre className={s.pre}>{(v.compile_log ?? "").slice(0, COMPILE_LOG_MAX)}</pre>}
                  {v.samples.length > 0 && <ul className={s.samples}>{v.samples.map((t) => <SampleRow key={t.name} r={t} detail={false} />)}</ul>}
                </div>
              )}
            </li>
          );
        })}
      </ul>
      {hasMore && <div><Button size="sm" onClick={onMore} loading={loadingMore}>Xem thêm</Button></div>}
    </div>
  );
}
