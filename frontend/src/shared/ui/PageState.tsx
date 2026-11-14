"use client";

import { RefreshCw } from "lucide-react";
import { usePathname, useRouter, useSearchParams } from "next/navigation";
import type { ReactNode } from "react";
import { ApiErrorNotice } from "@/shared/data/ApiErrorNotice";
import { Button } from "./Button";
import { EmptyState, InlineNotice, Skeleton } from "./Feedback";

export type RouteState = "loading" | "empty" | "error";

/** Đọc `?state=loading|empty|error` để minh hoạ trạng thái (SRS FR-X5). Không có tham số → null. */
export function useRouteState(): RouteState | null {
  const v = useSearchParams().get("state");
  if (process.env.NEXT_PUBLIC_MOCK_SCREENS === "0") return null; // ?state= chỉ còn cho màn mock
  return v === "loading" || v === "empty" || v === "error" ? v : null;
}

/**
 * Bao vùng nội dung của một route. Khi URL có `?state=…` thay `children` bằng trạng thái tương ứng:
 * loading = `loading` (skeleton đúng hình, mặc định 5 dòng), empty = `empty` (`EmptyState`: tiêu đề + tối đa một nút, DESIGN.md §23.4),
 * error = MỘT dòng nói chuyện gì hỏng + `Thử lại` (§23.5; thêm «Chữ bạn đã nhập vẫn còn» chỉ khi màn đang giữ chữ chưa lưu: `unsaved`).
 * Truyền `state` để ép trạng thái thật.
 */
export function PageState({
  children,
  loading,
  empty,
  error,
  unsaved,
  state,
  query,
  isEmpty,
  showTechnical,
}: {
  children: ReactNode;
  loading?: ReactNode;
  /** nội dung rỗng; thường là <EmptyState title action>…</EmptyState> */
  empty?: ReactNode;
  /** thay câu lỗi chuẩn bằng câu riêng của màn: một dòng nói chuyện gì hỏng */
  error?: { problem: ReactNode };
  /** màn đang giữ chữ người dùng chưa lưu: thêm «Chữ bạn đã nhập vẫn còn» (P-107-3) */
  unsaved?: boolean;
  state?: RouteState | null;
  /** truy vấn thật (TanStack Query): isPending → khung xương; isError → lỗi chuẩn từ ApiError + Thử lại gọi refetch; rỗng → `empty` */
  query?: { isPending: boolean; isError: boolean; error: unknown; data: unknown; refetch: () => unknown; isRefetching?: boolean };
  isEmpty?: (data: unknown) => boolean;
  /** TA / GV / Admin: hiện "Chi tiết kỹ thuật" (trace_id) trong lỗi */
  showTechnical?: boolean;
}) {
  const fromUrl = useRouteState();
  const router = useRouter();
  const pathname = usePathname();
  if (query) {
    if (query.isPending) return <>{loading ?? <Skeleton lines={5} />}</>;
    if (query.isError) {
      return <ApiErrorNotice error={query.error} showTechnical={showTechnical} onRetry={() => void query.refetch()} />;
    }
    if (isEmpty?.(query.data)) return <>{empty ?? <EmptyState title="Chưa có dữ liệu" />}</>;
    return <>{children}</>;
  }
  const s = state === undefined ? fromUrl : state;
  if (s === "loading") return <>{loading ?? <Skeleton lines={5} />}</>;
  if (s === "empty") return <>{empty ?? <EmptyState title="Chưa có dữ liệu" />}</>;
  if (s === "error") {
    return (
      <InlineNotice
        tone="danger"
        title={error?.problem ?? "Không tải được nội dung"}
        action={
          <Button size="sm" icon={<RefreshCw aria-hidden />} onClick={() => router.replace(pathname)}>
            Thử lại
          </Button>
        }
      >
        {unsaved ? "Chữ bạn đã nhập vẫn còn" : null}
      </InlineNotice>
    );
  }
  return <>{children}</>;
}
