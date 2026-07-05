"use client";

import { Suspense } from "react";

import { useEffect, useRef, useState, type RefObject } from "react";
import {
  AlertTriangle,
  CircleHelp,
  Loader2,
  Paperclip,
  Send,
  Square,
  Search,
  ThumbsDown,
  ThumbsUp,
  UserPlus,
  Zap,
  Trash2,
  Sparkles,
  BookOpen,
  Calendar,
  GraduationCap,
  Shield,
} from "lucide-react";
import { useSearchParams } from "next/navigation";
import { WorkspaceLayout } from "@/components/layout/WorkspaceLayout";
import { streamPrivateChat } from "@/services/aiClient";
import { chatService } from "@/services/chat.service";
import { extractErrorMessage } from "@/lib/utils";
import { usePrivateChatSessions } from "@/hooks/usePrivateChatSessions";
import { usePrivacyStore } from "@/store/privacyStore";
import { useUiStore } from "@/store/uiStore";
import type { ChatFeedback, ChatMessageWire, ChatSessionWire } from "@/types/chat";
import dynamic from "next/dynamic";
const MarkdownRenderer = dynamic(() => import("@/components/ui/MarkdownRenderer"), { ssr: false });

function toTimeLabel(date = new Date()): string {
  return date.toLocaleTimeString([], { hour: "2-digit", minute: "2-digit" });
}

function parseServerDate(raw: string): Date {
  const normalized = /([zZ]|[+-]\d{2}:\d{2})$/.test(raw) ? raw : `${raw}Z`;
  return new Date(normalized);
}

const SessionGroup = ({
  label,
  items,
  activeId,
  onSelect,
  onDelete,
}: {
  label: string;
  items: ChatSessionWire[];
  activeId: string | null;
  onSelect: (id: string) => void;
  onDelete: (id: string) => void;
}) => {
  const sessionTitles = useUiStore((state) => state.sessionTitles);
  
  return (
    <section>
      <p className="px-1 text-[10px] font-semibold uppercase tracking-wide text-slate-400">{label}</p>
      <ul className="mt-2 space-y-1.5">
        {items.map((session) => {
          const active = session.session_id === activeId;
          const isDraft = session.session_id.startsWith("DRAFT_");
          const title = isDraft
            ? "New Conversation"
            : sessionTitles[session.session_id] || session.title || `Conversation #${items.length - items.findIndex((s) => s.session_id === session.session_id)}`;
          return (
            <li key={session.session_id}>
              <div className={`group relative rounded-lg border transition-all duration-200 ${
                active
                  ? "border-indigo-200 bg-indigo-50 dark:bg-indigo-900/30 dark:border-indigo-800"
                  : "border-transparent hover:bg-slate-100 dark:hover:bg-slate-800"
              }`}>
                <button
                  type="button"
                  onClick={() => onSelect(session.session_id)}
                  className="w-full text-left px-3 py-2"
                >
                  <p className={`truncate text-sm font-medium ${active ? "text-indigo-900 dark:text-indigo-100" : "text-slate-900 dark:text-slate-100"}`}>{title}</p>
                  {!isDraft && <p className="mt-1 text-xs text-slate-500">{session.session_id.slice(0, 8)}</p>}
                  {isDraft && <p className="mt-1 text-xs text-indigo-500 italic">Draft — type to start</p>}
                </button>
                <button
                  type="button"
                  onClick={(e) => {
                    e.stopPropagation();
                    if (confirm("Delete this conversation?")) {
                      onDelete(session.session_id);
                    }
                  }}
                  className="absolute right-2 top-1/2 -translate-y-1/2 opacity-0 group-hover:opacity-100 transition-opacity p-1.5 rounded-lg hover:bg-red-100 dark:hover:bg-red-900/30 text-slate-400 hover:text-red-600 dark:hover:text-red-400"
                  title="Delete conversation"
                >
                  <Trash2 className="h-4 w-4" />
                </button>
              </div>
            </li>
          );
        })}
      </ul>
    </section>
  );
};

function WeeklyInsightWidget() {
  return (
    <div className="rounded-xl bg-blue-50 p-4 text-blue-900">
      <p className="text-[10px] font-bold uppercase tracking-wide text-blue-700">Weekly Insight</p>
      <p className="mt-2 text-sm">You've asked 12 questions about Algorithms this week.</p>
      <button type="button" className="mt-3 text-xs font-semibold text-blue-700 hover:text-blue-800">
        View detailed analytics
      </button>
    </div>
  );
}

import { memo } from "react";

