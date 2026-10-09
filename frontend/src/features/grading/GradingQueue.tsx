"use client";

import { useMemo, useState } from "react";
import { ListChecks, Plus, Send } from "lucide-react";
import { ActionList, ActionRow, Button, type Column, ConfirmIrreversible, DataTable, EmptyState, Field, FilterChips, InlineNotice, Input, Page, PageHeader, PageState, Panel, Select, Skeleton, StatusText, Tabs, Textarea, Toolbar } from "@/shared/ui";
import { COURSE_2, fmtScore, fmtShortDate, fmtTime, studentById, studentsOf } from "@/mock/core";
import { ASSIGNMENTS, BT03, BT03_SEED, bt03Submissions, type Assignment, type Submission } from "@/mock/assess";
import { isSubmissionApproved, studentNo } from "@/mock/derive";
import { logGrade } from "@/mock/audit";
import { bt03Total } from "@/mock/grades";
import { noteBt03Published } from "@/mock/notes";
import { KEYS, type Bt03State } from "@/mock/state";
import { useSession } from "@/shared/session/session";
import { simNowMs } from "@/shared/state/clock";
import { useDemoSlice, writeSlice } from "@/shared/state/demo";
import { useUndoLine } from "@/shared/lib/useUndoLine";
import s from "./Grading.module.css";

type Filter = "flag" | "unapproved" | "late";
type Draft = { id: string; title: string; kind: "ESSAY" | "QUIZ"; due: string; note: string };

const FILTERS: Array<{ value: Filter; label: string }> = [
  { value: "flag", label: "Cần xem kỹ" },
  { value: "unapproved", label: "Chưa duyệt" },
  { value: "late", label: "Nộp muộn" },
];

