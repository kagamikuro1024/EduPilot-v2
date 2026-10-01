"use client";

import { useState } from "react";
import { Mail, Search } from "lucide-react";
import { COURSES, STAFF, STUDENTS } from "@/mock/core";
import { useUndoLine } from "@/shared/lib/useUndoLine";
import { useDemoSlice } from "@/shared/state/demo";
import {
  Button,
  DataTable,
  EmptyState,
  Field,
  InlineNotice,
  Input,
  Page,
  PageHeader,
  PageState,
  Section,
  SegmentedControl,
  Skeleton,
  StatusText,
  Toolbar,
  type Column,
} from "@/shared/ui";
import s from "./admin.module.css";

type RoleFilter = "all" | "teacher" | "ta" | "admin" | "student";
type Row = { id: string; name: string; email: string; role: Exclude<RoleFilter, "all">; detail: string; invited?: boolean };

const ROLE_TEXT: Record<Exclude<RoleFilter, "all">, string> = { teacher: "Giảng viên", ta: "Trợ giảng", admin: "Quản trị viên", student: "Sinh viên" };

export function AdminUsers() {
  const [locked, setLocked] = useDemoSlice<string[]>("admin.locked", []);
  const [invites, setInvites] = useDemoSlice<string[]>("admin.invites", []);
  const [filter, setFilter] = useState<RoleFilter>("all");
  const [query, setQuery] = useState("");
  const [inviting, setInviting] = useState(false);
  const [email, setEmail] = useState("");
  const undo = useUndoLine();

  const all: Row[] = [
    { id: STAFF.teacher.id, name: `${STAFF.teacher.title} ${STAFF.teacher.name}`, email: STAFF.teacher.email, role: "teacher", detail: "Phụ trách 761987, 761988" },
    { id: STAFF.ta.id, name: STAFF.ta.name, email: STAFF.ta.email, role: "ta", detail: `Trợ giảng lớp ${COURSES[0].code}` },
    { id: STAFF.admin.id, name: STAFF.admin.name, email: STAFF.admin.email, role: "admin", detail: "Quản trị hệ thống" },
    ...invites.map((e) => ({ id: `inv-${e}`, name: e.split("@")[0], email: e, role: "teacher" as const, detail: "Đã mời, chờ nhận lời mời", invited: true })),
    ...STUDENTS.map((st) => ({
      id: st.id,
      name: st.name,
      email: st.email,
      role: "student" as const,
      detail: st.courseIds.length > 0 ? `Lớp ${st.courseIds.join(", ")} · ${st.code}` : `Chưa vào lớp nào · ${st.code}`,
    })),
  ];

  const q = query.trim().toLowerCase();
  const rows = all.filter((r) => (filter === "all" || r.role === filter) && (q === "" || r.name.toLowerCase().includes(q) || r.email.toLowerCase().includes(q) || r.detail.toLowerCase().includes(q)));

  function toggleLock(row: Row) {
    const isLocked = locked.includes(row.id);
    setLocked((prev) => (isLocked ? prev.filter((id) => id !== row.id) : [...prev, row.id]));
    undo.push(isLocked ? `Đã mở khoá tài khoản ${row.name}` : `Đã khoá tài khoản ${row.name}`, () =>
      setLocked((prev) => (isLocked ? [...prev, row.id] : prev.filter((id) => id !== row.id))),
    );
  }

  function invite() {
    const addr = email.trim();
    setInvites((prev) => [...prev, addr]);
    setInviting(false);
    setEmail("");
    undo.push(`Đã gửi link mời tới ${addr}, hạn 72 giờ`, () => setInvites((prev) => prev.filter((e) => e !== addr)));
  }

  const columns: Column<Row>[] = [
    {
      key: "name",
      header: "Họ tên",
      frozen: true,
      render: (r) => (
        <>
          <span className={s.name}>{r.name}</span>
          <span className={s.sub}>{r.email}</span>
        </>
      ),
    },
    { key: "role", header: "Vai trò", render: (r) => ROLE_TEXT[r.role] },
    { key: "detail", header: "Thuộc lớp", hideOnMobile: true, render: (r) => <span className={s.sub}>{r.detail}</span> },
    {
      key: "state",
      header: "Trạng thái",
      render: (r) =>
        r.invited ? (
          <StatusText tone="amber">Chờ nhận lời mời</StatusText>
        ) : locked.includes(r.id) ? (
          <StatusText tone="red">Đã khoá</StatusText>
        ) : (
          <StatusText tone="green">Đang dùng</StatusText>
        ),
    },
    {
      key: "action",
      header: "",
      align: "end",
      width: "120px",
      render: (r) =>
        r.invited ? null : (
          <Button size="sm" variant="ghost" onClick={() => toggleLock(r)}>
            {locked.includes(r.id) ? "Mở khoá" : "Khoá"}
          </Button>
        ),
    },
  ];

  return (
    <Page width="wide">
      <PageHeader
        title="Người dùng"
        description="Tài khoản giảng viên, trợ giảng, quản trị viên và sinh viên của toàn hệ thống."
        meta={
          <>
            <span>{all.length} tài khoản</span>
            <span>{STUDENTS.length} sinh viên</span>
            {locked.length > 0 && <span>{locked.length} tài khoản đang bị khoá</span>}
          </>
        }
        actions={
          inviting ? undefined : (
            <Button variant="primary" icon={<Mail aria-hidden />} onClick={() => setInviting(true)}>
              Mời giảng viên
            </Button>
          )
        }
      />

      <PageState
        loading={<Skeleton lines={8} />}
        empty={<EmptyState title="Chưa có tài khoản nào ngoài bạn">Mời giảng viên đầu tiên bằng email của trường; sinh viên tự vào lớp bằng mã tham gia, không cần tạo tài khoản sẵn.</EmptyState>}
        error={{ problem: "Không tải được danh sách người dùng.", recovery: "Mọi người vẫn đăng nhập bình thường. Thử lại sau ít phút." }}
      >
        {inviting && (
          <Section title="Mời giảng viên" description="Người được mời nhận một link đặt mật khẩu, dùng trong 72 giờ.">
            <div className={s.form}>
              <Field label="Email của trường" required helper="Chỉ nhận email tên miền của trường.">
                {(id, describedBy) => (
                  <Input id={id} aria-describedby={describedBy} type="email" value={email} onChange={(e) => setEmail(e.target.value)} placeholder="ten.ho@edupilot.test" autoComplete="off" />
                )}
              </Field>
            </div>
            <div className={s.formActions}>
              <Button variant="primary" disabled={!email.includes("@") || email.trim().length < 6} onClick={invite}>
                Gửi lời mời
              </Button>
              <Button variant="ghost" onClick={() => setInviting(false)}>
                Huỷ
              </Button>
            </div>
          </Section>
        )}

        <Toolbar
          end={
            <Field label="Tìm người dùng" className={s.searchField}>
              {(id) => (
                <span className={s.search}>
                  <Search aria-hidden />
                  <Input id={id} value={query} onChange={(e) => setQuery(e.target.value)} placeholder="Tên, email hoặc mã sinh viên" />
                </span>
              )}
            </Field>
          }
        >
          <SegmentedControl
            label="Lọc theo vai trò"
            value={filter}
            onChange={setFilter}
            options={[
              { value: "all", label: "Tất cả", count: all.length },
              { value: "teacher", label: "Giảng viên" },
              { value: "ta", label: "Trợ giảng" },
              { value: "admin", label: "Quản trị" },
              { value: "student", label: "Sinh viên", count: STUDENTS.length },
            ]}
          />
        </Toolbar>

        <InlineNotice tone="info" compact>
          Không tạo tài khoản sinh viên ở đây: sinh viên tự đăng ký rồi vào lớp bằng mã tham gia, giảng viên duyệt nếu lớp bật duyệt.
        </InlineNotice>

        <DataTable
          caption="Danh sách người dùng"
          columns={columns}
          rows={rows}
          rowKey={(r) => r.id}
          dense
          empty={
            <EmptyState title="Không có ai khớp bộ lọc này">
              Thử bỏ bớt từ khoá “{query}” hoặc chọn lại vai trò. Đang lọc: {filter === "all" ? "tất cả vai trò" : ROLE_TEXT[filter]}.
            </EmptyState>
          }
        />
        {undo.node}
      </PageState>
    </Page>
  );
}
