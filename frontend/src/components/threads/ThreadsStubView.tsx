import { BookOpen, Loader2, MessageSquare, Plus, Search, Sparkles } from "lucide-react";
import type { ThreadSidebarFilter } from "@/hooks/useThreadsList";
import type { ThreadListItemVm } from "@/services/threads.service";

const FILTER_OPTIONS: { key: ThreadSidebarFilter; label: string }[] = [
  { key: "ALL", label: "All" },
  { key: "UNANSWERED", label: "Unanswered" },
  { key: "ASSIGNMENT", label: "Assignment" },
  { key: "KNOWLEDGE", label: "Knowledge" },
];

function statusStyles(status: string): string {
  const s = status.toUpperCase();
  if (s === "ESCALATED") return "bg-rose-100 text-rose-700 ring-1 ring-rose-200 border-none animate-pulse";
  if (s === "OPEN" || s === "UNANSWERED") return "bg-amber-100 text-amber-700 ring-1 ring-amber-200 border-none";
  if (s === "RESOLVED") return "bg-emerald-100 text-emerald-700 ring-1 ring-emerald-200 border-none";
  return "bg-slate-100 text-slate-700 dark:bg-zinc-800 dark:text-slate-300";
}

export interface ThreadsStubViewProps {
  threads: ThreadListItemVm[];
  total: number;
  loading: boolean;
  error: string | null;
  onRetry: () => void | Promise<void>;
  searchQuery: string;
  onSearchChange: (value: string) => void;
  activeFilter: ThreadSidebarFilter;
  onFilterChange: (filter: ThreadSidebarFilter) => void;
  selectedThreadId: string | null;
  onSelectThread: (id: string) => void;
  onOpenCreate: () => void;
}

/**
 * Threads workspace: list from {@code GET /api/v1/threads} + empty detail pane until thread UI ships.
 */
