"use client";

import { useQuery, useQueryClient } from "@tanstack/react-query";
import Link from "next/link";
import { useRouter } from "next/navigation";
import { useState } from "react";
import { ApiError, ApiErrorNotice, apiClient } from "@/shared/data";
import { useClassCourse } from "@/features/members/classApi";
import { useSession } from "@/shared/session/session";
import { useUndoLine } from "@/shared/lib/useUndoLine";
import { Button, ButtonLink, ConfirmIrreversible, EmptyState, Field, InlineNotice, Input, type MenuItem, OverflowMenu, Page, PageHeader, PageState, Panel, Skeleton, StatusText, Tabs } from "@/shared/ui";
import { eKey, ePath, fmtClock, fmtWhen, fromInput, KIND_LABEL, STATUS_LABEL, STATUS_TONE, toInput, type ExamDetail, type Problem } from "./examApi";
import { ExamInfo } from "./ExamInfo";
import { ExamItems } from "./ExamItems";
import { ExamPreview } from "./ExamPreview";
import s from "./Exam.module.css";

type Tab = "info" | "items" | "preview";

/** `/exams/[id]` (Giảng viên / TA): soạn bài thi nháp, lên lịch, gia hạn khi đang mở (US-PE-04 AC12). */
export function ExamEditor({ id }: { id: string }) {
  const cc = useClassCourse();
  if (cc.state === "loading") return <Page width="wide"><PageHeader title="Bài thi" back={{ href: "/exams", label: "Bài thi" }} /><Panel><Skeleton lines={8} /></Panel></Page>;
  if (cc.state === "none")
    return (
      <Page width="wide">
        <PageHeader title="Bài thi" back={{ href: "/exams", label: "Bài thi" }} />
        <Panel><EmptyState title="Chưa chọn lớp">Chọn một lớp ở thanh trên để mở bài thi.</EmptyState></Panel>
      </Page>
    );
  return <Editor courseId={cc.course.id} id={id} isTeacher={cc.canManage} />;
}

/** Mã lỗi lên lịch → chỗ cần sửa: tab của trang này hoặc ngân hàng câu hỏi. */
function targetOf(p: Problem): { tab?: Tab; label: string } {
  switch (p.code) {
    case "OPENS_IN_PAST":
    case "CLOSES_BEFORE_OPENS":
    case "DURATION_TOO_SHORT":
    case "DURATION_EXCEEDS_WINDOW":
    case "LIMIT_OUT_OF_RANGE":
      return { tab: "info", label: "Sửa ở Thông tin" };
    case "VALUE_REQUIRED":
      return { tab: "info", label: "Điền ở Thông tin" };
    case "NO_ITEMS":
      return { tab: "items", label: "Thêm câu hỏi" };
    default:
      return { label: "Mở ngân hàng câu hỏi" }; // ITEM_NOT_APPROVED, CODE_TESTS_MISSING, TOTAL_WEIGHT_ZERO, REFERENCE_NOT_VERIFIED, CODE_TIME_BUDGET_EXCEEDED
  }
}

