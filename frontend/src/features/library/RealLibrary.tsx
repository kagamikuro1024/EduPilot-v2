"use client";

import { useRouter } from "next/navigation";
import { useState } from "react";
import { ApiError } from "@/shared/data";
import { ActionList, ActionRow, Button, ButtonLink, EmptyState, Field, InlineNotice, Input, OverflowMenu, Page, PageHeader, PageState, Panel, Select, Skeleton } from "@/shared/ui";
import { TYPE_LABEL, askAboutDoc, useLibrary, type LibItem } from "./libraryApi";
import r from "./RealLibrary.module.css";

const WEEKS = Array.from({ length: 20 }, (_, i) => String(i + 1));
const AGO = new Intl.DateTimeFormat("vi-VN", { day: "2-digit", month: "2-digit" });

export const metaOf = (d: Pick<LibItem, "type" | "week_no" | "updated_at">) => [TYPE_LABEL[d.type] ?? "Tài liệu", d.week_no ? `Tuần ${d.week_no}` : null, `cập nhật ${AGO.format(new Date(d.updated_at))}`].filter(Boolean).join(" · ");

/** Thư viện thật của sinh viên (DESIGN §14.15): ô tìm kiếm trước, lọc loại / tuần, dòng gọn có dấu loại tệp bằng chữ. */
export function RealLibrary({ courseId }: { courseId: string }) {
  const router = useRouter();
  const [f, setF] = useState({ q: "", type: "", week: "" });
  const list = useLibrary(courseId, f);
  const [err, setErr] = useState<string | null>(null);
  const filtered = f.q.trim().length >= 2 || f.type !== "" || f.week !== "";

  async function ask(d: LibItem) {
    setErr(null);
    try {
      router.push(`/chat?session=${await askAboutDoc(courseId, d.id)}`);
    } catch (e) {
      setErr(e instanceof ApiError ? e.userMessage : "Chưa mở được. Thử lại.");
    }
  }

  return (
    <Page width="full">
      <PageHeader title="Thư viện" />
      <div className={r.filters} role="search">
        <Field label="Tìm tài liệu">{(id) => <Input id={id} type="search" value={f.q} onChange={(e) => setF({ ...f, q: e.target.value })} placeholder="AES, chữ ký số, quy chế…" autoFocus />}</Field>
        <Field label="Loại">
          {(id) => (
            <Select id={id} value={f.type} onChange={(e) => setF({ ...f, type: e.target.value })}>
              <option value="">Mọi loại</option>
              {Object.entries(TYPE_LABEL).map(([k, v]) => <option key={k} value={k}>{v}</option>)}
            </Select>
          )}
        </Field>
        <Field label="Tuần">
          {(id) => (
            <Select id={id} value={f.week} onChange={(e) => setF({ ...f, week: e.target.value })}>
              <option value="">Mọi tuần</option>
              {WEEKS.map((w) => <option key={w} value={w}>Tuần {w}</option>)}
            </Select>
          )}
        </Field>
      </div>
      {err && <InlineNotice tone="danger" compact>{err}</InlineNotice>}
      <PageState
        query={list}
        isEmpty={() => list.items.length === 0}
        loading={<Panel><Skeleton lines={6} /></Panel>}
        empty={<Panel><EmptyState title={filtered ? "Không có tài liệu nào khớp." : "Chưa có tài liệu nào."}>{filtered && <Button variant="primary" onClick={() => setF({ q: "", type: "", week: "" })}>Xoá bộ lọc</Button>}</EmptyState></Panel>}
      >
        <Panel>
          <ActionList label="Tài liệu của lớp">
            {list.items.map((d) => (
              <ActionRow
                key={d.id}
                lead={<span className={r.kind}>{d.file_kind}</span>}
                title={d.title}
                context={d.snippet ? <p className={r.snippet}>{d.snippet}</p> : undefined}
                meta={metaOf(d)}
                href={`/library/${d.id}`}
                action={
                  <span className={r.rowActions}>
                    <ButtonLink href={`/library/${d.id}`} size="sm">Xem</ButtonLink>
                    {d.can_ask_ai && <OverflowMenu label={`Thêm với ${d.title}`} items={[{ label: "Hỏi AI về tài liệu", onSelect: () => void ask(d) }]} />}
                  </span>
                }
              />
            ))}
          </ActionList>
          {list.hasNextPage && <Button onClick={() => void list.fetchNextPage()} disabled={list.isFetchingNextPage}>Xem thêm</Button>}
        </Panel>
      </PageState>
    </Page>
  );
}