export function ThreadsStubView({
  threads,
  total,
  loading,
  error,
  onRetry,
  searchQuery,
  onSearchChange,
  activeFilter,
  onFilterChange,
  selectedThreadId,
  onSelectThread,
  onOpenCreate,
}: ThreadsStubViewProps) {
  return (
    <div className="flex min-h-[calc(100vh-4rem)] lg:h-[calc(100vh-4rem)] flex-col border-t border-slate-200 bg-white lg:flex-row lg:overflow-hidden">
      <aside className="flex w-full shrink-0 flex-col border-slate-200 dark:border-zinc-800 lg:w-[350px] lg:border-r bg-white/40 dark:bg-zinc-950/40 backdrop-blur-md z-10">
        <div className="space-y-4 border-b border-slate-200/60 dark:border-zinc-800/60 p-5">
          <div className="flex items-center justify-between gap-2">
            <h1 className="text-xl font-bold tracking-tight text-slate-900 dark:text-slate-100">Discussions</h1>
            <button
              type="button"
              onClick={onOpenCreate}
              className="inline-flex shrink-0 items-center gap-1.5 rounded-xl bg-indigo-600 px-3 py-2 text-xs font-bold text-white shadow-md shadow-indigo-500/20 transition-all hover:bg-indigo-700 hover:shadow-lg hover:-translate-y-0.5 active:scale-95"
            >
              <Plus className="h-4 w-4" />
              New
            </button>
          </div>
          <label className="relative block group">
            <Search className="pointer-events-none absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-slate-400 transition-colors group-focus-within:text-indigo-500" />
            <input
              type="search"
              value={searchQuery}
              onChange={(e) => onSearchChange(e.target.value)}
              placeholder="Search discussions..."
              className="w-full rounded-xl border border-slate-200 dark:border-zinc-800 bg-white/70 dark:bg-zinc-900/70 py-2.5 pl-9 pr-3 text-sm text-slate-700 dark:text-slate-200 placeholder:text-slate-400 outline-none backdrop-blur-sm transition-all focus:border-indigo-500 focus:bg-white dark:focus:bg-zinc-900 focus:ring-4 focus:ring-indigo-500/10"
              aria-label="Search threads by title"
            />
          </label>
          <div className="flex flex-wrap gap-2">
            {FILTER_OPTIONS.map(({ key, label }) => (
              <button
                key={key}
                type="button"
                onClick={() => onFilterChange(key)}
                className={`rounded-full px-3.5 py-1.5 text-[11px] font-bold uppercase tracking-wider transition-all duration-300 ${
                  activeFilter === key
                    ? "bg-indigo-600 text-white shadow-md shadow-indigo-500/20 scale-105"
                    : "border border-slate-200 dark:border-zinc-800 bg-white/50 dark:bg-zinc-900/50 text-slate-600 dark:text-slate-400 hover:border-indigo-300 dark:hover:border-indigo-700/50 hover:bg-indigo-50 dark:hover:bg-indigo-900/20 hover:text-indigo-700 dark:hover:text-indigo-300"
                }`}
              >
                {label}
              </button>
            ))}
          </div>
        </div>

        {error ? (
          <div className="mx-3 mt-3 rounded-lg border border-red-200 bg-red-50 px-3 py-2 text-sm text-red-800">
            <p>{error}</p>
            <button
              type="button"
              className="mt-2 text-xs font-semibold text-red-700 underline hover:text-red-900"
              onClick={() => void onRetry()}
            >
              Retry
            </button>
          </div>
        ) : null}

        <ul className="flex-1 space-y-3 overflow-y-auto overflow-x-hidden p-4 scroll-smooth" aria-busy={loading}>
          {loading && threads.length === 0 ? (
            <li className="flex flex-col items-center justify-center gap-3 py-16 text-sm font-medium text-slate-500">
              <Loader2 className="h-6 w-6 animate-spin text-indigo-500" aria-hidden />
              Đang tải danh sách...
            </li>
          ) : null}
          {!loading && threads.length === 0 && !error ? (
            <li className="py-16 text-center text-sm font-medium text-slate-500">Không tìm thấy chủ đề nào.</li>
          ) : null}
          {threads.map((t, index) => {
            const selected = selectedThreadId === t.id;
            return (
              <li key={t.id} className="animate-in fade-in slide-in-from-bottom-2 fill-mode-both" style={{ animationDelay: `${Math.min(index * 50, 500)}ms` }}>
                <button
                  type="button"
                  onClick={() => onSelectThread(t.id)}
                  aria-current={selected ? "true" : undefined}
                  className={`group relative w-full rounded-2xl border p-4 text-left transition-all duration-300 ${
                    selected
                      ? "border-indigo-500/50 bg-indigo-50/50 dark:bg-indigo-900/20 ring-4 ring-indigo-500/10 shadow-sm"
                      : "border-slate-200 dark:border-zinc-800 bg-white/50 dark:bg-zinc-900/50 hover:border-indigo-300/50 dark:hover:border-indigo-700/50 hover:bg-white dark:hover:bg-zinc-900 hover:shadow-md hover:-translate-y-0.5"
                  }`}
                >
                  <div className="flex items-start justify-between gap-3">
                    <p className={`text-sm font-bold leading-tight ${selected ? 'text-indigo-900 dark:text-indigo-100' : 'text-slate-900 dark:text-slate-100 group-hover:text-indigo-600 dark:group-hover:text-indigo-400 transition-colors'}`}>{t.title}</p>
                    <span className="shrink-0 text-[10px] font-bold uppercase tracking-wider text-slate-400">{t.relativeTime}</span>
                  </div>
                  <p className="mt-2 line-clamp-2 text-xs leading-relaxed text-slate-600 dark:text-slate-400">{t.snippet}</p>
                  <div className="mt-3 flex flex-wrap items-center gap-2 text-[10px] font-bold uppercase tracking-wider text-slate-500">
                    <span className={`rounded-md px-2 py-1 text-[10px] font-extrabold ${statusStyles(t.status)}`}>
                      {t.status}
                    </span>
                    {t.tags.slice(0, 2).map((tag) => (
                      <span key={tag} className="rounded-full border border-slate-200 dark:border-zinc-700 bg-white/50 dark:bg-zinc-900/50 px-2.5 py-1 text-slate-600 dark:text-slate-400">
                        {tag}
                      </span>
                    ))}
                    <span className="ml-auto flex items-center gap-1.5 text-slate-400 group-hover:text-indigo-500 transition-colors">
                      <MessageSquare className="h-3.5 w-3.5" />
                      {t.replyCount}
                    </span>
                  </div>
                </button>
              </li>
            );
          })}
        </ul>
        <div className="border-t border-slate-200/60 dark:border-zinc-800/60 px-5 py-3 text-[10px] font-bold uppercase tracking-widest text-slate-400 bg-slate-50/50 dark:bg-zinc-900/50 backdrop-blur-sm">
          {total} thread{total === 1 ? "" : "s"} · API
        </div>
      </aside>

      <div className="flex flex-1 flex-col bg-slate-50/80">
        <div className="flex flex-1 flex-col items-center justify-center px-6 py-10">
          {loading && threads.length > 0 ? (
            <Loader2 className="mb-4 h-8 w-8 animate-spin text-blue-600" aria-label="Refreshing" />
          ) : null}
          <div className="relative mb-6 rounded-2xl border-2 border-dashed border-slate-300 bg-white p-8">
            <MessageSquare className="h-12 w-12 text-slate-400" aria-hidden />
            <span className="absolute -right-1 -top-1 rounded-md bg-blue-600 p-1 text-white shadow-sm">
              <Sparkles className="h-3.5 w-3.5" aria-hidden />
            </span>
          </div>
          <h2 className="text-center text-xl font-bold tracking-tight text-slate-900 sm:text-2xl">
            {selectedThreadId ? "Thread detail" : "Select a thread to start learning"}
          </h2>
          <p className="mt-2 max-w-md text-center text-sm text-slate-600">
            {selectedThreadId
              ? "Message history and AI replies will appear here in a future iteration. List data is already loaded from the API."
              : "Join ongoing discussions, browse lecture-specific questions, or get instant support from our AI Teaching Assistant."}
          </p>
          <div className="mt-8 grid w-full max-w-lg grid-cols-1 gap-3 sm:grid-cols-2">
            <button
              type="button"
              onClick={onOpenCreate}
              className="flex flex-col items-start gap-2 rounded-xl border border-slate-200 bg-white p-4 text-left shadow-sm transition hover:border-blue-200 hover:ring-1 hover:ring-blue-100"
            >
              <span className="rounded-lg bg-blue-50 p-2 text-blue-600">
                <Plus className="h-5 w-5" />
              </span>
              <span className="font-semibold text-slate-900">Ask a Question</span>
              <span className="text-xs text-slate-600">Opens the new-thread form (POST /api/v1/threads).</span>
            </button>
            <a
              href="/documents"
              className="flex flex-col items-start gap-2 rounded-xl border border-slate-200 bg-white p-4 text-left shadow-sm transition hover:border-violet-200 hover:ring-1 hover:ring-violet-100"
            >
              <span className="rounded-lg bg-violet-50 p-2 text-violet-600">
                <BookOpen className="h-5 w-5" />
              </span>
              <span className="font-semibold text-slate-900">Knowledge Base</span>
              <span className="text-xs text-slate-600">Open course documents for context.</span>
            </a>
          </div>
          <div className="mt-10 flex flex-wrap items-center justify-center gap-4 text-xs font-semibold uppercase tracking-wide text-slate-500">
            <span>Live list</span>
            <span className="hidden h-3 w-px bg-slate-300 sm:block" aria-hidden />
            <span>{total} total</span>
            <span className="hidden h-3 w-px bg-slate-300 sm:block" aria-hidden />
            <span>GET /threads</span>
          </div>
        </div>
        <div className="flex flex-col gap-1 border-t border-slate-200 bg-white px-4 py-2 text-xs text-slate-500 sm:flex-row sm:items-center sm:justify-between">
          <span className="font-mono text-[11px] text-slate-400">Ctrl + K to search</span>
          <span className="text-slate-400">EduThread · API</span>
        </div>
      </div>
    </div>
  );
}
