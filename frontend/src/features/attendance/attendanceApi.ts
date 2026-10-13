import { useQuery } from "@tanstack/react-query";
import { apiClient } from "@/shared/data";

// Hợp đồng thật: backend-go/api/openapi.yaml (`/courses/{id}/sessions/{sid}/attendance`, `/courses/{id}/participation`).
export type Status = "PRESENT" | "LATE" | "EXCUSED" | "ABSENT";
export const STATUSES: Array<{ value: Status; label: string; key: string; verb: string }> = [
  { value: "PRESENT", label: "Có mặt", key: "1", verb: "Đã đánh có mặt" },
  { value: "LATE", label: "Muộn", key: "2", verb: "Đã đánh muộn" },
  { value: "EXCUSED", label: "Vắng phép", key: "3", verb: "Đã đánh vắng phép" },
  { value: "ABSENT", label: "Vắng", key: "4", verb: "Đã đánh vắng" },
];
export const labelOf = (s: Status) => STATUSES.find((x) => x.value === s)?.label ?? s;

export type GridRecord = { status: Status; note: string | null; version: number; marked_by: { full_name: string }; marked_at: string };
export type GridStudent = { id: string; full_name: string; student_code: string | null; record: GridRecord | null; speak_count: number };
export type Grid = {
  session: { id: string; session_no: number; starts_at: string; ends_at: string; room: string | null; topic: string | null; attendance_done_at: string | null };
  students: GridStudent[];
};
export type PutResult = { records: Array<{ student_id: string; status: Status; version: number }>; done_at: string | null };
export type Conflict = {
  student_id: string;
  mine: { status: Status; note?: string | null };
  current: { status: Status; note?: string | null; version: number; marked_by: { full_name: string }; marked_at: string };
};
export type SpeakCreated = { id: string };

export const gridKey = (course: string, session: string) => ["attendance", course, session] as const;

/** Lưới của một buổi. Chỉ đọc một lần khi mở (không tự tải lại: phần đang sửa ở máy là bản mới nhất, cập nhật phiên bản lấy từ phản hồi `PUT`). */
export const useGrid = (course: string, session: string | undefined) =>
  useQuery({
    queryKey: gridKey(course, session ?? ""),
    enabled: Boolean(session),
    staleTime: Infinity,
    gcTime: 0,
    refetchOnWindowFocus: false,
    queryFn: async ({ signal }) => (await apiClient.get<Grid>(`/courses/${course}/sessions/${session}/attendance`, { signal })).data,
  });

/** Khoá ở máy: có người dùng (máy dùng chung không lẫn của người trước). */
export const queueKey = (userId: string, course: string, session: string) => `ep:draft:${userId}:attendance:${course}:${session}`;
