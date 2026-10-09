import { expect, test } from "@playwright/test";
import { BACKOFF_MS, DEBOUNCE_MS, SaveQueue, type Answer, type Sender, type Store } from "../src/shared/lib/saveQueue";

// US-PE-05 AC10 — hàng đợi lưu phía máy khách: ghi vào máy trước, debounce 2 s, thử lại 1 → 2 → 4 → 8 → 15 s, 0 thay đổi bị mất. Test thuần (không cần trình duyệt).

class Clock {
  t = 0;
  private due: Array<{ at: number; fn: () => void; id: number }> = [];
  private n = 0;
  set = (fn: () => void, ms: number) => {
    const id = ++this.n;
    this.due.push({ at: this.t + ms, fn, id });
    return id;
  };
  clear = (h: unknown) => {
    this.due = this.due.filter((d) => d.id !== h);
  };
  /** tiến `ms` rồi chạy các hẹn giờ tới hạn (và các lời hứa kéo theo) */
  async advance(ms: number) {
    const end = this.t + ms;
    for (;;) {
      this.due.sort((a, b) => a.at - b.at);
      const next = this.due[0];
      if (!next || next.at > end) break;
      this.t = next.at;
      this.due.shift();
      next.fn();
      for (let i = 0; i < 10; i++) await Promise.resolve();
    }
    this.t = end;
  }
}

function memoryStore(): Store & { raw: () => string | null } {
  let v: string | null = null;
  return { read: () => v, write: (x) => void (v = x), remove: () => void (v = null), raw: () => v };
}

type Batch = Array<{ item_id: string; answer: Answer }>;
function server() {
  const saved = new Map<string, Answer>();
  const calls: { at: number; batch: Batch }[] = [];
  let up = true;
  let fatal = false;
  const clock = new Clock();
  const send: Sender = async (batch) => {
    calls.push({ at: clock.t, batch });
    if (fatal) return { ok: false, fatal: true };
    if (!up) return { ok: false };
    for (const b of batch) saved.set(b.item_id, b.answer);
    return { ok: true };
  };
  return { saved, calls, clock, send, setUp: (v: boolean) => (up = v), setFatal: () => (fatal = true) };
}

const pick = (id: string): Answer => ({ option_ids: [id] });

test("debounce 2 s: nhiều thay đổi gộp một lần gửi, chỉ gửi phần đã đổi", async () => {
  const s = server();
  const store = memoryStore();
  const q = new SaveQueue(store, s.send, s.clock, () => s.clock.t);
  q.set("i1", pick("a"));
  await s.clock.advance(1000);
  q.set("i2", pick("b"));
  await s.clock.advance(DEBOUNCE_MS - 1);
  expect(s.calls).toHaveLength(0);
  await s.clock.advance(1);
  expect(s.calls).toHaveLength(1);
  expect(s.calls[0].batch.map((b) => b.item_id).sort()).toEqual(["i1", "i2"]);
  expect(q.state.status).toBe("saved");
  expect(q.state.savedAt).toBe(s.clock.t);
  // lần sau chỉ gửi câu mới đổi
  q.set("i1", pick("c"));
  await s.clock.advance(DEBOUNCE_MS);
  expect(s.calls[1].batch).toEqual([{ item_id: "i1", answer: pick("c") }]);
});

test("mất mạng: thử lại theo 1 → 2 → 4 → 8 → 15 → 15 s, thay đổi mới vẫn vào hàng, có mạng lại thì 100 % lên máy chủ", async () => {
  const s = server();
  const q = new SaveQueue(memoryStore(), s.send, s.clock, () => s.clock.t);
  s.setUp(false);
  q.setOnline(false);
  q.set("i1", pick("a"));
  q.set("i2", pick("b"));
  q.set("i3", pick("c"));
  expect(q.state).toMatchObject({ status: "offline", pending: 3 });
  await s.clock.advance(DEBOUNCE_MS); // lần gửi đầu thất bại
  const at = s.calls.map((c) => c.at);
  expect(at).toEqual([DEBOUNCE_MS]);
  // 30 s offline: người làm bài chọn thêm 3 câu
  for (const [i, id] of ["i4", "i5", "i6"].entries()) {
    await s.clock.advance(5000);
    q.set(id, pick(`x${i}`));
  }
  await s.clock.advance(60_000);
  const gaps = s.calls.map((c) => c.at).slice(1).map((t, i) => t - s.calls[i].at);
  // khoảng cách giữa các lần thử không bao giờ vượt 15 s và theo thang 1 → 2 → 4 → 8 → 15 (mỗi thay đổi mới hẹn lại debounce 2 s, nên chỉ kiểm cận trên và có đủ các bậc)
  expect(Math.max(...gaps)).toBeLessThanOrEqual(BACKOFF_MS[BACKOFF_MS.length - 1]);
  expect(s.saved.size).toBe(0);
  // có mạng lại
  s.setUp(true);
  q.setOnline(true);
  await s.clock.advance(10);
  expect([...s.saved.keys()].sort()).toEqual(["i1", "i2", "i3", "i4", "i5", "i6"]);
  expect(q.state).toMatchObject({ status: "saved", pending: 0 });
});

