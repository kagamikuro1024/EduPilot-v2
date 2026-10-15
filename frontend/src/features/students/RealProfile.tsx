"use client";

import { useQuery } from "@tanstack/react-query";
import { useState } from "react";
import { ApiErrorNotice } from "@/shared/data/ApiErrorNotice";
import { apiClient } from "@/shared/data";
import { useCursorList } from "@/shared/data/useCursorList";
import { Button, ButtonLink, DataTable, DefinitionList, EmptyState, InlineNotice, Page, PageHeader, Panel, PrivateMark, Section, Skeleton, StatusText, Tabs, TrendChart } from "@/shared/ui";
import { ProfileNotes } from "./ProfileNotes";
import { ProfileOutreach } from "./ProfileOutreach";
import { STATUS_LABEL, signedPoints, type AttendanceRow, type Profile } from "./studentsApi";
import s from "./StudentProfile.module.css";

type TabId = "overview" | "attendance" | "grades" | "activity" | "notes";
const TABS: Array<{ value: TabId; label: string }> = [
  { value: "overview", label: "Tổng quan" },
  { value: "attendance", label: "Chuyên cần" },
  { value: "grades", label: "Điểm" },
  { value: "activity", label: "Hoạt động học" },
  { value: "notes", label: "Ghi chú" },
];
const shortDate = (iso: string) => new Date(iso).toLocaleDateString("vi-VN", { day: "2-digit", month: "2-digit", timeZone: "Asia/Ho_Chi_Minh" });

/**
 * Hồ sơ 360 thật (US-P5-05, DESIGN §14.7). MỖI lần mở trang là MỘT request `…/profile` và một dòng `audit_log` (TLR-7): truy vấn `staleTime: 0`, `refetchOnMount: "always"`, `gcTime: 0`,
 * không tự làm mới (đổi cửa sổ / mạng cũng không gọi lại), không thử lại tự động.
 */
export function RealProfile({ courseId, studentId, classCode, userId, canManage }: { courseId: string; studentId: string; classCode: string; userId: string; canManage: boolean }) {
  const [tab, setTab] = useState<TabId>("overview");
  const [composeNote, setComposeNote] = useState(0); // tăng mỗi lần bấm `Thêm ghi chú`: mở tab Ghi chú và đặt con trỏ vào ô nhập
  const [messaging, setMessaging] = useState(false);
  const q = useQuery({
    queryKey: ["student-profile", courseId, studentId],
    staleTime: 0,
    refetchOnMount: "always",
    gcTime: 0,
    retry: false,
    refetchOnWindowFocus: false,
    refetchOnReconnect: false,
    queryFn: async ({ signal }) => (await apiClient.get<Profile>(`/courses/${courseId}/students/${studentId}/profile`, { signal })).data,
  });
  const p = q.data;

  return (
    <Page>
      <PageHeader
        title={p?.student.full_name ?? "Hồ sơ sinh viên"}
        back={{ href: "/students", label: "Sinh viên" }}
        description={p ? `${p.student.student_code ?? ""} · ${classCode}` : undefined}
        actions={
          p && (
            <>
              <Button onClick={() => { setTab("notes"); setComposeNote((n) => n + 1); }}>Thêm ghi chú</Button>
              <Button onClick={() => setMessaging((v) => !v)} aria-expanded={messaging}>Nhắn sinh viên</Button>
            </>
          )
        }
      />
      {!p ? (
        q.isError ? (
          <ApiErrorNotice error={q.error} showTechnical onRetry={() => void q.refetch()} title="Không tải được hồ sơ sinh viên. Dữ liệu và ghi chú đang gõ không bị mất." />
        ) : (
          <Panel><Skeleton lines={8} /></Panel>
        )
      ) : (
        <>
          {p.risk.level !== "NONE" && p.risk.sentence && (
            <InlineNotice tone={p.risk.level === "AT_RISK" ? "warning" : "info"} title={p.risk.sentence}>
              <PrivateMark />
            </InlineNotice>
          )}
          {messaging && <ProfileOutreach courseId={courseId} studentId={studentId} name={p.student.full_name} onClose={() => setMessaging(false)} />}
          <Tabs label="Phần hồ sơ" value={tab} onChange={setTab} options={TABS} />
          {tab === "overview" && <Overview p={p} />}
          {tab === "attendance" && <AttendanceTab courseId={courseId} studentId={studentId} name={p.student.full_name} />}
          {tab === "grades" && <GradesTab p={p} canManage={canManage} />}
          {tab === "activity" && <ActivityTab p={p} />}
          {tab === "notes" && <ProfileNotes courseId={courseId} studentId={studentId} userId={userId} focusSignal={composeNote} />}
        </>
      )}
    </Page>
  );
}

