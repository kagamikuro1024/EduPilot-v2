import s from "./AuthShell.module.css";

/** Khung của màn tài khoản (đăng nhập, đăng ký, quên mật khẩu…): một cột ≤ 420 px, không thanh bên (SRS FEAT-account-security 7.1). */
export function AuthShell({ children }: { children: React.ReactNode }) {
  return (
    <main className={s.wrap}>
      <div className={s.col}>
        {/* eslint-disable-next-line @next/next/no-img-element */}
        <img src="/brand/logo-edupilot.svg" alt="EduPilot" data-part="brand" className={s.brand} />
        {children}
      </div>
      <p className={s.foot}>Cần giúp? Liên hệ giảng viên hoặc quản trị viên của bạn.</p>
    </main>
  );
}
