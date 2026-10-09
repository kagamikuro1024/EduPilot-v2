"use client";

import { ClipboardList, Plus } from "lucide-react";
import { useRouter } from "next/navigation";
import { useState } from "react";
import { ApiErrorNotice, apiClient, fieldErrors, useCursorList, useIdempotentMutation } from "@/shared/data";
import { useClassCourse } from "@/features/members/classApi";
import { useSession } from "@/shared/session/session";
import { ActionList, ActionRow, Button, ButtonLink, EmptyState, Field, Input, Page, PageHeader, PageState, Panel, Section, Skeleton, type Tone } from "@/shared/ui";
import { eKey, ePath, fmtClock, STATUS_LABEL, windowLine, type ExamDetail, type ExamRow, type ExamStatus, type StudentExam } from "./examApi";
import s from "./Exam.module.css";

const TONE: Record<ExamStatus, Tone> = { DRAFT: "neutral", SCHEDULED: "blue", OPEN: "green", CLOSED: "amber", PUBLISHED: "neutral" };

/** `/exams`: Staff thấy bảng việc theo trạng thái (US-PE-04 AC12); sinh viên thấy bài của lớp mình (AC13). */
export function ExamsHome() {
  const { role } = useSession();
  return role === "student" ? <StudentExams /> : <StaffExams />;
}

// ---- Giảng viên / TA ---------------------------------------------------------------------------------------------------------

function StaffExams() {
  const cc = useClassCourse();
  if (cc.state === "loading") return <Page width="wide"><PageHeader title="Bài thi" /><Panel><Skeleton lines={6} /></Panel></Page>;
  if (cc.state === "none")
    return (
      <Page width="wide">
        <PageHeader title="Bài thi" />
        <Panel>
          <EmptyState title="Chưa chọn lớp">Chọn một lớp ở thanh trên để xem bài thi của lớp.</EmptyState>
        </Panel>
      </Page>
    );
  return <Board courseId={cc.course.id} code={cc.course.class_code} />;
}

const GROUPS: Array<{ title: string; of: (e: ExamRow) => boolean; order: (a: ExamRow, b: ExamRow) => number }> = [
  { title: "Đang mở", of: (e) => e.effective_status === "OPEN", order: (a, b) => (a.closes_at ?? "").localeCompare(b.closes_at ?? "") },
  { title: "Sắp tới", of: (e) => e.effective_status === "SCHEDULED", order: (a, b) => (a.opens_at ?? "").localeCompare(b.opens_at ?? "") },
  { title: "Đã đóng", of: (e) => e.effective_status === "CLOSED" || e.effective_status === "PUBLISHED", order: (a, b) => (b.closes_at ?? "").localeCompare(a.closes_at ?? "") },
  { title: "Nháp", of: (e) => e.effective_status === "DRAFT", order: (a, b) => b.created_at.localeCompare(a.created_at) },
];

