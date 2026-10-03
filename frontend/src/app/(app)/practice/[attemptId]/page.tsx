import type { Metadata } from "next";
import { AttemptScreen } from "@/features/practice/AttemptScreen";

export const metadata: Metadata = { title: "Làm bài" };

export default async function Page({ params }: { params: Promise<{ attemptId: string }> }) {
  const { attemptId } = await params;
  return <AttemptScreen attemptId={attemptId} />;
}
