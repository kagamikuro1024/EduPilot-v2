"use client";

import { WifiOff } from "lucide-react";
import { useEffect, useRef, useSyncExternalStore } from "react";
import { subscribeUnsaved, unsavedCount } from "@/shared/lib/unsavedWork";
import { netStatus } from "./netStatus";
import s from "./OfflineBanner.module.css";

/** Dải một dòng khi mất mạng (`onLine=false` hoặc 2 lỗi NETWORK liên tiếp); đẩy nội dung xuống, không che, không chặn nhập liệu. */
export function OfflineBanner() {
  const off = useSyncExternalStore(netStatus.subscribe, netStatus.isOffline, () => false);
  const pending = useSyncExternalStore(subscribeUnsaved, unsavedCount, () => 0);
  const slot = useRef<HTMLDivElement>(null);
  // chiều cao dải → --ep-banner-h: khung app trừ đi, trang không cuộn thừa đúng bằng dải (US-AS-15 AC1)
  useEffect(() => {
    const el = slot.current;
    if (!el) return;
    const set = () => document.documentElement.style.setProperty("--ep-banner-h", `${el.offsetHeight}px`);
    set();
    const ro = new ResizeObserver(set);
    ro.observe(el);
    return () => {
      ro.disconnect();
      document.documentElement.style.removeProperty("--ep-banner-h");
    };
  }, []);
  return (
    <div className={s.slot} data-part="offline-banner" ref={slot}>
      {off && (
        <p className={s.banner} role="status" aria-live="polite">
          <WifiOff aria-hidden />
          {pending > 0 ? `Chưa có mạng · ${pending} thay đổi chờ lưu` : "Mất kết nối mạng."}
        </p>
      )}
    </div>
  );
}
