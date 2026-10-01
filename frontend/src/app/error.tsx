"use client";

import { Button, ButtonLink, Page, PageHeader } from "@/shared/ui";

// FR-X16 (00-AC13): route ném lỗi thì dừng ở màn tiếng Việt này; `Thử lại` dựng lại đúng route vừa lỗi.
export default function AppError({ reset }: { error: Error & { digest?: string }; reset: () => void }) {
  return (
    <Page>
      <PageHeader
        title="Trang này gặp sự cố"
        description="Dữ liệu mô phỏng của trang không dựng được. Bạn thử lại, hoặc quay về màn Hôm nay rồi mở lại sau."
        actions={
          <>
            <Button variant="primary" onClick={reset}>
              Thử lại
            </Button>
            <ButtonLink href="/">Về Hôm nay</ButtonLink>
          </>
        }
      />
    </Page>
  );
}
