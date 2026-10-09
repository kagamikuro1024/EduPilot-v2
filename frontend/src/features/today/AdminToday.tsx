"use client";

import { ApiErrorNotice } from "@/shared/data/ApiErrorNotice";
import { ActionList, Button, EmptyState, InlineNotice, Page, PageHeader, Panel, Section } from "@/shared/ui";
import { TodayActions } from "./TodayActions";
import { longDate } from "./format";
import { useToday, type AdminToday as Data } from "./todayApi";

/** "Hôm nay" của quản trị viên: việc của hệ thống (nhà cung cấp AI, ngân sách, lớp không giảng viên, lời mời hết hạn). */
export function AdminToday() {
  const q = useToday<Data>();
  const d = q.data;
  return (
    <Page>
      <PageHeader title={d ? (d.count > 0 ? `${d.count} việc cần xử lý hôm nay` : "Không có việc cần xử lý hôm nay.") : "Hôm nay"} description={`${longDate(new Date())} · việc của hệ thống`} />
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
            <Section title="Việc cần bạn xử lý" panel>
              <TodayActions actions={d.actions} showClass={false} canDismiss={false} />
            </Section>
          ) : (
            <Panel><EmptyState title="Hệ thống đang vận hành bình thường">Khi một nhà cung cấp AI lỗi, ngân sách sắp chạm trần hoặc một lớp không có giảng viên, việc sẽ xuất hiện ở đây.</EmptyState></Panel>
          )}
        </>
      )}
    </Page>
  );
}
