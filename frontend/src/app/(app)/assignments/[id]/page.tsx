import type { Metadata } from "next";
import { AssignmentScreen } from "@/features/assignments/AssignmentScreen";

export const metadata: Metadata = { title: "Bài tập" };

export default async function Page({ params }: { params: Promise<{ id: string }> }) {
  const { id } = await params;
  return <AssignmentScreen id={id} />;
}
