"use client";

import Link from "next/link";
import { useScroll, useMotionValueEvent } from "framer-motion";
import { useState } from "react";
import { Brain } from "lucide-react";

export default function Navbar() {
  const { scrollY } = useScroll();
  const [isScrolled, setIsScrolled] = useState(false);

  useMotionValueEvent(scrollY, "change", (latest) => {
    setIsScrolled(latest > 50);
  });

  return (
    <div className={`fixed top-0 inset-x-0 z-50 transition-all duration-300 ${
        isScrolled
          ? "h-20 py-3"
          : "h-24 py-5"
      }`}>
      <div className="max-w-7xl mx-auto px-6 h-full">
        <div className={`h-full px-6 flex items-center justify-between rounded-full transition-all duration-300 ${
          isScrolled
            ? "glass-panel bg-white/80 dark:bg-zinc-950/80 shadow-lg border-white/50"
            : "bg-transparent border-transparent"
        }`}>
          {/* Logo Section */}
          <Link href="/" className="flex items-center gap-3 group">
            <div className="relative flex h-12 w-12 shrink-0 items-center justify-center rounded-2xl bg-gradient-to-br from-indigo-600 to-violet-600 shadow-xl shadow-indigo-500/30 transition-all duration-300 group-hover:scale-105 group-hover:shadow-2xl group-hover:shadow-indigo-500/40">
              <Brain className="h-6 w-6 text-white drop-shadow-lg" strokeWidth={2.5} />
            </div>
            <div className="flex flex-col">
              <span className="bg-gradient-to-r from-indigo-600 to-violet-600 bg-clip-text text-2xl font-black text-transparent tracking-tight">
                EduPilot
              </span>
              <span className="text-[10px] font-medium uppercase tracking-widest text-slate-500 dark:text-slate-400">
                AI Teaching Assistant
              </span>
            </div>
          </Link>

          {/* Right Action Section */}
          <div className="flex items-center gap-6">
            <Link
              href="/login"
              className="text-sm font-bold text-slate-600 dark:text-slate-400 hover:text-indigo-600 dark:hover:text-indigo-400 transition-colors hidden sm:block"
            >
              Sign In
            </Link>
            <Link
              href="/login"
              className="bg-indigo-600 text-white text-[13px] font-bold px-8 py-3 rounded-full shadow-lg shadow-indigo-500/20 hover:shadow-xl hover:bg-indigo-700 hover:-translate-y-0.5 transition-all active:scale-95"
            >
              Get Started
            </Link>
          </div>
        </div>
      </div>
    </div>
  );
}
