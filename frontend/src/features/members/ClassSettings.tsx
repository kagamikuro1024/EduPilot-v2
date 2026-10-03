"use client";

import { useQuery, useQueryClient } from "@tanstack/react-query";
import { useState } from "react";
import { ApiError, apiClient, fieldErrors } from "@/shared/data";
import { Button, ButtonLink, ConfirmIrreversible, EmptyState, Field, InlineNotice, Input, Page, PageHeader, PageState, Section, Skeleton, Switch } from "@/shared/ui";
import s from "./ClassSettings.module.css";
import { classKey, useClassCourse, type JoinInfo } from "./classApi";

const ICT_DAY = new Intl.DateTimeFormat("en-CA", { timeZone: "Asia/Ho_Chi_Minh", year: "numeric", month: "2-digit", day: "2-digit" });

/** Mã tham gia và cài đặt tham gia của lớp. Giảng viên sửa; TA chỉ xem (không thấy nút tạo lại mã). */
export function ClassSettings() {
  const cc = useClassCourse();
  if (cc.state === "loading") return <Page><PageHeader title="Mã và cài đặt tham gia" /><Skeleton lines={4} /></Page>;
  if (cc.state === "none") {
    return (
      <Page>
        <PageHeader title="Mã và cài đặt tham gia" />
        <EmptyState title="Chưa chọn lớp">Chọn một lớp ở thanh trên. Quản trị viên mở lớp ở mục Lớp học.</EmptyState>
      </Page>
    );
  }
  return <Settings courseId={cc.course.id} code={cc.course.class_code} canManage={cc.canManage} />;
}

function Settings({ courseId, code, canManage }: { courseId: string; code: string; canManage: boolean }) {
  const qc = useQueryClient();
  const key = classKey(courseId, "join-code");
  const q = useQuery({ queryKey: key, staleTime: 0, queryFn: async ({ signal }) => (await apiClient.get<JoinInfo>(`/courses/${courseId}/join-code`, { signal })).data });
  const info = q.data;

  const [saved, setSaved] = useState(false);
  const [copied, setCopied] = useState<string | null>(null);
  const [confirm, setConfirm] = useState(false);
  const [regen, setRegen] = useState<{ loading: boolean; error?: string }>({ loading: false });

  async function copy(text: string, what: string) {
    try {
      await navigator.clipboard.writeText(text);
      setCopied(what);
    } catch {
      setCopied(null);
    }
  }

  async function regenerate() {
    setRegen({ loading: true });
    try {
      await apiClient.post(`/courses/${courseId}/join-code/regenerate`);
      setRegen({ loading: false });
      setConfirm(false);
      await qc.invalidateQueries({ queryKey: key });
    } catch (e) {
      setRegen({ loading: false, error: e instanceof ApiError ? e.userMessage : "Chưa tạo lại được. Hãy thử lại." });
    }
  }

  return (
    <Page>
      <PageHeader
        title="Mã và cài đặt tham gia"
        description={`Lớp ${code}. Chia sẻ mã hoặc liên kết cho sinh viên; bạn quyết định ai được vào.`}
        actions={<ButtonLink href="/class/members">Thành viên lớp</ButtonLink>}
      />
      <PageState query={{ isPending: q.isPending, isError: q.isError, error: q.error, data: q.data, refetch: q.refetch }} showTechnical>
        {info && (
          <>
            <Section title="Mã tham gia">
              <div className={s.codeRow}>
                <p className={s.code} data-part="join-code">{info.join_code}</p>
                <div className={s.actions}>
                  <Button onClick={() => void copy(info.join_code, "code")}>Sao chép mã</Button>
                  <Button onClick={() => void copy(info.join_url, "url")}>Sao chép liên kết</Button>
                  {canManage && <Button variant="ghost" onClick={() => { setRegen({ loading: false }); setConfirm(true); }}>Tạo lại mã</Button>}
                </div>
                {copied && <p className={s.readonly} role="status">Đã sao chép</p>}
                <p className={s.readonly}>{info.active_students} sinh viên đang học{info.pending > 0 ? ` · ${info.pending} chờ duyệt` : ""}.</p>
              </div>
            </Section>
            <Section title="Cài đặt tham gia">
              {/* key theo version: bản mới từ máy chủ (lưu xong, “Dùng bản mới”, tạo lại mã) nạp lại form từ đầu */}
              <SettingsForm
                key={info.version}
                courseId={courseId}
                info={info}
                canManage={canManage}
                saved={saved}
                onSaved={(data) => {
                  setSaved(true);
                  qc.setQueryData(key, data);
                }}
                onReload={() => {
                  setSaved(false);
                  void qc.invalidateQueries({ queryKey: key });
                }}
              />
            </Section>
            {canManage && <ShareSection courseId={courseId} />}
          </>
        )}
      </PageState>

      <ConfirmIrreversible
        open={confirm}
        onClose={() => setConfirm(false)}
        onConfirm={() => void regenerate()}
        title={`Tạo lại mã cho lớp ${code}?`}
        consequence={`Mã cũ sẽ ngừng hoạt động ngay. ${info?.active_students ?? 0} sinh viên đang ở trong lớp không bị ảnh hưởng.`}
        confirmLabel="Tạo lại mã"
        loading={regen.loading}
        error={regen.error}
      />
    </Page>
  );
}

