"use client";

import { useState } from "react";
import { ExternalLink } from "lucide-react";
import { ApiError } from "@/shared/data";
import { ApiErrorNotice } from "@/shared/data/ApiErrorNotice";
import { Button, DefinitionList, Drawer, Field, InlineNotice, Textarea } from "@/shared/ui";
import { type RequestRow, secs, STATUS_LABEL, TASK_LABEL, useOpenRequest } from "./api";
import s from "./observability.module.css";

/** Khoá → lý do → nội dung đã che. Mỗi lần bấm «Xem chi tiết» ghi một dòng nhật ký kiểm toán (lý do + người xem). */
export function RequestDrawer({ row, onClose }: { row: RequestRow; onClose: () => void }) {
  const [reason, setReason] = useState("");
  const [touched, setTouched] = useState(false);
  const open = useOpenRequest();
  const len = reason.trim().length;
  const reasonError = len >= 10 ? undefined : touched || len > 0 ? "Ít nhất 10 ký tự." : undefined;
  const d = open.data;
  const expired = open.error instanceof ApiError && open.error.code === "PAYLOAD_EXPIRED";
  return (
    <Drawer open onClose={onClose} title={`${TASK_LABEL[row.task] ?? row.task} · ${row.id.slice(0, 8)}`} description={`${new Date(row.at).toLocaleString("vi-VN")} · ${row.model ?? "—"}`}>
      {!d ? (
        <div className={s.gate}>
          <InlineNotice tone="privacy" title="Nội dung là câu hỏi của sinh viên">Tên và mã số đã được thay bằng nhãn. Lý do của bạn được ghi vào nhật ký kiểm toán.</InlineNotice>
          <Field label="Lý do" required error={reasonError}>
            {(id, describedBy) => (
              <Textarea id={id} aria-describedby={describedBy} rows={3} maxLength={200} value={reason} invalid={Boolean(reasonError)} onChange={(e) => setReason(e.target.value)} onBlur={() => setTouched(true)} placeholder="Ví dụ: xử lý khiếu nại câu trả lời sai lúc 08:40" />
            )}
          </Field>
          {expired ? <InlineNotice tone="info" compact>Nội dung đã hết hạn lưu.</InlineNotice> : open.isError && <ApiErrorNotice error={open.error} />}
          <Button variant="primary" disabled={len < 10 || open.isPending} onClick={() => open.mutate({ id: row.id, reason: reason.trim() })}>Xem chi tiết</Button>
        </div>
      ) : (
        <div className={s.detail}>
          <h3 className={s.detailHead}>Nội dung đã che</h3>
          <p className={s.masked} style={{ whiteSpace: "pre-wrap" }}>{d.masked_prompt}</p>
          <h3 className={s.detailHead}>Công cụ đã gọi</h3>
          {d.tools.length === 0 ? <p className={s.sub}>Không có</p> : <ul className={s.tools}>{d.tools.map((t) => <li key={t} className={s.mono}>{t}</li>)}</ul>}
          <DefinitionList
            items={[
              { term: "Độ tin cậy", value: d.confidence ?? "—" },
              { term: "Độ trễ", value: secs(row.latency_ms) },
              { term: "Kết quả", value: STATUS_LABEL[d.status] ?? d.status },
              { term: "Mã theo dõi", value: d.trace_url ? <a href={d.trace_url} target="_blank" rel="noreferrer">Mở trace <ExternalLink aria-hidden size={14} /></a> : <span className={s.mono}>{d.trace_id}</span> },
            ]}
          />
        </div>
      )}
    </Drawer>
  );
}
