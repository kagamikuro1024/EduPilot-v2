import type { Metadata } from "next";
import { Page, PageHeader } from "@/shared/ui";

// FR-X16 (00-AC13): mọi đường dẫn không có thật dừng ở màn tiếng Việt này, không bao giờ là màn mặc định của Next.js.
export const metadata: Metadata = { title: "Không tìm thấy trang" };

export default function NotFound() {
  return (
    <Page kind="read">
      <PageHeader title="Không tìm thấy trang" primary={{ label: "Về Hôm nay", href: "/" }} />
    </Page>
  );
}
