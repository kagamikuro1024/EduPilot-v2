"use client";

import { useQueryClient } from "@tanstack/react-query";
import { ArrowLeft } from "lucide-react";
import { usePathname, useRouter, useSearchParams } from "next/navigation";
import { useEffect, useLayoutEffect, useRef, useState } from "react";
import { ApiError, ApiErrorNotice, apiClient, useAutosaveDraft, useIdempotentMutation, useSSE } from "@/shared/data";
import { timeAgo } from "@/shared/lib/timeAgo";
import { useSession } from "@/shared/session/session";
import { ActionList, ActionRow, Button, Composer, EmptyState, InlineNotice, Page, PageHeader, PageState, Panel, SegmentedControl, Skeleton, StatusText } from "@/shared/ui";
import { ticketsKey, useTicketDetail, useTicketList, waitLabel, type InboxFilter, type TicketDetail, type TicketRow, type TicketStatus } from "./inboxApi";
import s from "./RealInbox.module.css";

const FILTERS: Array<{ value: InboxFilter; label: string }> = [
  { value: "open", label: "Đang chờ" },
  { value: "claimed", label: "Đã nhận" },
  { value: "answered", label: "Đã trả lời" },
  { value: "all", label: "Tất cả" },
];

const STATUS: Record<TicketStatus, { label: string; tone: "red" | "amber" | "green" | "neutral" }> = {
  OPEN: { label: "Đang chờ", tone: "red" },
  CLAIMED: { label: "Đã nhận", tone: "amber" },
  ANSWERED: { label: "Đã trả lời", tone: "green" },
  CLOSED: { label: "Đã đóng", tone: "neutral" },
};

const DISMISS: Array<{ value: string; label: string }> = [
  { value: "OUT_OF_SCOPE", label: "Ngoài phạm vi" },
  { value: "DUPLICATE", label: "Trùng câu đã trả lời" },
  { value: "NEED_MORE_INFO", label: "Cần thêm thông tin" },
];

