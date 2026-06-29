"use client";

import { useState } from "react";
import { useRouter } from "next/navigation";
import Link from "next/link";
import { ArrowLeft, Brain } from "lucide-react";
import { getPostAuthPath } from "@/lib/postAuthRedirect";
import { authService } from "@/services/auth.service";
import { useAuthStore } from "@/store/authStore";
import type { UserRole } from "@/types/auth";

export default function RegisterPage() {
  const router = useRouter();
  const setSession = useAuthStore((state) => state.setSession);

  const [fullName, setFullName] = useState("");
  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");
  const [role, setRole] = useState<UserRole>("STUDENT");
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setLoading(true);
    setError(null);

    try {
      const response = await authService.register({ fullName, email, password, role });
      const resolvedRole = response.role ?? role;
      setSession({ token: response.token, role: resolvedRole, fullName: response.fullName || fullName });
      router.push(getPostAuthPath(resolvedRole));
    } catch (err: any) {
      const serverMessage = err.response?.data?.message || err.response?.data?.error;
      setError(serverMessage || "Something went wrong. Please try again.");
    } finally {
      setLoading(false);
    }
  };

  return (
    <div className="min-h-screen flex items-center justify-center p-4 relative overflow-hidden">

      {/* Decorative Animated Blobs */}
      <div className="absolute top-[-10%] left-[-10%] w-96 h-96 bg-purple-300/20 dark:bg-purple-900/10 rounded-full filter blur-3xl opacity-70 animate-blob"></div>
      <div className="absolute top-[20%] right-[-10%] w-96 h-96 bg-cyan-300/20 dark:bg-cyan-900/10 rounded-full filter blur-3xl opacity-70 animate-blob animation-delay-2000"></div>
      <div className="absolute bottom-[-20%] left-[20%] w-96 h-96 bg-indigo-300/20 dark:bg-indigo-900/10 rounded-full filter blur-3xl opacity-70 animate-blob animation-delay-4000"></div>

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
        <div className="text-center mb-8">
          <Link href="/" className="inline-flex">
            <div className="relative flex h-20 w-20 items-center justify-center rounded-2xl bg-gradient-to-br from-indigo-600 to-violet-600 shadow-xl shadow-indigo-500/30 transform transition-all hover:scale-110 hover:-rotate-3 duration-300 animate-float">
              <Brain className="h-10 w-10 text-white drop-shadow-lg" strokeWidth={2.5} />
            </div>
          </Link>
          <h1 className="text-3xl font-extrabold text-slate-900 dark:text-white mb-2 tracking-tight">
            Create Account
          </h1>
          <p className="text-sm font-medium text-slate-500 dark:text-slate-400">
            Join the <span className="bg-gradient-to-r from-indigo-600 to-violet-600 bg-clip-text font-extrabold text-transparent">AI-powered</span> learning community
          </p>
        </div>

        <form onSubmit={handleSubmit} className="space-y-4">
          {error && (
            <div className="rounded-2xl border border-red-200 bg-red-50/50 backdrop-blur-md p-4 text-sm font-medium text-red-600 dark:border-red-800/50 dark:bg-red-900/10 mb-4 animate-in fade-in slide-in-from-top-2">
              {error}
            </div>
          )}

          <div className="group">
            <label htmlFor="fullName" className="block text-xs font-bold uppercase tracking-widest text-slate-500 dark:text-slate-400 mb-2 ml-1">
              Full Name
            </label>
            <input
              id="fullName"
              type="text"
              required
              value={fullName}
              onChange={(e) => setFullName(e.target.value)}
              className="w-full px-5 py-4 bg-slate-100/50 dark:bg-zinc-900/50 border border-slate-200 dark:border-zinc-800 rounded-2xl text-slate-900 dark:text-white placeholder-slate-400 focus:outline-none focus:ring-4 focus:ring-indigo-500/10 focus:border-indigo-500 dark:focus:ring-indigo-500/20 transition-all duration-300"
              placeholder="John Doe"
            />
          </div>

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
              className="w-full px-5 py-4 bg-slate-100/50 dark:bg-zinc-900/50 border border-slate-200 dark:border-zinc-800 rounded-2xl text-slate-900 dark:text-white placeholder-slate-400 focus:outline-none focus:ring-4 focus:ring-indigo-500/10 focus:border-indigo-500 dark:focus:ring-indigo-500/20 transition-all duration-300"
              placeholder="name@university.edu"
            />
          </div>

          <div className="group">
            <label htmlFor="password" className="block text-xs font-bold uppercase tracking-widest text-slate-500 dark:text-slate-400 mb-2 ml-1">
              Password
            </label>
            <input
              id="password"
              type="password"
              required
              value={password}
              onChange={(e) => setPassword(e.target.value)}
              className="w-full px-5 py-4 bg-slate-100/50 dark:bg-zinc-900/50 border border-slate-200 dark:border-zinc-800 rounded-2xl text-slate-900 dark:text-white placeholder-slate-400 focus:outline-none focus:ring-4 focus:ring-indigo-500/10 focus:border-indigo-500 dark:focus:ring-indigo-500/20 transition-all duration-300"
              placeholder="••••••••"
            />
          </div>

          <div className="group">
            <label htmlFor="role" className="block text-xs font-bold uppercase tracking-widest text-slate-500 dark:text-slate-400 mb-2 ml-1">
              I am a...
            </label>
            <div className="relative">
              <select
                id="role"
                value={role}
                onChange={(e) => setRole(e.target.value as UserRole)}
                className="w-full px-5 py-4 bg-slate-100/50 dark:bg-zinc-900/50 border border-slate-200 dark:border-zinc-800 rounded-2xl text-slate-900 dark:text-white focus:outline-none focus:ring-4 focus:ring-indigo-500/10 focus:border-indigo-500 dark:focus:ring-indigo-500/20 transition-all duration-300 cursor-pointer appearance-none font-medium"
              >
                <option value="STUDENT">Student</option>
                <option value="TA">Teaching Assistant (TA)</option>
              </select>
              <div className="pointer-events-none absolute inset-y-0 right-0 flex items-center px-4 text-slate-400">
                <svg className="h-4 w-4 fill-current" viewBox="0 0 20 20"><path d="M5.293 7.293a1 1 0 011.414 0L10 10.586l3.293-3.293a1 1 0 111.414 1.414l-4 4a1 1 0 01-1.414 0l-4-4a1 1 0 010-1.414z" /></svg>
              </div>
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
                Register
                <svg className="w-5 h-5 group-hover:translate-x-1 transition-transform" fill="none" stroke="currentColor" viewBox="0 0 24 24" xmlns="http://www.w3.org/2000/svg"><path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M14 5l7 7m0 0l-7 7m7-7H3" /></svg>
              </span>
            )}
            <div className="absolute inset-0 bg-white/20 translate-x-[-100%] group-hover:translate-x-[100%] transition-transform duration-1000 ease-in-out" />
          </button>

          <p className="text-center text-sm font-medium text-slate-500 dark:text-slate-400 mt-8 pt-6 border-t border-slate-100 dark:border-zinc-800">
            Already have an account?{" "}
            <Link href="/login" className="font-extrabold text-gradient hover:opacity-80 transition-opacity">
              Sign in instead
            </Link>
          </p>
        </form>
      </div>
    </div>
  );

}
