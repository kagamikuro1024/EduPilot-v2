"use client";

import { useEffect, useState } from "react";
import { useCursorList } from "@/shared/data";
import { ApiErrorNotice } from "@/shared/data/ApiErrorNotice";
import { X } from "lucide-react";
import { Button, type Column, DataTable, EmptyState, Field, Input, Page, PageHeader, PageState, Panel, SegmentedControl, Toolbar } from "@/shared/ui";
import s from "./admin.module.css";

type Row = {
  id: string; at: string; actor_id: string | null; actor_name: string | null; actor_role: string | null;
  entity: string; entity_id: string; action: string; course_id: string | null; trace_id: string | null; changed_fields: string[];
};
type Range = "24h" | "7d" | "30d";
const RANGE_MS: Record<Range, number> = { "24h": 86_400_000, "7d": 7 * 86_400_000, "30d": 30 * 86_400_000 };
const fromIso = (r: Range) => new Date(Date.now() - RANGE_MS[r]).toISOString();
const ROLE: Record<string, string> = { ADMIN: "Quản trị", TEACHER: "Giảng viên", TA: "Trợ giảng", STUDENT: "Sinh viên" };
const ENTITY = /^[a-z][a-z0-9_]{0,63}$/;

const personLabel = (r: Row) => (r.actor_name ? `${r.actor_name}${r.actor_role ? ` · ${ROLE[r.actor_role] ?? r.actor_role}` : ""}` : "Hệ thống");
const columns = (pick: (r: Row) => void): Column<Row>[] => [
  { key: "at", header: "Thời điểm", width: "170px", render: (r) => <span className="ep-num">{new Date(r.at).toLocaleString("vi-VN", { day: "2-digit", month: "2-digit", hour: "2-digit", minute: "2-digit", second: "2-digit" })}</span> },
  { key: "actor", header: "Người", frozen: true, primary: true, render: (r) => (r.actor_id ? <Button size="sm" variant="ghost" onClick={() => pick(r)} aria-label={`Lọc theo ${personLabel(r)}`}>{personLabel(r)}</Button> : personLabel(r)) },
  { key: "entity", header: "Thực thể", render: (r) => <span className={s.mono}>{r.entity}</span> },
  { key: "action", header: "Hành động", render: (r) => <span className={s.mono}>{r.action}</span> },
  { key: "fields", header: "Khoá đổi", render: (r) => (r.changed_fields.length ? <span className={s.mono}>{r.changed_fields.join(", ")}</span> : "—") },
];

/** Nhật ký kiểm toán (Admin): ai làm gì trên thực thể nào, không có giá trị trước / sau. Mỗi lần mở bảng cũng được ghi lại. */
export function AdminAudit() {
  const [range, setRange] = useState<Range>("7d");
  const [entityText, setEntityText] = useState("");
  const [entity, setEntity] = useState("");
  const [person, setPerson] = useState<{ id: string; label: string } | null>(null); // lọc theo người: bấm tên ở một dòng (API vẫn nhận actor_id)
  const actor = person?.id ?? "";
  useEffect(() => {
    const t = setTimeout(() => {
      const e = entityText.trim();
      if (e === "" || ENTITY.test(e)) setEntity(e); // giá trị sai không bao giờ thành truy vấn
    }, 400);
    return () => clearTimeout(t);
  }, [entityText]);

  // `from` chỉ đổi khi đổi khoảng (không theo từng lần dựng lại): mỗi truy vấn thành công ghi một dòng VIEW_LIST.
  const [from, setFrom] = useState(() => fromIso("7d"));
  const pick = (r: Range) => {
    setRange(r);
    setFrom(fromIso(r));
  };
  const validEntity = entityText.trim() === "" || ENTITY.test(entityText.trim());
  const list = useCursorList<Row>(["admin", "audit", range, from, entity, actor], "/admin/audit", {
    limit: 30,
    staleTime: 60_000,
    query: { from, entity: entity || undefined, actor_id: actor || undefined },
  });

  return (
    <Page kind="work">
      <PageHeader title="Nhật ký" />
      <Panel fault={list.isError && list.items.length === 0 ? <ApiErrorNotice error={list.error} showTechnical onRetry={() => void list.refetch()} /> : null}>
        <Toolbar>
          <SegmentedControl label="Khoảng thời gian" value={range} onChange={pick} options={[{ value: "24h", label: "24 giờ" }, { value: "7d", label: "7 ngày" }, { value: "30d", label: "30 ngày" }]} />
          <Field label="Thực thể" error={validEntity ? undefined : "Chữ thường, số, gạch dưới"}>
            {(id) => <Input id={id} value={entityText} onChange={(e) => setEntityText(e.target.value)} placeholder="llm_request" invalid={!validEntity} />}
          </Field>
          {person && <Button size="sm" iconEnd={<X aria-hidden />} onClick={() => setPerson(null)} aria-label="Bỏ lọc người">{`Người: ${person.label}`}</Button>}
        </Toolbar>
        <PageState query={{ isPending: list.isPending, isError: list.isError, error: list.error, data: list.items, refetch: list.refetch }} isEmpty={() => list.items.length === 0} empty={<EmptyState title="Không có dòng nào trong khoảng này" />} showTechnical>
          <DataTable
            caption="Nhật ký kiểm toán, mới nhất trước"
            columns={columns((r) => setPerson({ id: r.actor_id ?? "", label: personLabel(r) }))}
            rows={list.items}
            rowKey={(r) => r.id}
            dense
            mobile="scroll"
          />
          {list.hasNextPage && <Button onClick={() => void list.fetchNextPage()} loading={list.isFetchingNextPage}>Tải thêm</Button>}
        </PageState>
      </Panel>
    </Page>
  );
}
