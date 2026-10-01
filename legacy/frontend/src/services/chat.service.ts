import javaClient from "./javaClient";
import type {
  ChatFeedback,
  ChatMessageWire,
  ChatSessionWire,
  CreateChatSessionResponseWire,
} from "@/types/chat";

export const chatService = {
  async listSessions(): Promise<ChatSessionWire[]> {
    const response = await javaClient.get<ChatSessionWire[]>("/api/v1/chat/sessions");
    return Array.isArray(response.data) ? response.data : [];
  },

  async createSession(): Promise<string> {
    const response = await javaClient.post<CreateChatSessionResponseWire>("/api/v1/chat/sessions");
    const sessionId = response.data?.session_id?.trim();
    if (!sessionId) {
      throw new Error("Server did not return session_id.");
    }
    return sessionId;
  },

  async getMessages(sessionId: string): Promise<ChatMessageWire[]> {
    const response = await javaClient.get<ChatMessageWire[]>(`/api/v1/chat/sessions/${sessionId}/messages`);
    return Array.isArray(response.data) ? response.data : [];
  },

  async submitFeedback(messageId: string, feedback: ChatFeedback): Promise<void> {
    await javaClient.put(`/api/v1/chat/messages/${messageId}/feedback`, { feedback });
  },

  async deleteSession(sessionId: string): Promise<void> {
    await javaClient.delete(`/api/v1/chat/sessions/${sessionId}`);
  },
};
