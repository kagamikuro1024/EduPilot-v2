"use client";

import { CircleHelp, Plus } from "lucide-react";
import { useSearchParams } from "next/navigation";
import { useEffect, useMemo, useState } from "react";
import { useCursorList } from "@/shared/data";
import { useClassCourse } from "@/features/members/classApi";
import { useSession } from "@/shared/session/session";
import { Button, Drawer, EmptyState, Field, InlineNotice, Input, MenuList, Page, PageHeader, PageState, Popover, Select, Skeleton, StatusText, Toolbar, DataTable, type Column } from "@/shared/ui";
import { useUndoLine } from "@/shared/lib/useUndoLine";
import { QuestionForm } from "./QuestionForm";
import { QuestionReview } from "./QuestionReview";
import { SuggestPanel } from "./SuggestPanel";
import { DIFF_LABEL, ORIGIN_LABEL, qKey, qPath, REVIEW_LABEL, REVIEW_TONE, TYPE_LABEL, type Difficulty, type QType, type QuestionRow, type Review } from "./questionsApi";
import s from "./Questions.module.css";

/** Ngân hàng câu hỏi thật (US-PE-03): bảng có bộ lọc, nút chính `Tạo câu hỏi`, hàng mở Drawer `QuestionReview`. Chỉ Giảng viên / TA; sinh viên không có đường vào. */
export function QuestionBank() {
  const cc = useClassCourse();
  if (cc.state === "loading") return <Page width="wide"><PageHeader title="Ngân hàng câu hỏi" /><Skeleton lines={8} /></Page>;
  if (cc.state === "none")
    return (
      <Page width="wide">
        <PageHeader title="Ngân hàng câu hỏi" />
        <EmptyState title="Chưa chọn lớp">Chọn một lớp ở thanh trên để xem ngân hàng câu hỏi của lớp.</EmptyState>
      </Page>
    );
  return <Bank courseId={cc.course.id} code={cc.course.class_code} isTeacher={cc.canManage} />;
}

