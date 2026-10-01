"use client";

import { NOW, fmtLongDate, fmtTime } from "@/mock/core";
import { STATUS_STRIP, adminTasks } from "@/mock/system";
import { StatusStrip } from "@/features/observability/ObservabilityScreen";
import { useSimNow } from "@/shared/state/clock";
import { ActionList, ActionRow, ButtonLink, EmptyState, Page, PageHeader, PageState, Section, Skeleton } from "@/shared/ui";

/** "Hôm nay" của quản trị viên: việc của hệ thống, không phải việc của lớp (FLOWS F14). */
export function AdminHome() {
  const tasks = adminTasks(useSimNow());
  return (
    <Page>
      <PageHeader
        title="Hôm nay"
        description="Việc của hệ thống cần bạn quyết, xếp theo mức ảnh hưởng tới lớp đang học."
        meta={
          <>
            <span>
              {fmtLongDate(NOW)} · {fmtTime(NOW)}
            </span>
            <span>{tasks.length} việc đang chờ</span>
          </>
        }
      />

      <PageState
        loading={<Skeleton lines={6} />}
        empty={
          <EmptyState title="Hệ thống đang vận hành bình thường">
            Không có việc nào cần bạn xử lý. Khi một nhà cung cấp model hỏng, ngân sách sắp chạm trần, việc nền kẹt hoặc một lớp không có giảng viên hoạt động, việc sẽ xuất hiện ở đây.
          </EmptyState>
        }
        error={{
          problem: "Không tải được danh sách việc của hệ thống.",
          recovery: "Các lớp vẫn chạy bình thường. Thử lại, hoặc mở Quan sát AI để xem trạng thái trực tiếp.",
        }}
      >
        <Section title="Việc cần bạn xử lý">
          <ActionList label="Việc của quản trị viên">
            {tasks.map((t) => (
              <ActionRow key={t.id} tone={t.tone} href={t.href} redThread title={t.title} context={t.context} meta={t.meta} />
            ))}
          </ActionList>
        </Section>

        <Section
          title="Hệ thống hôm nay"
          description="Số liệu tính từ 00:00."
          action={
            <ButtonLink href="/observability" variant="text" size="sm">
              Mở Quan sát AI
            </ButtonLink>
          }
        >
          <StatusStrip items={STATUS_STRIP} />
        </Section>
      </PageState>
    </Page>
  );
}
