"use client";

import { useMutation, useQuery, useQueryClient, type QueryClient } from "@tanstack/react-query";
import { useCallback, useRef, useState } from "react";
import { ApiError, apiClient } from "@/shared/data";

// Kiểu và truy vấn của /settings/llm — khớp hợp đồng thật `backend-go/api/openapi.yaml` (tiền là CHUỖI, khoá API không bao giờ về).
export type LlmModel = { id: string; model: string; kind: "chat" | "embedding"; dims: number | null; price_in: string; price_out: string; enabled: boolean };
export type LastTest = { ok: boolean | null; at: string | null; error_kind: string | null };
export type Provider = {
  id: string;
  type: string;
  name: string;
  base_url: string | null;
  enabled: boolean;
  has_key: boolean;
  key_status: string;
  rpm_limit: number | null;
  tpm_limit: number | null;
  last_test: LastTest;
  circuit: string;
  models: LlmModel[];
  version: number;
};
export type ProviderList = { items: Provider[]; env_fallback: { active: boolean; providers: string[] } };
export type ChainItem = { model_id: string; provider_id: string; provider_name: string; model: string };
export type Route = { task: string; lane: string; chain: ChainItem[]; params: Record<string, number>; version: number };
export type RoutesRes = { items: Route[]; embedding: { model_id: string; provider_name: string; model: string; dims: number; reindex_required?: boolean } | null };
export type UsageItem = { key: string; calls: number; tokens_in: number; tokens_out: number; cost_est: string; latency_p50_ms: number; latency_p95_ms: number; errors: number; degraded: number };
export type Usage = { from: string; to: string; group: string; items: UsageItem[] };
export type Budget = {
  scope: string;
  daily_limit: string | null;
  monthly_limit: string | null;
  spent_today: string;
  spent_month: string;
  pct_today: number | null;
  pct_month: number | null;
  state: "ok" | "warn" | "exhausted";
  version: number;
};
export type TestResult = { ok: boolean; error_kind?: string; message?: string; latency_ms: number };

export const KEY = { providers: ["llm", "providers"], routes: ["llm", "routes"], budget: ["llm", "budget"], usage: (days: number) => ["llm", "usage", days] } as const;

export const useProviders = () => useQuery({ queryKey: KEY.providers, queryFn: async ({ signal }) => (await apiClient.get<ProviderList>("/admin/llm/providers", { signal })).data });
export const useRoutes = () => useQuery({ queryKey: KEY.routes, queryFn: async ({ signal }) => (await apiClient.get<RoutesRes>("/admin/llm/routes", { signal })).data });
export const useBudget = () => useQuery({ queryKey: KEY.budget, queryFn: async ({ signal }) => (await apiClient.get<Budget>("/admin/llm/budget", { signal })).data });
export const useUsage = (days: number) =>
  useQuery({
    queryKey: KEY.usage(days),
    queryFn: async ({ signal }) => {
      const to = new Date();
      const from = new Date(to.getTime() - days * 24 * 3600 * 1000);
      return (await apiClient.get<Usage>("/admin/llm/usage", { signal, query: { from: from.toISOString(), to: to.toISOString(), group: "task" } })).data;
    },
  });

/** Mô hình trò chuyện của các nhà đang bật, nhóm theo nhà cung cấp (cho chọn mô hình chính / dự phòng). */
export function chatGroups(providers: Provider[]) {
  return providers.filter((p) => p.enabled).map((p) => ({ provider: p.name, models: p.models.filter((m) => m.kind === "chat" && m.enabled).map((m) => ({ id: m.id, label: m.model })) })).filter((g) => g.models.length > 0);
}

/**
 * Lưu một thay đổi có `version`: theo dõi đang lưu / lỗi; gặp VERSION_CONFLICT thì giữ lại lần ghi cuối để "Giữ bản của tôi" gửi lại với
 * version hiện tại của máy chủ, còn "Dùng bản mới" chỉ làm mới truy vấn. Không bao giờ ghi đè im lặng (US-P1-05 AC9).
 */
export function useSaver(qc: QueryClient, invalidate: ReadonlyArray<readonly string[]>) {
  const [state, setState] = useState<{ pending: boolean; error?: ApiError }>({ pending: false });
  const last = useRef<((version?: number) => Promise<unknown>) | null>(null);
  const run = useCallback(
    async (fn: (version?: number) => Promise<unknown>, version?: number): Promise<boolean> => {
      last.current = fn;
      setState({ pending: true });
      try {
        await fn(version);
        await Promise.all(invalidate.map((k) => qc.invalidateQueries({ queryKey: k as string[] })));
        setState({ pending: false });
        return true;
      } catch (e) {
        setState({ pending: false, error: e instanceof ApiError ? e : new ApiError({ status: 0, code: "NETWORK" }) });
        return false;
      }
    },
    [qc, invalidate],
  );
  const keepMine = useCallback(() => (last.current ? run(last.current, state.error?.conflict?.currentVersion) : Promise.resolve(false)), [run, state.error]);
  const useTheirs = useCallback(async () => {
    setState({ pending: false });
    await Promise.all(invalidate.map((k) => qc.invalidateQueries({ queryKey: k as string[] })));
  }, [qc, invalidate]);
  const retry = useCallback(() => (last.current ? run(last.current) : Promise.resolve(false)), [run]);
  return { ...state, run, keepMine, useTheirs, retry, clear: () => setState({ pending: false }) };
}

export { useMutation, useQueryClient };
