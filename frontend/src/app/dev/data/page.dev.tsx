import { Suspense } from "react";
import DevData from "./DevData";

// Chỉ có ở `next dev` hoặc bản dựng cổng (xem pageExtensions ở next.config.ts).
export const metadata = { title: "Thử lớp dữ liệu" };

export default function Page() {
  return (
    <Suspense>
      <DevData />
    </Suspense>
  );
}
