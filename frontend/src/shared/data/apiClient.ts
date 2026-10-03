import { ApiError, AUTH_CODES } from "./ApiError";
import { netStatus } from "./netStatus";
import { tokenStore } from "./tokenStore";

// Máy khách HTTP duy nhất của frontend (cấm fetch trần ngoài shared/data — luật ep/no-raw-fetch). SRS FEAT-ui-foundation 6.1–6.3.
export const API_ORIGIN = (process.env.NEXT_PUBLIC_API_URL ?? "").replace(/\/$/, "");
const BASE = `${API_ORIGIN}/api/v1`;
const TIMEOUT_MS = 15_000;
const RETRY_DELAYS = [300, 900];
const RETRY_AFTER_MAX_S = 5;
const AUTH_EVENT_GAP_MS = 10_000;

export type ApiResult<T> = { data: T; status: number; etag?: string; replayed?: boolean };
export type RequestOpts = {
  query?: Record<string, string | number | boolean | undefined | null>;
  headers?: Record<string, string>;
  signal?: AbortSignal;
  /** PUT / PATCH / DELETE tự sinh Idempotency-Key (POST luôn có). */
  idempotent?: boolean;
  /** ép khoá (gửi lại cùng ý định). */
  idempotencyKey?: string;
};

const KEY_RE = /^[A-Za-z0-9._:-]{8,128}$/;
export function newIdempotencyKey(): string {
  return `ep-${crypto.randomUUID()}`;
}
export { KEY_RE as IDEMPOTENCY_KEY_RE };

const etagCache = new Map<string, { etag: string; data: unknown }>();
let lastAuthEvent = -Infinity;

function emitAuthExpired() {
  const now = Date.now();
  if (now - lastAuthEvent < AUTH_EVENT_GAP_MS) return;
  lastAuthEvent = now;
  if (typeof window !== "undefined") window.dispatchEvent(new CustomEvent("auth:expired"));
}

function resolve(path: string, query?: RequestOpts["query"]): string {
  let url: string;
  if (/^[a-z][a-z0-9+.-]*:\/\//i.test(path) || path.startsWith("//")) {
    const target = new URL(path, typeof window !== "undefined" ? window.location.href : "http://localhost");
    const own = API_ORIGIN || (typeof window !== "undefined" ? window.location.origin : "");
    if (target.origin !== new URL(own, "http://localhost").origin) {
      throw new ApiError({ status: 0, code: "BAD_TARGET", message: "Từ chối gửi tới origin khác." });
    }
    url = target.toString();
  } else {
    url = BASE + (path.startsWith("/") ? path : `/${path}`);
  }
  if (query) {
    const qs = new URLSearchParams();
    for (const [k, v] of Object.entries(query)) if (v !== undefined && v !== null && v !== "") qs.set(k, String(v));
    const s = qs.toString();
    if (s) url += (url.includes("?") ? "&" : "?") + s;
  }
  return url;
}

const sleep = (ms: number, signal?: AbortSignal) =>
  new Promise<void>((res, rej) => {
    const t = setTimeout(res, ms);
    signal?.addEventListener("abort", () => { clearTimeout(t); rej(new ApiError({ status: 0, code: "ABORTED" })); }, { once: true });
  });

function parseRetryAfter(h: string | null, body?: { retry_after?: unknown }): number | undefined {
  const fromBody = typeof body?.retry_after === "number" ? body.retry_after : undefined;
  if (fromBody && fromBody > 0) return fromBody;
  const n = h ? Number(h) : NaN;
  return Number.isFinite(n) && n > 0 ? n : undefined;
}

async function decodeError(res: Response): Promise<ApiError> {
  const retryHeader = res.headers.get("Retry-After");
  const headerTrace = res.headers.get("X-Request-Id") ?? undefined; // thân thiếu trace_id thì lấy từ header
  const text = await res.text().catch(() => "");
  const isJson = (res.headers.get("Content-Type") ?? "").includes("json");
  if (!text.trim() || !isJson) return new ApiError({ status: res.status, code: "BAD_GATEWAY", traceId: headerTrace, retryAfter: parseRetryAfter(retryHeader) });
  try {
    const b = JSON.parse(text) as { code?: string; message?: string; trace_id?: string; details?: unknown; retry_after?: number };
    if (typeof b.code !== "string") return new ApiError({ status: res.status, code: "BAD_GATEWAY", traceId: headerTrace });
    return new ApiError({ status: res.status, code: b.code, message: b.message, traceId: b.trace_id || headerTrace, details: b.details, retryAfter: parseRetryAfter(retryHeader, b) });
  } catch {
    return new ApiError({ status: res.status, code: "PARSE_ERROR", traceId: headerTrace });
  }
}

