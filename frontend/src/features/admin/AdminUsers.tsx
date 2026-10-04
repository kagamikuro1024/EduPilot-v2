"use client";

import { useQueryClient } from "@tanstack/react-query";
import { Mail, Search } from "lucide-react";
import { useEffect, useRef, useState } from "react";
import { ApiError, apiClient, useCursorList } from "@/shared/data";
import { useAuth } from "@/shared/session/AuthProvider";
import {
  Button,
  DataTable,
  EmptyState,
  Field,
  InlineNotice,
  Input,
  OverflowMenu,
  Page,
  PageHeader,
  PageState,
  SegmentedControl,
  StatusText,
  Toolbar,
  UndoLine,
  type Column,
} from "@/shared/ui";
import s from "./admin.module.css";
import { InvitePanel } from "./users/InvitePanel";
import { ROLE_TEXT, STATUS_TEXT, USERS_KEY, lastSeen, type AdminUser } from "./users/api";

type Filter = "all" | "TEACHER" | "TA" | "ADMIN" | "STUDENT";
type Line = { text: string; undo?: () => void };
type Conflict = { user: AdminUser; to: "ACTIVE" | "DISABLED"; currentVersion: number };

/** Người dùng của toàn hệ thống (Admin): mời giảng viên / trợ giảng, gửi lại lời mời, khoá / mở khoá. Dữ liệu thật qua `/admin/users`. */
export function AdminUsers() {
  const auth = useAuth();
  const qc = useQueryClient();
  const me = auth.user?.id;
  const [filter, setFilter] = useState<Filter>("all");
  const [query, setQuery] = useState("");
  const [q, setQ] = useState(""); // đã trễ 250 ms: không bắn một truy vấn cho mỗi phím
  const [inviting, setInviting] = useState(false);
  const [line, setLine] = useState<Line | null>(null);
  const [error, setError] = useState<{ message: string; retry?: () => void } | null>(null);
  const [conflict, setConflict] = useState<Conflict | null>(null);
  const [over, setOver] = useState<Record<string, AdminUser["status"]>>({}); // cập nhật lạc quan
  const versions = useRef<Record<string, number>>({});
  const inflight = useRef<Record<string, Promise<unknown>>>({});

  useEffect(() => {
    const t = setTimeout(() => setQ(query.trim()), 250);
    return () => clearTimeout(t);
  }, [query]);

  const list = useCursorList<AdminUser>([...USERS_KEY, filter, q], "/admin/users", { limit: 30, query: { role: filter === "all" ? undefined : filter, q: q || undefined } });
  const refresh = () => qc.invalidateQueries({ queryKey: USERS_KEY });

  /** Khoá / mở khoá: lạc quan, ghi lên máy chủ, dòng tĩnh "Đã … · Hoàn tác" 5 s. Hoàn tác = thao tác ngược (đợi lần ghi trước xong để có version đúng). */
  function setStatus(u: AdminUser, to: "ACTIVE" | "DISABLED", undoing = false) {
    setError(null);
    setConflict(null);
    setOver((o) => ({ ...o, [u.id]: to }));
    const back = to === "DISABLED" ? "ACTIVE" : "DISABLED";
    setLine({ text: to === "DISABLED" ? `Đã khoá ${u.full_name}` : `Đã mở khoá ${u.full_name}`, undo: undoing ? undefined : () => setStatus(u, back, true) });
    const run = (async () => {
      await inflight.current[u.id]?.catch(() => undefined);
      try {
        const version = versions.current[u.id] ?? u.version;
        const { data } = await apiClient.patch<AdminUser>(`/admin/users/${u.id}`, { status: to, version });
        versions.current[u.id] = data.version;
        await refresh();
      } catch (e) {
        setLine(null);
        if (e instanceof ApiError && e.code === "VERSION_CONFLICT" && e.conflict) setConflict({ user: u, to, currentVersion: e.conflict.currentVersion });
        else setError({ message: e instanceof ApiError ? errorText(e) : "Chưa lưu được. Hãy thử lại.", retry: () => setStatus(u, to) });
      } finally {
        setOver((o) => {
          const n = { ...o };
          delete n[u.id];
          return n;
        });
      }
    })();
    inflight.current[u.id] = run;
  }

  async function resend(u: AdminUser) {
    setError(null);
    try {
      await apiClient.post(`/admin/users/${u.id}/resend-invite`);
      setLine({ text: `Đã gửi lại lời mời tới ${u.email}, hạn 72 giờ.` });
    } catch (e) {
      setError({ message: e instanceof ApiError ? errorText(e) : "Chưa gửi được. Hãy thử lại.", retry: () => void resend(u) });
    }
  }

  const columns: Column<AdminUser>[] = [
    { key: "name", header: "Họ tên", frozen: true, render: (u) => <span className={s.name}>{u.full_name}</span> },
    { key: "email", header: "Email", render: (u) => <span className={s.sub}>{u.email}</span> },
    { key: "role", header: "Vai trò", render: (u) => ROLE_TEXT[u.role] },
    {
      key: "state",
      header: "Trạng thái",
      render: (u) => {
        const st = STATUS_TEXT[over[u.id] ?? u.status];
        return <StatusText tone={st.tone}>{st.text}</StatusText>;
      },
    },
    { key: "last", header: "Lần cuối", render: (u) => <span className={s.sub}>{lastSeen(u.last_login_at)}</span> },
    {
      key: "action",
      header: "",
      align: "end",
      width: "220px",
      render: (u) => {
        const status = over[u.id] ?? u.status;
        const self = u.id === me;
        return (
          <span className={s.rowActions}>
            {status === "INVITED" && (
              <Button size="sm" variant="ghost" onClick={() => void resend(u)}>
                Gửi lại lời mời
              </Button>
            )}
            {!self && (
              <OverflowMenu
                label={`Thao tác cho ${u.full_name}`}
                items={[status === "DISABLED" ? { label: "Mở khoá", onSelect: () => setStatus(u, "ACTIVE") } : { label: "Khoá tài khoản", danger: true, onSelect: () => setStatus(u, "DISABLED") }]}
              />
            )}
          </span>
        );
      },
    },
  ];

  return (
    <Page width="wide">
      <PageHeader
        title="Người dùng"
        description="Mời giảng viên và trợ giảng, khoá hoặc mở khoá tài khoản. Sinh viên tự đăng ký rồi vào lớp bằng mã tham gia."
        actions={
          inviting ? undefined : (
            <Button variant="primary" icon={<Mail aria-hidden />} onClick={() => setInviting(true)}>
              Mời giảng viên
            </Button>
          )
        }
      />

      {inviting && (
        <InvitePanel
          onClose={() => setInviting(false)}
          onSent={() => {
            setInviting(false);
            setLine({ text: "Đã gửi link mời, hạn 72 giờ." });
            void refresh();
          }}
        />
      )}

      <Toolbar
        end={
          <Field label="Tìm người dùng" className={s.searchField}>
            {(id) => (
              <span className={s.search}>
                <Search aria-hidden />
                <Input id={id} value={query} onChange={(e) => setQuery(e.target.value)} placeholder="Tên hoặc đầu email" />
              </span>
            )}
          </Field>
        }
      >
        <SegmentedControl
          label="Lọc theo vai trò"
          value={filter}
          onChange={setFilter}
          options={[
            { value: "all", label: "Tất cả" },
            { value: "TEACHER", label: "Giảng viên" },
            { value: "TA", label: "Trợ giảng" },
            { value: "ADMIN", label: "Quản trị" },
            { value: "STUDENT", label: "Sinh viên" },
          ]}
        />
      </Toolbar>

      {line && <UndoLine key={line.text} message={line.text} onUndo={line.undo} onDone={() => setLine(null)} />}
      {error && (
        <InlineNotice tone="danger" compact action={error.retry ? <Button size="sm" onClick={error.retry}>Thử lại</Button> : undefined}>
          {error.message}
        </InlineNotice>
      )}
      {conflict && (
        <InlineNotice
          tone="warning"
          compact
          action={
            <>
              <Button
                size="sm"
                onClick={() => {
                  versions.current[conflict.user.id] = conflict.currentVersion;
                  setStatus(conflict.user, conflict.to);
                }}
              >
                Giữ thay đổi của tôi
              </Button>
              <Button
                size="sm"
                variant="ghost"
                onClick={() => {
                  setConflict(null);
                  void refresh();
                }}
              >
                Dùng bản mới
              </Button>
            </>
          }
        >
          Tài khoản này vừa được người khác sửa. Giữ thay đổi của bạn hay dùng bản mới?
        </InlineNotice>
      )}

      <PageState query={{ isPending: list.isPending, isError: list.isError, error: list.error, data: list.items, refetch: list.refetch }} showTechnical>
        <DataTable
          caption="Danh sách người dùng"
          columns={columns}
          rows={list.items}
          rowKey={(u) => u.id}
          dense
          empty={<EmptyState title="Chưa có người dùng khớp bộ lọc.">Thử bỏ bớt từ khoá hoặc chọn lại vai trò.</EmptyState>}
          pagination={{ nextCursor: list.hasNextPage ? "next" : null, onLoadMore: () => void list.fetchNextPage(), loading: list.isFetchingNextPage }}
        />
      </PageState>
    </Page>
  );
}

function errorText(e: ApiError): string {
  const reason = (e.details as { reason?: string } | undefined)?.reason;
  if (e.code === "CONFLICT" && reason === "self") return "Bạn không thể tự khoá chính mình.";
  if (e.code === "CONFLICT" && reason === "last_admin") return "Không thể khoá quản trị viên cuối cùng.";
  if (e.code === "CONFLICT") return "Chỉ gửi lại được lời mời cho tài khoản chưa nhận.";
  if (e.code === "RATE_LIMITED") return e.retryAfter ? `Vừa gửi lời mời rồi. Thử lại sau ${e.retryAfter} giây.` : "Vừa gửi lời mời rồi. Thử lại sau ít phút.";
  return e.userMessage;
}
