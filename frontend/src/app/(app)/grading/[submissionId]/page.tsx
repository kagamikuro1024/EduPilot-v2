import type { Metadata } from "next";
import { SubmissionReview } from "@/features/grading/SubmissionReview";

export const metadata: Metadata = { title: "Duyệt bài" };

export default async function Page({ params }: { params: Promise<{ submissionId: string }> }) {
  const { submissionId } = await params;
  return <SubmissionReview submissionId={submissionId} />;
}
