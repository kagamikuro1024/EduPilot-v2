"use client";

import Link from "next/link";
import { motion, useScroll, useTransform, Variants } from "framer-motion";
import { Play, ArrowRight, Sparkles, MessageCircle, BookOpen } from "lucide-react";
import { useRef } from "react";

export default function HeroSection() {
  const containerRef = useRef(null);
  const { scrollY } = useScroll();
  const y1 = useTransform(scrollY, [0, 500], [0, 200]);
  const opacity = useTransform(scrollY, [0, 300], [1, 0]);

  const containerVariants: Variants = {
    hidden: { opacity: 0 },
    visible: {
      opacity: 1,
      transition: {
        staggerChildren: 0.15,
        delayChildren: 0.2,
      },
    },
  };

  const itemVariants: Variants = {
    hidden: { y: 40, opacity: 0 },
    visible: {
      y: 0,
      opacity: 1,
      transition: {
        duration: 1,
        ease: [0.16, 1, 0.3, 1] as const,
      },
    },
  };

  return (
    <section ref={containerRef} className="relative min-h-screen w-full flex items-center justify-center pt-32 pb-20 overflow-hidden bg-transparent">
      {/* Premium Background Mesh Gradient */}
      <div className="absolute inset-0 -z-20 overflow-hidden pointer-events-none">
        <div className="absolute top-[-10%] left-[-10%] w-[60%] h-[60%] rounded-full bg-indigo-500/10 blur-[120px] animate-pulse" />
        <div className="absolute bottom-[-10%] right-[-10%] w-[50%] h-[50%] rounded-full bg-violet-500/10 blur-[120px] animate-pulse" style={{ animationDelay: '3s' }} />
        <div className="absolute top-1/2 left-1/2 -translate-x-1/2 -translate-y-1/2 w-full h-full bg-[radial-gradient(circle_at_center,rgba(99,102,241,0.03)_0,transparent_70%)]" />
      </div>

      {/* Floating Elements for Context */}
      <motion.div style={{ y: y1, opacity }} className="absolute inset-0 -z-10 pointer-events-none select-none overflow-hidden">
        <motion.div 
          animate={{ y: [0, -20, 0], rotate: [0, 5, 0] }}
          transition={{ duration: 6, repeat: Infinity, ease: "easeInOut" }}
          className="absolute top-[20%] right-[15%] glass-panel p-4 rounded-2xl hidden lg:block"
        >
          <div className="flex items-center gap-3">
            <div className="w-10 h-10 rounded-xl bg-indigo-500 flex items-center justify-center">
              <MessageCircle className="w-5 h-5 text-white" />
            </div>
            <div>
              <div className="w-24 h-2 bg-slate-200 dark:bg-slate-700 rounded-full mb-2" />
              <div className="w-16 h-2 bg-slate-100 dark:bg-slate-800 rounded-full" />
            </div>
          </div>
        </motion.div>

        <motion.div 
          animate={{ y: [0, 25, 0], rotate: [0, -8, 0] }}
          transition={{ duration: 8, repeat: Infinity, ease: "easeInOut", delay: 1 }}
          className="absolute bottom-[25%] left-[12%] glass-panel p-4 rounded-2xl hidden lg:block"
        >
          <div className="flex items-center gap-3">
            <div className="w-10 h-10 rounded-xl bg-emerald-500 flex items-center justify-center">
              <BookOpen className="w-5 h-5 text-white" />
            </div>
            <div>
              <div className="w-20 h-2 bg-slate-200 dark:bg-slate-700 rounded-full mb-2" />
              <div className="w-28 h-2 bg-slate-100 dark:bg-slate-800 rounded-full" />
            </div>
          </div>
        </motion.div>
      </motion.div>

      <motion.div
        variants={containerVariants}
        initial="hidden"
        animate="visible"
        className="relative z-10 w-full max-w-7xl mx-auto px-6 text-center"
      >
        {/* Refined Eyebrow badge */}
        <motion.div variants={itemVariants} className="mb-8">
          <span className="inline-flex items-center gap-2 px-5 py-2 rounded-full glass-panel border-indigo-100/50 text-[11px] font-extrabold uppercase tracking-[0.2em] text-indigo-600 dark:text-indigo-400">
            <Sparkles className="w-3.5 h-3.5" />
            Vietnamese scholars' choice
          </span>
        </motion.div>

        {/* Core Headline */}
        <motion.h1
          variants={itemVariants}
          className="text-6xl md:text-8xl lg:text-9xl leading-[0.95] text-slate-900 dark:text-white tracking-tight max-w-6xl mx-auto font-black"
        >
          The Intelligent <br className="hidden sm:block" /> Assistant for <br className="hidden md:block" />
          <span className="text-gradient drop-shadow-sm">
            9,000+ Students
          </span>
        </motion.h1>

        {/* Subtext */}
        <motion.p
          variants={itemVariants}
          className="mt-10 text-xl md:text-2xl text-slate-600 dark:text-slate-400 max-w-3xl mx-auto leading-relaxed font-medium"
        >
          Verified answers from your course materials in seconds. <br className="hidden md:block" />
          Built to scale with your university journey.
        </motion.p>

        {/* Vibrant Actions */}
        <motion.div
          variants={itemVariants}
          className="mt-14 flex flex-col sm:flex-row items-center justify-center gap-6"
        >
          <Link
            href="/login"
            className="w-full sm:w-auto flex items-center justify-center gap-3 px-12 py-5 rounded-full bg-indigo-600 text-white text-lg font-bold shadow-2xl shadow-indigo-500/25 hover:bg-indigo-700 hover:shadow-indigo-500/40 hover:-translate-y-1 transition-all duration-300 active:scale-95 group"
          >
            <span>Start Learning</span>
            <ArrowRight className="w-5 h-5 group-hover:translate-x-1 transition-transform" />
          </Link>

          <a
            href="https://drive.google.com/file/d/1YW6TyoQ9L1EVqjyKlHOXYj8u-GhqCDe8/view"
            target="_blank"
            rel="noopener noreferrer"
            className="w-full sm:w-auto flex items-center justify-center gap-3 px-12 py-5 rounded-full glass-panel glass-panel-hover text-slate-900 dark:text-white text-lg font-bold group"
          >
            <div className="w-8 h-8 rounded-full bg-slate-100 dark:bg-zinc-800 flex items-center justify-center group-hover:scale-110 transition-transform">
              <Play className="w-3 h-3 fill-current ml-0.5" />
            </div>
            <span>Watch Demo</span>
          </a>
        </motion.div>

        {/* Trust Indicators */}
        <motion.div 
          variants={itemVariants}
          className="mt-20 pt-10 border-t border-slate-200/40 dark:border-white/5 flex flex-wrap justify-center gap-8 md:gap-16 opacity-50 grayscale hover:grayscale-0 transition-all duration-500"
        >
          {['University of Economics', 'Polytechnic Institute', 'National University'].map((uni) => (
            <span key={uni} className="text-sm font-bold tracking-widest uppercase text-slate-400">{uni}</span>
          ))}
        </motion.div>
      </motion.div>
    </section>
  );
}