function Editor({ courseId, id, isTeacher }: { courseId: string; id: string; isTeacher: boolean }) {
  const router = useRouter();
  const qc = useQueryClient();
  const { identity } = useSession();
  const key = eKey(courseId, id);
  const q = useQuery({ queryKey: key, queryFn: async ({ signal }) => (await apiClient.get<ExamDetail>(`${ePath(courseId)}/${id}`, { signal })).data });
  const [tab, setTab] = useState<Tab>("info");
  const undo = useUndoLine();
  const [busy, setBusy] = useState(false);
  const [problems, setProblems] = useState<Problem[] | null>(null);
  const [err, setErr] = useState<unknown>(null);
  const [extending, setExtending] = useState(false);
  const [confirmDelete, setConfirmDelete] = useState(false);

  const d = q.data;
  const put = (next: ExamDetail) => {
    qc.setQueryData(key, next);
    void qc.invalidateQueries({ queryKey: eKey(courseId, "list") });
  };
  const base = `${ePath(courseId)}/${id}`;

  async function run(op: () => Promise<void>) {
    setBusy(true);
    setErr(null);
    try {
      await op();
    } catch (e) {
      if (e instanceof ApiError && e.status === 422 && Array.isArray(e.details)) setProblems(e.details as Problem[]);
      else setErr(e);
    } finally {
      setBusy(false);
    }
  }

  const unschedule = () =>
    run(async () => {
      put((await apiClient.post<ExamDetail>(`${base}/unschedule`)).data);
      undo.push("Đã bỏ lịch · bài về trạng thái nháp");
    });
  const schedule = () =>
    run(async () => {
      const r = (await apiClient.post<ExamDetail>(`${base}/schedule`)).data;
      put(r);
      setProblems(null);
      undo.push(`Đã lên lịch · mở ${r.opens_at ? fmtWhen(r.opens_at) : ""}. Sinh viên sẽ nhận thông báo`, () => void unschedule());
    });
  const clone = () =>
    run(async () => {
      const r = (await apiClient.post<ExamDetail>(`${base}/clone`)).data;
      void qc.invalidateQueries({ queryKey: eKey(courseId, "list") });
      router.push(`/exams/${r.id}`);
    });
  const remove = () =>
    run(async () => {
      await apiClient.delete(base);
      void qc.invalidateQueries({ queryKey: eKey(courseId, "list") });
      router.push("/exams");
    });

  if (!d) {
    return (
      <Page width="wide">
        <PageHeader title="Bài thi" back={{ href: "/exams", label: "Bài thi" }} />
        <PageState query={q} showTechnical loading={<Panel><Skeleton lines={8} /></Panel>}>
          <Panel><Skeleton lines={8} /></Panel>
        </PageState>
      </Page>
    );
  }

  const st = d.effective_status;
  const canUnschedule = isTeacher && d.status === "SCHEDULED" && st === "SCHEDULED" && d.attempts.started === 0;
  const menu: MenuItem[] = [{ label: "Nhân bản bài thi", hint: "Bản nháp mới, không giờ", onSelect: () => void clone() }];
  if (isTeacher && (st === "SCHEDULED" || st === "OPEN")) menu.push({ label: "Gia hạn", onSelect: () => setExtending(true) });
  if (canUnschedule) menu.push({ label: "Bỏ lịch", hint: "Về nháp, thu hồi thông báo", onSelect: () => void unschedule() });
  if (isTeacher && d.status === "DRAFT") menu.push({ label: "Xoá bài thi nháp", danger: true, onSelect: () => setConfirmDelete(true) });

  const primary =
    d.status === "DRAFT" ? (
      isTeacher ? (
        <Button variant="primary" onClick={() => void schedule()} loading={busy}>Lên lịch</Button>
      ) : (
        <span className={s.noSchedule}>Chỉ giảng viên lên lịch được</span>
      )
    ) : st === "OPEN" && isTeacher ? (
      <Button variant="primary" onClick={() => setExtending(true)} aria-expanded={extending}>Gia hạn</Button>
    ) : st === "CLOSED" || st === "PUBLISHED" ? (
      <ButtonLink variant="primary" href={`/exams/${id}/results`}>Xem kết quả</ButtonLink>
    ) : null;

  return (
    <Page width="wide">
      <PageHeader
        title={d.title}
        back={{ href: "/exams", label: "Bài thi" }}
        description={
          <span className={s.headMeta}>
            <StatusText tone={STATUS_TONE[st]}>{STATUS_LABEL[st]}</StatusText>
            <span>{KIND_LABEL[d.kind]}</span>
            {d.opens_at && d.closes_at && <span>{fmtClock(d.opens_at)} – {fmtClock(d.closes_at)}</span>}
            {d.attempts.started > 0 && <span>{d.attempts.started} sinh viên đã bắt đầu</span>}
          </span>
        }
        actions={
          <>
            {primary}
            <OverflowMenu items={menu} label="Thêm hành động cho bài thi" />
          </>
        }
      />
      {undo.node}
      {err !== null && <ApiErrorNotice error={err} showTechnical onRetry={undefined} />}
      {extending && <Panel><Extend base={base} detail={d} onDone={(r) => { put(r); setExtending(false); undo.push(`Đã gia hạn đến ${r.closes_at ? fmtClock(r.closes_at) : ""}`); }} onCancel={() => setExtending(false)} /></Panel>}

      <Tabs<Tab>
        label="Phần của bài thi"
        value={tab}
        onChange={setTab}
        options={[{ value: "info", label: "Thông tin" }, { value: "items", label: "Câu hỏi", count: d.items_count }, { value: "preview", label: "Xem trước" }]}
      />
      <div className={s.tabBody}>
        <Panel>
          {problems && (
            <InlineNotice tone="danger" title={`Cần sửa ${problems.length} việc trước khi lên lịch`}>
              <ul className={s.problemList} aria-label="Việc cần sửa">
                {problems.map((p, i) => {
                  const t = targetOf(p);
                  return (
                    <li key={i}>
                      <span>{p.message}</span>{" "}
                      {t.tab ? <button type="button" className={s.fix} onClick={() => setTab(t.tab!)}>{t.label}</button> : <Link className={s.fix} href="/questions">{t.label}</Link>}
                    </li>
                  );
                })}
              </ul>
            </InlineNotice>
          )}
        {tab === "info" && <ExamInfo key={d.id} courseId={courseId} userId={identity?.sub} detail={d} onSaved={(r, note) => { put(r); if (note) undo.push(note); }} />}
        {tab === "items" && <ExamItems courseId={courseId} detail={d} onSaved={(r, note) => { put(r); setProblems(null); if (note) undo.push(note); }} />}
        {tab === "preview" && <ExamPreview courseId={courseId} examId={id} version={d.version} />}
        </Panel>
      </div>

      <ConfirmIrreversible
        open={confirmDelete}
        onClose={() => setConfirmDelete(false)}
        title="Xoá bài thi nháp?"
        consequence={`Bài thi nháp có ${d.items_count} câu sẽ bị xoá.`}
        confirmLabel="Xoá bài thi"
        loading={busy}
        error={err ? <ApiErrorNotice error={err} /> : undefined}
        onConfirm={() => void remove()}
      />
    </Page>
  );
}

