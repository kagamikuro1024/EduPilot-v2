"use client";

import Link from "next/link";
import { Brain } from "lucide-react";
import { usePathname } from "next/navigation";
import type { ReactNode } from "react";
import { useState } from "react";
import { useUiStore } from "@/store/uiStore";
import { Bell, LogOut, Settings, ChevronLeft, ChevronRight, User, Shield, HelpCircle, Keyboard } from "lucide-react";
import { getWorkspaceNavLinks } from "@/config/workspaceNav";
import { useAuthStore } from "@/store/authStore";
import type { UserRole } from "@/types/auth";
import AuthenticatedAvatar from "@/components/profile/AuthenticatedAvatar";

export type WorkspaceFooterLine2 = "operational" | "rights";

export interface WorkspaceStudentProfile {
  displayName: string;
  courseLabel: string;
  avatarInitials: string;
}

export interface WorkspaceLayoutProps {
  children: ReactNode;
  sidePanel?: ReactNode;
  sidePanelClassName?: string;
  searchPlaceholder?: string;
  showNotifications?: boolean;
  footerLine2?: WorkspaceFooterLine2;
  footerFloating?: ReactNode;
  studentProfile?: WorkspaceStudentProfile;
  avatarInitialsOverride?: string;
}

function roleBadgeLabel(role: UserRole | null): string {
  if (role === "TA") return "TA";
  if (role === "ADMIN") return "ADMIN";
  if (role === "STUDENT") return "STUDENT";
  return "STUDENT";
}

function defaultAvatarLetters(
  role: UserRole | null,
  studentProfile: WorkspaceStudentProfile | undefined,
  override: string | undefined,
  fullName: string | null,
): string {
  if (override) return override.slice(0, 2).toUpperCase();
  if (studentProfile) return studentProfile.avatarInitials.slice(0, 2).toUpperCase();
  if (fullName) {
    const initials = fullName
      .trim()
      .split(/\s+/)
      .slice(0, 2)
      .map((w) => w[0]?.toUpperCase())
      .join("");
    if (initials) return initials;
  }
  if (role === "TA") return "TA";
  if (role === "ADMIN") return "AD";
  return "ST";
}

function WorkspaceNavRow({
  href,
  label,
  Icon,
  active,
  collapsed,
}: {
  href: string;
  label: string;
  Icon: React.ComponentType<{ className?: string }>;
  active: boolean;
  collapsed: boolean;
}) {
  const className = `flex items-center gap-3 rounded-lg px-3 py-2 text-sm font-medium transition ${
    active
      ? "bg-blue-50 text-blue-600 ring-1 ring-blue-100"
      : "text-slate-600 hover:bg-slate-100 hover:text-slate-900"
  }`;

  const content = (
    <>
      <Icon className="h-4 w-4 shrink-0" />
      {!collapsed && <span>{label}</span>}
    </>
  );

  if (href === "#") {
    return (
      <span className={`${className} cursor-not-allowed opacity-70`} title={collapsed ? label : undefined}>
        {content}
      </span>
    );
  }

  return (
    <Link href={href} className={className} title={collapsed ? label : undefined}>
      {content}
    </Link>
  );
}