type Attempt<T> = () => Promise<ApiResult<T>>;

function retryable(err: ApiError, method: string, hasKey: boolean): boolean {
  if (err.code === "ABORTED" || err.code === "BAD_TARGET") return false;
  if (method === "GET") {
    if (err.code === "NETWORK" || err.code === "BAD_GATEWAY") return true;
    if (err.status === 502 || err.status === 503 || err.status === 504) return true;
    if (err.status === 429) return true;
    return false;
  }
  if (!hasKey) return false;
  if (err.code === "NETWORK") return true;
  return err.code === "IDEMPOTENCY_IN_PROGRESS";
}

async function withRetry<T>(attempt: Attempt<T>, method: string, hasKey: boolean, signal?: AbortSignal): Promise<ApiResult<T>> {
  for (let i = 0; ; i++) {
    try {
      return await attempt();
    } catch (e) {
      if (!(e instanceof ApiError) || i >= RETRY_DELAYS.length || !retryable(e, method, hasKey)) throw e;
      let wait: number;
      if (e.retryAfter !== undefined) {
        if (e.retryAfter > RETRY_AFTER_MAX_S) throw e; // quá lâu: không tự thử, trả lỗi kèm retryAfter
        wait = e.retryAfter * 1000;
      } else {
        wait = RETRY_DELAYS[i] * (0.8 + Math.random() * 0.4);
      }
      await sleep(wait, signal);
    }
  }
}

async function request<T>(method: string, path: string, body: unknown, opts: RequestOpts = {}): Promise<ApiResult<T>> {
  const url = resolve(path, opts.query); // BAD_TARGET ném TRƯỚC khi gửi — không kèm Authorization
  const upper = method.toUpperCase();
  const key = opts.idempotencyKey ?? (upper === "POST" || opts.idempotent ? newIdempotencyKey() : undefined);
  const hasBody = body !== undefined;

  const attempt: Attempt<T> = async () => {
    const headers: Record<string, string> = { Accept: "application/json", "X-Request-Id": crypto.randomUUID(), ...opts.headers };
    const tk = tokenStore.get();
    if (tk) headers.Authorization = `Bearer ${tk}`;
    if (hasBody) headers["Content-Type"] = "application/json";
    if (key) headers["Idempotency-Key"] = key;
    const cached = upper === "GET" ? etagCache.get(url) : undefined;
    if (cached) headers["If-None-Match"] = cached.etag;

    const timeout = AbortSignal.timeout(TIMEOUT_MS);
    const signal = opts.signal ? AbortSignal.any([opts.signal, timeout]) : timeout;
    let res: Response;
    try {
      res = await fetch(url, { method: upper, headers, body: hasBody ? JSON.stringify(body) : undefined, credentials: "include", signal });
    } catch {
      if (opts.signal?.aborted) throw new ApiError({ status: 0, code: "ABORTED" });
      netStatus.reportFailure();
      throw new ApiError({ status: 0, code: "NETWORK" });
    }
    netStatus.reportSuccess();

    if (res.status === 304 && cached) return { data: cached.data as T, status: 304, etag: cached.etag };
    if (!res.ok) {
      const err = await decodeError(res);
      if (AUTH_CODES.includes(err.code)) {
        tokenStore.clear();
        emitAuthExpired();
      }
      throw err;
    }
    const replayed = res.headers.get("Idempotent-Replayed") === "true" || undefined;
    const etag = res.headers.get("ETag") ?? undefined;
    if (res.status === 204) return { data: undefined as T, status: 204, replayed };
    let data: T;
    try {
      data = (await res.json()) as T;
    } catch {
      throw new ApiError({ status: res.status, code: "PARSE_ERROR" });
    }
    if (upper === "GET" && etag) etagCache.set(url, { etag, data });
    return { data, status: res.status, etag, replayed };
  };

  return withRetry(attempt, upper, Boolean(key), opts.signal);
}

export const apiClient = {
  get: <T>(path: string, opts?: RequestOpts) => request<T>("GET", path, undefined, opts),
  post: <T>(path: string, body?: unknown, opts?: RequestOpts) => request<T>("POST", path, body ?? {}, opts),
  put: <T>(path: string, body?: unknown, opts?: RequestOpts) => request<T>("PUT", path, body ?? {}, opts),
  patch: <T>(path: string, body?: unknown, opts?: RequestOpts) => request<T>("PATCH", path, body ?? {}, opts),
  delete: <T>(path: string, opts?: RequestOpts) => request<T>("DELETE", path, undefined, opts),
};

/** Dùng cho test: xoá cache ETag và bộ đếm sự kiện hết hạn. */
export function resetApiClientState() {
  etagCache.clear();
  lastAuthEvent = -Infinity;
}
