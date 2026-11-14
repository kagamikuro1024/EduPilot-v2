"use client";

import { useQueryClient } from "@tanstack/react-query";
import { useState } from "react";
import { ApiError, ApiErrorNotice, fieldErrors, newIdempotencyKey, useJob } from "@/shared/data";
import { Button, DefinitionList, EmptyState, Field, InlineNotice, Input, Page, PageHeader, Panel, PanelSection, PasswordInput, Select, Skeleton, StatusText, Switch } from "@/shared/ui";
import { useSession } from "@/shared/session/session";
import { dayLabel, fmtDM, fmtHM } from "@/features/calendar/calendarApi";
import {
  ERROR_SHORT, IMAP_TEST_TEXT, SECURITY_LABEL, STATUS_TEXT, STATUS_TONE, TEAMS_TEST_TEXT,
  integrationKey, saveImap, saveTeams, syncImapNow, testImap, testSmtp, testTeams, useAdminCourses, useImap, useSmtp, useTeams,
  type ImapConfig, type IntegrationView, type Security, type SmtpConfig, type TeamsConfig,
} from "./integrationsApi";
import s from "./settings.module.css";

const CONFLICT = "Cấu hình vừa được người khác đổi";
const SECURITIES: Security[] = ["TLS", "STARTTLS", "NONE"];
/** "09:12 23/10" */
const when = (iso: string) => `${fmtHM(iso)} ${fmtDM(dayLabel(iso))}`;

/** `/settings/integrations` (US-P7-05, US-P7-11): ba đường thư và Teams của MỘT lớp. Admin sửa, giảng viên xem và đồng bộ. */
export function IntegrationsSettings() {
  const { role, realCourseId } = useSession();
  const admin = role === "admin";
  const courses = useAdminCourses(admin);
  const [picked, setPicked] = useState("");
  const list = courses.data ?? [];
  const courseId = admin ? picked || list[0]?.id || "" : realCourseId && realCourseId !== "all" ? realCourseId : "";

  return (
    <Page kind="read">
      <PageHeader title="Tích hợp" />
      {admin && list.length > 0 && (
        <div className={s.picker}>
          <Field label="Lớp">
            {(id) => (
              <Select id={id} value={courseId} onChange={(e) => setPicked(e.target.value)} data-part="course-picker">
                {list.map((c) => (
                  <option key={c.id} value={c.id}>
                    {c.class_code} · {c.name}
                  </option>
                ))}
              </Select>
            )}
          </Field>
        </div>
      )}
      {admin && courses.isPending ? (
        <Panel>
          <Skeleton lines={8} />
        </Panel>
      ) : courseId ? (
        <Blocks key={courseId} courseId={courseId} admin={admin} />
      ) : (
        <Panel>
          <EmptyState title={admin ? "Chưa có lớp nào." : "Chọn một lớp ở thanh trên."} />
        </Panel>
      )}
    </Page>
  );
}

function Blocks({ courseId, admin }: { courseId: string; admin: boolean }) {
  const smtp = useSmtp(courseId);
  const imap = useImap(courseId);
  const teams = useTeams(courseId);

  if (smtp.isPending || imap.isPending || teams.isPending) {
    return (
      <Panel>
        <Skeleton lines={10} />
      </Panel>
    );
  }
  const failed = [smtp, imap, teams].find((q) => q.isError);
  if (failed || !smtp.data || !imap.data || !teams.data) {
    return (
      <Panel>
        <ApiErrorNotice
          error={failed?.error}
          title="Chưa đọc được trạng thái tích hợp."
          context="Các kết nối đang có vẫn chạy."
          showTechnical
          onRetry={() => void failed?.refetch()}
        />
      </Panel>
    );
  }

  return (
    <Panel>
      <PanelSection title="Thư gửi (SMTP)" action={<Status v={smtp.data} />}>
        <SmtpBlock courseId={courseId} v={smtp.data} admin={admin} />
      </PanelSection>
      <PanelSection title="Thư nhận (IMAP)" action={<Status v={imap.data} />}>
        <ImapBlock courseId={courseId} v={imap.data} admin={admin} />
      </PanelSection>
      <PanelSection
        title={
          <>
            Teams
            {teams.data.sandbox && (
              <span className={s.tag} data-part="teams-sandbox">
                Môi trường giả lập
              </span>
            )}
          </>
        }
        action={teams.data.available === false ? <span data-part="integration-status"><StatusText tone="neutral">Chưa bật</StatusText></span> : <Status v={teams.data} />}
      >
        <TeamsBlock courseId={courseId} v={teams.data} admin={admin} />
      </PanelSection>
    </Panel>
  );
}

