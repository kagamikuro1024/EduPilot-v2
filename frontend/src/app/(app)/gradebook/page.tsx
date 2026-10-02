import type { Metadata } from "next";
import { Gradebook } from "@/features/gradebook/Gradebook";

export const metadata: Metadata = { title: "Sổ điểm" };

export default function Page() {
  return <Gradebook />;
}
