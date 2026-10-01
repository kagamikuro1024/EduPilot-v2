import { useCallback, useEffect, useRef, useState } from "react";
import { extractErrorMessage } from "@/lib/utils";
import { threadsService, type ThreadListItemVm } from "@/services/threads.service";
import type { CreateThreadRequestBody } from "@/types/threads";
import { hasSensitiveSignals } from "@/lib/privacySignals";
import { intentService } from "@/services/intent.service";
const PRIVACY_HINT_MESSAGE =
  "Noi dung co dau hieu thong tin rieng tu (MSSV/diem/du lieu ca nhan). Nen gui qua kenh chat private.";
const PUBLIC_BLOCK_MESSAGE =
  "Khong the dang thread cong khai vi noi dung co nguy co lo thong tin rieng tu. Vui long dung chat private.";
const CLASSIFY_MIN_CHARS = 12;
const CLASSIFY_MIN_CHARS_FOR_NON_SENSITIVE = 32;


export type ThreadSidebarFilter = "ALL" | "UNANSWERED" | "ASSIGNMENT" | "KNOWLEDGE";

function queryForFilter(filter: ThreadSidebarFilter): { status?: string; tag?: string } {
  switch (filter) {
    case "UNANSWERED":
      return { status: "OPEN" };
    case "ASSIGNMENT":
      return { tag: "Assignment" };
    case "KNOWLEDGE":
      return { tag: "Knowledge" };
    default:
      return {};
  }
}

export function useThreadsList() {
  const [threads, setThreads] = useState<ThreadListItemVm[]>([]);
  const [total, setTotal] = useState(0);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [searchInput, setSearchInput] = useState("");
  const [debouncedSearch, setDebouncedSearch] = useState("");
  const [filter, setFilter] = useState<ThreadSidebarFilter>("ALL");
  const [selectedThreadId, setSelectedThreadId] = useState<string | null>(null);
  const [createOpen, setCreateOpenInternal] = useState(false);
  const [createSubmitting, setCreateSubmitting] = useState(false);
  const [createError, setCreateError] = useState<string | null>(null);
  const [createPrivacyHint, setCreatePrivacyHint] = useState<string | null>(null);
  const lastAnalyzedDraftRef = useRef("");

  const setCreateOpen = useCallback((open: boolean) => {
    if (open) {
      setCreateError(null);
      setCreatePrivacyHint(null);
    }
    setCreateOpenInternal(open);
  }, []);

  useEffect(() => {
    const t = window.setTimeout(() => setDebouncedSearch(searchInput.trim()), 350);
    return () => window.clearTimeout(t);
  }, [searchInput]);

  const load = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      const q = queryForFilter(filter);
      const { items, total: t } = await threadsService.listThreads({
        ...q,
        search: debouncedSearch || undefined,
        page: 1,
      });
      setThreads(items);
      setTotal(t);
    } catch (e) {
      setError(extractErrorMessage(e, "Could not load threads."));
      setThreads([]);
      setTotal(0);
    } finally {
      setLoading(false);
    }
  }, [filter, debouncedSearch]);

  useEffect(() => {
    void load();
  }, [load]);

  const analyzeCreateDraft = useCallback(async (title: string, content: string) => {
    const merged = `${title}\n${content}`.trim();
    if (!merged) {
      lastAnalyzedDraftRef.current = "";
      setCreatePrivacyHint(null);
      return;
    }
    if (merged.length < CLASSIFY_MIN_CHARS) {
      setCreatePrivacyHint(null);
      return;
    }
    if (lastAnalyzedDraftRef.current === merged) {
      return;
    }

    const localSensitive = hasSensitiveSignals(merged);
    const shouldCallClassifier = localSensitive || merged.length >= CLASSIFY_MIN_CHARS_FOR_NON_SENSITIVE;
    if (!shouldCallClassifier) {
      setCreatePrivacyHint(null);
      return;
    }

    lastAnalyzedDraftRef.current = merged;
    try {
      const res = await intentService.classify(merged, "PUBLIC");
      if (localSensitive || res.suggestedChannel === "PRIVATE") {
        // Do not leak technical classifier fallback reasons to users.
        setCreatePrivacyHint(localSensitive ? PRIVACY_HINT_MESSAGE : "Noi dung nay phu hop hon voi kenh private.");
      } else {
        setCreatePrivacyHint(null);
      }
    } catch {
      setCreatePrivacyHint(localSensitive ? PRIVACY_HINT_MESSAGE : null);
    }
  }, []);

  const submitCreate = async (body: CreateThreadRequestBody): Promise<string | null> => {
    setCreateSubmitting(true);
    setCreateError(null);
    try {
      const merged = `${body.title}\n${body.content}`.trim();
      const localSensitive = hasSensitiveSignals(merged);
      let classifyPrivate = false;
      try {
        const res = await intentService.classify(merged, "PUBLIC");
        classifyPrivate = res.suggestedChannel === "PRIVATE";
      } catch {
        classifyPrivate = false;
      }
      if (localSensitive || classifyPrivate) {
        setCreateError(localSensitive ? PUBLIC_BLOCK_MESSAGE : "Noi dung nay chi duoc phep gui qua kenh private.");
        return null;
      }

      const id = await threadsService.createThread(body);
      setCreateOpenInternal(false);
      setSelectedThreadId(id);
      await load();
      return id;
    } catch (e) {
      setCreateError(extractErrorMessage(e, "Could not create thread."));
      return null;
    } finally {
      setCreateSubmitting(false);
    }
  };

  return {
    threads,
    total,
    loading,
    error,
    reload: load,
    searchInput,
    setSearchInput,
    filter,
    setFilter,
    selectedThreadId,
    setSelectedThreadId,
    createOpen,
    setCreateOpen,
    createSubmitting,
    createError,
    createPrivacyHint,
    analyzeCreateDraft,
    submitCreate,
  };
}
