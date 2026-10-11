"use client";

import { useQueryClient } from "@tanstack/react-query";
import Link from "next/link";
import { useState } from "react";
import { useExamLock } from "@/features/chat/chatApi";
import { useAutosaveDraft } from "@/shared/data";
import { useSession } from "@/shared/session/session";
import { Button, EmptyState, Field, Input, InlineNotice, Page, PageHeader, PageState, Panel, Select, Skeleton, StatusText, Textarea } from "@/shared/ui";
import { ANSWER_LABEL, describeReasons, threadsKey, useThreadList, type ThreadRow, type ThreadView } from "./threadsApi";
import { usePostGate } from "./usePostGate";
import r from "./RealThreads.module.css";

const AGO = new Intl.DateTimeFormat("vi-VN", { day: "2-digit", month: "2-digit", hour: "2-digit", minute: "2-digit" });
const WEEKS = Array.from({ length: 20 }, (_, i) => String(i + 1));
const STATES = [
  { value: "", label: "Mọi trạng thái" },
  { value: "pending", label: "Chờ xác nhận" },
  { value: "verified", label: "Đã xác nhận" },
  { value: "none", label: "Chưa có câu trả lời" },
];

/** Threads thật (DESIGN §14.3): danh sách là chính, soạn tại chỗ sau `Đặt câu hỏi`. */
export function RealThreads({ courseId }: { courseId: string }) {
  const { role } = useSession();
  const [filter, setFilter] = useState({ q: "", tag: "", week: "", state: "" });
  const [showFilters, setShowFilters] = useState(false);
  const [composing, setComposing] = useState(false);
  const list = useThreadList(courseId, filter);
  const qc = useQueryClient();
  const filtered = filter.q !== "" || filter.tag !== "" || filter.week !== "" || filter.state !== "";

  return (
    <Page width="full">
      <PageHeader title="Threads" actions={!composing ? <Button variant="primary" onClick={() => setComposing(true)}>Đặt câu hỏi</Button> : undefined} />
      <div className={r.detail}>
        {composing && (
          <Panel>
            <Composer courseId={courseId} canChat={role === "student"} onCancel={() => setComposing(false)} onPosted={() => { setComposing(false); void qc.invalidateQueries({ queryKey: threadsKey(courseId) }); }} />
          </Panel>
        )}
        <Button variant="ghost" className={r.filterToggle} aria-expanded={showFilters} onClick={() => setShowFilters(!showFilters)}>Bộ lọc</Button>
        <div className={[r.filters, showFilters ? r.filtersOpen : ""].join(" ")} role="search">
          <Field label="Tìm theo tiêu đề">{(id) => <Input id={id} value={filter.q} onChange={(e) => setFilter({ ...filter, q: e.target.value })} placeholder="Ví dụ: cảnh báo học vụ" />}</Field>
          <Field label="Chủ đề">{(id) => <Input id={id} value={filter.tag} onChange={(e) => setFilter({ ...filter, tag: e.target.value.trim().toLowerCase() })} placeholder="quy-che" />}</Field>
          <Field label="Tuần">
            {(id) => (
              <Select id={id} value={filter.week} onChange={(e) => setFilter({ ...filter, week: e.target.value })}>
                <option value="">Mọi tuần</option>
                {WEEKS.map((w) => <option key={w} value={w}>Tuần {w}</option>)}
              </Select>
            )}
          </Field>
          <Field label="Trạng thái">
            {(id) => (
              <Select id={id} value={filter.state} onChange={(e) => setFilter({ ...filter, state: e.target.value })}>
                {STATES.map((s) => <option key={s.value} value={s.value}>{s.label}</option>)}
              </Select>
            )}
          </Field>
        </div>
        <Panel>
          <PageState
            query={list}
            isEmpty={() => list.items.length === 0}
            loading={<Skeleton lines={5} />}
            empty={
              <EmptyState title={filtered ? "Không có câu hỏi nào khớp." : "Chưa có câu hỏi nào. Đặt câu hỏi đầu tiên."}>
                {!filtered && !composing && <Button variant="primary" onClick={() => setComposing(true)}>Đặt câu hỏi</Button>}
              </EmptyState>
            }
          >
            <ul className={r.rows} data-part="thread-list">
              {list.items.map((t) => <li key={t.id}><Row t={t} /></li>)}
            </ul>
            {list.hasNextPage && <Button onClick={() => void list.fetchNextPage()} disabled={list.isFetchingNextPage}>Xem thêm</Button>}
          </PageState>
        </Panel>
      </div>
    </Page>
  );
}

