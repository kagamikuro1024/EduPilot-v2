"use client";

import { Check, ShieldCheck, ThumbsDown, ThumbsUp } from "lucide-react";
import { useEffect, useRef, useState } from "react";
import {
  CHAT_SESSIONS,
  D1_CITATIONS,
  LOW_CONFIDENCE_ANSWER,
  NEUTRAL_ANSWER,
  PII_EXPLAINER,
  QUIZ_ANSWER,
  REFUSAL_ANSWER,
  answerD1,
  matchScript,
  scanPersonal,
  sessionsFor,
  type Script,
} from "@/mock/chat";
import { COURSE_1, NOW, STAFF, STUDENT_B, fmtScore, fmtTime } from "@/mock/core";
import { qtOf } from "@/mock/grades";
import { QUIZ_SEED, quizKey, type QuizState } from "@/mock/practice";
import { BT03_SEED } from "@/mock/assess";
import { ATTENDANCE_SEED, committedOf, KEYS, type AttendanceState, type Bt03State, type Ticket } from "@/mock/state";
import { B_ABSENT_DATES } from "@/mock/student";
import { TICKETS_SEED, ticketD3 } from "@/mock/support";
import { noteNewTicket, noteTicketSent } from "@/mock/notes";
import { sentLabel } from "@/mock/derive";
import { useStreamedText } from "@/shared/lib/useStreamedText";
import { useSession } from "@/shared/session/session";
import { simNowMs, useSimNow } from "@/shared/state/clock";
import { useDemoSlice } from "@/shared/state/demo";
import { Button, Composer, DefinitionList, InlineNotice, Page, PageHeader, PageState, Panel, Skeleton, StatusText } from "@/shared/ui";
import { SessionSheet, sessionMeta } from "./SessionSheet";
import s from "./ChatScreen.module.css";

/** `stats`: số liệu của SV tại lúc câu trả lời được tạo — chip dưới tin nhắn khớp với chữ trong tin nhắn dù điểm danh đổi sau đó (DEMO-11). */
type Msg = { id: string; from: "sv" | "ai"; text: string; script?: Script; hidden?: number; stats?: { absences: number; speaks: number; bonus: number } };

const MSG_KEY = "chat.messages";
export const CHAT_DRAFT_KEY = "chat.draft";

