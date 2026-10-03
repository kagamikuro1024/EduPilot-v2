"use client";

import { useQueryClient } from "@tanstack/react-query";
import { Check } from "lucide-react";
import { useState } from "react";
import { apiClient } from "@/shared/data";
import { ActionList, ActionRow, Button, ButtonLink } from "@/shared/ui";
import s from "./Today.module.css";
import { classLabel } from "./format";
import { TODAY_KEY, type TodayItem } from "./todayApi";

const CTA: Record<string, string> = {
  JOIN_REQUEST: "Xem yêu cầu",
  EMAIL_MISMATCH: "Xem và xác nhận",
  COURSE_SETUP: "Mở thiết lập",
  LLM_PROVIDER_ERROR: "Mở cấu hình AI",
  LLM_BUDGET_EXHAUSTED: "Mở cấu hình AI",
  LLM_BUDGET_WARN: "Mở cấu hình AI",
  COURSE_NO_TEACHER: "Gán giảng viên",
  INVITE_EXPIRED: "Gửi lại lời mời",
};

/** Các việc đã xếp đúng thứ tự máy chủ trả. `showClass`: chế độ "Tất cả lớp của tôi" ghi rõ lớp nào. */
export function TodayActions({ actions, showClass, canDismiss }: { actions: TodayItem[]; showClass: boolean; canDismiss: boolean }) {
  const qc = useQueryClient();
  const [gone, setGone] = useState<Set<string>>(new Set());
  const [open, setOpen] = useState<Set<string>>(new Set());
  const [failed, setFailed] = useState<string | null>(null);

  async function dismiss(a: TodayItem) {
    if (!a.course) return;
    setFailed(null);
    setGone((g) => new Set(g).add(a.id)); // ẩn ngay; máy chủ ghi sau
    try {
      await apiClient.post(`/courses/${a.course.id}/setup/dismiss`);
      await qc.invalidateQueries({ queryKey: TODAY_KEY });
    } catch {
      setGone((g) => new Set([...g].filter((x) => x !== a.id)));
      setFailed("Chưa bỏ qua được. Hãy thử lại.");
    }
  }
  const toggle = (id: string) => setOpen((o) => (o.has(id) ? new Set([...o].filter((x) => x !== id)) : new Set(o).add(id)));

  return (
    <>
      {failed && <p className={s.note} role="status">{failed}</p>}
      <ActionList label="Việc cần xử lý">
        {actions.filter((a) => !gone.has(a.id)).map((a) => (
          <ActionRow
            key={a.id}
            tone={a.urgency === "overdue" ? "red" : a.urgency === "high" ? "amber" : "neutral"}
            href={a.steps ? undefined : a.href || undefined}
            redThread={!a.steps}
            data={{ "data-kind": a.kind }}
            title={a.title}
            context={
              <>
                {a.reason}
                {a.steps && (
                  <>
                    {" "}
                    <Button size="sm" variant="text" aria-expanded={open.has(a.id)} onClick={() => toggle(a.id)}>
                      {open.has(a.id) ? "Ẩn các bước" : `Xem ${a.steps.length} bước`}
                    </Button>
                    {open.has(a.id) && (
                      <ol className={s.steps}>
                        {a.steps.map((st) => (
                          <li key={st.key} className={st.done ? s.done : undefined}>
                            {st.done ? <Check aria-label="Đã xong" /> : <span className={s.pending} aria-hidden />}
                            <a href={st.href}>{st.label}</a>
                          </li>
                        ))}
                      </ol>
                    )}
                  </>
                )}
              </>
            }
            meta={showClass ? classLabel(a.course) : undefined}
            action={
              <span className={s.rowActions}>
                {a.href && (
                  <ButtonLink href={a.href} variant="text" size="sm">
                    {CTA[a.kind] ?? "Mở"}
                  </ButtonLink>
                )}
                {a.steps && canDismiss && (
                  <Button size="sm" variant="ghost" onClick={() => void dismiss(a)}>
                    Bỏ qua
                  </Button>
                )}
              </span>
            }
          />
        ))}
      </ActionList>
    </>
  );
}
