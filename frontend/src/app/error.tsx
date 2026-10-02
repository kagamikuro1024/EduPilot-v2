"use client";

import { useEffect } from "react";
import { resetDemo } from "@/shared/state/demo";
import { Button, ButtonLink, Page, PageHeader } from "@/shared/ui";

const HEAL_KEY = "ep_auto_heal";

// FR-X16 (00-AC13): route ném lỗi thì dừng ở màn tiếng Việt này; `Thử lại` dựng lại đúng route vừa lỗi.
// Lỗi thường do dữ liệu giả lập trong localStorage hỏng: tự xoá khoá hỏng và chạy tiếp một lần (không lặp vô hạn);
// nếu vẫn lỗi, màn này có `Đặt lại dữ liệu demo` để thoát mà không cần DevTools (00-1).
export default function AppError({ reset }: { error: Error & { digest?: string }; reset: () => void }) {
  useEffect(() => {
    try {
      const last = Number(sessionStorage.getItem(HEAL_KEY) ?? 0);
      if (Date.now() - last < 10_000) return;
      sessionStorage.setItem(HEAL_KEY, String(Date.now()));
      resetDemo();
      reset();
    } catch {
      /* không có sessionStorage: chỉ hiện màn lỗi */
    }
  }, [reset]);

  return (
    <Page>
      <PageHeader
        title="Trang này gặp sự cố"
        description="Dữ liệu mô phỏng của trang không dựng được. Bạn thử lại, đặt lại dữ liệu demo, hoặc quay về màn Hôm nay rồi mở lại sau."
        actions={
          <>
            <Button variant="primary" onClick={reset}>
              Thử lại
            </Button>
            <Button
              variant="secondary"
              onClick={() => {
                resetDemo();
                reset();
              }}
            >
              Đặt lại dữ liệu demo
            </Button>
            <ButtonLink href="/">Về Hôm nay</ButtonLink>
          </>
        }
      />
    </Page>
  );
}