type Creating = null | "MCQ" | "CODE" | "AI";

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
  const undo = useUndoLine();

  useEffect(() => {
    const t = setTimeout(() => setQ(text.trim()), 250);
    return () => clearTimeout(t);
  }, [text]);

  const list = useCursorList<QuestionRow>(qKey(courseId, "list", status, type, difficulty, origin, topic, q), qPath(courseId), {
    limit: 30,
    query: { review_status: status || undefined, type: type || undefined, difficulty: difficulty || undefined, origin: origin || undefined, topic: topic || undefined, q: q || undefined },
  });
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
        </span>
      ),
    },
    { key: "type", header: "Loại", width: "120px", render: (r) => <span className={s.meta}>{TYPE_LABEL[r.type]}</span> },
    { key: "difficulty", header: "Độ khó", width: "90px", render: (r) => <span className={s.meta}>{DIFF_LABEL[r.difficulty]}</span> },
    { key: "status", header: "Trạng thái", width: "130px", render: (r) => <StatusText tone={REVIEW_TONE[r.review_status]}>{REVIEW_LABEL[r.review_status]}</StatusText> },
    { key: "origin", header: "Nguồn", width: "100px", render: (r) => <span className={s.meta}>{ORIGIN_LABEL[r.origin]}</span> },
    { key: "used", header: "Dùng trong", width: "110px", align: "end", render: (r) => <span className={s.meta}>{r.used_in_exams > 0 ? `${r.used_in_exams} bài thi` : "—"}</span> },
  ];

  const pending = list.items.filter((r) => r.review_status === "PENDING").length;
  const createMenu = (close: () => void) => (
    <MenuList
      autoFocus
      onPicked={close}
      items={[
        { label: "Câu trắc nghiệm", onSelect: () => setCreating("MCQ") },
        { label: "Bài lập trình", onSelect: () => setCreating("CODE") },
        { label: "Gợi ý từ AI", hint: "Soạn nháp, bạn duyệt", onSelect: () => setCreating("AI") },
      ]}
    />
  );

  return (
    <Page width="wide">
      <PageHeader
        title="Ngân hàng câu hỏi"
        description={`Câu hỏi dùng cho bài thi và luyện đề của lớp ${code}. Chỉ giảng viên và trợ giảng thấy.`}
        meta={pending > 0 ? <span>{pending} câu chờ duyệt</span> : undefined}
        actions={
          <Popover
            label="Tạo câu hỏi"
            width={240}
            trigger={(p) => (
              <Button variant="primary" icon={<Plus aria-hidden />} onClick={p.toggle} aria-expanded={p["aria-expanded"]} aria-haspopup="true">
                Tạo câu hỏi
              </Button>
            )}
          >
            {createMenu}
          </Popover>
        }
      />
      {undo.node}
      {creating === "AI" && <SuggestPanel courseId={courseId} onCancel={() => setCreating(null)} onDone={(n) => { setCreating(null); undo.push(n > 0 ? `AI đã soạn ${n} câu nháp · đang chờ bạn duyệt` : "AI chưa có câu nào hợp lệ — thử đổi chủ đề hoặc thêm văn bản nguồn"); }} />}

      <Toolbar>
        <Field label="Tìm câu hỏi" className={s.search}>{(id) => <Input id={id} type="search" placeholder="Tiêu đề…" value={text} maxLength={100} onChange={(e) => setText(e.target.value)} />}</Field>
        <Field label="Trạng thái">{(id) => <Select id={id} value={status} onChange={(e) => setStatus(e.target.value)}><option value="">Tất cả</option>{(Object.keys(REVIEW_LABEL) as Review[]).map((k) => <option key={k} value={k}>{REVIEW_LABEL[k]}</option>)}</Select>}</Field>
        <Field label="Chủ đề">{(id) => <Select id={id} value={topic} onChange={(e) => setTopic(e.target.value)}><option value="">Tất cả</option>{topics.map((t) => <option key={t} value={t}>{t}</option>)}</Select>}</Field>
        <Field label="Độ khó">{(id) => <Select id={id} value={difficulty} onChange={(e) => setDifficulty(e.target.value)}><option value="">Tất cả</option>{(Object.keys(DIFF_LABEL) as Difficulty[]).map((k) => <option key={k} value={k}>{DIFF_LABEL[k]}</option>)}</Select>}</Field>
        <Field label="Loại">{(id) => <Select id={id} value={type} onChange={(e) => setType(e.target.value)}><option value="">Tất cả</option>{(Object.keys(TYPE_LABEL) as QType[]).map((k) => <option key={k} value={k}>{TYPE_LABEL[k]}</option>)}</Select>}</Field>
        <Field label="Nguồn">{(id) => <Select id={id} value={origin} onChange={(e) => setOrigin(e.target.value)}><option value="">Tất cả</option><option value="MANUAL">Soạn tay</option><option value="AI_DRAFT">AI</option></Select>}</Field>
      </Toolbar>

      <PageState
        query={list}
        isEmpty={() => list.items.length === 0}
        loading={<Skeleton lines={10} />}
        empty={
          filtered ? (
            <EmptyState title="Không có câu hỏi nào khớp bộ lọc" icon={<CircleHelp aria-hidden />}>Bỏ bớt bộ lọc để xem thêm.</EmptyState>
          ) : (
            <EmptyState title="Chưa có câu hỏi nào" icon={<CircleHelp aria-hidden />} action={<Popover label="Tạo câu hỏi" width={240} trigger={(p) => <Button variant="primary" onClick={p.toggle} aria-expanded={p["aria-expanded"]} aria-haspopup="true">Tạo câu hỏi</Button>}>{createMenu}</Popover>}>
              Chưa có câu hỏi nào. Tạo câu đầu tiên hoặc nhờ AI gợi ý nháp.
            </EmptyState>
          )
        }
        showTechnical
      >
        <DataTable
          caption={`Ngân hàng câu hỏi lớp ${code}`}
          columns={columns}
          rows={list.items}
          rowKey={(r) => r.id}
          onRowClick={(r) => setOpenId(r.id)}
          activeKey={openId ?? undefined}
          mobile="list"
          pagination={undefined}
        />
        {list.hasNextPage && <div className={s.more}><Button onClick={() => void list.fetchNextPage()} loading={list.isFetchingNextPage}>Xem thêm</Button></div>}
      </PageState>

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
