"use client";

import dynamic from "next/dynamic";
import { useState } from "react";
import { Button, Field, Select, StatusText } from "@/shared/ui";
// nhập thẳng, không qua thùng `@/shared/domain` (thùng kéo cả AttendanceGrid, Markdown… vào gói trang: TL-1)
import { LLMRouteTable, type RouteRow } from "@/shared/domain/LLMRouteTable";
import { chatGroups, type Provider, type Route } from "./api";
import { LANE_LABEL, TASKS, taskLabel } from "./labels";
import { MAX_CHAIN, changeLabel, chainItemFor, useRouteWriter } from "./routeWriter";
import s from "./llm.module.css";

// «Cài đặt nâng cao» của một tác vụ chỉ mở sau khi bấm ⇒ phân mảnh riêng (ngân sách script /settings/llm).
const Advanced = dynamic(() => import("./RouteAdvanced").then((m) => m.Advanced), { loading: () => null });

/** Phần 2 — Mô hình theo tác vụ (LLMRouteTable + hoàn tác + cài đặt nâng cao). */
export function RouteTableSection({ routes, providers, canEdit, onAddProvider }: { routes: Route[]; providers: Provider[]; canEdit: boolean; onAddProvider: () => void }) {
  const writer = useRouteWriter();
  const groups = chatGroups(providers);
  const [open, setOpen] = useState<string | null>(null);
  const [drafts, setDrafts] = useState<Record<string, Record<string, string>>>({});
  const byTask = new Map(routes.map((r) => [r.task, r]));
  const rows: RouteRow[] = TASKS.map((t) => ({ task: t.task, label: t.label, laneLabel: LANE_LABEL[t.lane], modelId: byTask.get(t.task)?.chain[0]?.model_id ?? "" }));

  return (
    <LLMRouteTable
      rows={rows}
      groups={groups}
      readOnly={!canEdit}
      openTask={open}
      onToggleAdvanced={(t) => {
        setOpen((o) => (o === t ? null : t));
        setDrafts((d) => (d[t] ? d : { ...d, [t]: Object.fromEntries(Object.entries(byTask.get(t)?.params ?? {}).map(([k, v]) => [k, String(v)])) }));
      }}
      emptyAction={canEdit ? <Button onClick={onAddProvider}>Thêm nhà cung cấp</Button> : undefined}
      onChange={(task, modelId) => {
        const route = byTask.get(task);
        const item = chainItemFor(providers, modelId);
        if (!route || !item) return;
        const rest = route.chain.filter((c) => c.model_id !== modelId);
        void writer.write({ task, chain: [item, ...rest].slice(0, MAX_CHAIN), params: route.params, label: changeLabel(task, item.model) });
      }}
      renderExtra={(row) => (
        <>
          {open === row.task && byTask.get(row.task) && (
            <Advanced
              route={byTask.get(row.task)!}
              draft={drafts[row.task] ?? {}}
              setDraft={(d) => setDrafts((x) => ({ ...x, [row.task]: d }))}
              readOnly={!canEdit}
              onSave={(params) => void writer.write({ task: row.task, chain: byTask.get(row.task)!.chain, params, label: `Đã lưu cài đặt nâng cao của ${taskLabel(row.task)}` })}
            />
          )}
          {writer.extra(row.task)}
        </>
      )}
    />
  );
}

/** Phần 3 — Chuỗi dự phòng theo từng tác vụ (Lên / Xuống bằng nút, dùng được bằng bàn phím). */
export function FallbackSection({ routes, providers, canEdit }: { routes: Route[]; providers: Provider[]; canEdit: boolean }) {
  const writer = useRouteWriter();
  const [pick, setPick] = useState<Record<string, string>>({});
  const groups = chatGroups(providers);
  return (
    <ul className={s.list}>
      {TASKS.map((t) => {
        const route = routes.find((r) => r.task === t.task);
        const chain = route?.chain ?? [];
        const candidates = groups.map((g) => ({ ...g, models: g.models.filter((m) => !chain.some((c) => c.model_id === m.id)) })).filter((g) => g.models.length > 0);
        const set = (next: typeof chain, label: string) => void writer.write({ task: t.task, chain: next, params: route?.params ?? {}, label });
        const move = (i: number, d: -1 | 1) => {
          const next = chain.slice();
          [next[i], next[i + d]] = [next[i + d], next[i]];
          set(next, `Đã đổi thứ tự dự phòng của ${t.label}`);
        };
        return (
          <li key={t.task} className={s.item} data-part="fallback-row" data-task={t.task}>
            <p className="ep-item-title">{t.label}</p>
            {chain.length === 0 ? (
              <p className={s.sub}>Mặc định máy chủ</p>
            ) : (
              <ol className={s.chain}>
                {chain.map((c, i) => (
                  <li key={c.model_id}>
                    <span className={s.chainIndex}>{i === 0 ? "Chính" : `Dự phòng ${i}`}</span>
                    <span>
                      {c.model} <span className={s.sub}>· {c.provider_name}</span>
                    </span>
                    {canEdit && i > 0 && (
                      <span className={s.chainActions}>
                        <Button size="sm" variant="ghost" disabled={i === 1} aria-label={`Đưa ${c.model} lên trước, ${t.label}`} onClick={() => move(i, -1)}>
                          Lên
                        </Button>
                        <Button size="sm" variant="ghost" disabled={i === chain.length - 1} aria-label={`Đưa ${c.model} xuống sau, ${t.label}`} onClick={() => move(i, 1)}>
                          Xuống
                        </Button>
                        <Button size="sm" variant="ghost" aria-label={`Bỏ ${c.model} khỏi dự phòng, ${t.label}`} onClick={() => set(chain.filter((x) => x.model_id !== c.model_id), `Đã bỏ ${c.model} khỏi dự phòng của ${t.label}`)}>
                          Bỏ
                        </Button>
                      </span>
                    )}
                  </li>
                ))}
              </ol>
            )}
            {chain.length === 1 && <StatusText tone="amber">Chưa có dự phòng</StatusText>}
            {canEdit && chain.length >= 1 && chain.length < MAX_CHAIN && candidates.length > 0 && (
              <div className={s.addFallback}>
                <Field label={`Thêm dự phòng cho ${t.label}`}>
                  {(id) => (
                    <Select id={id} value={pick[t.task] ?? ""} onChange={(e) => setPick({ ...pick, [t.task]: e.target.value })}>
                      <option value="">Chọn mô hình</option>
                      {candidates.flatMap((g) =>
                        g.models.map((m) => (
                          <option key={m.id} value={m.id}>
                            {g.provider} · {m.label}
                          </option>
                        )),
                      )}
                    </Select>
                  )}
                </Field>
                <Button
                  size="sm"
                  disabled={!pick[t.task]}
                  onClick={() => {
                    const item = chainItemFor(providers, pick[t.task]);
                    if (item) set([...chain, item], `Đã thêm ${item.model} vào dự phòng của ${t.label}`);
                    setPick({ ...pick, [t.task]: "" });
                  }}
                >
                  Thêm
                </Button>
              </div>
            )}
            {writer.extra(t.task)}
          </li>
        );
      })}
    </ul>
  );
}
