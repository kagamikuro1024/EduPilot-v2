"use client";

import {
  AtSign,
  BookOpen,
  CheckCircle2,
  ExternalLink,
  FileText,
  Loader2,
  LogOut,
  MessageSquare,
  Paperclip,
  Send,
  Square,
  Settings,
  Smile,
  ThumbsUp,
  Users,
  Eye,
  Clock3,
  BarChart3,
  Search,
} from "lucide-react";
import { useParams, useRouter, useSearchParams } from "next/navigation";
import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import { WorkspaceLayout } from "@/components/layout/WorkspaceLayout";
import { streamChat } from "@/services/aiClient";
import { threadsService, type ThreadPostWire } from "@/services/threads.service";
import { extractErrorMessage } from "@/lib/utils";
import { usePiiDebounce } from "@/hooks/usePiiDebounce";
import { usePrivacyStore } from "@/store/privacyStore";
import { useAuthStore } from "@/store/authStore";
import { AlertCircle } from "lucide-react";
import {
  mapSseMetadataToCitations,
  mergeRagDocs,
  type RAGDocument,
} from "@/features/threads/ragCitations";
import dynamic from "next/dynamic";
const MarkdownRenderer = dynamic(() => import("@/components/ui/MarkdownRenderer"), { ssr: false });

type SenderRole = "STUDENT" | "AI" | "TA";
type VerificationStatus = "UNVERIFIED" | "VERIFIED" | "CORRECTED" | "REJECTED";

interface Message {
  id: string;
  role: SenderRole;
  authorName: string;
  avatarUrl: string;
  content: string;
  timestamp: string;
  likes: number;
  isAcceptedAnswer: boolean;
  verificationStatus: VerificationStatus;
  verifiedByName: string | null;
  originalAiContent: string | null;
  attachedCitations: RAGDocument[];
}

interface ThreadInfo {
  breadcrumbs: string[];
  title: string;
  tags: string[];
  createdAgo: string;
  views: string;
  studentCount: string;
  aiAnswered: boolean;
  taVerified: boolean;
}

const SHOW_AI_TYPING = true;

const MOCK_THREAD_INFO: ThreadInfo = {
  breadcrumbs: ["CS101: Systems Programming", "Week 4: Memory Management"],
  title: "Understanding Pointers in C++: Memory address vs Values",
  tags: ["Week 4", "Assignment 2", "Pointers"],
  createdAgo: "Created 2h ago",
  views: "124 views",
  studentCount: "8 students",
  aiAnswered: true,
  taVerified: true,
};

const MOCK_SIMILAR_QUESTIONS: string[] = [
  "Difference between stack and heap pointers?",
  "Why does my pointer return 0x0000?",
  "Pointer arithmetic best practices",
];

function toInitials(name: string): string {
  const parts = name.trim().split(/\s+/).filter(Boolean);
  if (parts.length === 0) return "NA";
  if (parts.length === 1) return parts[0].slice(0, 2).toUpperCase();
  return `${parts[0][0]}${parts[1][0]}`.toUpperCase();
}

function toSenderRole(rawRole: string | null | undefined): SenderRole {
  const role = (rawRole ?? "").toUpperCase();
  if (role === "AI") return "AI";
  if (role === "TA" || role === "ADMIN") return "TA";
  return "STUDENT";
}

function toTimeLabel(iso: string | null | undefined): string {
  if (!iso) return "";
  const normalizedIso = /([zZ]|[+-]\d{2}:\d{2})$/.test(iso) ? iso : `${iso}Z`;
  const d = new Date(normalizedIso);
  if (Number.isNaN(d.getTime())) return "";
  return d.toLocaleTimeString([], { hour: "2-digit", minute: "2-digit" });
}

function parseCitation(item: Record<string, unknown>, fallbackId: string): RAGDocument | null {
  const title = typeof item.title === "string" ? item.title
    : typeof item.document_title === "string" ? item.document_title
    : typeof item.source_file === "string" ? item.source_file
    : null;
  if (!title) return null;
  const pageRaw = item.pageNumber ?? item.page_number ?? item.page;
  const pageNum = typeof pageRaw === "number" ? pageRaw : Number(pageRaw);
  return {
    id: String(item.id ?? item.chunk_id ?? fallbackId),
    title,
    pageNumber: Number.isFinite(pageNum) ? pageNum : 0,
    snippet: typeof item.snippet === "string" ? item.snippet : "",
  };
}

function mapWirePostToMessage(post: ThreadPostWire): Message {
  const likes = (post.reactions?.like ?? post.reactions?.thumbs_up ?? 0) as number;
  let citations = Array.isArray(post.citations)
    ? post.citations
        .map((c, i) => parseCitation(c, `${post.id}-citation-${i}`))
        .filter((x): x is RAGDocument => x != null)
    : [];

  const seen = new Set<string>();
  citations = citations.filter((c) => {
    const key = `${c.title.toLowerCase()}::${c.pageNumber}`;
    if (seen.has(key)) return false;
    seen.add(key);
    return true;
  });
  return {
    id: post.id,
    role: toSenderRole(post.author?.role),
    authorName: post.author?.name ?? "Unknown",
    avatarUrl: toInitials(post.author?.name ?? "Unknown"),
    content: post.content ?? "",
    timestamp: toTimeLabel(post.created_at),
    likes: typeof likes === "number" ? likes : Number(likes) || 0,
    isAcceptedAnswer: Boolean(post.is_accepted),
    verificationStatus: post.verification_status ?? "UNVERIFIED",
    verifiedByName: post.verified_by_ta?.name ?? null,
    originalAiContent: post.original_ai_content ?? null,
    attachedCitations: citations,
  };
}

