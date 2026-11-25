// `#token` của liên kết QR không tới được máy chủ và sẽ MẤT khi AuthGate chuyển tới `/login?next=` (chỉ giữ đường dẫn + query).
// Layout của `/checkin` cất `{c, token, at}` vào sessionStorage (KHÔNG localStorage) rồi xoá fragment khỏi thanh địa chỉ; trang chỉ đọc từ đây và xoá khi dùng (US-AS-09 AC1, AC2).

const KEY = "ep.checkin";
/** `#token` mới vừa được cất (điều hướng chỉ đổi hash / trang phục hồi từ bfcache): màn `/checkin` đang dựng phải gửi token mới. */
export const STASH_EVENT = "ep:checkin-stash";
const MAX_AGE_MS = 60_000;
/** Cú pháp token điểm danh (`attendance.ParseToken`, backend-go/internal/attendance/token.go): `uuid.bước.mac`. Fragment khác (vd skip link `#main`) KHÔNG phải token. */
const TOKEN_RE = /^[0-9a-f-]{36}\.[0-9]+\.[A-Za-z0-9]{16}$/;

export type Stash = { c: string; token: string; at: number };

export function stashFromLocation(): void {
  const token = location.hash.slice(1);
  if (!TOKEN_RE.test(token)) return; // không phải token: để nguyên URL (skip link `#main` phải còn tác dụng)
  try {
    sessionStorage.setItem(KEY, JSON.stringify({ c: new URLSearchParams(location.search).get("c") ?? "", token, at: Date.now() } satisfies Stash));
  } catch {
    /* kho bị chặn: không có token, màn rơi về ô nhập mã */
  }
  history.replaceState(history.state, "", location.pathname + location.search);
  window.dispatchEvent(new Event(STASH_EVENT));
}

export function readStash(): Stash | null {
  try {
    const raw = sessionStorage.getItem(KEY);
    return raw ? (JSON.parse(raw) as Stash) : null;
  } catch {
    return null;
  }
}

export function clearStash(): void {
  try {
    sessionStorage.removeItem(KEY);
  } catch {
    /* bỏ qua */
  }
}

export const stashExpired = (s: Stash, now = Date.now()) => now - s.at > MAX_AGE_MS;