export function GradingQueue() {
  const { role, course, user } = useSession();
  const [bt03, setBt03] = useDemoSlice<Bt03State>(KEYS.bt03, BT03_SEED);
  const [approvedIds] = useDemoSlice<string[]>("grading.approved", []);
  const [publishedIds, setPublishedIds] = useDemoSlice<string[]>("grading.published", []);
  const [drafts, setDrafts] = useDemoSlice<Draft[]>("grading.drafts", []);
  const [tab, setTab] = useState<"queue" | "assignments">("queue");
  const [filters, setFilters] = useState<Filter[]>(["flag", "unapproved"]);
  const [selected, setSelected] = useState<Set<string>>(new Set());
  const [publishing, setPublishing] = useState(false);
  const [creating, setCreating] = useState(false);
  const [form, setForm] = useState<Draft>({ id: "", title: "", kind: "ESSAY", due: "", note: "" });
  const undo = useUndoLine();

  const hasQueue = course.id !== COURSE_2;
  const all = useMemo(() => (hasQueue ? bt03Submissions(studentsOf(course.id)) : []), [course.id, hasQueue]);

  const isApproved = (sub: Submission) => isSubmissionApproved(sub, bt03.status, approvedIds);
  const isPublished = (sub: Submission) => (sub.id === "sub-bt03-sv-2" ? bt03.status === "published" : publishedIds.includes(sub.id));

  const passes = (x: Submission, fs: Filter[]) =>
    (!fs.includes("flag") || Boolean(x.flag)) && (!fs.includes("unapproved") || !isApproved(x)) && (!fs.includes("late") || x.lateDays > 0);

  /** Đổi bộ lọc: bài bị lọc khỏi danh sách thì bỏ chọn — `Công bố` không bao giờ chạy trên dữ liệu đang ẩn (03-6). */
  function changeFilters(next: Filter[]) {
    setFilters(next);
    setSelected((prev) => new Set([...prev].filter((id) => all.some((x) => x.id === id && passes(x, next)))));
  }

  const rows = useMemo(() => {
    const kept = all.filter((x) => passes(x, filters));
    // Bài của B (đang được lọc ưu tiên) đứng đầu; còn lại theo thứ tự chuẩn `sv-n` (SRS 4.8 N4).
    return kept.sort((a, b) => {
      if (a.id === "sub-bt03-sv-2") return -1;
      if (b.id === "sub-bt03-sv-2") return 1;
      return studentNo(a.studentId) - studentNo(b.studentId);
    });
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [all, filters, bt03.status, approvedIds]);

  const doneCount = all.filter(isApproved).length;
  // Chỉ tính lựa chọn đang NHÌN THẤY (kể cả khi bài vừa được duyệt làm nó rời bộ lọc "Chưa duyệt")
  const picked = rows.filter((x) => selected.has(x.id));
  const pickedUnapproved = picked.filter((x) => !isApproved(x)).length;
  const pickedPublished = picked.filter(isPublished).length;

  const columns: Column<Submission>[] = [
    {
      key: "student",
      header: "Sinh viên",
      frozen: true,
      width: "208px",
      render: (x) => {
        const st = studentById(x.studentId);
        return (
          <span className={s.who}>
            <span className={s.name}>{st?.name}</span>
            <span className={s.code}>{st?.code}</span>
          </span>
        );
      },
    },
    { key: "assignment", header: "Bài tập", width: "220px", render: () => `Bài tập 03 · ${BT03.title}` },
    { key: "source", header: "Nguồn", width: "160px", render: (x) => <span className={s.sub}>{x.source}</span> },
    {
      key: "ai",
      header: "Điểm AI (nháp)",
      align: "end",
      width: "120px",
      render: (x) => (
        <span className={s.score}>
          {x.id === "sub-bt03-sv-2" ? fmtScore(bt03.scores.reduce((a, b) => a + b, 0)) : fmtScore(x.ai)}
        </span>
      ),
    },
    {
      key: "state",
      header: "Trạng thái",
      width: "150px",
      render: (x) =>
        isPublished(x) ? <StatusText tone="green">Đã công bố</StatusText> : isApproved(x) ? <StatusText tone="blue">Đã duyệt</StatusText> : <StatusText tone="amber">Chưa duyệt</StatusText>,
    },
    { key: "flag", header: "Lý do cần xem kỹ", width: "260px", render: (x) => <span className={s.flag}>{x.flag ?? "—"}</span> },
    {
      key: "at",
      header: "Nộp lúc",
      width: "140px",
     
      render: (x) => (
        <span className={s.sub}>
          {fmtShortDate(x.submittedAt)} {fmtTime(x.submittedAt)}
          {x.lateDays > 0 ? ` · muộn ${x.lateDays} ngày` : ""}
        </span>
      ),
    },
  ];

  function publish() {
    const ids = picked.map((x) => x.id);
    if (ids.includes("sub-bt03-sv-2")) setBt03((prev) => ({ ...prev, status: "published" }));
    setPublishedIds((prev) => [...new Set([...prev, ...ids])]);
    for (const sub of picked) {
      noteBt03Published(sub.studentId);
      logGrade(sub.studentId, user.name, `Công bố điểm Bài tập 03: ${fmtScore(sub.id === "sub-bt03-sv-2" ? bt03Total(bt03).total : sub.ai)}`);
    }
    writeSlice(KEYS.meStamp, simNowMs());
    setSelected(new Set());
    undo.push(`Đã công bố điểm Bài tập 03 cho ${ids.length} sinh viên · sinh viên thấy điểm và nhận xét ngay`);
  }

  function saveDraft() {
    if (form.title.trim() === "") return;
    const draft: Draft = { ...form, id: `draft-${drafts.length + 1}` };
    setDrafts((prev) => [...prev, draft]);
    setForm({ id: "", title: "", kind: "ESSAY", due: "", note: "" });
    setCreating(false);
    undo.push(`Đã lưu nháp bài tập “${draft.title}” · chưa giao cho sinh viên`, () => setDrafts((prev) => prev.filter((d) => d.id !== draft.id)));
  }

  return (
    <Page width="wide">
      <PageHeader
        title="Chấm bài"
        description={`${course.label} · AI chấm nháp, giảng viên duyệt và công bố`}
        meta={
          <>
            <span>
              {doneCount}/{all.length} bài đã duyệt
            </span>
            <span>Bài tập 03 · hạn {fmtShortDate(BT03.due)}</span>
          </>
        }
        actions={
          tab === "queue" && role === "teacher" ? (
            <Button variant="primary" icon={<Send aria-hidden />} disabled={picked.length === 0 || pickedUnapproved > 0} onClick={() => setPublishing(true)}>
              Công bố
            </Button>
          ) : tab === "assignments" && role === "teacher" ? (
            <Button variant="primary" icon={<Plus aria-hidden />} onClick={() => setCreating(true)}>
              Tạo bài tập
            </Button>
          ) : undefined
        }
      />

      <div className={s.tabsRow}>
        <Tabs
          label="Phần của màn chấm bài"
          value={tab}
          onChange={setTab}
          options={[
            { value: "queue", label: "Hàng chờ chấm", count: all.filter((x) => !isApproved(x)).length },
            { value: "assignments", label: "Bài tập", count: ASSIGNMENTS.length + drafts.length },
          ]}
        />
      </div>

      <PageState
        loading={<Panel><Skeleton lines={10} /></Panel>}
        empty={
          <Panel><EmptyState title="Chưa có bài nào cần chấm" icon={<ListChecks aria-hidden />}>
            Lớp này chưa có bài tập nào đang mở. Khi sinh viên nộp bài, AI chấm nháp trước rồi bài sẽ xuất hiện ở đây để bạn duyệt.
          </EmptyState></Panel>
        }
        error={{ problem: "Không tải được hàng chờ chấm bài.", recovery: "Các bài đã duyệt vẫn được giữ. Thử lại sau ít phút." }}
        state={hasQueue ? undefined : "empty"}
      >
        {tab === "queue" ? (
          <Panel>
            {role === "ta" && (
              <InlineNotice tone="info" title="Chỉ giảng viên công bố điểm">
                Trợ giảng duyệt bài để chốt điểm nháp của AI. Việc công bố điểm cho sinh viên do giảng viên thực hiện.
              </InlineNotice>
            )}
            <Toolbar end={<span className={s.progressText}>{rows.length} bài khớp bộ lọc</span>}>
              <FilterChips
                label="Lọc hàng chờ chấm"
                value={filters}
                onChange={changeFilters}
                options={FILTERS.map((f) => ({
                  ...f,
                  // "Cần xem kỹ" đếm bài còn phải xem (có cờ, chưa duyệt) — cùng số với thẻ Hôm nay (SRS 4.8 N8)
                  count: all.filter((x) => (f.value === "flag" ? Boolean(x.flag) && !isApproved(x) : f.value === "unapproved" ? !isApproved(x) : x.lateDays > 0)).length,
                }))}
              />
            </Toolbar>

            {picked.length > 0 && pickedUnapproved > 0 && (
              <InlineNotice tone="warning" title={`${pickedUnapproved} bài đang chọn chưa được duyệt`}>
                Mở từng bài, kiểm tra điểm và nhận xét rồi bấm `Duyệt bài`. Chỉ công bố được những bài đã duyệt.
              </InlineNotice>
            )}
            {pickedPublished > 0 && (
              <InlineNotice tone="info" compact>
                {pickedPublished} bài trong lựa chọn đã công bố trước đó; công bố lại không đổi điểm của sinh viên.
              </InlineNotice>
            )}

            {undo.node}

            <DataTable
              caption="Hàng chờ chấm bài"
              columns={columns}
              rows={rows}
              rowKey={(x) => x.id}
              rowHref={(x) => `/grading/${x.id}`}
              selection={role === "teacher" ? { selected, onChange: setSelected } : undefined}
              empty={
                <EmptyState
                  title="Không có bài nào khớp bộ lọc"
                  action={
                    <Button onClick={() => changeFilters([])}>Bỏ bộ lọc</Button>
                  }
                >
                  Hàng chờ đang trống với bộ lọc hiện tại. Bỏ bớt điều kiện để xem toàn bộ {all.length} bài đã nộp.
                </EmptyState>
              }
            />
          </Panel>
        ) : (
          <Panel>
            {creating && (
              <div className={s.form}>
                <Field label="Tên bài tập" required>
                  {(id) => <Input id={id} value={form.title} placeholder="Ví dụ: Phân tích nhật ký tấn công" onChange={(e) => setForm({ ...form, title: e.target.value })} />}
                </Field>
                <div className={s.formRow}>
                  <Field label="Hình thức">
                    {(id) => (
                      <Select id={id} value={form.kind} onChange={(e) => setForm({ ...form, kind: e.target.value as Draft["kind"] })}>
                        <option value="ESSAY">Tự luận</option>
                        <option value="QUIZ">Trắc nghiệm</option>
                      </Select>
                    )}
                  </Field>
                  <Field label="Hạn nộp">{(id) => <Input id={id} type="date" value={form.due} onChange={(e) => setForm({ ...form, due: e.target.value })} />}</Field>
                </div>
                <Field label="Yêu cầu với sinh viên" helper="Lưu nháp để soạn tiếp; sinh viên chưa thấy cho tới khi bạn giao bài.">
                  {(id) => <Textarea id={id} rows={3} value={form.note} onChange={(e) => setForm({ ...form, note: e.target.value })} />}
                </Field>
                <div className={s.formActions}>
                  <Button variant="primary" onClick={saveDraft} disabled={form.title.trim() === ""}>
                    Lưu nháp
                  </Button>
                  <Button variant="ghost" onClick={() => setCreating(false)}>
                    Để sau
                  </Button>
                </div>
              </div>
            )}

            {undo.node}

            <ActionList label="Bài tập của lớp">
              {ASSIGNMENTS.map((a) => (
                <ActionRow
                  key={a.id}
                  tone={a.state === "grading" ? "amber" : a.state === "open" ? "blue" : "green"}
                  title={`${a.id.toUpperCase()} · ${a.title}`}
                  context={a.note}
                  meta={`${ASSIGNMENT_KIND[a.kind]} · hạn ${fmtShortDate(a.due)} ${fmtTime(a.due)} · ${a.submitted}/${a.size} đã nộp`}
                  action={<StatusText tone={a.state === "published" ? "green" : a.state === "grading" ? "amber" : "blue"}>{ASSIGNMENT_STATE[a.state]}</StatusText>}
                />
              ))}
              {drafts.map((d) => (
                <ActionRow
                  key={d.id}
                  title={d.title}
                  context={d.note || "Chưa có mô tả yêu cầu"}
                  meta={`${ASSIGNMENT_KIND[d.kind]}${d.due ? ` · hạn ${d.due}` : ""} · chưa giao cho sinh viên`}
                  action={<StatusText tone="neutral">Bản nháp</StatusText>}
                />
              ))}
            </ActionList>
          </Panel>
        )}
      </PageState>

      <ConfirmIrreversible
        open={publishing}
        onClose={() => setPublishing(false)}
        onConfirm={publish}
        title="Công bố điểm Bài tập 03"
        consequence={`Công bố cho ${picked.length} sinh viên. Sinh viên thấy ngay điểm, nhận xét theo từng tiêu chí và đoạn trích từ bài của mình; điểm quá trình trong sổ điểm được tính lại. Bài của ${
          picked.find((x) => x.id === "sub-bt03-sv-2") ? "Trần Thu Uyên" : "nhóm đã chọn"
        } công bố ở mức ${fmtScore(bt03Total(bt03).total)} (đã trừ ${fmtScore(bt03Total(bt03).late)} nộp muộn).`}
        confirmLabel="Công bố điểm"
      />
    </Page>
  );
}

const ASSIGNMENT_KIND: Record<Assignment["kind"], string> = { ESSAY: "Tự luận", QUIZ: "Trắc nghiệm" };
const ASSIGNMENT_STATE: Record<Assignment["state"], string> = {
  published: "Đã công bố điểm",
  grading: "Đang chấm",
  open: "Đang mở",
  draft: "Bản nháp",
};
