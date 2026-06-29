import { useCallback, useEffect, useState } from "react";
import { extractErrorMessage } from "@/lib/utils";
import { chatService } from "@/services/chat.service";
import type { ChatMessageWire, ChatSessionWire } from "@/types/chat";
import { useUiStore } from "@/store/uiStore";

export function usePrivateChatSessions() {
  const [sessions, setSessions] = useState<ChatSessionWire[]>([]);
  const [activeSessionId, setActiveSessionId] = useState<string | null>(null);
  const [messagesBySession, setMessagesBySession] = useState<Record<string, ChatMessageWire[]>>({});
  const [sessionsLoading, setSessionsLoading] = useState(true);
  const [messagesLoading, setMessagesLoading] = useState(false);
  const [createLoading, setCreateLoading] = useState(false);
  const [sessionsError, setSessionsError] = useState<string | null>(null);
  const [messagesError, setMessagesError] = useState<string | null>(null);

  const loadSessions = useCallback(async () => {
    setSessionsLoading(true);
    setSessionsError(null);
    try {
      const data = await chatService.listSessions();
      setSessions(data);
      if (data.length > 0) {
        setActiveSessionId((prev) => prev ?? data[0].session_id);
      }
    } catch (error) {
      setSessions([]);
      setActiveSessionId(null);
      setSessionsError(extractErrorMessage(error, "Could not load chat sessions."));
    } finally {
      setSessionsLoading(false);
    }
  }, []);

  const loadMessages = useCallback(async (sessionId: string, force = false) => {
    if (!force && messagesBySession[sessionId]) {
      return;
    }
    setMessagesLoading(true);
    setMessagesError(null);
    try {
      const data = await chatService.getMessages(sessionId);
      setMessagesBySession((prev) => ({ ...prev, [sessionId]: data }));
    } catch (error) {
      setMessagesError(extractErrorMessage(error, "Could not load messages."));
    } finally {
      setMessagesLoading(false);
    }
  }, [messagesBySession]);

  const createSession = useCallback(async (): Promise<string | null> => {
    // Create a local draft session — no API call yet.
    // The real session is created when the user sends the first message.
    const draftId = `DRAFT_${Date.now()}`;
    const createdSession: ChatSessionWire = { session_id: draftId };
    setSessions((prev) => [createdSession, ...prev]);
    setActiveSessionId(draftId);
    setMessagesBySession((prev) => ({ ...prev, [draftId]: [] }));
    return draftId;
  }, []);

  /**
   * Ensures a real backend session exists.
   * If the active session is a draft, creates it on the backend
   * and migrates local state to the real session ID.
   * Returns the real session ID.
   */
  const ensureSession = useCallback(async (sessionId: string): Promise<string | null> => {
    if (!sessionId.startsWith("DRAFT_")) return sessionId;
    setCreateLoading(true);
    try {
      const realId = await chatService.createSession();
      // Migrate draft → real session in all state
      setSessions((prev) =>
        prev.map((s) => s.session_id === sessionId ? { ...s, session_id: realId } : s)
      );
      setMessagesBySession((prev) => {
        const { [sessionId]: msgs, ...rest } = prev;
        return { ...rest, [realId]: msgs ?? [] };
      });
      // Also migrate the persisted title if it exists
      const { sessionTitles, setSessionTitle } = useUiStore.getState();
      if (sessionTitles[sessionId]) {
        setSessionTitle(realId, sessionTitles[sessionId]);
      }
      setActiveSessionId(realId);
      return realId;
    } catch (error) {
      setSessionsError(extractErrorMessage(error, "Could not create a new conversation."));
      return null;
    } finally {
      setCreateLoading(false);
    }
  }, []);

  const deleteSession = useCallback(async (sessionId: string): Promise<boolean> => {
    setSessionsError(null);
    // If it's a draft, just remove locally
    if (sessionId.startsWith("DRAFT_")) {
      setSessions((prev) => prev.filter((s) => s.session_id !== sessionId));
      setMessagesBySession((prev) => {
        const { [sessionId]: _, ...rest } = prev;
        return rest;
      });
      if (activeSessionId === sessionId) {
        const remaining = sessions.filter((s) => s.session_id !== sessionId);
        setActiveSessionId(remaining.length > 0 ? remaining[0].session_id : null);
      }
      return true;
    }
    try {
      await chatService.deleteSession(sessionId);
      setSessions((prev) => prev.filter((s) => s.session_id !== sessionId));
      setMessagesBySession((prev) => {
        const { [sessionId]: _, ...rest } = prev;
        return rest;
      });
      if (activeSessionId === sessionId) {
        const remaining = sessions.filter((s) => s.session_id !== sessionId);
        setActiveSessionId(remaining.length > 0 ? remaining[0].session_id : null);
      }
      return true;
    } catch (error) {
      setSessionsError(extractErrorMessage(error, "Could not delete conversation."));
      return false;
    }
  }, [activeSessionId, sessions, setActiveSessionId]);

  const updateSessionTitle = useCallback((sessionId: string, title: string) => {
    // 1. Update local state
    setSessions((prev) =>
      prev.map((s) => s.session_id === sessionId ? { ...s, title } : s)
    );
    // 2. Persist to localStorage
    useUiStore.getState().setSessionTitle(sessionId, title);
  }, []);

  useEffect(() => {
    void loadSessions();
  }, [loadSessions]);

  // Auto-select first session on load (but don't auto-create)
  useEffect(() => {
    if (sessionsLoading) return;
    if (activeSessionId) return;
    if (sessions.length > 0) {
      setActiveSessionId(sessions[0].session_id);
    }
  }, [sessions, sessionsLoading, activeSessionId, setActiveSessionId]);

  useEffect(() => {
    if (!activeSessionId) return;
    // Don't try to load messages for draft sessions
    if (activeSessionId.startsWith("DRAFT_")) return;
    void loadMessages(activeSessionId);
  }, [activeSessionId, loadMessages]);

  return {
    sessions,
    activeSessionId,
    setActiveSessionId,
    messagesBySession,
    sessionsLoading,
    messagesLoading,
    createLoading,
    sessionsError,
    messagesError,
    loadSessions,
    loadMessages,
    createSession,
    ensureSession,
    deleteSession,
    updateSessionTitle,
    setMessagesBySession,
    setMessagesError,
  };
}