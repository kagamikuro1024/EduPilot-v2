"use client";

import { useEffect, useRef, useState } from "react";
import { Button } from "@/shared/ui";
import s from "./CheckinScreen.module.css";

type Detector = { detect: (v: HTMLVideoElement) => Promise<Array<{ rawValue: string }>> };
type DetectorCtor = new (o: { formats: string[] }) => Detector;

export const hasBarcodeDetector = () => typeof window !== "undefined" && "BarcodeDetector" in window;

/**
 * Quét QR chạy được thật? Có `BarcodeDetector` + `qr_code` chưa đủ: máy thiếu module GMS (emulator, một số máy Android) báo có nhưng `detect()` ném lỗi
 * («not operational»). Nên thử đọc MỘT mã QR mẫu vẽ tại chỗ: ném lỗi / không ra kết quả / quá 3 giây = không hỗ trợ, chỉ hiện ô nhập mã (US-AS-11 AC3a, DEF-R10.8-04).
 * Bản không có `getSupportedFormats` (cũ) vẫn được thử.
 */
export async function canScanQr(): Promise<boolean> {
  if (!hasBarcodeDetector()) return false;
  const C = (window as unknown as { BarcodeDetector: DetectorCtor & { getSupportedFormats?: () => Promise<string[]> } }).BarcodeDetector;
  try {
    const formats = await C.getSupportedFormats?.();
    if (Array.isArray(formats) && !formats.includes("qr_code")) return false;
    const det = new C({ formats: ["qr_code"] });
    const { encode } = await import("uqr");
    const { data } = encode("https://edupilot.invalid/checkin", { ecc: "L", border: 4 });
    const scale = 4;
    const cv = document.createElement("canvas");
    cv.width = cv.height = data.length * scale;
    const g = cv.getContext("2d");
    if (!g) return false;
    g.fillStyle = "white";
    g.fillRect(0, 0, cv.width, cv.height);
    g.fillStyle = "black";
    data.forEach((row, y) => row.forEach((on, x) => on && g.fillRect(x * scale, y * scale, scale, scale)));
    const found = await Promise.race([det.detect(cv as unknown as HTMLVideoElement), new Promise<never[]>((r) => setTimeout(() => r([]), 3000))]);
    return found.length > 0;
  } catch {
    return false;
  }
}

/** Hợp lệ khi: cùng nguồn gốc với ứng dụng, đường dẫn `/checkin`, `c` là UUID, có fragment (DESIGN §14.8.2). Mã khác bị bỏ qua im lặng. */
export function parseCheckinQr(raw: string): { c: string; token: string } | null {
  try {
    const u = new URL(raw);
    const c = u.searchParams.get("c") ?? "";
    const token = u.hash.slice(1);
    if (u.origin !== location.origin || u.pathname !== "/checkin" || !token) return null;
    if (!/^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$/i.test(c)) return null;
    return { c, token };
  } catch {
    return null;
  }
}

/** Camera sau + `BarcodeDetector` ~5 khung/giây. `track.stop()` khi nhận mã, `Thoát`, rời route, ẩn tab. */
export function Scanner({ onCode, onExit, onDenied }: { onCode: (v: { c: string; token: string }) => void; onExit: () => void; onDenied: (why: "denied" | "failed") => void }) {
  const video = useRef<HTMLVideoElement>(null);
  const [ready, setReady] = useState(false);
  const cb = useRef({ onCode, onExit, onDenied });
  useEffect(() => {
    cb.current = { onCode, onExit, onDenied };
  });

  useEffect(() => {
    let stream: MediaStream | null = null;
    let timer: ReturnType<typeof setInterval> | null = null;
    let watchdog: ReturnType<typeof setInterval> | null = null;
    let alive = true;
    const stop = () => {
      alive = false;
      if (timer) clearInterval(timer);
      if (watchdog) clearInterval(watchdog);
      stream?.getTracks().forEach((t) => t.stop());
      stream = null;
    };
    const onHide = () => {
      if (document.visibilityState === "hidden") {
        stop();
        cb.current.onExit();
      }
    };
    document.addEventListener("visibilitychange", onHide);
    void (async () => {
      try {
        stream = await navigator.mediaDevices.getUserMedia({ video: { facingMode: "environment" } });
        if (!alive) return stop();
        const el = video.current;
        if (!el) return stop();
        el.srcObject = stream;
        // DEF-MOB-11 (PRM-09): Android 12+ tắt công tắc Camera toàn máy ⇒ getUserMedia vẫn resolve nhưng track không có khung
        // (videoWidth 0 / muted / ended), khung quét đen mà màn coi là đang chạy. ~3 s không có khung ⇒ camera hỏng: dừng, về ô nhập mã.
        // Đặt TRƯỚC `play()`: stream không khung có thể làm `play()` không bao giờ resolve.
        const t0 = Date.now();
        watchdog = setInterval(() => {
          if (!alive) return;
          const live = (stream?.getVideoTracks() ?? []).some((t) => t.readyState === "live" && !t.muted);
          if (el.videoWidth > 0 && live) {
            clearInterval(watchdog!);
            watchdog = null;
            return;
          }
          if (Date.now() - t0 < 3000) return;
          stop();
          cb.current.onDenied("failed");
        }, 250);
        await el.play();
        setReady(true);
        const det = new (window as unknown as { BarcodeDetector: DetectorCtor }).BarcodeDetector({ formats: ["qr_code"] });
        timer = setInterval(async () => {
          try {
            for (const r of await det.detect(el)) {
              const v = parseCheckinQr(r.rawValue);
              if (v) {
                stop();
                cb.current.onCode(v);
                return;
              }
            }
          } catch {
            /* khung không đọc được: thử khung sau */
          }
        }, 200);
      } catch (e) {
        const wasAlive = alive; // đã thoát / rời màn lúc đang khởi động (Back nhanh): `play()` bị huỷ không phải lỗi camera
        stop();
        if (!wasAlive) return;
        // quyền bị từ chối ≠ camera hỏng / đang bị app khác dùng (US-AS-15 AC14): hai câu khác nhau, không bao giờ lộ tên lỗi
        const n = (e as { name?: string } | null)?.name;
        cb.current.onDenied(n === "NotAllowedError" || n === "SecurityError" || n === "PermissionDeniedError" ? "denied" : "failed");
      }
    })();
    return () => {
      document.removeEventListener("visibilitychange", onHide);
      stop();
    };
  }, []);

  return (
    <div className={s.scanner} data-part="scanner">
      <video ref={video} className={s.video} playsInline muted aria-label="Camera quét mã điểm danh" data-ready={ready} />
      <Button size="sm" onClick={onExit}>Thoát</Button>
    </div>
  );
}
