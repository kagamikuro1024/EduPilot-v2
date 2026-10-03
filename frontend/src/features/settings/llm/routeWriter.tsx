"use client";

import { useQueryClient } from "@tanstack/react-query";
import { useCallback, useState, type ReactNode } from "react";
import { ApiError, ApiErrorNotice, apiClient } from "@/shared/data";
import { UndoLine } from "@/shared/ui";
import { ConflictNotice } from "./ConflictNotice";
import { KEY, type ChainItem, type Provider, type Route, type RoutesRes } from "./api";
import { taskLabel } from "./labels";

export const MAX_CHAIN = 4; // 1 mô hình chính + tối đa 3 dự phòng

type Pending = { task: string; chain: ChainItem[]; params: Record<string, number>; label: string };

/** Dựng ChainItem từ mô hình đã biết (để cập nhật lạc quan trước khi máy chủ trả về). */
export function chainItemFor(providers: Provider[], modelId: string): ChainItem | null {
  for (const p of providers) {
    const m = p.models.find((x) => x.id === modelId);
    if (m) return { model_id: m.id, provider_id: p.id, provider_name: p.name, model: m.model };
  }
  return null;
}

const patch = (res: RoutesRes, route: Route): RoutesRes => ({ ...res, items: res.items.map((r) => (r.task === route.task ? route : r)) });

/**
 * Ghi tuyến tác vụ: cập nhật lạc quan ngay (≤ 100 ms), dòng Hoàn tác 5 s tại hàng, Hoàn tác ghi NGAY, lỗi ⇒ hoàn UI + báo tại hàng,
 * 409 ⇒ hỏi giữ bản nào (US-P1-05 AC5, AC9). Một lần ghi = một PUT /admin/llm/routes mang cả chuỗi.
 */
export function useRouteWriter() {
  const qc = useQueryClient();
  const [undo, setUndo] = useState<Record<string, { label: string; prev: Route }>>({});
  const [errors, setErrors] = useState<Record<string, { error: ApiError; pending: Pending }>>({});

  const latest = useCallback((task: string) => qc.getQueryData<RoutesRes>(KEY.routes)?.items.find((r) => r.task === task), [qc]);

  const put = useCallback(
    async (task: string, chain: ChainItem[], params: Record<string, number>, version: number) => {
      const r = await apiClient.put<Route>("/admin/llm/routes", { task, chain: chain.map((c) => c.model_id), params, version }, { idempotent: true });
      return r.data;
    },
    [],
  );

  const write = useCallback(
    async (p: Pending, versionOverride?: number, withUndo = true) => {
      const cur = latest(p.task);
      if (!cur) return;
      const prev = cur;
      qc.setQueryData<RoutesRes>(KEY.routes, (res) => (res ? patch(res, { ...cur, chain: p.chain, params: p.params }) : res));
      setErrors((e) => {
        const { [p.task]: _drop, ...rest } = e;
        void _drop;
        return rest;
      });
      if (withUndo) setUndo((u) => ({ ...u, [p.task]: { label: p.label, prev } }));
      try {
        const saved = await put(p.task, p.chain, p.params, versionOverride ?? cur.version);
        qc.setQueryData<RoutesRes>(KEY.routes, (res) => (res ? patch(res, { ...cur, ...saved, chain: saved.chain ?? p.chain }) : res));
      } catch (e) {
        qc.setQueryData<RoutesRes>(KEY.routes, (res) => (res ? patch(res, prev) : res));
        setUndo((u) => {
          const { [p.task]: _d, ...rest } = u;
          void _d;
          return rest;
        });
        setErrors((x) => ({ ...x, [p.task]: { error: e instanceof ApiError ? e : new ApiError({ status: 0, code: "NETWORK" }), pending: p } }));
      }
    },
    [latest, put, qc],
  );

  const doUndo = useCallback(
    async (task: string) => {
      const u = undo[task];
      if (!u) return;
      setUndo((x) => {
        const { [task]: _d, ...rest } = x;
        void _d;
        return rest;
      });
      await write({ task, chain: u.prev.chain, params: u.prev.params, label: "" }, undefined, false); // ghi ngay, không chờ hết 5 s
    },
    [undo, write],
  );

  /** Dòng Hoàn tác / lỗi / xung đột của một tác vụ — đặt ngay dưới hàng đó. */
  const extra = useCallback(
    (task: string): ReactNode => (
      <>
        {undo[task] && <UndoLine message={undo[task].label} onUndo={() => void doUndo(task)} onDone={() => setUndo((x) => { const { [task]: _d, ...rest } = x; void _d; return rest; })} />}
        {errors[task]?.error.code === "VERSION_CONFLICT" && (
          <ConflictNotice
            onKeepMine={() => void write(errors[task].pending, errors[task].error.conflict?.currentVersion)}
            onUseTheirs={() => {
              setErrors((x) => { const { [task]: _d, ...rest } = x; void _d; return rest; });
              void qc.invalidateQueries({ queryKey: KEY.routes });
            }}
          />
        )}
        {errors[task] && errors[task].error.code !== "VERSION_CONFLICT" && <ApiErrorNotice error={errors[task].error} onRetry={() => void write(errors[task].pending)} />}
      </>
    ),
    [undo, errors, doUndo, write, qc],
  );

  return { write, extra };
}

export const changeLabel = (task: string, model: string) => `Đã chuyển ${taskLabel(task)} sang ${model}`;
