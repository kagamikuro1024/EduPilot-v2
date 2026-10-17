"use client";

import { useQuery } from "@tanstack/react-query";
import { useState } from "react";
import { ApiError, apiClient } from "@/shared/data";
import { InlineNotice, Section, Switch } from "@/shared/ui";
import { classKey } from "./classApi";

type Detail = { allow_study_time_opt_out: boolean; settings?: { version: number }; counts?: { students_active: number; students_pending: number } };

/**
 * `Cho sinh viên tắt đo thời gian học` (Q25): TEACHER đổi được cho tới khi lớp có sinh viên ghi danh đầu tiên; từ đó khoá (công tắc `disabled`, lý do ở chính công tắc).
 * TA luôn chỉ đọc. Lưu lạc quan; `409 COURSE_HAS_STUDENTS` (đua với sinh viên đầu tiên) hoặc sai version → hoàn lại + một dòng.
 */
export function StudyTimeSetting({ courseId, canManage }: { courseId: string; canManage: boolean }) {
  const q = useQuery({ queryKey: classKey(courseId, "study-time"), staleTime: 0, queryFn: async ({ signal }) => (await apiClient.get<Detail>(`/courses/${courseId}`, { signal })).data });
  const [pending, setPending] = useState<boolean | null>(null);
  const [notice, setNotice] = useState<string | null>(null);
  if (!q.data) return null;
  const hasStudents = (q.data.counts?.students_active ?? 0) + (q.data.counts?.students_pending ?? 0) > 0;
  const value = pending ?? q.data.allow_study_time_opt_out;

  async function change(next: boolean) {
    setPending(next);
    setNotice(null);
    try {
      await apiClient.put(`/courses/${courseId}/settings`, { allow_study_time_opt_out: next, version: q.data?.settings?.version });
    } catch (e) {
      setNotice(e instanceof ApiError && e.code === "COURSE_HAS_STUDENTS" ? "Lớp đã có sinh viên" : e instanceof ApiError && e.code === "VERSION_CONFLICT" ? "Vừa có người đổi. Đã nạp lại giá trị mới." : e instanceof ApiError ? e.userMessage : "Chưa lưu được.");
    } finally {
      await q.refetch();
      setPending(null);
    }
  }

  return (
    <Section panel title="Đo thời gian học">
      <Switch
        label="Cho sinh viên tắt đo thời gian học"
        checked={value}
        disabled={!canManage || hasStudents}
        title={canManage && hasStudents ? "Lớp đã có sinh viên" : undefined}
        onChange={(v) => void change(v)}
      />
      {notice && <InlineNotice tone="warning" compact>{notice}</InlineNotice>}
    </Section>
  );
}
