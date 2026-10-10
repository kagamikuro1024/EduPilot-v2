"use client";

import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { apiClient } from "@/shared/data";
import { ApiErrorNotice } from "@/shared/data/ApiErrorNotice";
import { PageState, Section, Switch } from "@/shared/ui";

// Hợp đồng thật: backend-go/api/openapi.yaml (`/me/settings`, `reminders`).
type Reminders = { exam: boolean; class_session: boolean; other: boolean };
type Settings = { reminders: Reminders; version: number };
const KEY = ["me", "settings"] as const;
const ROWS: [keyof Reminders, string][] = [["exam", "Bài thi"], ["class_session", "Buổi học"], ["other", "Sự kiện khác"]];

/** Nhắc trước 24 giờ theo loại — chỉ sinh viên nhận nhắc nên chỉ sinh viên thấy khối này. */
export function RemindersSection() {
  const qc = useQueryClient();
  const q = useQuery({ queryKey: KEY, queryFn: async ({ signal }) => (await apiClient.get<Settings>("/me/settings", { signal })).data });
  const save = useMutation({
    mutationFn: async (patch: Partial<Reminders>) => (await apiClient.put<Settings>("/me/settings", { reminders: patch, version: q.data?.version })).data,
    // Lạc quan: công tắc đổi ngay; lỗi (kể cả 409 sai version) thì lấy lại bản thật.
    onMutate: async (patch) => {
      await qc.cancelQueries({ queryKey: KEY });
      const prev = qc.getQueryData<Settings>(KEY);
      if (prev) qc.setQueryData<Settings>(KEY, { ...prev, reminders: { ...prev.reminders, ...patch } });
    },
    onSuccess: (d) => qc.setQueryData(KEY, d),
    onError: () => qc.invalidateQueries({ queryKey: KEY }),
  });
  return (
    <Section title="Nhắc trước 24 giờ" panel>
      <PageState query={q}>
        {save.isError && <ApiErrorNotice error={save.error} />}
        {ROWS.map(([k, label]) => (
          <Switch key={k} label={label} checked={q.data?.reminders[k] ?? false} onChange={(v) => save.mutate({ [k]: v })} />
        ))}
      </PageState>
    </Section>
  );
}
