"use client";

import { useQueryClient } from "@tanstack/react-query";
import { ArrowRight, Check } from "lucide-react";
import { useState } from "react";
import { apiClient } from "@/shared/data";
import { ActionList, ActionRow, Button, ButtonLink } from "@/shared/ui";
import s from "./Today.module.css";
import { classLabel, withoutClass } from "./format";
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
  PUBLISH_PENDING: "Công bố",
  GRADING_REVIEW: "Xem bài",
  GRADING_ERROR: "Mở hàng chờ",
  UNMATCHED_SUBMISSION: "Xem bài chưa khớp",
  APPEAL: "Xem yêu cầu",
};

type Group = "urgent" | "today" | "later";
const GROUPS: Array<{ key: Group; label: string }> = [
  { key: "urgent", label: "Gấp" },
  { key: "today", label: "Hôm nay" },
  { key: "later", label: "Khi rảnh" },
];
const QUIET_KINDS = new Set(["EXAM_SIMILARITY", "UNMATCHED_SUBMISSION"]);
/** Nhóm tính ở frontend từ `urgency` và `kind` (US-UX91-10 AC1); thứ tự máy chủ giữ nguyên trong nhóm. */
export const groupOf = (a: Pick<TodayItem, "urgency" | "kind">): Group => (a.urgency !== "normal" ? "urgent" : QUIET_KINDS.has(a.kind) ? "later" : "today");

/** Các việc đã xếp đúng thứ tự máy chủ trả, chia nhóm có nhãn chữ. `showClass`: chế độ "Tất cả lớp của tôi" ghi rõ lớp nào. */
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
  const visible = actions.filter((a) => !gone.has(a.id));
  const groups = GROUPS.map((g) => ({ ...g, items: visible.filter((a) => groupOf(a) === g.key) })).filter((g) => g.items.length > 0);
  const toggle = (id: string) => setOpen((o) => (o.has(id) ? new Set([...o].filter((x) => x !== id)) : new Set(o).add(id)));

  return (
    <>
      {failed && <p className={s.note} role="status">{failed}</p>}
      {groups.map((g, gi) => (
        <section key={g.key} className={s.group} data-group={g.key} aria-labelledby={`today-g-${g.key}`}>
          <h3 id={`today-g-${g.key}`} className={s.groupTitle} aria-label={`${g.label}, ${g.items.length} việc`}>
            {g.label} · {g.items.length}
          </h3>
          <ActionList label={g.label}>
            {g.items.map((a, ai) => (
              <ActionRow
                key={a.id}
                tone={a.urgency === "overdue" ? "red" : a.urgency === "high" ? "amber" : "neutral"}
                href={a.steps ? undefined : a.href || undefined}
                redThread={!a.steps}
                data={{ "data-kind": a.kind }}
                title={showClass ? a.title : withoutClass(a.title, a.course)}
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
                      // một nút đỏ duy nhất: việc đầu của nhóm đầu; các việc còn lại là ghost có mũi tên (US-UX91-10 AC2)
                      <ButtonLink href={a.href} size="sm" variant={gi === 0 && ai === 0 ? "primary" : "ghost"} iconEnd={gi === 0 && ai === 0 ? undefined : <ArrowRight aria-hidden />}>
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
        </section>
      ))}
    </>
  );
}