function IconSidebar() {
  return (
    <aside className="flex w-16 shrink-0 flex-col justify-between border-r border-slate-200 bg-white py-3">
      <div className="space-y-1">
        <button type="button" className="mx-auto flex h-10 w-10 items-center justify-center rounded-lg text-slate-500">
          <MessageSquare className="h-5 w-5" />
        </button>
        <button type="button" className="mx-auto flex h-10 w-10 items-center justify-center rounded-lg text-slate-500">
          <FileText className="h-5 w-5" />
        </button>
        <button type="button" className="mx-auto flex h-10 w-10 items-center justify-center rounded-lg text-slate-500">
          <BarChart3 className="h-5 w-5" />
        </button>
      </div>
      <div className="space-y-1">
        <button type="button" className="mx-auto flex h-10 w-10 items-center justify-center rounded-lg text-slate-500">
          <Settings className="h-5 w-5" />
        </button>
        <button type="button" className="mx-auto flex h-10 w-10 items-center justify-center rounded-lg text-slate-500">
          <LogOut className="h-5 w-5" />
        </button>
      </div>
    </aside>
  );
}

function ThreadHeader({ info, isScrolled }: { info: ThreadInfo; isScrolled?: boolean }) {
  return (
    <header className={`border-b border-slate-200 bg-white/70 backdrop-blur-md transition-all duration-300 ${isScrolled ? "px-6 py-2" : "px-6 py-4"}`}>
      <div className={`overflow-hidden transition-all duration-300 ${isScrolled ? "max-h-0 opacity-0" : "max-h-10 opacity-100"}`}>
        <p className="text-xs text-slate-500 mb-2">{info.breadcrumbs.join("  >  ")}</p>
      </div>
      <div className="flex items-center justify-between gap-3">
        <h1 className={`font-bold tracking-tight text-slate-900 transition-all duration-300 ${isScrolled ? "text-lg truncate" : "text-2xl"}`}>{info.title}</h1>
        <div className="flex items-center gap-2 shrink-0">
          {info.aiAnswered ? (
            <span className="rounded-full border border-green-200 bg-green-50 px-2.5 py-1 text-[10px] sm:text-xs font-semibold text-green-700 whitespace-nowrap">
              AI Answered
            </span>
          ) : null}
          {info.taVerified ? (
            <span className="rounded-full border border-blue-200 bg-blue-50 px-2.5 py-1 text-[10px] sm:text-xs font-semibold text-blue-700 whitespace-nowrap">
              Verified by TA
            </span>
          ) : null}
        </div>
      </div>
      <div className={`overflow-hidden transition-all duration-300 flex flex-wrap gap-2 ${isScrolled ? "max-h-0 opacity-0 mt-0" : "max-h-20 opacity-100 mt-3"}`}>
        {info.tags.map((tag) => (
          <span
            key={tag}
            className="rounded-full border border-slate-200 bg-slate-100 px-2.5 py-1 text-xs font-semibold text-slate-700"
          >
            {tag}
          </span>
        ))}
      </div>
    </header>
  );
}

function AvatarChip({ initials, role }: { initials: string; role: SenderRole }) {
  const tone =
    role === "AI"
      ? "from-slate-700 to-slate-900"
      : role === "TA"
        ? "from-blue-500 to-indigo-600"
        : "from-emerald-500 to-teal-600";
  return (
    <div
      className={`flex h-9 w-9 shrink-0 items-center justify-center rounded-full bg-gradient-to-br text-xs font-bold text-white ${tone}`}
      aria-hidden
    >
      {initials}
    </div>
  );
}


