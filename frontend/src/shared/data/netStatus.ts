// Trạng thái mạng cho <OfflineBanner>: hai lỗi NETWORK liên tiếp ⇒ mất mạng; một request thành công ⇒ có mạng (SRS 6.3).
let fails = 0;
let offline = typeof navigator !== "undefined" ? !navigator.onLine : false;
let forced = false; // do 2 lỗi NETWORK liên tiếp
const listeners = new Set<() => void>();
const emit = () => listeners.forEach((f) => f());

export const netStatus = {
  /** true khi `navigator.onLine === false` hoặc hai lỗi NETWORK liên tiếp. */
  isOffline: () => offline || forced,
  reportFailure() {
    fails += 1;
    if (fails >= 2 && !forced) {
      forced = true;
      emit();
    }
  },
  reportSuccess() {
    fails = 0;
    if (forced) {
      forced = false;
      emit();
    }
  },
  setBrowserOnline(online: boolean) {
    offline = !online;
    emit();
  },
  subscribe(f: () => void) {
    listeners.add(f);
    return () => {
      listeners.delete(f);
    };
  },
};

if (typeof window !== "undefined") {
  window.addEventListener("online", () => netStatus.setBrowserOnline(true));
  window.addEventListener("offline", () => netStatus.setBrowserOnline(false));
}