function Board({ courseId, code }: { courseId: string; code: string }) {
  const [creating, setCreating] = useState(false);
  const list = useCursorList<ExamRow>(eKey(courseId, "list"), ePath(courseId), { limit: 100 });
  return (
    <Page width="wide">
      <PageHeader
        title="Bài thi"
        description={`Bài kiểm tra tính điểm của lớp ${code}: soạn từ ngân hàng câu hỏi, đặt khung giờ rồi lên lịch.`}
        actions={
          <Button variant="primary" icon={<Plus aria-hidden />} onClick={() => setCreating(true)} aria-expanded={creating}>
            Tạo bài thi
          </Button>
        }
      />
      {creating && <NewExam courseId={courseId} onCancel={() => setCreating(false)} />}
      <PageState
        query={list}
        isEmpty={() => list.items.length === 0}
        loading={<Panel><ActionList loading={5} label="Đang tải bài thi" /></Panel>}
        empty={
          <Panel>
            <EmptyState title="Chưa có bài thi nào" icon={<ClipboardList aria-hidden />} action={creating ? undefined : <Button variant="primary" onClick={() => setCreating(true)}>Tạo bài thi</Button>}>
              Chưa có bài thi nào. Tạo bài thi đầu tiên từ ngân hàng câu hỏi.
            </EmptyState>
          </Panel>
        }
        showTechnical
      >
        {GROUPS.map((g) => {
          const rows = list.items.filter(g.of).sort(g.order);
          if (rows.length === 0) return null;
          return (
            <Section key={g.title} title={g.title} panel>
              <ActionList label={g.title}>
                {rows.map((e) => (
                  <ActionRow
                    key={e.id}
                    tone={TONE[e.effective_status]}
                    title={e.title}
                    context={windowLine(e)}
                    meta={e.attempts.started > 0 ? `${e.attempts.started} sinh viên đã bắt đầu` : e.effective_status === "DRAFT" ? "Chưa lên lịch" : undefined}
                    href={e.effective_status === "CLOSED" || e.effective_status === "PUBLISHED" ? `/exams/${e.id}/results?course=${courseId}` : `/exams/${e.id}`}
                    action={e.effective_status === "CLOSED" || e.effective_status === "PUBLISHED" ? <span className={s.rowHint}>Xem kết quả</span> : undefined}
                  />
                ))}
              </ActionList>
            </Section>
          );
        })}
        {list.hasNextPage && <div className={s.more}><Button onClick={() => void list.fetchNextPage()} loading={list.isFetchingNextPage}>Xem thêm</Button></div>}
      </PageState>
    </Page>
  );
}

/** Tạo bài nháp chỉ cần tiêu đề; giờ, câu hỏi và cài đặt điền ở trang soạn (`/exams/[id]`). */
function NewExam({ courseId, onCancel }: { courseId: string; onCancel: () => void }) {
  const router = useRouter();
  const [title, setTitle] = useState("");
  const create = useIdempotentMutation<string, ExamDetail>((t, key) => apiClient.post<ExamDetail>(ePath(courseId), { title: t }, { idempotencyKey: key }));
  const fe = fieldErrors(create.error);
  async function submit(e: React.FormEvent) {
    e.preventDefault();
    try {
      const r = await create.mutate(title.trim());
      router.push(`/exams/${r.data.id}`);
    } catch {
      /* lỗi hiện ngay dưới ô; chữ đã gõ giữ nguyên */
    }
  }
  return (
    <form className={s.newExam} onSubmit={submit} noValidate aria-label="Bài thi mới">
      <Field label="Tiêu đề" required error={fe.title} className={s.grow}>
        {(id, by) => <Input id={id} aria-describedby={by} invalid={Boolean(fe.title)} value={title} maxLength={120} autoFocus placeholder="Kiểm tra tuần 9" onChange={(e) => setTitle(e.target.value)} />}
      </Field>
      <Button type="submit" loading={create.pending} disabled={title.trim() === ""}>Tạo bản nháp</Button>
      <Button variant="text" onClick={onCancel}>Bỏ qua</Button>
      {create.error && !Object.keys(fe).length && <ApiErrorNotice error={create.error} showTechnical />}
    </form>
  );
}

// ---- Sinh viên ---------------------------------------------------------------------------------------------------------------

function StudentExams() {
  const { realCourses, realCourseId } = useSession();
  if (!realCourses) return <Page><PageHeader title="Bài thi" /><Panel><Skeleton lines={5} /></Panel></Page>;
  const mine = realCourses.filter((c) => c.role_in_course === "STUDENT" && (realCourseId === "all" || realCourseId === c.id || !realCourseId));
  if (mine.length === 0)
    return (
      <Page>
        <PageHeader title="Bài thi" />
        <Panel>
          <EmptyState title="Chưa có bài thi">Lớp của bạn chưa có bài thi nào.</EmptyState>
        </Panel>
      </Page>
    );
  return (
    <Page>
      <PageHeader title="Bài thi" description="Bài kiểm tra tính điểm của lớp. Đồng hồ chạy khi bạn bấm Bắt đầu làm bài." />
      {mine.map((c) => (
        <CourseExams key={c.id} courseId={c.id} code={mine.length > 1 ? c.class_code : undefined} />
      ))}
    </Page>
  );
}

