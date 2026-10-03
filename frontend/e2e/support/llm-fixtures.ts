import type { Page } from "@playwright/test";
import { asDemo } from "./session";

// Dữ liệu theo HỢP ĐỒNG THẬT của gateway (backend-go/api/openapi.yaml + golden/llm/*.json) cho máy chủ giả ở :3312.
export const API = "http://localhost:3312";
const id = (n: number) => `00000000-0000-7000-8000-${String(n).padStart(12, "0")}`;
export const IDS = { openai: id(1), gemini: id(2), m1: id(11), m2: id(12), m3: id(13), e1: id(21), e2: id(22) };

export const provider = (over: Record<string, unknown> = {}) => ({
  id: IDS.openai, type: "openai", name: "OpenAI", base_url: null, enabled: true, has_key: true, key_status: "ok", rpm_limit: null, tpm_limit: null,
  last_test: { ok: true, at: "2026-10-29T02:12:00Z", error_kind: null }, circuit: "closed", version: 2,
  models: [
    { id: IDS.m1, model: "gpt-4o-mini", kind: "chat", dims: null, price_in: "4000.0000", price_out: "16000.0000", enabled: true },
    { id: IDS.m2, model: "gpt-4o", kind: "chat", dims: null, price_in: "60000.0000", price_out: "240000.0000", enabled: true },
    { id: IDS.e1, model: "text-embedding-3-small", kind: "embedding", dims: 1536, price_in: "20.0000", price_out: "0.0000", enabled: true },
  ],
  ...over,
});
export const gemini = (over: Record<string, unknown> = {}) =>
  provider({
    id: IDS.gemini, type: "gemini", name: "Gemini", last_test: { ok: false, at: "2026-10-29T02:05:00Z", error_kind: "AUTH" }, version: 1,
    models: [
      { id: IDS.m3, model: "gemini-2.0-flash", kind: "chat", dims: null, price_in: "3000.0000", price_out: "12000.0000", enabled: true },
      { id: IDS.e2, model: "gemini-embedding", kind: "embedding", dims: 1536, price_in: "10.0000", price_out: "0.0000", enabled: true },
    ],
    ...over,
  });
export const providers = (items = [provider(), gemini()]) => ({ items, env_fallback: { active: false, providers: [] } });

const item = (modelId: string, model: string, provId: string, name: string) => ({ model_id: modelId, provider_id: provId, provider_name: name, model });
export const CHAIN3 = [item(IDS.m1, "gpt-4o-mini", IDS.openai, "OpenAI"), item(IDS.m2, "gpt-4o", IDS.openai, "OpenAI"), item(IDS.m3, "gemini-2.0-flash", IDS.gemini, "Gemini")];
export const routes = (over: { chat?: unknown[]; reindex?: boolean } = {}) => ({
  items: [
    { task: "CHAT", lane: "INTERACTIVE", chain: over.chat ?? CHAIN3, params: { temperature: 0.2 }, version: 3 },
    { task: "CLASSIFY", lane: "NEAR_REALTIME", chain: [CHAIN3[0]], params: {}, version: 1 },
    { task: "UTILITY", lane: "NEAR_REALTIME", chain: [CHAIN3[0]], params: {}, version: 1 },
    { task: "GRADING", lane: "BATCH", chain: [CHAIN3[1], CHAIN3[2]], params: {}, version: 1 },
    { task: "QUESTION_GEN", lane: "BATCH", chain: [CHAIN3[1]], params: {}, version: 1 },
    { task: "INSIGHT", lane: "BATCH", chain: [], params: {}, version: 0 },
    { task: "EMBEDDING", lane: "BATCH", chain: [item(IDS.e1, "text-embedding-3-small", IDS.openai, "OpenAI")], params: {}, version: 2 },
  ],
  embedding: { model_id: IDS.e1, provider_name: "OpenAI", model: "text-embedding-3-small", dims: 1536, reindex_required: over.reindex ?? false },
});
export const budget = (over: Record<string, unknown> = {}) => ({
  scope: "system", daily_limit: "80000.00", monthly_limit: "2000000.00", spent_today: "42000.0000", spent_month: "1240000.0000", pct_today: 52.5, pct_month: 62, state: "ok", version: 3, ...over,
});
export const usage = (items: unknown[] = [
  { key: "CHAT", calls: 1840, tokens_in: 912000, tokens_out: 301000, cost_est: "1240000.0000", latency_p50_ms: 420, latency_p95_ms: 1180, errors: 3, degraded: 12 },
  { key: "GRADING", calls: 60, tokens_in: 400000, tokens_out: 90000, cost_est: "88000.0000", latency_p50_ms: 3000, latency_p95_ms: 9000, errors: 0, degraded: 0 },
]) => ({ from: "2026-10-22T00:00:00Z", to: "2026-10-29T00:00:00Z", group: "task", items });

export const err = (status: number, code: string, extra: Record<string, unknown> = {}) => ({ status, body: { code, message: "m", trace_id: "b".repeat(32), ...extra } });

export function jwt(role: "ADMIN" | "TEACHER") {
  const b64 = (o: unknown) => Buffer.from(JSON.stringify(o)).toString("base64url");
  return `${b64({ alg: "HS256", typ: "JWT" })}.${b64({ sub: "00000000-0000-7000-8000-0000000000a0", role, email: role === "ADMIN" ? "admin@ptit.edu.vn" : "gv@ptit.edu.vn", exp: 4102444800 })}.c2ln`;
}

type Step = Record<string, unknown>;
export const script = (page: Page, s: Record<string, Step[]>) => page.request.post(`${API}/__ctl/script`, { data: s });
export const reset = (page: Page) => page.request.post(`${API}/__ctl/reset`);
export type Entry = { t: number; method: string; path: string; search: string; headers: Record<string, string>; body: string };
export const getLog = async (page: Page, path?: string): Promise<Entry[]> => {
  const j = (await (await page.request.get(`${API}/__ctl/log`)).json()) as { log: Entry[] };
  return path ? j.log.filter((l) => l.path === path) : j.log;
};

const P = "/api/v1/admin/llm";
export const base = (over: Record<string, Step[]> = {}): Record<string, Step[]> => ({
  [`GET ${P}/providers`]: [{ body: providers() }],
  [`GET ${P}/routes`]: [{ body: routes() }],
  [`GET ${P}/budget`]: [{ body: budget() }],
  [`GET ${P}/usage`]: [{ body: usage() }],
  ...over,
});

/** Mở /settings/llm: phiên mô phỏng → dán token (bộ nhớ) ở cổng dev → màn thật. `scripts` ghi đè phản hồi mặc định. */
export async function openLlm(page: Page, context: Parameters<typeof asDemo>[0], role: "ADMIN" | "TEACHER", scripts: Record<string, Step[]> = {}) {
  await reset(page);
  await script(page, base(scripts));
  await asDemo(context, role === "ADMIN" ? "admin" : "teacher");
  await page.goto("/settings/llm");
  await page.locator("main input[type=password]").fill(jwt(role));
  await page.getByRole("button", { name: "Dùng token" }).click();
}
