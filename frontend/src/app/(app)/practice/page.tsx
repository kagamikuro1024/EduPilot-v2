import type { Metadata } from "next";
import { PracticeScreen } from "@/features/practice/PracticeLazy";

export const metadata: Metadata = { title: "Luyện đề" };

export default function Page() {
  return <PracticeScreen />;
}
