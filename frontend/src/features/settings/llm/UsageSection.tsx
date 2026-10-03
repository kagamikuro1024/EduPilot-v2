"use client";

import { useQueryClient } from "@tanstack/react-query";
import { useState } from "react";
import { ApiErrorNotice, apiClient, fieldErrors } from "@/shared/data";
import { Button, DataTable, EmptyState, Field, InlineNotice, Input, SegmentedControl, Skeleton, type Column } from "@/shared/ui";
import { ConflictNotice } from "./ConflictNotice";
import { KEY, useBudget, useSaver, useUsage, type Budget, type UsageItem } from "./api";
import { fmtInt, fmtVnd, taskLabel } from "./labels";
import s from "./llm.module.css";

const cols: Column<UsageItem>[] = [
  { key: "task", header: "Tác vụ", render: (r) => taskLabel(r.key), primary: true },
  { key: "calls", header: "Lượt gọi", render: (r) => fmtInt(r.calls), align: "end" },
  { key: "tokens", header: "Token vào / ra", render: (r) => `${fmtInt(r.tokens_in)} / ${fmtInt(r.tokens_out)}`, align: "end" },
  { key: "cost", header: "Chi phí ước tính", render: (r) => fmtVnd(r.cost_est), align: "end" },
  { key: "p95", header: "Độ trễ p95", render: (r) => `${fmtInt(r.latency_p95_ms)} ms`, align: "end" },
  { key: "errors", header: "Lỗi", render: (r) => fmtInt(r.errors), align: "end" },
  { key: "degraded", header: "Chạy rút gọn", render: (r) => fmtInt(r.degraded), align: "end" },
];

function BudgetLine({ b, canEdit }: { b: Budget; canEdit: boolean }) {
  const qc = useQueryClient();
  const saver = useSaver(qc, [KEY.budget]);
  const [edit, setEdit] = useState(false);
  const [day, setDay] = useState(b.daily_limit ? String(Math.round(Number(b.daily_limit))) : "");
  const [month, setMonth] = useState(b.monthly_limit ? String(Math.round(Number(b.monthly_limit))) : "");
  const server = saver.error ? fieldErrors(saver.error) : {};
  const pct = b.pct_today ?? b.pct_month ?? 0;

  async function save() {
    const ok = await saver.run((v) => apiClient.put("/admin/llm/budget", { daily_limit: day.trim() || null, monthly_limit: month.trim() || null, version: v ?? b.version }, { idempotent: true }));
    if (ok) setEdit(false);
  }
  return (
    <div className={s.rows} data-part="budget">
      <p>
        Hôm nay {fmtVnd(b.spent_today)} / {b.daily_limit ? fmtVnd(b.daily_limit) : "không giới hạn"} · tháng này {fmtVnd(b.spent_month)} / {b.monthly_limit ? fmtVnd(b.monthly_limit) : "không giới hạn"}
      </p>
      <div className={s.meter} aria-hidden>
        <span style={{ width: `${Math.min(100, Number(pct))}%` }} />
      </div>
      {b.state === "warn" && <InlineNotice tone="warning">Đã dùng 80 % ngân sách ngày. Đến 100 %, việc chạy nền sẽ tạm dừng và chat chuyển sang mô hình rẻ hơn.</InlineNotice>}
      {b.state === "exhausted" && <InlineNotice tone="danger">Ngân sách đã dùng hết. Việc chạy nền tạm dừng; chat dùng mô hình rẻ hơn cho tới khi sang kỳ mới hoặc nâng hạn mức.</InlineNotice>}
      {canEdit &&
        (edit ? (
          <form className={s.form} onSubmit={(e) => { e.preventDefault(); void save(); }}>
            <div className={s.pair}>
              <Field label="Hạn mức ngày (đ)" helper="Để trống = không giới hạn" error={server.daily_limit}>
                {(id, d) => <Input id={id} aria-describedby={d} invalid={Boolean(server.daily_limit)} inputMode="numeric" value={day} onChange={(e) => setDay(e.target.value)} />}
              </Field>
              <Field label="Hạn mức tháng (đ)" error={server.monthly_limit}>
                {(id, d) => <Input id={id} aria-describedby={d} invalid={Boolean(server.monthly_limit)} inputMode="numeric" value={month} onChange={(e) => setMonth(e.target.value)} />}
              </Field>
            </div>
            {saver.error?.code === "VERSION_CONFLICT" && <ConflictNotice onKeepMine={() => void saver.keepMine().then((ok) => ok && setEdit(false))} onUseTheirs={() => void saver.useTheirs().then(() => setEdit(false))} />}
            {saver.error && saver.error.code !== "VERSION_CONFLICT" && Object.keys(server).length === 0 && <ApiErrorNotice error={saver.error} />}
            <div className={s.formActions}>
              <Button type="submit" variant="primary" loading={saver.pending}>Lưu</Button>
              <Button variant="ghost" onClick={() => { setEdit(false); saver.clear(); }}>Hủy</Button>
            </div>
          </form>
        ) : (
          <div>
            <Button variant="text" onClick={() => setEdit(true)}>Đổi hạn mức</Button>
          </div>
        ))}
    </div>
  );
}

/** Mức dùng và ngân sách (đọc phụ, không đánh số). Giảng viên chỉ xem (US-P1-05 AC8). */
export function UsageSection({ canEdit }: { canEdit: boolean }) {
  const [days, setDays] = useState(7);
  const usage = useUsage(days);
  const budget = useBudget();
  return (
    <div className={s.rows}>
      {budget.isPending ? <Skeleton lines={2} /> : budget.isError ? <ApiErrorNotice error={budget.error} onRetry={() => void budget.refetch()} /> : <BudgetLine key={budget.data.version} b={budget.data} canEdit={canEdit} />}
      <SegmentedControl
        label="Khoảng thời gian"
        value={String(days)}
        onChange={(v) => setDays(Number(v))}
        options={[{ value: "7", label: "7 ngày" }, { value: "30", label: "30 ngày" }]}
      />
      {usage.isError ? (
        <ApiErrorNotice error={usage.error} onRetry={() => void usage.refetch()} />
      ) : (
        <DataTable
          caption="Mức dùng theo tác vụ"
          columns={cols}
          rows={usage.data?.items ?? []}
          rowKey={(r) => r.key}
          loading={usage.isPending ? 3 : undefined}
          empty={
            <EmptyState title="Chưa có lượt gọi nào trong khoảng này." action={days === 7 ? <Button onClick={() => setDays(30)}>Xem 30 ngày</Button> : <Button onClick={() => setDays(7)}>Xem 7 ngày</Button>}>
              Khi hệ thống gọi mô hình, mức dùng sẽ hiện ở đây.
            </EmptyState>
          }
        />
      )}
    </div>
  );
}