const AiMessageBubble = memo(function AiMessageBubble({
  message,
  onFeedback,
  feedbackBusy,
}: {
  message: ChatMessageWire;
  onFeedback: (messageId: string, feedback: ChatFeedback) => void;
  feedbackBusy: boolean;
}) {
  const selectedFeedback = message.feedback ?? null;
  const canSubmitFeedback = !message.id.startsWith("chat-ai-");

  const statusLines = message.content.match(/🔍[^\n]+/g) || [];
  const cleanContent = message.content.replace(/🔍[^\n]+/g, "").trim();

  return (
    <div className="flex flex-col gap-3 group/msg">
      {message.is_escalated && (
        <div className="flex max-w-2xl items-start gap-3 rounded-2xl border border-amber-200 bg-amber-50/80 backdrop-blur-md p-4 text-amber-900 shadow-sm animate-in fade-in slide-in-from-top-2 duration-300">
          <AlertTriangle className="mt-0.5 h-5 w-5 shrink-0 text-amber-600" />
          <div className="text-sm font-medium leading-relaxed">
            {message.escalation_message}
          </div>
        </div>
      )}
      <article className="max-w-2xl rounded-3xl rounded-tl-sm border-l-4 border-indigo-500 bg-white/50 dark:bg-indigo-950/10 p-6 shadow-sm glass-panel glass-panel-hover transition-colors duration-300">
      <div className="flex items-center justify-between mb-4">
        <div className="inline-flex items-center gap-1.5 rounded-full bg-indigo-100/80 dark:bg-indigo-900/40 px-3 py-1 text-[10px] font-bold uppercase tracking-widest text-indigo-700 dark:text-indigo-300">
          <Zap className="h-3 w-3 fill-current" />
          Instant Answer
        </div>
        <span className="text-[10px] font-bold uppercase tracking-widest text-slate-400 opacity-0 group-hover/msg:opacity-100 transition-opacity">
          AI Assistant
        </span>
      </div>
      
      {statusLines.length > 0 && (
        <div className="flex flex-wrap gap-2 mb-4">
          {statusLines.map((status, idx) => (
            <div key={idx} className="inline-flex items-center gap-2 text-[10px] font-bold text-slate-500 dark:text-slate-400 bg-slate-100/50 dark:bg-slate-800/30 w-fit px-3 py-1.5 rounded-lg border border-slate-200/50 dark:border-zinc-800/50 animate-pulse">
              <Search className="h-3 w-3" />
              {status.replace("🔍", "").trim()}
            </div>
          ))}
        </div>
      )}

      <div className="text-sm leading-7 text-slate-700 dark:text-slate-200 font-medium prose prose-slate dark:prose-invert max-w-none">
        <MarkdownRenderer content={cleanContent} />
      </div>

      {/* --- Premium Feedback Bar --- */}
      <div className="mt-6 flex flex-wrap items-center gap-2 border-t border-slate-200/60 dark:border-zinc-800/50 pt-4">
        <button
          type="button"
          disabled={feedbackBusy || !canSubmitFeedback}
          onClick={() => onFeedback(message.id, "LIKE")}
          className={`inline-flex items-center gap-1.5 rounded-full px-4 py-2 text-xs font-bold transition-all duration-200 active:scale-95 disabled:opacity-40 disabled:cursor-not-allowed ${
            selectedFeedback === "LIKE"
              ? "bg-emerald-100 text-emerald-700 dark:bg-emerald-900/40 dark:text-emerald-300 shadow-sm ring-1 ring-emerald-200 dark:ring-emerald-800"
              : "text-slate-500 hover:bg-slate-100 hover:text-slate-700 dark:hover:bg-zinc-800 dark:hover:text-slate-300"
          }`}
        >
          <ThumbsUp className={`h-3.5 w-3.5 transition-transform duration-200 ${selectedFeedback === "LIKE" ? "scale-110" : ""}`} />
          Helpful
        </button>
        <button
          type="button"
          disabled={feedbackBusy || !canSubmitFeedback}
          onClick={() => onFeedback(message.id, "DISLIKE")}
          className={`inline-flex items-center gap-1.5 rounded-full px-4 py-2 text-xs font-bold transition-all duration-200 active:scale-95 disabled:opacity-40 disabled:cursor-not-allowed ${
            selectedFeedback === "DISLIKE"
              ? "bg-rose-100 text-rose-700 dark:bg-rose-900/40 dark:text-rose-300 shadow-sm ring-1 ring-rose-200 dark:ring-rose-800"
              : "text-slate-500 hover:bg-slate-100 hover:text-slate-700 dark:hover:bg-zinc-800 dark:hover:text-slate-300"
          }`}
        >
          <ThumbsDown className={`h-3.5 w-3.5 transition-transform duration-200 ${selectedFeedback === "DISLIKE" ? "scale-110" : ""}`} />
          Not helpful
        </button>

        <div className="mx-2 h-4 w-px bg-slate-200 dark:bg-zinc-800" />

        <button
          type="button"
          disabled={feedbackBusy || !canSubmitFeedback}
          onClick={() => onFeedback(message.id, "NEEDS_TA")}
          className={`inline-flex items-center gap-1.5 rounded-full px-4 py-2 text-xs font-bold transition-all duration-200 active:scale-95 disabled:opacity-40 disabled:cursor-not-allowed ${
            selectedFeedback === "NEEDS_TA"
              ? "bg-indigo-600 text-white shadow-lg shadow-indigo-500/30"
              : "border border-indigo-200 dark:border-indigo-800 text-indigo-600 dark:text-indigo-400 hover:bg-indigo-50 dark:hover:bg-indigo-900/30"
          }`}
        >
          <UserPlus className="h-3.5 w-3.5" />
          {selectedFeedback === "NEEDS_TA" ? "TA notified ✓" : "Needs TA"}
        </button>

        <span className="ml-auto text-[11px] font-bold text-slate-400 dark:text-slate-500 tracking-tight">{toTimeLabel(parseServerDate(message.created_at))}</span>
      </div>
    </article>
    </div>
  );
}, (prevProps, nextProps) => {
  return (
    prevProps.message.content === nextProps.message.content &&
    prevProps.message.feedback === nextProps.message.feedback &&
    prevProps.message.is_escalated === nextProps.message.is_escalated &&
    prevProps.feedbackBusy === nextProps.feedbackBusy
  );
});

