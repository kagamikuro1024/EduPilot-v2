"use client";

import { useEffect } from "react";
import { useAuthStore } from "@/store/authStore";
import { usePreferenceStore } from "@/store/preferenceStore";
import javaClient from "@/services/javaClient";

export default function PreferenceSyncProvider({ children }: { children: React.ReactNode }) {
  const token = useAuthStore((s) => s.token);
  const { preferences, setPreferences, syncHtmlElement } = usePreferenceStore();

  // 1. Listen for system color-scheme changes if theme is SYSTEM
  useEffect(() => {
    if (typeof window === "undefined") return;
    
    const mediaQuery = window.matchMedia("(prefers-color-scheme: dark)");
    const listener = () => {
      if (usePreferenceStore.getState().preferences.theme === "SYSTEM") {
        usePreferenceStore.getState().syncHtmlElement();
      }
    };
    
    mediaQuery.addEventListener("change", listener);
    return () => mediaQuery.removeEventListener("change", listener);
  }, []);

  // 2. Fetch preferences on mount/login
  useEffect(() => {
    syncHtmlElement(); // sync immediately on load using cache

    if (!token) return;

    javaClient.get("/api/v1/users/me/preferences")
      .then((res) => {
        if (res.data) {
          const theme = res.data.theme;
          const font_size = res.data.font_size;
          const reduce_motion = res.data.reduce_motion;
          const default_student_page = res.data.default_student_page;
          
          setPreferences({
            theme,
            font_size,
            reduce_motion,
            default_student_page,
          });
        }
      })
      .catch((err) => {
        console.warn("Failed to sync preferences with backend:", err);
        // Fallback silently
      });
  }, [token, setPreferences, syncHtmlElement]);

  return <>{children}</>;
}
