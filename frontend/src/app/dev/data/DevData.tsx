"use client";

import { useQuery } from "@tanstack/react-query";
import { useSearchParams } from "next/navigation";
import { useEffect, useState } from "react";
import {
  ApiError, ApiErrorNotice, apiClient, fieldErrors, netStatus, resetApiClientState, tokenStore, useAutosaveDraft, useCursorList, useIdempotentMutation, useJob, useSSE,
  useSSEStatus, useUndoableAction, newIdempotencyKey,
} from "@/shared/data";
import { Button, EmptyState, Field, Input, PageState, Textarea } from "@/shared/ui";
import { NotificationPopover, type NotificationItem } from "@/shared/shell/NotificationPopover";
import s from "./DevData.module.css";

declare global {
  interface Window {
    __ep: unknown;
    __events: string[];
  }
}

function ErrDisplay({ staff }: { staff: boolean }) {
  const [err, setErr] = useState<unknown>(null);
  return (
    <section className={s.block} data-part="err-display" aria-label="Hiển thị lỗi">
      <h2 className="ep-section-title">Hiển thị lỗi</h2>
      <div className={s.row}>
        <Button onClick={() => apiClient.post("/fail", {}).then(() => setErr(null), setErr)}>Gọi lỗi</Button>
      </div>
      {err !== null && <ApiErrorNotice error={err} showTechnical={staff} onRetry={() => setErr(null)} />}
    </section>
  );
}

function FieldErrors() {
  const [name, setName] = useState("");
  const [errs, setErrs] = useState<Record<string, string>>({});
  return (
    <section className={s.block} data-part="field-errors" aria-label="Lỗi theo ô">
      <h2 className="ep-section-title">Lỗi theo ô</h2>
      <Field label="Tên" error={errs.name}>
        {(id, d) => <Input id={id} aria-describedby={d} invalid={Boolean(errs.name)} value={name} onChange={(e) => setName(e.target.value)} />}
      </Field>
      <div className={s.row}>
        <Button onClick={() => apiClient.post("/items", { name }).then(() => setErrs({}), (e) => setErrs(fieldErrors(e)))}>Lưu</Button>
      </div>
    </section>
  );
}

type Item = { id: string };
function CursorList() {
  const q = useCursorList<Item>(["dev-items"], "/items", { limit: 500 });
  return (
    <section className={s.block} data-part="cursor-list" aria-label="Danh sách con trỏ">
      <h2 className="ep-section-title">Danh sách con trỏ</h2>
      <div className={s.row}>
        <Button disabled={!q.hasNextPage || q.isFetchingNextPage} onClick={() => void q.fetchNextPage()}>Tải trang kế</Button>
        <span className={s.out} data-part="cursor-count">{q.items.length}</span>
        {q.isError && <ApiErrorNotice error={q.error} />}
      </div>
    </section>
  );
}

function Idem() {
  const m = useIdempotentMutation((v: { name: string }, key) => apiClient.post("/items", v, { idempotencyKey: key }));
  return (
    <section className={s.block} data-part="idem" aria-label="Khoá Idempotency">
      <h2 className="ep-section-title">Khoá Idempotency</h2>
      <div className={s.row}>
        <Button variant="primary" onClick={() => void m.mutate({ name: "x" }).catch(() => {})}>Gửi</Button>
        {m.error && <Button onClick={() => void m.retry()?.catch(() => {})}>Gửi lại</Button>}
        <span className={s.out} data-part="idem-result">{m.result ? `ok replayed=${m.result.replayed}` : m.error ? m.error.code : "chưa gửi"}</span>
      </div>
    </section>
  );
}

function Listener({ n, type }: { n: number; type: string }) {
  // người nghe số 0 đăng ký khoá "ps" cho resync; các người nghe còn lại chỉ nhận sự kiện
  useSSE(type, (e) => {
    (window as unknown as { __sse: string[] }).__sse.push(`${n}:${e.id ?? ""}:${e.data}`);
  }, { invalidateKeys: n === 0 ? [["ps"]] : [] });
  return null;
}

function Sse() {
  const [mounted, setMounted] = useState(0);
  const st = useSSEStatus();
  useEffect(() => {
    (window as unknown as { __sse: string[] }).__sse = [];
  }, []);
  return (
    <section className={s.block} data-part="sse" aria-label="SSE">
      <h2 className="ep-section-title">SSE</h2>
      <div className={s.row}>
        <Button onClick={() => setMounted(3)}>Gắn 3 người nghe</Button>
        <Button onClick={() => setMounted(0)}>Gỡ hết</Button>
        <span className={s.out} data-part="sse-status">{st}</span>
      </div>
      {st === "degraded" && <p role="status">Bạn đang mở nhiều cửa sổ; cập nhật tự động tạm dừng.</p>}
      {Array.from({ length: mounted }, (_, i) => (
        <Listener key={i} n={i} type="msg" />
      ))}
    </section>
  );
}

function Job() {
  const [id, setId] = useState("");
  const j = useJob(id || null);
  return (
    <section className={s.block} data-part="job" aria-label="Việc dài">
      <h2 className="ep-section-title">Việc dài</h2>
      <Field label="Mã việc">{(fid) => <Input id={fid} value={id} onChange={(e) => setId(e.target.value)} />}</Field>
      <p className={s.out} data-part="job-state">{`${j.status}:${j.progress}${j.error ? `:${j.error}` : ""}`}</p>
    </section>
  );
}

