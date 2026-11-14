"use client";

import { useState } from "react";
import { useCursorList } from "@/shared/data";
import { Button, type Column, DataTable, EmptyState, PageState, Section, SegmentedControl, Select, Skeleton, StatusText } from "@/shared/ui";
import { FailedMail } from "./FailedMail";
import { RequestDrawer } from "./RequestDrawer";
import { type DayRow, type RequestRow, type TaskCost, money, pct, secs, STATUS_LABEL, TASK_LABEL, useAdminSummary } from "./api";
import s from "./observability.module.css";

type Filter = "all" | "problem" | "privacy";
type Range = "1h" | "24h" | "7d";
const RANGE_MS: Record<Range, number> = { "1h": 3_600_000, "24h": 86_400_000, "7d": 7 * 86_400_000 };
const fromIso = (r: Range) => new Date(Date.now() - RANGE_MS[r]).toISOString();

const DAY_COLUMNS: Column<DayRow>[] = [
  { key: "day", header: "Ngày", primary: true, render: (d) => new Date(`${d.day}T00:00:00`).toLocaleDateString("vi-VN", { weekday: "short", day: "2-digit", month: "2-digit" }) },
  { key: "requests", header: "Yêu cầu", align: "end", render: (d) => <span className="ep-num">{d.requests.toLocaleString("vi-VN")}</span> },
  { key: "p95", header: "p95", align: "end", render: (d) => <span className="ep-num">{secs(d.p95_ms)}</span> },
  { key: "errors", header: "Lỗi", align: "end", render: (d) => <span className="ep-num">{d.errors}</span> },
  { key: "esc", header: "Chuyển giảng viên", align: "end", render: (d) => <span className="ep-num">{d.escalations}</span> },
  { key: "pii", header: "Che thông tin", align: "end", render: (d) => <span className="ep-num">{d.pii_masked}</span> },
  { key: "cost", header: "Chi phí", align: "end", render: (d) => <span className="ep-num">{money(d.cost)}</span> },
];
const TASK_COLUMNS: Column<TaskCost>[] = [
  { key: "task", header: "Tác vụ", primary: true, render: (t) => TASK_LABEL[t.task] ?? t.task },
  { key: "requests", header: "Yêu cầu", align: "end", render: (t) => <span className="ep-num">{t.requests.toLocaleString("vi-VN")}</span> },
  { key: "cost", header: "Chi phí", align: "end", render: (t) => <span className="ep-num">{money(t.cost)}</span> },
];

const tone = (r: RequestRow) => (r.status === "ok" ? "green" : r.status === "degraded" || r.fallback_index > 0 ? "amber" : "red");

export function AdminView() {
  const summary = useAdminSummary();
  const [filter, setFilter] = useState<Filter>("all");
  const [range, setRange] = useState<Range>("24h");
  const [from, setFrom] = useState(() => fromIso("24h"));
  const pickRange = (r: Range) => { setRange(r); setFrom(fromIso(r)); }; // mốc `from` đặt lúc đổi khoảng (không đọc đồng hồ khi render)
  const [task, setTask] = useState("");
  const [open, setOpen] = useState<RequestRow | null>(null);
  const list = useCursorList<RequestRow>(["observability", "requests", filter, range, task], "/admin/observability/requests", { limit: 50, query: { filter, task: task || undefined, from } });

  const columns: Column<RequestRow>[] = [
    { key: "at", header: "Lúc", width: "88px", render: (r) => <span className="ep-num">{new Date(r.at).toLocaleTimeString("vi-VN", { hour: "2-digit", minute: "2-digit", second: "2-digit" })}</span> },
    { key: "task", header: "Tác vụ", frozen: true, render: (r) => <><span className={s.task}>{TASK_LABEL[r.task] ?? r.task}</span><span className={s.sub}>{r.course_code ? `Lớp ${r.course_code}` : "Ngoài lớp"}</span></> },
    { key: "model", header: "Model", render: (r) => <span className={s.mono}>{r.model ?? "—"}</span> },
    { key: "latency", header: "Độ trễ", align: "end", render: (r) => <span className="ep-num">{secs(r.latency_ms)}</span> },
    { key: "pii", header: "Che thông tin", align: "end", render: (r) => <span className="ep-num">{r.pii_masked_count || "—"}</span> },
    { key: "status", header: "Kết quả", render: (r) => <StatusText tone={tone(r)}>{STATUS_LABEL[r.status] ?? r.status}</StatusText> },
  ];

  const t = summary.data?.totals;
  return (
    <>
      <FailedMail />
      <PageState query={summary} isEmpty={() => (summary.data?.totals.requests ?? 0) === 0 && (summary.data?.days.length ?? 0) === 0} loading={<Skeleton lines={8} />} empty={<EmptyState title="Chưa có yêu cầu nào." />}>
        {t && (
          <p className={s.today} data-part="obs-today">
            {`24 giờ qua: ${t.requests.toLocaleString("vi-VN")} yêu cầu · p95 ${secs(t.p95_ms)} · lỗi ${pct(t.error_rate)} · dự phòng ${pct(t.fallback_rate)} · chi phí ${money(t.cost)}`}
          </p>
        )}
        <Section panel title="7 ngày qua">
          <DataTable caption="Số liệu theo ngày" columns={DAY_COLUMNS} rows={summary.data?.days ?? []} rowKey={(d) => d.day} dense />
        </Section>
        <Section panel title="Chi phí theo tác vụ">
          <DataTable caption="Chi phí 7 ngày theo tác vụ" columns={TASK_COLUMNS} rows={summary.data?.tasks ?? []} rowKey={(x) => x.task} dense />
        </Section>
        <Section
          panel
          title="Yêu cầu"
          action={
            <>
              <SegmentedControl
                label="Lọc yêu cầu"
                value={filter}
                onChange={setFilter}
                options={[{ value: "all", label: "Tất cả" }, { value: "problem", label: "Có vấn đề" }, { value: "privacy", label: "Thông tin cá nhân" }]}
              />
              <SegmentedControl label="Khoảng thời gian" value={range} onChange={pickRange} options={[{ value: "1h", label: "1 giờ" }, { value: "24h", label: "24 giờ" }, { value: "7d", label: "7 ngày" }]} />
              <Select aria-label="Tác vụ" value={task} onChange={(e) => setTask(e.target.value)}>
                <option value="">Mọi tác vụ</option>
                {Object.entries(TASK_LABEL).map(([k, v]) => <option key={k} value={k}>{v}</option>)}
              </Select>
            </>
          }
        >
          <PageState query={list} isEmpty={() => list.items.length === 0} loading={<Skeleton lines={6} />} empty={<EmptyState title="Không có yêu cầu nào khớp." />}>
            <DataTable caption="Yêu cầu gửi tới model, mới nhất trước" columns={columns} rows={list.items} rowKey={(r) => r.id} dense onRowClick={(r) => setOpen(r)} activeKey={open?.id} />
            {list.hasNextPage && <Button onClick={() => void list.fetchNextPage()} disabled={list.isFetchingNextPage}>Xem thêm</Button>}
          </PageState>
        </Section>
      </PageState>
      {open && <RequestDrawer row={open} onClose={() => setOpen(null)} />}
    </>
  );
}