/** Hộp thư hỗ trợ thật (SRS FEAT-escalation-notify 7.3): danh sách 380 px | chi tiết; ≤ 1100 px là danh sách → chi tiết qua `?ticket=`. */
export function RealInbox({ courseId }: { courseId: string }) {
  const { identity, realCourses } = useSession();
  const [filter, setFilter] = useState<InboxFilter>("open");
  const router = useRouter();
  const pathname = usePathname();
  const params = useSearchParams();
  const picked = params.get("ticket");
  const qc = useQueryClient();
  const list = useTicketList(courseId, filter);
  const listRef = useRef<HTMLElement>(null);
  const saved = useRef({ page: 0, list: 0 });

  // chuông báo có câu hỏi mới / người khác nhận: làm tươi danh sách (không chờ vòng thăm dò)
  useSSE("notification", () => void qc.invalidateQueries({ queryKey: ticketsKey }), { invalidateKeys: [ticketsKey] });

  const rows = list.items;
  const selected = rows.find((t) => t.id === picked) ?? (picked ? undefined : rows[0]);
  const detail = useTicketDetail(selected?.course.id, selected?.id);
  const courseLabel = courseId === "all" ? "Tất cả lớp của tôi" : (realCourses?.find((c) => c.id === courseId)?.class_code ?? "");

  function open(id: string) {
    saved.current = { page: window.scrollY, list: listRef.current?.scrollTop ?? 0 };
    router.push(`${pathname}?ticket=${encodeURIComponent(id)}`, { scroll: false });
  }
  function backToList() {
    router.push(pathname, { scroll: false });
  }
  useLayoutEffect(() => {
    if (picked) {
      window.scrollTo({ top: 0 });
      return;
    }
    window.scrollTo({ top: saved.current.page });
    if (listRef.current) listRef.current.scrollTop = saved.current.list;
  }, [picked]);

  // J / K chọn dòng kế / trước (không khi đang gõ)
  useEffect(() => {
    function onKey(e: KeyboardEvent) {
      const t = e.target as HTMLElement | null;
      if (t && (t.tagName === "TEXTAREA" || t.tagName === "INPUT" || t.isContentEditable)) return;
      if (e.metaKey || e.ctrlKey || e.altKey) return;
      if (e.key === "r") {
        const box = document.querySelector<HTMLTextAreaElement>("[data-part=inbox-reply] textarea");
        if (box) {
          e.preventDefault();
          box.focus();
        }
        return;
      }
      const i = rows.findIndex((r) => r.id === selected?.id);
      if (e.key === "j" && i >= 0 && i < rows.length - 1) open(rows[i + 1].id);
      if (e.key === "k" && i > 0) open(rows[i - 1].id);
    }
    window.addEventListener("keydown", onKey);
    return () => window.removeEventListener("keydown", onKey);
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [rows, selected?.id]);

  return (
    <Page width="full" className={s.page}>
      <PageHeader title="Hộp thư hỗ trợ" meta={<span>{courseLabel}</span>} />
      <PageState
        query={list}
        isEmpty={() => false}
        loading={<Panel><Skeleton lines={10} /></Panel>}
      >
        <div className={s.work}>
          <Panel padding="none">
            <div className={[s.split, picked ? s.showDetail : ""].join(" ")}>
              <section className={s.list} data-part="inbox-list" aria-label="Danh sách câu hỏi" ref={listRef}>
                <SegmentedControl
                  label="Lọc câu hỏi"
                  value={filter}
                  onChange={(next) => {
                    setFilter(next);
                    if (picked) backToList();
                  }}
                  options={FILTERS}
                />
                {rows.length === 0 ? (
                  <EmptyState title="Không có câu hỏi nào đang chờ.">
                    {filter !== "all" && <Button variant="text" onClick={() => setFilter("all")}>Xem tất cả</Button>}
                  </EmptyState>
                ) : (
                  <ActionList label="Câu hỏi cần xử lý">
                    {rows.map((t) => (
                      <Row key={t.id} t={t} showCourse={courseId === "all"} selected={t.id === selected?.id} onSelect={() => open(t.id)} />
                    ))}
                  </ActionList>
                )}
                {list.hasNextPage && (
                  <Button onClick={() => void list.fetchNextPage()} disabled={list.isFetchingNextPage}>Xem thêm</Button>
                )}
              </section>

              <section className={s.detail} data-part="inbox-detail" aria-label="Chi tiết câu hỏi">
                {selected ? (
                  <>
                    <Button className={s.back} variant="text" size="sm" icon={<ArrowLeft aria-hidden />} onClick={backToList}>Hộp thư</Button>
                    {detail.isPending ? <Skeleton lines={8} /> : detail.isError ? (
                      <ApiErrorNotice error={detail.error} onRetry={() => void detail.refetch()} showTechnical />
                    ) : (
                      <TicketPanel key={detail.data.ticket.id} d={detail.data} userId={identity?.sub} me={identity?.sub} onNext={() => { const i = rows.findIndex((r) => r.id === selected.id); if (i >= 0 && i < rows.length - 1) open(rows[i + 1].id); }} />
                    )}
                  </>
                ) : (
                  <EmptyState title="Chọn một câu hỏi để xem" />
                )}
              </section>
            </div>
          </Panel>
        </div>
      </PageState>
    </Page>
  );
}

function Row({ t, showCourse, selected, onSelect }: { t: TicketRow; showCourse: boolean; selected: boolean; onSelect: () => void }) {
  return (
    <ActionRow
      data={{ "data-ticket-id": t.id }}
      tone={t.sla_stage >= 2 && t.status === "OPEN" ? "red" : STATUS[t.status].tone}
      selected={selected}
      onSelect={onSelect}
      title={
        <span className={s.rowHead}>
          <span className={s.rowName}>{t.student.full_name}</span>
          <span className={s.rowTime}>{waitLabel(t.age_minutes)}</span>
        </span>
      }
      context={<span className={s.clamp}>{t.question_preview}</span>}
      meta={
        <span className={s.rowMeta}>
          {STATUS[t.status].label}
          {t.status === "CLAIMED" && t.claimed_by ? ` · ${t.claimed_by.full_name}` : ""}
          {showCourse ? ` · ${t.course.class_code}` : ""}
        </span>
      }
    />
  );
}

function TicketPanel({ d, userId, me, onNext }: { d: TicketDetail; userId?: string; me?: string; onNext: () => void }) {
  const t = d.ticket;
  const qc = useQueryClient();
  const base = `/courses/${t.course.id}/tickets/${t.id}`;
  const draft = useAutosaveDraft(`ticket:${t.id}`, { userId });
  const [err, setErr] = useState<unknown>(null);
  const [busy, setBusy] = useState(false);
  const [closing, setClosing] = useState(false);
  const mine = t.claimed_by?.id === me;
  const refresh = () => qc.invalidateQueries({ queryKey: ticketsKey });

  async function act(path: string, body?: object) {
    if (busy) return;
    setBusy(true);
    setErr(null);
    try {
      await apiClient.post(`${base}/${path}`, { ...body, version: t.version });
      setClosing(false);
    } catch (e) {
      setErr(e);
    } finally {
      setBusy(false);
      void refresh(); // 409 (người khác đã nhận) cũng cần dữ liệu mới
    }
  }

  const answer = useIdempotentMutation((text: string, key: string) => apiClient.post(`${base}/answer`, { body: text, version: t.version }, { idempotencyKey: key }));
  async function send(retry = false): Promise<boolean> {
    setErr(null);
    try {
      await (retry ? answer.retry() : answer.mutate(draft.value.trim())); // gửi lại dùng LẠI khoá Idempotency: không tạo tin thứ hai
      draft.clear();
      void refresh();
      return true;
    } catch (e) {
      setErr(e);
      if (e instanceof ApiError && e.code !== "PII_IN_ANSWER" && e.status !== 0) void refresh();
      return false;
    }
  }
  const sendErr = err instanceof ApiError && (err.code === "PII_IN_ANSWER" || err.code === "NETWORK" || err.status >= 500) ? err : null;

  const finished = t.status === "ANSWERED" || t.status === "CLOSED";
  const claimedByOther = t.status === "CLAIMED" && !mine;
  const canReply = (t.status === "CLAIMED" && mine) || t.status === "ANSWERED";

  return (
    <>
      <header className={s.detailHead}>
        <h3 className="ep-section-title">{t.student.full_name}</h3>
        <p className="ep-meta">{t.course.class_code}</p>
      </header>

      <div className={s.info}>
        {d.question && <p className={s.question}>{d.question.content}</p>}
        {d.ai_answer && (
          <p className={s.answer}>
            {d.ai_answer.content}
            {d.ai_answer.confidence ? <span className="ep-meta"> · Độ tin cậy {Number(d.ai_answer.confidence).toFixed(2).replace(".", ",")}</span> : null}
          </p>
        )}
        <dl className={s.facts}>
          <div>
            <dt>Đã chờ</dt>
            <dd>{waitLabel(t.age_minutes)}</dd>
          </div>
          <div>
            <dt>Trạng thái</dt>
            <dd><StatusText tone={STATUS[t.status].tone}>{STATUS[t.status].label}</StatusText></dd>
          </div>
        </dl>
      </div>

      {d.transcript.length > 0 && (
        <div className={s.reply}>
          {d.transcript.map((m) => (
            <div key={m.id}>
              <p className={s.answer}>{m.content}</p>
              <p className="ep-meta">{m.author === "student" ? t.student.full_name : m.author.full_name} · {timeAgo(m.created_at)}</p>
            </div>
          ))}
        </div>
      )}

      {t.status !== "CLOSED" && (
        <div
          className={s.reply}
          data-part="inbox-reply"
          onKeyDownCapture={(e) => {
            // ⌘↵ / Ctrl+↵: gửi rồi sang ticket kế
            if (e.key === "Enter" && (e.metaKey || e.ctrlKey) && canReply && draft.value.trim() && !answer.pending) {
              e.preventDefault();
              void send().then((ok) => ok && onNext());
            }
          }}
        >
          {claimedByOther && <InlineNotice tone="warning" compact>{t.claimed_by?.full_name} đã nhận.</InlineNotice>}
          {t.status === "OPEN" && (
            <div className={s.claimRow}>
              <Button variant="primary" onClick={() => void act("claim")} disabled={busy}>Nhận</Button>
              <Button variant="text" onClick={() => setClosing(!closing)}>Đóng không trả lời</Button>
            </div>
          )}
          {canReply && (
            <Composer
              label="Câu trả lời gửi sinh viên"
              value={draft.value}
              onChange={draft.setValue}
              onSubmit={() => void send()}
              busy={answer.pending}
              submitLabel="Gửi trả lời"
              placeholder="Trả lời ngắn gọn, đúng việc sinh viên hỏi…"
              error={sendErr?.userMessage}
              onRetry={sendErr && sendErr.code !== "PII_IN_ANSWER" ? () => void send(true) : undefined}
            />
          )}
          {t.status === "CLAIMED" && mine && (
            <div className={s.claimRow}>
              <Button variant="text" onClick={() => void act("release")} disabled={busy}>Bỏ nhận</Button>
              <Button variant="text" onClick={() => setClosing(!closing)}>Đóng không trả lời</Button>
            </div>
          )}
          {closing && (
            <div className={s.claimRow} role="group" aria-label="Lý do đóng">
              {DISMISS.map((x) => (
                <Button key={x.value} variant="secondary" size="sm" onClick={() => void act("dismiss", { reason: x.value })} disabled={busy}>{x.label}</Button>
              ))}
            </div>
          )}
          {err != null && !sendErr && <ApiErrorNotice error={err} showTechnical />}
          {finished && t.answered_by && <p className="ep-meta">{t.answered_by.full_name} đã trả lời.</p>}
        </div>
      )}
    </>
  );
}