const StudentMessageBubble = memo(function StudentMessageBubble({ message }: { message: ChatMessageWire }) {
  return (
    <div className="flex justify-end gap-3 group/user">
      <article className="max-w-xl rounded-3xl rounded-tr-sm bg-gradient-to-br from-indigo-600 to-violet-700 p-5 text-sm leading-7 text-white shadow-xl shadow-indigo-500/10 font-medium">
        <MarkdownRenderer content={message.content} />
      </article>
      <div className="flex h-10 w-10 shrink-0 items-center justify-center rounded-2xl bg-white dark:bg-zinc-900 border border-slate-200 dark:border-zinc-800 text-[10px] font-bold text-indigo-600 dark:text-indigo-400 shadow-sm transition-transform group-hover/user:scale-110">
        AR
      </div>
    </div>
  );
}, (prevProps, nextProps) => {
  return prevProps.message.content === nextProps.message.content;
});

function ChatMetaBar({ messageCount, isStreaming }: { messageCount: number; isStreaming: boolean }) {
  return (
    <div className="mb-2 flex items-center justify-center">
      <div className="inline-flex items-center gap-4 rounded-full bg-white/60 dark:bg-zinc-900/40 px-4 py-1.5 text-[10px] font-semibold uppercase tracking-wider text-slate-400 backdrop-blur-sm border border-slate-200/50 dark:border-zinc-800/50 shadow-sm">
        <span className="inline-flex items-center gap-1.5">
          <span className={`h-1.5 w-1.5 rounded-full ${isStreaming ? 'bg-amber-500 animate-pulse' : 'bg-emerald-500'} shadow-sm`} />
          {isStreaming ? 'Streaming' : 'Ready'}
        </span>
        <span className="h-3 w-px bg-slate-200 dark:bg-zinc-700" />
        <span className="inline-flex items-center gap-1">
          <Sparkles className="h-3 w-3 text-indigo-400" />
          GPT-5.4-mini
        </span>
        {messageCount > 0 && (
          <>
            <span className="h-3 w-px bg-slate-200 dark:bg-zinc-700" />
            <span className="inline-flex items-center gap-1">
              <CircleHelp className="h-3 w-3" />
              {messageCount} msg{messageCount !== 1 ? 's' : ''}
            </span>
          </>
        )}
      </div>
    </div>
  );
}

const SUGGESTION_CHIPS = [
  { icon: BookOpen, label: "Giải thích khái niệm", prompt: "Giải thích khái niệm " },
  { icon: Calendar, label: "Deadline bài tập", prompt: "Cho tôi biết deadline các bài tập sắp tới" },
  { icon: GraduationCap, label: "Ôn tập kiến thức", prompt: "Giúp tôi ôn tập về " },
  { icon: Shield, label: "Hướng dẫn bài lab", prompt: "Hướng dẫn tôi làm bài lab " },
];

