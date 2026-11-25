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

/** Đếm số lần phiên đổi (nhận / bỏ). Lời gọi làm mới bắt đầu ở thế hệ nào thì chỉ được áp kết quả nếu thế hệ chưa đổi (P-106-14). */
let epoch = 0;

function set(next: AuthSnapshot) {
  epoch++;
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

/** Tên thuộc tính `window` giữ lời gọi làm mới sớm do `<script>` trong `<head>` của root layout bắt đầu (U1). Dùng đúng MỘT lần. */
export const EARLY_REFRESH_KEY = "__epRefresh";
/** `window` giữ `{ u: url, p: Promise<dữ liệu | null> }` của GET sớm do `earlyPaint.ts` gửi (chỉ đọc thuần, một lần): `apiClient` dùng lại thay vì gửi GET thứ hai. */
export const EARLY_GET_KEY = "__epGet";

/** Lấy lời gọi làm mới sớm (nếu có) và xoá khỏi `window`: mỗi token làm mới chỉ được dùng một lần. */
function takeEarlyRefresh(): Promise<Response> | null {
  if (typeof window === "undefined") return null;
  const w = window as unknown as Record<string, Promise<Response> | undefined>;
  const p = w[EARLY_REFRESH_KEY];
  delete w[EARLY_REFRESH_KEY];
  return p ?? null;
}

async function doRefresh(): Promise<RefreshResult> {
  const early = takeEarlyRefresh();
  // Lời gọi sớm được gửi lúc tải trang (epoch 0), trước mọi đăng nhập / đăng xuất trong trang này: dù doRefresh chỉ bắt đầu sau khi chờ khoá `ep-refresh`, kết quả của nó
  // vẫn là của THỜI ĐIỂM GỬI — đăng nhập xong rồi mới nhận 401 cũ thì phải bị coi là lỗi thời, không được xoá phiên mới (P-106-14).
  const startedAt = early ? 0 : epoch;
  // Trong lúc chờ, người dùng đăng nhập (hoặc phiên khác đổi): kết quả làm mới bằng cookie cũ không được ghi đè / xoá phiên mới (P-106-14).
  const superseded = (): RefreshResult | null => (epoch === startedAt ? null : snap.user ? { ok: true, user: snap.user } : { ok: false, code: "SUPERSEDED" });
  let res: Response;
  try {
    res =
      (await early) ??
      (await fetch(`${API_BASE}/auth/refresh`, {
        method: "POST",
        credentials: "include",
        headers: { Accept: "application/json", "Content-Type": "application/json", "X-Request-Id": crypto.randomUUID() },
        signal: AbortSignal.timeout(15_000),
      }));
  } catch {
    return { ok: false, code: "NETWORK" }; // không rõ phiên còn hay mất: giữ nguyên trạng thái
  }
  if (res.ok) {
    try {
      const body = (await res.json()) as SessionPayload;
      const newer = superseded();
      if (newer) return newer;
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
  const newer = superseded();
  if (newer) return newer;
  if (res.status === 401) dropSession(code === "SESSION_REVOKED" ? "revoked" : "anonymous", reason);
  return { ok: false, code, reason };
}

/** Dùng cho test. */
export function resetAuthState() {
  inflight = null;
  tokenStore.clear();
  set(INITIAL);
}
