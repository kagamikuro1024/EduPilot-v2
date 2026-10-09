"use client";

import { useQueryClient } from "@tanstack/react-query";
import { Plus } from "lucide-react";
import { useState } from "react";
import { ApiError, apiClient, useCursorList } from "@/shared/data";
import { Button, type Column, ConfirmIrreversible, DataTable, EmptyState, InlineNotice, OverflowMenu, Page, PageHeader, PageState, Panel, StatusText, UndoLine } from "@/shared/ui";
import s from "./admin.module.css";
import { CoursePanel, type PanelMode } from "./courses/CoursePanel";
import { COURSES_KEY, sizeText, type AdminCourse } from "./courses/api";

/** Lớp học của toàn hệ thống (Admin): mở lớp, gán giảng viên / trợ giảng, sửa, lưu trữ. Dữ liệu thật qua `/admin/courses`; Admin không thêm được sinh viên. */
export function AdminCourses() {
  const qc = useQueryClient();
  const [panel, setPanel] = useState<PanelMode | null>(null);
  const [line, setLine] = useState<string | null>(null);
  const [confirm, setConfirm] = useState<AdminCourse | null>(null);
  const [archiving, setArchiving] = useState<{ loading: boolean; error?: string }>({ loading: false });

  const list = useCursorList<AdminCourse>(COURSES_KEY, "/admin/courses", { limit: 30 });
  const refresh = () => qc.invalidateQueries({ queryKey: COURSES_KEY });
  const running = list.items.filter((c) => c.status === "ACTIVE").length;

  async function archive(c: AdminCourse) {
    setArchiving({ loading: true });
    try {
      await apiClient.post(`/admin/courses/${c.id}/archive`);
      setArchiving({ loading: false });
      setConfirm(null);
      setLine(`Đã lưu trữ lớp ${c.class_code}.`);
      await refresh();
    } catch (e) {
      setArchiving({ loading: false, error: e instanceof ApiError ? (e.code === "COURSE_ARCHIVED" ? "Lớp này đã được lưu trữ." : e.userMessage) : "Chưa lưu trữ được. Hãy thử lại." });
    }
  }

  const columns: Column<AdminCourse>[] = [
    { key: "code", header: "Mã lớp", frozen: true, render: (c) => <span className="ep-num">{c.class_code}</span> },
    {
      key: "subject",
      header: "Học phần",
      render: (c) => (
        <>
          <span>{c.subject_code}</span>
          <span className={s.sub}>{c.name} · {c.semester}</span>
        </>
      ),
    },
    { key: "teacher", header: "Giảng viên", render: (c) => c.teacher?.full_name ?? <span className={s.sub}>Chưa gán</span> },
    { key: "size", header: "Sĩ số", align: "end", render: (c) => <span className="ep-num">{sizeText(c)}</span> },
    {
      key: "state",
      header: "Trạng thái",
      render: (c) => (
        <>
          <StatusText tone={c.status === "ACTIVE" ? "green" : "neutral"}>{c.status === "ACTIVE" ? "Đang học" : "Đã lưu trữ"}</StatusText>
          {c.students_pending > 0 && <span className={s.sub}>{c.students_pending} chờ duyệt</span>}
        </>
      ),
    },
    {
      key: "menu",
      header: "",
      width: "48px",
      align: "end",
      render: (c) =>
        c.status === "ARCHIVED" ? null : (
          <OverflowMenu
            label={`Hành động cho lớp ${c.class_code}`}
            items={[
              { label: "Sửa", onSelect: () => { setLine(null); setPanel({ kind: "edit", course: c }); } },
              { label: "Gán lại", onSelect: () => { setLine(null); setPanel({ kind: "assign", course: c }); } },
              { label: "Lưu trữ lớp", danger: true, onSelect: () => { setArchiving({ loading: false }); setConfirm(c); } },
            ]}
          />
        ),
    },
  ];

  return (
    <Page width="wide">
      <PageHeader
        title="Lớp học"
        description="Mở lớp, phân công giảng viên và lưu trữ lớp đã kết thúc."
        meta={list.isPending ? undefined : <span>{running} lớp đang chạy</span>}
        actions={
          panel ? undefined : (
            <Button variant="primary" icon={<Plus aria-hidden />} onClick={() => { setLine(null); setPanel({ kind: "open" }); }}>
              Mở lớp
            </Button>
          )
        }
      />

      {panel && (
        <CoursePanel
          key={panel.kind === "open" ? "open" : `${panel.kind}-${panel.course.id}`}
          mode={panel}
          onClose={() => setPanel(null)}
          onDone={(text) => {
            setPanel(null);
            setLine(text);
            void refresh();
          }}
        />
      )}
      {line && <UndoLine key={line} message={line} onDone={() => setLine(null)} />}
      {list.isError && list.items.length > 0 && <InlineNotice tone="danger" compact>Chưa làm mới được danh sách lớp.</InlineNotice>}

      <PageState query={{ isPending: list.isPending, isError: list.isError, error: list.error, data: list.items, refetch: list.refetch }} showTechnical>
        <Panel>
          <DataTable
            caption="Danh sách lớp học"
            columns={columns}
            rows={list.items}
            rowKey={(c) => c.id}
            empty={
              <EmptyState title="Chưa có lớp nào. Mở lớp đầu tiên." action={panel ? undefined : <Button variant="primary" icon={<Plus aria-hidden />} onClick={() => setPanel({ kind: "open" })}>Mở lớp</Button>}>
                Mở lớp rồi gán giảng viên; giảng viên nhận thông báo kèm mã tham gia và tự mời sinh viên.
              </EmptyState>
            }
            pagination={{ nextCursor: list.hasNextPage ? "next" : null, onLoadMore: () => void list.fetchNextPage(), loading: list.isFetchingNextPage }}
          />
        </Panel>
      </PageState>

      <ConfirmIrreversible
        open={confirm !== null}
        onClose={() => setConfirm(null)}
        onConfirm={() => confirm && void archive(confirm)}
        title={`Lưu trữ lớp ${confirm?.class_code ?? ""}?`}
        consequence={`Lưu trữ lớp ${confirm?.class_code ?? ""}: ${confirm?.students_active ?? 0} sinh viên chỉ còn quyền đọc; mã tham gia ngừng hoạt động.`}
        confirmLabel="Lưu trữ lớp"
        loading={archiving.loading}
        error={archiving.error}
      />
    </Page>
  );
}
