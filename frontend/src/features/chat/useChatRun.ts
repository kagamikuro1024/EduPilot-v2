"use client";

import { useCallback, useEffect, useRef, useState } from "react";
import { ApiError, apiClient } from "@/shared/data";
import { openChatStream, type ChatFrame } from "@/shared/data/chatStream"; // không qua barrel: chỉ chat thật cần (ngân sách JS mỗi route)
import type { ChatBlock, ChatCitation } from "./chatApi";

/** Một lượt đang chạy (hoặc vừa kết thúc, chờ lịch sử tải lại). `text` là văn bản đã khôi phục từ máy chủ. */
export type Live = {
  mid: string | null;
  userText: string | null;
  text: string;
  stage: "received" | "searching" | "generating";
  blocks: ChatBlock[];
  masked: number;
  degraded: boolean;
  low: boolean;
  citations: ChatCitation[];
  status: "streaming" | "done" | "failed" | "cancelled";
  error?: { code: string; retryAfter?: number };
};

type Start = ({ kind: "send"; sid: string; text: string; key: string } | { kind: "retry"; mid: string } | { kind: "resume"; mid: string }) & { onOpen?: () => void };

const RESUME_DELAYS = [500, 1500, 4000, 8000];
const runesOf = (s: string) => [...s].length;

/**
 * Máy trạng thái một lượt chat: gửi / thử lại / nối lại. Token gom theo khung hình (rAF), không phân tích lại markdown mỗi token.
 * Rớt kết nối → tự nối lại bằng Last-Event-ID (không mất, không lặp nhờ `off`). Lỗi TRƯỚC khi luồng mở (409, 422, 429…) ném lại cho người gọi để giữ chữ trong ô soạn.
 */
