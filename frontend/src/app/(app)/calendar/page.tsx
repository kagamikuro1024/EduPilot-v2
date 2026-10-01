import type { Metadata } from "next";
import { CalendarScreen } from "@/features/calendar/CalendarScreen";

export const metadata: Metadata = { title: "Lịch" };

export default function Page() {
  return <CalendarScreen />;
}
