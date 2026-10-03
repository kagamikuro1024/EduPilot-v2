"use client";

import { useQuery } from "@tanstack/react-query";
import { useState } from "react";
import { ApiError, apiClient, fieldErrors, newIdempotencyKey } from "@/shared/data";
import { useIdempotentMutation } from "@/shared/data/useIdempotentMutation";
import type { CursorPage } from "@/shared/data";
import { Button, Checkbox, Field, InlineNotice, Input, Section, Select } from "@/shared/ui";
import s from "../admin.module.css";
import type { AdminCourse, AssignResult, StaffUser } from "./api";

export type PanelMode = { kind: "open" } | { kind: "edit"; course: AdminCourse } | { kind: "assign"; course: AdminCourse };

function useStaff(role: "TEACHER" | "TA") {
  return useQuery({
    queryKey: ["admin", "users", "picker", role],
    staleTime: 30_000,
    queryFn: async ({ signal }): Promise<StaffUser[]> => {
      const page = (await apiClient.get<CursorPage<StaffUser>>("/admin/users", { signal, query: { role, limit: 100 } })).data;
      return page.items.filter((u) => u.status === "ACTIVE" || u.status === "INVITED");
    },
  });
}

type Conflict = { current: AdminCourse };

/** Mở lớp / sửa / gán lại: khung mở dần tại chỗ (không hộp thoại). Lỗi giữ nguyên chữ; gửi lại dùng CÙNG Idempotency-Key. */
export function CoursePanel({ mode, onDone, onClose }: { mode: PanelMode; onDone: (text: string) => void; onClose: () => void }) {
  const editing = mode.kind === "edit" ? mode.course : null;
  const assigning = mode.kind === "assign" ? mode.course : null;
  const base = editing ?? assigning;
  const [subject, setSubject] = useState(editing?.subject_code ?? "INT1006");
  const [classCode, setClassCode] = useState(editing?.class_code ?? "");
  const [name, setName] = useState(editing?.name ?? "An ninh mạng");
  const [semester, setSemester] = useState(editing?.semester ?? "2026-2027-HK1");
  const [capacity, setCapacity] = useState(editing?.capacity ? String(editing.capacity) : "");
  const [teacher, setTeacher] = useState(assigning?.teacher?.id ?? "");
  const [tas, setTas] = useState<string[]>([]);
  const [tasTouched, setTasTouched] = useState(false);
  const [conflict, setConflict] = useState<Conflict | null>(null);
  const [version, setVersion] = useState(base?.version ?? 1);

  const teachers = useStaff("TEACHER");
  const assistants = useStaff("TA");

  const open = useIdempotentMutation((v: Record<string, unknown>, key) => apiClient.post<{ id: string; teacher: { full_name: string } | null }>("/admin/courses", v, { idempotencyKey: key || newIdempotencyKey() }));
  const [other, setOther] = useState<{ pending: boolean; error?: unknown }>({ pending: false });

  const err = (mode.kind === "open" ? open.error : (other.error as ApiError | undefined)) ?? undefined;
  const fe = err ? fieldErrors(err) : {};
  const dup = err instanceof ApiError && err.code === "CONFLICT" && (err.details as { field?: string } | undefined)?.field === "class_code";
  const archived = err instanceof ApiError && err.code === "COURSE_ARCHIVED";
  const general = err && !dup && !archived && !(err instanceof ApiError && err.code === "VERSION_CONFLICT") && Object.keys(fe).length === 0 ? (err as ApiError).userMessage : null;
  const pending = mode.kind === "open" ? open.pending : other.pending;

  const capNum = capacity.trim() === "" ? null : Number(capacity);

  async function run(fn: () => Promise<unknown>) {
    setOther({ pending: true });
    setConflict(null);
    try {
      await fn();
      setOther({ pending: false });
    } catch (e) {
      setOther({ pending: false, error: e });
      if (e instanceof ApiError && e.code === "VERSION_CONFLICT" && e.conflict) setConflict({ current: e.conflict.current as AdminCourse });
    }
  }

  async function submit(e: React.FormEvent) {
    e.preventDefault();
    if (pending) return;
    if (mode.kind === "open") {
      const body: Record<string, unknown> = { subject_code: subject.trim(), class_code: classCode.trim(), name: name.trim(), semester: semester.trim() };
      if (capNum !== null) body.capacity = capNum;
      if (teacher) body.teacher_id = teacher;
      if (tas.length) body.ta_ids = tas;
      try {
        await open.mutate(body);
        const who = teachers.data?.find((t) => t.id === teacher)?.full_name;
        onDone(who ? `Đã mở lớp. Đã gửi thông báo phân công cho ${who}.` : "Đã mở lớp.");
      } catch {
        /* lỗi hiện tại ô / dòng thông báo; chữ đã gõ giữ nguyên */
      }
    } else if (mode.kind === "edit") {
      await run(async () => {
        await apiClient.put(`/admin/courses/${mode.course.id}`, { subject_code: subject.trim(), class_code: classCode.trim(), name: name.trim(), semester: semester.trim(), capacity: capNum, version });
        onDone("Đã lưu thay đổi của lớp.");
      });
    } else {
      await run(async () => {
        const body: Record<string, unknown> = {};
        if (teacher && teacher !== mode.course.teacher?.id) body.teacher_id = teacher;
        if (tasTouched) body.ta_ids = tas;
        if (Object.keys(body).length === 0) {
          onClose();
          return;
        }
        const { data } = await apiClient.post<AssignResult>(`/admin/courses/${mode.course.id}/assign`, body);
        const added = data.changed.teacher ? data.teacher?.full_name : undefined;
        onDone(added ? `Đã gán lại lớp. Đã gửi thông báo phân công cho ${added}.` : "Đã cập nhật đội ngũ của lớp.");
      });
    }
  }

  const title = mode.kind === "open" ? "Mở lớp mới" : mode.kind === "edit" ? `Sửa lớp ${editing?.class_code}` : `Gán lại lớp ${assigning?.class_code}`;
  const submitLabel = mode.kind === "open" ? "Mở lớp" : mode.kind === "edit" ? "Lưu thay đổi" : "Gán lại";

  return (
    <Section title={title} description={mode.kind === "open" ? "Người được gán nhận thông báo kèm mã tham gia ngay khi lớp mở; sinh viên vào sau bằng mã đó." : undefined}>
      <form onSubmit={submit} className={s.form} noValidate>
        {mode.kind !== "assign" && (
          <>
            <Field label="Học phần" error={fe.subject_code}>
              {(id, d) => <Input id={id} aria-describedby={d} invalid={!!fe.subject_code} autoComplete="off" required value={subject} onChange={(e) => setSubject(e.target.value)} />}
            </Field>
            <Field label="Mã lớp" error={dup ? "Mã lớp này đã có." : fe.class_code}>
              {(id, d) => <Input id={id} aria-describedby={d} invalid={dup || !!fe.class_code} autoComplete="off" required value={classCode} onChange={(e) => setClassCode(e.target.value)} />}
            </Field>
            <Field label="Tên lớp" error={fe.name}>
              {(id, d) => <Input id={id} aria-describedby={d} invalid={!!fe.name} autoComplete="off" required value={name} onChange={(e) => setName(e.target.value)} />}
            </Field>
            <Field label="Học kỳ" helper="Dạng 2026-2027-HK1." error={fe.semester}>
              {(id, d) => <Input id={id} aria-describedby={d} invalid={!!fe.semester} autoComplete="off" required value={semester} onChange={(e) => setSemester(e.target.value)} />}
            </Field>
            <Field label="Sĩ số" helper="Không bắt buộc, 1–1.000." error={fe.capacity}>
              {(id, d) => <Input id={id} aria-describedby={d} invalid={!!fe.capacity} inputMode="numeric" value={capacity} onChange={(e) => setCapacity(e.target.value)} />}
            </Field>
          </>
        )}
        {mode.kind !== "edit" && (
          <>
            <Field label="Giảng viên" error={fe.teacher_id}>
              {(id, d) => (
                <Select id={id} aria-describedby={d} value={teacher} onChange={(e) => setTeacher(e.target.value)}>
                  <option value="">{mode.kind === "open" ? "Chưa gán" : "Giữ nguyên"}</option>
                  {teachers.data?.map((t) => (
                    <option key={t.id} value={t.id}>
                      {t.full_name}
                    </option>
                  ))}
                </Select>
              )}
            </Field>
            <fieldset className={s.group}>
              <legend>Trợ giảng</legend>
              {assistants.data?.length === 0 && <span className={s.sub}>Chưa có trợ giảng nào để chọn.</span>}
              {assistants.data?.map((t) => (
                <Checkbox
                  key={t.id}
                  label={t.full_name}
                  checked={tas.includes(t.id)}
                  onChange={(e) => {
                    setTasTouched(true);
                    setTas((prev) => (e.target.checked ? [...prev, t.id] : prev.filter((x) => x !== t.id)));
                  }}
                />
              ))}
              {fe.ta_ids && <span className={s.fieldError} role="alert">{fe.ta_ids}</span>}
              {mode.kind === "assign" && <span className={s.sub}>{tasTouched ? "Danh sách này thay toàn bộ trợ giảng hiện có." : "Không chọn gì thì giữ nguyên trợ giảng hiện có."}</span>}
            </fieldset>
          </>
        )}
        {archived && <InlineNotice tone="warning" compact>Lớp này đã được lưu trữ.</InlineNotice>}
        {general && (
          <InlineNotice tone="danger" compact action={mode.kind === "open" ? <Button size="sm" onClick={() => void open.retry().then((r) => r && onDone("Đã mở lớp."), () => undefined)}>Gửi lại</Button> : undefined}>
            {general}
          </InlineNotice>
        )}
        {conflict && (
          <InlineNotice
            tone="warning"
            compact
            action={
              <>
                <Button size="sm" onClick={() => { setVersion(conflict.current.version); setConflict(null); setOther({ pending: false }); }}>Giữ thay đổi của tôi</Button>
                <Button size="sm" variant="ghost" onClick={onClose}>Dùng bản mới</Button>
              </>
            }
          >
            Lớp này vừa được người khác sửa. Giữ thay đổi của bạn hay dùng bản mới?
          </InlineNotice>
        )}
        <div className={s.formActions}>
          <Button type="submit" variant="primary" loading={pending} disabled={mode.kind !== "assign" && (!classCode.trim() || !name.trim())}>
            {submitLabel}
          </Button>
          <Button variant="ghost" onClick={onClose} disabled={pending}>
            Huỷ
          </Button>
        </div>
      </form>
    </Section>
  );
}