function MessageItem({
  message,
  canModerate,
  actionBusy,
  onVerify,
  onReject,
  onCorrect,
}: {
  message: Message;
  canModerate: boolean;
  actionBusy: boolean;
  onVerify: (messageId: string) => Promise<void>;
  onReject: (messageId: string) => Promise<void>;
  onCorrect: (messageId: string, content: string) => Promise<void>;
}) {
  const [editing, setEditing] = useState(false);
  const [draftCorrection, setDraftCorrection] = useState(message.content);
  const isLocalMessage = message.id.startsWith("local-");
  const canShowActions = canModerate && message.role === "AI" && !isLocalMessage;
  const pendingReview = message.role === "AI" && message.verificationStatus === "UNVERIFIED";

  const verificationBadge = (() => {
    if (message.role !== "AI") return null;
    if (message.verificationStatus === "UNVERIFIED") {
      return (
        <span className="rounded-full border border-amber-200 bg-amber-50 px-2 py-0.5 text-[10px] font-bold uppercase tracking-widest text-amber-800">
          Pending
        </span>
      );
    }
    if (message.verificationStatus === "VERIFIED") {
      return (
        <span className="rounded-full border border-emerald-200 bg-emerald-50 px-2 py-0.5 text-[10px] font-bold uppercase tracking-widest text-emerald-700">
          {message.verifiedByName ? `Verified by ${message.verifiedByName}` : "Verified"}
        </span>
      );
    }
    if (message.verificationStatus === "CORRECTED") {
      return (
        <span className="rounded-full border border-blue-200 bg-blue-50 px-2 py-0.5 text-[10px] font-bold uppercase tracking-widest text-blue-700">
          Edited
        </span>
      );
    }
    return (
      <span className="rounded-full border border-rose-200 bg-rose-50 px-2 py-0.5 text-[10px] font-bold uppercase tracking-widest text-rose-700">
        Rejected
      </span>
    );
  })();

  return (
    <article className={`border-b px-6 py-5 transition-all group/item ${
      message.role === "AI" ? "bg-white/40 dark:bg-indigo-950/5 border-l-4 border-indigo-500" : "bg-white dark:bg-transparent"
    } ${pendingReview ? "border-amber-200" : "border-slate-100 dark:border-zinc-800"}`}>
      <div className="flex items-start gap-4">
        <AvatarChip initials={message.avatarUrl} role={message.role} />
        <div className="min-w-0 flex-1">
          <div className="flex flex-wrap items-center gap-2 mb-2">
            <p className="text-sm font-bold text-slate-900 dark:text-slate-100">{message.authorName}</p>
            {message.role === "AI" ? (
              <span className="rounded-full bg-slate-100 dark:bg-zinc-800 px-2 py-0.5 text-[10px] font-bold uppercase tracking-widest text-slate-600 dark:text-slate-400">Bot</span>
            ) : null}
            {verificationBadge}
            <span className="text-[10px] font-bold uppercase tracking-widest text-slate-400">{message.timestamp}</span>
            {message.isAcceptedAnswer ? (
              <span className="ml-auto inline-flex items-center gap-1 text-[10px] font-bold uppercase tracking-widest text-emerald-600">
                <CheckCircle2 className="h-3.5 w-3.5" />
                Accepted
              </span>
            ) : null}
          </div>

          {message.verificationStatus === "REJECTED" ? (
            <div className="mt-2 rounded-xl border border-rose-100 bg-rose-50 p-3 text-sm text-rose-700 font-medium italic">
              AI answer was rejected by TA.
            </div>
          ) : (
            <div className="mt-2 text-sm leading-7 text-slate-700 dark:text-slate-200 font-medium prose prose-slate dark:prose-invert max-w-none">
              {message.role === "AI" && (
                <div className="flex flex-wrap gap-2 mb-4 not-prose">
                  {(message.content.match(/🔍[^\n]+/g) || []).map((status, idx) => (
                    <div key={idx} className="inline-flex items-center gap-2 text-[10px] font-bold text-slate-500 dark:text-slate-400 bg-slate-100/50 dark:bg-slate-800/30 w-fit px-3 py-1.5 rounded-lg border border-slate-200/50 dark:border-zinc-800/50 animate-pulse">
                      <Search className="h-3 w-3" />
                      {status.replace("🔍", "").trim()}
                    </div>
                  ))}
                </div>
              )}
              <MarkdownRenderer content={message.role === "AI" ? message.content.replace(/🔍[^\n]+/g, "").trim() : message.content} />
            </div>
          )}

          {canShowActions ? (
            <div className="mt-4 rounded-2xl border border-slate-200 dark:border-zinc-800 bg-slate-50/50 dark:bg-zinc-900/50 p-4 shadow-sm">
              <div className="flex flex-wrap items-center gap-2">
                <button
                  type="button"
                  disabled={actionBusy || message.verificationStatus === "VERIFIED"}
                  onClick={() => void onVerify(message.id)}
                  className="rounded-lg border border-emerald-300 bg-emerald-50 px-3 py-1.5 text-xs font-bold text-emerald-700 hover:bg-emerald-100 transition-colors disabled:opacity-50"
                >
                  Verify
                </button>
                <button
                  type="button"
                  disabled={actionBusy}
                  onClick={() => setEditing((prev) => !prev)}
                  className="rounded-lg border border-blue-300 bg-blue-50 px-3 py-1.5 text-xs font-bold text-blue-700 hover:bg-blue-100 transition-colors disabled:opacity-50"
                >
                  Edit
                </button>
                <button
                  type="button"
                  disabled={actionBusy || message.verificationStatus === "REJECTED"}
                  onClick={() => void onReject(message.id)}
                  className="rounded-lg border border-rose-300 bg-rose-50 px-3 py-1.5 text-xs font-bold text-rose-700 hover:bg-rose-100 transition-colors disabled:opacity-50"
                >
                  Reject
                </button>
              </div>
              {editing ? (
                <div className="mt-3 space-y-2">
                  <textarea
                    value={draftCorrection}
                    onChange={(event) => setDraftCorrection(event.target.value)}
                    className="min-h-24 w-full rounded-xl border border-slate-200 dark:border-zinc-800 bg-white dark:bg-zinc-900 p-3 text-sm font-medium outline-none focus:ring-2 focus:ring-indigo-500/20 focus:border-indigo-500"
                  />
                  <div className="flex items-center gap-2">
                    <button
                      type="button"
                      disabled={actionBusy || !draftCorrection.trim()}
                      onClick={() => void onCorrect(message.id, draftCorrection)}
                      className="rounded-lg bg-indigo-600 px-4 py-1.5 text-xs font-bold text-white hover:bg-indigo-700 transition-colors shadow-lg shadow-indigo-500/20 disabled:opacity-50"
                    >
                      Save edit
                    </button>
                    <button
                      type="button"
                      disabled={actionBusy}
                      onClick={() => {
                        setDraftCorrection(message.content);
                        setEditing(false);
                      }}
                      className="rounded-lg border border-slate-200 px-4 py-1.5 text-xs font-bold text-slate-700 hover:bg-slate-100 transition-colors disabled:opacity-50"
                    >
                      Cancel
                    </button>
                  </div>
                </div>
              ) : null}
            </div>
          ) : null}

          {message.role === "AI" && message.attachedCitations.length > 0 ? (
            <div className="mt-4 flex flex-wrap gap-2">
              {message.attachedCitations.map((citation) => (
                <span
                  key={citation.id}
                  className="inline-flex items-center gap-1.5 rounded-full border border-slate-200 dark:border-zinc-800 bg-white dark:bg-zinc-900 px-3 py-1.5 text-xs font-bold text-slate-700 dark:text-slate-300 shadow-sm hover:border-indigo-300 transition-colors cursor-pointer"
                >
                  <BookOpen className="h-3.5 w-3.5 text-indigo-500" />
                  {citation.title}
                  <span className="ml-1 text-[10px] text-slate-400 uppercase">p. {citation.pageNumber}</span>
                </span>
              ))}
            </div>
          ) : null}

          <div className="mt-4 flex items-center gap-4">
            <button
              type="button"
              className="inline-flex items-center gap-1.5 text-[10px] font-bold uppercase tracking-widest text-slate-500 hover:text-indigo-600 transition-colors"
            >
              <ThumbsUp className="h-3.5 w-3.5" />
              {message.likes} Likes
            </button>
            <button
              type="button"
              className="inline-flex items-center gap-1.5 text-[10px] font-bold uppercase tracking-widest text-slate-500 hover:text-indigo-600 transition-colors"
            >
              <MessageSquare className="h-3.5 w-3.5" />
              Reply
            </button>
          </div>
        </div>
      </div>
    </article>
  );
}

