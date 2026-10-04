import type { Metadata } from "next";
import { AuthShell } from "@/shared/shell/AuthShell";
import { InviteAccept } from "./InviteAccept";

export const metadata: Metadata = { title: "Lời mời tham gia EduPilot" };

export default function InvitePage() {
  return (
    <AuthShell>
      <InviteAccept />
    </AuthShell>
  );
}
