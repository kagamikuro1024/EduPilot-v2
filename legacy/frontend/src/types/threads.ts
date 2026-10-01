/**
 * Wire shapes for {@code GET /api/v1/threads} — mirror
 * {@link team._8.aitrogiang.dto.ThreadListResponse} / {@link team._8.aitrogiang.dto.ThreadItemDTO} / {@link team._8.aitrogiang.dto.AuthorDTO}.
 */
export interface ThreadAuthorWire {
  id: string;
  role: string;
  name: string;
  avatar: string | null;
}

export interface ThreadItemWire {
  id: string;
  title: string;
  author: ThreadAuthorWire;
  reply_count: number;
  status: string;
  tags: string[];
  last_message_preview: string;
  updated_at: string;
}

export interface ThreadListMetaWire {
  total: number;
  page: number;
}

export interface ThreadListResponseWire {
  data: ThreadItemWire[];
  meta: ThreadListMetaWire;
}

/** Body for {@code POST /api/v1/threads} — {@link team._8.aitrogiang.dto.CreateThreadRequest} */
export interface CreateThreadRequestBody {
  title: string;
  content: string;
  tags?: string[];
}

/** Response for {@code POST /api/v1/threads} — {@code Map.of("thread_id", threadId)} */
export interface CreateThreadResponseWire {
  thread_id: string;
}
