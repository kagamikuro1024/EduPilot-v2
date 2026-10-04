import type { Metadata } from "next";
import { JoinScreen } from "@/features/join/JoinScreen";

export const metadata: Metadata = { title: "Tham gia lớp", referrer: "no-referrer" };

export default function Page() {
  return <JoinScreen />;
}
