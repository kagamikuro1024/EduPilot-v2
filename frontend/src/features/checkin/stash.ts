// `#token` của liên kết QR không tới được máy chủ và sẽ MẤT khi AuthGate chuyển tới `/login?next=` (chỉ giữ đường dẫn + query).
// Layout của `/checkin` cất `{c, token, at}` vào sessionStorage (KHÔNG localStorage) rồi xoá fragment khỏi thanh địa chỉ; trang chỉ đọc từ đây và xoá khi dùng (US-AS-09 AC1, AC2).

const KEY = "ep.checkin";
const MAX_AGE_MS = 60_000;

export type Stash = { c: string; token: string; at: number };

export function stashFromLocation(): void {
  const token = location.hash.slice(1);
  if (!token) return;
  try {
    sessionStorage.setItem(KEY, JSON.stringify({ c: new URLSearchParams(location.search).get("c") ?? "", token, at: Date.now() } satisfies Stash));
  } catch {
    /* kho bị chặn: không có token, màn rơi về ô nhập mã */
  }
  history.replaceState(history.state, "", location.pathname + location.search);
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
