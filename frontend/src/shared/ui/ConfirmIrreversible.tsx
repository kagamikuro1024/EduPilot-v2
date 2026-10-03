"use client";

import type { ReactNode } from "react";
import { Button } from "./Button";
import { Dialog } from "./Dialog";

/**
 * Hộp xác nhận cho việc không đảo ngược được (DESIGN §10.11): nêu hậu quả bằng số, hai nút.
 * Đây là primitive duy nhất cho các xác nhận như `Tạo lại mã`, `Lưu trữ` lớp, `Nộp bài`, `Chốt điểm`…
 */
export function ConfirmIrreversible({
  open,
  onClose,
  onConfirm,
  title,
  consequence,
  confirmLabel,
  cancelLabel = "Để sau",
  disabledReason,
}: {
  open: boolean;
  onClose: () => void;
  onConfirm: () => void;
  title: ReactNode;
  /** hậu quả cụ thể: "Chốt 30 sinh viên; 30 chưa có điểm cuối kỳ." */
  consequence: ReactNode;
  /** nút là động từ: "Chốt điểm", "Tạo lại mã" */
  confirmLabel: string;
  cancelLabel?: string;
  /** có giá trị → không cho xác nhận, hiện lý do */
  disabledReason?: ReactNode;
}) {
  return (
    <Dialog
      open={open}
      onClose={onClose}
      title={title}
      description={disabledReason ?? consequence}
      footer={
        <>
          <Button variant="ghost" onClick={onClose}>
            {cancelLabel}
          </Button>
          <Button
            variant="primary"
            disabled={Boolean(disabledReason)}
            onClick={() => {
              onConfirm();
              onClose();
            }}
          >
            {confirmLabel}
          </Button>
        </>
      }
    />
  );
}