function AiTypingBlock() {
  return (
    <div className="border-b border-l-4 border-indigo-500 bg-white/40 dark:bg-indigo-950/5 px-6 py-6 animate-in fade-in slide-in-from-bottom-2 duration-300">
      <div className="flex items-start gap-4">
        <AvatarChip initials="AI" role="AI" />
        <div className="min-w-0 flex-1">
          <div className="flex items-center gap-2 mb-2">
            <p className="text-sm font-bold text-slate-900 dark:text-slate-100">EduBot</p>
            <span className="rounded-full bg-slate-100 dark:bg-zinc-800 px-2 py-0.5 text-[10px] font-bold uppercase tracking-widest text-slate-600 dark:text-slate-400">Bot</span>
            <span className="inline-flex items-center gap-1.5 text-[10px] font-bold uppercase tracking-widest text-indigo-700 dark:text-indigo-400">
              <Loader2 className="h-3.5 w-3.5 animate-spin" />
              AI is thinking...
            </span>
          </div>
          <div className="mt-4 space-y-3">
            <div className="h-2 w-full max-w-[480px] rounded-full bg-slate-200 dark:bg-slate-700 animate-pulse" />
            <div className="h-2 w-full max-w-[340px] rounded-full bg-slate-200 dark:bg-slate-700 animate-pulse" style={{ animationDelay: '0.2s' }} />
            <div className="h-2 w-full max-w-[200px] rounded-full bg-slate-200 dark:bg-slate-700 animate-pulse" style={{ animationDelay: '0.4s' }} />
          </div>
        </div>
      </div>
    </div>
  );
}

function ThreadSkeleton() {
  return (
    <div className="flex flex-col gap-8 px-6 py-8 animate-in fade-in duration-700">
      {[1, 2, 3].map((i) => (
        <div key={i} className={`flex items-start gap-4 ${i === 2 ? 'opacity-70' : i === 3 ? 'opacity-40' : 'opacity-100'}`}>
          <div className="h-9 w-9 shrink-0 rounded-full bg-slate-200 dark:bg-slate-800 animate-pulse" />
          <div className="flex-1 space-y-4">
            <div className="flex items-center gap-2">
              <div className="h-4 w-32 rounded bg-slate-200 dark:bg-slate-800 animate-pulse" />
              <div className="h-3 w-16 rounded-full bg-slate-100 dark:bg-slate-800/50 animate-pulse" />
            </div>
            <div className="space-y-2">
              <div className="h-3 w-full max-w-[85%] rounded bg-slate-100 dark:bg-slate-800/50 animate-pulse" />
              <div className="h-3 w-full max-w-[75%] rounded bg-slate-100 dark:bg-slate-800/50 animate-pulse" />
              {i === 1 && <div className="h-3 w-full max-w-[60%] rounded bg-slate-100 dark:bg-slate-800/50 animate-pulse" />}
            </div>
          </div>
        </div>
      ))}
    </div>
  );
}

function ThreadHeaderSkeleton() {
  return (
    <header className="border-b border-slate-200 bg-white/70 backdrop-blur-md px-6 py-4 animate-in fade-in duration-500">
      <div className="h-3 w-48 rounded bg-slate-200 dark:bg-slate-800 animate-pulse" />
      <div className="mt-4 flex items-start justify-between gap-3">
        <div className="h-8 w-3/4 max-w-2xl rounded bg-slate-200 dark:bg-slate-800 animate-pulse" />
      </div>
      <div className="mt-4 flex flex-wrap gap-2">
        <div className="h-6 w-16 rounded-full bg-slate-100 dark:bg-slate-800 animate-pulse" />
        <div className="h-6 w-20 rounded-full bg-slate-100 dark:bg-slate-800 animate-pulse" />
        <div className="h-6 w-14 rounded-full bg-slate-100 dark:bg-slate-800 animate-pulse" />
      </div>
    </header>
  );
}

function ThreadDetailsSkeleton() {
  return (
    <section className="mt-6 border-t border-slate-200 pt-5 animate-in fade-in duration-500">
      <div className="mb-4 h-3 w-32 rounded bg-slate-200 dark:bg-slate-800 animate-pulse" />
      <div className="space-y-4">
        <div className="h-3 w-24 rounded bg-slate-100 dark:bg-slate-800 animate-pulse" />
        <div className="h-3 w-20 rounded bg-slate-100 dark:bg-slate-800 animate-pulse" />
        <div className="h-3 w-28 rounded bg-slate-100 dark:bg-slate-800 animate-pulse" />
      </div>
    </section>
  );
}


