import type { Metadata } from "next";
import { ExamEditor } from "@/features/exam/ExamEditor";

export const metadata: Metadata = { title: "Soạn bài thi" };

export default async function Page({ params }: { params: Promise<{ id: string }> }) {
  const { id } = await params;
  return <ExamEditor id={id} />;
}
