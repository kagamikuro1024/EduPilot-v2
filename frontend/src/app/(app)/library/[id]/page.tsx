import type { Metadata } from "next";
import { LibraryDetail } from "@/features/library/LibraryScreen";

export const metadata: Metadata = { title: "Tài liệu" };

export default async function Page({ params }: { params: Promise<{ id: string }> }) {
  const { id } = await params;
  return <LibraryDetail id={id} />;
}
