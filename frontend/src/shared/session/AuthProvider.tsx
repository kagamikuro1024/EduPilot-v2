"use client";

import { useRouter } from "next/navigation";
import { createContext, useCallback, useContext, useDeferredValue, useEffect, useRef, useSyncExternalStore } from "react";
import { apiClient } from "@/shared/data/apiClient";
import { authStore, dropSession, refreshSession, type AuthSnapshot } from "@/shared/data/authSession";

type Auth = AuthSnapshot & {
  /** Đăng xuất thiết bị này: thu hồi phiên ở máy chủ, xoá bộ nhớ, về /login. */
  logout: () => Promise<void>;
};

const AuthContext = createContext<Auth | null>(null);

/**
 * Nguồn phiên đăng nhập duy nhất (US-P2-02): khi tải trang gọi `POST /auth/refresh` MỘT lần để lấy access token mới từ cookie
 * httpOnly (khung xương trong lúc chờ, không nháy /login); sau đó theo dõi sự kiện hết hạn / bị thu hồi từ `apiClient`.
 */
export function AuthProvider({ children }: { children: React.ReactNode }) {
  // `useSyncExternalStore` luôn cập nhật ĐỒNG BỘ: khi refresh xong, cả khung + trang dựng trong MỘT tác vụ dài (TBT). Giá trị hoãn cho React
  // dựng cây đó theo lát thời gian (ngắt được) thay vì một khối; token vẫn đọc đồng bộ ở nơi khác (`tokenStore`).
  const snap = useDeferredValue(useSyncExternalStore(authStore.subscribe, authStore.get, authStore.server));
  const router = useRouter();
  const started = useRef(false);

  useEffect(() => {
    if (started.current) return;
    started.current = true;
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
