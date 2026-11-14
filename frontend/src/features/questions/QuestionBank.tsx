"use client";

import { Plus } from "lucide-react";
import { useSearchParams } from "next/navigation";
import { useEffect, useMemo, useState } from "react";
import { useCursorList } from "@/shared/data";
import { useClassCourse } from "@/features/members/classApi";
import { useSession } from "@/shared/session/session";
import { Button, type Column, DataTable, Drawer, EmptyState, Field, InlineNotice, Input, MenuList, Page, PageHeader, PageState, Panel, Select, Skeleton, StatusText, Toolbar } from "@/shared/ui";
import { useUndoLine } from "@/shared/lib/useUndoLine";
import { QuestionForm } from "./QuestionForm";
import { QuestionReview } from "./QuestionReview";
import { BulkBar } from "./BulkBar";
import { ExtractPanel, GeneratePanel } from "./CreateFlows";
import { SuggestPanel } from "./SuggestPanel";
import { DIFF_LABEL, ORIGIN_LABEL, qKey, qPath, REVIEW_LABEL, REVIEW_TONE, TYPE_LABEL, type Difficulty, type QType, type QuestionRow, type Review } from "./questionsApi";
import s from "./Questions.module.css";

/** Ngân hàng câu hỏi thật (US-PE-03): bảng có bộ lọc, nút chính `Tạo câu hỏi`, hàng mở Drawer `QuestionReview`. Chỉ Giảng viên / TA; sinh viên không có đường vào. */
export function QuestionBank() {
  const cc = useClassCourse();
  if (cc.state === "loading") return <Page kind="work"><PageHeader title="Ngân hàng câu hỏi" /><Panel><Skeleton lines={8} /></Panel></Page>;
  if (cc.state === "none")
    return (
      <Page kind="work">
        <PageHeader title="Ngân hàng câu hỏi" />
        <Panel><EmptyState title="Chưa chọn lớp" /></Panel>
      </Page>
    );
  return <Bank courseId={cc.course.id} code={cc.course.class_code} isTeacher={cc.canManage} />;
}

type Creating = null | "MCQ" | "CODE" | "AI" | "EXTRACT" | "GENERATE";

/** Menu Tạo câu hỏi (US-P9-04 AC2): «Soạn tay» là nhóm cha mở ra Câu trắc nghiệm / Bài lập trình; mục chỉ có nhãn. */
function CreateMenu({ close, pick }: { close: () => void; pick: (c: Creating) => void }) {
  const [hand, setHand] = useState(false);
  const go = (c: Creating) => () => {
    pick(c);
    close();
  };
  if (hand) {
    return (
      <MenuList
        key="hand"
        autoFocus
        items={[
          { label: "Câu trắc nghiệm", onSelect: go("MCQ") },
          { label: "Bài lập trình", onSelect: go("CODE") },
        ]}
      />
    );
  }
  return (
    <MenuList
      key="root"
      autoFocus
      items={[
        { label: "Soạn tay", onSelect: () => setHand(true) },
        { label: "Trích từ đề", onSelect: go("EXTRACT") },
        { label: "Sinh từ tài liệu", onSelect: go("GENERATE") },
        { label: "Gợi ý từ AI", onSelect: go("AI") },
      ]}
    />
  );
}

