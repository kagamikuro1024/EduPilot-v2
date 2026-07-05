"use client";

import { useEffect, useState, useRef } from "react";
import { toast } from "sonner";
import { User, Mail, Shield, Award, Calendar, Loader2, Upload, Trash2, X, RefreshCw } from "lucide-react";
import { WorkspaceLayout } from "@/components/layout/WorkspaceLayout";
import AuthenticatedAvatar from "@/components/profile/AuthenticatedAvatar";
import { useAuthStore } from "@/store/authStore";
import javaClient from "@/services/javaClient";
import Link from "next/link";

interface UserProfile {
  id: string;
  full_name: string;
  email: string;
  student_code: string | null;
  role: string;
  avatar_available: boolean;
  created_at: string;
}

export default function ProfilePage() {
  const { fullName, avatarAvailable, avatarVersion, setAvatarState, updateFullName } = useAuthStore();
  
  const [profile, setProfile] = useState<UserProfile | null>(null);
  const [loading, setLoading] = useState(true);
  const [saving, setSaving] = useState(false);
  const [nameInput, setNameInput] = useState("");
  const [nameError, setNameError] = useState("");
  
  // Avatar uploading states
  const [selectedFile, setSelectedFile] = useState<File | null>(null);
  const [previewUrl, setPreviewUrl] = useState<string | null>(null);
  const [uploadingAvatar, setUploadingAvatar] = useState(false);
  const [confirmDelete, setConfirmDelete] = useState(false);
  
  const fileInputRef = useRef<HTMLInputElement>(null);

  // Fetch user profile on mount
  const fetchProfile = async () => {
    try {
      setLoading(true);
      const res = await javaClient.get<UserProfile>("/api/v1/users/me");
      setProfile(res.data);
      setNameInput(res.data.full_name || "");
      // Sync authStore state if it differs
      if (res.data.full_name !== fullName) {
        updateFullName(res.data.full_name);
      }
      if (res.data.avatar_available !== avatarAvailable) {
        setAvatarState(res.data.avatar_available);
      }
    } catch (err) {
      console.error(err);
      toast.error("Failed to load profile information");
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    fetchProfile();
  }, []);

  // Revoke preview object URL when it changes or component unmounts
  useEffect(() => {
    return () => {
      if (previewUrl) {
        URL.revokeObjectURL(previewUrl);
      }
    };
  }, [previewUrl]);

  // Handle name update submission
  const handleUpdateProfile = async (e: React.FormEvent) => {
    e.preventDefault();
    const trimmed = nameInput.trim();
    if (!trimmed) {
      setNameError("Name cannot be empty");
      return;
    }
    setNameError("");
    setSaving(true);
    try {
      const res = await javaClient.patch<UserProfile>("/api/v1/users/me", {
        full_name: trimmed,
      });
      setProfile(res.data);
      updateFullName(res.data.full_name);
      toast.success("Profile updated successfully");
    } catch (err: any) {
      console.error(err);
      const errMsg = err.response?.data?.message || "Failed to update profile";
      toast.error(errMsg);
    } finally {
      setSaving(false);
    }
  };

  // Handle avatar file selection
  const handleFileChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    const file = e.target.files?.[0];
    if (!file) return;

    // Client-side validations
    const allowedTypes = ["image/jpeg", "image/png", "image/webp"];
    if (!allowedTypes.includes(file.type)) {
      toast.error("Invalid file format. Only JPEG, PNG, and WebP are allowed.");
      return;
    }

    if (file.size > 2 * 1024 * 1024) {
      toast.error("File is too large. Maximum size allowed is 2 MB.");
      return;
    }

    setSelectedFile(file);
    if (previewUrl) {
      URL.revokeObjectURL(previewUrl);
    }
    setPreviewUrl(URL.createObjectURL(file));
  };

  const handleCancelSelection = () => {
    setSelectedFile(null);
    if (previewUrl) {
      URL.revokeObjectURL(previewUrl);
      setPreviewUrl(null);
    }
    if (fileInputRef.current) {
      fileInputRef.current.value = "";
    }
  };

  const handleUploadAvatar = async () => {
    if (!selectedFile) return;
    setUploadingAvatar(true);
    const formData = new FormData();
    formData.append("file", selectedFile);

    try {
      await javaClient.post("/api/v1/users/me/avatar", formData, {
        headers: {
          "Content-Type": "multipart/form-data",
        },
      });
      setAvatarState(true);
      if (profile) {
        setProfile({ ...profile, avatar_available: true });
      }
      handleCancelSelection();
      toast.success("Avatar updated successfully");
    } catch (err: any) {
      console.error(err);
      const errMsg = err.response?.data?.message || "Failed to upload avatar";
      toast.error(errMsg);
    } finally {
      setUploadingAvatar(false);
    }
  };

  const handleRemoveAvatar = async () => {
    setConfirmDelete(false);
    try {
      await javaClient.delete("/api/v1/users/me/avatar");
      setAvatarState(false);
      if (profile) {
        setProfile({ ...profile, avatar_available: false });
      }
      toast.success("Avatar removed successfully");
    } catch (err: any) {
      console.error(err);
      toast.error("Failed to remove avatar");
    }
  };

  // Compute profile completion percentage
  const calculateCompletion = () => {
    if (!profile) return 0;
    let points = 0;
    if (profile.full_name && profile.full_name.trim().length > 0) points += 25;
    if (profile.email) points += 25;
    if (profile.student_code) points += 25;
    if (profile.avatar_available) points += 25;
    return points;
  };

  const completionPercent = calculateCompletion();

  const formattedDate = profile?.created_at
    ? new Date(profile.created_at).toLocaleDateString("vi-VN", {
        year: "numeric",
        month: "long",
        day: "numeric",
      })
    : "";

  return (
    <WorkspaceLayout footerLine2="rights">
      <div className="px-6 py-8">
        <div className="mx-auto max-w-4xl space-y-6">
          
          {/* Header */}
          <div className="flex flex-col md:flex-row md:items-center md:justify-between gap-4 border-b border-slate-200 dark:border-zinc-800 pb-5">
            <div>
              <h1 className="text-3xl font-extrabold tracking-tight text-slate-900 dark:text-white">
                My Profile
              </h1>
              <p className="text-sm text-slate-500 dark:text-slate-400 mt-1">
                Manage your personal details and avatar preferences.
              </p>
            </div>
            <Link
              href="/settings"
              className="inline-flex items-center justify-center rounded-xl bg-white dark:bg-zinc-900 border border-slate-200 dark:border-zinc-800 px-4 py-2.5 text-sm font-semibold text-slate-700 dark:text-slate-300 shadow-sm hover:bg-slate-50 dark:hover:bg-zinc-850 transition"
            >
              Go to Settings
            </Link>
          </div>

          {loading ? (
            <div className="flex flex-col items-center justify-center py-20 gap-3">
              <Loader2 className="h-10 w-10 animate-spin text-indigo-600" />
              <p className="text-sm font-medium text-slate-500 dark:text-slate-400">
                Loading profile details...
              </p>
            </div>
          ) : !profile ? (
            <div className="rounded-2xl border border-red-200 bg-red-50/50 p-6 text-center text-sm font-medium text-red-600">
              Failed to load profile. Please try refreshing the page.
            </div>
          ) : (
            <div className="grid grid-cols-1 lg:grid-cols-3 gap-6">
              
              {/* Left Column: Avatar & Overview */}
              <div className="lg:col-span-1 space-y-6">
                
                {/* Profile Overview Card */}
                <div className="rounded-3xl border border-slate-200 dark:border-zinc-800 bg-white dark:bg-zinc-950 p-6 shadow-sm flex flex-col items-center text-center">
                  
                  {/* Avatar section */}
                  <div className="relative group">
                    <div className="relative h-28 w-28 rounded-full overflow-hidden shadow-xl ring-4 ring-indigo-500/10 dark:ring-indigo-400/20">
                      {previewUrl ? (
                        <img
                          src={previewUrl}
                          alt="Avatar preview"
                          className="h-full w-full object-cover"
                        />
                      ) : (
                        <AuthenticatedAvatar
                          fullName={fullName}
                          avatarAvailable={avatarAvailable}
                          avatarVersion={avatarVersion}
                          className="h-full w-full text-3xl font-extrabold"
                        />
                      )}
                    </div>
                  </div>

                  <h2 className="text-xl font-bold text-slate-900 dark:text-white mt-4">
                    {profile.full_name}
                  </h2>
                  <span className="mt-1.5 rounded-full bg-indigo-50 dark:bg-indigo-950/40 px-3 py-1 text-xs font-semibold uppercase tracking-wider text-indigo-600 dark:text-indigo-400 ring-1 ring-indigo-100 dark:ring-indigo-900/50">
                    {profile.role}
                  </span>

                  {/* Completion indicator */}
                  <div className="w-full mt-6 pt-5 border-t border-slate-100 dark:border-zinc-900">
                    <div className="flex items-center justify-between text-xs font-bold text-slate-500 dark:text-slate-400 uppercase tracking-wider">
                      <span>Profile Completion</span>
                      <span className="text-indigo-600 dark:text-indigo-400 font-extrabold">{completionPercent}%</span>
                    </div>
                    <div className="mt-2 h-2 w-full overflow-hidden rounded-full bg-slate-100 dark:bg-zinc-900">
                      <div
                        className="h-full rounded-full bg-indigo-600 transition-all duration-500"
                        style={{ width: `${completionPercent}%` }}
                      />
                    </div>
                  </div>
                </div>

                {/* Avatar Action Card */}
                <div className="rounded-3xl border border-slate-200 dark:border-zinc-800 bg-white dark:bg-zinc-950 p-6 shadow-sm">
                  <h3 className="text-sm font-bold text-slate-900 dark:text-white uppercase tracking-wider mb-4">
                    Manage Avatar
                  </h3>
                  
                  <div className="space-y-3">
                    <input
                      type="file"
                      ref={fileInputRef}
                      onChange={handleFileChange}
                      accept="image/png, image/jpeg, image/webp"
                      className="hidden"
                    />

                    {previewUrl ? (
                      <div className="space-y-2">
                        <button
                          type="button"
                          onClick={handleUploadAvatar}
                          disabled={uploadingAvatar}
                          className="w-full inline-flex items-center justify-center gap-2 rounded-xl bg-indigo-600 text-white px-4 py-2.5 text-sm font-bold hover:bg-indigo-500 transition disabled:opacity-50"
                        >
                          {uploadingAvatar ? (
                            <Loader2 className="h-4 w-4 animate-spin" />
                          ) : (
                            <Upload className="h-4 w-4" />
                          )}
                          Save New Avatar
                        </button>
                        <button
                          type="button"
                          onClick={handleCancelSelection}
                          disabled={uploadingAvatar}
                          className="w-full inline-flex items-center justify-center gap-2 rounded-xl border border-slate-200 dark:border-zinc-800 bg-white dark:bg-zinc-950 text-slate-700 dark:text-slate-300 px-4 py-2.5 text-sm font-bold hover:bg-slate-50 dark:hover:bg-zinc-900 transition disabled:opacity-50"
                        >
                          <X className="h-4 w-4" />
                          Cancel
                        </button>
                      </div>
                    ) : (
                      <>
                        <button
                          type="button"
                          onClick={() => fileInputRef.current?.click()}
                          className="w-full inline-flex items-center justify-center gap-2 rounded-xl border border-indigo-200 dark:border-indigo-900/50 bg-indigo-50/50 dark:bg-indigo-950/20 text-indigo-600 dark:text-indigo-400 px-4 py-2.5 text-sm font-bold hover:bg-indigo-50 dark:hover:bg-indigo-950/40 transition"
                        >
                          <Upload className="h-4 w-4" />
                          Upload Photo
                        </button>
                        
                        {profile.avatar_available && (
                          <div className="pt-2">
                            {confirmDelete ? (
                              <div className="rounded-xl bg-red-50 dark:bg-red-950/20 border border-red-200 dark:border-red-900/30 p-3 text-center space-y-2">
                                <p className="text-xs font-semibold text-red-700 dark:text-red-400">
                                  Are you sure you want to remove your avatar?
                                </p>
                                <div className="flex gap-2 justify-center">
                                  <button
                                    type="button"
                                    onClick={handleRemoveAvatar}
                                    className="rounded-lg bg-red-600 text-white px-3 py-1 text-xs font-bold hover:bg-red-500"
                                  >
                                    Yes, Remove
                                  </button>
                                  <button
                                    type="button"
                                    onClick={() => setConfirmDelete(false)}
                                    className="rounded-lg border border-slate-200 dark:border-zinc-800 bg-white dark:bg-zinc-950 px-3 py-1 text-xs font-bold text-slate-700 dark:text-slate-300 hover:bg-slate-50"
                                  >
                                    Cancel
                                  </button>
                                </div>
                              </div>
                            ) : (
                              <button
                                type="button"
                                onClick={() => setConfirmDelete(true)}
                                className="w-full inline-flex items-center justify-center gap-2 rounded-xl border border-red-200 dark:border-red-900/30 bg-red-50/20 dark:bg-red-950/10 text-red-600 dark:text-red-400 px-4 py-2.5 text-sm font-bold hover:bg-red-50 dark:hover:bg-red-950/30 transition"
                              >
                                <Trash2 className="h-4 w-4" />
                                Remove Avatar
                              </button>
                            )}
                          </div>
                        )}
                      </>
                    )}

                    <p className="text-[11px] text-slate-400 dark:text-slate-500 text-center mt-2">
                      Supports JPEG, PNG, or WebP. Max 2 MB size.
                    </p>
                  </div>
                </div>

              </div>

              {/* Right Column: Editable Profile Form & Details */}
              <div className="lg:col-span-2 space-y-6">
                
                {/* Editable Profile Form */}
                <form
                  onSubmit={handleUpdateProfile}
                  className="rounded-3xl border border-slate-200 dark:border-zinc-800 bg-white dark:bg-zinc-950 p-6 shadow-sm space-y-6"
                >
                  <h3 className="text-lg font-bold text-slate-900 dark:text-white border-b border-slate-100 dark:border-zinc-900 pb-3">
                    Profile Details
                  </h3>

                  <div className="space-y-4">
                    {/* Full Name Field */}
                    <div className="space-y-2">
                      <label
                        htmlFor="fullNameInput"
                        className="block text-xs font-bold uppercase tracking-wider text-slate-500 dark:text-slate-400"
                      >
                        Display Name
                      </label>
                      <div className="relative rounded-2xl shadow-sm">
                        <div className="pointer-events-none absolute inset-y-0 left-0 flex items-center pl-4 text-slate-400">
                          <User className="h-4 w-4" />
                        </div>
                        <input
                          id="fullNameInput"
                          type="text"
                          required
                          value={nameInput}
                          onChange={(e) => setNameInput(e.target.value)}
                          placeholder="Your full name"
                          className="w-full pl-11 pr-5 py-3.5 bg-slate-50/50 dark:bg-zinc-900/50 border border-slate-200 dark:border-zinc-800 rounded-2xl text-slate-900 dark:text-white placeholder-slate-400 focus:outline-none focus:ring-4 focus:ring-indigo-500/10 focus:border-indigo-500 dark:focus:ring-indigo-500/20 transition-all duration-200"
                        />
                      </div>
                      {nameError && (
                        <p className="text-xs font-semibold text-red-500 mt-1 pl-1">
                          {nameError}
                        </p>
                      )}
                    </div>

                    {/* Email Field (Read Only) */}
                    <div className="space-y-2">
                      <label className="block text-xs font-bold uppercase tracking-wider text-slate-500 dark:text-slate-400">
                        Email Address (Read-only)
                      </label>
                      <div className="relative rounded-2xl shadow-sm opacity-70">
                        <div className="pointer-events-none absolute inset-y-0 left-0 flex items-center pl-4 text-slate-400">
                          <Mail className="h-4 w-4" />
                        </div>
                        <input
                          type="email"
                          readOnly
                          value={profile.email}
                          className="w-full pl-11 pr-5 py-3.5 bg-slate-100/50 dark:bg-zinc-900/30 border border-slate-200 dark:border-zinc-800 rounded-2xl text-slate-500 dark:text-slate-400 cursor-not-allowed"
                        />
                      </div>
                    </div>

                    {/* Student Code (Read Only) */}
                    {profile.student_code && (
                      <div className="space-y-2">
                        <label className="block text-xs font-bold uppercase tracking-wider text-slate-500 dark:text-slate-400">
                          Student ID (Read-only)
                        </label>
                        <div className="relative rounded-2xl shadow-sm opacity-70">
                          <div className="pointer-events-none absolute inset-y-0 left-0 flex items-center pl-4 text-slate-400">
                            <Award className="h-4 w-4" />
                          </div>
                          <input
                            type="text"
                            readOnly
                            value={profile.student_code}
                            className="w-full pl-11 pr-5 py-3.5 bg-slate-100/50 dark:bg-zinc-900/30 border border-slate-200 dark:border-zinc-800 rounded-2xl text-slate-500 dark:text-slate-400 cursor-not-allowed"
                          />
                        </div>
                      </div>
                    )}

                    {/* Account Metadata Grid */}
                    <div className="grid grid-cols-1 md:grid-cols-2 gap-4 pt-4 border-t border-slate-100 dark:border-zinc-900">
                      
                      {/* Role (Read Only) */}
                      <div className="flex items-center gap-3 p-3.5 rounded-2xl bg-slate-50/50 dark:bg-zinc-900/30 border border-slate-100 dark:border-zinc-900">
                        <div className="flex h-10 w-10 items-center justify-center rounded-xl bg-violet-50 dark:bg-violet-950/30 text-violet-600 dark:text-violet-400">
                          <Shield className="h-5 w-5" />
                        </div>
                        <div>
                          <p className="text-[10px] font-bold uppercase tracking-wider text-slate-400">Role</p>
                          <p className="text-sm font-bold text-slate-700 dark:text-slate-300">{profile.role}</p>
                        </div>
                      </div>

                      {/* Created At (Read Only) */}
                      <div className="flex items-center gap-3 p-3.5 rounded-2xl bg-slate-50/50 dark:bg-zinc-900/30 border border-slate-100 dark:border-zinc-900">
                        <div className="flex h-10 w-10 items-center justify-center rounded-xl bg-emerald-50 dark:bg-emerald-950/30 text-emerald-600 dark:text-emerald-400">
                          <Calendar className="h-5 w-5" />
                        </div>
                        <div>
                          <p className="text-[10px] font-bold uppercase tracking-wider text-slate-400">Joined On</p>
                          <p className="text-sm font-bold text-slate-700 dark:text-slate-300">{formattedDate}</p>
                        </div>
                      </div>

                    </div>
                  </div>

                  {/* Submit section */}
                  <div className="flex justify-end pt-4">
                    <button
                      type="submit"
                      disabled={saving || nameInput.trim() === profile.full_name}
                      className="inline-flex items-center justify-center gap-2 rounded-2xl bg-indigo-600 hover:bg-indigo-500 text-white font-bold px-6 py-3 shadow-md hover:shadow-lg hover:shadow-indigo-500/10 active:scale-95 transition-all disabled:opacity-50 disabled:cursor-not-allowed"
                    >
                      {saving ? (
                        <>
                          <Loader2 className="h-4 w-4 animate-spin" />
                          Saving changes...
                        </>
                      ) : (
                        "Save Profile Changes"
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
