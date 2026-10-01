"use client";

import { redact } from "@/mock/chat";
import { Button, Dialog } from "@/shared/ui";
import s from "./Threads.module.css";

/**
 * Tường lửa thông tin cá nhân khi đăng bài công khai (DESIGN §14.3, INTEGRATION mục 2 #3):
 * đúng hai lối đi, không có lối "đăng nguyên văn".
 */
export function PIIChannelDialog({
  open,
  text,
  reasons,
  onClose,
  onPrivateChat,
  onRedactedPost,
}: {
  open: boolean;
  text: string;
  reasons: string[];
  onClose: () => void;
  onPrivateChat: () => void;
  onRedactedPost: (clean: string) => void;
}) {
  const clean = redact(text);
  return (
    <Dialog
      open={open}
      onClose={onClose}
      title="Bài này có thông tin cá nhân"
      description={`Bài viết chứa ${reasons.join(" và ")}. Threads là nơi cả lớp đọc được, nên bạn chọn một trong hai cách dưới đây.`}
      footer={
        <>
          <Button variant="ghost" onClick={onClose}>
            Quay lại sửa
          </Button>
          <Button onClick={() => onRedactedPost(clean)}>Ẩn thông tin rồi đăng</Button>
          <Button variant="primary" onClick={onPrivateChat}>
            Chuyển sang chat riêng
          </Button>
        </>
      }
    >
      <div className={s.dialogBody}>
        <p className={s.dialogLabel}>Chat riêng — chỉ bạn và trợ lý AI đọc được</p>
        <p className={s.dialogText}>{text}</p>
        <p className={s.dialogLabel}>Đăng công khai — bản đã ẩn thông tin</p>
        <p className={s.dialogText}>{clean}</p>
      </div>
    </Dialog>
  );
}