function Status({ v }: { v: IntegrationView<unknown> }) {
  return (
    <span data-part="integration-status">
      <StatusText tone={STATUS_TONE[v.status]}>{STATUS_TEXT[v.status]}</StatusText>
    </span>
  );
}

/** Dòng hỏng đang kéo dài: "Không kết nối được từ 09:12 23/10"; đang tạm dừng vì khoá sai thì nói việc cần làm. */
function Trouble({ v }: { v: IntegrationView<unknown> }) {
  if (v.status === "PAUSED") return <p className={s.trouble} data-part="integration-error">Cần nhập lại mật khẩu</p>;
  if (v.status !== "ERROR" || !v.last_error) return null;
  const head = ERROR_SHORT[v.last_error] ?? v.last_error;
  return (
    <p className={s.trouble} data-part="integration-error">
      {v.error_since ? `${head} từ ${when(v.error_since)}` : head}
    </p>
  );
}

// ---- thư gửi (chỉ đọc) ----------------------------------------------------------------------------------------------------

function SmtpBlock({ courseId, v, admin }: { courseId: string; v: IntegrationView<SmtpConfig>; admin: boolean }) {
  const [sent, setSent] = useState<string | null>(null);
  const [busy, setBusy] = useState(false);
  const c = v.config;

  const send = async () => {
    setBusy(true);
    setSent(null);
    try {
      await testSmtp(courseId);
      setSent("Đã gửi thư thử tới địa chỉ của bạn");
    } catch (e) {
      setSent(e instanceof ApiError ? e.userMessage : "Chưa gửi được thư thử.");
    } finally {
      setBusy(false);
    }
  };

  return (
    <div data-part="integration-block" data-kind="smtp">
      <DefinitionList
        items={[
          { term: "Máy chủ", value: c ? `${c.host}:${c.port}` : "—" },
          { term: "Bảo mật", value: c ? SECURITY_LABEL[c.security] ?? c.security : "—" },
          { term: "Địa chỉ gửi", value: c?.from ?? "—" },
        ]}
      />
      <Trouble v={v} />
      {admin && (
        <div className={s.inlineFormActions}>
          <Button size="sm" loading={busy} onClick={() => void send()}>
            Gửi thư thử
          </Button>
        </div>
      )}
      {sent && (
        <InlineNotice tone="info" compact>
          {sent}
        </InlineNotice>
      )}
    </div>
  );
}

// ---- thư nhận ---------------------------------------------------------------------------------------------------------------

type ImapForm = { host: string; port: string; security: Security; username: string; mailbox: string; poll: string; enabled: boolean };
const imapForm = (v: IntegrationView<ImapConfig>): ImapForm => ({
  host: v.config?.host ?? "",
  port: String(v.config?.port ?? 993),
  security: v.config?.security ?? "TLS",
  username: v.config?.username ?? "",
  mailbox: v.config?.mailbox ?? "INBOX",
  poll: String(v.config?.poll_seconds ?? 60),
  enabled: v.enabled,
});

type SyncResult = { received: number; created: number; matched: number; unmatched: number; ignored: number; failed: number };

