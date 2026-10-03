import type { Metadata } from "next";
import { MeScreen } from "@/features/me/MeScreen";

export const metadata: Metadata = { title: "Kết quả của tôi" };

export default function Page() {
  return <MeScreen />;
}