function Row({ t }: { t: ThreadRow }) {
  const label = t.answer_state ? ANSWER_LABEL[t.answer_state] : null;
  return (
    <div className={r.row}>
      <Link href={`/threads/${t.id}`} className={r.rowLink}>
        <span className={r.rowTitle}>{t.title}</span>
        <p className={r.preview}>{t.preview}</p>
      </Link>
      <div className={r.meta}>
        <span>{t.author.full_name}</span>
        {t.week_no !== null && <span>Tuần {t.week_no}</span>}
        {t.tags.map((x) => <span key={x}>#{x}</span>)}
        {label ? <StatusText tone={label.tone}>{label.text}</StatusText> : t.ai_state === "SKIPPED" ? <StatusText tone="amber">Đang chờ giảng viên</StatusText> : null}
        <span>{AGO.format(new Date(t.last_activity_at))}</span>
      </div>
    </div>
  );
}

function Composer({ courseId, canChat, onCancel, onPosted }: { courseId: string; canChat: boolean; onCancel: () => void; onPosted: () => void }) {
  const { identity } = useSession();
  const title = useAutosaveDraft(`thread:${courseId}:title`, { userId: identity?.sub });
  const body = useAutosaveDraft(`thread:${courseId}:body`, { userId: identity?.sub });
  const [week, setWeek] = useState("");
  const [tags, setTags] = useState("");
  const lock = useExamLock();
  const locked = lock.data?.locked === true;
  const until = lock.data?.until ? new Date(lock.data.until) : null;
  const tagList = tags.split(",").map((x) => x.trim()).filter(Boolean);
  const gate = usePostGate<ThreadView>({
    courseId,
    path: `/courses/${courseId}/threads`,
    title: title.value,
    body: body.value,
    payload: { title: title.value, body: body.value, tags: tagList, week_no: week ? Number(week) : null },
    canChat,
    onPosted: () => {
      title.clear();
      body.clear();
      onPosted();
    },
    onSwitched: () => {
      title.clear();
      body.clear();
    },
  });
  const empty = !title.value.trim() || !body.value.trim();
  return (
    <form className={r.composer} onSubmit={(e) => { e.preventDefault(); if (!empty && !locked) void gate.submit(false); }} data-part="thread-form">
      <Field label="Tiêu đề" required>{(id) => <Input id={id} value={title.value} onChange={(e) => title.setValue(e.target.value)} maxLength={200} disabled={locked} />}</Field>
      <Field label="Nội dung" required>{(id) => <Textarea id={id} rows={5} value={body.value} onChange={(e) => body.setValue(e.target.value)} disabled={locked} />}</Field>
      <div className={r.composerRow}>
        <Field label="Thẻ">{(id) => <Input id={id} value={tags} onChange={(e) => setTags(e.target.value)} placeholder="quy-che, tcp" disabled={locked} />}</Field>
        <Field label="Tuần">
          {(id) => (
            <Select id={id} value={week} onChange={(e) => setWeek(e.target.value)} disabled={locked}>
              <option value="">Không chọn</option>
              {WEEKS.map((w) => <option key={w} value={w}>Tuần {w}</option>)}
            </Select>
          )}
        </Field>
      </div>
      {gate.check && !gate.check.allowed && (
        <p className={r.noticeLine} role="status" data-part="pii-notice">Phát hiện {describeReasons(gate.check.reasons)}</p>
      )}
      {locked && <p className={r.noticeLine} role="status">Chat tạm khóa trong lúc bạn làm bài thi.{until ? ` Dùng lại được sau ${until.toLocaleTimeString("vi-VN", { hour: "2-digit", minute: "2-digit" })}.` : ""}</p>}
      {gate.error && (
        <InlineNotice tone="danger" compact action={<Button size="sm" onClick={() => void gate.submit(false)}>Gửi lại</Button>}>{gate.error.userMessage}</InlineNotice>
      )}
      <div className={r.actions}>
        <Button type="submit" variant="primary" disabled={empty || locked || gate.pending}>Đăng</Button>
        <Button variant="ghost" onClick={onCancel}>Huỷ</Button>
      </div>
      {gate.dialog}
    </form>
  );
}

export { Composer as ThreadComposer };