function ComposerBox({
  draft,
  onChange,
  onSend,
  onStop,
  sending,
  isStreaming,
  isPii,
  isRateLimited,
  onSwitchToPrivate,
  isScrolled,
}: {
  draft: string;
  onChange: (next: string) => void;
  onSend: () => void;
  onStop: () => void;
  sending: boolean;
  isStreaming: boolean;
  isPii: boolean;
  isRateLimited: boolean;
  onSwitchToPrivate: () => void;
  isScrolled?: boolean;
}) {
  const [isFocused, setIsFocused] = useState(false);
  const isCompact = isScrolled && !isFocused && !draft && !isStreaming;

  return (
    <div className={`border-t border-slate-200 bg-white/90 backdrop-blur-md transition-all duration-300 ${isCompact ? "px-4 py-2" : "px-6 py-4"}`}>
      {isPii && (
        <div className="mb-3 flex items-center justify-between rounded-lg border border-red-200 bg-red-50 p-3 text-sm text-red-700 shadow-sm animate-in fade-in slide-in-from-top-1">
          <div className="flex items-center gap-2">
            <AlertCircle className="h-4 w-4 shrink-0" />
            <p>
              <span className="font-bold">⚠️ Security Block:</span> Content contains sensitive PII information. We recommend discussing privately.
            </p>
          </div>
          <button
            type="button"
            onClick={onSwitchToPrivate}
            className="shrink-0 rounded-md bg-red-600 px-3 py-1 text-xs font-semibold text-white hover:bg-red-700 active:scale-95 transition-all"
          >
            Switch to Private Chat
          </button>
        </div>
      )}
      <div className={`rounded-xl border transition-all duration-300 glass-panel focus-within:ring-2 focus-within:ring-indigo-500/50 focus-within:border-indigo-500 ${isPii ? 'border-red-300 ring-2 ring-red-100' : 'border-slate-300'} ${isCompact ? "px-3 py-2" : "px-4 py-3"}`}>
        <div className="flex items-start gap-2 text-sm text-slate-500">
          <Paperclip className={`transition-all duration-300 ${isCompact ? "h-3.5 w-3.5 mt-0.5 opacity-70" : "h-4 w-4 mt-0.5"}`} />
          <textarea
            value={draft}
            disabled={sending || isRateLimited}
            onChange={(e) => onChange(e.target.value)}
            onFocus={() => setIsFocused(true)}
            onBlur={() => setIsFocused(false)}
            placeholder={isRateLimited ? "System is busy, please wait..." : isCompact ? "Reply..." : "Reply to this thread... Use @AI for direct assistance or @TA to flag for review."}
            rows={isCompact ? 1 : 2}
            className={`w-full resize-none bg-transparent text-sm text-slate-700 placeholder:text-slate-500 outline-none disabled:cursor-not-allowed disabled:opacity-50 transition-all duration-300 ${isCompact ? "min-h-[20px]" : "min-h-[44px]"}`}
          />
        </div>
        <div className={`overflow-hidden transition-all duration-300 flex items-center justify-between gap-3 ${isCompact ? "max-h-0 opacity-0 mt-0" : "max-h-12 opacity-100 mt-4"}`}>
          <div className="flex items-center gap-3 text-slate-500">
            <button type="button" className="rounded p-1 hover:bg-slate-100">
              <Smile className="h-4 w-4" />
            </button>
            <button type="button" className="rounded p-1 hover:bg-slate-100">
              <AtSign className="h-4 w-4" />
            </button>
          </div>
          {isStreaming ? (
            <button
              type="button"
              onClick={onStop}
              className="inline-flex items-center gap-2 rounded-lg bg-red-500 px-4 py-2 text-sm font-semibold text-white hover:bg-red-600"
            >
              <Square className="h-4 w-4 fill-current" />
              Stop
            </button>
          ) : (
            <button
              type="button"
              onClick={onSend}
              disabled={sending || !draft.trim() || isPii || isRateLimited}
              className={`inline-flex items-center gap-2 rounded-lg px-4 py-2 text-sm font-semibold text-white transition-all duration-200 ${
                isPii || isRateLimited
                  ? "bg-slate-400 cursor-not-allowed opacity-50" 
                  : "bg-indigo-600 hover:bg-indigo-700 active:scale-95 disabled:hover:scale-100"
              }`}
            >
              <Send className="h-4 w-4" />
              {sending ? "Sending..." : "Send"}
            </button>
          )}
        </div>
      </div>
      <div className={`overflow-hidden transition-all duration-300 ${isCompact ? "max-h-0 opacity-0 mt-0" : "max-h-6 opacity-100 mt-2"}`}>
        <p className="text-center text-[11px] text-slate-400">
          Press <span className="font-semibold text-slate-500">Shift + Enter</span> for new line
        </p>
      </div>
    </div>
  );
}

