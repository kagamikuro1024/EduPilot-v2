"use client";

import { useEffect, useMemo, useRef, useState } from "react";
import { ApiError, useCursorList } from "@/shared/data";
import { Button, type Column, DataTable, EmptyState, Field, InlineNotice, Input, PageState, Panel, PanelSection, Select, Skeleton, UndoLine } from "@/shared/ui";
import { classKey, type Member } from "@/features/members/classApi";
import { useAssignments } from "@/features/assignments/assignmentApi";
import { fmtLong } from "@/features/calendar/calendarApi";
import { MAIL_REASON_TEXT, SOURCE_LABEL, dismissSubmission, matchSubmission, submissionFileUrl, undismissSubmission, useIgnoredMail, useStudentVersions, useUnmatched, type MailInbound, type Unmatched } from "./submissionApi";
import s from "./Unmatched.module.css";

const CONFLICT = "Bài này vừa được xử lý ở nơi khác";
const TA_LOCKED = "Chỉ giảng viên sửa sau khi đã công bố điểm";

/** Tab `Chưa khớp` của `/grading` (US-P7-04 AC4, AC9): gán tay hoặc bỏ qua, có hoàn tác. */
export function UnmatchedTab({ courseId }: { courseId: string }) {
  const list = useUnmatched(courseId);
  const assignments = useAssignments(courseId);
  const [openId, setOpenId] = useState<string | null>(null);
  const [gone, setGone] = useState<Set<string>>(new Set());
  const [notice, setNotice] = useState<string | null>(null);
  const [line, setLine] = useState<{ text: string; undo: () => void } | null>(null);

  const rows = useMemo(() => list.items.filter((r) => !gone.has(r.id)), [list.items, gone]);
  const open = rows.find((r) => r.id === openId) ?? null;
  const reload = () => {
    setGone(new Set());
    void list.refetch();
  };

  function fail(e: unknown) {
    if (e instanceof ApiError && e.code === "VERSION_CONFLICT") {
      setNotice(CONFLICT);
      reload();
      return;
    }
    if (e instanceof ApiError && e.status === 403) {
      setNotice(TA_LOCKED);
      return;
    }
    setNotice(e instanceof ApiError ? e.userMessage : "Chưa thực hiện được. Hãy thử lại.");
  }

  async function openFile(row: Unmatched) {
    try {
      const url = await submissionFileUrl(courseId, row.id);
      window.open(url, "_blank", "noopener,noreferrer");
    } catch (e) {
      fail(e);
    }
  }

  const columns: Column<Unmatched>[] = [
    {
      key: "file",
      header: "Tệp",
      primary: true,
      render: (r) => (
        <Button
          variant="text"
          size="sm"
          onClick={(e) => {
            e.stopPropagation();
            void openFile(r);
          }}
        >
          {r.file_name}
        </Button>
      ),
    },
    { key: "source", header: "Nguồn", render: (r) => SOURCE_LABEL[r.source] ?? r.source },
    { key: "at", header: "Nhận lúc", render: (r) => <span className={s.at}>{fmtLong(r.received_at)}</span> },
    { key: "reason", header: "Lý do", render: (r) => r.reason_text },
    { key: "suggest", header: "Gợi ý", render: (r) => r.suggestion?.full_name ?? "" },
  ];

  return (
    <PageState
      query={{ isPending: list.isPending, isError: list.isError, error: list.error, data: rows, refetch: list.refetch }}
      isEmpty={(d) => (d as Unmatched[]).length === 0}
      loading={<Panel><Skeleton lines={8} /></Panel>}
      empty={<Panel><EmptyState title="Không có bài chưa khớp" /></Panel>}
      showTechnical
    >
      <Panel>
        {notice && (
          <InlineNotice tone="warning" compact action={<Button size="sm" onClick={() => setNotice(null)}>Đã hiểu</Button>}>
            {notice}
          </InlineNotice>
        )}
        {line && <UndoLine key={line.text} message={line.text} onUndo={line.undo} onDone={() => setLine(null)} />}
        <DataTable
          caption="Bài nộp chưa khớp"
          columns={columns}
          rows={rows}
          rowKey={(r) => r.id}
          onRowClick={(r) => setOpenId(r.id === openId ? null : r.id)}
          activeKey={openId ?? undefined}
          rowAttrs={(r) => ({ "data-part": "unmatched-row", "data-submission-id": r.id })}
        />
        {list.hasNextPage && (
          <div className={s.more}>
            <Button loading={list.isFetchingNextPage} onClick={() => void list.fetchNextPage()}>
              Tải thêm
            </Button>
          </div>
        )}
        {open && (
          <AssignBox
            key={open.id}
            courseId={courseId}
            row={open}
            assignments={assignments.items.map((a) => ({ id: a.id, title: a.title }))}
            onDone={(kind) => {
              setGone((g) => new Set([...g, open.id]));
              setOpenId(null);
              setNotice(null);
              if (kind === "dismiss") {
                setLine({
                  text: "Đã bỏ qua",
                  undo: () => {
                    setGone((g) => new Set([...g].filter((x) => x !== open.id)));
                    undismissSubmission(courseId, open.id).then(() => void list.refetch(), fail);
                  },
                });
              }
              void list.refetch();
            }}
            onError={fail}
          />
        )}
        <IgnoredMail courseId={courseId} />
      </Panel>
    </PageState>
  );
}

