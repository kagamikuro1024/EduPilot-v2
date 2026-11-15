"use client";

import { useQueryClient } from "@tanstack/react-query";
import { Plus } from "lucide-react";
import Link from "next/link";
import { useSearchParams } from "next/navigation";
import { useState } from "react";
import { fmtLong } from "@/features/calendar/calendarApi";
import { useSession } from "@/shared/session/session";
import { Button, type Column, DataTable, EmptyState, PageState, Panel, Skeleton, StatusText } from "@/shared/ui";
import { AssignmentDrawer } from "./AssignmentDrawer";
import { assignmentsKey, useAssignments, type Assignment } from "./assignmentApi";
import s from "./Assignments.module.css";

/**
 * Tab `Bài tập` của `/grading` (US-P7-02 AC1): một `Panel` + `DataTable` phân trang con trỏ.
 * TEACHER tạo / sửa trong `Drawer`; TA chỉ xem.
 */
export function AssignmentsTab({ courseId, teacher, creating, onCreatingChange }: {
  courseId: string;
  teacher: boolean;
  creating: boolean;
  onCreatingChange: (v: boolean) => void;
}) {
  const { identity } = useSession();
  const qc = useQueryClient();
  const list = useAssignments(courseId);
  const [openId, setOpenId] = useState<string | null>(useSearchParams().get("open")); // liên kết sâu tới ngăn của một bài tập
  const open = creating ? "new" : openId;

  const close = () => {
    onCreatingChange(false);
    setOpenId(null);
  };
  const refresh = () => void qc.invalidateQueries({ queryKey: assignmentsKey(courseId) });

  const columns: Column<Assignment>[] = [
    {
      key: "title",
      header: "Tiêu đề",
      primary: true,
      // GV-BUG (j): dòng là LIÊN KẾT THẬT (mở tab mới, chuột giữa, Ctrl-click, bàn phím / trình đọc màn hình); bấm thường mở ngăn bên
      render: (a) => (
        <Link
          href={`/grading?tab=assignments&open=${a.id}`}
          onClick={(e) => {
            if (e.metaKey || e.ctrlKey || e.shiftKey || e.altKey || e.button !== 0) return;
            e.preventDefault();
            e.stopPropagation();
            setOpenId(a.id);
          }}
        >
          {a.title}
        </Link>
      ),
    },
    { key: "due", header: "Hạn", render: (a) => <span className={s.due}>{fmtLong(a.due_at)}</span> },
    {
      key: "state",
      header: "Trạng thái",
      part: "col-state",
      render: (a) => <StatusText tone={a.published_at ? "green" : "neutral"}>{a.published_at ? "Đã công bố" : "Nháp"}</StatusText>,
    },
    { key: "submitted", header: "Đã nộp / Tổng", align: "end", render: (a) => <span className={s.count}>{`${a.counts?.submitted ?? 0} / ${a.counts?.total ?? 0}`}</span> },
    { key: "graded", header: "Đã chấm / Đã công bố", align: "end", render: (a) => <span className={s.count}>{`${a.counts?.graded ?? 0} / ${a.counts?.published ?? 0}`}</span> },
  ];

  return (
    <>
      <PageState
        query={{ isPending: list.isPending, isError: list.isError, error: list.error, data: list.items, refetch: list.refetch }}
        isEmpty={(d) => (d as Assignment[]).length === 0}
        loading={<Panel><Skeleton lines={8} /></Panel>}
        empty={
          <Panel>
            <EmptyState
              title="Lớp chưa có bài tập."
              action={teacher ? <Button variant="primary" icon={<Plus aria-hidden />} onClick={() => onCreatingChange(true)}>Tạo bài tập</Button> : undefined}
            />
          </Panel>
        }
        showTechnical
      >
        <Panel>
          <DataTable
            caption="Bài tập của lớp"
            columns={columns}
            rows={list.items}
            rowKey={(a) => a.id}
            onRowClick={(a) => setOpenId(a.id)}
            activeKey={openId ?? undefined}
            rowAttrs={(a) => ({ "data-part": "assignment-row", "data-assignment-id": a.id })}
          />
          {list.hasNextPage && (
            <div className={s.more}>
              <Button loading={list.isFetchingNextPage} onClick={() => void list.fetchNextPage()}>
                Tải thêm
              </Button>
            </div>
          )}
        </Panel>
      </PageState>

      {open && (
        <AssignmentDrawer
          key={open}
          courseId={courseId}
          teacher={teacher}
          id={open}
          userId={identity?.sub}
          onClose={close}
          onSaved={(a) => {
            refresh();
            onCreatingChange(false);
            setOpenId(a.id);
          }}
          onDeleted={() => {
            refresh();
            close();
          }}
        />
      )}
    </>
  );
}
