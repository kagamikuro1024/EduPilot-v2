"use client";

import { useQueryClient } from "@tanstack/react-query";
import dynamic from "next/dynamic";
import { useState } from "react";
import { ApiError, ApiErrorNotice, apiClient, useUndoableAction } from "@/shared/data";
import { Button, InlineNotice, SegmentedControl, StatusText, Switch } from "@/shared/ui";
import { DECISION_KEY, GATES, MODELS, PROVIDER_STATUS, useDecision, type DecisionGate, type DecisionOverview, type DecisionProvider } from "./decisionApi";
import { KEY, PROBE_TIMEOUT_MS, type Provider, type TestResult } from "./api";
import { testMessage } from "./labels";
import s from "./llm.module.css";

// «Cài đặt nâng cao» và «Theo lớp» chỉ mở sau khi bấm ⇒ phân mảnh riêng, không nằm trong JS lượt tải đầu (ngân sách script /settings/llm).
const Advanced = dynamic(() => import("./DecisionExtras").then((m) => m.Advanced), { loading: () => null });
const CourseTable = dynamic(() => import("./DecisionExtras").then((m) => m.CourseTable), { loading: () => null });
const ProviderForm = dynamic(() => import("./ProviderForm").then((m) => m.ProviderForm), { loading: () => null });


/** Khối «Quyết định nhanh» (US-DM-06): hàng provider rồi ba hàng cổng; Hoàn tác tại chỗ, không hộp xác nhận, không toast. */
export function DecisionSection({ providers, canEdit }: { providers: Provider[]; canEdit: boolean }) {
  const q = useDecision();
  if (q.isPending) return <p className={s.sub}>Đang tải…</p>;
  if (q.isError) return <ApiErrorNotice error={q.error} title="Chưa tải được phần này." onRetry={() => void q.refetch()} />;
  const rec = providers.find((p) => p.type === "decision" && p.enabled) ?? providers.find((p) => p.type === "decision");
  return (
    <div className={s.rows} data-part="decision-section">
      <ProviderRow data={q.data} rec={rec} canEdit={canEdit} />
      <ul className={s.list}>
        {[...q.data.gates].sort((a, b) => GATES.findIndex((x) => x.gate === a.gate) - GATES.findIndex((x) => x.gate === b.gate)).map((g) => (
          <GateRow key={g.gate} gate={g} provider={q.data.provider} canEdit={canEdit} />
        ))}
      </ul>
    </div>
  );
}

function ProviderRow({ data, rec, canEdit }: { data: DecisionOverview; rec: Provider | undefined; canEdit: boolean }) {
  const qc = useQueryClient();
  const p = data.provider;
  const st = PROVIDER_STATUS[p?.status ?? "none"];
  const [err, setErr] = useState<ApiError | null>(null);
  const [test, setTest] = useState<{ pending: boolean; result?: TestResult; error?: ApiError }>({ pending: false });
  const [keyForm, setKeyForm] = useState(false);
  const refresh = () => Promise.all([qc.invalidateQueries({ queryKey: DECISION_KEY }), qc.invalidateQueries({ queryKey: KEY.providers })]);
  const models = (sel: string) => MODELS.map((m) => ({ model: m, kind: "decision", price_in: rec?.models.find((x) => x.model === m)?.price_in ?? "0", price_out: "0", enabled: m === sel }));

  async function choose(model: string) {
    if (!rec || model === p?.model) return;
    setErr(null);
    try {
      await apiClient.put(`/admin/llm/providers/${rec.id}`, { type: "decision", name: rec.name, base_url: rec.base_url, enabled: true, rpm_limit: rec.rpm_limit, tpm_limit: rec.tpm_limit, models: models(model), version: rec.version });
      await refresh();
    } catch (e) {
      setErr(e instanceof ApiError ? e : new ApiError({ status: 0, code: "NETWORK" }));
      if (e instanceof ApiError && e.code === "VERSION_CONFLICT") await refresh();
    }
  }

  async function add() {
    setErr(null);
    try {
      await apiClient.post("/admin/llm/providers", { type: "decision", name: "Quyết định", enabled: true, models: models("decision-fake") }, { idempotent: true });
      await refresh();
    } catch (e) {
      setErr(e instanceof ApiError ? e : new ApiError({ status: 0, code: "NETWORK" }));
    }
  }

  async function runTest() {
    if (!p) return;
    setTest({ pending: true });
    try {
      const r = await apiClient.post<TestResult>(`/admin/llm/providers/${p.id}/test`, {}, { timeoutMs: PROBE_TIMEOUT_MS });
      setTest({ pending: false, result: r.data });
      void refresh();
    } catch (e) {
      setTest({ pending: false, error: e instanceof ApiError ? e : new ApiError({ status: 0, code: "NETWORK" }) });
    }
  }

  return (
    <div className={s.item} data-part="decision-provider">
      <div className={s.providerGrid}>
        <div className={s.info}>
          <p className="ep-item-title">Quyết định</p>
          {p && canEdit && rec ? (
            <SegmentedControl label="Mô hình quyết định" value={p.model as (typeof MODELS)[number]} onChange={(m) => void choose(m)} options={MODELS.map((m) => ({ value: m, label: m }))} />
          ) : (
            p && <p className={s.mono}>{p.model}</p>
          )}
        </div>
        <div className={s.status} data-part="decision-provider-status">
          <StatusText tone={st.tone}>{st.text}</StatusText>
        </div>
        {canEdit && (
          <div className={s.action}>
            {p ? (
              <>
                <Button size="sm" variant="secondary" loading={test.pending} onClick={() => void runTest()}>
                  {test.pending ? "Đang kiểm tra…" : "Test kết nối"}
                </Button>
                {!p.fake && rec && (
                  <Button size="sm" variant="ghost" onClick={() => setKeyForm(true)}>
                    Nhập khoá
                  </Button>
                )}
              </>
            ) : (
              <Button size="sm" onClick={() => void add()}>
                Thêm
              </Button>
            )}
          </div>
        )}
      </div>
      {test.result && (
        <InlineNotice tone={test.result.ok ? "success" : "danger"} compact>
          {test.result.ok ? `Kết nối tốt · ${test.result.latency_ms} ms` : testMessage(test.result.error_kind, test.result.message)}
        </InlineNotice>
      )}
      {test.error && <ApiErrorNotice error={test.error} />}
      {err && (err.code === "DECISION_REAL_DISABLED" ? <InlineNotice tone="warning" compact>{err.userMessage}</InlineNotice> : <ApiErrorNotice error={err} />)}
      {keyForm && rec && <ProviderForm key="key" mode="key" provider={rec} onClose={() => setKeyForm(false)} />}
    </div>
  );
}

