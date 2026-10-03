import type { Metadata } from "next";
import { TodayScreen } from "@/features/today/TodayScreen";

export const metadata: Metadata = { title: "Hôm nay" };

// "Hôm nay" thật: vai lấy từ phiên đăng nhập, dữ liệu từ /me/today và /courses/{id}/today.
export default function Page() {
  return <TodayScreen />;
}