export function WorkspaceLayout({
  children,
  sidePanel,
  sidePanelClassName = "hidden w-80 shrink-0 overflow-y-auto border-l border-slate-200 bg-white lg:block",
  searchPlaceholder = "Search threads...",
  showNotifications = true,
  footerLine2 = "operational",
  footerFloating,
  studentProfile,
  avatarInitialsOverride,
}: WorkspaceLayoutProps) {
  const pathname = usePathname() ?? "";
  const role = useAuthStore((s) => s.role);
  const fullName = useAuthStore((s) => s.fullName);
  const avatarAvailable = useAuthStore((s) => s.avatarAvailable);
  const avatarVersion = useAuthStore((s) => s.avatarVersion);
  const logout = useAuthStore((s) => s.logout);
  const navLinks = getWorkspaceNavLinks(role);
  const avatarLetters = defaultAvatarLetters(role, studentProfile, avatarInitialsOverride, fullName);

  const sidebarCollapsed = useUiStore((s) => s.sidebarCollapsed);
  const toggleSidebar = useUiStore((s) => s.toggleSidebar);
  const [userMenuOpen, setUserMenuOpen] = useState(false);

  return (
    <div className="relative flex h-screen min-h-0 flex-col bg-slate-50 text-slate-900">
      <header className="flex h-14 shrink-0 items-center gap-4 border-b border-slate-200 bg-white px-6 shadow-sm">
        <Link href="/" className="flex items-center gap-3 group">
          <div className="relative flex h-11 w-11 shrink-0 items-center justify-center rounded-xl bg-gradient-to-br from-indigo-600 to-violet-600 shadow-lg shadow-indigo-500/25 transition-all duration-300 group-hover:scale-105 group-hover:shadow-xl">
            <Brain className="h-6 w-6 text-white drop-shadow" strokeWidth={2.5} />
          </div>
          <div className="flex flex-col">
            <span className="bg-gradient-to-r from-indigo-600 to-violet-600 bg-clip-text text-lg font-black text-transparent">
              EduPilot
            </span>
            <span className="text-[9px] font-medium uppercase tracking-wider text-slate-400">
              AI Tutor
            </span>
          </div>
        </Link>
        <div className="flex flex-1 items-center justify-end gap-3">
          <span className="rounded-md bg-slate-100 px-2 py-1 text-xs font-semibold uppercase tracking-wide text-slate-600 ring-1 ring-slate-200">
            {roleBadgeLabel(role)}
          </span>
          {showNotifications && (
            <button
              type="button"
              className="relative rounded-full p-2 text-slate-500 transition hover:bg-slate-100 hover:text-slate-800"
              aria-label="Notifications"
            >
              <Bell className="h-5 w-5" />
              <span className="absolute top-1 right-1 h-2.5 w-2.5 rounded-full bg-red-500"></span>
            </button>
          )}
          <div className="relative">
            <button
              type="button"
              onClick={() => setUserMenuOpen(!userMenuOpen)}
              className={`group relative flex items-center gap-2 rounded-full p-0.5 transition-all duration-300 ${
                userMenuOpen
                  ? "ring-2 ring-indigo-500 ring-offset-2 ring-offset-white"
                  : "hover:ring-2 hover:ring-indigo-400/50 hover:ring-offset-2 hover:ring-offset-white"
              }`}
            >
              <div className="relative">
                <AuthenticatedAvatar
                  fullName={fullName}
                  avatarAvailable={avatarAvailable}
                  avatarVersion={avatarVersion}
                  className="h-9 w-9 text-sm font-bold"
                />
                {/* Online status dot */}
                <span className="absolute -bottom-0.5 -right-0.5 flex h-3.5 w-3.5 items-center justify-center rounded-full border-2 border-white bg-emerald-500">
                  <span className="h-1.5 w-1.5 animate-ping rounded-full bg-emerald-300" />
                </span>
              </div>
            </button>

            {/* Dropdown Menu */}
            {userMenuOpen && (
              <>
                <div
                  className="fixed inset-0 z-40"
                  onClick={() => setUserMenuOpen(false)}
                />
                <div className="absolute right-0 top-full mt-3 w-72 z-50 animate-in fade-in slide-in-from-top-2 duration-200">
                  {/* Dropdown arrow */}
                  <div className="absolute -top-1.5 right-4 h-3 w-3 rotate-45 rounded-sm border-l border-t border-white/80 bg-gradient-to-br from-indigo-600 to-violet-600" />
                  
                  <div className="overflow-hidden rounded-2xl border border-white/20 bg-white shadow-2xl shadow-slate-900/20">
                    {/* User Profile Header — Gradient Banner */}
                    <div className="relative overflow-hidden bg-gradient-to-br from-indigo-600 via-violet-600 to-purple-700 px-5 py-5">
                      {/* Decorative circles */}
                      <div className="absolute -right-6 -top-6 h-24 w-24 rounded-full bg-white/10" />
                      <div className="absolute -left-4 -bottom-8 h-20 w-20 rounded-full bg-white/5" />
                      
                      <div className="relative flex items-center gap-3.5">
                        <div className="shrink-0">
                          <AuthenticatedAvatar
                            fullName={fullName}
                            avatarAvailable={avatarAvailable}
                            avatarVersion={avatarVersion}
                            className="h-12 w-12 text-sm font-bold"
                          />
                        </div>
                        <div className="min-w-0 flex-1">
                          <p className="truncate text-sm font-bold text-white">
                            {studentProfile?.displayName || fullName || "User"}
                          </p>
                          {studentProfile?.courseLabel && (
                            <p className="truncate text-xs text-indigo-200">{studentProfile.courseLabel}</p>
                          )}
                          <div className="mt-1 inline-flex items-center gap-1.5 rounded-full bg-white/15 px-2 py-0.5 backdrop-blur-sm">
                            <span className="h-1.5 w-1.5 rounded-full bg-emerald-400 shadow-sm shadow-emerald-400/50" />
                            <span className="text-[10px] font-semibold text-white/90">Online</span>
                          </div>
                        </div>
                      </div>
                    </div>

                    {/* Menu Items */}
                    <div className="p-1.5">
                      {/* Profile */}
                      <Link
                        href="/profile"
                        className="group flex items-center gap-3 rounded-xl px-3.5 py-2.5 transition-all duration-150 hover:bg-indigo-50"
                        onClick={() => setUserMenuOpen(false)}
                      >
                        <div className="flex h-8 w-8 items-center justify-center rounded-lg bg-indigo-100 text-indigo-600 transition-colors group-hover:bg-indigo-200">
                          <User className="h-4 w-4" />
                        </div>
                        <div className="flex-1">
                          <p className="text-sm font-semibold text-slate-800">My Profile</p>
                          <p className="text-[11px] text-slate-400">View and edit profile</p>
                        </div>
                      </Link>

                      {/* Settings */}
                      <Link
                        href="/settings"
                        className="group flex items-center gap-3 rounded-xl px-3.5 py-2.5 transition-all duration-150 hover:bg-slate-50"
                        onClick={() => setUserMenuOpen(false)}
                      >
                        <div className="flex h-8 w-8 items-center justify-center rounded-lg bg-slate-100 text-slate-600 transition-colors group-hover:bg-slate-200">
                          <Settings className="h-4 w-4" />
                        </div>
                        <div className="flex-1">
                          <p className="text-sm font-semibold text-slate-800">Settings</p>
                          <p className="text-[11px] text-slate-400">Preferences & config</p>
                        </div>
                      </Link>

                      {/* Role Badge */}
                      <div className="group flex items-center gap-3 rounded-xl px-3.5 py-2.5">
                        <div className="flex h-8 w-8 items-center justify-center rounded-lg bg-violet-100 text-violet-600">
                          <Shield className="h-4 w-4" />
                        </div>
                        <div className="flex-1">
                          <p className="text-sm font-semibold text-slate-800">Role</p>
                          <p className="text-[11px] text-slate-400">{roleBadgeLabel(role)}</p>
                        </div>
                        <span className="rounded-full bg-violet-100 px-2.5 py-0.5 text-[10px] font-bold uppercase tracking-wider text-violet-700">
                          {roleBadgeLabel(role)}
                        </span>
                      </div>

                      {/* Keyboard shortcuts */}
                      <Link
                        href="#"
                        className="group flex items-center gap-3 rounded-xl px-3.5 py-2.5 transition-all duration-150 hover:bg-slate-50"
                        onClick={(e) => { e.preventDefault(); setUserMenuOpen(false); }}
                      >
                        <div className="flex h-8 w-8 items-center justify-center rounded-lg bg-amber-100 text-amber-600 transition-colors group-hover:bg-amber-200">
                          <Keyboard className="h-4 w-4" />
                        </div>
                        <div className="flex-1">
                          <p className="text-sm font-semibold text-slate-800">Shortcuts</p>
                          <p className="text-[11px] text-slate-400">Keyboard shortcuts</p>
                        </div>
                        <kbd className="rounded-md bg-slate-100 px-1.5 py-0.5 text-[10px] font-mono font-semibold text-slate-500">?</kbd>
                      </Link>
                    </div>

                    {/* Divider */}
                    <div className="mx-4 border-t border-slate-100" />

                    {/* Sign Out */}
                    <div className="p-1.5">
                      <button
                        type="button"
                        onClick={() => {
                          logout();
                          setUserMenuOpen(false);
                        }}
                        className="group flex w-full items-center gap-3 rounded-xl px-3.5 py-2.5 transition-all duration-150 hover:bg-red-50"
                      >
                        <div className="flex h-8 w-8 items-center justify-center rounded-lg bg-red-100 text-red-500 transition-colors group-hover:bg-red-200">
                          <LogOut className="h-4 w-4" />
                        </div>
                        <div className="flex-1 text-left">
                          <p className="text-sm font-semibold text-red-600">Sign Out</p>
                          <p className="text-[11px] text-red-400">Log out of your account</p>
                        </div>
                      </button>
                    </div>

                    {/* Footer */}
                    <div className="border-t border-slate-100 bg-slate-50/50 px-4 py-2.5 text-center">
                      <p className="text-[10px] font-medium text-slate-400">
                        EduPilot AI v1.0 • <span className="text-emerald-500">All systems operational</span>
                      </p>
                    </div>
                  </div>
                </div>
              </>
            )}
          </div>
        </div>
      </header>

      <div className="flex min-h-0 flex-1">
        <aside
          className={`flex flex-col border-r border-slate-200 bg-white transition-all duration-300 ${
            sidebarCollapsed ? "w-16" : "w-64"
          }`}
        >
          <div className="flex items-center justify-between p-4 border-b border-slate-100">
            {!sidebarCollapsed && (
              <p className="text-xs font-semibold uppercase tracking-wide text-slate-400">Workspace</p>
            )}
            <button
              type="button"
              onClick={toggleSidebar}
              className="rounded-lg p-1 text-slate-400 hover:bg-slate-100 hover:text-slate-600 transition"
            >
              {sidebarCollapsed ? <ChevronRight className="h-4 w-4" /> : <ChevronLeft className="h-4 w-4" />}
            </button>
          </div>
          <div className="flex flex-1 flex-col gap-1 p-2 overflow-y-auto">
            {navLinks.map((item) => (
              <WorkspaceNavRow
                key={item.label}
                href={item.href}
                label={item.label}
                Icon={item.Icon}
                active={item.isActive(pathname)}
                collapsed={sidebarCollapsed}
              />
            ))}
          </div>
          {!sidebarCollapsed && (
            <div className="border-t border-slate-100 p-4 text-xs text-slate-400">
              <p>© 2024 EduPilot AI</p>
              {footerLine2 === "operational" ? (
                <p className="mt-1 text-emerald-600">All systems operational.</p>
              ) : (
                <p className="mt-1 text-slate-500">All rights reserved.</p>
              )}
            </div>
          )}
        </aside>

        <div className="flex min-w-0 flex-1">
          <main className="min-w-0 flex-1 overflow-y-auto">{children}</main>
          {sidePanel != null && <aside className={sidePanelClassName}>{sidePanel}</aside>}
        </div>
      </div>

      {footerFloating}
    </div>
  );
}