/** Chat riêng của sinh viên (DESIGN §14.2): hỏi về chính mình, ẩn thông tin cá nhân trước khi gửi. */
export function ChatScreen() {
  const { user, course, studentId } = useSession();
  const now = useSimNow();
  /** Phiên chat thuộc về từng sinh viên (E2): B có 4 phiên seed, A / C / D chưa có phiên nào. */
  const sessions = sessionsFor(studentId ?? "");
  const [msgs, setMsgs] = useDemoSlice<Msg[]>(`${MSG_KEY}.${studentId ?? "none"}`, []);
  const [draft, setDraft] = useDemoSlice<string>(CHAT_DRAFT_KEY, "");
  const [tickets, setTickets] = useDemoSlice<Ticket[]>(KEYS.tickets, TICKETS_SEED);
  const [attendance] = useDemoSlice<AttendanceState>(KEYS.attendance, ATTENDANCE_SEED);
  const [bt03] = useDemoSlice<Bt03State>(KEYS.bt03, BT03_SEED);
  const [quiz] = useDemoSlice<QuizState>(quizKey(studentId), QUIZ_SEED);
  const [streaming, setStreaming] = useState<{ id: string; full: string } | null>(null);
  const [openSession, setOpenSession] = useState<string | null>(null);
  const [sheet, setSheet] = useState(false);
  const endRef = useRef<HTMLDivElement>(null);

  const stream = useStreamedText(streaming?.full ?? null, { cps: 55 });
  useEffect(() => {
    endRef.current?.scrollIntoView({ block: "end" });
  }, [msgs.length, stream.text]);

  const quizDoing = quiz.status === "doing";
  const personal = scanPersonal(draft);
  const past = CHAT_SESSIONS.find((h) => h.id === openSession);

  /** `Hỏi tiếp`: mở phiên mới, mang câu hỏi cũ sang ô soạn làm ngữ cảnh. */
  function askAgain(session: { title: string }) {
    setOpenSession(null);
    setSheet(false);
    setMsgs([]);
    setDraft(`Hỏi tiếp về “${session.title}”: `);
  }

  function send() {
    const text = draft.trim();
    if (!text) return;
    const script = matchScript(text);
    const stats = qtOf(STUDENT_B.id, attendance, bt03);
    const dates = [...B_ABSENT_DATES];
    const today = committedOf(attendance[course.id]?.[10]);
    if (today && today.marks[STUDENT_B.id] === "absent") dates.push("29/10");
    const answer = quizDoing
      ? QUIZ_ANSWER
      : script === "d1"
        ? answerD1({ absences: stats.absences, dates, speaks: stats.speaks, bonus: stats.bonus, penalty: stats.penalty })
        : script === "d2"
          ? REFUSAL_ANSWER
          : script === "d3"
            ? LOW_CONFIDENCE_ANSWER
            : NEUTRAL_ANSWER;
    const seq = msgs.length;
    const aiId = `ai-${seq}`;
    setMsgs((prev) => [
      ...prev,
      { id: `sv-${seq}`, from: "sv", text, hidden: personal.count },
      { id: aiId, from: "ai", text: answer, script: quizDoing ? "other" : script, stats: { absences: stats.absences, speaks: stats.speaks, bonus: stats.bonus } },
    ]);
    setDraft("");
    setOpenSession(null);
    setStreaming({ id: aiId, full: answer });
    if (script === "d3" && !quizDoing) {
      setTickets((prev) => (prev.some((t) => t.id === "tk-d3") ? prev : [ticketD3(simNowMs()), ...prev]));
      noteNewTicket(COURSE_1, "tk-d3");
      noteTicketSent(STUDENT_B.id, "tk-d3");
    }
  }

  return (
    <Page width="full" className={s.page}>
      <PageHeader
        title="Chat riêng"
        description={`Hỏi về điểm, chuyên cần và bài học của chính bạn — ${course.label}. Giảng viên không đọc được phiên chat này.`}
        actions={
          msgs.length > 0 || openSession ? (
            <Button
              onClick={() => {
                setMsgs([]);
                setOpenSession(null);
              }}
            >
              Phiên mới
            </Button>
          ) : undefined
        }
      />

      <div className={s.layout}>
        <nav className={s.history} data-part="chat-history" aria-label="Phiên trước">
          <Panel>
          <p className={s.historyLabel}>Phiên trước</p>
          {sessions.length === 0 ? (
            <p className={s.historyEmpty}>Chưa có phiên nào</p>
          ) : (
            <ul>
              {sessions.map((h) => (
                <li key={h.id}>
                  <button
                    type="button"
                    className={[s.historyItem, openSession === h.id ? s.historyOn : ""].join(" ")}
                    aria-pressed={openSession === h.id}
                    onClick={() => setOpenSession(openSession === h.id ? null : h.id)}
                  >
                    <span className={s.historyTitle}>{h.title}</span>
                    <span className={s.historyMeta}>{sessionMeta(h, now)}</span>
                  </button>
                </li>
              ))}
            </ul>
          )}
          </Panel>
        </nav>

        <div className={s.column}>
          <button
            type="button"
            className={s.sessionsBtn}
            data-part="chat-sessions-button"
            onClick={() => setSheet(true)}
            aria-haspopup="dialog"
          >
            Phiên trước ({sessions.length})
          </button>

          <Panel>
          <div className={s.inner}>
          <PageState
            loading={
              <div className={s.thread} data-part="chat-thread">
                <Skeleton lines={2} />
                <Skeleton lines={4} />
              </div>
            }
            empty={
              <div className={s.thread} data-part="chat-thread">
                <InlineNotice title="Phiên chat này chưa có tin nhắn">
                  Hỏi một câu về điểm hoặc chuyên cần của bạn, ví dụ “Em đã nghỉ mấy buổi rồi ạ?”.
                </InlineNotice>
              </div>
            }
          >
            <div className={s.thread} data-part="chat-thread">
              {past ? (
                <>
                  <p className={s.sessionNote}>
                    Phiên đã kết thúc · {sessionMeta(past, now)}. Bấm “Hỏi tiếp” để mở phiên mới mang ngữ cảnh phiên này.
                  </p>
                  <Bubble from="sv" text={past.q} />
                  <Bubble from="ai" text={past.a} />
                  <p className={s.sessionActions}>
                    <Button variant="primary" onClick={() => askAgain(past)}>
                      Hỏi tiếp
                    </Button>
                  </p>
                </>
              ) : msgs.length === 0 ? (
                <Intro name={user.name} />
              ) : (
                msgs.map((m) =>
                  m.from === "sv" ? (
                    <UserMessage key={m.id} msg={m} />
                  ) : (
                    <AiMessage
                      key={m.id}
                      msg={m}
                      shown={streaming?.id === m.id && !stream.done ? stream.text : m.text}
                      live={streaming?.id === m.id && !stream.done}
                      stats={m.stats ?? qtOf(STUDENT_B.id, attendance, bt03)}
                      ticket={tickets.find((t) => t.id === "tk-d3")}
                      onClose={() =>
                        setTickets((prev) => prev.map((t) => (t.id === "tk-d3" ? { ...t, closedBySv: true, status: "closed" } : t)))
                      }
                    />
                  ),
                )
              )}
              <div ref={endRef} />
            </div>
          </PageState>

          <div className={s.composer} data-part="chat-composer">
            {quizDoing && (
              <InlineNotice tone="warning" compact>
                Bạn đang làm QUIZ01. Trong lúc làm bài, Chat riêng chỉ trả lời câu hỏi thủ tục.
              </InlineNotice>
            )}
            <Composer
              value={draft}
              onChange={setDraft}
              onSubmit={send}
              busy={Boolean(streaming) && !stream.done}
              onStop={stream.stop}
              placeholder="Hỏi về điểm, chuyên cần, hạn nộp của bạn…"
              label="Câu hỏi của bạn"
              notice={
                personal.count > 0 ? (
                  <span className={s.shield}>
                    <ShieldCheck aria-hidden />
                    Câu hỏi có {personal.reasons.join(" và ")}. Những thông tin này được ẩn trước khi gửi cho AI.
                  </span>
                ) : undefined
              }
            />
          </div>
          </div>
          </Panel>
        </div>
      </div>

      <SessionSheet
        open={sheet}
        onClose={() => setSheet(false)}
        sessions={sessions}
        now={now}
        openSession={openSession}
        onPick={(id) => {
          setOpenSession(id);
          setSheet(false);
        }}
        onNew={() => {
          setOpenSession(null);
          setMsgs([]);
          setSheet(false);
        }}
      />
    </Page>
  );
}

