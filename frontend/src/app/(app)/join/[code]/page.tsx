import type { Metadata } from "next";
import { JoinPreview } from "@/features/join/JoinPreview";

export const metadata: Metadata = { title: "Tham gia lớp" };

export default async function Page({ params }: { params: Promise<{ code: string }> }) {
  const { code } = await params;
  return <JoinPreview code={code} />;
}