function GateRow({ gate, provider, canEdit }: { gate: DecisionGate; provider: DecisionProvider | null; canEdit: boolean }) {
  const qc = useQueryClient();
  const meta = GATES.find((x) => x.gate === gate.gate)!;
  const [note, setNote] = useState<string | null>(null);
  const [adv, setAdv] = useState(false);
  const [courses, setCourses] = useState(false);

  const patch = (fn: (g: DecisionGate) => DecisionGate) =>
    qc.setQueryData<DecisionOverview>(DECISION_KEY, (o) => (o ? { ...o, gates: o.gates.map((g) => (g.gate === gate.gate ? fn(g) : g)) } : o));
  const latest = () => qc.getQueryData<DecisionOverview>(DECISION_KEY)?.gates.find((g) => g.gate === gate.gate) ?? gate;
  const put = async (body: Record<string, unknown>) => {
    const r = await apiClient.put<DecisionGate>(`/admin/llm/decision/gates/${gate.gate}`, { ...body, version: latest().version });
    patch((g) => ({ ...g, ...r.data }));
    return r.data;
  };
  const conflict = (e: unknown) => {
    if (e instanceof ApiError && e.code === "VERSION_CONFLICT") {
      setNote("Cài đặt vừa được người khác đổi. Đã tải lại.");
      void qc.invalidateQueries({ queryKey: DECISION_KEY });
      return true;
    }
    return false;
  };

  const toggle = useUndoableAction<boolean>({
    apply: (next) => patch((g) => ({ ...g, enabled: next })),
    rollback: (next) => patch((g) => ({ ...g, enabled: !next })),
    commit: (next) => put({ enabled: next }),
    compensate: (next) => put({ enabled: !next }),
    label: (next) => `Đã ${next ? "bật" : "tắt"} ${meta.label}`,
    handled: conflict,
  });

  const lockedTip = provider && !provider.fake ? "Gửi nội dung đã che tới nhà cung cấp" : undefined;
  const state = gate.enabled ? `Bật ở ${gate.courses_enabled} lớp` : "Tắt";

  return (
    <li className={s.item} data-part="decision-gate" data-gate={gate.gate}>
      <div className={s.providerGrid}>
        <div className={s.info}>
          <Switch label={meta.label} checked={gate.enabled} disabled={!canEdit || toggle.pending} title={gate.enabled ? undefined : lockedTip} onChange={(v) => { setNote(null); void toggle.run(v); }} />
        </div>
        <div className={s.status} data-part="decision-gate-state">
          <StatusText tone={gate.enabled ? (gate.courses_enabled > 0 ? "green" : "amber") : "neutral"}>{state}</StatusText>
        </div>
        <div className={s.action}>
          <Button size="sm" variant="ghost" aria-expanded={adv} onClick={() => setAdv((v) => !v)}>
            Cài đặt nâng cao
          </Button>
          <Button size="sm" variant="ghost" aria-expanded={courses} onClick={() => setCourses((v) => !v)}>
            Theo lớp
          </Button>
        </div>
      </div>
      {toggle.node}
      {note && <InlineNotice tone="warning" compact>{note}</InlineNotice>}
      {adv && <Advanced gate={gate} meta={meta} canEdit={canEdit} put={put} conflict={conflict} />}
      {courses && <CourseTable gate={gate} canEdit={canEdit} />}
    </li>
  );
}
