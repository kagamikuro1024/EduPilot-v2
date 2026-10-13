"use client";

import { Button, Dialog } from "@/shared/ui";

/**
 * Hộp chặn thông tin cá nhân ở kênh công khai (SRS 4.3.3, nguyên tắc 4): chỉ nêu SỐ LƯỢNG theo loại, không in lại
 * giá trị tìm thấy. SV có hai lối (chat riêng / ẩn rồi đăng); GV, TA không có chat riêng nên là (quay lại sửa / ẩn rồi đăng).
 */
export function PIIChannelDialog({
  open,
  summary,
  onClose,
  onPrivateChat,
  onRedactedPost,
  redactDisabled,
  redactLabel = "Ẩn thông tin rồi đăng",
}: {
  open: boolean;
  /** "1 địa chỉ email, 1 số điện thoại" */
  summary: string;
  onClose: () => void;
  /** có = vai Sinh viên; không có = GV / TA */
  onPrivateChat?: () => void;
  onRedactedPost: () => void;
  /** lý do khoá lối "Ẩn rồi đăng" (câu hỏi riêng tư không có gì để ẩn) — hiện khi đưa chuột / focus */
  redactDisabled?: string;
  redactLabel?: string;
}) {
  return (
    <Dialog
      open={open}
      onClose={onClose}
      title="Bài này có thông tin cá nhân"
      description={`Threads là nơi cả lớp cùng đọc. Chúng tôi tìm thấy: ${summary}.`}
      footer={
        <>
          {onPrivateChat ? (
            <Button onClick={onPrivateChat}>Chuyển sang chat riêng</Button>
          ) : (
            <Button variant="ghost" onClick={onClose}>
              Quay lại sửa
            </Button>
          )}
          <Button variant="primary" onClick={onRedactedPost} disabled={Boolean(redactDisabled)} title={redactDisabled}>
            {redactLabel}
          </Button>
        </>
      }
    />
  );
}

/**
 * Hộp hỗ trợ khi bài có tín hiệu khó khăn (SRS escalation-notify 4.9.4): không lưu bài, không báo ai. Lối chính `Mở chat riêng` mang nháp theo
 * (chỉ vai Sinh viên có chat riêng), lối phụ `Quay lại sửa` giữ nguyên chữ trong ô soạn.
 */
export function CrisisSupportDialog({ open, message, onClose, onPrivateChat }: { open: boolean; message: string; onClose: () => void; onPrivateChat?: () => void }) {
  return (
    <Dialog
      open={open}
      onClose={onClose}
      title="Bạn không phải đi qua chuyện này một mình"
      description={message}
      footer={
        <>
          <Button variant={onPrivateChat ? "ghost" : "primary"} onClick={onClose}>
            Quay lại sửa
          </Button>
          {onPrivateChat && (
            <Button variant="primary" onClick={onPrivateChat}>
              Mở chat riêng
            </Button>
          )}
        </>
      }
    />
  );
}
