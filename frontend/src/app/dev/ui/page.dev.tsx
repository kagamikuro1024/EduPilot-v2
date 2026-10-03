import { Suspense } from "react";
import DevUi from "./DevUi";

// Tệp `page.dev.tsx` chỉ được tính là trang khi `next dev` hoặc bản dựng đặt NEXT_PUBLIC_DEV_TOOLS=1 (xem pageExtensions ở
// next.config.ts). Bản "như production" không có route này nên mã trang không vào bundle và /dev/* trả 404 (US-PU-02 AC16).
export const metadata = { title: "Thư viện thành phần" };

export default function Page() {
  return (
    <Suspense>
      <DevUi />
    </Suspense>
  );
}
