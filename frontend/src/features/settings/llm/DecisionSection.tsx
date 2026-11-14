"use client";

import { useQueryClient } from "@tanstack/react-query";
import dynamic from "next/dynamic";
import { useEffect, useState } from "react";
import { ApiError, ApiErrorNotice, apiClient, fieldErrors, useCursorList, useUndoableAction } from "@/shared/data";
import { Button, DataTable, EmptyState, Field, Input, InlineNotice, SegmentedControl, StatusText, Switch } from "@/shared/ui";
import { DECISION_KEY, GATES, MODELS, PROVIDER_STATUS, useDecision, type DecisionCourse, type DecisionGate, type DecisionOverview, type DecisionProvider } from "./decisionApi";
import { KEY, type Provider, type TestResult } from "./api";
import { testMessage } from "./labels";
import s from "./llm.module.css";

const ProviderForm = dynamic(() => import("./ProviderForm").then((m) => m.ProviderForm), { loading: () => null });

const asDec = (n: string) => String(Number(n)).replace(".", ","); // "0.050" → "0,05"
const toWire = (v: string) => v.trim().replace(",", ".");

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
      const r = await apiClient.post<TestResult>(`/admin/llm/providers/${p.id}/test`, {});
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

function Advanced({ gate, meta, canEdit, put, conflict }: { gate: DecisionGate; meta: (typeof GATES)[number]; canEdit: boolean; put: (b: Record<string, unknown>) => Promise<DecisionGate>; conflict: (e: unknown) => boolean }) {
  const [th, setTh] = useState(asDec(gate.threshold));
  const [to, setTo] = useState(String(gate.timeout_ms));
  const [errs, setErrs] = useState<Record<string, string>>({});
  const save = useUndoableAction<{ th: string; to: number; prevTh: string; prevTo: number }>({
    apply: () => undefined,
    rollback: (a) => { setTh(asDec(a.prevTh)); setTo(String(a.prevTo)); },
    commit: async (a) => {
      setErrs({});
      try {
        await put({ threshold: a.th, timeout_ms: a.to });
      } catch (e) {
        setErrs(fieldErrors(e));
        throw e;
      }
    },
    compensate: (a) => put({ threshold: a.prevTh, timeout_ms: a.prevTo }),
    label: () => "Đã lưu cài đặt nâng cao",
    handled: (e) => conflict(e) || e instanceof ApiError && e.code === "VALIDATION_FAILED",
  });
  return (
    <div className={s.advanced} data-part="decision-advanced">
      <div className={s.pair}>
        <Field label={`Ngưỡng ${meta.unit}`} error={errs.threshold}>
          {(id, d) => <Input id={id} aria-describedby={d} invalid={Boolean(errs.threshold)} inputMode="decimal" placeholder={meta.range} readOnly={!canEdit} title={`Từ ${meta.range}`} value={th} onChange={(e) => setTh(e.target.value)} />}
        </Field>
        <Field label="Thời hạn (ms)" error={errs.timeout_ms}>
          {(id, d) => <Input id={id} aria-describedby={d} invalid={Boolean(errs.timeout_ms)} inputMode="numeric" placeholder="200–5000" readOnly={!canEdit} value={to} onChange={(e) => setTo(e.target.value)} />}
        </Field>
      </div>
      {canEdit && (
        <div>
          <Button size="sm" variant="primary" loading={save.pending} onClick={() => void save.run({ th: toWire(th), to: Number(to), prevTh: gate.threshold, prevTo: gate.timeout_ms })}>
            Lưu
          </Button>
        </div>
      )}
      {save.node}
    </div>
  );
}

function CourseTable({ gate, canEdit }: { gate: DecisionGate; canEdit: boolean }) {
  const qc = useQueryClient();
  const [text, setText] = useState("");
  const [q, setQ] = useState("");
  useEffect(() => {
    const t = window.setTimeout(() => setQ(text.trim()), 300);
    return () => window.clearTimeout(t);
  }, [text]);
  const list = useCursorList<DecisionCourse>(["llm", "decision", "courses", gate.gate, q], "/admin/llm/decision/courses", { limit: 30, query: { gate: gate.gate, q }, idOf: (c) => c.course_id });
  const [over, setOver] = useState<Record<string, { enabled: boolean; version: number }>>({});
  const cur = (c: DecisionCourse) => over[c.course_id] ?? { enabled: c.enabled, version: c.version };
  const put = async (c: DecisionCourse, enabled: boolean) => {
    const r = await apiClient.put<{ enabled: boolean; version: number }>(`/courses/${c.course_id}/decision-gates/${gate.gate}`, { enabled, version: cur(c).version });
    setOver((o) => ({ ...o, [c.course_id]: { enabled: r.data.enabled, version: r.data.version } }));
    void qc.invalidateQueries({ queryKey: DECISION_KEY });
  };
  const toggle = useUndoableAction<{ c: DecisionCourse; next: boolean }>({
    apply: ({ c, next }) => setOver((o) => ({ ...o, [c.course_id]: { enabled: next, version: cur(c).version } })),
    rollback: ({ c, next }) => setOver((o) => ({ ...o, [c.course_id]: { enabled: !next, version: cur(c).version } })),
    commit: ({ c, next }) => put(c, next),
    compensate: ({ c, next }) => put(c, !next),
    label: ({ c, next }) => `Đã ${next ? "bật" : "tắt"} cho ${c.name}`,
    handled: (e) => {
      if (e instanceof ApiError && e.code === "VERSION_CONFLICT") {
        setOver({});
        void list.refetch();
        return true;
      }
      return false;
    },
    itemKey: ({ c }) => c.course_id,
  });
  const off = !gate.enabled;
  return (
    <div className={s.advanced} data-part="decision-courses">
      <Field label="Tìm lớp">{(id) => <Input id={id} type="search" placeholder="Tên hoặc mã lớp" value={text} onChange={(e) => setText(e.target.value)} />}</Field>
      {list.isError ? (
        <ApiErrorNotice error={list.error} onRetry={() => void list.refetch()} />
      ) : (
        <DataTable<DecisionCourse>
          caption={`Lớp và công tắc của ${GATES.find((g) => g.gate === gate.gate)?.label}`}
          rows={list.items}
          rowKey={(c) => c.course_id}
          loading={list.isPending ? 4 : undefined}
          mobile="scroll"
          empty={<EmptyState title="Không có lớp nào" />}
          columns={[
            { key: "name", header: "Lớp", primary: true, render: (c) => <><span className="ep-item-title">{c.name}</span> <span className={s.sub}>{c.class_code}</span></> },
            { key: "on", header: "Bật", width: "120px", align: "end", render: (c) => <Switch label={<span className="ep-sr-only">{`Bật cho ${c.name}`}</span>} checked={cur(c).enabled} disabled={!canEdit || off} title={off ? "Đang tắt toàn hệ thống" : undefined} onChange={(v) => void toggle.run({ c, next: v })} /> },
          ]}
        />
      )}
      {list.hasNextPage && (
        <div>
          <Button size="sm" variant="ghost" loading={list.isFetchingNextPage} onClick={() => void list.fetchNextPage()}>
            Xem thêm
          </Button>
        </div>
      )}
      {toggle.node}
    </div>
  );
}
