import type { Metadata } from "next";
import { Suspense } from "react";
import { AuthShell } from "@/shared/shell/AuthShell";
import { ResetPassword } from "./ResetPassword";

export const metadata: Metadata = { title: "Đặt mật khẩu mới" };

export default function ResetPasswordPage() {
  return (
    <AuthShell>
      <Suspense fallback={null}>
        <ResetPassword />
      </Suspense>
    </AuthShell>
  );
}