function Bank({ courseId, code, isTeacher }: { courseId: string; code: string; isTeacher: boolean }) {
  const params = useSearchParams();
  const { identity } = useSession();
  const [status, setStatus] = useState(params.get("review_status") ?? "");
  const [type, setType] = useState("");
  const [difficulty, setDifficulty] = useState("");
  const [origin, setOrigin] = useState("");
  const [topic, setTopic] = useState("");
  const [text, setText] = useState("");
  const [q, setQ] = useState("");
  const [openId, setOpenId] = useState<string | null>(null);
  const [creating, setCreating] = useState<Creating>(null);
  const [sel, setSel] = useState<Set<string>>(new Set());
  const [failed, setFailed] = useState<Map<string, string>>(new Map());
  const [hidden, setHidden] = useState<Set<string>>(new Set()); // đang chờ gửi "Loại" (cửa sổ hoàn tác)
  const undo = useUndoLine();

  useEffect(() => {
    const t = setTimeout(() => setQ(text.trim()), 250);
    return () => clearTimeout(t);
  }, [text]);

  const list = useCursorList<QuestionRow>(qKey(courseId, "list", status, type, difficulty, origin, topic, q), qPath(courseId), {
    limit: 30,
    query: { review_status: status || undefined, type: type || undefined, difficulty: difficulty || undefined, origin: origin || undefined, topic: topic || undefined, q: q || undefined } });
  const topics = useMemo(() => [...new Set(list.items.map((r) => r.topic))].sort((a, b) => a.localeCompare(b, "vi")), [list.items]);
  const filtered = Boolean(status || type || difficulty || origin || topic || q);

  const columns: Column<QuestionRow>[] = [
    {
      key: "title",
      header: "Câu hỏi",
      primary: true,
      render: (r) => (
        <span className={s.titleCell}>
          <span className={s.text}>{r.title}</span>
          <span className={s.meta}>{r.topic}</span>
          {failed.get(r.id) && <span className={s.rowFail} role="alert">{failed.get(r.id)}</span>}
        </span>
      ) },
    { key: "type", header: "Loại", width: "120px", render: (r) => <span className={s.meta}>{TYPE_LABEL[r.type]}</span> },
    { key: "difficulty", header: "Độ khó", width: "90px", render: (r) => <span className={s.meta}>{DIFF_LABEL[r.difficulty]}</span> },
    { key: "status", header: "Trạng thái", width: "130px", render: (r) => <StatusText tone={REVIEW_TONE[r.review_status]}>{REVIEW_LABEL[r.review_status]}</StatusText> },
    { key: "origin", header: "Nguồn", width: "100px", render: (r) => <span className={s.meta}>{ORIGIN_LABEL[r.origin]}</span> },
    { key: "used", header: "Dùng trong", width: "110px", align: "end", render: (r) => <span className={s.meta}>{r.used_in_exams > 0 ? `${r.used_in_exams} bài thi` : "—"}</span> },
  ];

  const clearFilters = () => {
    setText("");
    setStatus("");
    setTopic("");
    setDifficulty("");
    setType("");
    setOrigin("");
  };
  const rows = useMemo(() => list.items.filter((r) => !hidden.has(r.id)), [list.items, hidden]);
  const pending = rows.filter((r) => r.review_status === "PENDING").length;
  return (
    <Page kind="work">
      <PageHeader
        title="Ngân hàng câu hỏi"
        meta={[pending > 0 ? <span key="m0">{pending} câu chờ duyệt</span> : undefined]}
        primary={{ label: "Tạo câu hỏi", icon: <Plus aria-hidden />, menu: (close) => <CreateMenu close={close} pick={setCreating} /> }}
      />
      {undo.node}
      {creating === "EXTRACT" && <ExtractPanel courseId={courseId} onCancel={() => setCreating(null)} onDone={(n) => { void list.refetch(); undo.push(n > 0 ? `Đã trích ${n} câu · đang chờ bạn duyệt` : "Chưa trích được câu nào. Xem lý do bên dưới."); }} />}
      {creating === "GENERATE" && <GeneratePanel courseId={courseId} onCancel={() => setCreating(null)} onDone={(n) => { void list.refetch(); undo.push(n > 0 ? `Đã sinh ${n} câu · đang chờ bạn duyệt` : "Chưa sinh được câu nào. Xem lý do bên dưới."); }} />}
      {creating === "AI" && <SuggestPanel courseId={courseId} onCancel={() => setCreating(null)} onDone={(n) => { setCreating(null); undo.push(n > 0 ? `AI đã soạn ${n} câu nháp · đang chờ bạn duyệt` : "AI chưa có câu nào hợp lệ — thử đổi chủ đề hoặc thêm văn bản nguồn"); }} />}

      <Panel>
      <Toolbar>
        <Field label="Tìm câu hỏi" className={s.search}>{(id) => <Input id={id} type="search" placeholder="Tiêu đề…" value={text} maxLength={100} onChange={(e) => setText(e.target.value)} />}</Field>
        <Field label="Trạng thái">{(id) => <Select id={id} value={status} onChange={(e) => setStatus(e.target.value)}><option value="">Tất cả</option>{(Object.keys(REVIEW_LABEL) as Review[]).map((k) => <option key={k} value={k}>{REVIEW_LABEL[k]}</option>)}</Select>}</Field>
        <Field label="Chủ đề">{(id) => <Select id={id} value={topic} onChange={(e) => setTopic(e.target.value)}><option value="">Tất cả</option>{topics.map((t) => <option key={t} value={t}>{t}</option>)}</Select>}</Field>
        <Field label="Độ khó">{(id) => <Select id={id} value={difficulty} onChange={(e) => setDifficulty(e.target.value)}><option value="">Tất cả</option>{(Object.keys(DIFF_LABEL) as Difficulty[]).map((k) => <option key={k} value={k}>{DIFF_LABEL[k]}</option>)}</Select>}</Field>
        <Field label="Loại">{(id) => <Select id={id} value={type} onChange={(e) => setType(e.target.value)}><option value="">Tất cả</option>{(Object.keys(TYPE_LABEL) as QType[]).map((k) => <option key={k} value={k}>{TYPE_LABEL[k]}</option>)}</Select>}</Field>
        <Field label="Nguồn">{(id) => <Select id={id} value={origin} onChange={(e) => setOrigin(e.target.value)}><option value="">Tất cả</option><option value="MANUAL">Soạn tay</option><option value="AI_DRAFT">AI gợi ý</option><option value="EXTRACTED">Trích từ đề</option><option value="GENERATED">Sinh từ tài liệu</option></Select>}</Field>
      </Toolbar>

      {sel.size > 0 && <BulkBar courseId={courseId} selected={sel} topics={topics} onSelected={setSel} onFailed={setFailed} onHide={setHidden} onChanged={() => void list.refetch()} undo={undo} />}

      <PageState
        query={list}
        isEmpty={() => list.items.length === 0}
        loading={<Skeleton lines={10} />}
        empty={
          filtered ? (
            <EmptyState title="Không có câu hỏi khớp bộ lọc" action={<Button onClick={clearFilters}>Xoá bộ lọc</Button>} />
          ) : (
            <EmptyState title="Chưa có câu hỏi" />
          )
        }
        showTechnical
      >
        <DataTable
          caption={`Ngân hàng câu hỏi lớp ${code}`}
          columns={columns}
          rows={rows}
          rowKey={(r) => r.id}
          selection={{ selected: sel, onChange: setSel }}
          onRowClick={(r) => setOpenId(r.id)}
          activeKey={openId ?? undefined}
          mobile="list"
          pagination={undefined}
        />
        {list.hasNextPage && <div className={s.more}><Button onClick={() => void list.fetchNextPage()} loading={list.isFetchingNextPage}>Xem thêm</Button></div>}
      </PageState>
      </Panel>

      <Drawer open={creating === "MCQ" || creating === "CODE"} wide onClose={() => setCreating(null)} title={creating === "CODE" ? "Bài lập trình mới" : "Câu trắc nghiệm mới"}>
        {(creating === "MCQ" || creating === "CODE") && (
          <QuestionForm
            courseId={courseId}
            userId={identity?.sub}
            createType={creating === "CODE" ? "CODE" : "MCQ_SINGLE"}
            onCancel={() => setCreating(null)}
            onSaved={(d) => {
              setCreating(null);
              void list.refetch();
              setOpenId(d.id);
              undo.push(`Đã tạo câu hỏi "${d.title}" ở trạng thái Nháp`);
            }}
          />
        )}
      </Drawer>

      {openId && (
        <QuestionReview
          key={openId}
          courseId={courseId}
          id={openId}
          userId={identity?.sub}
          canEditTests={isTeacher}
          onClose={() => setOpenId(null)}
          onOpen={setOpenId}
          onChanged={(note) => {
            void list.refetch();
            if (note) undo.push(note);
          }}
        />
      )}
      {list.isError && list.items.length > 0 && <InlineNotice tone="warning">Không tải thêm được. Danh sách đang hiện là dữ liệu đã có.</InlineNotice>}
    </Page>
  );
}
