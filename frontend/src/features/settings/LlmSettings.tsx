"use client";

import { useState } from "react";
import { KeyRound, PlugZap } from "lucide-react";
import { ADVANCED, BUDGET, EMBEDDING, FALLBACK_CHAIN, PROVIDERS, TASK_ROUTES } from "@/mock/system";
import { useUndoLine } from "@/shared/lib/useUndoLine";
import { useSession } from "@/shared/session/session";
import { useDemoSlice } from "@/shared/state/demo";
import { BarList, Button, ButtonLink, DefinitionList, EmptyState, Field, InlineNotice, Input, Page, PageHeader, PageState, Section, Select, Skeleton, StatusText } from "@/shared/ui";
import s from "./settings.module.css";

type TestResult = { ok: true; latencyMs: number } | { ok: false; problem: string; recovery: string };

export function LlmSettings() {
  const { role } = useSession();
  const canEdit = role === "admin";
  const [keys, setKeys] = useDemoSlice<Record<string, string>>("settings.keys", {});
  const [routes, setRoutes] = useDemoSlice<Record<string, string>>("settings.routes", {});
  const [embedding, setEmbedding] = useDemoSlice<string>("settings.embedding", EMBEDDING.model);
  const [tested, setTested] = useState<Record<string, TestResult>>({});
  const [editingKey, setEditingKey] = useState<string | null>(null);
  const [keyDraft, setKeyDraft] = useState("");
  const undo = useUndoLine();

  function saveKey(providerId: string) {
    const tail = keyDraft.trim().slice(-4) || "0000";
    setKeys((prev) => ({ ...prev, [providerId]: tail }));
    setEditingKey(null);
    setKeyDraft("");
    setTested((prev) => ({ ...prev, [providerId]: { ok: true, latencyMs: 356 } }));
  }

  function changeRoute(id: string, task: string, from: string, to: string) {
    setRoutes((prev) => ({ ...prev, [id]: to }));
    undo.push(`Đã đổi model của “${task}” sang ${to}`, () => setRoutes((prev) => ({ ...prev, [id]: from })));
  }

  function changeEmbedding(to: string) {
    const from = embedding;
    setEmbedding(to);
    undo.push(`Đã đổi mô hình embedding sang ${to}`, () => setEmbedding(from));
  }

  return (
    <Page>
      <PageHeader
        title="Cấu hình LLM"
        description="Tác vụ nào chạy bằng model nào, hỏng thì chuyển sang đâu, và tiêu bao nhiêu tiền."
        meta={
          <>
            <span>Áp dụng cho toàn hệ thống</span>
            <span>Cập nhật gần nhất 27/10, 23:10</span>
          </>
        }
      />

      <PageState
        loading={
          <div className={s.loading}>
            <Skeleton lines={4} />
            <Skeleton lines={5} />
          </div>
        }
        empty={
          <EmptyState title="Chưa cấu hình model nào" action={<ButtonLink href="/settings/llm">Xem cấu hình mặc định</ButtonLink>}>
            Mỗi tác vụ (trả lời sinh viên, chấm bài, embedding) cần một model chính và một model dự phòng. Chọn model cho tác vụ đầu tiên để AI bắt đầu trả lời.
          </EmptyState>
        }
        error={{
          problem: "Không đọc được cấu hình LLM.",
          recovery: "Hệ thống vẫn đang chạy bằng cấu hình đã lưu, sinh viên không bị ảnh hưởng. Thử lại sau ít phút trước khi đổi gì.",
        }}
      >
        {!canEdit && (
          <InlineNotice tone="info" compact>
            Chỉ quản trị viên hệ thống đổi được cấu hình này. Bạn xem để biết lớp đang chạy bằng model nào và còn bao nhiêu ngân sách.
          </InlineNotice>
        )}

        <Section part="settings-section" title="Kết nối nhà cung cấp" description="Khoá API chỉ ghi được: lưu xong không đọc lại được, chỉ còn 4 ký tự cuối.">
          <div className={s.configPanel}>
          <ul className={s.rows}>
            {PROVIDERS.map((p) => {
              const tail = keys[p.id] ?? p.keyTail;
              const failing = p.id === "gemini" && !keys[p.id];
              const result = tested[p.id] ?? (keys[p.id] ? { ok: true as const, latencyMs: 356 } : undefined);
              return (
                <li key={p.id} className={s.row}>
                  <div className={s.providerGrid}>
                    <div className={s.providerInfo}>
                      <p className="ep-item-title">{p.name}</p>
                      <p className={s.sub}>{p.kind}</p>
                      <p className={s.mono}>{p.endpoint}</p>
                    </div>
                    <div className={s.providerStatus} data-part="provider-status">
                      <StatusText tone={failing ? "amber" : "green"}>
                        ••••{tail} · {failing ? "khoá bị từ chối" : "đã kết nối"}
                      </StatusText>
                      <p className={s.sub}>Kiểm gần nhất: {p.lastCheck}</p>
                    </div>
                    <div className={s.providerAction} data-part="provider-action">
                      {canEdit && (
                        <>
                          <Button size="sm" icon={<PlugZap aria-hidden />} onClick={() => setTested((prev) => ({ ...prev, [p.id]: failing ? p.test : { ok: true, latencyMs: p.test.ok ? p.test.latencyMs : 356 } }))}>
                            Test kết nối
                          </Button>
                          <Button
                            size="sm"
                            variant="ghost"
                            icon={<KeyRound aria-hidden />}
                            onClick={() => {
                              setEditingKey(editingKey === p.id ? null : p.id);
                              setKeyDraft("");
                            }}
                          >
                            Đổi khoá
                          </Button>
                        </>
                      )}
                    </div>
                  </div>

                  {canEdit && editingKey === p.id && (
                    <div className={s.inlineForm}>
                      <Field label="Khoá API mới" helper="Dán khoá vào đây. Lưu xong màn này chỉ còn hiện 4 ký tự cuối.">
                        {(id, describedBy) => (
                          <Input id={id} aria-describedby={describedBy} type="password" value={keyDraft} onChange={(e) => setKeyDraft(e.target.value)} placeholder="sk-…" autoComplete="off" />
                        )}
                      </Field>
                      <div className={s.inlineFormActions}>
                        <Button variant="primary" size="sm" disabled={keyDraft.trim().length < 8} onClick={() => saveKey(p.id)}>
                          Lưu khoá
                        </Button>
                        <Button variant="ghost" size="sm" onClick={() => setEditingKey(null)}>
                          Huỷ
                        </Button>
                      </div>
                    </div>
                  )}

                  {result && result.ok && <StatusText tone="green">Kết nối được · {result.latencyMs} ms</StatusText>}
                  {result && !result.ok && (
                    <InlineNotice tone="danger" title={result.problem}>
                      {result.recovery}
                    </InlineNotice>
                  )}
                </li>
              );
            })}
          </ul>
          </div>
        </Section>

        <Section part="settings-section" title="Tác vụ nào dùng model nào" description="Đổi ở đây có hiệu lực ngay cho yêu cầu tiếp theo; yêu cầu đang chạy vẫn dùng model cũ.">
          <div className={s.configPanel}>
          <ul className={s.rows}>
            {TASK_ROUTES.map((r) => {
              const model = routes[r.id] ?? r.model;
              return (
                <li key={r.id} className={s.taskRow}>
                  <div>
                    <p className="ep-item-title">{r.task}</p>
                    <p className={s.sub}>{r.note}</p>
                  </div>
                  {canEdit ? (
                    <Select aria-label={`Model cho ${r.task}`} value={model} onChange={(e) => changeRoute(r.id, r.task, model, e.target.value)} className={s.modelSelect}>
                      {r.options.map((o) => (
                        <option key={o} value={o}>
                          {o}
                        </option>
                      ))}
                    </Select>
                  ) : (
                    <span className={s.mono}>{model}</span>
                  )}
                </li>
              );
            })}
          </ul>
          </div>
          {undo.node}
        </Section>

        <Section part="settings-section" title="Chuỗi dự phòng" description="Khi nhà cung cấp đầu tiên hỏng, yêu cầu tự chuyển xuống nhà cung cấp kế tiếp, không báo lỗi cho sinh viên.">
          <div className={s.configPanel}>
          <ol className={s.chain}>
            {FALLBACK_CHAIN.map((name, i) => (
              <li key={name}>
                <span className={s.chainIndex}>{i + 1}</span>
                {name}
                {i === 0 && <span className={s.sub}> · đang dùng</span>}
              </li>
            ))}
          </ol>
          </div>
        </Section>

        <Section
          part="settings-section"
          title="Mô hình embedding"
          description="Tách riêng vì đổi mô hình này buộc phải đánh chỉ mục lại toàn bộ tài liệu, không chỉ đổi cấu hình."
        >
          <div className={s.configPanel}>
          <DefinitionList
            items={[
              {
                term: "Mô hình",
                value: canEdit ? (
                  <Select aria-label="Mô hình embedding" value={embedding} onChange={(e) => changeEmbedding(e.target.value)} className={s.modelSelect}>
                    {EMBEDDING.options.map((o) => (
                      <option key={o} value={o}>
                        {o}
                      </option>
                    ))}
                  </Select>
                ) : (
                  embedding
                ),
              },
              { term: "Số chiều", value: `${EMBEDDING.dims}` },
              { term: "Đã đánh chỉ mục", value: `${EMBEDDING.chunks} · lần cuối ${EMBEDDING.indexedAt}` },
            ]}
          />
          {embedding !== EMBEDDING.model && (
            <InlineNotice tone="warning" title="Phải đánh chỉ mục lại toàn bộ tài liệu">
              {EMBEDDING.chunks} sẽ được tính lại bằng {embedding} (khoảng 25 phút). Trong lúc đó AI vẫn trả lời được nhưng tìm tài liệu kém chính xác hơn, và chi phí của ngày hôm nay tăng thêm khoảng 60.000 đ.
            </InlineNotice>
          )}
          </div>
        </Section>

        <Section part="settings-section" title="Ngân sách" description="Trần chi tiêu cho toàn hệ thống. Chạm trần không tắt chat, chỉ hạ model.">
          <div className={s.configPanel}>
          <BarList
            items={[
              { label: `Hôm nay · ${BUDGET.dayUsed} / ${BUDGET.dayCap}`, value: BUDGET.dayPercent, tone: "ink" },
              { label: `Tháng 10 · ${BUDGET.monthUsed} / ${BUDGET.monthCap}`, value: BUDGET.monthPercent, tone: "amber" },
            ]}
            max={100}
            format={(v) => `${v}%`}
          />
          <p className={s.sub}>{BUDGET.note}</p>
          </div>
        </Section>

        <details className={s.advanced}>
          <summary>Cài đặt nâng cao</summary>
          <DefinitionList items={ADVANCED.map((a) => ({ term: a.term, value: a.value }))} />
        </details>
      </PageState>
    </Page>
  );
}