function ChatComposer({
  draft,
  onChange,
  onSend,
  onStop,
  sending,
  isStreaming,
  inputRef,
  showSuggestions,
}: {
  draft: string;
  onChange: (value: string) => void;
  onSend: () => void;
  onStop: () => void;
  sending: boolean;
  isStreaming: boolean;
  inputRef: RefObject<HTMLTextAreaElement | null>;
  showSuggestions?: boolean;
}) {
  const handleKeyDown = (e: React.KeyboardEvent) => {
    if (e.key === "Enter" && !e.shiftKey) {
      e.preventDefault();
      onSend();
    }
  };

  // Auto-resize textarea
  const handleInput = (e: React.ChangeEvent<HTMLTextAreaElement>) => {
    onChange(e.target.value);
    const el = e.target;
    el.style.height = 'auto';
    el.style.height = Math.min(el.scrollHeight, 160) + 'px';
  };

  return (
    <div className="mx-auto w-full max-w-3xl space-y-3">
      {/* Suggestion Chips — only when no messages */}
      {showSuggestions && !draft && (
        <div className="flex flex-wrap justify-center gap-2 animate-in fade-in slide-in-from-bottom-3 duration-500">
          {SUGGESTION_CHIPS.map((chip) => (
            <button
              key={chip.label}
              type="button"
              onClick={() => {
                onChange(chip.prompt);
                inputRef.current?.focus();
              }}
              className="group inline-flex items-center gap-2 rounded-full border border-slate-200 dark:border-zinc-700 bg-white dark:bg-zinc-900 px-4 py-2 text-xs font-medium text-slate-600 dark:text-slate-300 shadow-sm transition-all duration-200 hover:border-indigo-300 hover:bg-indigo-50 dark:hover:bg-indigo-950/30 hover:text-indigo-700 dark:hover:text-indigo-300 hover:shadow-md hover:-translate-y-0.5 active:scale-95"
            >
              <chip.icon className="h-3.5 w-3.5 text-slate-400 group-hover:text-indigo-500 transition-colors" />
              {chip.label}
            </button>
          ))}
        </div>
      )}

      {/* Composer Input */}
      <div className={`relative rounded-2xl border bg-white dark:bg-zinc-900 shadow-lg transition-all duration-300 ${
        sending
          ? 'border-amber-200 dark:border-amber-800 shadow-amber-500/5'
          : 'border-slate-200 dark:border-zinc-700 shadow-indigo-500/5 focus-within:border-indigo-400 focus-within:shadow-xl focus-within:shadow-indigo-500/10'
      }`}>
        <div className="flex items-end gap-2 p-3">
          {/* Attach button */}
          <button
            type="button"
            className="mb-0.5 shrink-0 rounded-xl p-2 text-slate-400 transition-all hover:bg-slate-100 dark:hover:bg-zinc-800 hover:text-slate-600 active:scale-90"
            title="Attach file"
          >
            <Paperclip className="h-5 w-5" />
          </button>

          {/* Text Input */}
          <div className="relative flex-1 min-w-0">
            <textarea
              ref={inputRef}
              value={draft}
              disabled={sending}
              onChange={handleInput}
              onKeyDown={handleKeyDown}
              rows={1}
              placeholder={sending ? "AI đang xử lý câu hỏi của bạn..." : "Hỏi bất kỳ điều gì về khóa học..."}
              className="w-full resize-none bg-transparent py-2 text-sm font-medium text-slate-800 dark:text-slate-100 placeholder:text-slate-400 outline-none disabled:cursor-not-allowed disabled:opacity-50 leading-relaxed max-h-40"
              style={{ height: 'auto', minHeight: '24px' }}
            />
          </div>

          {/* Action Buttons */}
          <div className="mb-0.5 flex shrink-0 items-center gap-1.5">
            {isStreaming ? (
              <button
                type="button"
                onClick={onStop}
                className="flex items-center gap-2 rounded-xl bg-gradient-to-r from-rose-500 to-pink-500 px-4 py-2.5 text-xs font-bold text-white shadow-lg shadow-rose-500/25 transition-all hover:shadow-xl hover:shadow-rose-500/30 active:scale-95"
                aria-label="Stop generating"
              >
                <Square className="h-3.5 w-3.5 fill-current" />
                Stop
              </button>
            ) : (
              <button
                type="button"
                onClick={onSend}
                disabled={sending || !draft.trim()}
                className={`group flex items-center gap-2 rounded-xl px-4 py-2.5 text-xs font-bold text-white shadow-lg transition-all duration-200 active:scale-95 disabled:cursor-not-allowed disabled:opacity-40 disabled:shadow-none ${
                  draft.trim()
                    ? 'bg-gradient-to-r from-indigo-600 to-violet-600 shadow-indigo-500/25 hover:shadow-xl hover:shadow-indigo-500/30'
                    : 'bg-slate-300 dark:bg-zinc-700'
                }`}
              >
                <Send className={`h-4 w-4 transition-transform duration-200 ${draft.trim() ? 'group-hover:translate-x-0.5 group-hover:-translate-y-0.5' : ''}`} />
                Send
              </button>
            )}
          </div>
        </div>

        {/* Bottom hints bar */}
        <div className="flex items-center justify-between border-t border-slate-100 dark:border-zinc-800 px-4 py-1.5">
          <div className="flex items-center gap-3 text-[10px] text-slate-400">
            <span className="inline-flex items-center gap-1">
              <kbd className="rounded bg-slate-100 dark:bg-zinc-800 px-1 py-0.5 font-mono text-[9px] font-semibold text-slate-500">Enter</kbd>
              <span>gửi</span>
            </span>
            <span className="inline-flex items-center gap-1">
              <kbd className="rounded bg-slate-100 dark:bg-zinc-800 px-1 py-0.5 font-mono text-[9px] font-semibold text-slate-500">Shift+Enter</kbd>
              <span>xuống dòng</span>
            </span>
          </div>
          {draft.length > 0 && (
            <span className={`text-[10px] font-semibold tabular-nums ${draft.length > 2000 ? 'text-rose-500' : 'text-slate-400'}`}>
              {draft.length}
            </span>
          )}
        </div>
      </div>
    </div>
  );
}


