"use client";

import { useEffect, useRef, useState } from "react";
import { Button } from "@/shared/ui";
import s from "./CheckinScreen.module.css";

type Detector = { detect: (v: HTMLVideoElement) => Promise<Array<{ rawValue: string }>> };
type DetectorCtor = new (o: { formats: string[] }) => Detector;

export const hasBarcodeDetector = () => typeof window !== "undefined" && "BarcodeDetector" in window;

/**
 * Quét QR chạy được thật? `BarcodeDetector` có mà không đọc `qr_code` (danh sách định dạng rỗng / thiếu, hàm dựng ném lỗi) thì không hiện nút quét:
 * bấm vào chỉ rơi sang «từ chối camera» (DEF-R10.8-04). Bản không có `getSupportedFormats` (cũ) coi là có hỗ trợ.
 */
export async function canScanQr(): Promise<boolean> {
  if (!hasBarcodeDetector()) return false;
  const C = (window as unknown as { BarcodeDetector: DetectorCtor & { getSupportedFormats?: () => Promise<string[]> } }).BarcodeDetector;
  try {
    const formats = await C.getSupportedFormats?.();
    if (Array.isArray(formats) && !formats.includes("qr_code")) return false;
    new C({ formats: ["qr_code"] });
    return true;
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
export function Scanner({ onCode, onExit, onDenied }: { onCode: (v: { c: string; token: string }) => void; onExit: () => void; onDenied: () => void }) {
  const video = useRef<HTMLVideoElement>(null);
  const [ready, setReady] = useState(false);
  const cb = useRef({ onCode, onExit, onDenied });
  useEffect(() => {
    cb.current = { onCode, onExit, onDenied };
  });

  useEffect(() => {
    let stream: MediaStream | null = null;
    let timer: ReturnType<typeof setInterval> | null = null;
    let alive = true;
    const stop = () => {
      alive = false;
      if (timer) clearInterval(timer);
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
      } catch {
        stop();
        cb.current.onDenied();
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
