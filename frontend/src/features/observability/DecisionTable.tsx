"use client";

import { useQuery } from "@tanstack/react-query";
import { useState } from "react";
import { ApiErrorNotice, apiClient } from "@/shared/data";
import { DataTable, EmptyState, Section, SegmentedControl, Skeleton } from "@/shared/ui";
import { GATES, type GateId } from "@/features/settings/llm/decisionApi";

type Row = { gate: GateId; state: "off" | "on"; courses_enabled: number; courses_total: number; calls: number; fast_pct: number; fallback_pct: number; fail_closed_pct: number; p50_ms: number; p95_ms: number; error_pct: number; cost_vnd: string };
type Window = "24h" | "7d";

const pct = (n: number) => `${new Intl.NumberFormat("vi-VN", { maximumFractionDigits: 1 }).format(n)}%`;
const ms = (n: number) => `${Math.round(n)} ms`;
const vnd = (s: string) => `${new Intl.NumberFormat("vi-VN", { maximumFractionDigits: 0 }).format(Number(s))} đ`;

/** Bảng «Quyết định nhanh» (US-DM-07): theo cổng, từ llm_audit; cổng tắt hiện «—»; không nội dung. Chỉ ADMIN. */
export function DecisionTable() {
  const [window, setWindow] = useState<Window>("24h");
  const q = useQuery({ queryKey: ["observability", "decision", window], queryFn: async ({ signal }) => (await apiClient.get<{ gates: Row[] }>("/admin/observability/decision", { signal, query: { window } })).data.gates });
  const label = (g: GateId) => GATES.find((x) => x.gate === g)?.label ?? g;
  const dash = <span aria-label="không có số">—</span>;
  const num = (r: Row, v: string) => (r.state === "off" ? dash : v);

  return (
    <Section
      title="Quyết định nhanh"
      part="decision-observability"
      action={<SegmentedControl label="Cửa sổ thời gian" value={window} onChange={setWindow} options={[{ value: "24h", label: "24 giờ" }, { value: "7d", label: "7 ngày" }]} />}
    >
      {q.isPending ? (
        <Skeleton lines={4} />
      ) : q.isError ? (
        <ApiErrorNotice error={q.error} onRetry={() => void q.refetch()} />
      ) : (
        <>
          <DataTable<Row>
            caption="Quyết định nhanh theo cổng"
            rows={q.data}
            rowKey={(r) => r.gate}
            dense
            mobile="scroll"
            columns={[
              { key: "gate", header: "Cổng", primary: true, frozen: true, render: (r) => label(r.gate) },
              { key: "state", header: "Trạng thái", render: (r) => (r.state === "off" ? "Tắt" : `Bật ở ${r.courses_enabled}/${r.courses_total} lớp`) },
              { key: "calls", header: "Lời gọi", align: "end", render: (r) => num(r, String(r.calls)) },
              { key: "fast", header: "Nhanh", align: "end", render: (r) => num(r, pct(r.fast_pct)) },
              { key: "fb", header: "Dự phòng", align: "end", render: (r) => num(r, pct(r.fallback_pct)) },
              { key: "fc", header: "Chặn an toàn", align: "end", render: (r) => num(r, pct(r.fail_closed_pct)) },
              { key: "p50", header: "p50", align: "end", render: (r) => num(r, ms(r.p50_ms)) },
              { key: "p95", header: "p95", align: "end", render: (r) => num(r, ms(r.p95_ms)) },
              { key: "err", header: "Lỗi", align: "end", render: (r) => num(r, pct(r.error_pct)) },
              { key: "cost", header: "Chi phí", align: "end", render: (r) => num(r, vnd(r.cost_vnd)) },
            ]}
          />
          {q.data.every((r) => r.calls === 0) && <EmptyState title="Chưa có lời gọi nào" />}
        </>
      )}
    </Section>
  );
}
