// Token JWT CHỈ nằm trong bộ nhớ: không localStorage / sessionStorage / cookie / URL (AGENTS.md, SRS 5.3). Mất khi tải lại trang.
let token: string | null = null;
const listeners = new Set<() => void>();

export const tokenStore = {
  get: () => token,
  set(next: string | null) {
    token = next && next.trim() ? next.trim() : null;
    listeners.forEach((f) => f());
  },
  clear() {
    this.set(null);
  },
  subscribe(f: () => void) {
    listeners.add(f);
    return () => {
      listeners.delete(f);
    };
  },
};
