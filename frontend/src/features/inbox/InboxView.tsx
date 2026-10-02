"use client";

import { ArrowLeft } from "lucide-react";
import { usePathname, useRouter, useSearchParams } from "next/navigation";
import { useEffect, useLayoutEffect, useRef, useState } from "react";
import { STUDENTS, courseById, fmtTime, NOW } from "@/mock/core";
import { agoLabel, ticketAgeMin } from "@/mock/derive";
import { KEYS, type Ticket, type TicketStatus } from "@/mock/state";
import { noteTicketAnswered } from "@/mock/notes";
import { mergeTickets } from "@/mock/support";
import { waitText, OVERDUE_MIN } from "@/mock/staff";
import { useSimNow } from "@/shared/state/clock";
import { useSession } from "@/shared/session/session";
import { useDemoSlice } from "@/shared/state/demo";
import {
  ActionList,
  ActionRow,
  Button,
  Checkbox,
  Composer,
  EmptyState,
  InlineNotice,
  Page,
  PageHeader,
  PageState,
  SegmentedControl,
  Skeleton,
  StatusText,
  useRouteState,
} from "@/shared/ui";
import s from "./InboxView.module.css";

type Filter = TicketStatus | "all";

const FILTERS: Array<{ value: Filter; label: string }> = [
  { value: "open", label: "Đang chờ" },
  { value: "claimed", label: "Đã nhận" },
  { value: "answered", label: "Đã trả lời" },
  { value: "all", label: "Tất cả" },
];

const STATUS: Record<TicketStatus, { label: string; tone: "red" | "amber" | "green" | "neutral" }> = {
  open: { label: "Đang chờ", tone: "red" },
  claimed: { label: "Đã nhận", tone: "amber" },
  answered: { label: "Đã trả lời", tone: "green" },
  closed: { label: "Đã đóng", tone: "neutral" },
};

