import { useAuthStore } from "@/store/authStore";
import { toast } from "sonner";

/**
 * ARCHITECTURAL BOUNDARY FIX:
 *
 * BEFORE (VIOLATION):
 *   Frontend → Python AI Service (direct call — bypasses Java Gateway)
 *
 * AFTER (CORRECT):
 *   Frontend → Java Spring Boot → Python AI Service (via gRPC)
 *
 * The Java Gateway enforces:
 *   1. JWT Authentication
 *   2. Rate Limiting (15 req/student/day)
 *   3. DB State Management (save messages before & after stream)
 *   4. Graceful error handling if Python is down
 *
 * Frontend MUST only know about the Java base URL.
 */
function getJavaApiUrl(): string {
  return process.env.NEXT_PUBLIC_JAVA_API_URL || "http://localhost:8080";
}

interface ResponseMetadata {
  agent_used?: string;
  citations?: Array<{
    source_file: string;
    page_number?: number;
    document_id?: string;
    source_uri?: string;
    snippet?: string;
    [key: string]: unknown;
  }>;
  [key: string]: unknown;
}

interface StreamCallbacks {
  onChunk: (chunk: string) => void;
  onEscalation?: (message: string) => void;
  onFinish?: (metadata?: ResponseMetadata) => void;
  onError?: (error: string) => void;
  onRateLimit?: (message: string) => void;
  onSecurityBlock?: () => void;
}

interface StreamOptions {
  autoTriggered?: boolean;
}

type ClassifierErrorCode = "ERR_PII_DETECTED" | "INTENT_VIOLATION" | "MALICIOUS_INJECTION";

interface ClassifierErrorPayload {
  error_code?: ClassifierErrorCode | string;
  error?: string;
  message?: string;
  suggested_channel?: "PRIVATE" | null | string;
}

async function readErrorPayload(response: Response): Promise<ClassifierErrorPayload | null> {
  try {
    return (await response.json()) as ClassifierErrorPayload;
  } catch {
    return null;
  }
}

/** Policy / attack paths — never suggest switching channel (distinct from PII leak UX). */
const POLICY_REJECTION_TOAST =
  "Yêu cầu bị từ chối do vi phạm chính sách an toàn của hệ thống.";

function handleClassifierHttpError(payload: ClassifierErrorPayload | null, callbacks: StreamCallbacks): boolean {
  const errorCode = payload?.error_code ?? payload?.error;

  // PII only (regex / firewall layer). Do not treat INTENT_VIOLATION as PII — classifier/policy blocks use that code too.
  if (errorCode === "ERR_PII_DETECTED") {
    toast.warning("Câu hỏi chứa thông tin cá nhân. Vui lòng chuyển sang kênh Chat 1v1.");
    callbacks.onSecurityBlock?.();
    return true;
  }

  if (errorCode === "INTENT_VIOLATION" || errorCode === "MALICIOUS_INJECTION") {
    toast.error(POLICY_REJECTION_TOAST);
    callbacks.onSecurityBlock?.();
    return true;
  }

  return false;
}

/**
 * Parses a single SSE block (one or more lines separated by single newlines).
 * A block ends with a double newline in the stream.
 */
export function handleSseEvent(event: string, callbacks: StreamCallbacks): boolean {
  if (!event.trim()) return false;
  
  // Standard SSE splitting
  const lines = event.replace(/\r/g, "").split("\n");
  
  let eventType = "message";
  let dataBuffer = "";

  for (const line of lines) {
    const trimmedLine = line.trim();
    if (!trimmedLine) continue;

    // Correct SSE field parsing
    if (trimmedLine.toLowerCase().startsWith("event:")) {
      eventType = trimmedLine.slice(6).trim().toLowerCase();
    } else if (trimmedLine.toLowerCase().startsWith("data:")) {
      // Extract data payload, stripping ONLY the first "data:" prefix
      const dataPayload = line.trimStart().slice(5).trimStart();
      dataBuffer += (dataBuffer ? "\n" : "") + dataPayload;
    } else if (trimmedLine.toLowerCase().startsWith("id:")) {
      // Ignore id for now
    } else if (trimmedLine.toLowerCase().startsWith("retry:")) {
      // Ignore retry for now
    } else {
      // This might be a continuation of a multi-line field if the backend is non-standard
      // but according to spec, unknown fields should be ignored.
      // However, some backends send raw data without prefix.
      if (!dataBuffer && !trimmedLine.includes(":")) {
         dataBuffer = trimmedLine;
      }
    }
  }

  const cleanData = dataBuffer.trim();

  // 1. Handle error events explicitly
  if (eventType === "error") {
    if (cleanData) {
      try {
        const parsed = JSON.parse(cleanData);
        callbacks.onError?.(parsed.error ?? parsed.message ?? "AI service error");
      } catch {
        // Handle raw text error (e.g. Python's "Internal Server Error")
        callbacks.onError?.(cleanData || "AI service error");
      }
    } else {
      callbacks.onError?.("AI service error");
    }
    return true;
  }

  if (!cleanData) return false;

  // 2. Helper to process a JSON chunk
  const processJson = (jsonStr: string): boolean => {
    try {
      // Defensive check: if it's an SSE field name, don't parse it as JSON
      const lower = jsonStr.toLowerCase();
      if (lower.startsWith("event:") || lower.startsWith("data:") || lower.startsWith("id:")) {
        return false;
      }

      const data = JSON.parse(jsonStr);
      
      if (data.type === "system_error" || data.error) {
        if (data.code === 429) {
          callbacks.onRateLimit?.(data.message ?? data.error);
        } else {
          callbacks.onError?.(data.message ?? data.error ?? "AI service error");
        }
        return true;
      }

      if (data.type === "escalation") {
        callbacks.onEscalation?.(data.message);
      } else if (typeof data.chunk === "string" && data.chunk.length > 0) {
        callbacks.onChunk(data.chunk);
      }

      if (data.is_finished === true) {
        callbacks.onFinish?.(data.metadata);
        return true;
      }
    } catch (e) {
      // Only log if it's not a common non-JSON chunk like "[DONE]"
      if (jsonStr !== "[DONE]") {
        console.warn("[aiClient] Failed to parse SSE JSON chunk:", e instanceof Error ? e.message : "Unknown error", jsonStr);
      }
    }
    return false;
  };

  // 3. Process the accumulated data
  try {
    // Attempt parsing as a single JSON object (most common)
    return processJson(cleanData);
  } catch {
    // Fallback: handle cases where multiple JSON objects are sent in one block
    const lines = cleanData.split("\n").map(s => s.trim()).filter(Boolean);
    let shouldStop = false;
    for (const line of lines) {
      if (processJson(line)) shouldStop = true;
    }
    return shouldStop;
  }
}