function RelevantDocumentsPanel({ docs }: { docs: RAGDocument[] }) {
  return (
    <section>
      <h3 className="mb-3 text-xs font-semibold uppercase tracking-wide text-slate-500">Relevant Documents</h3>
      {docs.length === 0 ? (
        <p className="rounded-xl border border-dashed border-slate-200 bg-white px-3 py-4 text-xs leading-5 text-slate-500">
          No citation metadata yet. Relevant documents will appear after AI responds with sources.
        </p>
      ) : (
        <div className="space-y-2">
          {docs.map((doc) => (
            <article key={doc.id} className="rounded-xl border border-slate-200 bg-white/70 backdrop-blur-sm p-3 hover:-translate-y-0.5 transition-transform duration-200 hover:shadow-md">
              <div className="flex items-start justify-between gap-2">
                <p className="text-sm font-semibold text-slate-900">{doc.title}</p>
                <span className="rounded-full bg-slate-100 px-2 py-0.5 text-[10px] font-semibold text-slate-600">
                  p. {doc.pageNumber}
                </span>
              </div>
              {doc.snippet ? <p className="mt-1 text-xs italic leading-5 text-slate-500">{doc.snippet}</p> : null}
              <button type="button" className="mt-2 inline-flex items-center gap-1 text-xs font-medium text-slate-500">
                <ExternalLink className="h-3.5 w-3.5" />
                Open source
              </button>
            </article>
          ))}
        </div>
      )}
    </section>
  );
}

function SimilarQuestionsPanel({ items }: { items: string[] }) {
  return (
    <section className="mt-6 border-t border-slate-200 pt-5">
      <h3 className="mb-3 text-xs font-semibold uppercase tracking-wide text-slate-500">Similar Questions</h3>
      <ul className="space-y-2">
        {items.map((item) => (
          <li key={item} className="text-xs leading-5 text-blue-700 hover:text-blue-800">
            • {item}
          </li>
        ))}
      </ul>
    </section>
  );
}

function ThreadDetailsPanel({ info }: { info: ThreadInfo }) {
  return (
    <section className="mt-6 border-t border-slate-200 pt-5">
      <h3 className="mb-3 text-xs font-semibold uppercase tracking-wide text-slate-500">Thread Details</h3>
      <ul className="space-y-2 text-xs text-slate-600">
        <li className="inline-flex items-center gap-2">
          <Clock3 className="h-3.5 w-3.5 text-slate-400" />
          {info.createdAgo}
        </li>
        <li className="inline-flex items-center gap-2">
          <Eye className="h-3.5 w-3.5 text-slate-400" />
          {info.views}
        </li>
        <li className="inline-flex items-center gap-2">
          <Users className="h-3.5 w-3.5 text-slate-400" />
          {info.studentCount}
        </li>
      </ul>
    </section>
  );
}

