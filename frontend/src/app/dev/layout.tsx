import { notFound } from "next/navigation";

// Công cụ dev chỉ có ở `next dev` hoặc bản dựng đặt NEXT_PUBLIC_DEV_TOOLS=1 (cờ thời điểm dựng); bản "như production" → 404.
const ENABLED = process.env.NODE_ENV !== "production" || process.env.NEXT_PUBLIC_DEV_TOOLS === "1";

export default function DevLayout({ children }: { children: React.ReactNode }) {
  if (!ENABLED) notFound();
  return children;
}
