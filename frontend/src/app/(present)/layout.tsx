import { AuthGate } from "@/shared/session/AuthGate";

// Route trình chiếu NGOÀI khung ứng dụng (DESIGN §14.8.1, P-108-8): cùng cổng đăng nhập nhưng không có AppShell (không thanh bên, thanh trên, `Page`).
export default function PresentLayout({ children }: { children: React.ReactNode }) {
  return <AuthGate>{children}</AuthGate>;
}
