"use client";

import { useQuery } from "@tanstack/react-query";
import { ApiErrorNotice } from "@/shared/data/ApiErrorNotice";
import { apiClient } from "@/shared/data";
import { DefinitionList, EmptyState, Page, PageHeader, Panel, Section, Skeleton, TrendChart } from "@/shared/ui";
import { EVENT_LABEL, STATUS_LABEL, attendanceSentence, dayMonthOf, signedPoints, type MyAttendance, type MyParticipation, type MyStudyTime } from "./meApi";
import s from "./Me.module.css";

const dayMonth = (ymd: string) => `${ymd.slice(8, 10)}/${ymd.slice(5, 7)}`;

/** Số liệu đọc mới mỗi lần mở / quay lại tab (điểm danh không cache — luật 15); lỗi mạng giữ nội dung đã tải. */
function useMine<T>(courseId: string, leaf: string, query = "") {
  return useQuery({
    queryKey: ["me", courseId, leaf, query],
    staleTime: 0,
    refetchOnMount: "always",
    retry: false,
    queryFn: async ({ signal }) => (await apiClient.get<T>(`/courses/${courseId}/me/${leaf}${query}`, { signal })).data,
  });
}

/** Kết quả của tôi (DESIGN §14.9): chuyên cần, điểm cộng phát biểu, nhịp học nhỏ ở cuối. Không điểm thành phần (P6), không xếp hạng. */
export function RealMe({ courseId, classCode }: { courseId: string; classCode: string }) {
  const att = useMine<MyAttendance>(courseId, "attendance", "?limit=100");
  const part = useMine<MyParticipation>(courseId, "participation", "?limit=10");
  const study = useMine<MyStudyTime>(courseId, "study-time", "?weeks=8");
  const retry = () => {
    void att.refetch();
    void part.refetch();
    void study.refetch();
  };
  const failed = (!att.data && att.isError) || (!part.data && part.isError);
  return (
    <Page>
      <PageHeader title="Kết quả của tôi" description={classCode} />
      {failed ? (
        <ApiErrorNotice error={att.error ?? part.error} showTechnical onRetry={retry} title="Không tải được kết quả của bạn. Dữ liệu của bạn không bị mất." />
      ) : !att.data || !part.data ? (
        <Panel><Skeleton lines={6} /></Panel>
      ) : (
        <>
          <Attendance a={att.data.attendance} />
          <Participation p={part.data.participation} />
          {study.data && <StudyTrend t={study.data} />}
        </>
      )}
    </Page>
  );
}

function Attendance({ a }: { a: MyAttendance["attendance"] }) {
  const missed = a.sessions.filter((x) => x.status !== "PRESENT");
  return (
    <Section panel title="Chuyên cần">
      {a.recorded === 0 ? (
        <EmptyState title="Chưa có buổi nào được điểm danh." />
      ) : (
        <>
          <p className={s.headline}>{attendanceSentence(a)}</p>
          {missed.length > 0 && <DefinitionList items={missed.map((x) => ({ term: `Buổi ${x.session_no} · ${dayMonthOf(x.starts_at)}`, value: STATUS_LABEL[x.status] ?? x.status }))} />}
        </>
      )}
    </Section>
  );
}

function Participation({ p }: { p: MyParticipation["participation"] }) {
  return (
    <Section panel title="Điểm cộng đã ghi">
      {p.events.length === 0 && p.speak_count === 0 ? (
        <EmptyState title="Chưa có điểm cộng nào được ghi." />
      ) : (
        <>
          <p className={s.headline}>{`Phát biểu ${p.speak_count} lần · ${signedPoints(p.points_total)}.`}</p>
          <DefinitionList items={p.events.map((e) => ({ term: dayMonth(e.date), value: `${EVENT_LABEL[e.type] ?? e.type} ${signedPoints(e.points)}` }))} />
        </>
      )}
    </Section>
  );
}

/** Xu hướng 8 tuần chỉ để tham khảo: không "mục tiêu", không so sánh, không tô đỏ tuần thấp (US-P5-06 AC5). */
function StudyTrend({ t }: { t: MyStudyTime }) {
  if (!t.tracking.enabled) return <Section panel title="Thời gian học"><p className={s.sub}>Đang tắt</p></Section>;
  if (t.weeks.length === 0) return null;
  return (
    <Section panel title="Thời gian học">
      <div className={s.trend}>
        <TrendChart label="Phút học mỗi tuần" height={96} points={t.weeks.map((w) => ({ x: dayMonthOf(w.week_start), full: `Tuần từ ${dayMonthOf(w.week_start)}`, y: w.minutes }))} format={(v) => `${v} phút`} />
      </div>
      <p className={s.sub}>{`Tuần này: ${t.this_week_minutes} phút`}</p>
    </Section>
  );
}
