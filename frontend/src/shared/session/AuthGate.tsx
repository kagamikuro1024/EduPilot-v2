"use client";

import { usePathname, useRouter, useSearchParams } from "next/navigation";
import { Suspense, useEffect, useMemo, useSyncExternalStore } from "react";
import { Skeleton } from "@/shared/ui";
import { DEV_TOOLS, useAuth } from "./AuthProvider";
import { parseDemoCookies } from "./cookies";
import { SessionProvider } from "./session";

const noopSubscribe = () => () => {};

/** Khung xương toàn trang trong lúc xác định phiên (không nháy /login). */
export function AuthSkeleton() {
  return (
    <div style={{ padding: "var(--ep-space-12) var(--ep-space-6)", maxWidth: 960, margin: "0 auto" }}>
      <Skeleton lines={5} />
    </div>
  );
}

/**
 * Cổng của mọi route cần đăng nhập: chưa biết phiên → khung xương; chưa đăng nhập / bị thu hồi → `/login?next=…`;
 * đã đăng nhập → SessionProvider (vai từ JWT). Build NEXT_PUBLIC_DEV_TOOLS=1 còn nhận phiên mô phỏng từ cookie khi chưa có phiên thật.
 */
export function AuthGate({ children }: { children: React.ReactNode }) {
  return (
    <Suspense fallback={<AuthSkeleton />}>
      <Gate>{children}</Gate>
    </Suspense>
  );
}

function Gate({ children }: { children: React.ReactNode }) {
  const auth = useAuth();
  const router = useRouter();
  const pathname = usePathname();
  const search = useSearchParams().toString();
  // cookie chỉ đọc được ở trình duyệt: server/hydrate thấy `null` (chưa kiểm) → `demo` undefined, tránh lệch hydrate
  const rawCookie = useSyncExternalStore(noopSubscribe, () => document.cookie, () => null);
  const demo = useMemo(() => (rawCookie === null ? undefined : DEV_TOOLS ? parseDemoCookies(rawCookie) : null), [rawCookie]);

  const useDemo = auth.status !== "authenticated" && demo != null;
  const bounce = (auth.status === "anonymous" || auth.status === "revoked") && demo === null;

  useEffect(() => {
    if (!bounce) return;
    if (auth.status === "anonymous" && auth.reason === "logout") return; // AuthProvider.logout đã đưa về /login
    const here = `${pathname}${search ? `?${search}` : ""}`;
    const q = new URLSearchParams({ next: here });
    if (auth.status === "revoked") q.set("revoked", auth.reason ?? "1");
    router.replace(`/login?${q.toString()}`);
  }, [bounce, auth.status, auth.reason, pathname, search, router]);

  if (auth.status === "authenticated") return <SessionProvider fullName={auth.user?.full_name}>{children}</SessionProvider>;
  if (useDemo) return <SessionProvider demo={demo}>{children}</SessionProvider>;
  return <AuthSkeleton />;
}