function ChatContent() {
  const searchParams = useSearchParams();
  const [draft, setDraft] = useState("");
  const [sending, setSending] = useState(false);
  const [isTyping, setIsTyping] = useState(false);

  const [streamError, setStreamError] = useState<string | null>(null);
  const [feedbackBusyByMessage, setFeedbackBusyByMessage] = useState<Record<string, boolean>>({});
  const scrollViewportRef = useRef<HTMLDivElement | null>(null);
  const composerInputRef = useRef<HTMLTextAreaElement | null>(null);
  const abortControllerRef = useRef<AbortController | null>(null);
  const {
    sessions,
    activeSessionId,
    setActiveSessionId,
    messagesBySession,
    sessionsLoading,
    messagesLoading,
    createLoading,
    sessionsError,
    messagesError,
    createSession,
    ensureSession,
    deleteSession,
    updateSessionTitle,
    loadMessages,
    setMessagesBySession,
    setMessagesError,
  } = usePrivateChatSessions();

  const { pendingPrivateText, clearPendingPrivateText } = usePrivacyStore();

  useEffect(() => {
    if (pendingPrivateText) {
      setDraft(pendingPrivateText);
      clearPendingPrivateText();
    }
  }, [pendingPrivateText, clearPendingPrivateText]);

  const sessionTitles = useUiStore((state) => state.sessionTitles);
  const activeSession = sessions.find((s) => s.session_id === activeSessionId);
  const isDraftSession = activeSessionId?.startsWith("DRAFT_");
  
  const activeIndex = sessions.findIndex((s) => s.session_id === activeSessionId);
  const defaultTitle = activeIndex >= 0 ? `Conversation #${sessions.length - activeIndex}` : "Private AI Tutor Conversation";
  
  const activeTitle = activeSessionId
    ? isDraftSession
      ? "New Conversation — type your first message"
      : sessionTitles[activeSessionId] || activeSession?.title || defaultTitle
    : "Private AI Tutor — Click '+ New Conversation' to start";
  const messages = activeSessionId ? (messagesBySession[activeSessionId] ?? []) : [];

  useEffect(() => {
    if (!scrollViewportRef.current) return;
    const viewport = scrollViewportRef.current;
    viewport.scrollTo({
      top: viewport.scrollHeight,
      behavior: isTyping ? "auto" : "smooth",
    });
  }, [activeSessionId, messages, isTyping]);

  useEffect(() => {
    setDraft("");
    const id = window.setTimeout(() => composerInputRef.current?.focus(), 0);
    return () => window.clearTimeout(id);
  }, [activeSessionId]);

  useEffect(() => {
    const prefill = searchParams.get("prefill");
    if (!prefill) return;
    setDraft(prefill);
  }, [searchParams]);

  useEffect(() => {
    return () => {
      abortControllerRef.current?.abort();
      abortControllerRef.current = null;
    };
  }, []);

  const handleCreateConversation = async () => {
    const sessionId = await createSession();
    if (sessionId) {
      setStreamError(null);
      setMessagesError(null);
    }
  };

  const handleStop = () => {
    abortControllerRef.current?.abort();
    abortControllerRef.current = null;
    setIsTyping(false);
    setSending(false);
  };

  const handleSend = async () => {
    if (!activeSessionId) return;
    const content = draft.trim();
    if (!content || sending) return;

    // Lazy session creation: if this is a draft, create real session first
    let sessionId = activeSessionId;
    if (sessionId.startsWith("DRAFT_")) {
      const realId = await ensureSession(sessionId);
      if (!realId) return; // creation failed
      sessionId = realId;
    }

    // Auto-generate title from first message (truncated to 50 chars)
    const currentMessages = messagesBySession[sessionId] ?? [];
    if (currentMessages.length === 0) {
      const autoTitle = content.length > 50 ? content.slice(0, 50) + "..." : content;
      updateSessionTitle(sessionId, autoTitle);
    }

    const nowIso = new Date().toISOString();
    const studentLocalId = `chat-student-${Date.now()}`;
    const aiLocalId = `chat-ai-${Date.now() + 1}`;

    setSending(true);
    setStreamError(null);
    setDraft("");

    const aiMessageTemplate: ChatMessageWire = {
      id: aiLocalId,
      author: { id: "ai_bot", role: "AI", name: "Tro Giang AI", avatar: null },
      content: "",
      created_at: nowIso,
    };

    setMessagesBySession((prev) => ({
      ...prev,
      [sessionId]: [
        ...(prev[sessionId] ?? []),
        {
          id: studentLocalId,
          author: { id: "me", role: "STUDENT", name: "You", avatar: null },
          content,
          created_at: nowIso,
        },
      ],
    }));
    setIsTyping(true);
    let streamFailed = false;
    const controller = new AbortController();
    abortControllerRef.current = controller;

    try {
      await streamPrivateChat(sessionId, content, {
        onChunk: (chunk) => {
          setIsTyping(false);
          setMessagesBySession((prev) => {
            const currentSessionMsgs = prev[sessionId] ?? [];
            const exists = currentSessionMsgs.find((m) => m.id === aiLocalId);
            if (exists) {
              return {
                ...prev,
                [sessionId]: currentSessionMsgs.map((m) =>
                  m.id === aiLocalId ? { ...m, content: `${m.content}${chunk}` } : m
                ),
              };
            }
            return {
              ...prev,
              [sessionId]: [...currentSessionMsgs, { ...aiMessageTemplate, content: chunk }],
            };
          });
        },
        onEscalation: (msg) => {
          setIsTyping(false);
          setMessagesBySession((prev) => {
            const currentSessionMsgs = prev[sessionId] ?? [];
            const exists = currentSessionMsgs.find((m) => m.id === aiLocalId);
            if (exists) {
              return {
                ...prev,
                [sessionId]: currentSessionMsgs.map((m) =>
                  m.id === aiLocalId ? { ...m, is_escalated: true, escalation_message: msg } : m
                ),
              };
            }
            return {
              ...prev,
              [sessionId]: [...currentSessionMsgs, { ...aiMessageTemplate, is_escalated: true, escalation_message: msg }],
            };
          });
        },
        onFinish: () => {
          setIsTyping(false);
          setMessagesBySession((prev) => {
            const currentSessionMsgs = prev[sessionId] ?? [];
            const exists = currentSessionMsgs.find((m) => m.id === aiLocalId);
            if (!exists) {
              return {
                ...prev,
                [sessionId]: [...currentSessionMsgs, aiMessageTemplate],
              };
            }
            return prev;
          });
        },
        onError: (error) => {
          streamFailed = true;
          setIsTyping(false);
          setMessagesBySession((prev) => {
            const currentSessionMsgs = prev[sessionId] ?? [];
            const exists = currentSessionMsgs.find((m) => m.id === aiLocalId);
            if (exists) {
              return {
                ...prev,
                [sessionId]: currentSessionMsgs.map((m) =>
                  m.id === aiLocalId ? { ...m, content: error } : m
                ),
              };
            }
            return {
              ...prev,
              [sessionId]: [...currentSessionMsgs, { ...aiMessageTemplate, content: error }],
            };
          });
        },
        onRateLimit: (msg) => {
          streamFailed = true;
          setIsTyping(false);
          setSending(false);
          const errorMsg = msg || "Hệ thống đang quá tải.";
          setMessagesBySession((prev) => {
            const currentSessionMsgs = prev[sessionId] ?? [];
            const exists = currentSessionMsgs.find((m) => m.id === aiLocalId);
            if (exists) {
              return {
                ...prev,
                [sessionId]: currentSessionMsgs.map((m) =>
                  m.id === aiLocalId ? { ...m, content: errorMsg } : m
                ),
              };
            }
            return {
              ...prev,
              [sessionId]: [...currentSessionMsgs, { ...aiMessageTemplate, content: errorMsg }],
            };
          });
        },
        onSecurityBlock: () => {
          streamFailed = true;
          setIsTyping(false);
          setSending(false);
        },
      }, controller.signal);
    } catch (error) {
      streamFailed = true;
      setIsTyping(false);
      const errorMessage = extractErrorMessage(error, "Failed to stream AI response.");
      setMessagesBySession((prev) => {
        const currentSessionMsgs = prev[sessionId] ?? [];
        const exists = currentSessionMsgs.find((m) => m.id === aiLocalId);
        if (exists) {
          return {
            ...prev,
            [sessionId]: currentSessionMsgs.map((m) =>
              m.id === aiLocalId ? { ...m, content: errorMessage } : m
            ),
          };
        }
        return {
          ...prev,
          [sessionId]: [...currentSessionMsgs, { ...aiMessageTemplate, content: errorMessage }],
        };
      });
    } finally {
      // Failsafe to avoid stuck "AI is typing..." if SSE closes unexpectedly.
      setIsTyping(false);
      setSending(false);
      abortControllerRef.current = null;
      // Reload messages from server to replace optimistic local IDs
      // with real persisted IDs so feedback buttons become clickable.
      if (!streamFailed) {
        setTimeout(() => {
          void loadMessages(sessionId, true);
        }, 1500);
      }
    }
  };

  const handleSubmitFeedback = async (messageId: string, feedback: ChatFeedback) => {
    if (!activeSessionId) return;
    const sessionId = activeSessionId;
    setFeedbackBusyByMessage((prev) => ({ ...prev, [messageId]: true }));
    setMessagesBySession((prev) => ({
      ...prev,
      [sessionId]: (prev[sessionId] ?? []).map((m) => (m.id === messageId ? { ...m, feedback } : m)),
    }));
    try {
      await chatService.submitFeedback(messageId, feedback);
    } catch (error) {
      setMessagesError(extractErrorMessage(error, "Could not submit feedback."));
      await Promise.resolve();
    } finally {
      setFeedbackBusyByMessage((prev) => ({ ...prev, [messageId]: false }));
    }
  };

  return (
    <WorkspaceLayout footerLine2="rights" avatarInitialsOverride="AR" searchPlaceholder="Search in your private chat...">
      <div className="flex h-full">
        <aside className="flex w-72 shrink-0 flex-col border-r border-slate-200 bg-slate-50 px-3 py-3">
          <div className="space-y-2">
            <label className="relative block">
              <Search className="pointer-events-none absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-slate-400" />
              <input
                type="search"
                placeholder="Search history..."
                className="w-full rounded-lg border border-slate-200 bg-white py-2 pl-9 pr-3 text-sm text-slate-700 placeholder:text-slate-400 outline-none ring-blue-100 focus:border-blue-300 focus:ring-4"
              />
            </label>
            <button
              type="button"
              onClick={handleCreateConversation}
              disabled={createLoading}
              className="w-full rounded-lg border border-slate-300 bg-white px-3 py-2 text-sm font-semibold text-slate-700 hover:bg-slate-100"
            >
              {createLoading ? "Creating..." : "+ New Conversation"}
            </button>
          </div>

          <div className="mt-4 flex-1 space-y-4 overflow-y-auto pb-3">
            {sessionsLoading ? (
              <div className="space-y-2">
                <div className="h-11 animate-pulse rounded-lg bg-slate-200" />
                <div className="h-11 animate-pulse rounded-lg bg-slate-200" />
                <div className="h-11 animate-pulse rounded-lg bg-slate-200" />
              </div>
            ) : sessions.length > 0 ? (
              <SessionGroup label="Conversations" items={sessions} activeId={activeSessionId} onSelect={setActiveSessionId} onDelete={deleteSession} />
            ) : (
              <p className="rounded-lg border border-dashed border-slate-300 p-3 text-xs text-slate-500 text-center">
                No conversations yet.
              </p>
            )}
          </div>

          <WeeklyInsightWidget />
          <p className="mt-3 text-center text-[10px] text-slate-400">© 2024 EduPilot AI. All rights reserved.</p>
        </aside>

        <main className="relative flex min-w-0 flex-1 flex-col bg-slate-50/50 dark:bg-zinc-950/50">
          {/* Premium Sticky Header */}
          <header className="sticky top-0 z-20 flex h-16 shrink-0 items-center justify-between border-b border-slate-200/60 bg-white/80 px-6 backdrop-blur-xl dark:border-zinc-800/60 dark:bg-zinc-900/80 shadow-sm">
            <div className="flex items-center gap-3 min-w-0">
              <div className="flex shrink-0 h-9 w-9 items-center justify-center rounded-xl bg-gradient-to-tr from-indigo-500 to-violet-500 text-white shadow-md shadow-indigo-500/20">
                <Sparkles className="h-4 w-4" />
              </div>
              <h1 className="text-sm font-bold text-slate-800 dark:text-slate-100 truncate pr-4">
                {activeTitle}
              </h1>
            </div>
            <div className="flex shrink-0 items-center gap-2">
              {/* Future actions: Export, Share, etc. */}
            </div>
          </header>

          <div ref={scrollViewportRef} className="flex-1 overflow-y-auto px-6 py-6 pb-40 scroll-smooth">
            {sessionsError ? (
              <div className="mb-3 rounded-lg border border-red-200 bg-red-50 px-3 py-2 text-sm text-red-700">{sessionsError}</div>
            ) : null}
            {messagesError ? (
              <div className="mb-3 rounded-lg border border-amber-200 bg-amber-50 px-3 py-2 text-sm text-amber-700">{messagesError}</div>
            ) : null}
            {streamError ? (
              <div className="mb-3 rounded-lg border border-red-200 bg-red-50 px-3 py-2 text-sm text-red-700">{streamError}</div>
            ) : null}

            <div className="space-y-4">
              {messagesLoading && messages.length === 0 ? (
                <div className="space-y-2">
                  <div className="h-16 animate-pulse rounded-2xl bg-slate-200" />
                  <div className="h-20 animate-pulse rounded-2xl bg-slate-200" />
                </div>
              ) : null}
              {messages.map((message) =>
                message.author.role === "AI" ? (
                  <AiMessageBubble
                    key={message.id}
                    message={message}
                    onFeedback={handleSubmitFeedback}
                    feedbackBusy={Boolean(feedbackBusyByMessage[message.id])}
                  />
                ) : (
                  <StudentMessageBubble key={message.id} message={message} />
                ),
              )}
              {isTyping ? (
                <div className="max-w-2xl rounded-3xl rounded-tl-sm border-l-4 border-indigo-500 bg-white/50 dark:bg-indigo-950/10 p-6 shadow-sm glass-panel animate-in fade-in slide-in-from-bottom-2 duration-300">
                  <div className="inline-flex items-center gap-2 text-[10px] font-bold uppercase tracking-widest text-indigo-700 dark:text-indigo-400">
                    <Loader2 className="h-3.5 w-3.5 animate-spin" />
                    AI is thinking...
                  </div>
                  <div className="mt-4 space-y-3">
                    <div className="h-2 w-full max-w-[440px] rounded-full animate-shimmer" />
                    <div className="h-2 w-full max-w-[320px] rounded-full animate-shimmer" style={{ animationDelay: '0.2s' }} />
                    <div className="h-2 w-full max-w-[180px] rounded-full animate-shimmer" style={{ animationDelay: '0.4s' }} />
                  </div>
                </div>
              ) : null}
            </div>
          </div>

          <div className="sticky bottom-0 border-t border-slate-200/50 bg-slate-50/80 px-4 pb-4 pt-3 backdrop-blur-xl dark:border-zinc-800/50 dark:bg-zinc-950/80">
            <ChatMetaBar messageCount={messages.length} isStreaming={isTyping || sending} />
            <ChatComposer
              draft={draft}
              onChange={setDraft}
              onSend={handleSend}
              onStop={handleStop}
              sending={sending || !activeSessionId}
              isStreaming={isTyping || sending}
              inputRef={composerInputRef as React.RefObject<HTMLTextAreaElement>}
              showSuggestions={messages.length === 0}
            />
            <p className="mt-3 flex items-center justify-center gap-1.5 text-center text-[10px] text-slate-400">
              <Shield className="h-3 w-3 text-slate-300" />
              EduPilot AI can make mistakes. Verify important info against the{" "}
              <span className="font-semibold text-indigo-500 hover:text-indigo-600 cursor-pointer transition-colors">Official Course Assistant</span>.
            </p>
          </div>
        </main>
      </div>
    </WorkspaceLayout>
  );
}

export default function ChatPage() {
  return (
    <Suspense fallback={<div className="flex h-screen w-full items-center justify-center bg-slate-50"><Loader2 className="h-6 w-6 animate-spin text-indigo-500" /></div>}>
      <ChatContent />
    </Suspense>
  );
}