export default function ThreadDetailPage() {
  const params = useParams<{ id: string }>();
  const searchParams = useSearchParams();
  const threadId = String(params?.id ?? "").trim();
  const [threadInfo, setThreadInfo] = useState<ThreadInfo>(MOCK_THREAD_INFO);
  const [messages, setMessages] = useState<Message[]>([]);
  const [ragDocs, setRagDocs] = useState<RAGDocument[]>([]);
  const [draft, setDraft] = useState("");
  const [sending, setSending] = useState(false);
  const [isTyping, setIsTyping] = useState(false);
  const [isScrolled, setIsScrolled] = useState(false);

  const [loadError, setLoadError] = useState<string | null>(null);
  const [messagesLoaded, setMessagesLoaded] = useState(false);
  const [moderationBusy, setModerationBusy] = useState(false);
  const abortControllerRef = useRef<AbortController | null>(null);
  const autoTriggeredRef = useRef(false);
  const scrollViewportRef = useRef<HTMLDivElement | null>(null);

  const router = useRouter();
  const role = useAuthStore((state) => state.role);
  const { setPendingPrivateText } = usePrivacyStore();
  const { isPii } = usePiiDebounce(draft);
  const canModerate = role === "TA" || role === "ADMIN";

  const handleSwitchToPrivate = () => {
    setPendingPrivateText(draft);
    setDraft("");
    router.push("/chat");
  };

  const handleScroll = useCallback((e: React.UIEvent<HTMLDivElement>) => {
    const { scrollTop, scrollHeight, clientHeight } = e.currentTarget;
    setIsScrolled((prev) => {
      if (prev) {
        return scrollTop > 10;
      } else {
        return scrollTop > 50 && scrollHeight - clientHeight > 200;
      }
    });
  }, []);

  const loadThreadData = useCallback(async () => {
    if (!threadId) return;
    setMessagesLoaded(false);
    setLoadError(null);
    const [listResult, messagesResult] = await Promise.allSettled([
      threadsService.listThreads({ page: 1 }),
      threadsService.getMessages(threadId),
    ]);

    if (messagesResult.status === "rejected") {
      throw messagesResult.reason;
    }
    const messageRows = messagesResult.value;

    if (listResult.status === "fulfilled") {
      const matched = listResult.value.items.find((x) => x.id === threadId);
      if (matched) {
        setThreadInfo((prev) => ({
          ...prev,
          title: matched.title || prev.title,
          tags: matched.tags.length > 0 ? matched.tags : prev.tags,
          aiAnswered: messageRows.some((m) => toSenderRole(m.author?.role) === "AI"),
          taVerified: messageRows.some((m) => m.verification_status === "VERIFIED"),
        }));
      }
    } else {
      console.warn("Thread list fetch failed while loading detail metadata:", listResult.reason);
    }

    if (messageRows.length > 0) {
      const mapped = messageRows.map(mapWirePostToMessage);
      setMessages((prev) => {
        const locals = prev.filter((m) => m.id.startsWith("local-"));
        return [...mapped, ...locals];
      });
      const docPool = mapped.flatMap((m) => m.attachedCitations);
      setRagDocs(docPool.length > 0 ? mergeRagDocs([], docPool) : []);
    } else {
      setMessages((prev) => prev.filter((m) => m.id.startsWith("local-")));
      setRagDocs([]);
    }
    setMessagesLoaded(true);
  }, [threadId]);

  useEffect(() => {
    let cancelled = false;
    const load = async () => {
      try {
        await loadThreadData();
      } catch (error: any) {
        if (cancelled) return;
        const errMsg = error?.message?.toLowerCase() || "";
        const errName = error?.name || "";
        if (errName === "CanceledError" || errName === "AbortError" || errMsg.includes("aborted") || errMsg.includes("canceled")) {
          return;
        }
        setLoadError(extractErrorMessage(error, "Failed to load thread detail."));
      }
    };
    void load();
    return () => {
      cancelled = true;
    };
  }, [loadThreadData]);

  useEffect(() => {
    if (!scrollViewportRef.current) return;
    const viewport = scrollViewportRef.current;
    viewport.scrollTo({
      top: viewport.scrollHeight,
      behavior: isTyping ? "auto" : "smooth",
    });
  }, [messages, isTyping]);

  useEffect(() => {
    return () => {
      abortControllerRef.current?.abort();
      abortControllerRef.current = null;
    };
  }, []);

  const showAiTyping = SHOW_AI_TYPING && isTyping;

  const handleStop = () => {
    abortControllerRef.current?.abort();
    abortControllerRef.current = null;
    setIsTyping(false);
    setSending(false);
  };

  const startAiResponse = useCallback(
    async (content: string, options: { autoTriggered?: boolean } = {}) => {
      if (!threadId || !content.trim()) return;
      const aiMessageId = `local-ai-${Date.now()}`;
      const nowLabel = toTimeLabel(new Date().toISOString());
      setLoadError(null);
      setIsTyping(true);
      const controller = new AbortController();
      abortControllerRef.current = controller;

      await streamChat(
        threadId,
        content,
        {
          onChunk: (chunk) => {
            setIsTyping(false);
            setMessages((prev) => {
              const exists = prev.find((m) => m.id === aiMessageId);
              if (exists) {
                return prev.map((m) =>
                  m.id === aiMessageId ? { ...m, content: `${m.content}${chunk}` } : m
                );
              }
              return [
                ...prev,
                {
                  id: aiMessageId,
                  role: "AI",
                  authorName: "EduBot",
                  avatarUrl: "AI",
                  content: chunk,
                  timestamp: nowLabel,
                  likes: 0,
                  isAcceptedAnswer: false,
                  verificationStatus: "UNVERIFIED",
                  verifiedByName: null,
                  originalAiContent: null,
                  attachedCitations: [],
                },
              ];
            });
          },
          onFinish: (metadata: unknown) => {
            const mappedCitations = mapSseMetadataToCitations(metadata);
            if (mappedCitations.length > 0) {
              setRagDocs((ragPrev) => mergeRagDocs(ragPrev, mappedCitations));
            }

            setMessages((prev) => {
              const exists = prev.find((m) => m.id === aiMessageId);
              let nextMessages = prev;

              if (!exists) {
                nextMessages = [
                  ...prev,
                  {
                    id: aiMessageId,
                    role: "AI",
                    authorName: "EduBot",
                    avatarUrl: "AI",
                    content: "",
                    timestamp: nowLabel,
                    likes: 0,
                    isAcceptedAnswer: false,
                    verificationStatus: "UNVERIFIED",
                    verifiedByName: null,
                    originalAiContent: null,
                    attachedCitations: [],
                  },
                ];
              }

              if (mappedCitations.length > 0) {
                nextMessages = nextMessages.map((m) =>
                  m.id === aiMessageId ? { ...m, attachedCitations: mappedCitations } : m
                );
              }

              return nextMessages;
            });
            setIsTyping(false);
          },
          onError: (error) => {
            setIsTyping(false);
            setMessages((prev) => {
              const exists = prev.find((m) => m.id === aiMessageId);
              if (exists) {
                return prev.map((m) => (m.id === aiMessageId ? { ...m, content: error } : m));
              }
              return [
                ...prev,
                {
                  id: aiMessageId,
                  role: "AI",
                  authorName: "EduBot",
                  avatarUrl: "AI",
                  content: error,
                  timestamp: nowLabel,
                  likes: 0,
                  isAcceptedAnswer: false,
                  verificationStatus: "UNVERIFIED",
                  verifiedByName: null,
                  originalAiContent: null,
                  attachedCitations: [],
                },
              ];
            });
          },
          onRateLimit: (msg) => {
            setIsTyping(false);
            setSending(false);
            const errorMsg = msg || "Hệ thống đang quá tải.";
            setMessages((prev) => {
              const exists = prev.find((m) => m.id === aiMessageId);
              if (exists) {
                return prev.map((m) => (m.id === aiMessageId ? { ...m, content: errorMsg } : m));
              }
              return [
                ...prev,
                {
                  id: aiMessageId,
                  role: "AI",
                  authorName: "EduBot",
                  avatarUrl: "AI",
                  content: errorMsg,
                  timestamp: nowLabel,
                  likes: 0,
                  isAcceptedAnswer: false,
                  verificationStatus: "UNVERIFIED",
                  verifiedByName: null,
                  originalAiContent: null,
                  attachedCitations: [],
                },
              ];
            });
          },
          onSecurityBlock: () => {
            setIsTyping(false);
            setSending(false);
          },
        },
        controller.signal,
        { autoTriggered: options.autoTriggered === true },
      );
    },
    [threadId],
  );

  useEffect(() => {
    if (!threadId || !messagesLoaded || autoTriggeredRef.current) return;
    if (searchParams.get("auto_ai") !== "1") return;
    if (messages.some((message) => message.role === "AI")) return;

    const storageKey = `thread:${threadId}:autoAiQuestion`;
    const question = typeof window === "undefined" ? "" : window.sessionStorage.getItem(storageKey)?.trim() ?? "";
    if (!question) return;

    autoTriggeredRef.current = true;
    window.sessionStorage.removeItem(storageKey);
    void startAiResponse(question, { autoTriggered: true });
  }, [messages, messagesLoaded, searchParams, startAiResponse, threadId]);

  const handleSend = async () => {
    const content = draft.trim();
    if (!content || !threadId || sending) return;

    setSending(true);
    setLoadError(null);
    setDraft("");
    const nowLabel = toTimeLabel(new Date().toISOString());

    const optimisticMessage: Message = {
      id: `local-human-${Date.now()}`,
      role: "STUDENT",
      authorName: "You",
      avatarUrl: "YO",
      content,
      timestamp: nowLabel,
      likes: 0,
      isAcceptedAnswer: false,
      verificationStatus: "UNVERIFIED",
      verifiedByName: null,
      originalAiContent: null,
      attachedCitations: [],
    };
    setMessages((prev) => [...prev, optimisticMessage]);

    try {
      const persistedId = await threadsService.sendMessage(threadId, content);
      setMessages((prev) => prev.map((m) => (m.id === optimisticMessage.id ? { ...m, id: persistedId } : m)));

      if (/@AI\b/i.test(content)) {
        // Message already persisted via /threads/{id}/messages above.
        // Mark ask-ai as auto-triggered to avoid duplicate STUDENT inserts.
        await startAiResponse(content, { autoTriggered: true });
      }
    } catch (error) {
      setMessages((prev) => prev.filter((m) => m.id !== optimisticMessage.id));
      setLoadError(extractErrorMessage(error, "Failed to send message."));
    } finally {
      setIsTyping(false);
      setSending(false);
      abortControllerRef.current = null;
    }
  };

  const withModeration = useCallback(
    async (task: () => Promise<void>) => {
      setModerationBusy(true);
      setLoadError(null);
      try {
        await task();
        await loadThreadData();
      } catch (error) {
        setLoadError(extractErrorMessage(error, "Failed to update verification status."));
      } finally {
        setModerationBusy(false);
      }
    },
    [loadThreadData],
  );

  const handleVerify = useCallback(
    async (messageId: string) => {
      await withModeration(async () => {
        await threadsService.verifyMessage(messageId);
      });
    },
    [withModeration],
  );

  const handleReject = useCallback(
    async (messageId: string) => {
      await withModeration(async () => {
        await threadsService.rejectMessage(messageId);
      });
    },
    [withModeration],
  );

  const handleCorrect = useCallback(
    async (messageId: string, content: string) => {
      await withModeration(async () => {
        await threadsService.correctMessage(messageId, content.trim());
      });
    },
    [withModeration],
  );

  const mainFeed = useMemo(
    () => {
      if (!messagesLoaded) {
        return <ThreadSkeleton />;
      }
      return messages.length > 0 ? (
        messages.map((message) => (
          <MessageItem
            key={message.id}
            message={message}
            canModerate={canModerate}
            actionBusy={moderationBusy}
            onVerify={handleVerify}
            onReject={handleReject}
            onCorrect={handleCorrect}
          />
        ))
      ) : (
        <div className="border-b border-slate-200 bg-white px-6 py-10 text-sm text-slate-500 animate-in fade-in duration-500">No messages yet.</div>
      );
    },
    [canModerate, handleCorrect, handleReject, handleVerify, messages, moderationBusy, messagesLoaded],
  );

  return (
    <WorkspaceLayout footerLine2="operational" showNotifications={false} searchPlaceholder="Search in thread...">
      <div className="flex h-full bg-slate-50">
        <IconSidebar />

        <section className="flex min-w-0 flex-1 flex-col border-r border-slate-200 relative">
          <div className="z-20 shrink-0">
            {!messagesLoaded ? <ThreadHeaderSkeleton /> : <ThreadHeader info={threadInfo} isScrolled={isScrolled} />}
            {loadError ? (
              <div className="border-b border-red-200 bg-red-50 px-6 py-2 text-sm text-red-700">{loadError}</div>
            ) : null}
          </div>

          <div 
            ref={scrollViewportRef} 
            className="flex-1 overflow-y-auto pb-4 scroll-smooth"
            onScroll={handleScroll}
          >
            {mainFeed}
            {showAiTyping ? <AiTypingBlock /> : null}
          </div>
          
          <div className="sticky bottom-0 z-20">
            <ComposerBox
              draft={draft}
              onChange={setDraft}
              onSend={handleSend}
              onStop={handleStop}
              sending={sending}
              isStreaming={isTyping}
              isPii={isPii}
              isRateLimited={false}
              onSwitchToPrivate={handleSwitchToPrivate}
              isScrolled={isScrolled}
            />
          </div>
        </section>

        <aside className="hidden w-[320px] shrink-0 border-l border-slate-200 bg-slate-50 px-4 py-5 lg:block">
          <RelevantDocumentsPanel docs={ragDocs} />
          <SimilarQuestionsPanel items={MOCK_SIMILAR_QUESTIONS} />
          {!messagesLoaded ? <ThreadDetailsSkeleton /> : <ThreadDetailsPanel info={threadInfo} />}
        </aside>
      </div>
    </WorkspaceLayout>
  );
}