async function streamViaGateway(
  endpointPathBuilder: (safeContextId: string) => string,
  contextId: string,
  message: string,
  callbacks: StreamCallbacks,
  signal?: AbortSignal,
  options: StreamOptions = {}
) {
  const baseUrl = getJavaApiUrl().replace(/\/$/, "");

  const token = useAuthStore.getState().token?.trim();
  if (!token) {
    callbacks.onError?.("Bạn cần đăng nhập để sử dụng Trợ giảng AI.");
    return;
  }

  const safeContextId = contextId?.trim();
  if (!safeContextId) {
    callbacks.onError?.("ID ngữ cảnh không hợp lệ.");
    return;
  }

  try {
    const response = await fetch(
      `${baseUrl}${endpointPathBuilder(safeContextId)}`,
      {
        method: "POST",
        signal,
        headers: {
          "Content-Type": "application/json",
          Authorization: `Bearer ${token}`,
          "Cache-Control": "no-cache",
        },
        body: JSON.stringify({ message, autoTriggered: options.autoTriggered === true }),
      }
    );

    if (!response.ok) {
      if (response.status === 401) {
        useAuthStore.getState().logout();
        return;
      }
      if (response.status === 403) {
        const payload = await readErrorPayload(response);
        if (handleClassifierHttpError(payload, callbacks)) {
          return;
        }
        callbacks.onError?.(payload?.message ?? `HTTP Error: ${response.status}`);
        return;
      }
      if (response.status === 429) {
        callbacks.onRateLimit?.(
          "Bạn đã hết lượt hỏi AI hôm nay. Vui lòng đợi TA hỗ trợ hoặc thử lại vào ngày mai."
        );
        return;
      }
      if (response.status === 503) {
        callbacks.onError?.(
          "Trợ giảng AI đang tạm thời không hoạt động. Câu hỏi của bạn đã được ghi nhận, TA sẽ hỗ trợ sớm."
        );
        return;
      }
      throw new Error(`HTTP Error: ${response.status}`);
    }

    if (!response.body) {
      throw new Error("No response body — SSE stream unavailable.");
    }

    const reader = response.body.getReader();
    const decoder = new TextDecoder();
    let buffer = "";

    while (true) {
      const { value, done } = await reader.read();
      if (done) {
        // Flush final decoder state and process any remaining event
        buffer += decoder.decode();
        if (buffer.trim()) {
          const shouldStop = handleSseEvent(buffer, callbacks);
          if (shouldStop) return;
        }
        // Failsafe: if stream closes without explicit is_finished event,
        // still notify UI to stop typing indicator.
        callbacks.onFinish?.();
        break;
      }

      buffer += decoder.decode(value, { stream: true });
      const events = buffer.split(/\r?\n\r?\n/);
      buffer = events.pop() ?? "";

      for (const event of events) {
        const shouldStop = handleSseEvent(event, callbacks);
        if (shouldStop) {
          return;
        }
      }
    }
  } catch (error: unknown) {
    const errorName = typeof error === "object" && error !== null && "name" in error
      ? String((error as { name?: unknown }).name)
      : "";
    if (errorName === "AbortError") {
      // Intentional user cancellation: do not surface as system error.
      return;
    }
    const errorMessage = error instanceof Error ? error.message : "Network error";
    callbacks.onError?.(errorMessage);
  }
}

/**
 * Streams AI response by calling the Java SSE Gateway.
 * Java internally communicates with the Python AI service via gRPC.
 *
 * @param threadId - Forum thread ID providing context for the AI
 * @param message  - The user's message. Empty string triggers first-time AI response.
 * @param callbacks - Handlers for stream chunks, completion, and errors
 */
export const streamChat = async (
  threadId: string,
  message: string,
  callbacks: StreamCallbacks,
  signal?: AbortSignal,
  options: StreamOptions = {}
) => {
  return streamViaGateway(
    (safeThreadId) => `/api/v1/threads/${safeThreadId}/ask-ai`,
    threadId,
    message,
    callbacks,
    signal,
    options
  );
};

export const streamPrivateChat = async (
  sessionId: string,
  message: string,
  callbacks: StreamCallbacks,
  signal?: AbortSignal,
  options: StreamOptions = {}
) => {
  return streamViaGateway(
    (safeSessionId) => `/api/v1/chat/sessions/${safeSessionId}/ask-ai`,
    sessionId,
    message,
    callbacks,
    signal,
    options
  );
};