const STUDENT_GROUPS: Array<{ title: string; of: (e: StudentExam) => boolean; order: (a: StudentExam, b: StudentExam) => number }> = [
  { title: "Đang mở", of: (e) => e.status === "OPEN", order: (a, b) => (a.closes_at ?? "").localeCompare(b.closes_at ?? "") },
  { title: "Sắp tới", of: (e) => e.status === "SCHEDULED", order: (a, b) => (a.opens_at ?? "").localeCompare(b.opens_at ?? "") },
  { title: "Đã có điểm", of: (e) => e.status === "PUBLISHED", order: (a, b) => (b.closes_at ?? "").localeCompare(a.closes_at ?? "") },
  { title: "Đã đóng", of: (e) => e.status === "CLOSED", order: (a, b) => (b.closes_at ?? "").localeCompare(a.closes_at ?? "") },
];

/** Mỗi hàng một hành động (AC13): Tiếp tục · Bắt đầu làm bài · Xem kết quả · không có. */
function actionFor(e: StudentExam): { label: string } | null {
  if (e.my_attempt?.status === "IN_PROGRESS" && (e.status === "OPEN" || e.status === "SCHEDULED")) return { label: "Tiếp tục" };
  if (e.status === "PUBLISHED" && e.my_attempt) return { label: "Xem kết quả" };
  if (e.status === "OPEN" && !e.my_attempt) return { label: "Bắt đầu làm bài" };
  return null;
}

function contextFor(e: StudentExam): string {
  const dur = e.duration_minutes ? ` · ${e.duration_minutes} phút` : "";
  if (e.status === "OPEN") return `Mở đến ${e.closes_at ? fmtClock(e.closes_at) : "—"}${dur}`;
  if (e.status === "SCHEDULED") return `Mở lúc ${e.opens_at ? fmtClock(e.opens_at) : "—"}${dur}`;
  if (e.status === "PUBLISHED") return e.my_score ? `Điểm ${e.my_score.replace(".", ",")} / ${e.max_score.replace(".", ",")}` : "Bạn không làm bài này";
  return e.my_attempt ? "Đã nộp — điểm hiện khi giảng viên công bố" : "Đã đóng";
}

function CourseExams({ courseId, code }: { courseId: string; code?: string }) {
  const list = useCursorList<StudentExam>(eKey(courseId, "mine"), ePath(courseId), { limit: 100 });
  return (
    <PageState
      query={list}
      isEmpty={() => list.items.length === 0}
      loading={<Panel><ActionList loading={3} label="Đang tải bài thi" /></Panel>}
      empty={<Panel><EmptyState title={code ? `Lớp ${code}` : "Chưa có bài thi"}>Lớp của bạn chưa có bài thi nào.</EmptyState></Panel>}
    >
      {STUDENT_GROUPS.map((g) => {
        const rows = list.items.filter(g.of).sort(g.order);
        if (rows.length === 0) return null;
        return (
          <Section key={g.title} title={code ? `${g.title} · lớp ${code}` : g.title} panel>
            <ActionList label={g.title}>
              {rows.map((e) => {
                const act = actionFor(e);
                return (
                  <ActionRow
                    key={e.id}
                    tone={TONE[e.status]}
                    title={e.title}
                    context={contextFor(e)}
                    meta={STATUS_LABEL[e.status]}
                    action={act ? <ButtonLink href={`/exams/${e.id}/take`} variant="secondary">{act.label}</ButtonLink> : undefined}
                  />
                );
              })}
            </ActionList>
          </Section>
        );
      })}
    </PageState>
  );
}
