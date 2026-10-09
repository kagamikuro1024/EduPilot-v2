// Đồng hồ làm bài (US-PE-05 AC3). Máy chủ là nguồn duy nhất của hạn nộp: trình duyệt chỉ HIỂN THỊ thời gian còn lại, không quyết định gì.
// offset = server_time − (t_gửi + t_nhận) / 2, đo ở MỖI phản hồi lưu / tải (bù nửa độ trễ mạng); hiển thị còn lại = deadline − (Date.now() + offset).

/** Mỗi lần cập nhật offset không đổi quá chừng này (ms): đếm ngược không nhảy ngược khi mạng đo lệch một nhịp. Lần đo ĐẦU nhận nguyên giá trị (máy lệch ±5 phút phải đúng ngay). */
export const MAX_OFFSET_STEP_MS = 1000;

export type ExamClock = {
  /** thời điểm máy chủ hiện tại (ms epoch) theo đồng hồ máy khách `clientNowMs` */
  now(clientNowMs: number): number;
  /** nhận một phản hồi: `serverTimeMs` là `server_time` của thân, `sentMs` / `receivedMs` là Date.now() lúc gửi / nhận */
  observe(serverTimeMs: number, sentMs: number, receivedMs: number): void;
  readonly offset: number | null;
};

export function makeExamClock(): ExamClock {
  let offset: number | null = null;
  return {
    get offset() {
      return offset;
    },
    now: (clientNowMs) => clientNowMs + (offset ?? 0),
    observe(serverTimeMs, sentMs, receivedMs) {
      const measured = serverTimeMs - (sentMs + receivedMs) / 2;
      if (offset === null) offset = measured;
      else offset += Math.max(-MAX_OFFSET_STEP_MS, Math.min(MAX_OFFSET_STEP_MS, measured - offset));
    },
  };
}

/** Thời gian còn lại (ms, không âm) tới `deadlineMs` (theo đồng hồ máy chủ). */
export function remainingMs(deadlineMs: number, clock: ExamClock, clientNowMs: number): number {
  return Math.max(0, deadlineMs - clock.now(clientNowMs));
}

const p2 = (n: number) => String(n).padStart(2, "0");

/** `mm:ss`; từ 1 giờ trở lên `h:mm:ss`. Làm tròn LÊN giây để không hiện 00:00 khi còn mili giây. */
export function formatRemaining(ms: number): string {
  const total = Math.ceil(Math.max(0, ms) / 1000);
  const h = Math.floor(total / 3600);
  const m = Math.floor((total % 3600) / 60);
  const s = total % 60;
  return h > 0 ? `${h}:${p2(m)}:${p2(s)}` : `${p2(m)}:${p2(s)}`;
}

/** Mốc cảnh báo: còn ≤ 1 phút → 1, ≤ 5 phút → 5, ngược lại null. */
export function warnLevel(ms: number): 1 | 5 | null {
  if (ms <= 0) return null;
  if (ms <= 60_000) return 1;
  if (ms <= 300_000) return 5;
  return null;
}
