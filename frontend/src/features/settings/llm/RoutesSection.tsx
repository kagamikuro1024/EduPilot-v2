"use client";

import { useState } from "react";
import { Button, Field, Input, InlineNotice, Select } from "@/shared/ui";
import { LLMRouteTable, type RouteRow } from "@/shared/domain";
import { chatGroups, type Provider, type Route } from "./api";
import { LANE_LABEL, TASKS, taskLabel } from "./labels";
import { MAX_CHAIN, changeLabel, chainItemFor, useRouteWriter } from "./routeWriter";
import s from "./llm.module.css";

const PARAMS = [
  { key: "temperature", label: "Nhiệt độ", unit: "0–2", lo: 0, hi: 2, int: false },
  { key: "max_tokens", label: "Số token tối đa mỗi câu trả lời", unit: "1–32.768", lo: 1, hi: 32768, int: true },
  { key: "timeout_s", label: "Thời gian chờ", unit: "giây, 1–300", lo: 1, hi: 300, int: true },
  { key: "retries", label: "Số lần thử lại", unit: "0–5", lo: 0, hi: 5, int: true },
] as const;

function Advanced({ route, draft, setDraft, onSave, readOnly }: { route: Route; draft: Record<string, string>; setDraft: (d: Record<string, string>) => void; onSave: (p: Record<string, number>) => void; readOnly: boolean }) {
  const [errs, setErrs] = useState<Record<string, string>>({});
  function save() {
    const e: Record<string, string> = {};
    const out: Record<string, number> = {};
    for (const p of PARAMS) {
      const raw = (draft[p.key] ?? "").trim().replace(",", ".");
      if (raw === "") continue;
      const n = Number(raw);
      if (!Number.isFinite(n) || n < p.lo || n > p.hi || (p.int && !Number.isInteger(n))) e[p.key] = `${p.label}: nhập ${p.int ? "số nguyên" : "số"} trong khoảng ${p.unit}.`;
      else out[p.key] = n;
    }
    setErrs(e);
    if (Object.keys(e).length === 0) onSave(out);
  }
  return (
    <div className={s.advanced} data-part="route-advanced">
      <div className={s.pair}>
        {PARAMS.map((p) => (
          <Field key={p.key} label={p.label} helper={`Đơn vị: ${p.unit}. Để trống = mặc định.`} error={errs[p.key]}>
            {(id, d) => (
              <Input id={id} aria-describedby={d} invalid={Boolean(errs[p.key])} inputMode="decimal" readOnly={readOnly} value={draft[p.key] ?? ""} onChange={(e) => setDraft({ ...draft, [p.key]: e.target.value })} />
            )}
          </Field>
        ))}
      </div>
      {!readOnly && (
        <div>
          <Button variant="primary" size="sm" onClick={save}>
            Lưu
          </Button>
        </div>
      )}
      <span className="ep-sr-only">{route.task}</span>
    </div>
  );
}

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
              <p className={s.sub}>Chưa chọn mô hình chính — đang dùng cấu hình mặc định của máy chủ.</p>
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
            {chain.length === 1 && <InlineNotice tone="warning" compact>Chưa có dự phòng — nếu nhà cung cấp lỗi, người dùng sẽ thấy câu trả lời rút gọn.</InlineNotice>}
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
