"use client";

import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { apiClient } from "@/shared/data";
import { ApiErrorNotice } from "@/shared/data/ApiErrorNotice";
import { PageState, Section, Switch } from "@/shared/ui";

// Hợp đồng thật: backend-go/api/openapi.yaml (`/me/settings`, `notify_ticket_by_mail`, `notify_answer_by_mail`).
type Flags = { notify_ticket_by_mail: boolean; notify_answer_by_mail: boolean };
type Settings = Flags & { version: number };
const KEY = ["me", "settings"] as const;

/** Công tắc thư theo vai: TA / giảng viên bật "Mail báo ticket mới" (mặc định tắt); sinh viên "Mail khi giảng viên trả lời" (mặc định bật). Công tắc không áp dụng thì không hiện. */
export function MailSection({ role }: { role: "student" | "ta" | "teacher" }) {
  const qc = useQueryClient();
  const q = useQuery({ queryKey: KEY, queryFn: async ({ signal }) => (await apiClient.get<Settings>("/me/settings", { signal })).data });
  const flag: keyof Flags = role === "student" ? "notify_answer_by_mail" : "notify_ticket_by_mail";
  const save = useMutation({
    mutationFn: async (v: boolean) => (await apiClient.put<Settings>("/me/settings", { [flag]: v, version: q.data?.version })).data,
    // Lạc quan, không toast: công tắc đổi ngay; lỗi (kể cả 409 sai version) thì lấy lại bản thật.
    onMutate: async (v) => {
      await qc.cancelQueries({ queryKey: KEY });
      const prev = qc.getQueryData<Settings>(KEY);
      if (prev) qc.setQueryData<Settings>(KEY, { ...prev, [flag]: v });
    },
    onSuccess: (d) => qc.setQueryData(KEY, d),
    onError: () => qc.invalidateQueries({ queryKey: KEY }),
  });
  return (
    <Section title="Thư" panel>
      <PageState query={q}>
        {save.isError && <ApiErrorNotice error={save.error} />}
        <Switch label={role === "student" ? "Mail khi giảng viên trả lời" : "Mail báo ticket mới"} checked={q.data?.[flag] ?? false} onChange={(v) => save.mutate(v)} />
      </PageState>
    </Section>
  );
}
