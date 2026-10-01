"use client";

import { useEffect, useState } from "react";
import { useAuthStore } from "@/store/authStore";
import javaClient from "@/services/javaClient";

interface AuthenticatedAvatarProps {
  fullName: string | null;
  avatarAvailable: boolean;
  avatarVersion?: string | number;
  className?: string;
}

export default function AuthenticatedAvatar({
  fullName,
  avatarAvailable,
  avatarVersion,
  className = "h-9 w-9 text-sm font-bold",
}: AuthenticatedAvatarProps) {
  const token = useAuthStore((s) => s.token);
  const [avatarUrl, setAvatarUrl] = useState<string | null>(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState(false);

  useEffect(() => {
    let active = true;
    let url: string | null = null;

    if (!token || !avatarAvailable) {
      setAvatarUrl(null);
      setError(false);
      return;
    }

    setLoading(true);
    setError(false);

    javaClient
      .get("/api/v1/users/me/avatar", { responseType: "blob" })
      .then((res) => {
        if (!active) return;
        url = URL.createObjectURL(res.data);
        setAvatarUrl(url);
        setLoading(false);
      })
      .catch((err) => {
        if (!active) return;
        console.warn("Failed to load authenticated avatar:", err);
        setError(true);
        setLoading(false);
      });

    return () => {
      active = false;
      if (url) {
        URL.revokeObjectURL(url);
      }
    };
  }, [token, avatarAvailable, avatarVersion]);

  const initials = fullName
    ? fullName
        .trim()
        .split(/\s+/)
        .slice(0, 2)
        .map((w) => w[0]?.toUpperCase())
        .join("")
    : "ST";

  if (avatarUrl && !error && !loading) {
    return (
      <img
        src={avatarUrl}
        alt={fullName || "User Avatar"}
        className={`rounded-full object-cover shrink-0 ${className}`}
      />
    );
  }

  return (
    <div
      className={`flex shrink-0 items-center justify-center rounded-full bg-gradient-to-br from-indigo-500 via-violet-500 to-purple-600 text-white shadow-lg shadow-indigo-500/25 ${className}`}
    >
      {initials}
    </div>
  );
}
