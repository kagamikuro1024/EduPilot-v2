"use client";

import { useState, useEffect } from "react";
import { toast } from "sonner";
import {
  Settings,
  Paintbrush,
  Accessibility,
  Compass,
  Lock,
  LogOut,
  Loader2,
  Eye,
  EyeOff,
  User,
  ShieldAlert,
} from "lucide-react";
import { WorkspaceLayout } from "@/components/layout/WorkspaceLayout";
import { useAuthStore } from "@/store/authStore";
import {
  usePreferenceStore,
  ThemePreference,
  FontSizePreference,
  DefaultPagePreference,
} from "@/store/preferenceStore";
import javaClient from "@/services/javaClient";
import Link from "next/link";

export default function SettingsPage() {
  const { role, logout } = useAuthStore();
  const { preferences, setPreferences, syncHtmlElement } = usePreferenceStore();

  const isStudent = role === "STUDENT";

  const [savingPrefs, setSavingPrefs] = useState(false);

  // Password change states
  const [showPasswordModal, setShowPasswordModal] = useState(false);
  const [currentPassword, setCurrentPassword] = useState("");
  const [newPassword, setNewPassword] = useState("");
  const [confirmPassword, setConfirmPassword] = useState("");
  const [showCurrent, setShowCurrent] = useState(false);
  const [showNew, setShowNew] = useState(false);
  const [showConfirm, setShowConfirm] = useState(false);
  const [changingPassword, setChangingPassword] = useState(false);
  const [passwordError, setPasswordError] = useState("");

  const updatePreferenceOnServer = async (updatedFields: Partial<typeof preferences>) => {
    setSavingPrefs(true);
    try {
      // Map frontend fields (snake_case) to backend request DTO
      const payload: any = {};
      if (updatedFields.theme !== undefined) payload.theme = updatedFields.theme;
      if (updatedFields.font_size !== undefined) payload.font_size = updatedFields.font_size;
      if (updatedFields.reduce_motion !== undefined) payload.reduce_motion = updatedFields.reduce_motion;
      if (updatedFields.default_student_page !== undefined) payload.default_student_page = updatedFields.default_student_page;

      const res = await javaClient.patch("/api/v1/users/me/preferences", payload);
      setPreferences(res.data);
      toast.success("Settings updated successfully");
    } catch (err: any) {
      console.error(err);
      toast.error("Failed to save settings to server");
    } finally {
      setSavingPrefs(false);
    }
  };

  const handleThemeChange = (theme: ThemePreference) => {
    setPreferences({ theme });
    updatePreferenceOnServer({ theme });
  };

  const handleFontSizeChange = (font_size: FontSizePreference) => {
    setPreferences({ font_size });
    updatePreferenceOnServer({ font_size });
  };

  const handleReduceMotionChange = (reduce_motion: boolean) => {
    setPreferences({ reduce_motion });
    updatePreferenceOnServer({ reduce_motion });
  };

  const handleDefaultPageChange = (default_student_page: DefaultPagePreference) => {
    setPreferences({ default_student_page });
    updatePreferenceOnServer({ default_student_page });
  };

  const handleResetPreferences = () => {
    const defaults = {
      theme: "SYSTEM" as ThemePreference,
      font_size: "DEFAULT" as FontSizePreference,
      reduce_motion: false,
      default_student_page: "ASSIGNMENTS" as DefaultPagePreference,
    };
    setPreferences(defaults);
    updatePreferenceOnServer(defaults);
  };

  const handleChangePasswordSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setPasswordError("");

    if (newPassword !== confirmPassword) {
      setPasswordError("Confirmation password does not match");
      return;
    }

    if (newPassword.length < 8 || newPassword.length > 72) {
      setPasswordError("Password must be between 8 and 72 characters");
      return;
    }

    // Password strength check (regex matches backend)
    const hasUpper = /[A-Z]/.test(newPassword);
    const hasLower = /[a-z]/.test(newPassword);
    const hasDigit = /[0-9]/.test(newPassword);
    if (!hasUpper || !hasLower || !hasDigit) {
      setPasswordError("Password must contain at least one uppercase letter, one lowercase letter, and one digit");
      return;
    }

    if (newPassword === currentPassword) {
      setPasswordError("New password cannot be the same as the current password");
      return;
    }

    setChangingPassword(true);
    try {
      await javaClient.patch("/api/v1/users/me/password", {
        current_password: currentPassword,
        new_password: newPassword,
      });
      
      toast.success("Password changed successfully! Logging out...");
      
      // Clear password states
      setCurrentPassword("");
      setNewPassword("");
      setConfirmPassword("");
      setShowPasswordModal(false);
      
      // Auto logout after password change
      setTimeout(() => {
        logout();
      }, 1500);

    } catch (err: any) {
      console.error(err);
      const errMsg = err.response?.data?.message || "Failed to change password. Please verify current password.";
      setPasswordError(errMsg);
    } finally {
      setChangingPassword(false);
    }
  };

  return (
    <WorkspaceLayout footerLine2="rights">
      <div className="px-6 py-8">
        <div className="mx-auto max-w-4xl space-y-6">

          {/* Header */}
          <div className="flex flex-col md:flex-row md:items-center md:justify-between gap-4 border-b border-slate-200 dark:border-zinc-800 pb-5">
            <div>
              <h1 className="text-3xl font-extrabold tracking-tight text-slate-900 dark:text-white">
                Settings
              </h1>
              <p className="text-sm text-slate-500 dark:text-slate-400 mt-1">
                Customize your display, preferences, and security settings.
              </p>
            </div>
            <Link
              href="/profile"
              className="inline-flex items-center justify-center rounded-xl bg-white dark:bg-zinc-900 border border-slate-200 dark:border-zinc-800 px-4 py-2.5 text-sm font-semibold text-slate-700 dark:text-slate-300 shadow-sm hover:bg-slate-50 dark:hover:bg-zinc-850 transition"
            >
              Go to Profile
            </Link>
          </div>

          <div className="grid grid-cols-1 md:grid-cols-3 gap-6">

            {/* Left Nav Menu */}
            <div className="md:col-span-1 space-y-2">
              <nav className="flex flex-col gap-1 rounded-2xl bg-white dark:bg-zinc-950 p-3 border border-slate-200 dark:border-zinc-800 shadow-sm">
                <a href="#appearance" className="flex items-center gap-3 rounded-xl px-4 py-3 text-sm font-bold text-slate-700 dark:text-slate-300 hover:bg-slate-50 dark:hover:bg-zinc-900 transition">
                  <Paintbrush className="h-4 w-4 text-indigo-600 dark:text-indigo-400" />
                  Appearance
                </a>
                <a href="#accessibility" className="flex items-center gap-3 rounded-xl px-4 py-3 text-sm font-bold text-slate-700 dark:text-slate-300 hover:bg-slate-50 dark:hover:bg-zinc-900 transition">
                  <Accessibility className="h-4 w-4 text-indigo-600 dark:text-indigo-400" />
                  Accessibility
                </a>
                {isStudent && (
                  <a href="#landing-page" className="flex items-center gap-3 rounded-xl px-4 py-3 text-sm font-bold text-slate-700 dark:text-slate-300 hover:bg-slate-50 dark:hover:bg-zinc-900 transition">
                    <Compass className="h-4 w-4 text-indigo-600 dark:text-indigo-400" />
                    Landing Page
                  </a>
                )}
                <a href="#account" className="flex items-center gap-3 rounded-xl px-4 py-3 text-sm font-bold text-slate-700 dark:text-slate-300 hover:bg-slate-50 dark:hover:bg-zinc-900 transition">
                  <Lock className="h-4 w-4 text-indigo-600 dark:text-indigo-400" />
                  Account & Security
                </a>
              </nav>
            </div>

            {/* Right Panels */}
            <div className="md:col-span-2 space-y-6">

              {/* Appearance Section */}
              <section id="appearance" className="scroll-mt-6 rounded-3xl border border-slate-200 dark:border-zinc-800 bg-white dark:bg-zinc-950 p-6 shadow-sm space-y-6">
                <h3 className="text-lg font-bold text-slate-900 dark:text-white flex items-center gap-2 border-b border-slate-100 dark:border-zinc-900 pb-3">
                  <Paintbrush className="h-5 w-5 text-indigo-600 dark:text-indigo-400" />
                  Theme Customization
                </h3>

                <div className="space-y-4">
                  <label className="block text-xs font-bold uppercase tracking-wider text-slate-500 dark:text-slate-400">
                    Application Theme
                  </label>
                  
                  <div className="grid grid-cols-3 gap-3">
                    {(["LIGHT", "DARK", "SYSTEM"] as ThemePreference[]).map((t) => {
                      const active = preferences.theme === t;
                      return (
                        <button
                          key={t}
                          type="button"
                          onClick={() => handleThemeChange(t)}
                          className={`rounded-xl border p-4 text-center text-sm font-bold transition flex flex-col items-center justify-center gap-2 ${
                            active
                              ? "border-indigo-600 bg-indigo-50/50 dark:border-indigo-500 dark:bg-indigo-950/20 text-indigo-600 dark:text-indigo-400 ring-2 ring-indigo-500/20"
                              : "border-slate-250 dark:border-zinc-800 bg-white dark:bg-zinc-900 text-slate-700 dark:text-slate-300 hover:border-slate-300 dark:hover:border-zinc-700"
                          }`}
                        >
                          <span className="capitalize">{t.toLowerCase()}</span>
                        </button>
                      );
                    })}
                  </div>
                </div>
              </section>

              {/* Accessibility Section */}
              <section id="accessibility" className="scroll-mt-6 rounded-3xl border border-slate-200 dark:border-zinc-800 bg-white dark:bg-zinc-950 p-6 shadow-sm space-y-6">
                <h3 className="text-lg font-bold text-slate-900 dark:text-white flex items-center gap-2 border-b border-slate-100 dark:border-zinc-900 pb-3">
                  <Accessibility className="h-5 w-5 text-indigo-600 dark:text-indigo-400" />
                  Accessibility Preferences
                </h3>

                {/* Font Size Scaling */}
                <div className="space-y-3">
                  <label className="block text-xs font-bold uppercase tracking-wider text-slate-500 dark:text-slate-400">
                    Text Font Size
                  </label>

                  <div className="grid grid-cols-3 gap-3">
                    {(["SMALL", "DEFAULT", "LARGE"] as FontSizePreference[]).map((f) => {
                      const active = preferences.font_size === f;
                      return (
                        <button
                          key={f}
                          type="button"
                          onClick={() => handleFontSizeChange(f)}
                          className={`rounded-xl border p-4 text-center text-sm font-bold transition ${
                            active
                              ? "border-indigo-600 bg-indigo-50/50 dark:border-indigo-500 dark:bg-indigo-950/20 text-indigo-600 dark:text-indigo-400 ring-2 ring-indigo-500/20"
                              : "border-slate-200 dark:border-zinc-800 bg-white dark:bg-zinc-900 text-slate-700 dark:text-slate-300 hover:border-slate-300 dark:hover:border-zinc-700"
                          }`}
                        >
                          <span className="capitalize">{f.toLowerCase()}</span>
                        </button>
                      );
                    })}
                  </div>
                </div>

                {/* Reduced Motion Toggle */}
                <div className="flex items-center justify-between p-4 rounded-2xl bg-slate-50/50 dark:bg-zinc-900/30 border border-slate-100 dark:border-zinc-900 mt-2">
                  <div>
                    <h4 className="text-sm font-bold text-slate-900 dark:text-white">Reduced Motion</h4>
                    <p className="text-xs text-slate-500 dark:text-slate-400 mt-0.5">
                      Minimize or disable animations for layouts and menus.
                    </p>
                  </div>
                  <button
                    type="button"
                    onClick={() => handleReduceMotionChange(!preferences.reduce_motion)}
                    className={`relative inline-flex h-6 w-11 shrink-0 cursor-pointer rounded-full border-2 border-transparent transition-colors duration-200 ease-in-out focus:outline-none ${
                      preferences.reduce_motion ? "bg-indigo-600" : "bg-slate-200 dark:bg-zinc-800"
                    }`}
                  >
                    <span
                      className={`pointer-events-none inline-block h-5 w-5 transform rounded-full bg-white shadow ring-0 transition duration-200 ease-in-out ${
                        preferences.reduce_motion ? "translate-x-5" : "translate-x-0"
                      }`}
                    />
                  </button>
                </div>

                {/* Reset Preferences */}
                <div className="pt-2 flex justify-end">
                  <button
                    type="button"
                    onClick={handleResetPreferences}
                    className="inline-flex items-center justify-center rounded-xl border border-slate-200 dark:border-zinc-800 bg-white dark:bg-zinc-950 text-slate-750 dark:text-slate-300 px-4 py-2.5 text-xs font-bold hover:bg-slate-50 dark:hover:bg-zinc-900 transition"
                  >
                    Reset Appearance Settings
                  </button>
                </div>
              </section>

              {/* Default Landing Page Section (Student Only) */}
              {isStudent && (
                <section id="landing-page" className="scroll-mt-6 rounded-3xl border border-slate-200 dark:border-zinc-800 bg-white dark:bg-zinc-950 p-6 shadow-sm space-y-6">
                  <h3 className="text-lg font-bold text-slate-900 dark:text-white flex items-center gap-2 border-b border-slate-100 dark:border-zinc-900 pb-3">
                    <Compass className="h-5 w-5 text-indigo-600 dark:text-indigo-400" />
                    Default Landing Page
                  </h3>

                  <div className="space-y-4">
                    <p className="text-xs text-slate-500 dark:text-slate-400 mt-0.5">
                      Choose which screen to show immediately after signing in.
                    </p>

                    <div className="grid grid-cols-2 gap-3">
                      {(["ASSIGNMENTS", "CHAT"] as DefaultPagePreference[]).map((page) => {
                        const active = preferences.default_student_page === page;
                        return (
                          <button
                            key={page}
                            type="button"
                            onClick={() => handleDefaultPageChange(page)}
                            className={`rounded-xl border p-4 text-center text-sm font-bold transition flex flex-col items-center justify-center gap-2 ${
                              active
                                ? "border-indigo-600 bg-indigo-50/50 dark:border-indigo-500 dark:bg-indigo-950/20 text-indigo-600 dark:text-indigo-400 ring-2 ring-indigo-500/20"
                                : "border-slate-200 dark:border-zinc-800 bg-white dark:bg-zinc-900 text-slate-700 dark:text-slate-300 hover:border-slate-300 dark:hover:border-zinc-700"
                            }`}
                          >
                            <span className="capitalize">{page.toLowerCase()}</span>
                          </button>
                        );
                      })}
                    </div>
                  </div>
                </section>
              )}

              {/* Account and Security Section */}
              <section id="account" className="scroll-mt-6 rounded-3xl border border-slate-200 dark:border-zinc-800 bg-white dark:bg-zinc-950 p-6 shadow-sm space-y-6">
                <h3 className="text-lg font-bold text-slate-900 dark:text-white flex items-center gap-2 border-b border-slate-100 dark:border-zinc-900 pb-3">
                  <Lock className="h-5 w-5 text-indigo-600 dark:text-indigo-400" />
                  Account & Security
                </h3>

                <div className="space-y-4">
                  {/* Account Actions */}
                  <div className="flex flex-col sm:flex-row gap-3">
                    <button
                      type="button"
                      onClick={() => setShowPasswordModal(true)}
                      className="inline-flex items-center justify-center rounded-xl bg-indigo-600 hover:bg-indigo-500 text-white font-bold px-4 py-3 shadow-md transition"
                    >
                      Change Password
                    </button>
                    <button
                      type="button"
                      onClick={() => logout()}
                      className="inline-flex items-center justify-center gap-2 rounded-xl border border-red-200 dark:border-red-900/30 bg-red-50/10 text-red-650 hover:bg-red-50 px-4 py-3 text-sm font-bold transition"
                    >
                      <LogOut className="h-4 w-4" />
                      Sign Out
                    </button>
                  </div>

                  {/* Security Guidance Banner */}
                  <div className="flex gap-3.5 p-4 rounded-2xl bg-amber-50/50 dark:bg-amber-950/15 border border-amber-200/50 dark:border-amber-900/30 mt-4">
                    <ShieldAlert className="h-5 w-5 text-amber-600 dark:text-amber-550 shrink-0 mt-0.5" />
                    <div>
                      <h4 className="text-xs font-bold text-amber-800 dark:text-amber-400 uppercase tracking-wider">
                        Security Guidance
                      </h4>
                      <ul className="list-disc pl-4 text-xs text-amber-700/90 dark:text-amber-400/80 mt-1.5 space-y-1">
                        <li>Choose a strong password combining uppercase, lowercase, numbers, and symbols.</li>
                        <li>Do not use the same password across multiple academic platforms.</li>
                        <li>Never share your account credentials or JWT tokens with others.</li>
                      </ul>
                    </div>
                  </div>
                </div>
              </section>

            </div>

          </div>

          {/* Change Password Modal */}
          {showPasswordModal && (
            <div className="fixed inset-0 z-50 flex items-center justify-center p-4">
              {/* Backdrop */}
              <div
                className="fixed inset-0 bg-slate-900/40 backdrop-blur-sm"
                onClick={() => setShowPasswordModal(false)}
              />

              {/* Modal Container */}
              <div className="relative w-full max-w-md overflow-hidden rounded-3xl border border-slate-200 dark:border-zinc-800 bg-white dark:bg-zinc-950 p-6 shadow-2xl animate-in fade-in zoom-in duration-200">
                
                <h3 className="text-xl font-bold text-slate-900 dark:text-white flex items-center gap-2 mb-4">
                  <Lock className="h-5 w-5 text-indigo-600" />
                  Change Password
                </h3>

                {passwordError && (
                  <div className="rounded-xl border border-red-200 bg-red-50/50 p-3.5 text-xs font-semibold text-red-600 mb-4">
                    {passwordError}
                  </div>
                )}

                <form onSubmit={handleChangePasswordSubmit} className="space-y-4">
                  {/* Current Password */}
                  <div className="space-y-1.5">
                    <label className="block text-xs font-bold uppercase tracking-wider text-slate-500">
                      Current Password
                    </label>
                    <div className="relative">
                      <input
                        type={showCurrent ? "text" : "password"}
                        required
                        value={currentPassword}
                        onChange={(e) => setCurrentPassword(e.target.value)}
                        className="w-full pl-4 pr-11 py-3 bg-slate-50 dark:bg-zinc-900/50 border border-slate-200 dark:border-zinc-800 rounded-2xl text-slate-900 dark:text-white placeholder-slate-400 focus:outline-none focus:ring-4 focus:ring-indigo-500/10 focus:border-indigo-500"
                        placeholder="••••••••"
                      />
                      <button
                        type="button"
                        onClick={() => setShowCurrent(!showCurrent)}
                        className="absolute right-3.5 top-1/2 -translate-y-1/2 text-slate-400 hover:text-slate-650"
                      >
                        {showCurrent ? <EyeOff className="h-4.5 w-4.5" /> : <Eye className="h-4.5 w-4.5" />}
                      </button>
                    </div>
                  </div>

                  {/* New Password */}
                  <div className="space-y-1.5">
                    <label className="block text-xs font-bold uppercase tracking-wider text-slate-500">
                      New Password
                    </label>
                    <div className="relative">
                      <input
                        type={showNew ? "text" : "password"}
                        required
                        value={newPassword}
                        onChange={(e) => setNewPassword(e.target.value)}
                        className="w-full pl-4 pr-11 py-3 bg-slate-50 dark:bg-zinc-900/50 border border-slate-200 dark:border-zinc-800 rounded-2xl text-slate-900 dark:text-white placeholder-slate-400 focus:outline-none focus:ring-4 focus:ring-indigo-500/10 focus:border-indigo-500"
                        placeholder="••••••••"
                      />
                      <button
                        type="button"
                        onClick={() => setShowNew(!showNew)}
                        className="absolute right-3.5 top-1/2 -translate-y-1/2 text-slate-400 hover:text-slate-650"
                      >
                        {showNew ? <EyeOff className="h-4.5 w-4.5" /> : <Eye className="h-4.5 w-4.5" />}
                      </button>
                    </div>
                  </div>

                  {/* Confirm New Password */}
                  <div className="space-y-1.5">
                    <label className="block text-xs font-bold uppercase tracking-wider text-slate-500">
                      Confirm New Password
                    </label>
                    <div className="relative">
                      <input
                        type={showConfirm ? "text" : "password"}
                        required
                        value={confirmPassword}
                        onChange={(e) => setConfirmPassword(e.target.value)}
                        className="w-full pl-4 pr-11 py-3 bg-slate-50 dark:bg-zinc-900/50 border border-slate-200 dark:border-zinc-800 rounded-2xl text-slate-900 dark:text-white placeholder-slate-400 focus:outline-none focus:ring-4 focus:ring-indigo-500/10 focus:border-indigo-500"
                        placeholder="••••••••"
                      />
                      <button
                        type="button"
                        onClick={() => setShowConfirm(!showConfirm)}
                        className="absolute right-3.5 top-1/2 -translate-y-1/2 text-slate-400 hover:text-slate-650"
                      >
                        {showConfirm ? <EyeOff className="h-4.5 w-4.5" /> : <Eye className="h-4.5 w-4.5" />}
                      </button>
                    </div>
                  </div>

                  {/* Actions */}
                  <div className="flex justify-end gap-3 pt-3">
                    <button
                      type="button"
                      onClick={() => setShowPasswordModal(false)}
                      className="rounded-xl border border-slate-200 px-4 py-2.5 text-sm font-bold text-slate-700 hover:bg-slate-50 transition"
                    >
                      Cancel
                    </button>
                    <button
                      type="submit"
                      disabled={changingPassword}
                      className="inline-flex items-center justify-center gap-2 rounded-xl bg-indigo-600 hover:bg-indigo-500 text-white font-bold px-4 py-2.5 shadow-sm transition disabled:opacity-50"
                    >
                      {changingPassword ? (
                        <>
                          <Loader2 className="h-4 w-4 animate-spin" />
                          Saving...
                        </>
                      ) : (
                        "Save Password"
                      )}
                    </button>
                  </div>
                </form>

              </div>
            </div>
          )}

        </div>
      </div>
    </WorkspaceLayout>
  );
}
