import type { Metadata } from "next";
import { ThreadDetail } from "@/features/threads/ThreadDetail";

export const metadata: Metadata = { title: "Thread" };

export default async function Page({ params }: { params: Promise<{ id: string }> }) {
  const { id } = await params;
  return <ThreadDetail id={id} />;
}
