import type { Metadata } from "next";
import { JoinScreen } from "@/features/join/JoinScreen";

export const metadata: Metadata = { title: "Tham gia lớp", referrer: "no-referrer" };

export default async function Page({ params }: { params: Promise<{ code: string }> }) {
  const { code } = await params;
  return <JoinScreen initialCode={code} />;
}