function ImapBlock({ courseId, v, admin }: { courseId: string; v: IntegrationView<ImapConfig>; admin: boolean }) {
  const qc = useQueryClient();
  const [f, setF] = useState<ImapForm>(() => imapForm(v));
  const [password, setPassword] = useState("");
  const [seen, setSeen] = useState(v.version);
  const [busy, setBusy] = useState(false);
  const [notice, setNotice] = useState<string | null>(null);
  const [test, setTest] = useState<{ ok: boolean; text: string } | null>(null);
  const [err, setErr] = useState<ApiError | null>(null);
  const [job, setJob] = useState<string | null>(null);
  const [wait, setWait] = useState(0);
  const sync = useJob(job);

  // Máy chủ có bản mới (vừa lưu, hoặc vừa tải lại sau 409): hiện đúng giá trị đang có, xoá ô mật khẩu.
  if (seen !== v.version) {
    setSeen(v.version);
    setF(imapForm(v));
    setPassword("");
  }

  const fe = err ? fieldErrors(err) : {};
  const canEdit = admin && !v.read_only;
  const reload = () => qc.invalidateQueries({ queryKey: integrationKey(courseId, "imap") });

  async function run(op: () => Promise<void>) {
    setBusy(true);
    try {
      await op();
    } catch (e) {
      const x = e instanceof ApiError ? e : new ApiError({ status: 0, code: "NETWORK" });
      setErr(x);
      if (x.code === "VERSION_CONFLICT") {
        setNotice(CONFLICT);
        await reload();
      } else if (x.code !== "VALIDATION_FAILED") {
        setNotice(x.userMessage);
      }
    } finally {
      setBusy(false);
    }
  }

  const save = () =>
    void run(async () => {
      setErr(null);
      setNotice(null);
      await saveImap(courseId, {
        host: f.host.trim(),
        port: Number(f.port),
        security: f.security,
        username: f.username.trim(),
        ...(password ? { password } : {}),
        mailbox: f.mailbox.trim() || "INBOX",
        poll_seconds: Number(f.poll),
        enabled: f.enabled,
        version: v.version,
      });
      setPassword("");
      await reload();
    });

  const check = () =>
    void run(async () => {
      setErr(null);
      setTest(null);
      const out = await testImap(courseId);
      setTest(out.ok ? { ok: true, text: "Kết nối tốt" } : { ok: false, text: IMAP_TEST_TEXT[out.kind ?? ""] ?? "Chưa kết nối được" });
      await reload();
    });

  const syncNow = () =>
    void run(async () => {
      setErr(null);
      setNotice(null);
      setWait(0);
      try {
        const out = await syncImapNow(courseId, newIdempotencyKey());
        setJob(out.job_id);
      } catch (e) {
        if (e instanceof ApiError && e.status === 429) {
          setWait(e.retryAfter ?? 30);
          return;
        }
        throw e;
      }
    });

  const result = sync.status === "SUCCEEDED" ? (sync.result as SyncResult | undefined) : undefined;
  const running = Boolean(job) && sync.status !== "SUCCEEDED" && sync.status !== "FAILED";

  return (
    <div data-part="integration-block" data-kind="imap">
      {canEdit ? (
        <div className={s.grid}>
          <Field label="Máy chủ" error={fe.host}>
            {(id) => <Input id={id} value={f.host} invalid={Boolean(fe.host)} placeholder="imap.ptit.edu.vn" onChange={(e) => setF({ ...f, host: e.target.value })} />}
          </Field>
          <Field label="Cổng" error={fe.port}>
            {(id) => <Input id={id} value={f.port} inputMode="numeric" invalid={Boolean(fe.port)} onChange={(e) => setF({ ...f, port: e.target.value })} />}
          </Field>
          <Field label="Bảo mật" error={fe.security}>
            {(id) => (
              <Select id={id} value={f.security} onChange={(e) => setF({ ...f, security: e.target.value as Security })}>
                {SECURITIES.map((x) => (
                  <option key={x} value={x}>
                    {SECURITY_LABEL[x]}
                  </option>
                ))}
              </Select>
            )}
          </Field>
          <Field label="Tài khoản" error={fe.username}>
            {(id) => <Input id={id} value={f.username} autoComplete="off" invalid={Boolean(fe.username)} onChange={(e) => setF({ ...f, username: e.target.value })} />}
          </Field>
          <Field label="Mật khẩu" error={fe.password}>
            {(id) => (
              <PasswordInput
                id={id}
                value={password}
                autoComplete="new-password"
                spellCheck={false}
                placeholder="Để trống = giữ cũ"
                invalid={Boolean(fe.password)}
                onChange={(e) => setPassword(e.target.value)}
              />
            )}
          </Field>
          <Field label="Hộp thư" error={fe.mailbox}>
            {(id) => <Input id={id} value={f.mailbox} invalid={Boolean(fe.mailbox)} onChange={(e) => setF({ ...f, mailbox: e.target.value })} />}
          </Field>
          <Field label="Chu kỳ (giây)" error={fe.poll_seconds}>
            {(id) => <Input id={id} value={f.poll} inputMode="numeric" invalid={Boolean(fe.poll_seconds)} onChange={(e) => setF({ ...f, poll: e.target.value })} />}
          </Field>
          <Switch label="Bật" checked={f.enabled} onChange={(next) => setF({ ...f, enabled: next })} />
        </div>
      ) : (
        <DefinitionList
          items={[
            { term: "Máy chủ", value: v.config ? `${v.config.host}:${v.config.port}` : "—" },
            { term: "Bảo mật", value: v.config ? SECURITY_LABEL[v.config.security] ?? v.config.security : "—" },
            { term: "Tài khoản", value: v.config?.username ?? "—" },
            { term: "Hộp thư", value: v.config?.mailbox ?? "—" },
            { term: "Chu kỳ", value: v.config ? `${v.config.poll_seconds} giây` : "—" },
            { term: "Bật", value: v.enabled ? "Có" : "Không" },
          ]}
        />
      )}

      <Trouble v={v} />
      {v.last_sync_at && <p className={s.meta} data-part="imap-last-sync">{`Lần đồng bộ cuối ${when(v.last_sync_at)}`}</p>}

      <div className={s.inlineFormActions}>
        {canEdit && (
          <Button size="sm" loading={busy} onClick={save}>
            Lưu
          </Button>
        )}
        {admin && (
          <Button size="sm" disabled={busy || !v.configured} title={v.configured ? undefined : "Chưa có cấu hình để thử"} onClick={check}>
            Test kết nối
          </Button>
        )}
        {!admin && (
          <Button size="sm" disabled={busy || wait > 0 || running} title={wait > 0 ? `Chờ ${wait} giây nữa` : undefined} onClick={syncNow}>
            Đồng bộ ngay
          </Button>
        )}
      </div>

      {notice && (
        <InlineNotice tone="warning" compact action={<Button size="sm" onClick={() => setNotice(null)}>Đã hiểu</Button>}>
          {notice}
        </InlineNotice>
      )}
      {test && (
        <InlineNotice tone={test.ok ? "info" : "warning"} compact>
          <span data-part="imap-test">{test.text}</span>
        </InlineNotice>
      )}
      {running && <p className={s.meta} data-part="sync-progress">{`Đang đồng bộ ${sync.progress}%`}</p>}
      {result && (
        <p className={s.meta} data-part="sync-result">
          {`Nhận ${result.received} · tạo ${result.created} · khớp ${result.matched} · chưa khớp ${result.unmatched} · bỏ qua ${result.ignored} · lỗi ${result.failed}`}
        </p>
      )}
    </div>
  );
}

