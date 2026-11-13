import type { Metadata } from "next";
import { PracticeHistory } from "@/features/practice/PracticeLazy";

export const metadata: Metadata = { title: "Lịch sử luyện tập" };

export default function Page() {
  return <PracticeHistory />;
}
