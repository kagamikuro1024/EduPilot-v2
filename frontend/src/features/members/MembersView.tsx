"use client";

import { useQuery, useQueryClient } from "@tanstack/react-query";
import { Search } from "lucide-react";
import { useSearchParams } from "next/navigation";
import { useEffect, useMemo, useState } from "react";
import { ApiError, apiClient, useCursorList } from "@/shared/data";
import { Button, ButtonLink, type Column, DataTable, EmptyState, Field, InlineNotice, Input, OverflowMenu, Page, PageHeader, PageState, Panel, SegmentedControl, Select, Skeleton, Tabs, Toolbar, UndoLine } from "@/shared/ui";
import s from "./MembersView.module.css";
import { classKey, useClassCourse, type Member, type MemberCounts } from "./classApi";
import { TODAY_KEY } from "@/features/today/todayApi";
import { RosterImport } from "./RosterImport";

type Tab = "members" | "pending" | "staff" | "import";
type Line = { text: string; undo?: () => void };
type Item = Member;

const MISMATCH = "Email chưa khớp MSSV";

/** Thành viên lớp, hàng chờ duyệt và trợ giảng (dữ liệu thật). TA xem và duyệt; chỉ giảng viên mời ra, đổi trợ giảng và duyệt hàng "email chưa khớp MSSV". */
export function MembersView() {
  const cc = useClassCourse();
  if (cc.state === "loading") return <Page><PageHeader title="Thành viên lớp" /><Panel><Skeleton lines={5} /></Panel></Page>;
  if (cc.state === "none") {
    return (
      <Page>
        <PageHeader title="Thành viên lớp" />
        <Panel><EmptyState title="Chưa chọn lớp">Chọn một lớp ở thanh trên. Quản trị viên mở lớp ở mục Lớp học.</EmptyState></Panel>
      </Page>
    );
  }
  return <Members courseId={cc.course.id} code={cc.course.class_code} canManage={cc.canManage} />;
}

