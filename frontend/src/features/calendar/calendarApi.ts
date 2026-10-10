"use client";

import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { useEffect } from "react";
import { apiClient, useCursorList } from "@/shared/data";

// Hợp đồng thật: backend-go/api/openapi.yaml (`/courses/{id}/calendar*`, `/me/calendar/ics-token`).
export type CalType = "CLASS_SESSION" | "EXAM" | "OTHER";
export type CalItem = {
  id: string; source: "class_session" | "weekly_exam" | "calendar_event"; type: CalType; title: string; starts_at: string; ends_at: string | null; location: string | null;
  href: string | null; editable: boolean; personal_state: "NOT_STARTED" | "IN_PROGRESS" | "SUBMITTED" | null; status: string | null; version?: number;
};
export type EventInput = { type: "EXAM" | "OTHER"; title: string; starts_at: string; ends_at: string | null; location: string | null; version?: number };

export const calKey = (course: string) => ["calendar", course] as const;

export const STATE_LABEL: Record<string, string> = { NOT_STARTED: "Chưa làm", IN_PROGRESS: "Đang làm", SUBMITTED: "Đã nộp" };
export const TYPE_LABEL: Record<CalType, string> = { CLASS_SESSION: "Buổi học", EXAM: "Bài thi", OTHER: "Sự kiện" };

// Giờ Việt Nam: múi giờ cố định UTC+7 (không DST). "Nhãn ngày" = nửa đêm UTC của ngày dương lịch ở Việt Nam, để làm số học ngày bằng getUTC*.
export const VN = 7 * 3_600_000;
export const DAY = 86_400_000;
const WD = ["Chủ Nhật", "Thứ Hai", "Thứ Ba", "Thứ Tư", "Thứ Năm", "Thứ Sáu", "Thứ Bảy"];
export const WD_SHORT = ["T2", "T3", "T4", "T5", "T6", "T7", "CN"];

export const dayLabel = (t: number | string | Date): number => {
  const x = new Date(new Date(t).getTime() + VN);
  return Date.UTC(x.getUTCFullYear(), x.getUTCMonth(), x.getUTCDate());
};
export const labelToIso = (label: number): string => new Date(label - VN).toISOString();
export const weekStartLabel = (label: number): number => label - ((new Date(label).getUTCDay() + 6) % 7) * DAY;
const p2 = (n: number) => String(n).padStart(2, "0");
export const fmtDM = (label: number) => `${p2(new Date(label).getUTCDate())}/${p2(new Date(label).getUTCMonth() + 1)}`;
export const fmtHM = (t: string | number | Date) => {
  const x = new Date(new Date(t).getTime() + VN);
  return `${p2(x.getUTCHours())}:${p2(x.getUTCMinutes())}`;
};
/** "Thứ Hai, 21/09 · 14:00" */
export const fmtLong = (iso: string) => `${WD[new Date(dayLabel(iso)).getUTCDay()]}, ${fmtDM(dayLabel(iso))} · ${fmtHM(iso)}`;
/** giá trị cho <input type="datetime-local"> theo giờ Việt Nam */
export const toLocalInput = (iso: string) => new Date(new Date(iso).getTime() + VN).toISOString().slice(0, 16);
export const fromLocalInput = (v: string) => new Date(`${v}:00+07:00`).toISOString();

/** Đỏ chỉ cho bài thi trong 48 giờ tới (nghĩa "cần hành động"). */
export const isSoonExam = (e: Pick<CalItem, "type" | "starts_at" | "ends_at">, now = Date.now()) => {
  const start = new Date(e.starts_at).getTime();
  const end = e.ends_at ? new Date(e.ends_at).getTime() : start;
  return e.type === "EXAM" && end >= now && start - now <= 48 * 3_600_000;
};

/** Tải hết các trang của khoảng (≤ 62 ngày) rồi trả danh sách phẳng. */
export function useCalendar(course: string, from: string, to: string) {
  const list = useCursorList<CalItem>([...calKey(course), from, to], `/courses/${course}/calendar`, { limit: 100, query: { from, to } });
  const { hasNextPage, isFetchingNextPage, fetchNextPage } = list;
  useEffect(() => {
    if (hasNextPage && !isFetchingNextPage) void fetchNextPage();
  }, [hasNextPage, isFetchingNextPage, fetchNextPage]);
  return list;
}

export function useEventMutations(course: string) {
  const qc = useQueryClient();
  const done = () => qc.invalidateQueries({ queryKey: calKey(course) });
  const base = `/courses/${course}/calendar/events`;
  return {
    create: useMutation({ mutationFn: (b: EventInput) => apiClient.post(base, b), onSuccess: done }),
    update: useMutation({ mutationFn: ({ id, ...b }: EventInput & { id: string }) => apiClient.put(`${base}/${id}`, b), onSuccess: done }),
    remove: useMutation({ mutationFn: (id: string) => apiClient.delete(`${base}/${id}`), onSuccess: done }),
  };
}

const ICS_KEY = ["me", "ics-token"] as const;
export const useIcsState = (enabled: boolean) =>
  useQuery({ queryKey: ICS_KEY, enabled, queryFn: async ({ signal }) => (await apiClient.get<{ exists: boolean }>("/me/calendar/ics-token", { signal })).data });

export function useIssueIcs() {
  const qc = useQueryClient();
  return useMutation({ mutationFn: async () => (await apiClient.post<{ url: string }>("/me/calendar/ics-token")).data.url, onSuccess: () => qc.invalidateQueries({ queryKey: ICS_KEY }) });
}
