import { API_BASE } from "./endpoint";
import { tokenStore } from "./tokenStore";

// Trạng thái phiên đăng nhập (US-P2-02): access token ở bộ nhớ (`tokenStore`), refresh token ở cookie httpOnly (JS không thấy).
// Máy trạng thái: initializing → authenticated | anonymous | revoked. Không localStorage / sessionStorage.
export type AuthUser = { id: string; email: string; full_name: string; role: "ADMIN" | "TEACHER" | "TA" | "STUDENT"; status: string; email_verified: boolean };
export type SessionPayload = { access_token: string; token_type: string; expires_in: number; user: AuthUser };
export type AuthSnapshot = { status: "initializing" | "authenticated" | "anonymous" | "revoked"; user: AuthUser | null; reason?: string };

const INITIAL: AuthSnapshot = { status: "initializing", user: null };
let snap: AuthSnapshot = INITIAL;
const listeners = new Set<() => void>();

function set(next: AuthSnapshot) {
  snap = next;
  listeners.forEach((f) => f());
}

export const authStore = {
  get: () => snap,
  server: () => INITIAL,
  subscribe(f: () => void) {
    listeners.add(f);
    return () => {
      listeners.delete(f);
    };
  },
};

/** Nhận phiên mới (đăng nhập / làm mới thành công). */
export function acceptSession(p: SessionPayload) {
  tokenStore.set(p.access_token);
  set({ status: "authenticated", user: p.user });
}

/** Bỏ phiên ở máy này (đăng xuất, hết hạn, bị thu hồi). */
export function dropSession(status: "anonymous" | "revoked", reason?: string) {
  tokenStore.clear();
  set({ status, user: null, reason });
}

export type RefreshResult = { ok: true; user: AuthUser } | { ok: false; code: string; reason?: string };

let inflight: Promise<RefreshResult> | null = null;

/**
 * Làm mới phiên bằng cookie `ep_rt`. Gộp trong tab (một lời gọi tại một thời điểm) và tuần tự giữa các tab bằng
 * `navigator.locks` tên `ep-refresh`: tab sau dùng cookie đã xoay của tab trước nên không bị coi là dùng lại token cũ.
 */
export function refreshSession(): Promise<RefreshResult> {
  inflight ??= run().finally(() => {
    inflight = null;
  });
  return inflight;
}

async function run(): Promise<RefreshResult> {
  // ponytail: không có navigator.locks (trình duyệt cũ) → chỉ gộp trong tab; hai tab đồng thời có thể bị coi là dùng lại token.
  if (typeof navigator !== "undefined" && navigator.locks) {
    const r: RefreshResult = await navigator.locks.request("ep-refresh", () => doRefresh());
    return r;
  }
  return doRefresh();
}

async function doRefresh(): Promise<RefreshResult> {
  let res: Response;
  try {
    res = await fetch(`${API_BASE}/auth/refresh`, {
      method: "POST",
      credentials: "include",
      headers: { Accept: "application/json", "Content-Type": "application/json", "X-Request-Id": crypto.randomUUID() },
      signal: AbortSignal.timeout(15_000),
    });
  } catch {
    return { ok: false, code: "NETWORK" }; // không rõ phiên còn hay mất: giữ nguyên trạng thái
  }
  if (res.ok) {
    try {
      const body = (await res.json()) as SessionPayload;
      acceptSession(body);
      return { ok: true, user: body.user };
    } catch {
      return { ok: false, code: "PARSE_ERROR" };
    }
  }
  let code = "BAD_GATEWAY";
  let reason: string | undefined;
  try {
    const b = (await res.json()) as { code?: string; details?: { reason?: string } };
    if (typeof b.code === "string") code = b.code;
    reason = b.details?.reason;
  } catch {
    /* thân không đọc được: coi như lỗi cổng */
  }
  if (res.status === 401) dropSession(code === "SESSION_REVOKED" ? "revoked" : "anonymous", reason);
  return { ok: false, code, reason };
}

/** Dùng cho test. */
export function resetAuthState() {
  inflight = null;
  tokenStore.clear();
  set(INITIAL);
}
