import type { Metadata } from "next";
import { Suspense } from "react";
import { AuthShell } from "@/shared/shell/AuthShell";
import { LoginForm } from "./LoginForm";

export const metadata: Metadata = { title: "Đăng nhập" };

export default function LoginPage() {
  return (
    <AuthShell>
      <Suspense fallback={null}>
        <LoginForm />
      </Suspense>
    </AuthShell>
  );
}