/** Gia hạn khi đang mở: chỉ lùi giờ đóng muộn hơn, tối đa 24 giờ; sinh viên đang làm được tính lại hạn. Mốc nhanh +15 / +30 / +60 phút. */
function Extend({ base, detail, onDone, onCancel }: { base: string; detail: ExamDetail; onDone: (d: ExamDetail) => void; onCancel: () => void }) {
  const [v, setV] = useState(toInput(detail.closes_at));
  const [pending, setPending] = useState(false);
  const [err, setErr] = useState<unknown>(null);
  const plus = (min: number) => {
    const t = detail.closes_at ? new Date(new Date(detail.closes_at).getTime() + min * 60_000).toISOString() : null;
    setV(toInput(t));
  };
  async function submit(e: React.FormEvent) {
    e.preventDefault();
    const closes = fromInput(v);
    if (!closes) return;
    setPending(true);
    setErr(null);
    try {
      onDone((await apiClient.post<ExamDetail>(`${base}/extend`, { closes_at: closes })).data);
    } catch (x) {
      setErr(x);
    } finally {
      setPending(false);
    }
  }
  const fe = err instanceof ApiError && Array.isArray(err.details) ? (err.details as Problem[])[0]?.message : undefined;
  return (
    <form className={s.extend} onSubmit={submit} noValidate aria-label="Gia hạn bài thi">
      <Field label="Giờ đóng mới" helper="Muộn hơn giờ đóng hiện tại, tối đa thêm 24 giờ. Giờ Việt Nam." error={fe} className={s.grow}>
        {(id, by) => <Input id={id} aria-describedby={by} type="datetime-local" invalid={Boolean(fe)} value={v} onChange={(e) => setV(e.target.value)} />}
      </Field>
      <span className={s.chips}>
        {[15, 30, 60].map((m) => <Button key={m} size="sm" variant="ghost" onClick={() => plus(m)}>+{m} phút</Button>)}
      </span>
      <Button type="submit" loading={pending}>Gia hạn</Button>
      <Button variant="text" onClick={onCancel}>Bỏ qua</Button>
      {err !== null && !fe && <ApiErrorNotice error={err} showTechnical />}
    </form>
  );
}
