"use client";

import { ApiErrorNotice } from "@/shared/data/ApiErrorNotice";
import { ActionList, Button, EmptyState, InlineNotice, Page, PageHeader, Panel } from "@/shared/ui";
import { TodayActions } from "./TodayActions";
import { } from "./format";
import { useToday, type AdminToday as Data } from "./todayApi";

/** "Hôm nay" của quản trị viên: việc của hệ thống (nhà cung cấp AI, ngân sách, lớp không giảng viên, lời mời hết hạn). */
export function AdminToday() {
  const q = useToday<Data>();
  const d = q.data;
  return (
    <Page kind="read">
      <PageHeader title={d ? (d.count > 0 ? `${d.count} việc cần xử lý hôm nay` : "Không có việc cần xử lý hôm nay.") : "Hôm nay"} />
      {!d ? (
        q.isError ? (
          <ApiErrorNotice error={q.error} showTechnical onRetry={() => void q.refetch()} title="Chưa tải được việc hôm nay. Dữ liệu của bạn không bị ảnh hưởng." />
        ) : (
          <Panel><ActionList loading={3} label="Đang tải việc" /></Panel>
        )
      ) : (
        <>
          {q.isError && (
            <InlineNotice tone="warning" compact action={<Button size="sm" onClick={() => void q.refetch()}>Thử lại</Button>}>
              Chưa làm mới được. Đang hiện dữ liệu lần trước.
            </InlineNotice>
          )}
          {d.actions.length > 0 ? (
            <TodayActions actions={d.actions} showClass={false} canDismiss={false} />
          ) : (
            <Panel><EmptyState title="Hệ thống đang vận hành bình thường" /></Panel>
          )}
        </>
      )}
    </Page>
  );
}
