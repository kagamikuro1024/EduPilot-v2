"use client";

import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { apiClient } from "@/shared/data";
import { ApiErrorNotice } from "@/shared/data/ApiErrorNotice";
import { Section, Switch } from "@/shared/ui";

// Hợp đồng thật: backend-go/api/openapi.yaml (`/me/settings` → `track_study_time`, `study_time.courses[]`).
type Row = { course_id: string; class_code: string; name: string; locked: boolean };
type Settings = { track_study_time: boolean; version: number; study_time?: { acked: boolean; courses: Row[] } };
const KEY = ["me", "settings"] as const;

/**
 * Nhóm "Thời gian học" (SRS crm-360 7.8): mỗi lớp đang học một dòng. Lớp không cho tắt → công tắc BẬT và khoá, lý do ở chính công tắc; lớp cho tắt → mọi dòng
 * phản ánh MỘT cờ toàn tài khoản. Chưa bấm `Đã hiểu` (chưa đo) thì không có dòng nào.
 */
export function StudyTimeSection() {
  const qc = useQueryClient();
  const q = useQuery({ queryKey: KEY, queryFn: async ({ signal }) => (await apiClient.get<Settings>("/me/settings", { signal })).data });
  const save = useMutation({
    mutationFn: async (v: boolean) => (await apiClient.put<Settings>("/me/settings", { track_study_time: v, version: q.data?.version })).data,
    // Lạc quan, không toast: công tắc đổi ngay; lỗi (kể cả 409 sai version) thì lấy lại bản thật.
    onMutate: async (v) => {
      await qc.cancelQueries({ queryKey: KEY });
      const prev = qc.getQueryData<Settings>(KEY);
      if (prev) qc.setQueryData<Settings>(KEY, { ...prev, track_study_time: v });
    },
    onSuccess: (d) => qc.setQueryData<Settings>(KEY, (cur) => ({ ...d, study_time: d.study_time ?? cur?.study_time })),
    onError: () => qc.invalidateQueries({ queryKey: KEY }),
  });
  const st = q.data?.study_time;
  if (!st?.acked || st.courses.length === 0) return null;
  return (
    <Section title="Thời gian học" panel>
      {save.isError && <ApiErrorNotice error={save.error} />}
      {st.courses.map((c) => (
        <Switch
          key={c.course_id}
          label={`${c.class_code} · ${c.name}`}
          checked={c.locked || q.data?.track_study_time === true}
          disabled={c.locked}
          title={c.locked ? "Lớp này luôn ghi thời gian học" : undefined}
          onChange={(v) => save.mutate(v)}
        />
      ))}
    </Section>
  );
}