/** Hộp thư hỗ trợ: danh sách 380px + chi tiết, ở điện thoại là danh sách → chi tiết (DESIGN §14.5). */
export function InboxView() {
  const { user, course, courses, isAll } = useSession();
  const [stored, setStored] = useDemoSlice<Ticket[]>(KEYS.tickets, []);
  const [filter, setFilter] = useState<Filter>("open");
  const router = useRouter();
  const pathname = usePathname();
  const params = useSearchParams();
  const picked = params.get("ticket");
  // dưới 1100px: danh sách → chi tiết là hai bước; nhớ vị trí cuộn để quay lại đúng chỗ (02-AC9)
  const listRef = useRef<HTMLElement>(null);
  const saved = useRef({ page: 0, list: 0 });
  const [draft, setDraft] = useState("");
  const [knowledge, setKnowledge] = useState(false);
  const nowMs = useSimNow();
  const routeState = useRouteState();

  const courseIds = isAll ? courses.map((c) => c.id) : [course.id];
  const tickets = mergeTickets(stored).filter((t) => courseIds.includes(t.courseId));
  const rows = tickets.filter((t) => filter === "all" || t.status === filter);
  const selected = tickets.find((t) => t.id === picked) ?? rows[0] ?? tickets[0] ?? null;

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
      // 02-AC13: phiếu được chọn sẵn và hàng của nó nằm trong khung nhìn của danh sách
      listRef.current?.querySelector<HTMLElement>(`[data-ticket-id="${CSS.escape(picked)}"]`)?.scrollIntoView({ block: "nearest" });
      return;
    }
    window.scrollTo({ top: saved.current.page });
    if (listRef.current) listRef.current.scrollTop = saved.current.list;
  }, [picked]);
  const prevFilter = useRef(filter);
  useEffect(() => {
    // chọn bộ lọc khác thì danh sách về đầu (lần dựng đầu không đụng: phiếu `?ticket=` đã được cuộn vào khung nhìn)
    if (prevFilter.current === filter) return;
    prevFilter.current = filter;
    if (listRef.current) listRef.current.scrollTop = 0;
  }, [filter]);

  function update(id: string, patch: Partial<Ticket>) {
    setStored(mergeTickets(stored).map((t) => (t.id === id ? { ...t, ...patch } : t)));
  }

  function send(t: Ticket) {
    update(t.id, { status: "answered", answer: { by: user.name, text: draft.trim(), saveAsKnowledge: knowledge } });
    noteTicketAnswered(t.studentId, t.id);
    setDraft("");
    setKnowledge(false);
  }

  const student = selected ? STUDENTS.find((x) => x.id === selected.studentId) : undefined;
  const mine = selected?.claimedBy === user.name;

  return (
    <Page width="full" className={s.page}>
      <PageHeader
        title="Hộp thư hỗ trợ"
        description="Câu hỏi được chuyển cho người thật khi AI không đủ chắc chắn hoặc sinh viên yêu cầu."
        meta={<span>{course.label}{isAll ? " và lớp còn lại" : ""}</span>}
      />
      <PageState
        state={routeState ?? (tickets.length === 0 ? "empty" : null)}
        loading={<Skeleton lines={10} />}
        empty={
          <EmptyState title="Không còn câu hỏi đang chờ">
            Khi AI không đủ chắc chắn hoặc sinh viên bấm “Nhờ giảng viên hỗ trợ”, câu hỏi sẽ xuất hiện ở đây.
          </EmptyState>
        }
        error={{
          problem: "Không tải được hộp thư hỗ trợ.",
          recovery: "Câu trả lời bạn đang soạn vẫn được giữ. Thử lại, hoặc mở lại sau ít phút.",
        }}
      >
        <div className={[s.split, picked ? s.showDetail : ""].join(" ")}>
          <section className={s.list} data-part="inbox-list" aria-label="Danh sách câu hỏi" ref={listRef}>
            <SegmentedControl
              label="Lọc câu hỏi"
              value={filter}
              onChange={(next) => {
                setFilter(next);
                if (picked) backToList();
              }}
              options={FILTERS.map((f) => ({ ...f, count: f.value === "all" ? tickets.length : tickets.filter((t) => t.status === f.value).length }))}
            />
            {rows.length === 0 ? (
              <EmptyState title="Không còn câu hỏi đang chờ">Chuyển bộ lọc sang “Tất cả” để xem câu đã trả lời.</EmptyState>
            ) : (
              <ActionList label="Câu hỏi cần xử lý">
                {rows.map((t) => {
                  const who = STUDENTS.find((x) => x.id === t.studentId);
                  const age = ticketAgeMin(t, nowMs);
                  const overdue = age >= 1440 && t.status === "open";
                  return (
                    <ActionRow
                      key={t.id}
                      data={{ "data-ticket-id": t.id }}
                      tone={age >= OVERDUE_MIN && t.status === "open" ? "red" : STATUS[t.status].tone}
                      selected={Boolean(!picked ? t.id === selected?.id : t.id === picked)}
                      onSelect={() => open(t.id)}
                      title={
                        <span className={s.rowHead}>
                          <span className={s.rowName}>{who?.name ?? "Sinh viên"}</span>
                          <span className={s.rowTime}>{agoLabel(nowMs - age * 60000, nowMs)}</span>
                        </span>
                      }
                      context={<span className={s.clamp}>{t.question}</span>}
                      meta={
                        <span className={s.rowMeta}>
                          {STATUS[t.status].label}
                          {overdue ? " · Quá 24 giờ" : ""} · {t.reason}
                        </span>
                      }
                    />
                  );
                })}
              </ActionList>
            )}
          </section>

          <section className={s.detail} data-part="inbox-detail" aria-label="Chi tiết câu hỏi">
            {selected && student ? (
              <>
                <Button className={s.back} variant="text" size="sm" icon={<ArrowLeft aria-hidden />} onClick={backToList}>
                  Hộp thư
                </Button>
                <header className={s.detailHead}>
                  <h2 className="ep-section-title">{student.name}</h2>
                  <p className="ep-meta">
                    {student.code} · {courseById(selected.courseId).label}
                  </p>
                </header>

                <div className={s.info}>
                  <p className={s.question}>{selected.question}</p>
                  <dl className={s.facts}>
                    <div>
                      <dt>Đã chờ</dt>
                      <dd>{waitText(ticketAgeMin(selected, nowMs))}</dd>
                    </div>
                    <div>
                      <dt>Lý do chuyển</dt>
                      <dd>{selected.reason}</dd>
                    </div>
                    <div>
                      <dt>Trạng thái</dt>
                      <dd>
                        <StatusText tone={STATUS[selected.status].tone}>{STATUS[selected.status].label}</StatusText>
                      </dd>
                    </div>
                    <div>
                      <dt>AI đã tra</dt>
                      <dd>Quy chế môn học (tr. 2), Đề cương học phần — không đoạn nào trả lời đủ</dd>
                    </div>
                  </dl>
                </div>

                <div className={s.reply} data-part="inbox-reply">
                  <h3 className="ep-section-title">Trả lời của bạn</h3>
                  {selected.status === "answered" && selected.answer ? (
                    <>
                      <p className={s.answer}>{selected.answer.text}</p>
                      <p className="ep-meta">
                        {selected.answer.by} · {fmtTime(NOW)}
                      </p>
                      <InlineNotice tone="success" compact>
                        Đã gửi thư thông báo tới email của sinh viên (mô phỏng)
                        {selected.answer.saveAsKnowledge ? " · đã lưu thành tri thức cho lớp" : ""}
                      </InlineNotice>
                    </>
                  ) : (
                    <>
                      {selected.status === "claimed" && !mine && (
                        <InlineNotice tone="warning" compact>
                          {selected.claimedBy} đã nhận lúc {selected.claimedAt}
                        </InlineNotice>
                      )}
                      {selected.status === "open" && (
                        <div className={s.claimRow}>
                          <Button variant="primary" onClick={() => update(selected.id, { status: "claimed", claimedBy: user.name, claimedAt: fmtTime(NOW) })}>
                            Nhận
                          </Button>
                          <span className="ep-meta">Nhận để người khác biết bạn đang xử lý câu này.</span>
                        </div>
                      )}
                      {selected.status === "claimed" && mine && (
                        <p className="ep-meta">Bạn đã nhận lúc {selected.claimedAt}. Sinh viên thấy trạng thái “Đang chờ giảng viên”.</p>
                      )}
                      <Composer
                        label="Câu trả lời gửi sinh viên"
                        value={draft}
                        onChange={setDraft}
                        onSubmit={() => send(selected)}
                        submitLabel="Gửi trả lời"
                        disabled={!mine}
                        placeholder={mine ? "Trả lời ngắn gọn, đúng việc sinh viên hỏi…" : "Nhận câu hỏi trước khi trả lời"}
                        tools={<Checkbox label="Lưu thành tri thức của lớp" checked={knowledge} onChange={(e) => setKnowledge(e.target.checked)} disabled={!mine} />}
                      />
                    </>
                  )}
                </div>
              </>
            ) : (
              <EmptyState title="Chọn một câu hỏi để xem">Danh sách bên trái xếp theo thời gian chờ.</EmptyState>
            )}
          </section>
        </div>
      </PageState>
    </Page>
  );
}
