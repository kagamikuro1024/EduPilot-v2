"use client";

import { useCursorList } from "@/shared/data";
import { timeAgo } from "@/shared/lib/timeAgo";
import { Button, type Column, DataTable, EmptyState, PageState, Section, Skeleton } from "@/shared/ui";

// Hợp đồng thật: backend-go/api/openapi.yaml (`GET /admin/observability/failed-mail`). Không thân thư, không payload; người nhận đã che.
type FailedMailRow = { id: string; template: string; to_masked: string; attempts: number; last_error: string | null; updated_at: string };

const TEMPLATE: Record<string, string> = {
  escalation_answered: "Trả lời của giảng viên", outreach_message: "Tin nhắn của giảng viên", ticket_reminder: "Nhắc câu hỏi quá hạn", ticket_new: "Câu hỏi mới",
  thread_reply: "Trả lời trên Threads", verify_email: "Xác minh email", reset_password: "Đặt lại mật khẩu", invite_staff: "Lời mời", invite_student: "Lời mời vào lớp",
  email_exists: "Email đã đăng ký", password_changed: "Đổi mật khẩu", account_locked: "Khoá tài khoản", reminder: "Nhắc lịch",
};

const columns: Column<FailedMailRow>[] = [
  { key: "template", header: "Thư", render: (r) => TEMPLATE[r.template] ?? r.template },
  { key: "to", header: "Người nhận", render: (r) => <span className="ep-num">{r.to_masked}</span> },
  { key: "attempts", header: "Số lần thử", align: "end", render: (r) => <span className="ep-num">{r.attempts}</span> },
  { key: "error", header: "Lỗi", render: (r) => r.last_error ?? "—" },
  { key: "at", header: "Lúc", render: (r) => timeAgo(r.updated_at) },
];

/** Thư chưa gửi được (chỉ Quản trị viên): thư DEAD, mới nhất trước. Rỗng → một dòng, không nút. */
export function FailedMail() {
  const list = useCursorList<FailedMailRow>(["admin", "failed-mail"], "/admin/observability/failed-mail", { limit: 30 });
  return (
    <Section panel title="Thư chưa gửi được">
      <PageState query={list} isEmpty={() => list.items.length === 0} loading={<Skeleton lines={3} />} empty={<EmptyState title="Không có thư lỗi." />}>
        <DataTable caption="Thư chưa gửi được, mới nhất trước" columns={columns} rows={list.items} rowKey={(r) => r.id} dense />
        {list.hasNextPage && <Button onClick={() => void list.fetchNextPage()} disabled={list.isFetchingNextPage}>Xem thêm</Button>}
      </PageState>
    </Section>
  );
}
