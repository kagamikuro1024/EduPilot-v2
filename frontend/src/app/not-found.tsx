import type { Metadata } from "next";
import { ButtonLink, Page, PageHeader } from "@/shared/ui";

// FR-X16 (00-AC13): mọi đường dẫn không có thật dừng ở màn tiếng Việt này, không bao giờ là màn mặc định của Next.js.
export const metadata: Metadata = { title: "Không tìm thấy trang" };

export default function NotFound() {
  return (
    <Page>
      <PageHeader
        title="Không tìm thấy trang"
        // ui-allow: trạng thái 404 nêu lý do + việc kế tiếp (DESIGN §15), không phải câu giải thích tính năng
        description="Đường dẫn bạn mở không tồn tại hoặc đã đổi. Kiểm tra lại liên kết hoặc quay về màn Hôm nay."
        actions={
          <ButtonLink href="/" variant="primary">
            Về Hôm nay
          </ButtonLink>
        }
      />
    </Page>
  );
}
