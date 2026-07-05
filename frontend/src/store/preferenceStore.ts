import { create } from "zustand";
import { persist, createJSONStorage } from "zustand/middleware";

export type ThemePreference = "LIGHT" | "DARK" | "SYSTEM";
export type FontSizePreference = "SMALL" | "DEFAULT" | "LARGE";
export type DefaultPagePreference = "ASSIGNMENTS" | "CHAT";

export interface UserPreferences {
  theme: ThemePreference;
  font_size: FontSizePreference;
  reduce_motion: boolean;
  default_student_page: DefaultPagePreference;
}

interface PreferenceState {
  preferences: UserPreferences;
  setPreferences: (prefs: Partial<UserPreferences>) => void;
  syncHtmlElement: () => void;
}

const defaultPreferences: UserPreferences = {
  theme: "SYSTEM",
  font_size: "DEFAULT",
  reduce_motion: false,
  default_student_page: "ASSIGNMENTS",
};

const noopStorage = {
  getItem: () => null,
  setItem: () => {},
  removeItem: () => {},
};

export const usePreferenceStore = create<PreferenceState>()(
  persist(
    (set, get) => ({
      preferences: defaultPreferences,
      setPreferences: (newPrefs) => {
        set((state) => {
          const updated = { ...state.preferences, ...newPrefs };
          return { preferences: updated };
        });
        get().syncHtmlElement();
      },
      syncHtmlElement: () => {
        if (typeof window === "undefined") return;
        const { theme, font_size, reduce_motion } = get().preferences;
        
        // 1. Sync theme
        const isDark =
          theme === "DARK" ||
          (theme === "SYSTEM" &&
            typeof window !== "undefined" &&
            window.matchMedia &&
            window.matchMedia("(prefers-color-scheme: dark)").matches);
        if (isDark) {
          document.documentElement.classList.add("dark");
        } else {
          document.documentElement.classList.remove("dark");
        }

        // 2. Sync font size
        document.documentElement.setAttribute("data-font-size", font_size);

        // 3. Sync reduce motion
        if (reduce_motion) {
          document.documentElement.classList.add("reduce-motion");
        } else {
          document.documentElement.classList.remove("reduce-motion");
        }
      },
    }),
    {
      name: "user-preferences",
      storage: createJSONStorage(() => (typeof window !== "undefined" ? window.localStorage : noopStorage)),
    }
  )
);
