import { create } from "zustand";
import { persist, createJSONStorage } from "zustand/middleware";
import Cookies from "js-cookie";
import type { UserRole } from "@/types/auth";
import { parseRoleFromJwt } from "@/lib/jwtRole";

const noopStorage = {
  getItem: () => null,
  setItem: () => {},
  removeItem: () => {},
};

interface AuthState {
  token: string | null;
  role: UserRole | null;
  fullName: string | null;
  avatarAvailable: boolean;
  avatarVersion: number;
  /** Preferred: set token + role from AuthResponse after login/register. */
  setSession: (session: { token: string; role?: UserRole | null; fullName?: string | null; avatarAvailable?: boolean }) => void;
  /** Clears session or sets token only (infers role from JWT when possible). */
  setToken: (token: string | null) => void;
  /** If token exists but role is missing (legacy persisted state), derive role from JWT. */
  ensureRoleFromToken: () => void;
  logout: () => void;
  setAvatarState: (available: boolean) => void;
  updateFullName: (fullName: string) => void;
}

function syncAuthCookie(token: string | null) {
  if (token) {
    Cookies.set("auth-token", token, { expires: 1 });
  } else {
    Cookies.remove("auth-token");
  }
}

export const useAuthStore = create<AuthState>()(
  persist(
    (set, get) => ({
      token: null,
      role: null,
      fullName: null,
      avatarAvailable: false,
      avatarVersion: 0,
      setSession: ({ token, role, fullName, avatarAvailable }) => {
        const resolved = (role ?? parseRoleFromJwt(token)) as UserRole | null;
        set({
          token,
          role: resolved,
          fullName: fullName ?? null,
          avatarAvailable: avatarAvailable ?? false,
          avatarVersion: get().avatarVersion + 1,
        });
        syncAuthCookie(token);
      },
      setToken: (token) => {
        if (!token) {
          set({ token: null, role: null, fullName: null, avatarAvailable: false, avatarVersion: 0 });
          syncAuthCookie(null);
          return;
        }
        const inferred = parseRoleFromJwt(token);
        set({ token, role: inferred });
        syncAuthCookie(token);
      },
      ensureRoleFromToken: () => {
        const { token, role } = get();
        if (!token || role != null) return;
        const inferred = parseRoleFromJwt(token);
        if (inferred) set({ role: inferred });
      },
      logout: () => {
        set({ token: null, role: null, fullName: null, avatarAvailable: false, avatarVersion: 0 });
        Cookies.remove("auth-token");
        if (typeof window !== "undefined") {
          localStorage.removeItem("auth-storage");
          window.location.href = "/login";
        }
      },
      setAvatarState: (available) => {
        set((state) => ({
          avatarAvailable: available,
          avatarVersion: state.avatarVersion + 1,
        }));
      },
      updateFullName: (fullName) => {
        set({ fullName });
      },
    }),
    {
      name: "auth-storage",
      storage: createJSONStorage(() => (typeof window !== "undefined" ? window.localStorage : noopStorage)),
    }
  )
);