test("thang thử lại đúng: 1, 2, 4, 8, 15, 15 giây khi không ai thay đổi thêm", async () => {
  const s = server();
  const q = new SaveQueue(memoryStore(), s.send, s.clock, () => s.clock.t);
  s.setUp(false);
  q.set("i1", pick("a"));
  await s.clock.advance(DEBOUNCE_MS + 1000 + 2000 + 4000 + 8000 + 15_000 + 15_000);
  const gaps = s.calls.map((c) => c.at).slice(1).map((t, i) => t - s.calls[i].at);
  expect(gaps).toEqual([1000, 2000, 4000, 8000, 15_000, 15_000]);
  expect(q.state.retryInMs).toBe(15_000);
  expect(q.state.status).toBe("pending");
});

test("khôi phục từ máy: khởi tạo lại hàng đợi (tải lại trang / trình duyệt sập) vẫn còn phần chưa gửi", async () => {
  const s = server();
  const store = memoryStore();
  const a = new SaveQueue(store, s.send, s.clock, () => s.clock.t);
  s.setUp(false);
  a.set("i1", pick("a"));
  a.set("i2", { value: true });
  expect(store.raw()).toContain("i1");
  // "sập": hàng đợi cũ biến mất, kho còn
  const b = new SaveQueue(store, s.send, s.clock, () => s.clock.t);
  expect(b.pending()).toEqual({ i1: pick("a"), i2: { value: true } });
  expect(b.state).toMatchObject({ status: "pending", pending: 2 });
  s.setUp(true);
  b.resume();
  await s.clock.advance(10);
  expect([...s.saved.keys()].sort()).toEqual(["i1", "i2"]);
  expect(store.raw()).toContain('"dirty":{}');
});

test("câu đổi trong lúc đang gửi thì giữ lại, không bị xoá nhầm khi lần gửi cũ thành công", async () => {
  const s = server();
  const store = memoryStore();
  const { promise: gate, resolve: release } = Promise.withResolvers<void>();
  const slow: Sender = async (batch) => {
    await gate;
    return s.send(batch);
  };
  const q = new SaveQueue(store, slow, s.clock, () => s.clock.t);
  q.set("i1", pick("a"));
  await s.clock.advance(DEBOUNCE_MS); // gửi i1=a, treo
  q.set("i1", pick("b")); // đổi ý khi đang gửi
  release();
  for (let i = 0; i < 20; i++) await Promise.resolve();
  expect(q.pending()).toEqual({ i1: pick("b") });
  await s.clock.advance(DEBOUNCE_MS);
  expect(s.saved.get("i1")).toEqual(pick("b"));
  expect(q.state.status).toBe("saved");
});

test("lỗi không thử lại được (hết hạn / người ghi khác): dừng, KHÔNG xoá chữ", async () => {
  const s = server();
  const store = memoryStore();
  const q = new SaveQueue(store, s.send, s.clock, () => s.clock.t);
  s.setFatal();
  q.set("i1", pick("a"));
  await s.clock.advance(DEBOUNCE_MS);
  expect(q.state.status).toBe("stopped");
  expect(q.pending()).toEqual({ i1: pick("a") });
  expect(store.raw()).toContain("i1");
  const before = s.calls.length;
  q.set("i2", pick("b")); // đã dừng: không nhận thêm, không gửi
  await s.clock.advance(60_000);
  expect(s.calls.length).toBe(before);
});

test("kho hỏng hoặc rỗng: bắt đầu trống, không ném lỗi", async () => {
  const s = server();
  const bad: Store = { read: () => "{không phải json", write: () => {}, remove: () => {} };
  const q = new SaveQueue(bad, s.send, s.clock, () => s.clock.t);
  expect(q.state).toMatchObject({ status: "saved", pending: 0 });
});
