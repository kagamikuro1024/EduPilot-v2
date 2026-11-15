import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { apiClient } from "@/shared/data";

// Hợp đồng thật: backend-go/api/openapi.yaml (12 thao tác điểm danh tự phục vụ, tag `attendance`).
export type CheckinWindow = {
  id: string;
  round: number;
  opened_at: string;
  closes_at: string;
  closed_at: string | null;
  headcount: number | null;
  checked_in: number;
  over_headcount: boolean;
};
export type CheckinDisplay = {
  qr_url: string;
  short_code: string;
  step_expires_at: string;
  server_time: string;
  rotate_seconds: number;
  round: number;
  checked_in: number;
  headcount: number | null;
  closes_at: string;
};
export type CloseResult = { window: CheckinWindow; written: { present: number; late: number }; unchecked: number; skipped_existing: number };
export type SpotStudent = { student_id: string; full_name: string; student_code: string; reason: "FLAGGED" | "RANDOM" };
export type CheckinFlag = { code: string; label: string };
export type CheckinRow = {
  id: string;
  window_id: string;
  round: number;
  student: { id: string; full_name: string; student_code: string };
  created_at: string;
  channel: "WEB" | "APP" | "CODE";
  assurance: 1 | 2 | 3;
  flags: CheckinFlag[];
  cleared: boolean;
};

export const windowsKey = (course: string, session: string) => ["checkin-windows", course, session] as const;
export const checkinsKey = (course: string, session: string) => ["checkins", course, session] as const;

/** Cửa sổ của một buổi (≤ 3 vòng, không phân trang). `refetchMs` > 0 thì hỏi lại theo chu kỳ (chỉ khi tab hiện). */
export const useWindows = (course: string, session: string | undefined, refetchMs: number | false = false) =>
  useQuery({
    queryKey: windowsKey(course, session ?? ""),
    enabled: Boolean(session),
    staleTime: 0,
    refetchInterval: refetchMs,
    refetchIntervalInBackground: false,
    queryFn: async ({ signal }) => (await apiClient.get<{ items: CheckinWindow[] }>(`/courses/${course}/sessions/${session}/checkin-windows`, { signal })).data.items,
  });

/** Lượt điểm danh của buổi (Giảng viên / TA). Một trang tối đa 100: lớp ≤ 100 sinh viên dùng đủ; lớp lớn đọc tiếp theo cursor. */
export const useCheckins = (course: string, session: string | undefined, enabled: boolean, refetchMs: number | false = false) =>
  useQuery({
    queryKey: checkinsKey(course, session ?? ""),
    enabled: enabled && Boolean(session),
    staleTime: 0,
    refetchInterval: refetchMs,
    refetchIntervalInBackground: false,
    queryFn: async ({ signal }) => {
      const rows: CheckinRow[] = [];
      let cursor: string | null = null;
      for (let i = 0; i < 20; i++) {
        const { data }: { data: { items: CheckinRow[]; next_cursor: string | null } } = await apiClient.get(`/courses/${course}/sessions/${session}/checkins?limit=100${cursor ? `&cursor=${encodeURIComponent(cursor)}` : ""}`, { signal });
        rows.push(...data.items);
        cursor = data.next_cursor;
        if (!cursor) break;
      }
      return rows;
    },
  });

export function useCheckinActions(course: string, session: string) {
  const qc = useQueryClient();
  const refresh = () => Promise.all([qc.invalidateQueries({ queryKey: windowsKey(course, session) }), qc.invalidateQueries({ queryKey: checkinsKey(course, session) })]);
  return {
    open: useMutation({ mutationFn: async () => (await apiClient.post<{ id: string; round: number; closes_at: string }>(`/courses/${course}/sessions/${session}/checkin-windows`, {})).data, onSuccess: refresh }),
    close: useMutation({ mutationFn: async (wid: string) => (await apiClient.post<CloseResult>(`/courses/${course}/checkin-windows/${wid}/close`, {})).data, onSuccess: refresh }),
    headcount: useMutation({ mutationFn: async (v: { wid: string; headcount: number | null }) => (await apiClient.patch<CheckinWindow>(`/courses/${course}/checkin-windows/${v.wid}`, { headcount: v.headcount })).data, onSuccess: refresh }),
    spot: useMutation({ mutationFn: async (wid: string) => (await apiClient.post<{ students: SpotStudent[] }>(`/courses/${course}/checkin-windows/${wid}/spot-check`, { n: 5 })).data.students }),
    absent: useMutation({ mutationFn: async () => (await apiClient.post<{ marked: number }>(`/courses/${course}/sessions/${session}/absent-unchecked`, {})).data, onSuccess: () => Promise.all([refresh(), qc.invalidateQueries({ queryKey: ["attendance", course, session] })]) }),
    clear: useMutation({ mutationFn: async (cid: string) => (await apiClient.post<CheckinRow>(`/courses/${course}/checkins/${cid}/clear-flags`, {})).data, onSuccess: refresh }),
  };
}
