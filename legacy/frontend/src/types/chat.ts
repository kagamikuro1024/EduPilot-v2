export type ChatAuthorRole = "STUDENT" | "AI" | "TA";
export type ChatFeedback = "LIKE" | "DISLIKE" | "NEEDS_TA";

export interface ChatSessionWire {
  session_id: string;
  title?: string;
  created_at?: string;
}

export interface CreateChatSessionResponseWire {
  session_id: string;
}

export interface ChatAuthorWire {
  id: string;
  role: ChatAuthorRole;
  name: string;
  avatar: string | null;
}

export interface ChatMessageWire {
  id: string;
  author: ChatAuthorWire;
  content: string;
  created_at: string;
  feedback?: ChatFeedback | null;
  is_escalated?: boolean;
  escalation_message?: string;
}