export function useChatRun(onSettled: () => Promise<unknown>) {
  const [live, setLive] = useState<Live | null>(null);
  const cur = useRef<Live | null>(null);
  const runes = useRef(0);
  const raf = useRef(0);
  const ctrl = useRef<AbortController | null>(null);
  const settled = useRef(onSettled);
  useEffect(() => {
    settled.current = onSettled;
  });

  const flush = useCallback(() => {
    raf.current = 0;
    setLive(cur.current ? { ...cur.current } : null);
  }, []);
  const push = useCallback(() => {
    if (!raf.current) raf.current = requestAnimationFrame(flush);
  }, [flush]);

  const abort = useCallback(() => {
    ctrl.current?.abort();
    ctrl.current = null;
  }, []);
  useEffect(
    () => () => {
      abort();
      if (raf.current) cancelAnimationFrame(raf.current);
    },
    [abort],
  );

  const run = useCallback(
    async (st: Start): Promise<void> => {
      abort();
      const c = new AbortController();
      ctrl.current = c;
      runes.current = 0;
      let lastId = "";
      let mid = st.kind === "send" ? null : st.mid;
      let terminal = false;
      let opened = false;
      cur.current = { mid, userText: st.kind === "send" ? st.text : null, text: "", stage: "received", blocks: [], masked: 0, degraded: false, low: false, citations: [], status: "streaming" };
      setLive({ ...cur.current });

      const on = (f: ChatFrame) => {
        const l = cur.current;
        if (!l) return;
        if (f.id) lastId = f.id;
        if (!opened) {
          opened = true;
          st.onOpen?.(); // luồng đã mở: máy chủ đã nhận tin → người gọi xoá nháp
        }
        const d = f.data;
        switch (f.event) {
          case "status":
            if (typeof d.message_id === "string") {
              mid = d.message_id;
              l.mid = mid;
            }
            l.stage = (d.stage as Live["stage"]) ?? l.stage;
            if (d.stage === "received") {
              l.text = ""; // lượt mới (thử lại) bắt đầu lại từ đầu
              runes.current = 0;
            }
            break;
          case "snapshot":
            l.text = String(d.t ?? "");
            runes.current = runesOf(l.text);
            break;
          case "token": {
            const t = String(d.t ?? "");
            const off = Number(d.off ?? runes.current);
            const skip = Math.max(0, runes.current - off); // phần đã có (nối lại): bỏ, không lặp
            if (off <= runes.current && skip < runesOf(t)) {
              const add = skip ? [...t].slice(skip).join("") : t;
              l.text += add;
              runes.current += runesOf(add);
            }
            l.stage = "generating";
            break;
          }
          case "block":
            l.blocks = [...l.blocks, { kind: String(d.kind), data: d.data }];
            break;
          case "notice":
            if (typeof d.masked === "number") l.masked = d.masked;
            if (d.degraded) l.degraded = true;
            break;
          case "done":
            terminal = true;
            l.status = "done";
            l.citations = (d.citations as ChatCitation[]) ?? [];
            l.degraded = Boolean(d.degraded);
            l.low = d.low_confidence === true;
            if (typeof d.content === "string") l.text = d.content;
            break;
          case "error":
            terminal = true;
            l.status = d.code === "CANCELLED" ? "cancelled" : "failed";
            l.error = { code: String(d.code), retryAfter: typeof d.retry_after === "number" ? d.retry_after : undefined };
            break;
        }
        push();
      };

      const first = (): Parameters<typeof openChatStream>[0] => {
        if (st.kind === "send") return { method: "POST", path: `/chat/sessions/${st.sid}/messages`, body: { content: st.text }, idempotencyKey: st.key, signal: c.signal, onFrame: on };
        if (st.kind === "retry") return { method: "POST", path: `/chat/messages/${st.mid}/retry`, signal: c.signal, onFrame: on };
        return { method: "GET", path: `/chat/messages/${st.mid}/stream`, signal: c.signal, onFrame: on };
      };

      let spec = first();
      for (let attempt = 0; ; attempt++) {
        try {
          await openChatStream(spec);
        } catch (e) {
          if (!(e instanceof ApiError)) throw e;
          if (e.code === "ABORTED") return;
          // lỗi trước khi có luồng (gửi / thử lại): trả cho người gọi — nháp giữ nguyên
          if (!mid || (e.code !== "NETWORK" && !terminal)) {
            cur.current = null;
            setLive(null);
            throw e;
          }
        }
        if (terminal) break;
        if (!mid || attempt >= RESUME_DELAYS.length) {
          const l = cur.current;
          if (l) {
            l.status = "failed";
            l.error = { code: "NETWORK" };
            push();
          }
          break;
        }
        await new Promise((r) => setTimeout(r, RESUME_DELAYS[attempt]));
        if (c.signal.aborted) return;
        spec = { method: "GET", path: `/chat/messages/${mid}/stream`, lastEventId: lastId || undefined, signal: c.signal, onFrame: on };
      }
      if (raf.current) cancelAnimationFrame(raf.current);
      flush();
      await settled.current();
      if (ctrl.current === c) {
        ctrl.current = null;
        // thành công: lịch sử đã có bản bền → bỏ lượt tạm. Thất bại / Dừng: GIỮ để còn dòng lỗi (kèm thời gian chờ) và nút Thử lại cho tới khi người dùng làm gì đó.
        if (cur.current?.status === "done") {
          cur.current = null;
          setLive(null);
        }
      }
    },
    [abort, flush, push],
  );

  /** Nút Dừng: huỷ tới provider; luồng sẽ tự đóng bằng sự kiện CANCELLED. */
  const stop = useCallback(async () => {
    const mid = cur.current?.mid;
    if (!mid) {
      abort();
      cur.current = null;
      setLive(null);
      return;
    }
    await apiClient.post(`/chat/messages/${mid}/cancel`).catch(() => undefined);
  }, [abort]);

  const clear = useCallback(() => {
    abort();
    cur.current = null;
    setLive(null);
  }, [abort]);

  return {
    live,
    clear,
    /** Gửi một tin: ném ApiError nếu bị từ chối trước khi luồng mở. `key` (UUID) giữ nguyên khi gửi lại cùng ý định; `onOpen` gọi khi luồng đã mở. */
    send: (sid: string, text: string, key: string, onOpen?: () => void) => run({ kind: "send", sid, text, key, onOpen }),
    retry: (mid: string, onOpen?: () => void) => run({ kind: "retry", mid, onOpen }),
    resume: (mid: string) => run({ kind: "resume", mid }),
    stop,
  };
}
