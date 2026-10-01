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
}: {
  open: boolean;
  /** "1 địa chỉ email, 1 số điện thoại" */
  summary: string;
  onClose: () => void;
  /** có = vai Sinh viên; không có = GV / TA */
  onPrivateChat?: () => void;
  onRedactedPost: () => void;
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
          <Button variant="primary" onClick={onRedactedPost}>
            Ẩn thông tin rồi đăng
          </Button>
        </>
      }
    />
  );
}
