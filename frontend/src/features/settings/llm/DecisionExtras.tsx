"use client";

import { useQueryClient } from "@tanstack/react-query";
import { useEffect, useState } from "react";
import { ApiError, ApiErrorNotice, apiClient, fieldErrors, useCursorList, useUndoableAction } from "@/shared/data";
import { Button, DataTable, EmptyState, Field, Input, Switch } from "@/shared/ui";
import { DECISION_KEY, GATES, type DecisionCourse, type DecisionGate } from "./decisionApi";
import s from "./llm.module.css";

const asDec = (n: string) => String(Number(n)).replace(".", ","); // "0.050" → "0,05"
const toWire = (v: string) => v.trim().replace(",", ".");

export function Advanced({ gate, meta, canEdit, put, conflict }: { gate: DecisionGate; meta: (typeof GATES)[number]; canEdit: boolean; put: (b: Record<string, unknown>) => Promise<DecisionGate>; conflict: (e: unknown) => boolean }) {
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

export function CourseTable({ gate, canEdit }: { gate: DecisionGate; canEdit: boolean }) {
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
