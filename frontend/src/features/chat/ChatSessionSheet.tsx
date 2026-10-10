"use client";

import type { ReactNode } from "react";
import { Button, Dialog } from "@/shared/ui";

/** Danh sách phiên ở < 1100 px (Dialog neo đáy ở ≤ 719 px); `Phiên mới` ở chân. */
export function ChatSessionSheet({ open, onClose, onNew, children }: { open: boolean; onClose: () => void; onNew: () => void; children: ReactNode }) {
  return (
    <Dialog open={open} onClose={onClose} title="Phiên trước" footer={<Button variant="primary" onClick={onNew}>Phiên mới</Button>}>
      {children}
    </Dialog>
  );
}