function Overview({ p }: { p: Profile }) {
  const a = p.attendance;
  const present = a.present + a.late + a.excused;
  const parts = [
    a.recorded === 0 ? "Chưa có buổi nào được ghi điểm danh" : `Có mặt ${present}/${a.recorded} buổi, vắng ${a.absent}`,
    `${p.participation.speak_count} lần phát biểu`,
    p.study.tracked ? `tuần này học ${p.study.minutes_7d} phút` : null,
  ].filter(Boolean);
  return (
    <Section panel title="Tổng quan">
      <p className={s.narrative}>{parts.join("; ")}.</p>
      <DefinitionList
        items={[
          { term: "Chuyên cần", value: a.recorded === 0 ? "—" : `${present}/${a.recorded} buổi · vắng ${a.absent}` },
          { term: "Phát biểu", value: `${p.participation.speak_count} lần · ${signedPoints(p.participation.points_total)} điểm cộng` },
          { term: "Học 7 ngày", value: p.study.tracked ? `${p.study.minutes_7d} phút` : "Không đo" },
          { term: "Tuần trước", value: `${p.activity.asks_last_week} câu hỏi · ${p.activity.posts_last_week} bài đăng` },
        ]}
      />
    </Section>
  );
}

function AttendanceTab({ courseId, studentId, name }: { courseId: string; studentId: string; name: string }) {
  const list = useCursorList<AttendanceRow>(["student-attendance", courseId, studentId], `/courses/${courseId}/students/${studentId}/attendance`, { limit: 50, idOf: (r) => r.session.id });
  return (
    <Section panel title="Chuyên cần">
      <DataTable
        caption={`Chuyên cần của ${name}`}
        dense
        loading={list.isPending ? 5 : undefined}
        columns={[
          { key: "n", header: "Buổi", width: "90px", render: (r) => `Buổi ${r.session.session_no}` },
          { key: "d", header: "Ngày", render: (r) => shortDate(r.session.starts_at) },
          { key: "m", header: "Trạng thái", render: (r) => <StatusText tone={r.status === "ABSENT" ? "red" : r.status === "LATE" ? "amber" : r.status === "EXCUSED" ? "blue" : "green"}>{STATUS_LABEL[r.status]}</StatusText> },
          { key: "note", header: "Ghi chú buổi", render: (r) => r.note ?? "" },
        ]}
        rows={list.items}
        rowKey={(r) => r.session.id}
        pagination={{ nextCursor: list.hasNextPage ? (list.data?.pages.at(-1)?.next_cursor ?? null) : null, onLoadMore: () => void list.fetchNextPage(), loading: list.isFetchingNextPage }}
        empty={list.isError ? <ApiErrorNotice error={list.error} onRetry={() => void list.refetch()} title="Không tải được chuyên cần." /> : <EmptyState title="Chưa có buổi nào được ghi điểm danh." />}
      />
    </Section>
  );
}

function GradesTab({ p, canManage }: { p: Profile; canManage: boolean }) {
  return (
    <Section panel title="Điểm">
      <DefinitionList items={[{ term: "Điểm cộng đã ghi", value: signedPoints(p.participation.points_total) }]} />
      <EmptyState title="Lớp chưa có công thức điểm đã xác nhận." action={canManage ? <ButtonLink href="/gradebook">Xác nhận công thức</ButtonLink> : undefined} />
    </Section>
  );
}

function ActivityTab({ p }: { p: Profile }) {
  return (
    <Section panel title="Hoạt động học">
      {p.study.tracked ? (
        <TrendChart label={`Phút học mỗi tuần của ${p.student.full_name}`} points={p.study.weeks.map((w) => ({ x: shortDate(w.week_start), full: `Tuần từ ${shortDate(w.week_start)}`, y: w.minutes }))} format={(v) => `${v} phút`} />
      ) : (
        <EmptyState title="Không đo thời gian học của sinh viên này." />
      )}
      <DefinitionList items={[{ term: "Câu hỏi tuần trước", value: String(p.activity.asks_last_week) }, { term: "Bài đăng tuần trước", value: String(p.activity.posts_last_week) }]} />
    </Section>
  );
}
