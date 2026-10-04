import type { Metadata } from "next";
import { SecuritySettings } from "@/features/settings/security/SecuritySettings";

export const metadata: Metadata = { title: "Tài khoản và bảo mật" };

export default function Page() {
  return <SecuritySettings />;
}
