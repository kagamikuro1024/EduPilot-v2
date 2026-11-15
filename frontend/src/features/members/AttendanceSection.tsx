"use client";

import { useQuery } from "@tanstack/react-query";
import { useEffect, useRef, useState } from "react";
import { ApiError, apiClient, fieldErrors } from "@/shared/data";
import { Button, Field, InlineNotice, Input, Section, SegmentedControl, StatusText, Switch, Textarea } from "@/shared/ui";
import s from "./ClassSettings.module.css";
import { classKey } from "./classApi";

// Hợp đồng thật: backend-go/api/openapi.yaml (`GET /courses/{id}` → `attendance`, `PUT /courses/{id}/settings {attendance}` — thay TOÀN BỘ chính sách).
type Geofence = { lat: number; lng: number; radius_m: number };
type Policy = { self_checkin: boolean; rotate_seconds: number; late_after_min: number; require: "WEB" | "APP"; geofence?: Geofence | null; campus_cidrs?: string[] };
type Detail = { attendance?: Policy; settings?: { version: number } };

const SAVE_DELAY_MS = 600;
const vi4 = (n: number) => n.toFixed(4).replace(".", ",");

/**
 * `Điểm danh bằng mã` (DESIGN §14.8): công tắc; bật thì thêm Chu kỳ mã, Tính muộn sau, Yêu cầu; «Vị trí» và «Mạng trường» mở dần.
 * Lưu tự động sau khi dừng gõ, gửi cả chính sách; lỗi 422 hiện tại ô. TA chỉ xem.
 */
