"use client";

import { useState } from "react";
import { useQueryClient } from "@tanstack/react-query";
import { ApiError, ApiErrorNotice, apiClient, fieldErrors, useIdempotentMutation } from "@/shared/data";
import { Button, Field, Input, Select } from "@/shared/ui";
import { ConflictNotice } from "./ConflictNotice";
import { DEFAULT_MODELS, TYPES, TYPE_LABEL, testMessage } from "./labels";
import { KEY, useSaver, type Provider } from "./api";
import s from "./llm.module.css";

type ModelDraft = { key: number; model: string; kind: "chat" | "embedding"; priceIn: string; priceOut: string };
export type FormMode = "add" | "edit" | "key";

let seq = 0; // khoá React cho dòng mô hình (không gửi lên máy chủ)
const newModel = (model = "", kind: "chat" | "embedding" = "chat", priceIn = "0", priceOut = "0"): ModelDraft => ({ key: ++seq, model, kind, priceIn, priceOut });

/**
 * Form xổ tại chỗ để thêm / sửa nhà cung cấp hoặc chỉ đổi khoá (US-P1-05 AC2–AC4, AC7).
 * Khoá API là ô GHI-CHỈ: type=password, autocomplete=new-password, trống khi mở; sau khi lưu ô tự trống và không lưu nháp.
 * Khoá sai (422 PROVIDER_AUTH_FAILED) ⇒ lỗi dưới ô khoá, không lưu gì, các ô khác giữ chữ.
 */
