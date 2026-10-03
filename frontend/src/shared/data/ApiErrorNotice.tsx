"use client";

import { RefreshCw } from "lucide-react";
import { useEffect, useId, useState } from "react";
import { Button, InlineNotice } from "@/shared/ui";
import { ApiError } from "./ApiError";

const BUSY = ["RATE_LIMITED", "NOT_READY", "OVERLOADED", "SERVICE_UNAVAILABLE"];

/**
 * Hiển thị MỘT lỗi của lớp dữ liệu: lời tiếng Việt từ `userMessage` (không mã trần). Lỗi "bận" có `retry_after` thì đếm ngược mỗi giây
 * ("Hệ thống đang bận. Bạn có thể thử lại sau N giây."), nút Thử lại khoá tới hết giờ. `trace_id` chỉ nằm trong "Chi tiết kỹ thuật"
 * và CHỈ khi `showTechnical` (TA / GV / Admin) — Sinh viên không bao giờ thấy.
 */
export function ApiErrorNotice({ error, onRetry, showTechnical = false, title, context }: { error: unknown; onRetry?: () => void; showTechnical?: boolean; /** thay câu `userMessage` bằng tiêu đề riêng của màn */ title?: string; /** câu trấn an dưới tiêu đề (dữ liệu an toàn…) */ context?: string }) {
  const e = error instanceof ApiError ? error : fallback;
  // key theo từng lỗi cụ thể ⇒ lỗi mới đếm ngược lại từ đầu mà không cần setState trong effect
  return <Notice key={ident(e)} e={e} onRetry={onRetry} showTechnical={showTechnical} title={title} context={context} />;
}

const fallback = new ApiError({ status: 0, code: "INTERNAL" });
const ids = new WeakMap<object, number>();
let seq = 0;
const ident = (o: object) => {
  if (!ids.has(o)) ids.set(o, ++seq);
  return ids.get(o);
};

function Notice({ e, onRetry, showTechnical, title, context }: { e: ApiError; onRetry?: () => void; showTechnical: boolean; title?: string; context?: string }) {
  const busy = BUSY.includes(e.code) && (e.retryAfter ?? 0) > 0;
  const [left, setLeft] = useState(busy ? Math.ceil(e.retryAfter ?? 0) : 0);
  const hintId = useId();
  useEffect(() => {
    if (!busy) return;
    const t = setInterval(() => setLeft((n) => (n > 0 ? n - 1 : 0)), 1000); // cập nhật tối đa 1 lần / giây
    return () => clearInterval(t);
  }, [busy]);

  const waiting = busy && left > 0;
  const message = busy ? (left > 0 ? `Hệ thống đang bận. Bạn có thể thử lại sau ${left} giây.` : "Hệ thống đã sẵn sàng. Bạn có thể thử lại.") : e.userMessage;
  const technical = showTechnical && (e.traceId || e.code) ? `${e.code}${e.traceId ? ` · trace_id ${e.traceId}` : ""}` : undefined;
  if (e.code === "ABORTED") return null;
  return (
    <InlineNotice
      tone={busy ? "warning" : "danger"}
      title={busy ? undefined : (title ?? message)}
      technical={technical}
      action={
        onRetry && (
          <Button size="sm" icon={<RefreshCw aria-hidden />} disabled={waiting} aria-describedby={waiting ? hintId : undefined} onClick={onRetry}>
            Thử lại
          </Button>
        )
      }
    >
      {busy ? message : context}
      {waiting && (
        <span id={hintId} className="ep-sr-only">
          Nút Thử lại bị khoá tới khi hết thời gian chờ.
        </span>
      )}
    </InlineNotice>
  );
}
