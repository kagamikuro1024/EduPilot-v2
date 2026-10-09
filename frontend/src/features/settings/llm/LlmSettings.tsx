"use client";

import { useState } from "react";
import { ApiError, ApiErrorNotice } from "@/shared/data";
import { InlineNotice, Page, PageHeader, Panel, Section, Skeleton } from "@/shared/ui";
import { useSession } from "@/shared/session/session";
import { EmbeddingSection } from "./EmbeddingSection";
import { ProviderSection } from "./ProviderSection";
import { FallbackSection, RouteTableSection } from "./RoutesSection";
import { UsageSection } from "./UsageSection";
import { useProviders, useRoutes } from "./api";
import s from "./llm.module.css";

/**
 * /settings/llm — màn THẬT (US-P1-05): dữ liệu qua apiClient + TanStack Query, JWT. Admin sửa; Giảng viên chỉ xem.
 * Sinh viên / TA bị khung chặn trước khi tới đây (nav.ts `canOpen`) nên không gọi API.
 */
export function LlmSettings() {
  const { role } = useSession();
  const canEdit = role === "admin";
  const providers = useProviders();
  const routes = useRoutes();
  const [adding, setAdding] = useState(0); // tăng để mở form thêm từ bảng rỗng của phần 2

  const notConfigured = [providers.error, routes.error].some((e) => e instanceof ApiError && e.code === "LLM_NOT_CONFIGURED");
  const failed = !notConfigured && (providers.isError ? providers : routes.isError ? routes : null);
  const pending = !notConfigured && (providers.isPending || routes.isPending);

  return (
    <Page width="wide">
      <PageHeader title="Cấu hình LLM" description="Tác vụ nào chạy bằng mô hình nào, hỏng thì chuyển sang đâu, và tiêu bao nhiêu tiền." />
      {!canEdit && <InlineNotice tone="info">Chỉ quản trị viên được thay đổi cấu hình.</InlineNotice>}
      {pending ? (
        <Panel>
          <Skeleton lines={8} />
        </Panel>
      ) : failed ? (
        <ApiErrorNotice error={failed.error} title="Chưa tải được cấu hình." context="Cấu hình hiện có không bị ảnh hưởng." showTechnical onRetry={() => void failed.refetch()} />
      ) : (
        <>
          <Section panel part="settings-section" title="Kết nối nhà cung cấp" description="Khoá API chỉ ghi được: lưu xong không đọc lại được." className={s.section}>
            <ProviderSection key={adding} initialAdd={adding > 0} providers={providers.data?.items ?? []} canEdit={canEdit} envFallback={providers.data?.env_fallback ?? { active: true, providers: [] }} />
          </Section>
          <Section panel part="settings-section" title="Mô hình theo tác vụ" description="Đổi ở đây có hiệu lực cho yêu cầu tiếp theo; yêu cầu đang chạy vẫn dùng mô hình cũ." className={s.section}>
            <RouteTableSection routes={routes.data?.items ?? []} providers={providers.data?.items ?? []} canEdit={canEdit} onAddProvider={() => setAdding((n) => n + 1)} />
          </Section>
          <Section panel part="settings-section" title="Chuỗi dự phòng" description="Khi mô hình chính hỏng, yêu cầu tự chuyển xuống mô hình kế tiếp, không báo lỗi cho người dùng." className={s.section}>
            <FallbackSection routes={routes.data?.items ?? []} providers={providers.data?.items ?? []} canEdit={canEdit} />
          </Section>
          <Section panel part="settings-section" title="Mô hình tìm kiếm tài liệu" className={`${s.section} ${s.embed}`}>
            {routes.data ? <EmbeddingSection routes={routes.data} providers={providers.data?.items ?? []} canEdit={canEdit} /> : <Skeleton lines={2} />}
          </Section>
          <Section panel part="usage-section" title="Mức dùng và ngân sách" description="Chi phí ước tính từ số token đã dùng; con số chính thức theo hoá đơn của nhà cung cấp." className={s.section}>
            <UsageSection canEdit={canEdit} />
          </Section>
        </>
      )}
    </Page>
  );
}