export function ProviderForm({ mode, provider, onClose }: { mode: FormMode; provider?: Provider; onClose: () => void }) {
  const qc = useQueryClient();
  const [type, setType] = useState(provider?.type ?? "openai");
  const [name, setName] = useState(provider?.name ?? "");
  const [baseUrl, setBaseUrl] = useState(provider?.base_url ?? "");
  const [apiKey, setApiKey] = useState("");
  const [models, setModels] = useState<ModelDraft[]>(() => (provider ? provider.models.map((m) => newModel(m.model, m.kind, m.price_in, m.price_out)) : DEFAULT_MODELS.openai.map((m) => newModel(m.model, m.kind))));
  const [rpm, setRpm] = useState(provider?.rpm_limit?.toString() ?? "");
  const [tpm, setTpm] = useState(provider?.tpm_limit?.toString() ?? "");
  const [advanced, setAdvanced] = useState(false);
  const [local, setLocal] = useState<Record<string, string>>({});
  const edit = useSaver(qc, [KEY.providers, KEY.routes]);
  const create = useIdempotentMutation((body: object, idemKey) => apiClient.post<Provider>("/admin/llm/providers", body, { idempotencyKey: idemKey }));

  const needsUrl = type === "openai_compatible";
  const err = (mode === "add" ? create.error : edit.error) ?? null;
  const server = err ? fieldErrors(err) : {};
  const pending = mode === "add" ? create.pending : edit.pending;
  const offline = err?.code === "NETWORK" || JSON.stringify(err?.details ?? "").includes("PROVIDER_UNREACHABLE");
  const authMsg = server.api_key ?? undefined;

  function changeType(next: string) {
    setType(next);
    if (mode === "add") setModels(DEFAULT_MODELS[next].map((m) => newModel(m.model, m.kind)));
  }

  function validate(): boolean {
    const e: Record<string, string> = {};
    if (mode !== "key") {
      if (!name.trim()) e.name = "Nhập tên để nhận ra nhà cung cấp này.";
      if (needsUrl && !baseUrl.trim()) e.base_url = "Máy chủ riêng cần địa chỉ, ví dụ https://llm.truong.edu.vn/v1.";
      if (rpm && !/^[1-9]\d*$/.test(rpm)) e.rpm_limit = "Nhập số nguyên dương.";
      if (tpm && !/^[1-9]\d*$/.test(tpm)) e.tpm_limit = "Nhập số nguyên dương.";
      models.forEach((m, i) => {
        if (!m.model.trim()) e[`models[${i}].model`] = "Nhập tên mô hình.";
        if (!/^\d+(\.\d+)?$/.test(m.priceIn) || !/^\d+(\.\d+)?$/.test(m.priceOut)) e[`models[${i}].price`] = "Giá là số không âm (đ / 1 triệu token).";
      });
    }
    if (mode === "add" && !apiKey && type !== "fake") e.api_key = "Nhập khoá API của nhà cung cấp.";
    if (mode === "key" && !apiKey) e.api_key = "Nhập khoá mới. Muốn giữ khoá cũ thì bấm Hủy.";
    setLocal(e);
    if (Object.keys(e).length > 0) setAdvanced((a) => a || Object.keys(e).some((k) => k.startsWith("models[") || k === "rpm_limit" || k === "tpm_limit" || (k === "base_url" && !needsUrl)));
    return Object.keys(e).length === 0;
  }

  function body(skipVerify: boolean) {
    const payload: Record<string, unknown> = { type: provider?.type ?? type, name: (mode === "key" ? provider?.name : name.trim()) ?? name, skip_verify: skipVerify };
    if (mode === "key") {
      payload.base_url = provider?.base_url ?? null;
      payload.enabled = provider?.enabled;
    } else {
      payload.base_url = baseUrl.trim() || null;
      payload.rpm_limit = rpm ? Number(rpm) : null;
      payload.tpm_limit = tpm ? Number(tpm) : null;
      payload.models = models.map((m) => ({ model: m.model.trim(), kind: m.kind, dims: m.kind === "embedding" ? 1536 : null, price_in: m.priceIn, price_out: m.priceOut, enabled: true }));
      if (provider) payload.enabled = provider.enabled;
    }
    if (apiKey) payload.api_key = apiKey; // để trống = giữ khoá cũ (không gửi trường)
    return payload;
  }

  async function submit(skipVerify = false) {
    if (!validate()) return;
    if (mode === "add") {
      const ok = await create.mutate(body(skipVerify)).then(() => true, () => false);
      if (ok) {
        setApiKey("");
        await qc.invalidateQueries({ queryKey: KEY.providers });
        onClose();
      }
      return;
    }
    const ok = await edit.run((v) => apiClient.put(`/admin/llm/providers/${provider!.id}`, { ...body(skipVerify), version: v ?? provider!.version }));
    setApiKey("");
    if (ok) onClose();
  }

  const fe = (k: string) => local[k] ?? server[k];
  const title = mode === "add" ? "Thêm nhà cung cấp" : mode === "key" ? `Đổi khoá của ${provider?.name}` : `Sửa ${provider?.name}`;

  return (
    <form
      className={s.form}
      data-part="provider-form"
      aria-label={title}
      onSubmit={(e) => {
        e.preventDefault();
        if (!pending) void submit();
      }}
    >
      <h3 className="ep-item-title">{title}</h3>
      {mode !== "key" && (
        <>
          {mode === "add" && (
            <Field label="Loại nhà cung cấp">
              {(id) => (
                <Select id={id} value={type} onChange={(e) => changeType(e.target.value)}>
                  {TYPES.map((t) => (
                    <option key={t} value={t}>
                      {TYPE_LABEL[t]}
                    </option>
                  ))}
                </Select>
              )}
            </Field>
          )}
          <Field label="Tên hiển thị" required error={fe("name")}>
            {(id, d) => <Input id={id} aria-describedby={d} invalid={Boolean(fe("name"))} value={name} onChange={(e) => setName(e.target.value)} maxLength={80} />}
          </Field>
          {needsUrl && (
            <Field label="Địa chỉ máy chủ" required helper="Địa chỉ gốc của máy chủ tương thích OpenAI." error={fe("base_url")}>
              {(id, d) => <Input id={id} aria-describedby={d} invalid={Boolean(fe("base_url"))} value={baseUrl} onChange={(e) => setBaseUrl(e.target.value)} placeholder="https://llm.truong.edu.vn/v1" spellCheck={false} />}
            </Field>
          )}
        </>
      )}
      <Field
        label="Khoá API"
        required={mode !== "edit" && type !== "fake"}
        helper={mode === "add" ? "Khoá chỉ ghi được: lưu xong sẽ không đọc lại được." : "Để trống để giữ khoá hiện tại"}
        error={fe("api_key") ?? authMsg}
      >
        {(id, d) => (
          <Input id={id} aria-describedby={d} invalid={Boolean(fe("api_key"))} type="password" autoComplete="new-password" spellCheck={false} value={apiKey} onChange={(e) => setApiKey(e.target.value)} />
        )}
      </Field>

      {mode !== "key" && (
        <div>
          <Button variant="text" size="sm" aria-expanded={advanced} aria-controls="provider-advanced" onClick={() => setAdvanced((a) => !a)}>
            Cài đặt nâng cao
          </Button>
          <div id="provider-advanced" hidden={!advanced} className={s.advanced}>
            {!needsUrl && (
              <Field label="Địa chỉ máy chủ (tuỳ chọn)" helper="Chỉ điền khi đi qua một cổng trung gian." error={fe("base_url")}>
                {(id, d) => <Input id={id} aria-describedby={d} invalid={Boolean(fe("base_url"))} value={baseUrl} onChange={(e) => setBaseUrl(e.target.value)} spellCheck={false} />}
              </Field>
            )}
            <div className={s.pair}>
              <Field label="Giới hạn lượt gọi mỗi phút" helper="Để trống = mặc định của máy chủ" error={fe("rpm_limit")}>
                {(id, d) => <Input id={id} aria-describedby={d} invalid={Boolean(fe("rpm_limit"))} inputMode="numeric" value={rpm} onChange={(e) => setRpm(e.target.value)} />}
              </Field>
              <Field label="Giới hạn token mỗi phút" error={fe("tpm_limit")}>
                {(id, d) => <Input id={id} aria-describedby={d} invalid={Boolean(fe("tpm_limit"))} inputMode="numeric" value={tpm} onChange={(e) => setTpm(e.target.value)} />}
              </Field>
            </div>
            <p className="ep-item-title">Mô hình</p>
            {models.map((m, i) => (
              <div key={m.key} className={s.modelRow} data-part="model-row">
                <Field label="Tên mô hình" error={fe(`models[${i}].model`)}>
                  {(id, d) => (
                    <Input id={id} aria-describedby={d} invalid={Boolean(fe(`models[${i}].model`))} value={m.model} spellCheck={false} onChange={(e) => setModels((ms) => ms.map((x) => (x.key === m.key ? { ...x, model: e.target.value } : x)))} />
                  )}
                </Field>
                <Field label="Dùng cho">
                  {(id) => (
                    <Select id={id} value={m.kind} onChange={(e) => setModels((ms) => ms.map((x) => (x.key === m.key ? { ...x, kind: e.target.value as ModelDraft["kind"] } : x)))}>
                      <option value="chat">Trả lời / chấm bài</option>
                      <option value="embedding">Tìm kiếm tài liệu (1536 chiều)</option>
                    </Select>
                  )}
                </Field>
                <Field label="Giá vào (đ / 1 triệu token)" error={fe(`models[${i}].price`)}>
                  {(id, d) => <Input id={id} aria-describedby={d} invalid={Boolean(fe(`models[${i}].price`))} inputMode="decimal" value={m.priceIn} onChange={(e) => setModels((ms) => ms.map((x) => (x.key === m.key ? { ...x, priceIn: e.target.value } : x)))} />}
                </Field>
                <Field label="Giá ra (đ / 1 triệu token)">
                  {(id) => <Input id={id} inputMode="decimal" value={m.priceOut} onChange={(e) => setModels((ms) => ms.map((x) => (x.key === m.key ? { ...x, priceOut: e.target.value } : x)))} />}
                </Field>
                <Button size="sm" variant="ghost" aria-label={`Bỏ mô hình ${m.model || i + 1}`} onClick={() => setModels((ms) => ms.filter((x) => x.key !== m.key))}>
                  Bỏ
                </Button>
              </div>
            ))}
            <Button size="sm" onClick={() => setModels((ms) => [...ms, newModel()])}>
              Thêm mô hình
            </Button>
          </div>
        </div>
      )}

      {err && err.code === "VERSION_CONFLICT" && mode !== "add" && <ConflictNotice pending={edit.pending} onKeepMine={() => void edit.keepMine().then((ok) => ok && onClose())} onUseTheirs={() => void edit.useTheirs().then(onClose)} />}
      {err && err.code !== "VERSION_CONFLICT" && !authMsg && Object.keys(server).length === 0 && <ApiErrorNotice error={err} />}
      {authMsg && <p className="ep-meta">Chưa lưu gì. Các ô khác vẫn giữ nguyên.</p>}
      {offline && (
        <p className="ep-meta">
          {testMessage("UNREACHABLE")} Máy chủ trong trường đang tắt?{" "}
          <Button variant="text" size="sm" onClick={() => void submit(true)}>
            Lưu mà không kiểm tra
          </Button>
        </p>
      )}
      {mode === "add" && create.error?.code === "NETWORK" && (
        <Button size="sm" onClick={() => void create.retry()?.then(() => qc.invalidateQueries({ queryKey: KEY.providers })).then(onClose, () => {})}>
          Gửi lại
        </Button>
      )}

      <div className={s.formActions}>
        <Button type="submit" variant="primary" loading={pending}>
          Lưu
        </Button>
        <Button variant="ghost" onClick={onClose}>
          Hủy
        </Button>
      </div>
    </form>
  );
}

export type { ApiError };