function Intro({ name }: { name: string }) {
  return (
    <div className={s.intro}>
      <p className="ep-item-title">Chào {name.split(" ").slice(-1)[0]}, bạn muốn hỏi gì?</p>
      <p className={s.introText}>
        Phiên chat này chỉ bạn đọc được. Bạn có thể hỏi về số buổi đã vắng, điểm cộng phát biểu, hạn nộp bài hoặc nội dung bài
        giảng. Khi AI chưa đủ chắc chắn, câu hỏi sẽ được chuyển cho giảng viên.
      </p>
      <p className={s.introMeta}>Hôm nay {fmtTime(NOW)} · lớp đang học buổi 10</p>
    </div>
  );
}

function Bubble({ from, text }: { from: "sv" | "ai"; text: string }) {
  return (
    <div className={from === "sv" ? s.fromSv : s.fromAi}>
      <p className={s.who}>{from === "sv" ? "Bạn" : "Trợ lý AI"}</p>
      <p className={s.text}>{text}</p>
    </div>
  );
}

function UserMessage({ msg }: { msg: Msg }) {
  const [why, setWhy] = useState(false);
  return (
    <div className={s.fromSv}>
      <p className={s.who}>Bạn</p>
      <p className={s.text}>{msg.text}</p>
      {Boolean(msg.hidden) && (
        <>
          <p className={s.privacy}>
            <ShieldCheck aria-hidden />
            Đã ẩn {msg.hidden} thông tin cá nhân trước khi gửi cho AI
            <button type="button" className={s.linkBtn} onClick={() => setWhy((v) => !v)} aria-expanded={why}>
              Tìm hiểu
            </button>
          </p>
          {why && <p className={s.privacyWhy}>{PII_EXPLAINER}</p>}
        </>
      )}
    </div>
  );
}

