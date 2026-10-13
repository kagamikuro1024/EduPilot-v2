"use client";

import { WifiOff } from "lucide-react";
import { useSyncExternalStore } from "react";
import { subscribeUnsaved, unsavedCount } from "@/shared/lib/unsavedWork";
import { netStatus } from "./netStatus";
import s from "./OfflineBanner.module.css";

/** Dải một dòng khi mất mạng (`onLine=false` hoặc 2 lỗi NETWORK liên tiếp); đẩy nội dung xuống, không che, không chặn nhập liệu. */
export function OfflineBanner() {
  const off = useSyncExternalStore(netStatus.subscribe, netStatus.isOffline, () => false);
  const pending = useSyncExternalStore(subscribeUnsaved, unsavedCount, () => 0);
  return (
    <div className={s.slot} data-part="offline-banner">
      {off && (
        <p className={s.banner} role="status" aria-live="polite">
          <WifiOff aria-hidden />
          {pending > 0 ? `Chưa có mạng · ${pending} thay đổi chờ lưu` : "Mất kết nối mạng. Chữ bạn đã nhập vẫn được giữ; hệ thống sẽ gửi lại khi có mạng."}
        </p>
      )}
    </div>
  );
}