function Members({ courseId, code, canManage }: { courseId: string; code: string; canManage: boolean }) {
  const qc = useQueryClient();
  const initial = useSearchParams().get("tab");
  const [tab, setTab] = useState<Tab>(initial === "pending" ? "pending" : initial === "staff" && canManage ? "staff" : initial === "import" && canManage ? "import" : "members");
  const [query, setQuery] = useState("");
  const [q, setQ] = useState("");
  const [role, setRole] = useState<"all" | "STUDENT">("all");
  const [gone, setGone] = useState<Set<string>>(new Set()); // ẩn lạc quan
  const [selected, setSelected] = useState<Set<string>>(new Set());
  const [line, setLine] = useState<Line | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [confirmRow, setConfirmRow] = useState<Item | null>(null);

  useEffect(() => {
    const t = setTimeout(() => setQ(query.trim()), 250);
    return () => clearTimeout(t);
  }, [query]);

  const listTab = tab === "import" ? "members" : tab; // tab nhập danh sách không có danh sách riêng: dùng lại bộ nhớ đệm của "Thành viên"
  const status = listTab === "pending" ? "PENDING" : "ACTIVE";
  const roleFilter = listTab === "staff" ? "TA" : role === "all" ? undefined : role;
  const list = useCursorList<Item>(classKey(courseId, "members", listTab, roleFilter ?? "", q), `/courses/${courseId}/members`, {
    limit: 30,
    query: { status, role: roleFilter, q: q || undefined },
    idOf: (m) => m.user_id,
  });
  // useCursorList gộp theo `id`: dòng thành viên dùng user_id làm khoá
  const items = useMemo(() => list.items.filter((m) => !gone.has(m.user_id)), [list.items, gone]);

  const counts = useQuery({
    queryKey: classKey(courseId, "counts"),
    queryFn: async ({ signal }): Promise<MemberCounts> =>
      (await apiClient.get<{ counts: MemberCounts }>(`/courses/${courseId}/members`, { signal, query: { limit: 1, status: "ACTIVE", role: "STUDENT" } })).data.counts,
  });
  // Việc của mình làm xong thì "Hôm nay" cũng làm mới ngay (máy chủ xoá cache theo sự kiện ≤ 2 s).
  const refresh = async () => {
    await qc.invalidateQueries({ queryKey: classKey(courseId) });
    void qc.invalidateQueries({ queryKey: TODAY_KEY });
  };
  const hide = (ids: string[]) => setGone((g) => new Set([...g, ...ids]));
  const show = (ids: string[]) => setGone((g) => new Set([...g].filter((x) => !ids.includes(x))));
  const path = (uid: string, op = "") => `/courses/${courseId}/members/${uid}${op}`;

  function fail(e: unknown): string {
    if (e instanceof ApiError) {
      if (e.code === "COURSE_FULL") return "Lớp đã đủ sĩ số. Hãy báo giảng viên.";
      if (e.code === "COURSE_ARCHIVED") return "Lớp này đã được lưu trữ.";
      if (e.code === "FORBIDDEN") return "Bạn không có quyền làm việc này.";
      return e.userMessage;
    }
    return "Chưa thực hiện được. Hãy thử lại.";
  }

  /** Duyệt / từ chối một hay nhiều yêu cầu: ẩn ngay, ghi lên máy chủ, dòng tĩnh "Đã … · Hoàn tác" 5 s (Hoàn tác gọi undo cho từng người). */
  async function decide(op: "approve" | "reject", rows: Item[], confirmMismatch = false) {
    if (rows.length === 0) return;
    setError(null);
    setConfirmRow(null);
    const ids = rows.map((r) => r.user_id);
    hide(ids);
    setSelected(new Set());
    const results = await Promise.allSettled(rows.map((r) => apiClient.post(path(r.user_id, `/${op}`), op === "approve" ? { confirm_mismatch: confirmMismatch } : undefined)));
    const ok = rows.filter((_, i) => results[i].status === "fulfilled");
    const bad = rows.filter((_, i) => results[i].status === "rejected");
    if (bad.length > 0) {
      show(bad.map((r) => r.user_id));
      const first = results.find((r): r is PromiseRejectedResult => r.status === "rejected");
      setError(`${op === "approve" ? "Chưa duyệt được" : "Chưa từ chối được"} ${bad.length} yêu cầu. ${fail(first?.reason)}`);
    }
    if (ok.length > 0) {
      const what = ok.length === 1 ? ok[0].full_name : `${ok.length} yêu cầu`;
      setLine({
        text: op === "approve" ? (ok.length === 1 ? `Đã duyệt ${what}` : `Đã duyệt ${what}`) : ok.length === 1 ? `Đã từ chối ${what}` : `Đã từ chối ${what}`,
        undo: () => {
          show(ok.map((r) => r.user_id));
          void Promise.allSettled(ok.map((r) => apiClient.post(path(r.user_id, "/undo")))).then(() => refresh());
        },
      });
    }
    await refresh();
  }

  async function remove(m: Item) {
    setError(null);
    hide([m.user_id]);
    try {
      await apiClient.delete(path(m.user_id), { idempotent: true });
      setLine({
        text: `Đã mời ${m.full_name} ra khỏi lớp`,
        undo: () => {
          show([m.user_id]);
          void apiClient.post(path(m.user_id, "/undo")).then(() => refresh(), () => refresh());
        },
      });
    } catch (e) {
      show([m.user_id]);
      setError(fail(e));
    }
    await refresh();
  }

  const nameCell = (m: Item) => (
    <>
      <span className={s.name}>{m.full_name}</span>
      <span className={s.sub}>{m.email}</span>
      {tab === "pending" && m.warning === "EMAIL_MISMATCH" && <span className={s.warn}>{MISMATCH}</span>}
    </>
  );
  const columns: Column<Item>[] = [
    { key: "name", header: "Họ tên", frozen: true, render: nameCell },
    { key: "code", header: "MSSV", render: (m) => <span className="ep-num">{m.student_code || "—"}</span> },
    { key: "via", header: "Vào lớp", render: (m) => <span className={s.sub}>{m.joined_via === "CODE" ? "Bằng mã" : m.joined_via === "ROSTER" ? "Danh sách lớp" : "Được gán"}</span> },
    {
      key: "action",
      header: "",
      align: "end",
      width: "260px",
      render: (m) => {
        if (tab === "pending") {
          const mismatch = m.warning === "EMAIL_MISMATCH";
          return (
            <span className={s.rowActions}>
              {mismatch && !canManage ? (
                <span className={s.sub}>Chờ giảng viên duyệt</span>
              ) : (
                <Button size="sm" variant="primary" onClick={() => (mismatch ? setConfirmRow(m) : void decide("approve", [m]))}>
                  Duyệt
                </Button>
              )}
              <Button size="sm" variant="ghost" onClick={() => void decide("reject", [m])}>
                Từ chối
              </Button>
            </span>
          );
        }
        if (tab === "members" && canManage && m.role_in_course === "STUDENT") {
          return <OverflowMenu label={`Thao tác cho ${m.full_name}`} items={[{ label: "Mời ra khỏi lớp", danger: true, onSelect: () => void remove(m) }]} />;
        }
        return null;
      },
    },
  ];
  const cols = tab === "staff" ? columns.filter((c) => c.key !== "code" && c.key !== "via" && c.key !== "action") : columns;

  const pendingCount = counts.data?.pending;
  return (
    <Page width="wide">
      <PageHeader
        title="Thành viên lớp"
        description={`Lớp ${code}. Duyệt yêu cầu vào lớp và quản lý người học.`}
        actions={<ButtonLink href="/class/settings">Mã và cài đặt tham gia</ButtonLink>}
      />
      <Tabs
        label="Thành viên lớp"
        value={tab}
        onChange={(t) => {
          setTab(t);
          setSelected(new Set());
        }}
        options={[
          { value: "members", label: "Thành viên", count: counts.data?.active },
          { value: "pending", label: "Chờ duyệt", count: pendingCount },
          ...(canManage ? [{ value: "staff" as const, label: "Trợ giảng" }, { value: "import" as const, label: "Nhập danh sách" }] : []),
        ]}
      />

      {tab === "import" ? (
        <RosterImport courseId={courseId} />
      ) : tab === "staff" ? (
        <Staff courseId={courseId} items={items} loading={list.isPending} onChanged={refresh} setLine={setLine} setError={setError} fail={fail} />
      ) : null}

      {line && <UndoLine key={line.text} message={line.text} onUndo={line.undo} onDone={() => setLine(null)} />}
      {error && <InlineNotice tone="danger" compact>{error}</InlineNotice>}
      {confirmRow && (
        <InlineNotice
          tone="warning"
          compact
          action={
            <>
              <Button size="sm" variant="primary" onClick={() => void decide("approve", [confirmRow], true)}>Duyệt {confirmRow.full_name}</Button>
              <Button size="sm" variant="ghost" onClick={() => setConfirmRow(null)}>Để sau</Button>
            </>
          }
        >
          Email của người này khác email trong danh sách lớp. Chỉ duyệt nếu bạn chắc đúng là sinh viên này.
        </InlineNotice>
      )}

      {tab === "pending" && selected.size > 0 && (
        <div className={s.bulk}>
          <Button variant="primary" onClick={() => void decide("approve", items.filter((m) => selected.has(m.user_id)))}>Duyệt {selected.size}</Button>
          <Button variant="ghost" onClick={() => void decide("reject", items.filter((m) => selected.has(m.user_id)))}>Từ chối {selected.size}</Button>
        </div>
      )}

      {tab !== "staff" && tab !== "import" && (
        <Panel>
          <Toolbar
            end={
              <Field label="Tìm thành viên" className={s.searchField}>
                {(id) => (
                  <span className={s.search}>
                    <Search aria-hidden />
                    <Input id={id} value={query} onChange={(e) => setQuery(e.target.value)} placeholder="Tên, MSSV hoặc đầu email" />
                  </span>
                )}
              </Field>
            }
          >
            {tab === "members" && (
              <SegmentedControl
                label="Lọc theo vai"
                value={role}
                onChange={setRole}
                options={[
                  { value: "all", label: "Tất cả" },
                  { value: "STUDENT", label: "Sinh viên" },
                ]}
              />
            )}
          </Toolbar>
          <PageState query={{ isPending: list.isPending, isError: list.isError, error: list.error, data: list.items, refetch: list.refetch }} showTechnical>
            <DataTable
              caption={tab === "pending" ? "Yêu cầu chờ duyệt" : "Thành viên lớp"}
              columns={cols}
              rows={items}
              rowKey={(m) => m.user_id}
              dense
              selection={tab === "pending" ? { selected, onChange: (next) => setSelected(new Set([...next].filter((id) => items.find((m) => m.user_id === id)?.warning !== "EMAIL_MISMATCH"))) } : undefined}
              empty={
                tab === "pending" ? (
                  <EmptyState title="Chưa có yêu cầu nào chờ duyệt.">Khi sinh viên vào lớp bằng mã mà lớp cần duyệt, yêu cầu sẽ hiện ở đây.</EmptyState>
                ) : (
                  <EmptyState title="Chưa có thành viên khớp.">Chia sẻ mã tham gia ở “Mã và cài đặt tham gia” để sinh viên vào lớp.</EmptyState>
                )
              }
              pagination={{ nextCursor: list.hasNextPage ? "next" : null, onLoadMore: () => void list.fetchNextPage(), loading: list.isFetchingNextPage }}
            />
          </PageState>
        </Panel>
      )}
    </Page>
  );
}

