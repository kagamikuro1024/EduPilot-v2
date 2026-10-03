import type { Metadata } from "next";
import { GradingQueue } from "@/features/grading/GradingQueue";

export const metadata: Metadata = { title: "Chấm bài" };

export default function Page() {
  return <GradingQueue />;
}