function AiMessage({
  msg,
  shown,
  live,
  stats,
  ticket,
  onClose,
}: {
  msg: Msg;
  shown: string;
  live: boolean;
  stats: { absences: number; speaks: number; bonus: number };
  ticket?: Ticket;
  onClose: () => void;
}) {
  const [vote, setVote] = useState<"up" | "down" | null>(null);
  const [sources, setSources] = useState(true); // nguồn mở sẵn dưới câu trả lời (SRS §14.2)

  return (
    <div className={s.fromAi}>
      <p className={s.who}>Trợ lý AI</p>
      <p className={s.text} aria-live="polite">
        {shown}
        {live && <span className={s.caret} aria-hidden />}
      </p>

      {!live && msg.script === "d1" && (
        <>
          <div className={s.block}>
            <DefinitionList
              items={[
                { term: "Buổi vắng", value: `${stats.absences} buổi` },
                { term: "Phát biểu", value: `${stats.speaks} lần · +${fmtScore(stats.bonus, 2)}` },
              ]}
            />
          </div>
          <button type="button" className={s.sourceBtn} onClick={() => setSources((v) => !v)} aria-expanded={sources}>
            Nguồn tham khảo ({D1_CITATIONS.length})
          </button>
          {sources && (
            <ul className={s.sources}>
              {D1_CITATIONS.map((c) => (
                <li key={c.title}>
                  <span className={s.sourceTitle}>{c.title}</span>
                  <span className={s.sourceMeta}>{c.locator}</span>
                </li>
              ))}
            </ul>
          )}
        </>
      )}

      {!live && msg.script === "d3" ? (
        <TeacherHandoff ticket={ticket} onClose={onClose} />
      ) : (
        !live &&
        msg.script !== "d2" && (
          <div className={s.feedback}>
            <button type="button" className={s.linkBtn} aria-pressed={vote === "up"} onClick={() => setVote("up")}>
              <ThumbsUp aria-hidden /> Hữu ích
            </button>
            <button type="button" className={s.linkBtn} aria-pressed={vote === "down"} onClick={() => setVote("down")}>
              <ThumbsDown aria-hidden /> Không hữu ích
            </button>
            <AskTeacher />
            {vote && <span className={s.voted}>Đã ghi nhận, cảm ơn bạn.</span>}
          </div>
        )
      )}
    </div>
  );
}

/** D3: AI tự chuyển câu hỏi cho giảng viên — chỗ nút thay bằng trạng thái chờ. */
function TeacherHandoff({ ticket, onClose }: { ticket?: Ticket; onClose: () => void }) {
  const now = useSimNow();
  // Câu trả lời của giảng viên ở lại trong phiên kể cả sau `Đã rõ`: đóng câu hỏi chỉ đổi nút thành trạng thái (DEMO-4)
  if (ticket?.answer) {
    return (
      <div className={s.teacher}>
        <p className={s.teacherWho}>
          <Check aria-hidden />
          Giảng viên {STAFF.teacher.name} trả lời
        </p>
        <p className={s.text}>{ticket.answer.text}</p>
        {ticket.closedBySv ? (
          <StatusText tone="green">Câu hỏi đã đóng</StatusText>
        ) : (
          <Button size="sm" onClick={onClose}>
            Đã rõ
          </Button>
        )}
      </div>
    );
  }
  if (ticket?.closedBySv) {
    return (
      <p className={s.waiting}>
        <StatusText tone="green">Câu hỏi đã đóng</StatusText>
      </p>
    );
  }
  return (
    <p className={s.waiting}>
      <StatusText tone="amber">Đang chờ giảng viên · {sentLabel(ticket, now)}</StatusText>
    </p>
  );
}

function AskTeacher() {
  const [sent, setSent] = useState(false);
  const [tickets, setTickets] = useDemoSlice<Ticket[]>(KEYS.tickets, TICKETS_SEED);
  const now = useSimNow();
  if (sent) return <StatusText tone="amber">Đang chờ giảng viên · {sentLabel(tickets.find((t) => t.id === "tk-d3"), now)}</StatusText>;
  return (
    <button
      type="button"
      className={s.linkBtn}
      onClick={() => {
        setTickets((prev) => (prev.some((t) => t.id === "tk-d3") ? prev : [ticketD3(simNowMs()), ...prev]));
        noteNewTicket(COURSE_1, "tk-d3");
        noteTicketSent(STUDENT_B.id, "tk-d3");
        setSent(true);
      }}
    >
      Nhờ giảng viên hỗ trợ
    </button>
  );
}
