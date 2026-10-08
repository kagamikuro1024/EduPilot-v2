import type { Metadata } from "next";
import { ResultsPage } from "@/features/exam/results/ResultsPage";

export const metadata: Metadata = { title: "Kết quả bài thi" };

export default async function Page({ params }: { params: Promise<{ id: string }> }) {
  const { id } = await params;
  return <ResultsPage id={id} />;
}
