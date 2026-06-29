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
  /** Preferred: set token + role from AuthResponse after login/register. */
  setSession: (session: { token: string; role?: UserRole | null; fullName?: string | null }) => void;
  /** Clears session or sets token only (infers role from JWT when possible). */
  setToken: (token: string | null) => void;
  /** If token exists but role is missing (legacy persisted state), derive role from JWT. */
  ensureRoleFromToken: () => void;
  logout: () => void;
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
      setSession: ({ token, role, fullName }) => {
        const resolved = (role ?? parseRoleFromJwt(token)) as UserRole | null;
        set({ token, role: resolved, fullName: fullName ?? null });
        syncAuthCookie(token);
      },
      setToken: (token) => {
        if (!token) {
          set({ token: null, role: null, fullName: null });
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
        set({ token: null, role: null, fullName: null });
        Cookies.remove("auth-token");
        if (typeof window !== "undefined") {
          localStorage.removeItem("auth-storage");
          window.location.href = "/login";
        }
      },
    }),
    {
      name: "auth-storage",
      storage: createJSONStorage(() => (typeof window !== "undefined" ? window.localStorage : noopStorage)),
    }
  )
);
