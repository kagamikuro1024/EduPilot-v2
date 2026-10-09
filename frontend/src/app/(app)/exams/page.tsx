import type { Metadata } from "next";
import { ExamsHome } from "@/features/exam/ExamsHome";

export const metadata: Metadata = { title: "Bài thi" };

export default function Page() {
  return <ExamsHome />;
}
