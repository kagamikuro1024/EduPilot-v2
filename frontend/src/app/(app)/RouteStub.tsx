import { Page, PageHeader } from "@/shared/ui";

/** Trang tạm cho route chưa dựng nội dung (US-PROTO-01…04 thay thế). Giữ điều hướng, chặn quyền, ⌘K chạy được. */
export function RouteStub({ title }: { title: string }) {
  return (
    <Page>
      <PageHeader title={title} description="Màn này đang được dựng trong bản mô phỏng." />
    </Page>
  );
}