type Candidate = { id: string; full_name: string; email: string };

/** Trợ giảng của lớp: giảng viên thêm / bớt; mỗi lần lưu thay TOÀN BỘ tập trợ giảng (PUT …/assistants). */
function Staff({
  courseId,
  items,
  loading,
  onChanged,
  setLine,
  setError,
  fail,
}: {
  courseId: string;
  items: Item[];
  loading: boolean;
  onChanged: () => Promise<unknown>;
  setLine: (l: Line | null) => void;
  setError: (e: string | null) => void;
  fail: (e: unknown) => string;
}) {
  const [term, setTerm] = useState("");
  const [pick, setPick] = useState("");
  const cand = useQuery({
    queryKey: classKey(courseId, "candidates", term),
    queryFn: async ({ signal }) => (await apiClient.get<{ items: Candidate[] }>(`/courses/${courseId}/assistant-candidates`, { signal, query: { q: term || undefined } })).data.items,
  });
  const have = new Set(items.map((m) => m.user_id));
  const options = (cand.data ?? []).filter((c) => !have.has(c.id));

  async function put(ids: string[], text: string, back?: string[]) {
    setError(null);
    try {
      await apiClient.put(`/courses/${courseId}/assistants`, { ta_ids: ids });
      setLine({ text, undo: back ? () => void apiClient.put(`/courses/${courseId}/assistants`, { ta_ids: back }).then(() => onChanged(), () => onChanged()) : undefined });
    } catch (e) {
      setError(fail(e));
    }
    await onChanged();
  }

  const current = items.map((m) => m.user_id);
  return (
    <Panel>
    <div className={s.stack}>
      <div className={s.add}>
        <Field label="Thêm trợ giảng">
          {(id) => (
            <>
              <Input id={id} value={term} onChange={(e) => setTerm(e.target.value)} placeholder="Tìm theo tên hoặc đầu email" />
              <Select aria-label="Chọn trợ giảng" value={pick} onChange={(e) => setPick(e.target.value)}>
                <option value="">Chọn một người</option>
                {options.map((c) => (
                  <option key={c.id} value={c.id}>{c.full_name} · {c.email}</option>
                ))}
              </Select>
            </>
          )}
        </Field>
        <Button variant="primary" disabled={!pick} onClick={() => { const who = options.find((c) => c.id === pick); setPick(""); void put([...current, pick], `Đã thêm ${who?.full_name ?? "trợ giảng"}`, current); }}>
          Thêm
        </Button>
      </div>
      {loading ? (
        <Skeleton lines={3} />
      ) : items.length === 0 ? (
        <EmptyState title="Lớp chưa có trợ giảng.">Thêm trợ giảng để họ cùng duyệt yêu cầu và trả lời sinh viên.</EmptyState>
      ) : (
        <DataTable
          caption="Trợ giảng của lớp"
          columns={[
            { key: "name", header: "Họ tên", frozen: true, render: (m: Item) => <><span className={s.name}>{m.full_name}</span><span className={s.sub}>{m.email}</span></> },
            {
              key: "action",
              header: "",
              align: "end",
              width: "120px",
              render: (m: Item) => <Button size="sm" variant="ghost" onClick={() => void put(current.filter((x) => x !== m.user_id), `Đã bớt ${m.full_name}`, current)}>Bớt</Button>,
            },
          ]}
          rows={items}
          rowKey={(m) => m.user_id}
          dense
        />
      )}
    </div>
    </Panel>
  );
}
