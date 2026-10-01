"use client";

import { useEffect, useMemo, useState } from "react";
import { Check, CircleHelp, Pencil, SlidersHorizontal, Sparkles, X } from "lucide-react";
import {
  Button,
  type Column,
  DataTable,
  Drawer,
  EmptyState,
  Field,
  FilterChips,
  InlineNotice,
  OverflowMenu,
  Page,
  PageHeader,
  PageState,
  Popover,
  Select,
  Skeleton,
  StatusText,
  Textarea,
  Toolbar,
} from "@/shared/ui";
import {
  DIFFICULTY_LABEL,
  GENERATED_QUESTIONS,
  KIND_LABEL,
  QUESTIONS,
  SOURCE_LABEL,
  STATUS_LABEL,
  TOPICS,
  type QDifficulty,
  type QKind,
  type QSource,
  type QStatus,
  type Question,
} from "@/mock/questions";
import { useDemoSlice } from "@/shared/state/demo";
import { useUndoLine } from "@/shared/lib/useUndoLine";
import s from "./Questions.module.css";

const STATUS_TONE: Record<QStatus, "amber" | "green" | "neutral"> = { pending: "amber", approved: "green", rejected: "neutral" };

export function QuestionBank() {
  const [overrides, setOverrides] = useDemoSlice<Record<string, QStatus>>("questions.status", {});
  const [edits, setEdits] = useDemoSlice<Record<string, string>>("questions.edits", {});
  const [generated, setGenerated] = useDemoSlice<string[]>("questions.generated", []);
  const [states, setStates] = useState<QStatus[]>([]);
  const [topic, setTopic] = useState("");
  const [difficulty, setDifficulty] = useState("");
  const [kind, setKind] = useState("");
  const [source, setSource] = useState("");
  const [openId, setOpenId] = useState<string | null>(null);
  const [editing, setEditing] = useState(false);
  const [draft, setDraft] = useState("");
  const [genTopic, setGenTopic] = useState(TOPICS[0]);
  const [generating, setGenerating] = useState(false);
  const [progress, setProgress] = useState(0);
  const [genOpen, setGenOpen] = useState(false);
  const undo = useUndoLine();

  const items = useMemo<Question[]>(
    () =>
      [...GENERATED_QUESTIONS.filter((q) => generated.includes(q.id)), ...QUESTIONS].map((q) => ({
        ...q,
        text: edits[q.id] ?? q.text,
        status: overrides[q.id] ?? q.status,
      })),
    [overrides, edits, generated],
  );

  const rows = items.filter(
    (q) =>
      (states.length === 0 || states.includes(q.status)) &&
      (topic === "" || q.topic === topic) &&
      (difficulty === "" || q.difficulty === difficulty) &&
      (kind === "" || q.kind === kind) &&
      (source === "" || q.source === source),
  );
  const open = items.find((q) => q.id === openId) ?? null;
  const pending = items.filter((q) => q.status === "pending").length;
  const filtered = states.length > 0 || topic || difficulty || kind || source;

  useEffect(() => {
    if (!generating) return;
    const t = window.setInterval(() => setProgress((p) => Math.min(100, p + 9)), 120);
    return () => window.clearInterval(t);
  }, [generating]);

  function generate() {
    setProgress(0);
    setGenerating(true);
    window.setTimeout(() => {
      setGenerating(false);
      setGenOpen(false);
      const ids = GENERATED_QUESTIONS.map((q) => q.id);
      setGenerated(ids);
      undo.push(`Đã thêm ${ids.length} câu hỏi nháp về ${genTopic} · đang chờ bạn duyệt`, () => setGenerated([]));
    }, 1500);
  }

  function setStatus(q: Question, next: QStatus, label: string) {
    const before = overrides[q.id] ?? q.status;
    setOverrides((prev) => ({ ...prev, [q.id]: next }));
    undo.push(label, () => setOverrides((prev) => ({ ...prev, [q.id]: before })));
  }

  const columns: Column<Question>[] = [
    { key: "text", header: "Câu hỏi", render: (q) => <span className={s.text}>{q.text}</span> },
    { key: "topic", header: "Chủ đề", width: "200px", hideOnMobile: true, render: (q) => <span className={s.meta}>{q.topic}</span> },
    { key: "difficulty", header: "Độ khó", width: "110px", hideOnMobile: true, render: (q) => <span className={s.meta}>{DIFFICULTY_LABEL[q.difficulty]}</span> },
    { key: "kind", header: "Loại", width: "130px", hideOnMobile: true, render: (q) => <span className={s.meta}>{KIND_LABEL[q.kind]}</span> },
    { key: "source", header: "Nguồn", width: "180px", hideOnMobile: true, render: (q) => <span className={s.meta}>{SOURCE_LABEL[q.source]}</span> },
    { key: "status", header: "Trạng thái", width: "130px", render: (q) => <StatusText tone={STATUS_TONE[q.status]}>{STATUS_LABEL[q.status]}</StatusText> },
  ];

  return (
    <Page width="wide">
      <PageHeader
        title="Ngân hàng câu hỏi"
        description="Câu hỏi dùng cho luyện đề và đề kiểm tra của học phần An ninh mạng."
        meta={
          <>
            <span>{items.length} câu</span>
            <span>{pending} câu chờ duyệt</span>
            <span>AI chỉ soạn nháp — câu hỏi vào đề khi giảng viên duyệt</span>
          </>
        }
        actions={
          <Button variant="primary" icon={<Sparkles aria-hidden />} onClick={() => setGenOpen(true)} disabled={genOpen}>
            Tạo câu hỏi
          </Button>
        }
      />

      <PageState
        loading={<Skeleton lines={12} />}
        empty={
          <EmptyState title="Ngân hàng câu hỏi còn trống" icon={<CircleHelp aria-hidden />} action={<Button variant="primary" onClick={() => setGenOpen(true)}>Tạo câu hỏi</Button>}>
            Chưa có câu hỏi nào cho học phần này. Tạo câu hỏi từ bài giảng đã tải lên, rồi duyệt những câu bạn muốn dùng.
          </EmptyState>
        }
        error={{ problem: "Không tải được ngân hàng câu hỏi.", recovery: "Các câu đã duyệt vẫn được giữ. Thử lại sau ít phút." }}
      >
        {genOpen && (
          <div className={s.gen}>
            <div className={s.genRow}>
              <Field className={s.genField} label="Tạo câu hỏi từ bài giảng của chủ đề" helper="Câu hỏi sinh ra luôn ở trạng thái chờ duyệt.">
                {(id) => (
                  <Select id={id} value={genTopic} disabled={generating} onChange={(e) => setGenTopic(e.target.value as (typeof TOPICS)[number])}>
                    {TOPICS.map((t) => (
                      <option key={t} value={t}>
                        {t}
                      </option>
                    ))}
                  </Select>
                )}
              </Field>
              <Button
                variant="primary"
                loading={generating}
                onClick={generate}
              >
                Tạo 5 câu nháp
              </Button>
              <Button variant="ghost" onClick={() => setGenOpen(false)} disabled={generating}>
                Để sau
              </Button>
            </div>
            {generating && (
              <>
                <span className={s.bar}>
                  <span className={s.barFill} style={{ width: `${progress}%` }} />
                </span>
                <p className={s.genText}>Đang đọc bài giảng và soạn câu hỏi nháp…</p>
              </>
            )}
          </div>
        )}

        <Toolbar
          end={
            <Popover
              label="Bộ lọc câu hỏi"
              width={280}
              trigger={(p) => (
                <Button icon={<SlidersHorizontal aria-hidden />} onClick={p.toggle} aria-expanded={p["aria-expanded"]} aria-haspopup="true">
                  Bộ lọc
                </Button>
              )}
            >
              <div className={s.filterPanel}>
                <Field label="Chủ đề">
                  {(id) => (
                    <Select id={id} value={topic} onChange={(e) => setTopic(e.target.value)}>
                      <option value="">Tất cả chủ đề</option>
                      {TOPICS.map((t) => (
                        <option key={t} value={t}>
                          {t}
                        </option>
                      ))}
                    </Select>
                  )}
                </Field>
                <Field label="Độ khó">
                  {(id) => (
                    <Select id={id} value={difficulty} onChange={(e) => setDifficulty(e.target.value)}>
                      <option value="">Mọi độ khó</option>
                      {(Object.keys(DIFFICULTY_LABEL) as QDifficulty[]).map((d) => (
                        <option key={d} value={d}>
                          {DIFFICULTY_LABEL[d]}
                        </option>
                      ))}
                    </Select>
                  )}
                </Field>
                <Field label="Loại câu hỏi">
                  {(id) => (
                    <Select id={id} value={kind} onChange={(e) => setKind(e.target.value)}>
                      <option value="">Mọi loại</option>
                      {(Object.keys(KIND_LABEL) as QKind[]).map((k) => (
                        <option key={k} value={k}>
                          {KIND_LABEL[k]}
                        </option>
                      ))}
                    </Select>
                  )}
                </Field>
                <Field label="Nguồn">
                  {(id) => (
                    <Select id={id} value={source} onChange={(e) => setSource(e.target.value)}>
                      <option value="">Mọi nguồn</option>
                      {(Object.keys(SOURCE_LABEL) as QSource[]).map((k) => (
                        <option key={k} value={k}>
                          {SOURCE_LABEL[k]}
                        </option>
                      ))}
                    </Select>
                  )}
                </Field>
              </div>
            </Popover>
          }
        >
          <FilterChips
            label="Lọc theo trạng thái"
            value={states}
            onChange={setStates}
            options={[
              { value: "pending", label: "Chờ duyệt", count: items.filter((q) => q.status === "pending").length },
              { value: "approved", label: "Đã duyệt", count: items.filter((q) => q.status === "approved").length },
              { value: "rejected", label: "Đã loại", count: items.filter((q) => q.status === "rejected").length },
            ]}
          />
        </Toolbar>

        {undo.node}

        <DataTable
          caption="Ngân hàng câu hỏi"
          columns={columns}
          rows={rows}
          rowKey={(q) => q.id}
          onRowClick={(q) => {
            setOpenId(q.id);
            setEditing(false);
          }}
          activeKey={openId ?? undefined}
          empty={
            <EmptyState
              title="Không có câu hỏi nào khớp bộ lọc"
              action={
                <Button
                  onClick={() => {
                    setStates([]);
                    setTopic("");
                    setDifficulty("");
                    setKind("");
                    setSource("");
                  }}
                >
                  Bỏ bộ lọc
                </Button>
              }
            >
              Bộ lọc hiện tại không còn câu hỏi nào. Bỏ bớt điều kiện để xem lại toàn bộ {items.length} câu.
            </EmptyState>
          }
        />
        {filtered && <p className={s.meta}>Đang xem {rows.length} trong {items.length} câu.</p>}
      </PageState>

      <Drawer
        open={Boolean(open)}
        onClose={() => setOpenId(null)}
        wide
        title={open ? `${KIND_LABEL[open.kind]} · ${open.topic}` : ""}
        description={open ? `${DIFFICULTY_LABEL[open.difficulty]} · ${SOURCE_LABEL[open.source]} · ${STATUS_LABEL[open.status]}` : undefined}
        footer={
          open && (
            <div className={s.detailActions}>
              <OverflowMenu
                label="Thêm hành động với câu hỏi"
                items={[
                  {
                    label: "Loại khỏi ngân hàng",
                    icon: <X aria-hidden />,
                    danger: true,
                    onSelect: () => {
                      setStatus(open, "rejected", "Đã loại câu hỏi khỏi ngân hàng");
                      setOpenId(null);
                    },
                  },
                ]}
              />
              <Button
                icon={<Pencil aria-hidden />}
                onClick={() => {
                  setDraft(open.text);
                  setEditing(true);
                }}
                disabled={editing}
              >
                Chỉnh sửa
              </Button>
              {editing ? (
                <Button
                  variant="primary"
                  onClick={() => {
                    setEdits((prev) => ({ ...prev, [open.id]: draft }));
                    setEditing(false);
                    undo.push("Đã sửa nội dung câu hỏi", () => setEdits((prev) => ({ ...prev, [open.id]: open.text })));
                  }}
                >
                  Lưu câu hỏi
                </Button>
              ) : (
                <Button
                  variant="primary"
                  icon={<Check aria-hidden />}
                  disabled={open.status === "approved"}
                  onClick={() => {
                    setStatus(open, "approved", "Đã duyệt câu hỏi vào ngân hàng");
                    setOpenId(null);
                  }}
                >
                  {open.status === "approved" ? "Đã duyệt" : "Duyệt"}
                </Button>
              )}
            </div>
          )
        }
      >
        {open && (
          <div className={s.detail}>
            {open.status === "pending" && (
              <InlineNotice tone="info" compact>
                Câu hỏi này do AI soạn nháp từ bài giảng. Đọc lại nội dung và đáp án trước khi duyệt.
              </InlineNotice>
            )}
            {editing ? (
              <Field label="Nội dung câu hỏi">{(id) => <Textarea id={id} rows={3} value={draft} onChange={(e) => setDraft(e.target.value)} />}</Field>
            ) : (
              <p className={s.question}>{open.text}</p>
            )}
            {open.options && (
              <ul className={s.options}>
                {open.options.map((o) => (
                  <li key={o} className={[s.option, open.answer.startsWith(o) ? s.optionRight : ""].join(" ")}>
                    {open.answer.startsWith(o) && <span className={s.optionMark}>Đáp án</span>}
                    <span>{o}</span>
                  </li>
                ))}
              </ul>
            )}
            <p className={s.answer}>{open.options ? open.answer : `Gợi ý chấm: ${open.answer}`}</p>
          </div>
        )}
      </Drawer>
    </Page>
  );
}
