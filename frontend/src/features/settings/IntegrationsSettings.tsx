"use client";

import { useState } from "react";
import { INTEGRATIONS } from "@/mock/system";
import { useSession } from "@/shared/session/session";
import { Button, DefinitionList, InlineNotice, Page, PageHeader, PageState, Skeleton, StatusText } from "@/shared/ui";
import s from "./settings.module.css";

export function IntegrationsSettings() {
  const { role } = useSession();
  const canEdit = role === "admin";
  const [result, setResult] = useState<Record<string, boolean>>({});

  return (
    <Page>
      <PageHeader
        title="Tích hợp"
        description="Thư đi, thư đến và Teams — ba đường EduPilot nối ra ngoài trường."
        meta={<span>Kiểm gần nhất 08:55 hôm nay</span>}
      />

      <PageState
        loading={
          <div className={s.loading}>
            <Skeleton lines={4} />
            <Skeleton lines={4} />
          </div>
        }
        error={{
          problem: "Không đọc được trạng thái tích hợp.",
          recovery: "Các kết nối đang có vẫn chạy, thư không bị mất. Thử lại sau ít phút.",
        }}
      >
        {!canEdit && (
          <InlineNotice tone="info" compact>
            Chỉ quản trị viên hệ thống đổi được tích hợp. Bạn xem để biết thư thông báo và bài nộp qua email có đang chạy không.
          </InlineNotice>
        )}

        {INTEGRATIONS.map((it) => (
          <section key={it.id} className={s.integration}>
            <div className={s.integrationHead}>
              <div>
                <h2 className="ep-section-title">{it.name}</h2>
                <p className={s.integrationPurpose}>{it.purpose}</p>
              </div>
              <StatusText tone={it.state === "connected" ? "green" : it.state === "waiting" ? "amber" : "neutral"}>{it.stateText}</StatusText>
            </div>

            <DefinitionList items={it.fields.map((f) => ({ term: f.term, value: f.value }))} />

            {canEdit && (
              <div className={s.inlineFormActions}>
                <Button size="sm" onClick={() => setResult((prev) => ({ ...prev, [it.id]: true }))}>
                  {it.actionLabel}
                </Button>
              </div>
            )}

            {result[it.id] &&
              (it.result.ok ? (
                <StatusText tone="green">{it.result.text}</StatusText>
              ) : (
                <InlineNotice tone="warning" title={it.state === "unset" ? "Chưa cấu hình nên chưa kiểm được" : "Chưa dùng được vì còn chờ bên ngoài"}>
                  {it.result.text}
                </InlineNotice>
              ))}
          </section>
        ))}
      </PageState>
    </Page>
  );
}
