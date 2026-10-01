"use client";

import { useState } from "react";
import { Plus } from "lucide-react";
import { COURSES, SUBJECT } from "@/mock/core";
import { COURSE_ADMIN_META, TEACHER_OPTIONS, courseLastActive } from "@/mock/system";
import { useSimNow } from "@/shared/state/clock";
import { useUndoLine } from "@/shared/lib/useUndoLine";
import { useDemoSlice } from "@/shared/state/demo";
import {
  Button,
  ConfirmIrreversible,
  DataTable,
  EmptyState,
  Field,
  Input,
  OverflowMenu,
  Page,
  PageHeader,
  PageState,
  Section,
  Select,
  Skeleton,
  StatusText,
  type Column,
} from "@/shared/ui";
import s from "./admin.module.css";

type NewCourse = { code: string; name: string; teacher: string; size: number; opened: string };
type Row = { code: string; name: string; teacher: string; size: number; state: "active" | "new" | "archived"; opened: string; lastActive: string };

const STATE_TEXT = {
  active: { tone: "green" as const, text: "Đang học" },
  new: { tone: "blue" as const, text: "Mới mở" },
  archived: { tone: "neutral" as const, text: "Đã lưu trữ" },
};

export function AdminCourses() {
  const [extra, setExtra] = useDemoSlice<NewCourse[]>("admin.courses", []);
  const [archived, setArchived] = useDemoSlice<string[]>("admin.archived", []);
  const [opening, setOpening] = useState(false);
  const [confirm, setConfirm] = useState<Row | null>(null);
  const [code, setCode] = useState("761989");
  const [teacher, setTeacher] = useState(TEACHER_OPTIONS[1]);
  const [size, setSize] = useState("30");
  const nowMs = useSimNow();
  const undo = useUndoLine();

  const rows: Row[] = [
    ...COURSES.map((c) => ({
      code: c.code,
      name: c.name,
      teacher: c.teacher,
      size: c.size,
      state: (archived.includes(c.code) ? "archived" : c.state) as Row["state"],
      opened: COURSE_ADMIN_META[c.id]?.opened ?? "—",
      lastActive: courseLastActive(c.id, nowMs),
    })),
    ...extra.map((c) => ({ ...c, state: (archived.includes(c.code) ? "archived" : "new") as Row["state"], lastActive: "Chưa có hoạt động" })),
  ];

  function openCourse() {
    const next: NewCourse = { code: code.trim(), name: SUBJECT.name, teacher, size: Number(size) || 30, opened: "29/10/2026" };
    setExtra((prev) => [...prev, next]);
    setOpening(false);
    setCode(String(Number(code) + 1));
    undo.push(`Đã mở lớp ${next.code} · Đã gửi thông báo phân công tới ${next.teacher}`, () => setExtra((prev) => prev.filter((c) => c.code !== next.code)));
  }

  function archive(row: Row) {
    setArchived((prev) => [...prev, row.code]);
    undo.push(`Đã lưu trữ lớp ${row.code}`, () => setArchived((prev) => prev.filter((c) => c !== row.code)));
  }

  const columns: Column<Row>[] = [
    { key: "code", header: "Mã lớp", frozen: true, render: (r) => <span className="ep-num">{r.code}</span> },
    {
      key: "name",
      header: "Môn học",
      render: (r) => (
        <>
          <span>{r.name}</span>
          <span className={s.sub}>Mở ngày {r.opened}</span>
        </>
      ),
    },
    { key: "teacher", header: "Giảng viên", render: (r) => r.teacher },
    { key: "size", header: "Sĩ số", align: "end", render: (r) => <span className="ep-num">{r.size}</span> },
    {
      key: "state",
      header: "Trạng thái",
      render: (r) => (
        <>
          <StatusText tone={STATE_TEXT[r.state].tone}>{STATE_TEXT[r.state].text}</StatusText>
          <span className={s.sub}>{r.lastActive}</span>
        </>
      ),
    },
    {
      key: "menu",
      header: "",
      width: "48px",
      align: "end",
      render: (r) =>
        r.state === "archived" ? null : <OverflowMenu items={[{ label: "Lưu trữ lớp", danger: true, onSelect: () => setConfirm(r) }]} label={`Hành động cho lớp ${r.code}`} />,
    },
  ];

  return (
    <Page width="wide">
      <PageHeader
        title="Lớp học"
        description="Mở lớp, phân công giảng viên và lưu trữ lớp đã kết thúc."
        meta={<span>{rows.filter((r) => r.state !== "archived").length} lớp đang chạy · học kỳ HK1 2026–2027</span>}
        actions={
          opening ? undefined : (
            <Button variant="primary" icon={<Plus aria-hidden />} onClick={() => setOpening(true)}>
              Mở lớp
            </Button>
          )
        }
      />

      <PageState
        loading={<Skeleton lines={6} />}
        empty={
          <EmptyState
            title="Chưa có lớp nào trong học kỳ này"
            action={
              <Button variant="primary" icon={<Plus aria-hidden />} onClick={() => setOpening(true)}>
                Mở lớp
              </Button>
            }
          >
            Mở lớp đầu tiên rồi phân công giảng viên; giảng viên nhận thông báo và tự mời sinh viên bằng mã tham gia.
          </EmptyState>
        }
        error={{
          problem: "Không tải được danh sách lớp.",
          recovery: "Các lớp vẫn chạy bình thường, chỉ danh sách này chưa đọc được. Thử lại sau ít phút.",
        }}
      >
        {opening && (
          <Section title="Mở lớp mới" description="Lớp mới nhận giảng viên ngay; sinh viên vào sau bằng mã tham gia.">
            <div className={s.form}>
              <Field label="Mã lớp" required>
                {(id) => <Input id={id} value={code} onChange={(e) => setCode(e.target.value)} inputMode="numeric" />}
              </Field>
              <Field label="Môn học">{(id) => <Input id={id} value={SUBJECT.name} readOnly />}</Field>
              <Field label="Giảng viên phụ trách" required helper="Người này nhận thông báo phân công ngay khi lớp được mở.">
                {(id) => (
                  <Select id={id} value={teacher} onChange={(e) => setTeacher(e.target.value)}>
                    {TEACHER_OPTIONS.map((t) => (
                      <option key={t} value={t}>
                        {t}
                      </option>
                    ))}
                  </Select>
                )}
              </Field>
              <Field label="Sĩ số dự kiến">{(id) => <Input id={id} value={size} onChange={(e) => setSize(e.target.value)} inputMode="numeric" />}</Field>
            </div>
            <div className={s.formActions}>
              <Button variant="primary" disabled={code.trim().length < 4 || rows.some((r) => r.code === code.trim())} onClick={openCourse}>
                Mở lớp
              </Button>
              <Button variant="ghost" onClick={() => setOpening(false)}>
                Huỷ
              </Button>
              {rows.some((r) => r.code === code.trim()) && <span className={s.sub}>Mã lớp {code.trim()} đã có rồi, chọn mã khác.</span>}
            </div>
          </Section>
        )}

        <div className={s.tableBlock}>
          <DataTable caption="Danh sách lớp học" columns={columns} rows={rows} rowKey={(r) => r.code} />
        </div>
        {undo.node}
      </PageState>

      <ConfirmIrreversible
        open={confirm !== null}
        onClose={() => setConfirm(null)}
        onConfirm={() => confirm && archive(confirm)}
        title={`Lưu trữ lớp ${confirm?.code ?? ""}?`}
        consequence={`${confirm?.size ?? 0} sinh viên vẫn xem được điểm và tài liệu, nhưng lớp thành chỉ đọc: mã tham gia tắt, việc nền dừng, không ai nộp bài hay hỏi thêm được nữa.`}
        confirmLabel="Lưu trữ lớp"
      />
    </Page>
  );
}