function SettingsForm({ courseId, info, canManage, saved, onSaved, onReload }: { courseId: string; info: JoinInfo; canManage: boolean; saved: boolean; onSaved: (i: JoinInfo) => void; onReload: () => void }) {
  const [enabled, setEnabled] = useState(info.enabled);
  const [approval, setApproval] = useState(info.require_approval);
  const [expires, setExpires] = useState(info.expires_at ? ICT_DAY.format(new Date(info.expires_at)) : "");
  const [capacity, setCapacity] = useState(info.capacity === null ? "" : String(info.capacity));
  const [domain, setDomain] = useState(info.allowed_email_domain ?? "");
  const [pending, setPending] = useState(false);
  const [error, setError] = useState<unknown>(null);
  const [conflict, setConflict] = useState<JoinInfo | null>(null);

  const fe = error ? fieldErrors(error) : {};
  const general = error && Object.keys(fe).length === 0 ? (error instanceof ApiError ? error.userMessage : "Chưa lưu được. Hãy thử lại.") : null;

  async function save(version: number) {
    if (pending || !canManage) return;
    setPending(true);
    setError(null);
    setConflict(null);
    try {
      const { data } = await apiClient.put<JoinInfo>(`/courses/${courseId}/join-settings`, {
        enabled,
        require_approval: approval,
        expires_at: expires ? `${expires}T23:59:59+07:00` : null,
        capacity: capacity.trim() === "" ? null : Number(capacity),
        allowed_email_domain: domain.trim() === "" ? null : domain.trim(),
        version,
      });
      onSaved(data);
    } catch (err) {
      if (err instanceof ApiError && err.code === "VERSION_CONFLICT" && err.conflict) setConflict(err.conflict.current as JoinInfo);
      else setError(err);
    } finally {
      setPending(false);
    }
  }

  return (
    <form
      className={s.form}
      noValidate
      onSubmit={(e) => {
        e.preventDefault();
        void save(info.version);
      }}
    >
      <Switch label="Cho phép tham gia bằng mã" checked={enabled} onChange={setEnabled} disabled={!canManage} />
      <Switch label="Cần giảng viên duyệt" description="Sinh viên vào bằng mã sẽ chờ bạn duyệt." checked={approval} onChange={setApproval} disabled={!canManage} />
      <div className={s.fields}>
        <Field label="Hết hạn" helper="Để trống nếu không giới hạn." error={fe.expires_at}>
          {(id, d) => <Input id={id} aria-describedby={d} invalid={!!fe.expires_at} type="date" value={expires} disabled={!canManage} onChange={(e) => setExpires(e.target.value)} />}
        </Field>
        <Field label="Giới hạn sĩ số" helper="Để trống nếu không giới hạn." error={fe.capacity}>
          {(id, d) => <Input id={id} aria-describedby={d} invalid={!!fe.capacity} inputMode="numeric" value={capacity} disabled={!canManage} onChange={(e) => setCapacity(e.target.value)} />}
        </Field>
        <Field label="Giới hạn tên miền email" helper="Ví dụ ptit.edu.vn. Để trống nếu không giới hạn." error={fe.allowed_email_domain}>
          {(id, d) => <Input id={id} aria-describedby={d} invalid={!!fe.allowed_email_domain} autoCapitalize="none" spellCheck={false} value={domain} disabled={!canManage} onChange={(e) => setDomain(e.target.value)} />}
        </Field>
      </div>
      {general && <InlineNotice tone="danger" compact>{general}</InlineNotice>}
      {conflict && (
        <InlineNotice
          tone="warning"
          compact
          action={
            <>
              <Button size="sm" onClick={() => void save(conflict.version)}>Giữ thay đổi của tôi</Button>
              <Button size="sm" variant="ghost" onClick={onReload}>Dùng bản mới</Button>
            </>
          }
        >
          Cài đặt này vừa được người khác đổi. Giữ thay đổi của bạn hay dùng bản mới?
        </InlineNotice>
      )}
      {canManage ? (
        <div className={s.actions}>
          <Button type="submit" variant="primary" loading={pending}>Lưu cài đặt</Button>
          {saved && <span className={s.readonly} role="status">Đã lưu cài đặt.</span>}
        </div>
      ) : (
        <p className={s.readonly}>Chỉ giảng viên được thay đổi.</p>
      )}
    </form>
  );
}

