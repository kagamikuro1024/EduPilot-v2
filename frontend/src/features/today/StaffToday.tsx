"use client";

import { ApiErrorNotice } from "@/shared/data/ApiErrorNotice";
import { useSession } from "@/shared/session/session";
import { ActionList, ActionRow, Button, ButtonLink, EmptyState, InlineNotice, Page, PageHeader, Panel, Section } from "@/shared/ui";
import s from "./Today.module.css";
import { TodayActions } from "./TodayActions";
import { formatWhen } from "@/shared/lib/datetime";
import { classLabel, longDate } from "./format";
import { useToday, type StaffToday as Data } from "./todayApi";

/** "Hôm nay" của giảng viên / TA: "N việc cần xử lý hôm nay", việc xếp bằng luật cứng, buổi học sắp tới. Không hero số liệu, không thẻ KPI. */
export function StaffToday() {
  const { role } = useSession();
  const q = useToday<Data>();
  const d = q.data;
  const scope = "Tất cả lớp của tôi";
  return (
    <Page kind="read">
      <PageHeader title={d ? (d.count > 0 ? `${d.count} việc cần xử lý hôm nay` : "Không có việc cần xử lý hôm nay.") : "Hôm nay"} meta={q.isAll ? [longDate(new Date()), scope] : [longDate(new Date())]} />
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
          <div className={s.split}>
            <div className={s.mainCol}>
              {d.actions.length > 0 ? (
                <>
                  <TodayActions actions={d.actions} showClass={q.isAll} canDismiss={role === "teacher"} />
                  {d.count > d.actions.length && <p className={s.note}>Đang hiện {d.actions.length} việc đầu tiên.</p>}
                </>
              ) : (
                <Panel>
                  <EmptyState title="Bạn đã xử lý hết việc." />
                </Panel>
              )}
            </div>
            {(d.attention.length > 0 || d.upcoming.length > 0) && (
              <aside className={s.side} aria-label="Lớp học">
                {d.attention.length > 0 && (
                  <Section title="Lớp cần chú ý">
                    <ActionList label="Sinh viên cần chú ý">
                      {d.attention.map((a) => (
                        <ActionRow key={`${a.course.id}-${a.student.id}`} title={a.student.full_name} context={a.reason} meta={q.isAll ? classLabel(a.course) : undefined} href={`/students/${a.student.id}?course=${a.course.id}`} />
                      ))}
                    </ActionList>
                    <ButtonLink href={`/students?filter=attention${q.isAll ? "" : `&course=${d.attention[0].course.id}`}`} variant="ghost">
                      Xem tất cả ({d.attention_total})
                    </ButtonLink>
                  </Section>
                )}
                {d.upcoming.length > 0 && (
                  <Section title="Sắp tới">
                    <ActionList label="Buổi học sắp tới">
                      {d.upcoming.map((u) => (
                        <ActionRow key={`${u.course.id}-${u.at}`} title={u.title} context={`${formatWhen(u.at)}${u.place ? ` · ${u.place}` : ""}`} meta={q.isAll ? classLabel(u.course) : undefined} href="/calendar" />
                      ))}
                    </ActionList>
                  </Section>
                )}
              </aside>
            )}
          </div>
        </>
      )}
    </Page>
  );
}
