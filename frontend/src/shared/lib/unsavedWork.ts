// Việc chưa lên máy chủ của màn đang mở (điểm danh…): để "Đăng xuất" hỏi trước khi bỏ, và dọn kho nháp của người đó trên máy dùng chung (FEAT-crm-360 AC13).
const sources = new Map<string, () => number>();
const listeners = new Set<() => void>();

/** Nguồn báo số đếm đã đổi (OfflineBanner nghe để nêu "N thay đổi chờ lưu"). */
export function notifyUnsaved() {
  listeners.forEach((f) => f());
}
export function subscribeUnsaved(f: () => void): () => void {
  listeners.add(f);
  return () => {
    listeners.delete(f);
  };
}

/** Đăng ký nguồn đếm thay đổi chờ lưu; trả hàm huỷ đăng ký. */
export function registerUnsaved(id: string, count: () => number): () => void {
  sources.set(id, count);
  notifyUnsaved();
  return () => {
    if (sources.get(id) === count) sources.delete(id);
    notifyUnsaved();
  };
}

export function unsavedCount(): number {
  let n = 0;
  for (const c of sources.values()) n += c();
  return n;
}

/** Xoá mọi khoá nháp `ep:draft:<userId>:<area>:…` của người này (đăng xuất ở máy dùng chung). */
export function wipeDrafts(userId: string, area: string) {
  if (typeof localStorage === "undefined") return;
  const prefix = `ep:draft:${userId}:${area}:`;
  try {
    for (let i = localStorage.length - 1; i >= 0; i--) {
      const k = localStorage.key(i);
      if (k?.startsWith(prefix)) localStorage.removeItem(k);
    }
  } catch {
    /* kho không truy cập được */
  }
}
