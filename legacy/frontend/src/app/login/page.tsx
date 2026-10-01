"use client";

import { Eye, EyeOff, ArrowLeft, Brain } from "lucide-react";
import { useState } from "react";
import { useRouter } from "next/navigation";
import Link from "next/link";
import { getPostAuthPath } from "@/lib/postAuthRedirect";
import { authService } from "@/services/auth.service";
import { useAuthStore } from "@/store/authStore";
import AuthBackground from "@/components/auth/AuthBackground";
import { extractErrorMessage } from "@/lib/utils";
import javaClient from "@/services/javaClient";

export default function LoginPage() {
  const router = useRouter();
  const setSession = useAuthStore((state) => state.setSession);

  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");
  const [showPassword, setShowPassword] = useState(false);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setLoading(true);
    setError(null);

    try {
      const response = await authService.login({ email, password });
      setSession({ token: response.token, role: response.role, fullName: response.fullName });
      
      if (response.role === "STUDENT") {
        try {
          const prefsRes = await javaClient.get("/api/v1/users/me/preferences");
          const defaultPage = prefsRes.data?.default_student_page;
          if (defaultPage === "CHAT") {
            router.push("/chat");
          } else {
            router.push("/assignments");
          }
        } catch (prefErr) {
          console.warn("Failed to fetch user preferences on login, falling back to assignments:", prefErr);
          router.push("/assignments");
        }
      } else {
        router.push(getPostAuthPath(response.role));
      }
    } catch (err: unknown) {
      setError(extractErrorMessage(err, "Invalid email or password. Please try again."));
    } finally {
      setLoading(false);
    }
  };

  return (
    <div className="min-h-screen flex items-center justify-center p-4 relative overflow-hidden">

      <AuthBackground />

      {/* Back to Home Button */}
      <Link
        href="/"
        className="absolute top-6 left-6 flex items-center gap-2 rounded-full bg-white/80 dark:bg-zinc-900/80 px-4 py-2 text-sm font-medium text-slate-700 dark:text-slate-300 shadow-lg backdrop-blur-sm transition-all hover:bg-white dark:hover:bg-zinc-900 hover:shadow-xl group"
      >
        <ArrowLeft className="h-4 w-4 transition-transform group-hover:-translate-x-1" />
        <span>Back to Home</span>
      </Link>

      {/* Glassmorphism Card */}
      <div className="w-full max-w-md glass-panel glass-panel-hover p-10 rounded-[2.5rem] relative z-10 animate-in fade-in zoom-in duration-500">
        <div className="text-center mb-10">
          <Link href="/" className="inline-flex">
            <div className="relative flex h-20 w-20 items-center justify-center rounded-2xl bg-gradient-to-br from-indigo-600 to-violet-600 shadow-xl shadow-indigo-500/30 transform transition-all hover:scale-110 hover:rotate-3 duration-300 animate-float">
              <Brain className="h-10 w-10 text-white drop-shadow-lg" strokeWidth={2.5} />
            </div>
          </Link>
          <h1 className="text-3xl font-extrabold text-slate-900 dark:text-white mb-2 tracking-tight">
            Welcome Back
          </h1>
          <p className="text-sm font-medium text-slate-500 dark:text-slate-400">
            Sign in to your <span className="bg-gradient-to-r from-indigo-600 to-violet-600 bg-clip-text font-extrabold text-transparent">AI Assistant</span>
          </p>
        </div>

        <form onSubmit={handleSubmit} className="space-y-5">
          {error && (
            <div className="rounded-2xl border border-red-200 bg-red-50/50 backdrop-blur-md p-4 text-sm font-medium text-red-600 dark:border-red-800/50 dark:bg-red-900/10 mb-4 animate-in fade-in slide-in-from-top-2">
              {error}
            </div>
          )}

          <div className="group">
            <label htmlFor="email" className="block text-xs font-bold uppercase tracking-widest text-slate-500 dark:text-slate-400 mb-2 ml-1">
              Email Address
            </label>
            <input
              id="email"
              type="email"
              required
              value={email}
              onChange={(e) => setEmail(e.target.value)}
              autoComplete="email"
              className="w-full px-5 py-4 bg-slate-100/50 dark:bg-zinc-900/50 border border-slate-200 dark:border-zinc-800 rounded-2xl text-slate-900 dark:text-white placeholder-slate-400 focus:outline-none focus:ring-4 focus:ring-indigo-500/10 focus:border-indigo-500 dark:focus:ring-indigo-500/20 transition-all duration-300"
              placeholder="name@university.edu"
            />
          </div>

          <div className="group">
            <div className="flex items-center justify-between mb-2 ml-1 mr-1">
              <label htmlFor="password" className="block text-xs font-bold uppercase tracking-widest text-slate-500 dark:text-slate-400">
                Password
              </label>
              <Link href="#" className="text-xs font-bold text-indigo-600 dark:text-indigo-400 hover:text-violet-600 transition-colors">
                Forgot password?
              </Link>
            </div>
            <div className="relative">
              <input
                id="password"
                type={showPassword ? "text" : "password"}
                required
                value={password}
                onChange={(e) => setPassword(e.target.value)}
                autoComplete="current-password"
                className="w-full px-5 py-4 pr-12 bg-slate-100/50 dark:bg-zinc-900/50 border border-slate-200 dark:border-zinc-800 rounded-2xl text-slate-900 dark:text-white placeholder-slate-400 focus:outline-none focus:ring-4 focus:ring-indigo-500/10 focus:border-indigo-500 dark:focus:ring-indigo-500/20 transition-all duration-300"
                placeholder="••••••••"
              />
              <button
                type="button"
                onClick={() => setShowPassword(!showPassword)}
                className="absolute right-4 top-1/2 -translate-y-1/2 text-slate-400 hover:text-slate-600 dark:hover:text-slate-300 transition-colors"
              >
                {showPassword ? <EyeOff className="h-5 w-5" /> : <Eye className="h-5 w-5" />}
              </button>
            </div>
          </div>

          <button
            type="submit"
            disabled={loading}
            className="w-full py-4 px-4 bg-gradient-to-r from-indigo-600 to-violet-600 hover:from-indigo-500 hover:to-violet-500 text-white font-bold rounded-2xl shadow-[0_4px_14px_0_rgb(79,70,229,0.39)] hover:shadow-[0_6px_20px_rgba(79,70,229,0.23)] active:scale-95 transform transition-all duration-300 disabled:opacity-50 disabled:cursor-not-allowed mt-4 flex justify-center items-center group overflow-hidden relative"
          >
            {loading ? (
              <svg className="h-5 w-5 animate-spin text-white" fill="none" viewBox="0 0 24 24">
                <circle className="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" strokeWidth="4"></circle>
                <path className="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"></path>
              </svg>
            ) : (
              <span className="flex items-center gap-2 relative z-10">
                Sign In
                <svg className="w-5 h-5 group-hover:translate-x-1 transition-transform" fill="none" stroke="currentColor" viewBox="0 0 24 24" xmlns="http://www.w3.org/2000/svg"><path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M14 5l7 7m0 0l-7 7m7-7H3" /></svg>
              </span>
            )}
            <div className="absolute inset-0 bg-white/20 translate-x-[-100%] group-hover:translate-x-[100%] transition-transform duration-1000 ease-in-out" />
          </button>

          <p className="text-center text-sm font-medium text-slate-500 dark:text-slate-400 mt-8 pt-6 border-t border-slate-100 dark:border-zinc-800">
            Don't have an account?{" "}
            <Link href="/register" className="font-extrabold text-gradient hover:opacity-80 transition-opacity">
              Create an account
            </Link>
          </p>
        </form>
      </div>
    </div>
  );
}
