"use client";

import dynamic from "next/dynamic";
import { useEffect } from "react";
import { useAuth } from "@/shared/session/AuthProvider";
import s from "./Landing.module.css";
import { isReturning, setReturning } from "@/shared/session/returning";

// Khung ứng dụng nạp riêng: khách vào `/` không tải mã khung / phiên / dữ liệu mô phỏng. Người quay lại (có cờ) nạp trước ngay khi trang chạy để không chờ thêm sau khi phiên xác nhận.
const loadAuthed = () => import("./AuthedHome");
const AuthedHome = dynamic(loadAuthed, { loading: () => null });

/**
 * `/`: đã đăng nhập → khung ứng dụng + "Hôm nay"; còn lại → giới thiệu (máy chủ đã vẽ sẵn, truyền vào dạng `landing`) hoặc khung "Hôm nay" chờ phiên (`pre`) cho người quay lại.
 * Hai khung cùng nằm trong HTML; CSS chọn khung nào hiện theo `html[data-returning]` (đặt bởi script trước khi vẽ). Phiên xác định là chưa đăng nhập thì gỡ dấu → giới thiệu hiện.
 */
export function HomeGate({ landing, pre }: { landing: React.ReactNode; pre: React.ReactNode }) {
  const auth = useAuth();
  const anonymous = auth.status === "anonymous" || auth.status === "revoked";
  useEffect(() => {
    if (anonymous) setReturning(false);
  }, [anonymous]);
  useEffect(() => {
    if (isReturning()) void loadAuthed();
  }, []);
  useEffect(() => {
    if (auth.status === "authenticated") document.title = "Hôm nay · EduPilot";
  }, [auth.status]);

  if (auth.status === "authenticated") {
    return <AuthedHome fullName={auth.user?.full_name} />;
  }
  return (
    <>
      <div className={s.landingRoot} data-part="landing-root">{landing}</div>
      <div className={s.preRoot} data-part="home-pre">{pre}</div>
    </>
  );
}
