import type { Metadata } from "next";
import { QuestionBank } from "@/features/questions/QuestionBank";

export const metadata: Metadata = { title: "Ngân hàng câu hỏi" };

export default function Page() {
  return <QuestionBank />;
}
