import javaClient from "./javaClient";
import { formatRelativeUpdatedAt } from "@/lib/formatRelativeUpdatedAt";
import type {
  CreateThreadRequestBody,
  CreateThreadResponseWire,
  ThreadItemWire,
  ThreadListResponseWire,
} from "@/types/threads";

export interface ThreadListQuery {
  /** Backend: {@link team._8.aitrogiang.model.ThreadStatus} name, e.g. OPEN, RESOLVED */
  status?: string;
  /** Filter by tag name (case-insensitive on server) */
  tag?: string;
  /** Title substring search (server matches title only) */
  search?: string;
  page?: number;
}

export interface ThreadListItemVm {
  id: string;
  title: string;
  snippet: string;
  relativeTime: string;
  replyCount: number;
  status: string;
  tags: string[];
}

export interface ThreadAuthorWire {
  id: string;
  role: string;
  name: string;
  avatar: string | null;
}

export interface ThreadPostWire {
  id: string;
  author: ThreadAuthorWire;
  content: string;
  original_ai_content: string | null;
  verification_status: "UNVERIFIED" | "VERIFIED" | "CORRECTED" | "REJECTED";
  verified_by_ta: ThreadAuthorWire | null;
  created_at: string;
  reactions: Record<string, number>;
  is_accepted: boolean;
  citations: Array<Record<string, unknown>> | null;
}

function mapItemToVm(row: ThreadItemWire & Record<string, unknown>): ThreadListItemVm {
  const preview =
    (typeof row.last_message_preview === "string" && row.last_message_preview) ||
    (typeof row.lastMessagePreview === "string" && row.lastMessagePreview) ||
    "";
  const updated =
    (typeof row.updated_at === "string" && row.updated_at) ||
    (typeof row.updatedAt === "string" && row.updatedAt) ||
    "";
  const rawReply = row.reply_count ?? row.replyCount;
  const replyCount = typeof rawReply === "number" ? rawReply : Number(rawReply) || 0;

  return {
    id: String(row.id ?? ""),
    title: String(row.title ?? ""),
    snippet: preview.trim() || "No preview yet.",
    relativeTime: formatRelativeUpdatedAt(updated),
    replyCount,
    status: String(row.status ?? ""),
    tags: Array.isArray(row.tags) ? (row.tags as string[]) : [],
  };
}

export const threadsService = {
  async listThreads(query: ThreadListQuery = {}): Promise<{ items: ThreadListItemVm[]; total: number; page: number }> {
    const page = query.page ?? 1;
    const params: Record<string, string | number> = { page };
    if (query.status?.trim()) params.status = query.status.trim();
    if (query.tag?.trim()) params.tag = query.tag.trim();
    if (query.search?.trim()) params.search = query.search.trim();

    const response = await javaClient.get<ThreadListResponseWire>("/api/v1/threads", { params });
    const body = response.data;
    const rows = Array.isArray(body.data) ? body.data : [];
    const meta = body.meta;
    return {
      items: rows.map((r) => mapItemToVm(r as ThreadItemWire & Record<string, unknown>)),
      total: meta?.total ?? 0,
      page: meta?.page ?? page,
    };
  },

  async createThread(body: CreateThreadRequestBody): Promise<string> {
    const response = await javaClient.post<CreateThreadResponseWire>("/api/v1/threads", body);
    const id = response.data?.thread_id?.trim();
    if (!id) {
      throw new Error("Server did not return thread_id.");
    }
    return id;
  },

  async getMessages(threadId: string): Promise<ThreadPostWire[]> {
    const response = await javaClient.get<ThreadPostWire[]>(`/api/v1/threads/${threadId}/messages`);
    return Array.isArray(response.data) ? response.data : [];
  },

  async sendMessage(threadId: string, content: string): Promise<string> {
    const response = await javaClient.post<{ id: string }>(`/api/v1/threads/${threadId}/messages`, {
      content,
    });
    const id = response.data?.id?.trim();
    if (!id) {
      throw new Error("Server did not return message id.");
    }
    return id;
  },

  async verifyMessage(messageId: string): Promise<void> {
    await javaClient.put(`/api/v1/messages/${messageId}/verify`);
  },

  async correctMessage(messageId: string, content: string): Promise<void> {
    await javaClient.put(`/api/v1/messages/${messageId}/correct`, { content });
  },

  async rejectMessage(messageId: string): Promise<void> {
    await javaClient.put(`/api/v1/messages/${messageId}/reject`);
  },
};