type ShareSource = { id: string; class_code: string; name: string; semester: string; documents: number };

/** "Dùng lại nội dung từ lớp khác": chỉ hiện khi có lớp cùng học phần mà giảng viên cũng dạy. Mỗi nguồn một nút phụ; kết quả hiện tại chỗ. */
function ShareSection({ courseId }: { courseId: string }) {
  const sources = useQuery({
    queryKey: classKey(courseId, "share-sources"),
    queryFn: async ({ signal }) => (await apiClient.get<{ items: ShareSource[] }>(`/courses/${courseId}/share-sources`, { signal })).data.items,
  });
  const [busy, setBusy] = useState<string | null>(null);
  const [results, setResults] = useState<Record<string, string>>({});

  async function share(src: ShareSource) {
    setBusy(src.id);
    try {
      const r = await apiClient.post<{ shared: { documents: number } }>(`/courses/${courseId}/share-from`, { source_course_id: src.id, what: ["documents"] });
      const n = r.data.shared.documents;
      setResults((m) => ({ ...m, [src.id]: n > 0 ? `Đã dùng lại ${n} tài liệu từ lớp ${src.class_code}.` : `Lớp ${src.class_code} chưa có tài liệu để dùng lại.` }));
    } catch (e) {
      setResults((m) => ({ ...m, [src.id]: e instanceof ApiError ? e.userMessage : "Chưa dùng lại được. Hãy thử lại." }));
    } finally {
      setBusy(null);
    }
  }

  if (!sources.data || sources.data.length === 0) return null;
  return (
    <Section title="Dùng lại nội dung từ lớp khác">
      <ul className={s.sources}>
        {sources.data.map((src) => (
          <li key={src.id} className={s.source}>
            <span>
              <b>Lớp {src.class_code}</b> · {src.name} · {src.semester} · {src.documents} tài liệu
            </span>
            <Button size="sm" onClick={() => void share(src)} disabled={busy === src.id}>Dùng lại tài liệu</Button>
            {results[src.id] && <span className={s.readonly} role="status">{results[src.id]}</span>}
          </li>
        ))}
      </ul>
    </Section>
  );
}
