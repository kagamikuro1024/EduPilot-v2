import { expect, test } from "@playwright/test";
import { formatRemaining, makeExamClock, MAX_OFFSET_STEP_MS, remainingMs, warnLevel } from "../src/shared/lib/examClock";

// US-PE-05 AC3 — đồng hồ làm bài: hiển thị = deadline − (Date.now() + offset), offset đo ở mỗi phản hồi. Test thuần (không cần trình duyệt).

const SERVER_NOW = Date.UTC(2026, 11, 1, 2, 0, 0);
const DEADLINE = SERVER_NOW + 45 * 60_000;

/** Một phản hồi: máy khách gửi lúc `sent` (đồng hồ máy khách), mất `rtt` ms; máy chủ đóng dấu `server_time` ở GIỮA hành trình. */
function sample(clientSkewMs: number, serverMs: number, rtt: number) {
  const sent = serverMs + clientSkewMs - rtt / 2;
  return { serverTime: serverMs, sent, received: sent + rtt };
}

for (const skew of [-5 * 60_000, 0, 5 * 60_000, 37_000]) {
  test(`đồng hồ lệch ${skew / 1000} s: thời gian còn lại sai ≤ 1 s sau lần đo đầu`, () => {
    const c = makeExamClock();
    const s = sample(skew, SERVER_NOW, 240);
    c.observe(s.serverTime, s.sent, s.received);
    for (const elapsed of [0, 1500, 61_000, 20 * 60_000]) {
      const clientNow = SERVER_NOW + skew + elapsed;
      const want = DEADLINE - (SERVER_NOW + elapsed);
      expect(Math.abs(remainingMs(DEADLINE, c, clientNow) - want), `sau ${elapsed} ms`).toBeLessThanOrEqual(1000);
    }
  });
}

test("đo lại ở mỗi phản hồi: offset đổi không quá 1 s mỗi lần → đếm ngược không nhảy ngược", () => {
  const c = makeExamClock();
  const skew = 5 * 60_000;
  let s = sample(skew, SERVER_NOW, 100);
  c.observe(s.serverTime, s.sent, s.received);
  const first = c.offset!;
  // một phản hồi đo lệch +5 s (mạng giật): chỉ được nhích MAX_OFFSET_STEP_MS
  s = sample(skew + 5000, SERVER_NOW + 10_000, 100);
  c.observe(s.serverTime, s.sent, s.received);
  expect(Math.abs(c.offset! - first)).toBeLessThanOrEqual(MAX_OFFSET_STEP_MS);
  // liên tiếp nhiều phản hồi cùng một độ lệch thật: hội tụ về đúng
  for (let i = 0; i < 8; i++) {
    s = sample(skew + 5000, SERVER_NOW + 20_000 + i * 10_000, 100);
    c.observe(s.serverTime, s.sent, s.received);
  }
  expect(Math.abs(c.offset! - (-(skew + 5000)))).toBeLessThanOrEqual(1000);
  // không bao giờ nhích ngược quá ngưỡng giữa hai lần đo liên tiếp
  const c2 = makeExamClock();
  let prev = 0;
  for (let i = 0; i < 20; i++) {
    const jitter = (i % 2 ? 1 : -1) * 700;
    s = sample(skew + jitter, SERVER_NOW + i * 5000, 120);
    c2.observe(s.serverTime, s.sent, s.received);
    if (i > 0) expect(Math.abs(c2.offset! - prev)).toBeLessThanOrEqual(MAX_OFFSET_STEP_MS);
    prev = c2.offset!;
  }
});

test("hết hạn do máy chủ: còn lại không âm, về 0 khi quá hạn", () => {
  const c = makeExamClock();
  const s = sample(0, SERVER_NOW, 50);
  c.observe(s.serverTime, s.sent, s.received);
  expect(remainingMs(DEADLINE, c, DEADLINE + 5000)).toBe(0);
  expect(remainingMs(DEADLINE, c, DEADLINE - 61_000)).toBeGreaterThan(59_000);
});

test("định dạng: mm:ss, h:mm:ss, làm tròn lên giây, chữ số cố định bề rộng", () => {
  expect(formatRemaining(45 * 60_000)).toBe("45:00");
  expect(formatRemaining(9_000)).toBe("00:09");
  expect(formatRemaining(200)).toBe("00:01"); // còn mili giây vẫn hiện 00:01, không 00:00
  expect(formatRemaining(0)).toBe("00:00");
  expect(formatRemaining(-5)).toBe("00:00");
  expect(formatRemaining(3_600_000)).toBe("1:00:00");
  expect(formatRemaining(3_725_000)).toBe("1:02:05");
  expect(formatRemaining(59 * 60_000 + 59_001)).toBe("1:00:00");
});

test("mốc cảnh báo: 5 phút và 1 phút (chỉ đổi ở hai mốc, để aria-live đọc đúng hai lần)", () => {
  expect(warnLevel(10 * 60_000)).toBeNull();
  expect(warnLevel(300_001)).toBeNull();
  expect(warnLevel(300_000)).toBe(5);
  expect(warnLevel(61_000)).toBe(5);
  expect(warnLevel(60_000)).toBe(1);
  expect(warnLevel(1)).toBe(1);
  expect(warnLevel(0)).toBeNull();
});
