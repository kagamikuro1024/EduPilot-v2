"use client";

import { useQuery } from "@tanstack/react-query";
import { useEffect, useRef, useState } from "react";
import { ApiError, apiClient } from "@/shared/data";
import { InlineNotice, Section } from "@/shared/ui";
import s from "./ClassSettings.module.css";
import { classKey } from "./classApi";

type CourseSettings = { settings?: { escalation_threshold: string; version: number } };

const MIN = 0.5;
const MAX = 0.95;
const SAVE_DELAY_MS = 400;

const vi = (n: number) => n.toFixed(2).replace(".", ",");

/**
 * `Ngưỡng chuyển cho giảng viên` (0,50–0,95, bước 0,05): lạc quan — kéo là giá trị đổi ngay, lưu sau khi dừng kéo; 409 → nạp lại giá trị mới nhất.
 * TA chỉ xem. Không có câu mô tả: hai đầu thanh là "Ít chuyển" / "Chuyển nhiều".
 */
export function ThresholdSection({ courseId, canManage }: { courseId: string; canManage: boolean }) {
  const key = classKey(courseId, "settings");
  const q = useQuery({ queryKey: key, staleTime: 0, queryFn: async ({ signal }) => (await apiClient.get<CourseSettings>(`/courses/${courseId}`, { signal })).data.settings ?? null });
  const server = q.data ? Number(q.data.escalation_threshold) : null;
  const [value, setValue] = useState<number | null>(null);
  const [notice, setNotice] = useState<string | null>(null);
  const version = useRef(0);
  const timer = useRef<ReturnType<typeof setTimeout> | null>(null);
  const dirty = useRef(false);

  useEffect(() => {
    if (q.data && !dirty.current) {
      setValue(Number(q.data.escalation_threshold));
      version.current = q.data.version;
    }
  }, [q.data]);
  useEffect(() => () => { if (timer.current) clearTimeout(timer.current); }, []);

  async function save(next: number) {
    try {
      const { data } = await apiClient.put<CourseSettings>(`/courses/${courseId}/settings`, { escalation_threshold: next.toFixed(2), version: version.current });
      version.current = data.settings?.version ?? version.current;
      dirty.current = false;
      setNotice(null);
    } catch (e) {
      dirty.current = false;
      if (e instanceof ApiError && e.code === "VERSION_CONFLICT") setNotice("Vừa có người đổi. Đã nạp lại giá trị mới.");
      else setNotice(e instanceof ApiError ? e.userMessage : "Chưa lưu được.");
      const fresh = await q.refetch(); // quay về giá trị thật của máy chủ
      if (fresh.data) {
        setValue(Number(fresh.data.escalation_threshold));
        version.current = fresh.data.version;
      } else if (server !== null) setValue(server);
    }
  }

  function change(raw: string) {
    const next = Math.min(MAX, Math.max(MIN, Math.round(Number(raw) * 20) / 20));
    dirty.current = true;
    setValue(next);
    setNotice(null);
    if (timer.current) clearTimeout(timer.current);
    timer.current = setTimeout(() => void save(next), SAVE_DELAY_MS);
  }

  if (!q.data || value === null) return null;
  return (
    <Section panel title="Ngưỡng chuyển cho giảng viên">
      <div className={s.threshold}>
        <span className={s.readonly}>Ít chuyển</span>
        <input
          type="range"
          className={s.range}
          aria-label="Ngưỡng chuyển cho giảng viên"
          min={MIN}
          max={MAX}
          step={0.05}
          value={value}
          disabled={!canManage}
          onChange={(e) => change(e.target.value)}
        />
        <span className={s.readonly}>Chuyển nhiều</span>
        <output className={s.thresholdValue} data-part="threshold-value">{vi(value)}</output>
      </div>
      {notice && <InlineNotice tone="warning" compact>{notice}</InlineNotice>}
    </Section>
  );
}
