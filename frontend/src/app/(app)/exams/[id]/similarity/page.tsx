import type { Metadata } from "next";
import { SimilarityPage } from "@/features/exam/similarity/SimilarityPage";

export const metadata: Metadata = { title: "Nghi giống nhau" };

export default async function Page({ params }: { params: Promise<{ id: string }> }) {
  const { id } = await params;
  return <SimilarityPage id={id} />;
}
