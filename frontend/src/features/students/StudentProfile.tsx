"use client";

import { useState } from "react";
import { NOW, STUDENT_B, courseById, fmtScore, fmtShortDate, studentById } from "@/mock/core";
import { attendanceStats, qtOf } from "@/mock/grades";
import { BT03_SEED } from "@/mock/assess";
import { NOTES_KEY, NOTES_SEED, absentSessions, activitySeries, riskSentence, rosterOf, sessionsOf, type StudentNote } from "@/mock/roster";
import {
  ATTENDANCE_SEED,
  committedOf,
  CURRENT_SESSION,
  KEYS,
  MEMBERS_SEED,
  type AttendanceState,
  type Bt03State,
  type MembersState,
} from "@/mock/state";
import { useUndoLine } from "@/shared/lib/useUndoLine";
import { useSession } from "@/shared/session/session";
import { useDemoSlice } from "@/shared/state/demo";
import {
  Button,
  ButtonLink,
  DataTable,
  DefinitionList,
  EmptyState,
  InlineNotice,
  Page,
  PageHeader,
  PageState,
  PrivateMark,
  Section,
  Skeleton,
  StatusText,
  Tabs,
  Textarea,
  TrendChart,
  useRouteState,
} from "@/shared/ui";
import s from "./StudentProfile.module.css";

type TabId = "overview" | "attendance" | "grades" | "activity" | "notes";

const TABS: Array<{ value: TabId; label: string }> = [
  { value: "overview", label: "Tổng quan" },
  { value: "attendance", label: "Chuyên cần" },
  { value: "grades", label: "Điểm" },
  { value: "activity", label: "Hoạt động học" },
  { value: "notes", label: "Ghi chú" },
];

