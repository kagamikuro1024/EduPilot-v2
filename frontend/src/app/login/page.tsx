import type { Metadata } from "next";
import { LoginChoices } from "./LoginChoices";
import s from "./login.module.css";

export const metadata: Metadata = { title: "Đăng nhập" };

export default function LoginPage() {
  return (
    <main className={s.wrap}>
      {/* eslint-disable-next-line @next/next/no-img-element */}
      <img src="/brand/logo-edupilot.svg" alt="EduPilot" height={40} className={s.logo} />
      <h1 className="ep-page-title">Đăng nhập</h1>
      <p className={s.lede}>Đây là bản mô phỏng: mọi dữ liệu đều là giả. Chọn một tài khoản để xem EduPilot theo vai trò đó.</p>
      <LoginChoices />
      <p className={s.foot}>Bản mô phỏng · dữ liệu giả</p>
    </main>
  );
}
