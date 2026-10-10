"use client";

import { ChevronLeft, ChevronRight } from "lucide-react";
import { useMemo, useState, useSyncExternalStore } from "react";
import { EVENT_LABEL, WEEK_DAY_SHORT, eventsFor, isNow, shortLabel, startOfWeek, type CalEvent } from "@/mock/calendar";
import { NOW, fmtLongDate, fmtShortDate, fmtTime } from "@/mock/core";
import { QUIZ_SEED, quizKey, type QuizState } from "@/mock/practice";
import { KEYS, type CalendarExtra } from "@/mock/state";
import { useUndoLine } from "@/shared/lib/useUndoLine";
import { useSession } from "@/shared/session/session";
import { useDemoSlice } from "@/shared/state/demo";
import { ActionList, ActionRow, Button, ButtonLink, EmptyState, IconButton, Page, PageHeader, PageState, Panel, SegmentedControl, Skeleton, StatusText, Toolbar } from "@/shared/ui";
import s from "./Calendar.module.css";

type View = "week" | "month" | "list";
const DAY = 86_400_000;
const MOBILE = "(max-width: 719px)";

function subscribeNarrow(cb: () => void) {
  const mql = window.matchMedia(MOBILE);
  mql.addEventListener("change", cb);
  return () => mql.removeEventListener("change", cb);
}

function getNarrow() {
  return window.matchMedia(MOBILE).matches;
}

/** Lịch: thấy việc kế tiếp (DESIGN §14.16). Dùng chung cho sinh viên và giảng viên / trợ giảng. */
export function DemoCalendar() {
  const { role, course, studentId } = useSession();
  const [extras] = useDemoSlice<CalendarExtra[]>(KEYS.calendarExtras, []);
  const [quiz] = useDemoSlice<QuizState>(quizKey(studentId), QUIZ_SEED);
  const [chosenView, setChosenView] = useState<View | null>(null);
  const [offset, setOffset] = useState(0);
  const undo = useUndoLine();
  const isStudent = role === "student";
  /** 375 px mặc định Danh sách; người dùng chọn kiểu khác thì giữ lựa chọn đó. */
  const narrow = useSyncExternalStore(subscribeNarrow, getNarrow, () => false);
  const view = chosenView ?? (narrow ? "list" : "week");

  const events = useMemo(() => eventsFor(course.id, extras, quiz.status === "submitted"), [course.id, extras, quiz.status]);

  const weekStart = new Date(startOfWeek(NOW).getTime() + offset * 7 * DAY);
  const monthAnchor = new Date(NOW.getFullYear(), NOW.getMonth() + offset, 1);

  const inWeek = events.filter((e) => e.start >= weekStart && e.start.getTime() < weekStart.getTime() + 7 * DAY);
  const upcoming = events.filter((e) => e.start.getTime() >= NOW.getTime() - 3 * 3600_000).slice(0, 12);

  const title = view === "month" ? `Tháng ${monthAnchor.getMonth() + 1} năm ${monthAnchor.getFullYear()}` : view === "week" ? `Tuần ${fmtShortDate(weekStart)} – ${fmtShortDate(new Date(weekStart.getTime() + 6 * DAY))}` : "Sắp tới";

  return (
    <Page width="wide">
      <PageHeader
        title="Lịch"
        description={`${course.label} · ${course.schedule} · ${course.room}`}
        actions={
          <Button onClick={() => undo.push("Đã sao chép link lịch (mô phỏng)")}>Thêm vào lịch</Button>
        }
      />
      <Toolbar
        end={
          view === "list" ? undefined : (
            <span className={s.nav}>
              <IconButton label="Khoảng trước" size="sm" onClick={() => setOffset((o) => o - 1)}>
                <ChevronLeft aria-hidden />
              </IconButton>
              <Button size="sm" onClick={() => setOffset(0)}>
                Hôm nay
              </Button>
              <IconButton label="Khoảng sau" size="sm" onClick={() => setOffset((o) => o + 1)}>
                <ChevronRight aria-hidden />
              </IconButton>
            </span>
          )
        }
      >
        <SegmentedControl
          label="Kiểu xem lịch"
          value={view}
          onChange={(v) => {
            setChosenView(v);
            setOffset(0);
          }}
          options={[
            { value: "week", label: "Tuần" },
            { value: "month", label: "Tháng" },
            { value: "list", label: "Danh sách" },
          ]}
        />
        <span className={s.range}>{title}</span>
      </Toolbar>
      {undo.node}

      <PageState
        loading={<Skeleton lines={8} />}
        empty={
          <Panel><EmptyState title="Lớp này chưa có sự kiện nào" action={<ButtonLink href="/" variant="primary">Về Hôm nay</ButtonLink>}>
            Buổi học, hạn nộp và lịch thi sẽ hiện ở đây khi giảng viên tạo.
          </EmptyState></Panel>
        }
      >
        {view === "week" && <WeekView start={weekStart} events={inWeek} isStudent={isStudent} />}
        {view === "month" && <MonthView anchor={monthAnchor} events={events} />}
        {view === "list" && <ListView events={upcoming} isStudent={isStudent} />}
      </PageState>
    </Page>
  );
}

