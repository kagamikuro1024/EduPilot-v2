import type { Metadata } from "next";
import { ExamTake } from "@/features/exam/take/ExamTake";

export const metadata: Metadata = { title: "Làm bài thi" };

export default async function Page({ params }: { params: Promise<{ id: string }> }) {
  const { id } = await params;
  return <ExamTake id={id} />;
}
