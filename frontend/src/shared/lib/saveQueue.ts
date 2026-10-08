// Hàng đợi lưu câu trả lời phía máy khách (US-PE-05 AC10, SRS 4.3.6). Thuần logic: lưu trữ, bộ hẹn giờ và hàm gửi được truyền vào nên kiểm được không cần trình duyệt.
//
// Mỗi thay đổi ghi NGAY vào kho (localStorage khoá `exam:<attempt_id>`) rồi mới hẹn gửi: tải lại trang, sập trình duyệt hay mất mạng đều không mất chữ. Gửi debounce 2 s và khi mạng
// trở lại / tab hiện lại; thành công xoá phần đã gửi (chỉ khi câu đó không đổi trong lúc gửi); thất bại giữ nguyên và thử lại sau 1 → 2 → 4 → 8 → 15 s.

export type Answer = { option_ids: string[] } | { value: boolean };
export type Dirty = Record<string, Answer>;

export type Store = { read(): string | null; write(v: string): void; remove(): void };
export type Timers = { set(fn: () => void, ms: number): unknown; clear(h: unknown): void };
/** Gửi một lô; trả `ok` khi máy chủ nhận. Lỗi KHÔNG thử lại được (vd. 409 hết hạn / người ghi khác) báo qua `fatal` để hàng đợi dừng và giữ nguyên chữ. */
export type Sender = (items: Array<{ item_id: string; answer: Answer }>) => Promise<{ ok: true } | { ok: false; fatal?: boolean }>;

export type QueueState = { status: "saved" | "pending" | "offline" | "stopped"; pending: number; savedAt: number | null; retryInMs: number | null };

export const DEBOUNCE_MS = 2000;
export const BACKOFF_MS = [1000, 2000, 4000, 8000, 15000] as const;
const MAX_BYTES = 64 * 1024;

/** `savedAt` giữ cho bộ dọn nháp cũ của `useAutosaveDraft` (xoá khoá `ep:draft:*` thiếu `savedAt` hoặc quá 30 ngày) không xoá nhầm hàng đợi. */
type Persisted = { rev_local: number; dirty: Dirty; saved_at: number | null; savedAt: number };

export class SaveQueue {
  private dirty: Dirty = {};
  private rev = 0;
  private savedAt: number | null = null;
  private timer: unknown = null;
  private inflight = false;
  private fails = 0;
  private stopped = false;
  private online = true;
  private listeners = new Set<(s: QueueState) => void>();

  constructor(
    private store: Store,
    private send: Sender,
    private timers: Timers,
    private now: () => number,
  ) {
    try {
      const raw = store.read();
      if (raw) {
        const p = JSON.parse(raw) as Persisted;
        this.dirty = p.dirty ?? {};
        this.rev = p.rev_local ?? 0;
        this.savedAt = p.saved_at ?? null;
      }
    } catch {
      /* kho hỏng: bắt đầu trống */
    }
  }

  /** Đổi hàm gửi (đóng gói ngữ cảnh mới của màn hình: tab, lượt…) mà không tạo lại hàng đợi. */
  setSender(send: Sender) {
    this.send = send;
  }

  /** Phần chưa lên máy chủ (đã đọc lại từ kho khi khởi tạo): áp lên đề đã tải để giữ chữ của người làm bài. */
  pending(): Dirty {
    return { ...this.dirty };
  }

  get state(): QueueState {
    const n = Object.keys(this.dirty).length;
    const status = this.stopped ? "stopped" : n === 0 ? "saved" : !this.online ? "offline" : "pending";
    return { status, pending: n, savedAt: this.savedAt, retryInMs: this.fails > 0 ? BACKOFF_MS[Math.min(this.fails - 1, BACKOFF_MS.length - 1)] : null };
  }

  subscribe(fn: (s: QueueState) => void): () => void {
    this.listeners.add(fn);
    return () => this.listeners.delete(fn);
  }

  private emit() {
    const s = this.state;
    for (const l of this.listeners) l(s);
  }

  private persist() {
    const body = JSON.stringify({ rev_local: this.rev, dirty: this.dirty, saved_at: this.savedAt, savedAt: this.now() } satisfies Persisted);
    if (body.length <= MAX_BYTES) this.store.write(body);
    // vượt 64 KiB: giữ trong bộ nhớ (vẫn gửi được); mỗi câu tối đa 4 KiB nên bài thật không bao giờ tới mức này
  }

  /** Ghi một thay đổi: vào kho ngay, hẹn gửi sau DEBOUNCE_MS. */
  set(itemId: string, answer: Answer) {
    if (this.stopped) return;
    this.dirty[itemId] = answer;
    this.rev++;
    this.persist();
    this.schedule(DEBOUNCE_MS);
    this.emit();
  }

  private schedule(ms: number) {
    if (this.timer !== null) this.timers.clear(this.timer);
    this.timer = this.timers.set(() => {
      this.timer = null;
      void this.flush();
    }, ms);
  }

  /** Mạng trở lại / tab hiện lại / bấm thử lại: gửi ngay, đặt lại nhịp thử. */
  resume() {
    this.online = true;
    this.fails = 0;
    if (Object.keys(this.dirty).length > 0) this.schedule(0);
    this.emit();
  }

  setOnline(on: boolean) {
    this.online = on;
    if (on) this.resume();
    else this.emit();
  }

  /** Dừng hẳn (không còn là người ghi / đã nộp): giữ nguyên chữ ở kho, không gửi nữa. */
  stop() {
    this.stopped = true;
    if (this.timer !== null) this.timers.clear(this.timer);
    this.timer = null;
    this.emit();
  }

  /** Làm tiếp sau khi `stop()` (giành lại quyền ghi): gửi lại phần còn giữ ở máy. */
  restart() {
    this.stopped = false;
    this.fails = 0;
    if (Object.keys(this.dirty).length > 0) this.schedule(0);
    this.emit();
  }

  /** Sau khi nộp: xoá khoá của hàng đợi. */
  clear() {
    this.dirty = {};
    this.store.remove();
    this.emit();
  }

  async flush(): Promise<void> {
    if (this.inflight || this.stopped) return;
    const keys = Object.keys(this.dirty);
    if (keys.length === 0) return;
    this.inflight = true;
    const sent: Dirty = { ...this.dirty };
    let res: Awaited<ReturnType<Sender>>;
    try {
      res = await this.send(keys.map((k) => ({ item_id: k, answer: sent[k] })));
    } catch {
      res = { ok: false };
    }
    this.inflight = false;
    if (res.ok) {
      for (const k of keys) if (this.dirty[k] === sent[k]) delete this.dirty[k]; // câu đổi trong lúc gửi thì giữ lại
      this.fails = 0;
      this.savedAt = this.now();
      this.persist();
      if (Object.keys(this.dirty).length > 0) this.schedule(DEBOUNCE_MS);
    } else if (res.fatal) {
      this.stop();
      return;
    } else {
      this.fails++;
      this.schedule(BACKOFF_MS[Math.min(this.fails - 1, BACKOFF_MS.length - 1)]);
    }
    this.emit();
  }
}
