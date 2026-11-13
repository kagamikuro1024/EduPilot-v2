import type { Metadata } from "next";
import { ThreadsScreen } from "@/features/threads/ThreadsScreen";

export const metadata: Metadata = { title: "Hỏi đáp công khai" };

export default function Page() {
  return <ThreadsScreen />;
}
