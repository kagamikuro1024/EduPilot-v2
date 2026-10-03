"use client";

import { useQueryClient } from "@tanstack/react-query";
import { useState } from "react";
import { ApiError, ApiErrorNotice, apiClient } from "@/shared/data";
import { Button, ConfirmIrreversible, InlineNotice, OverflowMenu, StatusText, Switch } from "@/shared/ui";
import { ConflictNotice } from "./ConflictNotice";
import { ProviderForm, type FormMode } from "./ProviderForm";
import { KEY, useSaver, type Provider, type TestResult } from "./api";
import { TYPE_LABEL, providerStatus, testMessage } from "./labels";
import s from "./llm.module.css";

type Test = { pending: boolean; result?: TestResult; error?: ApiError };

/** Phần 1 — Kết nối nhà cung cấp (US-P1-05 AC2–AC4). Giảng viên chỉ xem. */
export function ProviderSection({ providers, canEdit, envFallback, initialAdd = false }: { providers: Provider[]; canEdit: boolean; envFallback: { active: boolean; providers: string[] }; /** mở sẵn form thêm (từ nút ở phần 2 khi chưa có nhà cung cấp) */ initialAdd?: boolean }) {
  const qc = useQueryClient();
  const [form, setForm] = useState<{ mode: FormMode; id?: string } | null>(initialAdd && canEdit ? { mode: "add" } : null);
  const [del, setDel] = useState<Provider | null>(null);
  const [delError, setDelError] = useState<ApiError | null>(null);
  const [deleting, setDeleting] = useState(false);
  const [tests, setTests] = useState<Record<string, Test>>({});
  const saver = useSaver(qc, [KEY.providers, KEY.routes]);
  const [saveFor, setSaveFor] = useState<string | null>(null);

  async function runTest(p: Provider) {
    setTests((t) => ({ ...t, [p.id]: { pending: true } }));
    try {
      const r = await apiClient.post<TestResult>(`/admin/llm/providers/${p.id}/test`, {});
      setTests((t) => ({ ...t, [p.id]: { pending: false, result: r.data } }));
      void qc.invalidateQueries({ queryKey: KEY.providers });
    } catch (e) {
      setTests((t) => ({ ...t, [p.id]: { pending: false, error: e instanceof ApiError ? e : new ApiError({ status: 0, code: "NETWORK" }) } }));
    }
  }

  function toggle(p: Provider, enabled: boolean) {
    setSaveFor(p.id);
    void saver.run((v) => apiClient.put(`/admin/llm/providers/${p.id}`, { type: p.type, name: p.name, base_url: p.base_url, enabled, rpm_limit: p.rpm_limit, tpm_limit: p.tpm_limit, version: v ?? p.version }));
  }

  async function remove() {
    if (!del) return;
    setDeleting(true);
    try {
      await apiClient.delete(`/admin/llm/providers/${del.id}`, { idempotent: true });
      setDel(null);
      setDelError(null);
      await Promise.all([qc.invalidateQueries({ queryKey: KEY.providers }), qc.invalidateQueries({ queryKey: KEY.routes })]);
    } catch (e) {
      setDelError(e instanceof ApiError ? e : new ApiError({ status: 0, code: "NETWORK" }));
    } finally {
      setDeleting(false);
    }
  }

  return (
    <div className={s.rows}>
      {(providers.length === 0 || envFallback.active) && providers.length === 0 && (
        <InlineNotice
          tone="info"
          action={canEdit && form?.mode !== "add" ? <Button size="sm" onClick={() => setForm({ mode: "add" })}>Thêm nhà cung cấp</Button> : undefined}
        >
          Đang dùng cấu hình mặc định của máy chủ. Thêm nhà cung cấp để thay đổi.
        </InlineNotice>
      )}
      <ul className={s.list}>
        {providers.map((p) => {
          const st = providerStatus(p);
          const t = tests[p.id];
          return (
            <li key={p.id} className={s.item} data-part="provider-row">
              <div className={s.providerGrid}>
                <div className={s.info}>
                  <p className="ep-item-title">{p.name}</p>
                  <p className={s.sub}>{TYPE_LABEL[p.type] ?? p.type}</p>
                  {p.type === "openai_compatible" && p.base_url && <p className={s.mono}>{p.base_url}</p>}
                </div>
                <div className={s.status} data-part="provider-status">
                  <StatusText tone={st.tone === "neutral" ? "neutral" : st.tone}>{st.text}</StatusText>
                  {p.has_key && <p className={s.sub}>Khoá API •••••••• · đã kết nối</p>}
                </div>
                {canEdit && (
                  <div className={s.action} data-part="provider-action">
                    <Switch label={p.enabled ? "Đang bật" : "Đang tắt"} checked={p.enabled} onChange={(v) => toggle(p, v)} />
                    <Button size="sm" variant="secondary" loading={t?.pending} onClick={() => void runTest(p)}>
                      {t?.pending ? "Đang kiểm tra…" : "Test kết nối"}
                    </Button>
                    <OverflowMenu
                      label={`Thêm hành động cho ${p.name}`}
                      items={[
                        { label: "Sửa", onSelect: () => setForm({ mode: "edit", id: p.id }) },
                        { label: "Đổi khoá", onSelect: () => setForm({ mode: "key", id: p.id }) },
                        { label: "Xoá", onSelect: () => { setDelError(null); setDel(p); } },
                      ]}
                    />
                  </div>
                )}
              </div>
              {t?.result && (
                <InlineNotice tone={t.result.ok ? "success" : "danger"} compact>
                  {t.result.ok ? `Kết nối tốt · ${t.result.latency_ms} ms` : testMessage(t.result.error_kind, t.result.message)}
                </InlineNotice>
              )}
              {t?.error && <ApiErrorNotice error={t.error} />}
              {saveFor === p.id && saver.error?.code === "VERSION_CONFLICT" && <ConflictNotice pending={saver.pending} onKeepMine={() => void saver.keepMine()} onUseTheirs={() => void saver.useTheirs()} />}
              {saveFor === p.id && saver.error && saver.error.code !== "VERSION_CONFLICT" && <ApiErrorNotice error={saver.error} onRetry={() => void saver.retry()} />}
              {form?.id === p.id && <ProviderForm key={form.mode} mode={form.mode} provider={p} onClose={() => setForm(null)} />}
            </li>
          );
        })}
      </ul>
      {canEdit && form?.mode === "add" && <ProviderForm mode="add" onClose={() => setForm(null)} />}
      {canEdit && form?.mode !== "add" && providers.length > 0 && (
        <div>
          <Button variant="text" onClick={() => setForm({ mode: "add" })}>
            Thêm nhà cung cấp
          </Button>
        </div>
      )}
      <ConfirmIrreversible
        open={Boolean(del)}
        onClose={() => { setDel(null); setDelError(null); }}
        onConfirm={() => void remove()}
        title={`Xoá ${del?.name ?? "nhà cung cấp"}?`}
        consequence="Nhà cung cấp và các mô hình của nó sẽ bị xoá. Không hoàn tác được."
        confirmLabel="Xoá nhà cung cấp"
        cancelLabel="Để sau"
        loading={deleting}
        error={delError ? delError.userMessage : undefined}
      />
    </div>
  );
}
