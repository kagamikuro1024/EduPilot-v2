import type { Metadata } from "next";
import { AuthShell } from "@/shared/shell/AuthShell";
import { RegisterForm } from "./RegisterForm";

export const metadata: Metadata = { title: "Tạo tài khoản" };

export default function RegisterPage() {
  return (
    <AuthShell>
      <RegisterForm />
    </AuthShell>
  );
}
