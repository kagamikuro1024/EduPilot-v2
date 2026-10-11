import { ApiError } from "./ApiError";
import { API_BASE } from "./endpoint";
import { refreshSession } from "./authSession";
import { netStatus } from "./netStatus";
import { tokenStore } from "./tokenStore";

// SSE qua POST / GET có Authorization cho chat riêng (SRS FEAT-private-chat-pii 4.7.2): fetch + ReadableStream, không EventSource, không token trên URL.
// Chỉ nằm ở shared/data (luật ep/no-raw-fetch). Trả khi luồng đóng; lỗi trước khi mở luồng ném ApiError (cùng mã như apiClient).
export type ChatFrame = { id?: string; event: string; data: Record<string, unknown> };

export type OpenChatStream = {
  method: "POST" | "GET";
  path: string;
  body?: unknown;
  /** POST gửi tin: UUID = client_msg_id (chống trùng ở DB). */
  idempotencyKey?: string;
  lastEventId?: string;
  signal: AbortSignal;
  onFrame: (f: ChatFrame) => void;
};

async function open(o: OpenChatStream, retried: boolean): Promise<Response> {
  const headers: Record<string, string> = { Accept: "text/event-stream", "X-Request-Id": crypto.randomUUID() };
  const tk = tokenStore.get();
  if (tk) headers.Authorization = `Bearer ${tk}`;
  if (o.body !== undefined) headers["Content-Type"] = "application/json";
  if (o.idempotencyKey) headers["Idempotency-Key"] = o.idempotencyKey;
  if (o.lastEventId) headers["Last-Event-ID"] = o.lastEventId;
  let res: Response;
  try {
    res = await fetch(`${API_BASE}${o.path}`, { method: o.method, headers, body: o.body === undefined ? undefined : JSON.stringify(o.body), credentials: "include", signal: o.signal });
  } catch {
    if (o.signal.aborted) throw new ApiError({ status: 0, code: "ABORTED" });
    netStatus.reportFailure();
    throw new ApiError({ status: 0, code: "NETWORK" });
  }
  netStatus.reportSuccess();
  if (res.ok) return res;
  let code = "BAD_GATEWAY";
  let details: unknown;
  let retryAfter: number | undefined;
  let message: string | undefined;
  try {
    const b = (await res.json()) as { code?: string; message?: string; details?: unknown; retry_after?: number };
    if (typeof b.code === "string") ({ code, details, message } = { code: b.code, details: b.details, message: b.message });
    if (typeof b.retry_after === "number") retryAfter = b.retry_after;
  } catch { /* thân không phải JSON */ }
  if (code === "TOKEN_EXPIRED" && !retried && (await refreshSession()).ok) return open(o, true);
  throw new ApiError({ status: res.status, code, details, message, retryAfter });
}

/** Mở luồng và gọi onFrame cho từng khung `id/event/data`; comment (`: ping`) bị bỏ. */
export async function openChatStream(o: OpenChatStream): Promise<void> {
  const res = await open(o, false);
  if (!res.body) throw new ApiError({ status: res.status, code: "PARSE_ERROR" });
  const reader = res.body.pipeThrough(new TextDecoderStream()).getReader();
  let buf = "";
  try {
    for (;;) {
      let chunk: ReadableStreamReadResult<string>;
      try {
        chunk = await reader.read();
      } catch {
        if (o.signal.aborted) throw new ApiError({ status: 0, code: "ABORTED" });
        throw new ApiError({ status: 0, code: "NETWORK" });
      }
      if (chunk.done) return;
      buf += chunk.value;
      let i: number;
      while ((i = buf.indexOf("\n\n")) >= 0) {
        const raw = buf.slice(0, i);
        buf = buf.slice(i + 2);
        const f = parseFrame(raw);
        if (f) o.onFrame(f);
      }
    }
  } finally {
    reader.cancel().catch(() => undefined);
  }
}

function parseFrame(raw: string): ChatFrame | null {
  let id: string | undefined;
  let event = "";
  const data: string[] = [];
  for (const line of raw.split("\n")) {
    if (line.startsWith(":")) continue;
    if (line.startsWith("id:")) id = line.slice(3).trim();
    else if (line.startsWith("event:")) event = line.slice(6).trim();
    else if (line.startsWith("data:")) data.push(line.slice(5).trimStart());
  }
  if (!event) return null;
  try {
    return { id, event, data: data.length ? (JSON.parse(data.join("\n")) as Record<string, unknown>) : {} };
  } catch {
    return null;
  }
}
