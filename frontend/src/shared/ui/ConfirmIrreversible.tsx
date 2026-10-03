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
  loading,
  error,
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
  /** truyền `loading`/`error` ⇒ người dùng tự đóng hộp khi xong; không truyền ⇒ đóng ngay sau khi xác nhận. Đang thực hiện: Esc / bấm nền / Đóng bị khoá, nút xác nhận hiện trạng thái chờ */
  loading?: boolean;
  /** thực hiện lỗi: hộp giữ nguyên, hiện lỗi (role=alert) và nút xác nhận đổi thành "Thử lại" */
  error?: ReactNode;
}) {
  return (
    <Dialog
      open={open}
      onClose={onClose}
      title={title}
      description={disabledReason ?? consequence}
      dismissible={!loading}
      error={error}
      footer={
        <>
          <Button variant="ghost" onClick={onClose} disabled={loading}>
            {cancelLabel}
          </Button>
          <Button
            variant="primary"
            disabled={Boolean(disabledReason)}
            loading={loading}
            onClick={() => {
              onConfirm();
              if (loading === undefined && error === undefined) onClose(); // không điều khiển trạng thái ⇒ đóng ngay như trước
            }}
          >
            {error ? "Thử lại" : confirmLabel}
          </Button>
        </>
      }
    />
  );
}
