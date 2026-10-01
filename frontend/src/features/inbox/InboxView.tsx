"use client";

import { ArrowLeft } from "lucide-react";
import { useState } from "react";
import { STUDENTS, at, ago, courseById, fmtTime, NOW } from "@/mock/core";
import { KEYS, type Ticket, type TicketStatus } from "@/mock/state";
import { mergeTickets } from "@/mock/support";
import { waitText, OVERDUE_MIN } from "@/mock/staff";
import { useSession } from "@/shared/session/session";
import { useDemoSlice } from "@/shared/state/demo";
import {
  ActionList,
  ActionRow,
  Button,
  Checkbox,
  Composer,
  DefinitionList,
  EmptyState,
  InlineNotice,
  Page,
  PageHeader,
  PageState,
  SegmentedControl,
  Section,
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
  const [picked, setPicked] = useState<string | null>(null);
  const [draft, setDraft] = useState("");
  const [knowledge, setKnowledge] = useState(false);
  const routeState = useRouteState();

  const courseIds = isAll ? courses.map((c) => c.id) : [course.id];
  const tickets = mergeTickets(stored).filter((t) => courseIds.includes(t.courseId));
  const rows = tickets.filter((t) => filter === "all" || t.status === filter);
  const selected = tickets.find((t) => t.id === picked) ?? rows[0] ?? tickets[0] ?? null;

  function update(id: string, patch: Partial<Ticket>) {
    setStored(mergeTickets(stored).map((t) => (t.id === id ? { ...t, ...patch } : t)));
  }

  function send(t: Ticket) {
    update(t.id, { status: "answered", answer: { by: user.name, text: draft.trim(), saveAsKnowledge: knowledge } });
    setDraft("");
    setKnowledge(false);
  }

  const student = selected ? STUDENTS.find((x) => x.id === selected.studentId) : undefined;
  const mine = selected?.claimedBy === user.name;

  return (
    <Page width="full">
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
        <div className={[s.split, selected && picked ? s.showDetail : ""].join(" ")}>
          <div className={s.list}>
            <SegmentedControl
              label="Lọc câu hỏi"
              value={filter}
              onChange={(next) => {
                setFilter(next);
                setPicked(null);
              }}
              options={FILTERS.map((f) => ({ ...f, count: f.value === "all" ? tickets.length : tickets.filter((t) => t.status === f.value).length }))}
            />
            {rows.length === 0 ? (
              <EmptyState title="Không còn câu hỏi đang chờ">Chuyển bộ lọc sang “Tất cả” để xem câu đã trả lời.</EmptyState>
            ) : (
              <ActionList label="Câu hỏi cần xử lý">
                {rows.map((t) => {
                  const who = STUDENTS.find((x) => x.id === t.studentId);
                  return (
                    <ActionRow
                      key={t.id}
                      tone={t.ageMin >= OVERDUE_MIN && t.status === "open" ? "red" : STATUS[t.status].tone}
                      selected={t.id === selected?.id}
                      onSelect={() => setPicked(t.id)}
                      title={who?.name ?? "Sinh viên"}
                      context={t.question}
                      meta={
                        <>
                          {ago(at(-t.ageMin))} · {STATUS[t.status].label} · {t.reason}
                        </>
                      }
                    />
                  );
                })}
              </ActionList>
            )}
          </div>

          <div className={s.detail}>
            {selected && student ? (
              <>
                <Button className={s.back} variant="text" size="sm" icon={<ArrowLeft aria-hidden />} onClick={() => setPicked(null)}>
                  Danh sách câu hỏi
                </Button>
                <Section title={student.name} description={`${student.code} · ${courseById(selected.courseId).label}`}>
                  <p className={s.question}>{selected.question}</p>
                  <DefinitionList
                    items={[
                      { term: "Đã chờ", value: waitText(selected.ageMin) },
                      { term: "Lý do chuyển", value: selected.reason },
                      { term: "Trạng thái", value: <StatusText tone={STATUS[selected.status].tone}>{STATUS[selected.status].label}</StatusText> },
                      { term: "AI đã tra", value: "Quy chế môn học (tr. 2), Đề cương học phần — không đoạn nào trả lời đủ" },
                    ]}
                  />
                </Section>

                <Section title="Trả lời của bạn">
                  {selected.status === "answered" && selected.answer ? (
                    <>
                      <p className={s.answer}>{selected.answer.text}</p>
                      <p className="ep-meta">{selected.answer.by} · {fmtTime(NOW)}</p>
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
                </Section>
              </>
            ) : (
              <EmptyState title="Chọn một câu hỏi để xem">Danh sách bên trái xếp theo thời gian chờ.</EmptyState>
            )}
          </div>
        </div>
      </PageState>
    </Page>
  );
}
