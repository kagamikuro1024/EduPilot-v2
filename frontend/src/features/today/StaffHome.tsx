"use client";

import { Check } from "lucide-react";
import Link from "next/link";
import { NOW, fmtLongDate } from "@/mock/core";
import { attendanceStats } from "@/mock/grades";
import { needsAttention, riskSentence, rosterOf } from "@/mock/roster";
import { staffTasks, upcoming } from "@/mock/staff";
import {
  ATTENDANCE_SEED,
  KEYS,
  MEMBERS_SEED,
  SCHEMES_SEED,
  type AttendanceState,
  type Bt03State,
  type MembersState,
  type SchemesState,
  type Ticket,
} from "@/mock/state";
import { BT03_SEED } from "@/mock/assess";
import { mergeTickets } from "@/mock/support";
import { useSession } from "@/shared/session/session";
import { useDemoSlice } from "@/shared/state/demo";
import {
  ActionList,
  ActionRow,
  ButtonLink,
  DefinitionList,
  EmptyState,
  Page,
  PageHeader,
  PageState,
  PrivateMark,
  Section,
  Skeleton,
  StatusText,
  useRouteState,
} from "@/shared/ui";
import s from "./StaffHome.module.css";

/** "Hôm nay" của giảng viên / trợ giảng: việc cần người quyết định, không phải bảng số liệu (DESIGN §14.1). */
export function StaffHome() {
  const { role, courses, course } = useSession();
  const [stored] = useDemoSlice<Ticket[]>(KEYS.tickets, []);
  const [attendance] = useDemoSlice<AttendanceState>(KEYS.attendance, ATTENDANCE_SEED);
  const [members] = useDemoSlice<MembersState>(KEYS.members, MEMBERS_SEED);
  const [bt03] = useDemoSlice<Bt03State>(KEYS.bt03, BT03_SEED);
  const [schemes] = useDemoSlice<SchemesState>(KEYS.schemes, SCHEMES_SEED);
  const routeState = useRouteState();

  const staffRole = role === "ta" ? "ta" : "teacher";
  const courseIds = courses.map((c) => c.id);
  const tasks = staffTasks({ role: staffRole, courseIds, tickets: mergeTickets(stored), attendance, members, bt03, schemes });

  const roster = courses
    .flatMap((c) => rosterOf(c.id, members))
    .filter((st, i, all) => all.findIndex((x) => x.id === st.id) === i);
  const watch = needsAttention(roster);

  const emptyTasks = (
    <EmptyState title="Không còn việc cần bạn quyết định" action={<ButtonLink href="/students">Xem sinh viên cần chú ý</ButtonLink>}>
      Khi có câu hỏi chờ, buổi học đang diễn ra hoặc bài chấm cần xem kỹ, việc sẽ xuất hiện ở đây.
    </EmptyState>
  );

  return (
    <Page>
      <PageHeader
        title="Hôm nay"
        description={`${fmtLongDate(NOW)} · tuần ${course.week} · ${courses.map((c) => c.code).join(" và ")}`}
      />
      <PageState state={routeState} loading={<Skeleton lines={8} />} empty={emptyTasks}>
        <Section title={`${tasks.length} việc cần xử lý hôm nay`} description="Xếp theo mức khẩn và hệ quả; việc đã xử lý sẽ rời khỏi danh sách.">
          {tasks.length === 0 ? (
            emptyTasks
          ) : (
            <ActionList label="Việc cần xử lý hôm nay">
              {tasks.map((t) => (
                <ActionRow
                  key={t.id}
                  tone={t.tone}
                  href={t.steps ? undefined : t.href}
                  redThread
                  title={t.title}
                  context={
                    t.steps ? (
                      <>
                        {t.context}
                        <ol className={s.steps}>
                          {t.steps.map((step) => (
                            <li key={step.label} className={step.done ? s.done : undefined}>
                              {step.done && <Check aria-hidden />}
                              <Link href={step.href} className="ep-link">
                                {step.label}
                              </Link>
                            </li>
                          ))}
                        </ol>
                      </>
                    ) : (
                      t.context
                    )
                  }
                  meta={t.course}
                  action={
                    <ButtonLink href={t.href} size="sm" variant={t.urgent ? "primary" : "secondary"}>
                      {t.actionLabel}
                    </ButtonLink>
                  }
                />
              ))}
            </ActionList>
          )}
        </Section>

        {watch.length > 0 && (
          <Section title="Lớp cần chú ý" action={<PrivateMark />}>
            <ActionList label="Sinh viên cần chú ý">
              {watch.map((st) => (
                <ActionRow
                  key={st.id}
                  tone={st.risk === "high" ? "red" : "amber"}
                  href={`/students/${st.id}`}
                  title={st.name}
                  context={riskSentence(st, attendanceStats(st, attendance)) ?? undefined}
                  meta={`${st.code} · ${st.courseIds.join(", ")}`}
                />
              ))}
            </ActionList>
          </Section>
        )}

        <Section title="Sắp tới">
          <DefinitionList
            items={upcoming(course.id).map((u) => ({
              term: u.now ? <StatusText tone="red">{u.when}</StatusText> : u.when,
              value: (
                <>
                  {u.title}
                  <span className={s.note}>{u.note}</span>
                </>
              ),
            }))}
          />
        </Section>
      </PageState>
    </Page>
  );
}

