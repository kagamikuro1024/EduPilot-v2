"use client";

import type { ReactNode } from "react";
import { ActionList, ActionRow, Button, PageHeader, Section } from "@/shared/ui";
import { ATTENTION, STUDENT_DAY, STUDENT_NEXT, TEACHER_STATS, TEACHER_TASKS, UPCOMING } from "./fixtures";
import s from "./PanelSamples.module.css";

/** Panel tạm của trang mẫu (US-UI-01): chưa có `shared/ui/Panel.tsx` (US-UI-02). Tiêu đề vùng nằm NGOÀI panel. */
function SamplePanel({ children, label }: { children: ReactNode; label: string }) {
  return (
    <div className={s.panel} data-ep-panel aria-label={label} role="group">
      {children}
    </div>
  );
}

/** Ô nhấn: chỉ cho số liệu / trạng thái quan trọng; ≤ 3 mỗi panel. */
function Strong({ children }: { children: ReactNode }) {
  return (
    <div className={s.strong} data-tone="strong">
      {children}
    </div>
  );
}

function Teacher() {
  return (
    <>
      <Section title="Việc cần xử lý hôm nay">
        <SamplePanel label="Việc cần xử lý hôm nay">
          <div className={s.strongRow}>
            {TEACHER_STATS.map((x) => (
              <Strong key={x.id}>
                <span className={s.num}>{x.value}</span> {x.label}
              </Strong>
            ))}
          </div>
          <ActionList label="Việc cần xử lý">
            {TEACHER_TASKS.map((t) => (
              <ActionRow key={t.id} tone={t.tone} title={t.title} context={t.context} meta={t.meta} action={<Button size="sm" variant={t.primary ? "primary" : "secondary"}>{t.action}</Button>} />
            ))}
          </ActionList>
        </SamplePanel>
      </Section>
      <Section title="Lớp cần chú ý">
        <SamplePanel label="Lớp cần chú ý">
          <ActionList label="Lớp cần chú ý">
            {ATTENTION.map((a) => (
              <ActionRow key={a.id} tone="amber" title={a.title} context={a.context} meta={a.meta} />
            ))}
          </ActionList>
        </SamplePanel>
      </Section>
      <Section title="Sắp tới">
        <SamplePanel label="Sắp tới">
          <ActionList label="Sắp tới">
            {UPCOMING.map((u) => (
              <ActionRow key={u.id} title={u.title} context={u.context} meta={u.meta} />
            ))}
          </ActionList>
        </SamplePanel>
      </Section>
    </>
  );
}

function Student() {
  return (
    <>
      <Section title="Việc nên làm tiếp">
        <SamplePanel label="Việc nên làm tiếp">
          <div className={s.next}>
            <div className={s.nextText}>
              <p className={s.nextTitle}>{STUDENT_NEXT.title}</p>
              <p className={s.reason}>{STUDENT_NEXT.reason}</p>
            </div>
            <Strong>{STUDENT_NEXT.minutes}</Strong>
          </div>
          <div className={s.nextAction}>
            <Button variant="primary">{STUDENT_NEXT.action}</Button>
          </div>
        </SamplePanel>
      </Section>
      <Section title="Hôm nay">
        <SamplePanel label="Hôm nay">
          <ol className={s.timeline} aria-label="Dòng thời gian hôm nay">
            {STUDENT_DAY.map((d) => (
              <li key={d.id} className={s.moment}>
                <time className={s.time}>{d.time}</time>
                <span className={s.dot} data-need={d.need || undefined} aria-hidden />
                <span className={s.momentText}>
                  <span className={s.momentTitle}>{d.title}</span>
                  <span className={s.momentContext}>{d.context}</span>
                </span>
              </li>
            ))}
          </ol>
        </SamplePanel>
      </Section>
    </>
  );
}

/** Trang "Hôm nay" mẫu của một vai, trong khung ứng dụng thật (layout.tsx). */
export function PanelSamples({ role }: { role: "teacher" | "student" }) {
  return (
    <div className={s.sample}>
      <PageHeader title={role === "teacher" ? "4 việc cần xử lý hôm nay" : "Hôm nay"} description={role === "teacher" ? "Thứ Tư, 9 tháng 10 · Tất cả lớp của tôi" : "Thứ Tư, 9 tháng 10 · Lớp 761987"} />
      {role === "teacher" ? <Teacher /> : <Student />}
    </div>
  );
}
