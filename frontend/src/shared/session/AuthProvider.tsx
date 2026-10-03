"use client";

import { useRouter } from "next/navigation";
import { createContext, useCallback, useContext, useEffect, useRef, useSyncExternalStore } from "react";
import { apiClient } from "@/shared/data/apiClient";
import { authStore, dropSession, refreshSession, type AuthSnapshot } from "@/shared/data/authSession";
import { clearDemoSession, hasDemoSession } from "./cookies";

type Auth = AuthSnapshot & {
  /** Đăng xuất thiết bị này: thu hồi phiên ở máy chủ, xoá bộ nhớ, về /login. */
  logout: () => Promise<void>;
};

const AuthContext = createContext<Auth | null>(null);

/** Build công cụ dev: cookie `ep_demo_role` còn thì dùng phiên mô phỏng (không gọi máy chủ) — xem `AuthGate`. */
export const DEV_TOOLS = process.env.NEXT_PUBLIC_DEV_TOOLS === "1";

/**
 * Nguồn phiên đăng nhập duy nhất (US-P2-02): khi tải trang gọi `POST /auth/refresh` MỘT lần để lấy access token mới từ cookie
 * httpOnly (khung xương trong lúc chờ, không nháy /login); sau đó theo dõi sự kiện hết hạn / bị thu hồi từ `apiClient`.
 */
export function AuthProvider({ children }: { children: React.ReactNode }) {
  const snap = useSyncExternalStore(authStore.subscribe, authStore.get, authStore.server);
  const router = useRouter();
  const started = useRef(false);

  useEffect(() => {
    if (started.current) return;
    started.current = true;
    if (DEV_TOOLS && hasDemoSession()) {
      dropSession("anonymous"); // phiên mô phỏng: không đụng máy chủ; AuthGate nhận ra bằng cookie
      return;
    }
    void refreshSession().then((r) => {
      if (!r.ok && authStore.get().status === "initializing") dropSession("anonymous"); // mạng lỗi lúc tải: coi như chưa đăng nhập
    });
  }, []);

  useEffect(() => {
    const onExpired = () => dropSession("anonymous");
    window.addEventListener("auth:expired", onExpired);
    return () => window.removeEventListener("auth:expired", onExpired);
  }, []);

  const logout = useCallback(async () => {
    try {
      await apiClient.post("/auth/logout");
    } catch {
      /* máy chủ không với tới: vẫn xoá phiên ở máy này */
    }
    clearDemoSession();
    router.replace("/login");
    dropSession("anonymous", "logout"); // cổng không thêm ?next= khi chính người dùng đăng xuất
  }, [router]);

  return <AuthContext.Provider value={{ ...snap, logout }}>{children}</AuthContext.Provider>;
}

export function useAuth(): Auth {
  const a = useContext(AuthContext);
  if (!a) throw new Error("useAuth cần nằm trong AuthProvider");
  return a;
}
