import Link from "next/link";
import { Panel } from "@/shared/ui/Panel";
import { PageHeader } from "@/shared/ui/Layout";
import s from "./AuthShell.module.css";

/** Khung của màn tài khoản (`homeLink`: logo là liên kết về trang giới thiệu — chỉ /login) (đăng nhập, đăng ký, quên mật khẩu…): nền canvas, một cột ≤ 440 px, không thanh bên (SRS FEAT-account-security 7.1, FEAT-ui-panels 7.1). */
export function AuthShell({ children, homeLink }: { children: React.ReactNode; homeLink?: boolean }) {
  return (
    <main className={s.wrap}>
      <div className={s.col}>
        {homeLink ? (
          <Link href="/" aria-label="EduPilot — trang giới thiệu" data-part="brand-link" className={s.brandLink}>
            {/* eslint-disable-next-line @next/next/no-img-element */}
            <img src="/brand/logo-edupilot.svg" alt="EduPilot" data-part="brand" className={s.brand} />
          </Link>
        ) : (
          // eslint-disable-next-line @next/next/no-img-element
          <img src="/brand/logo-edupilot.svg" alt="EduPilot" data-part="brand" className={s.brand} />
        )}
        {children}
      </div>
      <p className={s.foot}>Cần giúp? Liên hệ giảng viên hoặc quản trị viên của bạn.</p>
    </main>
  );
}

/** Một màn của khung đăng nhập: `h1` NGOÀI panel ngay trên nó, nội dung trong MỘT `Panel` (không Panel thứ hai). */
export function AuthPanel({ title, children }: { title: React.ReactNode; children: React.ReactNode }) {
  return (
    <>
      <PageHeader title={title} />
      <Panel>
        <div className={s.stack}>{children}</div>
      </Panel>
    </>
  );
}