function Autosave() {
  const d = useAutosaveDraft("dev.note");
  const [thrown, setThrown] = useState("");
  return (
    <section className={s.block} data-part="autosave" aria-label="Nháp tự lưu">
      <h2 className="ep-section-title">Nháp tự lưu</h2>
      <Field label="Ghi chú">{(id) => <Textarea id={id} rows={2} value={d.value} onChange={(e) => d.setValue(e.target.value)} />}</Field>
      <div className={s.row}>
        <span className={s.out} data-part="draft-status">{d.status}</span>
        <Button onClick={() => d.clear()}>Gửi xong</Button>
        <Button
          onClick={() => {
            try {
              // eslint-disable-next-line react-hooks/rules-of-hooks -- cố ý gọi sai để kiểm tra từ chối khoá bí mật
              useAutosaveDraft("llm.apiKey");
            } catch (e) {
              setThrown(String((e as Error).message));
            }
          }}
        >
          Thử khoá bí mật
        </Button>
        <span className={s.out} data-part="draft-throw">{thrown}</span>
      </div>
    </section>
  );
}

function Undo() {
  const [on, setOn] = useState(false);
  const u = useUndoableAction<boolean>({
    apply: (v) => setOn(v),
    rollback: (v) => setOn(!v),
    commit: (v) => apiClient.put("/flag", { on: v }),
    compensate: (v) => apiClient.put("/flag", { on: !v }),
    label: (v) => (v ? "Đã bật" : "Đã tắt"),
  });
  return (
    <section className={s.block} data-part="undo" aria-label="Hoàn tác">
      <h2 className="ep-section-title">Hoàn tác tại chỗ</h2>
      <div className={s.row}>
        <span className={s.out} data-part="undo-value">{on ? "bật" : "tắt"}</span>
        <Button onClick={() => void u.run(!on)}>Đổi</Button>
      </div>
      {u.node}
    </section>
  );
}

function Offline() {
  const [v, setV] = useState("");
  return (
    <section className={s.block} data-part="offline" aria-label="Mất mạng">
      <h2 className="ep-section-title">Mất mạng</h2>
      <Field label="Ô nhập">{(id) => <Input id={id} value={v} onChange={(e) => setV(e.target.value)} />}</Field>
      <div className={s.row}>
        <Button onClick={() => void apiClient.get("/ping").catch(() => {})}>Gọi /ping</Button>
      </div>
    </section>
  );
}

function BellDemo() {
  const seed = (n: number): NotificationItem => ({ id: `n${n}`, title: `Thông báo số ${n}`, context: "Hộp thư hỗ trợ", when: "2 phút trước", href: "/dev/data", read: false });
  const [items, setItems] = useState<NotificationItem[]>([seed(1), seed(2)]);
  const unread = items.filter((i) => !i.read).length;
  return (
    <section className={s.block} data-part="bell-demo" aria-label="Chuông thông báo">
      <h2 className="ep-section-title">Chuông thông báo</h2>
      <div className={s.row}>
        <NotificationPopover items={items} unread={unread} onRead={(id) => setItems((xs) => xs.map((x) => (x.id === id ? { ...x, read: true } : x)))} />
        <Button onClick={() => setItems((xs) => xs.map((x) => ({ ...x, read: true })))}>Đọc hết</Button>
        <Button onClick={() => setItems((xs) => [seed(xs.length + 1), ...xs])}>Thêm một thông báo</Button>
        <Button onClick={() => setItems([])}>Xoá hết</Button>
        <Button onClick={() => setItems((xs) => [...xs])}>Hiển thị lại</Button>
      </div>
    </section>
  );
}

function PageStateDemo({ staff }: { staff: boolean }) {
  const q = useQuery({ queryKey: ["ps"], queryFn: async () => (await apiClient.get<{ items: string[] }>("/ps")).data });
  return (
    <section className={s.block} data-part="pagestate" aria-label="PageState">
      <h2 className="ep-section-title">PageState</h2>
      <PageState query={q} isEmpty={(d) => (d as { items: string[] }).items.length === 0} showTechnical={staff} empty={<EmptyState title="Chưa có gì">Không có mục nào.</EmptyState>}>
        <p data-part="ps-ok">{q.data?.items.join(", ")}</p>
      </PageState>
    </section>
  );
}

function Token() {
  const [t, setT] = useState("");
  return (
    <section className={s.block} data-part="token" aria-label="Token">
      <h2 className="ep-section-title">Token</h2>
      <Field label="Dán token">{(id) => <Input id={id} type="password" value={t} onChange={(e) => setT(e.target.value)} />}</Field>
      <div className={s.row}>
        <Button onClick={() => { tokenStore.set(t); setT(""); }}>Dùng</Button>
      </div>
    </section>
  );
}

export default function DevData() {
  const staff = useSearchParams().get("as") !== "student";
  useEffect(() => {
    window.__events = [];
    const f = () => window.__events.push("auth:expired");
    window.addEventListener("auth:expired", f);
    window.__ep = { apiClient, ApiError, fieldErrors, tokenStore, netStatus, resetApiClientState, newIdempotencyKey };
    return () => window.removeEventListener("auth:expired", f);
  }, []);
  return (
    <div className={s.wrap}>
      <h1 className="ep-page-title">Thử lớp dữ liệu</h1>
      <ErrDisplay staff={staff} />
      <FieldErrors />
      <CursorList />
      <Idem />
      <Sse />
      <Job />
      <Autosave />
      <Undo />
      <Offline />
      <BellDemo />
      <PageStateDemo staff={staff} />
      <Token />
    </div>
  );
}