/** "Thư không xử lý được" (US-P7-05 AC9): chỉ hiện khi có ít nhất một thư bị bỏ qua. */
function IgnoredMail({ courseId }: { courseId: string }) {
  const mail = useIgnoredMail(courseId);
  if (mail.items.length === 0) return null;

  const columns: Column<MailInbound>[] = [
    { key: "from", header: "Người gửi", primary: true, render: (m) => m.from_email },
    { key: "subject", header: "Tiêu đề", render: (m) => m.subject },
    { key: "at", header: "Giờ nhận", render: (m) => <span className={s.at}>{fmtLong(m.received_at)}</span> },
    { key: "reason", header: "Lý do", render: (m) => (m.reason ? MAIL_REASON_TEXT[m.reason] ?? m.reason : "") },
  ];

  return (
    <PanelSection title="Thư không xử lý được">
      <DataTable
        caption="Thư không xử lý được"
        columns={columns}
        rows={mail.items}
        rowKey={(m) => m.id}
        rowAttrs={(m) => ({ "data-part": "mail-row", "data-mail-id": m.id })}
      />
      {mail.hasNextPage && (
        <div className={s.more}>
          <Button loading={mail.isFetchingNextPage} onClick={() => void mail.fetchNextPage()}>
            Tải thêm
          </Button>
        </div>
      )}
    </PanelSection>
  );
}

function AssignBox({ courseId, row, assignments, onDone, onError }: {
  courseId: string;
  row: Unmatched;
  assignments: Array<{ id: string; title: string }>;
  onDone: (kind: "match" | "dismiss") => void;
  onError: (e: unknown) => void;
}) {
  const [assignmentId, setAssignmentId] = useState(row.assignment_id ?? "");
  const [studentId, setStudentId] = useState(row.suggestion?.student_id ?? "");
  const [query, setQuery] = useState("");
  const [q, setQ] = useState("");
  const [reasonOpen, setReasonOpen] = useState(false);
  const [reason, setReason] = useState("");
  const [busy, setBusy] = useState(false);
  const running = useRef(false);

  useEffect(() => {
    const t = setTimeout(() => setQ(query.trim()), 250);
    return () => clearTimeout(t);
  }, [query]);

  const students = useCursorList<Member>(classKey(courseId, "members", "unmatched", q), `/courses/${courseId}/members`, {
    limit: 50,
    query: { status: "ACTIVE", role: "STUDENT", q: q || undefined },
    idOf: (m) => m.user_id,
  });
  const options = students.items;
  const versions = useStudentVersions(courseId, assignmentId, studentId).data?.versions ?? 0;
  // gợi ý của máy chủ có thể không nằm trong trang đang tải: vẫn giữ lựa chọn mặc định
  const known = options.some((m) => m.user_id === studentId);

  async function run(kind: "match" | "dismiss") {
    if (running.current) return; // bấm đúp: đúng một yêu cầu
    running.current = true;
    setBusy(true);
    try {
      if (kind === "match") await matchSubmission(courseId, row.id, { assignment_id: assignmentId, student_id: studentId, version: row.version });
      else await dismissSubmission(courseId, row.id, reason.trim(), row.version);
      onDone(kind);
    } catch (e) {
      onError(e);
    } finally {
      running.current = false;
      setBusy(false);
    }
  }

  return (
    <PanelSection title={`Gán ${row.file_name}`}>
      <div className={s.form} data-part="assign-box">
        <Field label="Bài tập">
          {(id) => (
            <Select id={id} value={assignmentId} onChange={(e) => setAssignmentId(e.target.value)}>
              <option value="">Chọn bài tập</option>
              {assignments.map((a) => (
                <option key={a.id} value={a.id}>
                  {a.title}
                </option>
              ))}
            </Select>
          )}
        </Field>
        <Field label="Tìm sinh viên">
          {(id) => <Input id={id} value={query} placeholder="Tên hoặc MSSV" onChange={(e) => setQuery(e.target.value)} />}
        </Field>
        <Field label="Sinh viên">
          {(id) => (
            <Select id={id} value={studentId} onChange={(e) => setStudentId(e.target.value)}>
              <option value="">Chọn sinh viên</option>
              {!known && row.suggestion && (
                <option value={row.suggestion.student_id}>{row.suggestion.full_name}</option>
              )}
              {options.map((m) => (
                <option key={m.user_id} value={m.user_id}>
                  {`${m.full_name} · ${m.student_code}`}
                </option>
              ))}
            </Select>
          )}
        </Field>
        {reasonOpen && (
          <Field label="Lý do bỏ qua">
            {(id) => <Input id={id} value={reason} maxLength={200} placeholder="Nộp nhầm lớp" onChange={(e) => setReason(e.target.value)} />}
          </Field>
        )}
        {versions > 0 && <p className={s.note} data-part="student-versions">{`Sinh viên này đã có ${versions} phiên bản`}</p>}
        <div className={s.actions}>
          <Button variant="primary" disabled={!assignmentId || !studentId} loading={busy} onClick={() => void run("match")}>
            Gán
          </Button>
          {reasonOpen ? (
            <Button disabled={reason.trim().length === 0} loading={busy} onClick={() => void run("dismiss")}>
              Xác nhận bỏ qua
            </Button>
          ) : (
            <Button variant="ghost" onClick={() => setReasonOpen(true)}>
              Bỏ qua
            </Button>
          )}
        </div>
      </div>
    </PanelSection>
  );
}