/** Hồ sơ 360: hiểu một sinh viên trước khi ra quyết định dạy học (DESIGN §14.7). */
export function StudentProfile({ id }: { id: string }) {
  const { course, user } = useSession();
  const [attendance] = useDemoSlice<AttendanceState>(KEYS.attendance, ATTENDANCE_SEED);
  const [members] = useDemoSlice<MembersState>(KEYS.members, MEMBERS_SEED);
  const [bt03] = useDemoSlice<Bt03State>(KEYS.bt03, BT03_SEED);
  const [notes, setNotes] = useDemoSlice<Record<string, StudentNote[]>>(NOTES_KEY, NOTES_SEED);
  const [tab, setTab] = useState<TabId>("overview");
  const [draft, setDraft] = useState("");
  const [adding, setAdding] = useState(false);
  const undo = useUndoLine();
  const routeState = useRouteState();

  const student = studentById(id);
  const inClass = student ? rosterOf(course.id, members).some((x) => x.id === student.id) : false;

  if (!student || !inClass) {
    return (
      <Page>
        <PageHeader title="Hồ sơ sinh viên" back={{ href: "/students", label: "Sinh viên" }} />
        <EmptyState title="Sinh viên này không thuộc lớp đang chọn" action={<ButtonLink href="/students">Về danh sách sinh viên</ButtonLink>}>
          Đổi lớp ở bộ chọn phía trên, hoặc mở lại từ danh sách sinh viên của lớp.
        </EmptyState>
      </Page>
    );
  }

  const stats = attendanceStats(student, attendance);
  const grade = qtOf(student.id, attendance, bt03);
  const risk = riskSentence(student, stats);
  const mine = notes[student.id] ?? [];
  const absent = absentSessions(student);
  const today = committedOf(attendance[course.id]?.[CURRENT_SESSION]);
  const activity = activitySeries(student);

  const rowsAttendance = sessionsOf(course.id)
    .filter((x) => x.state === "recorded" || (x.state === "current" && today))
    .map((x) => {
      const mark = x.state === "current" ? today?.marks[student.id] ?? "present" : absent.includes(x.n) ? "absent" : "present";
      const speaks = x.state === "current" ? today?.speaks[student.id] ?? 0 : 0;
      return { ...x, mark, speaks };
    });

  const sid = student.id;
  function addNote() {
    const text = draft.trim();
    if (!text) return;
    const note: StudentNote = { id: `n-${Date.now()}`, at: fmtShortDate(NOW), by: user.name, text };
    setNotes((prev) => ({ ...prev, [sid]: [note, ...(prev[sid] ?? [])] }));
    setDraft("");
    setAdding(false);
    undo.push("Đã thêm ghi chú", () => setNotes((prev) => ({ ...prev, [sid]: (prev[sid] ?? []).filter((n) => n.id !== note.id) })));
  }

  return (
    <Page>
      <PageHeader
        title={student.name}
        back={{ href: "/students", label: "Sinh viên" }}
        description={`${student.code} · ${courseById(course.id).label}`}
        actions={
          <>
            <Button onClick={() => setAdding(true)}>Thêm ghi chú</Button>
            <Button variant="primary" onClick={() => undo.push(`Đã gửi lời nhắn riêng tới ${student.name} (mô phỏng)`)}>
              Nhắn riêng
            </Button>
          </>
        }
      />
      <PageState
        state={routeState}
        loading={<Skeleton lines={8} />}
        empty={<EmptyState title="Chưa có dữ liệu học tập của sinh viên này">Dữ liệu xuất hiện sau buổi học hoặc bài nộp đầu tiên.</EmptyState>}
        error={{ problem: "Không tải được hồ sơ sinh viên.", recovery: "Ghi chú bạn đang gõ vẫn được giữ. Thử lại, hoặc mở lại sau ít phút." }}
      >
        {risk && (
          <InlineNotice tone={student.risk === "high" ? "warning" : "info"} title={risk}>
            <PrivateMark />
          </InlineNotice>
        )}
        {undo.node}

        <Tabs label="Phần hồ sơ" value={tab} onChange={setTab} options={TABS} />

        {tab === "overview" && (
          <Section title="Tổng quan">
            <p className={s.narrative}>
              {student.name} dự {stats.recorded - stats.absences}/{stats.recorded} buổi đã ghi, phát biểu {stats.speaks} lần
              {grade.qt === null ? " và chưa có điểm quá trình" : `, điểm quá trình hiện tại ${fmtScore(grade.qt)} (tạm tính)`}. Thời gian học trung bình{" "}
              {student.activityMin} phút mỗi tuần.
            </p>
            <DefinitionList
              items={[
                { term: "Chuyên cần", value: `${stats.recorded - stats.absences}/${stats.recorded} buổi · vắng ${stats.absences}` },
                { term: "Phát biểu", value: `${stats.speaks} lần · cộng ${fmtScore(grade.bonus)} điểm` },
                { term: "Điểm quá trình", value: grade.qt === null ? "Chưa có" : `${fmtScore(grade.qt)} (tạm tính)` },
                { term: "Bài còn thiếu", value: [grade.bt01 === null && "Bài tập 01", grade.bt02 === null && "Bài tập 02"].filter(Boolean).join(", ") || "Không" },
              ]}
            />
          </Section>
        )}

        {tab === "attendance" && (
          <Section title="Chuyên cần" description={`Vắng ${stats.absences}/${stats.recorded} buổi đã ghi · từ buổi vắng thứ 3 trừ 0,5 điểm mỗi buổi`}>
            <DataTable
              caption={`Chuyên cần của ${student.name}`}
              dense
              columns={[
                { key: "n", header: "Buổi", width: "90px", render: (r) => `Buổi ${r.n}` },
                { key: "d", header: "Ngày", render: (r) => fmtShortDate(r.date) },
                {
                  key: "m",
                  header: "Trạng thái",
                  render: (r) =>
                    r.mark === "absent" ? (
                      <StatusText tone="red">Vắng</StatusText>
                    ) : r.mark === "late" ? (
                      <StatusText tone="amber">Muộn</StatusText>
                    ) : r.mark === "excused" ? (
                      <StatusText tone="blue">Vắng phép</StatusText>
                    ) : (
                      <StatusText tone="green">Có mặt</StatusText>
                    ),
                },
                { key: "s", header: "Phát biểu", align: "end", render: (r) => (r.speaks > 0 ? `${r.speaks} lần` : "—") },
              ]}
              rows={rowsAttendance}
              rowKey={(r) => String(r.n)}
              empty={<EmptyState title="Lớp chưa có buổi nào được ghi điểm danh">Điểm danh buổi đầu tiên để dữ liệu chuyên cần xuất hiện.</EmptyState>}
            />
          </Section>
        )}

        {tab === "grades" && (
          <Section title="Điểm" description="Điểm quá trình tạm tính theo công thức đã xác nhận của lớp.">
            <DefinitionList
              items={[
                { term: "Bài tập 01", value: grade.bt01 === null ? "Chưa nộp" : fmtScore(grade.bt01) },
                { term: "Bài tập 02", value: grade.bt02 === null ? "Chưa nộp" : fmtScore(grade.bt02) },
                {
                  term: "Bài tập 03",
                  value: grade.bt03 === null ? (student.id === STUDENT_B.id ? "Đang chấm, chưa công bố" : "Chưa nộp") : fmtScore(grade.bt03),
                },
                { term: "Trung bình bài tập", value: grade.avg === null ? "—" : fmtScore(grade.avg, 2) },
                { term: "Cộng phát biểu", value: `+${fmtScore(grade.bonus)}` },
                { term: "Trừ vắng", value: grade.penalty > 0 ? `−${fmtScore(grade.penalty)}` : "0" },
                { term: "Điểm quá trình", value: grade.qt === null ? "Chưa có" : fmtScore(grade.qt) },
              ]}
            />
          </Section>
        )}

        {tab === "activity" && (
          <Section title="Hoạt động học" description="Phút học mỗi tuần — chỉ để tham khảo, không dùng để chấm điểm.">
            <TrendChart label={`Phút học mỗi tuần của ${student.name}`} points={activity} tone={student.activityMin < 45 ? "red" : "ink"} format={(v) => `${v} phút`} />
          </Section>
        )}

        {tab === "notes" && (
          <Section title="Ghi chú" action={<PrivateMark />}>
            {adding && (
              <div className={s.compose}>
                <Textarea
                  autoFocus
                  rows={3}
                  value={draft}
                  placeholder="Điều bạn quan sát được và việc đã làm — sinh viên không thấy nội dung này."
                  onChange={(e) => setDraft(e.target.value)}
                />
                <div className={s.composeBar}>
                  <Button variant="primary" onClick={addNote} disabled={draft.trim() === ""}>
                    Lưu ghi chú
                  </Button>
                  <Button variant="text" onClick={() => setAdding(false)}>
                    Để sau
                  </Button>
                </div>
              </div>
            )}
            {mine.length === 0 && !adding ? (
              <EmptyState title="Chưa có ghi chú nào" action={<Button onClick={() => setAdding(true)}>Thêm ghi chú</Button>}>
                Ghi lại điều bạn quan sát được để lần sau không phải nhớ lại; chỉ giảng viên và trợ giảng đọc được.
              </EmptyState>
            ) : (
              <ul className={s.notes}>
                {mine.map((n) => (
                  <li key={n.id}>
                    <p>{n.text}</p>
                    <p className="ep-meta">
                      {n.by} · {n.at}
                    </p>
                  </li>
                ))}
              </ul>
            )}
          </Section>
        )}
      </PageState>
    </Page>
  );
}
