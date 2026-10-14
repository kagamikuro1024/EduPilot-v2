"use client";

import { useQuery } from "@tanstack/react-query";
import { useEffect, useRef, useState } from "react";
import { ApiError, apiClient } from "@/shared/data";
import { InlineNotice, Section } from "@/shared/ui";
import s from "./ClassSettings.module.css";
import { classKey } from "./classApi";

type Risk = { attendance_ok_min: string; activity_min_per_week: number; study_min_minutes: number; signals_required: number; absent_alert: number };
type CourseSettings = { settings?: { version: number; risk: Risk } };
type Field = keyof Risk;

const SAVE_DELAY_MS = 400;
const vi = (n: number) => n.toFixed(2).replace(".", ",");

// Năm dòng: nhãn + thanh + giá trị (US-P5-04 AC6). Khoảng và bước theo SRS crm-360 4.7.3.
const ROWS: Array<{ field: Field; label: string; min: number; max: number; step: number; show: (n: number) => string }> = [
  { field: "attendance_ok_min", label: "Tỷ lệ có mặt tối thiểu", min: 0.5, max: 1, step: 0.05, show: vi },
  { field: "activity_min_per_week", label: "Hoạt động tối thiểu mỗi tuần", min: 0, max: 20, step: 1, show: String },
  { field: "study_min_minutes", label: "Phút học tối thiểu mỗi tuần", min: 0, max: 600, step: 5, show: String },
  { field: "signals_required", label: "Số tín hiệu thấp để tính", min: 1, max: 3, step: 1, show: String },
  { field: "absent_alert", label: "Vắng nhiều từ", min: 1, max: 20, step: 1, show: (n) => `${n} buổi` },
];

const num = (r: Risk, f: Field) => Number(r[f]);

/** `Ngưỡng cần chú ý`: lưu lạc quan (kéo là đổi ngay, lưu sau khi dừng); 409 → hoàn lại + một dòng. TA chỉ xem. */
export function RiskSection({ courseId, canManage }: { courseId: string; canManage: boolean }) {
  const q = useQuery({ queryKey: classKey(courseId, "risk"), staleTime: 0, queryFn: async ({ signal }) => (await apiClient.get<CourseSettings>(`/courses/${courseId}`, { signal })).data.settings ?? null });
  const [draft, setDraft] = useState<Partial<Record<Field, number>>>({});
  const [notice, setNotice] = useState<string | null>(null);
  const timers = useRef<Partial<Record<Field, ReturnType<typeof setTimeout>>>>({});
  useEffect(() => () => { for (const t of Object.values(timers.current)) clearTimeout(t); }, []);

  async function save(field: Field, value: number) {
    try {
      await apiClient.put(`/courses/${courseId}/settings`, { risk: { [field]: field === "attendance_ok_min" ? value.toFixed(2) : value }, version: q.data?.version });
      setNotice(null);
    } catch (e) {
      setNotice(e instanceof ApiError && e.code === "VERSION_CONFLICT" ? "Vừa có người đổi. Đã nạp lại giá trị mới." : e instanceof ApiError ? e.userMessage : "Chưa lưu được.");
    } finally {
      await q.refetch(); // phiên bản mới + giá trị thật của máy chủ (hoàn lại nếu lưu lỗi)
      setDraft((d) => {
        const { [field]: _gone, ...rest } = d; // eslint-disable-line @typescript-eslint/no-unused-vars
        return rest;
      });
    }
  }

  function change(field: Field, raw: string, step: number) {
    const value = Math.round(Number(raw) / step) * step;
    setDraft((d) => ({ ...d, [field]: value }));
    setNotice(null);
    clearTimeout(timers.current[field]);
    timers.current[field] = setTimeout(() => void save(field, value), SAVE_DELAY_MS);
  }

  const risk = q.data?.risk;
  if (!risk) return null; // chưa tải (hoặc phản hồi không mang ngưỡng): không dựng khối rỗng
  return (
    <Section panel title="Ngưỡng cần chú ý">
      <div className={s.riskRows}>
        {ROWS.map((r) => {
          const value = draft[r.field] ?? num(risk, r.field);
          return (
            <div key={r.field} className={s.riskRow}>
              <label htmlFor={`risk-${r.field}`} className={s.riskLabel}>{r.label}</label>
              <input id={`risk-${r.field}`} type="range" className={s.range} min={r.min} max={r.max} step={r.step} value={value} disabled={!canManage} onChange={(e) => change(r.field, e.target.value, r.step)} />
              <output htmlFor={`risk-${r.field}`} className={s.riskValue} data-part={`risk-${r.field}`}>{r.show(value)}</output>
            </div>
          );
        })}
      </div>
      {notice && <InlineNotice tone="warning" compact>{notice}</InlineNotice>}
    </Section>
  );
}
