"use client";

import { Button, InlineNotice } from "@/shared/ui";

/** 409 VERSION_CONFLICT: hỏi giữ bản nào, không ghi đè im lặng và không mất chữ đã nhập (US-P1-05 AC9). */
export function ConflictNotice({ onKeepMine, onUseTheirs, pending }: { onKeepMine: () => void; onUseTheirs: () => void; pending?: boolean }) {
  return (
    <InlineNotice tone="warning" title="Cài đặt này vừa được người khác đổi. Giữ bản của bạn hay dùng bản mới?">
      <span style={{ display: "inline-flex", gap: "var(--ep-space-2)", marginTop: "var(--ep-space-2)" }}>
        <Button size="sm" loading={pending} onClick={onKeepMine}>
          Giữ bản của tôi
        </Button>
        <Button size="sm" variant="ghost" onClick={onUseTheirs}>
          Dùng bản mới
        </Button>
      </span>
    </InlineNotice>
  );
}
