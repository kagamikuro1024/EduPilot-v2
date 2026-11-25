"use client";

import { useRouter } from "next/navigation";
import { SlidersHorizontal } from "lucide-react";
import { useState } from "react";
import { ApiError } from "@/shared/data";
import { ActionList, ActionRow, Button, ButtonLink, Drawer, EmptyState, Field, InlineNotice, Input, OverflowMenu, Page, PageHeader, PageState, Panel, Select, Skeleton } from "@/shared/ui";
import { TYPE_LABEL, askAboutDoc, useLibrary, type LibItem } from "./libraryApi";
import r from "./RealLibrary.module.css";
import { formatDay } from "@/shared/lib/datetime";
import { useMinWidth } from "@/shared/lib/useMinWidth";

const WEEKS = Array.from({ length: 20 }, (_, i) => String(i + 1));

export const metaOf = (d: Pick<LibItem, "type" | "week_no" | "updated_at">) => [TYPE_LABEL[d.type] ?? "Tài liệu", d.week_no ? `Tuần ${d.week_no}` : null, `cập nhật ${formatDay(d.updated_at)}`].filter(Boolean).join(" · ");

/** Thư viện thật của sinh viên (DESIGN §14.15): ô tìm kiếm trước, lọc loại / tuần, dòng gọn có dấu loại tệp bằng chữ. */
export function RealLibrary({ courseId }: { courseId: string }) {
  const router = useRouter();
  const wide = useMinWidth(720);
  const [focusOnLoad] = useState(() => typeof window !== "undefined" && window.matchMedia("(min-width: 720px)").matches); // di động: không tự focus (bàn phím che nửa màn)
  const [sheet, setSheet] = useState(false);
  const [f, setF] = useState({ q: "", type: "", week: "", category: "" });
  const list = useLibrary(courseId, f);
  const all = useLibrary(courseId, { q: "", type: "", week: "", category: "" }); // danh sách không lọc (đã đệm khi chưa lọc): nguồn các chủ đề để chọn
  const categories = [...new Set(all.items.map((d) => d.category).filter((c): c is string => !!c))].sort((a, b) => a.localeCompare(b, "vi"));
  const [err, setErr] = useState<string | null>(null);
  const activeFilters = [f.type, f.week, f.category].filter(Boolean).length;
  const filtered = f.q.trim().length >= 2 || f.type !== "" || f.week !== "" || f.category !== "";

  async function ask(d: LibItem) {
    setErr(null);
    try {
      router.push(`/chat?session=${await askAboutDoc(courseId, d.id)}`);
    } catch (e) {
      setErr(e instanceof ApiError ? e.userMessage : "Chưa mở được. Thử lại.");
    }
  }

  const selects = (
    <>
      <Field label="Loại">
        {(id) => (
          <Select id={id} value={f.type} onChange={(e) => setF({ ...f, type: e.target.value })}>
            <option value="">Mọi loại</option>
            {Object.entries(TYPE_LABEL).map(([k, v]) => <option key={k} value={k}>{v}</option>)}
          </Select>
        )}
      </Field>
      {(categories.length > 0 || f.category !== "") && (
        <Field label="Chủ đề">
          {(id) => (
            <Select id={id} value={f.category} onChange={(e) => setF({ ...f, category: e.target.value })}>
              <option value="">Mọi chủ đề</option>
              {categories.map((c) => <option key={c} value={c}>{c}</option>)}
            </Select>
          )}
        </Field>
      )}
      <Field label="Tuần">
        {(id) => (
          <Select id={id} value={f.week} onChange={(e) => setF({ ...f, week: e.target.value })}>
            <option value="">Mọi tuần</option>
            {WEEKS.map((w) => <option key={w} value={w}>Tuần {w}</option>)}
          </Select>
        )}
      </Field>
    </>
  );

  return (
    <Page kind="read">
      <PageHeader title="Thư viện" />
      <div className={r.filters} role="search">
        <Field label="Tìm tài liệu">{(id) => <Input id={id} type="search" value={f.q} onChange={(e) => setF({ ...f, q: e.target.value })} placeholder="AES, chữ ký số, quy chế…" autoFocus={focusOnLoad} />}</Field>
        {wide && selects}
        {!wide && (
          <Button variant="secondary" className={r.filterToggle} icon={<SlidersHorizontal aria-hidden />} onClick={() => setSheet(true)}>
            Bộ lọc{activeFilters > 0 ? ` · ${activeFilters}` : ""}
          </Button>
        )}
      </div>
      {!wide && (
        <Drawer sheet open={sheet} onClose={() => setSheet(false)} title="Bộ lọc" footer={<Button variant="primary" onClick={() => setSheet(false)}>Xong</Button>}>
          <div className={r.sheetFields}>{selects}</div>
        </Drawer>
      )}
      {err && <InlineNotice tone="danger" compact>{err}</InlineNotice>}
      <PageState
        query={list}
        isEmpty={() => list.items.length === 0}
        loading={<Panel><Skeleton lines={6} /></Panel>}
        empty={<Panel><EmptyState title={filtered ? "Không có tài liệu nào khớp" : "Chưa có tài liệu nào"} /></Panel>}
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
                inlineAction
                action={
                  <span className={r.rowActions}>
                    {wide && <ButtonLink href={`/library/${d.id}`} size="sm">Xem</ButtonLink>}
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
