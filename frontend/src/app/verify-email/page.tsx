import type { Metadata } from "next";
import { Suspense } from "react";
import { AuthShell } from "@/shared/shell/AuthShell";
import { VerifyEmail } from "./VerifyEmail";

export const metadata: Metadata = { title: "Xác minh email" };

export default function VerifyEmailPage() {
  return (
    <AuthShell>
      <Suspense fallback={null}>
        <VerifyEmail />
      </Suspense>
    </AuthShell>
  );
}