// ---- Teams ------------------------------------------------------------------------------------------------------------------

function TeamsBlock({ courseId, v, admin }: { courseId: string; v: IntegrationView<TeamsConfig>; admin: boolean }) {
  const qc = useQueryClient();
  const [tenant, setTenant] = useState(v.config?.tenant_id ?? "");
  const [client, setClient] = useState(v.config?.client_id ?? "");
  const [secret, setSecret] = useState("");
  const [enabled, setEnabled] = useState(v.enabled);
  const [seen, setSeen] = useState(v.version);
  const [busy, setBusy] = useState(false);
  const [notice, setNotice] = useState<string | null>(null);
  const [test, setTest] = useState<{ ok: boolean; text: string } | null>(null);
  const [err, setErr] = useState<ApiError | null>(null);

  if (seen !== v.version) {
    setSeen(v.version);
    setTenant(v.config?.tenant_id ?? "");
    setClient(v.config?.client_id ?? "");
    setEnabled(v.enabled);
    setSecret("");
  }

  const fe = err ? fieldErrors(err) : {};
  const off = v.available === false;
  const canEdit = admin && !v.read_only;
  const reload = () => qc.invalidateQueries({ queryKey: integrationKey(courseId, "teams") });

  async function run(op: () => Promise<void>) {
    setBusy(true);
    try {
      await op();
    } catch (e) {
      const x = e instanceof ApiError ? e : new ApiError({ status: 0, code: "NETWORK" });
      setErr(x);
      if (x.code === "VERSION_CONFLICT") {
        setNotice(CONFLICT);
        await reload();
      } else if (x.code !== "VALIDATION_FAILED") {
        setNotice(x.userMessage);
      }
    } finally {
      setBusy(false);
    }
  }

  const save = () =>
    void run(async () => {
      setErr(null);
      setNotice(null);
      await saveTeams(courseId, { tenant_id: tenant.trim(), client_id: client.trim(), ...(secret ? { client_secret: secret } : {}), enabled, version: v.version });
      setSecret("");
      await reload();
    });

  const check = () =>
    void run(async () => {
      setErr(null);
      setTest(null);
      const out = await testTeams(courseId);
      setTest(out.ok ? { ok: true, text: "Kết nối tốt" } : { ok: false, text: TEAMS_TEST_TEXT[out.kind ?? ""] ?? "Chưa kết nối được" });
      await reload();
    });

  return (
    <div data-part="integration-block" data-kind="teams">
      {canEdit ? (
        <div className={s.grid}>
          <Field label="Tenant" error={fe.tenant_id}>
            {(id) => <Input id={id} value={tenant} invalid={Boolean(fe.tenant_id)} onChange={(e) => setTenant(e.target.value)} />}
          </Field>
          <Field label="Client ID" error={fe.client_id}>
            {(id) => <Input id={id} value={client} invalid={Boolean(fe.client_id)} onChange={(e) => setClient(e.target.value)} />}
          </Field>
          <Field label="Client secret" error={fe.client_secret}>
            {(id) => (
              <PasswordInput
                id={id}
                value={secret}
                autoComplete="new-password"
                spellCheck={false}
                placeholder="Để trống = giữ cũ"
                invalid={Boolean(fe.client_secret)}
                onChange={(e) => setSecret(e.target.value)}
              />
            )}
          </Field>
          <Switch label="Bật" checked={enabled} onChange={setEnabled} />
        </div>
      ) : (
        <DefinitionList
          items={[
            { term: "Tenant", value: v.config?.tenant_id ?? "—" },
            { term: "Client ID", value: v.config?.client_id ?? "—" },
            { term: "Bật", value: v.enabled ? "Có" : "Không" },
          ]}
        />
      )}

      <Trouble v={v} />

      {(canEdit || admin) && (
        <div className={s.inlineFormActions}>
          {canEdit && (
            <Button size="sm" loading={busy} disabled={off} title={off ? "Trường chưa bật Teams" : undefined} onClick={save}>
              Lưu
            </Button>
          )}
          {admin && (
            <Button size="sm" disabled={busy || off || !v.configured} title={off ? "Trường chưa bật Teams" : v.configured ? undefined : "Chưa có cấu hình để thử"} onClick={check}>
              Kiểm tra kết nối
            </Button>
          )}
        </div>
      )}

      {notice && (
        <InlineNotice tone="warning" compact action={<Button size="sm" onClick={() => setNotice(null)}>Đã hiểu</Button>}>
          {notice}
        </InlineNotice>
      )}
      {test && (
        <InlineNotice tone={test.ok ? "info" : "warning"} compact>
          <span data-part="teams-test">{test.text}</span>
        </InlineNotice>
      )}
    </div>
  );
}