export function AttendanceSection({ courseId, canManage }: { courseId: string; canManage: boolean }) {
  const q = useQuery({ queryKey: classKey(courseId, "attendance"), staleTime: 0, queryFn: async ({ signal }) => (await apiClient.get<Detail>(`/courses/${courseId}`, { signal })).data });
  const server = q.data?.attendance;
  const [p, setP] = useState<Policy | null>(null);
  const [cidrs, setCidrs] = useState("");
  const [errors, setErrors] = useState<Record<string, string>>({});
  const [notice, setNotice] = useState<string | null>(null);
  const [locating, setLocating] = useState(false);
  const version = useRef(0);
  const timer = useRef<ReturnType<typeof setTimeout> | null>(null);
  const dirty = useRef(false);

  useEffect(() => {
    if (server && !dirty.current) {
      setP(server);
      setCidrs((server.campus_cidrs ?? []).join("\n"));
    }
    if (q.data?.settings) version.current = q.data.settings.version;
  }, [q.data, server]);
  useEffect(() => () => { if (timer.current) clearTimeout(timer.current); }, []);

  async function save(next: Policy, cidrText: string) {
    const body: Policy = {
      self_checkin: next.self_checkin,
      rotate_seconds: next.rotate_seconds,
      late_after_min: next.late_after_min,
      require: next.require,
      ...(next.geofence ? { geofence: next.geofence } : {}),
      campus_cidrs: cidrText.split("\n").map((x) => x.trim()).filter(Boolean),
    };
    try {
      await apiClient.put(`/courses/${courseId}/settings`, { attendance: body, version: version.current });
      setErrors({});
      setNotice(null);
    } catch (e) {
      const fe = fieldErrors(e);
      setErrors(fe);
      if (Object.keys(fe).length === 0) setNotice(e instanceof ApiError && e.code === "VERSION_CONFLICT" ? "Vừa có người đổi. Đã nạp lại giá trị mới." : e instanceof ApiError ? e.userMessage : "Chưa lưu được.");
      if (!(e instanceof ApiError) || e.status !== 422) dirty.current = false;
    }
    const fresh = await q.refetch(); // phiên bản mới (và giá trị thật của máy chủ nếu lưu lỗi)
    if (fresh.data?.settings) version.current = fresh.data.settings.version;
    if (!dirty.current && fresh.data?.attendance) {
      setP(fresh.data.attendance);
      setCidrs((fresh.data.attendance.campus_cidrs ?? []).join("\n"));
    }
  }

  function change(next: Policy, cidrText = cidrs) {
    dirty.current = true;
    setP(next);
    setCidrs(cidrText);
    setNotice(null);
    if (timer.current) clearTimeout(timer.current);
    timer.current = setTimeout(() => void save(next, cidrText).then(() => { dirty.current = false; }), SAVE_DELAY_MS);
  }

  function locateHere() {
    if (!p || !navigator.geolocation) return;
    setLocating(true);
    navigator.geolocation.getCurrentPosition(
      (pos) => {
        setLocating(false);
        change({ ...p, geofence: { lat: pos.coords.latitude, lng: pos.coords.longitude, radius_m: p.geofence?.radius_m ?? 150 } });
      },
      () => {
        setLocating(false);
        setErrors({ "attendance.geofence": "Chưa lấy được vị trí. Hãy cho phép trình duyệt dùng vị trí." });
      },
      { enableHighAccuracy: true, timeout: 15000 },
    );
  }

  if (!q.data || !p) return null;
  const geoOn = Boolean(p.geofence);
  const num = (v: string, lo: number, hi: number) => Math.min(hi, Math.max(lo, Math.round(Number(v) || 0)));

  return (
    <Section title="Điểm danh bằng mã">
      <div className={s.form} data-part="attendance-settings">
        <Switch label="Điểm danh bằng mã" checked={p.self_checkin} disabled={!canManage} onChange={(v) => change({ ...p, self_checkin: v })} />
        {p.self_checkin && (
          <>
            <div className={s.fields}>
              <Field label="Chu kỳ mã (giây)" error={errors["attendance.rotate_seconds"]}>
                {(id, d) => <Input id={id} aria-describedby={d} type="number" inputMode="numeric" min={5} max={60} value={p.rotate_seconds} disabled={!canManage} onChange={(e) => change({ ...p, rotate_seconds: num(e.target.value, 5, 60) })} />}
              </Field>
              <Field label="Tính muộn sau (phút)" error={errors["attendance.late_after_min"]}>
                {(id, d) => <Input id={id} aria-describedby={d} type="number" inputMode="numeric" min={0} max={120} value={p.late_after_min} disabled={!canManage} onChange={(e) => change({ ...p, late_after_min: num(e.target.value, 0, 120) })} />}
              </Field>
            </div>
            <Field label="Yêu cầu" error={errors["attendance.require"]}>
              {() => (
                <SegmentedControl
                  label="Yêu cầu"
                  value={p.require}
                  onChange={(v) => canManage && change({ ...p, require: v })}
                  options={[{ value: "WEB", label: "Web hoặc app" }, { value: "APP", label: "Chỉ app" }]}
                />
              )}
            </Field>
            <Switch
              label="Vị trí"
              checked={geoOn}
              disabled={!canManage}
              onChange={(v) => {
                if (!v) change({ ...p, geofence: null });
                else locateHere();
              }}
            />
            {geoOn && p.geofence && (
              <div className={s.fields}>
                <Field label="Bán kính (m)" error={errors["attendance.geofence.radius_m"] ?? errors["attendance.geofence"]}>
                  {(id, d) => (
                    <Input id={id} aria-describedby={d} type="number" inputMode="numeric" min={20} max={5000} value={p.geofence!.radius_m} disabled={!canManage} onChange={(e) => change({ ...p, geofence: { ...p.geofence!, radius_m: num(e.target.value, 20, 5000) } })} />
                  )}
                </Field>
                <div className={s.actions}>
                  <Button disabled={!canManage || locating} onClick={locateHere}>Dùng vị trí hiện tại</Button>
                  <StatusText tone="neutral">{vi4(p.geofence.lat)} · {vi4(p.geofence.lng)}</StatusText>
                </div>
                <p className={s.readonly} data-part="geo-notice">Lớp này kiểm tra vị trí khi điểm danh.</p>
              </div>
            )}
            {!geoOn && errors["attendance.geofence"] && <InlineNotice tone="warning" compact>{errors["attendance.geofence"]}</InlineNotice>}
            <Field label="Mạng trường" error={errors["attendance.campus_cidrs"]}>
              {(id, d) => <Textarea id={id} aria-describedby={d} rows={2} placeholder="10.0.0.0/8" value={cidrs} disabled={!canManage} onChange={(e) => change(p, e.target.value)} />}
            </Field>
          </>
        )}
        {notice && <InlineNotice tone="warning" compact>{notice}</InlineNotice>}
      </div>
    </Section>
  );
}
