"use client";

import { ApiErrorNotice } from "@/shared/data/ApiErrorNotice";
import { useSession } from "@/shared/session/session";
import { ActionList, ActionRow, Button, EmptyState, InlineNotice, Page, PageHeader, Panel, Section } from "@/shared/ui";
import s from "./Today.module.css";
import { TodayActions } from "./TodayActions";
import { classLabel, hhmm, longDate, shortDay } from "./format";
import { useToday, type StaffToday as Data } from "./todayApi";

/** "Hôm nay" của giảng viên / TA: "N việc cần xử lý hôm nay", việc xếp bằng luật cứng, buổi học sắp tới. Không hero số liệu, không thẻ KPI. */
export function StaffToday() {
  const { role, realCourses, realCourseId } = useSession();
  const q = useToday<Data>();
  const d = q.data;
  const scope = q.isAll ? "Tất cả lớp của tôi" : `Lớp ${realCourses?.find((c) => c.id === realCourseId)?.class_code ?? ""}`;
  return (
    <Page>
      <PageHeader title={d ? (d.count > 0 ? `${d.count} việc cần xử lý hôm nay` : "Không có việc cần xử lý hôm nay.") : "Hôm nay"} description={`${longDate(new Date())} · ${scope}`} />
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
            <Section title="Việc cần xử lý" panel>
              <TodayActions actions={d.actions} showClass={q.isAll} canDismiss={role === "teacher"} />
              {d.count > d.actions.length && <p className={s.note}>Đang hiện {d.actions.length} việc đầu tiên.</p>}
            </Section>
          ) : (
            <Panel>
              <EmptyState title="Bạn đã xử lý hết việc.">Khi có yêu cầu vào lớp hoặc việc mới, chúng sẽ hiện ở đây.</EmptyState>
            </Panel>
          )}
          {d.upcoming.length > 0 && (
            <Section title="Sắp tới" panel>
              <ActionList label="Buổi học sắp tới">
                {d.upcoming.map((u) => (
                  <ActionRow key={`${u.course.id}-${u.at}`} title={u.title} context={`${shortDay(u.at)} · ${hhmm(u.at)}${u.place ? ` · ${u.place}` : ""}`} meta={q.isAll ? classLabel(u.course) : undefined} href="/calendar" />
                ))}
              </ActionList>
            </Section>
          )}
        </>
      )}
    </Page>
  );
}
