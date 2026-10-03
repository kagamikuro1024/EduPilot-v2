import type { Metadata } from "next";
import { AuthShell } from "@/shared/shell/AuthShell";
import { ForgotForm } from "./ForgotForm";

export const metadata: Metadata = { title: "Quên mật khẩu" };

export default function ForgotPasswordPage() {
  return (
    <AuthShell>
      <ForgotForm />
    </AuthShell>
  );
}
