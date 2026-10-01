import type { LucideIcon } from "lucide-react";
import {
  AlertTriangle,
  BarChart3,
  Calendar,
  FileText,
  LayoutList,
  MessageSquare,
} from "lucide-react";
import type { UserRole } from "@/types/auth";

export interface WorkspaceNavLinkDef {
  href: string;
  label: string;
  Icon: LucideIcon;
  /** Return true when this nav entry should show the active (blue) state */
  isActive: (pathname: string) => boolean;
}

export function getWorkspaceNavLinks(role: UserRole | null): WorkspaceNavLinkDef[] {
  const staff = role === "TA" || role === "ADMIN";
  if (staff) {
    return [
      {
        href: "/assignments",
        label: "Assignments",
        Icon: Calendar,
        isActive: (p) => p.startsWith("/assignments"),
      },
      {
        href: "/threads",
        label: "Threads",
        Icon: LayoutList,
        isActive: (p) => p.startsWith("/threads"),
      },
      {
        href: "/documents",
        label: "Documents",
        Icon: FileText,
        isActive: (p) => p.startsWith("/documents"),
      },
      {
        href: "/analytics",
        label: "Analytics",
        Icon: BarChart3,
        isActive: (p) => p.startsWith("/analytics"),
      },
      {
        href: "/at-risk",
        label: "At-Risk Students",
        Icon: AlertTriangle,
        isActive: (p) => p.startsWith("/at-risk"),
      },
    ];
  }
  return [
    {
      href: "/assignments",
      label: "Assignments",
      Icon: Calendar,
      isActive: (p) => p.startsWith("/assignments"),
    },
    {
      href: "/chat",
      label: "Chat",
      Icon: MessageSquare,
      isActive: (p) => p.startsWith("/chat"),
    },
    {
      href: "/threads",
      label: "Threads",
      Icon: LayoutList,
      isActive: (p) => p.startsWith("/threads"),
    },
    {
      href: "#",
      label: "Documents",
      Icon: FileText,
      isActive: () => false,
    },
  ];
}
