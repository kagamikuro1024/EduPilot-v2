"use client";

import { useRouter } from "next/navigation";
import { useEffect, useRef, useState, type ReactNode } from "react";
import { ApiError, apiClient, newIdempotencyKey, seedDraft } from "@/shared/data";
import { useSession } from "@/shared/session/session";
import { PIIChannelDialog } from "./PIIChannelDialog";
import { countReasons, type Precheck, type Reason } from "./threadsApi";

type Blocked = { reasons: Reason[]; personal_question: boolean };

/**
 * Cổng đăng bài công khai (SRS 4.9): precheck khi ngừng gõ 800 ms (một dòng báo loại thông tin, không mở hộp thoại), đăng với khoá MỚI sau mọi 4xx và khi thân đổi
 * (cùng khoá chỉ khi mất mạng / 5xx), 422 PII_DETECTED → hộp thoại đúng hai lối. Chữ trong ô soạn không bao giờ bị xoá khi lỗi.
 */
export function usePostGate<T>(o: {
  courseId: string;
  path: string;
  title?: string;
  body: string;
  payload: Record<string, unknown>;
  /** sinh viên có chat riêng; GV / TA không (hộp thoại chỉ còn Quay lại sửa / Ẩn rồi đăng) */
  canChat: boolean;
  onPosted: (r: T) => void;
  /** nháp đã chuyển sang chat riêng → xoá nháp phía Threads */
  onSwitched: () => void;
}): { check: Precheck | null; submit: (redact?: boolean) => Promise<void>; pending: boolean; error: ApiError | null; dialog: ReactNode } {
  const router = useRouter();
  const { identity } = useSession();
  const [check, setCheck] = useState<Precheck | null>(null);
  const [pending, setPending] = useState(false);
  const [error, setError] = useState<ApiError | null>(null);
  const [blocked, setBlocked] = useState<Blocked | null>(null);
  const key = useRef<{ sig: string; key: string } | null>(null);
  const title = o.title ?? "";

  useEffect(() => {
    if (!title.trim() && !o.body.trim()) {
      setCheck(null); // eslint-disable-line react-hooks/set-state-in-effect -- xoá hết chữ thì dòng báo biến mất
      return;
    }
    const c = new AbortController();
    const t = setTimeout(() => {
      apiClient.post<Precheck>(`/courses/${o.courseId}/threads/precheck`, { title, body: o.body }, { signal: c.signal }).then((r) => setCheck(r.data), () => undefined);
    }, 800);
    return () => {
      clearTimeout(t);
      c.abort();
    };
  }, [title, o.body, o.courseId]);

  async function submit(redact = false) {
    setError(null);
    setPending(true);
    const sig = JSON.stringify([o.path, o.payload, redact]);
    if (key.current?.sig !== sig) key.current = { sig, key: newIdempotencyKey() };
    try {
      const r = await apiClient.post<T>(o.path, { ...o.payload, redact }, { idempotencyKey: key.current.key });
      key.current = null;
      o.onPosted(r.data);
    } catch (e) {
      const err = e instanceof ApiError ? e : new ApiError({ status: 0, code: "NETWORK" });
      if (err.status >= 400 && err.status < 500) key.current = null; // khoá mới sau mọi 4xx; NETWORK / 5xx giữ khoá để gửi lại chỉ tạo một bài
      if (err.code === "PII_DETECTED") {
        const d = (err.details ?? {}) as Partial<Blocked>;
        setBlocked({ reasons: d.reasons ?? [], personal_question: Boolean(d.personal_question) });
      } else {
        setError(err);
      }
    } finally {
      setPending(false);
    }
  }

  async function toPrivateChat() {
    setBlocked(null);
    try {
      const r = await apiClient.post<{ session_id: string; draft: { title: string; body: string } }>("/chat/sessions/from-draft", { course_id: o.courseId, title, body: o.body });
      seedDraft(`chat:${o.courseId}:${r.data.session_id}`, r.data.draft.body, identity?.sub);
      o.onSwitched();
      router.push(`/chat?session=${r.data.session_id}`);
    } catch (e) {
      setError(e instanceof ApiError ? e : new ApiError({ status: 0, code: "NETWORK" }));
    }
  }

  const dialog = (
    <PIIChannelDialog
      open={blocked !== null}
      summary={blocked ? countReasons(blocked.reasons) || "câu hỏi riêng tư" : ""}
      onClose={() => setBlocked(null)}
      onPrivateChat={o.canChat ? () => void toPrivateChat() : undefined}
      onRedactedPost={() => {
        setBlocked(null);
        void submit(true);
      }}
      redactDisabled={blocked?.personal_question ? "Câu hỏi về điểm của riêng bạn không đăng công khai được." : undefined}
    />
  );
  return { check, submit, pending, error, dialog };
}
