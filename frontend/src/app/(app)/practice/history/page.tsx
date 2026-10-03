import type { Metadata } from "next";
import { PracticeHistory } from "@/features/practice/PracticeHistory";

export const metadata: Metadata = { title: "Lịch sử luyện tập" };

export default function Page() {
  return <PracticeHistory />;
}
