"use client";

import { useQueryClient } from "@tanstack/react-query";
import { useState } from "react";
import { ApiError, apiClient } from "@/shared/data";
import { ConfirmIrreversible, Field, InlineNotice, Select } from "@/shared/ui";
import { KEY, type Provider, type RoutesRes } from "./api";
import s from "./llm.module.css";

/** Phần 4 — Mô hình tìm kiếm tài liệu: MỘT mô hình, 1536 chiều cố định; đổi cần xác nhận vì phải lập chỉ mục lại (US-P1-05 AC6). */
export function EmbeddingSection({ routes, providers, canEdit }: { routes: RoutesRes; providers: Provider[]; canEdit: boolean }) {
  const qc = useQueryClient();
  const current = routes.embedding;
  const route = routes.items.find((r) => r.task === "EMBEDDING");
  const [next, setNext] = useState<string | null>(null);
  const [error, setError] = useState<ApiError | null>(null);
  const [busy, setBusy] = useState(false);
  const [reindex, setReindex] = useState(false);
  const options = providers.filter((p) => p.enabled).map((p) => ({ provider: p.name, models: p.models.filter((m) => m.kind === "embedding" && m.dims === 1536 && m.enabled) })).filter((g) => g.models.length > 0);
  const target = options.flatMap((g) => g.models).find((m) => m.id === next);

  async function applyChange() {
    if (!next) return;
    setBusy(true);
    try {
      const r = await apiClient.put<{ reindex_required?: boolean }>("/admin/llm/routes", { task: "EMBEDDING", chain: [next], params: {}, version: route?.version ?? 0 }, { idempotent: true });
      setReindex(Boolean(r.data.reindex_required));
      setNext(null);
      setError(null);
      await qc.invalidateQueries({ queryKey: KEY.routes });
    } catch (e) {
      setError(e instanceof ApiError ? e : new ApiError({ status: 0, code: "NETWORK" }));
    } finally {
      setBusy(false);
    }
  }

  return (
    <div className={s.rows}>
      <p className={s.sub}>Đổi mục này cần lập chỉ mục lại tài liệu.</p>
      <div className={s.embedRow} data-part="embedding-current">
        <div>
          <p className="ep-item-title">{current ? current.model : "Chưa chọn — đang dùng cấu hình mặc định của máy chủ"}</p>
          <p className={s.sub}>
            {current ? `${current.provider_name} · ` : ""}
            1536 chiều
          </p>
        </div>
        {canEdit && options.length > 0 && (
          <Field label="Đổi mô hình tìm kiếm tài liệu">
            {(id) => (
              <Select id={id} value={current?.model_id ?? ""} onChange={(e) => e.target.value && e.target.value !== current?.model_id && (setError(null), setNext(e.target.value))}>
                {!current && <option value="">Chọn mô hình</option>}
                {options.flatMap((g) =>
                  g.models.map((m) => (
                    <option key={m.id} value={m.id} data-dims={m.dims ?? ""}>
                      {g.provider} · {m.model}
                    </option>
                  )),
                )}
              </Select>
            )}
          </Field>
        )}
      </div>
      {(reindex || current?.reindex_required) && (
        <InlineNotice tone="warning" title="Cần lập chỉ mục lại">
          Tài liệu đã nạp cần được lập chỉ mục lại với mô hình mới. Việc này chưa tự chạy.
        </InlineNotice>
      )}
      <ConfirmIrreversible
        open={Boolean(next)}
        onClose={() => { setNext(null); setError(null); }}
        onConfirm={() => void applyChange()}
        title={`Đổi sang ${target?.model ?? "mô hình mới"}?`}
        consequence="Đổi mô hình tìm kiếm tài liệu. Mọi tài liệu đã nạp sẽ phải lập chỉ mục lại; trong lúc đó tìm kiếm có thể kém chính xác."
        confirmLabel="Đổi mô hình"
        cancelLabel="Để sau"
        loading={busy}
        error={error ? error.userMessage : undefined}
      />
    </div>
  );
}
