"use client";

import { keepPreviousData, useQuery } from "@tanstack/react-query";
import { apiClient } from "@/shared/data";
import { useSession } from "@/shared/session/session";

// Hợp đồng thật: backend-go/api/openapi.yaml (`GET /me/today`, `GET /courses/{id}/today`).
export type TodayCourse = { id: string; class_code: string };
export type TodayStep = { key: string; label: string; done: boolean; href: string };
export type TodayItem = {
  id: string;
  kind: string;
  title: string;
  reason: string;
  urgency: "overdue" | "high" | "normal";
  href: string;
  course: TodayCourse | null;
  age_minutes?: number;
  estimate_minutes?: number;
  steps?: TodayStep[];
};
export type TodaySession = { at: string; ends_at: string; title: string; place: string; state: "NOW" | "NEXT" | "DONE"; course: TodayCourse };
export type StudentToday = { no_course: boolean; email_verified: boolean; recommended: TodayItem | null; timeline: TodaySession[]; continue: unknown[] };
export type StaffToday = { count: number; actions: TodayItem[]; attention: unknown[]; upcoming: { at: string; title: string; place: string; course: TodayCourse }[] };
export type AdminToday = { count: number; actions: TodayItem[] };

export const TODAY_KEY = ["today"] as const;

/** Đường dẫn "Hôm nay": một lớp thật đang chọn ⇒ `/courses/{id}/today`; "Tất cả lớp", Admin, chưa có lớp ⇒ `/me/today`. */
export function useToday<T>() {
  const { role, realCourses, realCourseId } = useSession();
  const ready = role === "admin" || realCourses !== null;
  const path = role !== "admin" && realCourseId && realCourseId !== "all" ? `/courses/${realCourseId}/today` : "/me/today";
  const q = useQuery({
    queryKey: [...TODAY_KEY, path],
    enabled: ready,
    staleTime: 0,
    refetchInterval: 60_000, // việc xử lý xong biến ≤ 60 s dù mất sự kiện (SRS 4.7)
    placeholderData: keepPreviousData, // làm mới thất bại vẫn giữ dữ liệu cũ, không nháy trống
    queryFn: async ({ signal }) => (await apiClient.get<T>(path, { signal })).data,
  });
  return { ...q, path, isAll: path === "/me/today" && role !== "admin" };
}