function EventLine({ e, isStudent }: { e: CalEvent; isStudent: boolean }) {
  return (
    <span className={[s.event, s[e.kind]].join(" ")}>
      <span className={s.eventTime}>{e.kind === "deadline" ? `Hạn ${fmtTime(e.start)}` : fmtTime(e.start)}</span>
      <span className={s.eventTitle} data-part="cal-event">{e.title}</span>
      {isNow(e) && <StatusText tone="green">Đang diễn ra</StatusText>}
      {isStudent && e.studentNote && <span className={s.eventNote}>{e.studentNote}</span>}
    </span>
  );
}

function WeekView({ start, events, isStudent }: { start: Date; events: CalEvent[]; isStudent: boolean }) {
  const days = Array.from({ length: 7 }, (_, i) => new Date(start.getTime() + i * DAY));
  if (events.length === 0) {
    return (
      <Panel><EmptyState title="Tuần này không có sự kiện nào" action={<ButtonLink href="/practice" variant="primary">Luyện đề</ButtonLink>}>
        Không có buổi học hay hạn nộp trong tuần. Dùng thời gian trống để ôn chủ đề bạn hay sai.
      </EmptyState></Panel>
    );
  }
  return (
    <Panel>
    <div className={s.week}>
      {days.map((d, i) => {
        const items = events.filter((e) => e.start.toDateString() === d.toDateString());
        const today = d.toDateString() === NOW.toDateString();
        return (
          <section key={d.toISOString()} className={[s.day, today ? s.today : ""].join(" ")}>
            <h3 className={s.dayHead}>
              <span className={s.dayName}>{WEEK_DAY_SHORT[i]}</span>
              <span className={s.dayNum}>{fmtShortDate(d)}</span>
            </h3>
            {items.length === 0 ? (
              <p className={s.dayEmpty}>—</p>
            ) : (
              items.map((e) => <EventLine key={e.id} e={e} isStudent={isStudent} />)
            )}
          </section>
        );
      })}
    </div>
    </Panel>
  );
}

function MonthView({ anchor, events }: { anchor: Date; events: CalEvent[] }) {
  const first = startOfWeek(new Date(anchor));
  const cells = Array.from({ length: 42 }, (_, i) => new Date(first.getTime() + i * DAY));
  return (
    <Panel>
    <div className={s.month}>
      {WEEK_DAY_SHORT.map((d) => (
        <p key={d} className={s.monthHead}>
          {d}
        </p>
      ))}
      {cells.map((d) => {
        const items = events.filter((e) => e.start.toDateString() === d.toDateString());
        const other = d.getMonth() !== anchor.getMonth();
        return (
          <div key={d.toISOString()} className={[s.cell, other ? s.otherMonth : "", d.toDateString() === NOW.toDateString() ? s.today : ""].join(" ")}>
            <span className={s.cellNum}>{d.getDate()}</span>
            {items.slice(0, 2).map((e) => (
              <span key={e.id} className={[s.chip, s[e.kind]].join(" ")} title={e.title} aria-label={e.title}>
                {shortLabel(e.title)}
              </span>
            ))}
            {items.length > 2 && <span className={s.more}>+{items.length - 2} việc khác</span>}
          </div>
        );
      })}
    </div>
    </Panel>
  );
}

function ListView({ events, isStudent }: { events: CalEvent[]; isStudent: boolean }) {
  if (events.length === 0) {
    return (
      <Panel><EmptyState title="Không còn sự kiện nào sắp tới" action={<ButtonLink href="/practice" variant="primary">Luyện đề</ButtonLink>}>
        Học kỳ đã hết lịch. Bạn vẫn luyện đề được bất cứ lúc nào.
      </EmptyState></Panel>
    );
  }
  return (
    <Panel>
    <ActionList label="Sự kiện sắp tới">
      {events.map((e) => (
        <ActionRow
          key={e.id}
          tone={isNow(e) ? "green" : e.kind === "deadline" ? "amber" : e.kind === "exam" ? "red" : "neutral"}
          href={e.href}
          title={e.title}
          context={`${fmtLongDate(e.start)} · ${fmtTime(e.start)}${e.endLabel ? `–${e.endLabel}` : ""}${e.where ? ` · ${e.where}` : ""}`}
          meta={
            <>
              {EVENT_LABEL[e.kind]}
              {isNow(e) && (
                <>
                  {" · "}
                  <StatusText tone="green">Đang diễn ra</StatusText>
                </>
              )}
              {isStudent && e.studentNote ? ` · ${e.studentNote}` : ""}
            </>
          }
        />
      ))}
    </ActionList>
    </Panel>
  );
}
