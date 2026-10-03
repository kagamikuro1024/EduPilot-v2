import { mkdirSync, rmSync, statSync } from "node:fs";
import { tmpdir } from "node:os";
import { join } from "node:path";

// Máy chủ giả :3312 là MỘT tiến trình dùng chung và có trạng thái (kịch bản, nhật ký) ⇒ hai tệp spec chạy song song sẽ giẫm nhau.
// Khoá liên tiến trình bằng thư mục (mkdir nguyên tử): mỗi tệp / ca dùng máy chủ giả giữ khoá tới khi xong.
const LOCK = join(tmpdir(), "edupilot-e2e-fake-api.lock");
const STALE_MS = 150_000;
const sleep = (ms: number) => new Promise((r) => setTimeout(r, ms));

export async function acquireFakeApi(): Promise<void> {
  const start = Date.now();
  for (;;) {
    try {
      mkdirSync(LOCK);
      return;
    } catch {
      try {
        if (Date.now() - statSync(LOCK).mtimeMs > STALE_MS) rmSync(LOCK, { recursive: true, force: true }); // khoá mồ côi do tiến trình chết
      } catch { /* vừa được nhả */ }
      if (Date.now() - start > 300_000) throw new Error("Không lấy được khoá máy chủ giả sau 5 phút");
      await sleep(100);
    }
  }
}

export function releaseFakeApi(): void {
  rmSync(LOCK, { recursive: true, force: true });
}
