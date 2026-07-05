"use client";

import Link from "next/link";
import Image from "next/image";
import { GitBranch, AlertTriangle, Globe, Send, Briefcase } from "lucide-react";

export default function Footer() {
  return (
    <footer className="bg-transparent pt-32 pb-12 relative overflow-hidden">
      {/* Decorative background */}
      <div className="absolute bottom-0 left-1/2 -translate-x-1/2 w-full h-full -z-10 pointer-events-none">
        <div className="absolute bottom-0 left-1/2 -translate-x-1/2 w-[800px] h-[300px] bg-indigo-500/5 rounded-full blur-[120px]" />
      </div>

      <div className="max-w-7xl mx-auto px-6">
        <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-12 mb-20">
          {/* Logo & About */}
          <div className="lg:col-span-1">
            <div className="relative h-9 w-40 mb-6">
              <Image 
                src="/logo.svg" 
                alt="EduPilot Logo" 
                fill
                className="object-contain opacity-90 hover:opacity-100 transition-opacity" 
              />
            </div>
            <p className="text-sm text-slate-500 dark:text-slate-400 font-medium leading-relaxed max-w-xs">
              Building the future of academic assistance with intelligent, verified, and safe AI tools for students worldwide.
            </p>
          </div>

          {/* Quick Links */}
          <div>
            <h4 className="text-sm font-bold uppercase tracking-widest text-slate-900 dark:text-white mb-6">Product</h4>
            <ul className="space-y-4">
              {['Features', 'Safety', 'Team', 'Case Studies'].map((link) => (
                <li key={link}>
                  <Link href="#" className="text-sm font-medium text-slate-500 dark:text-slate-400 hover:text-indigo-600 dark:hover:text-indigo-400 transition-colors">
                    {link}
                  </Link>
                </li>
              ))}
            </ul>
          </div>

          {/* Legal */}
          <div>
            <h4 className="text-sm font-bold uppercase tracking-widest text-slate-900 dark:text-white mb-6">Legal</h4>
            <ul className="space-y-4">
              {['Privacy Policy', 'Terms of Service', 'Cookie Policy', 'Disclaimer'].map((link) => (
                <li key={link}>
                  <Link href="#" className="text-sm font-medium text-slate-500 dark:text-slate-400 hover:text-indigo-600 dark:hover:text-indigo-400 transition-colors">
                    {link}
                  </Link>
                </li>
              ))}
            </ul>
          </div>

          {/* Socials & Repo */}
          <div>
            <h4 className="text-sm font-bold uppercase tracking-widest text-slate-900 dark:text-white mb-6">Connect</h4>
            <div className="flex gap-4 mb-8">
              {[Globe, Send, Briefcase].map((Icon, i) => (
                <Link key={i} href="#" className="w-10 h-10 rounded-xl glass-panel flex items-center justify-center text-slate-500 hover:text-indigo-600 hover:scale-110 transition-all">
                  <Icon className="w-5 h-5" />
                </Link>
              ))}
            </div>
            <Link 
              href="https://github.com/kagamikuro1024/TA_Agent" 
              target="_blank"
              rel="noopener noreferrer"
              className="inline-flex items-center gap-3 px-6 py-3 rounded-2xl glass-panel glass-panel-hover text-sm font-bold text-slate-700 dark:text-slate-200 group"
            >
              <GitBranch className="w-4 h-4 text-indigo-500 group-hover:rotate-12 transition-transform" />
              <span>GitHub Repository</span>
            </Link>
          </div>
        </div>

        {/* Accuracy Disclaimer Section */}
        <div className="mb-16 p-8 rounded-[2.5rem] glass-panel border-amber-100/50 dark:border-amber-900/20 bg-amber-50/30 dark:bg-amber-950/10 flex flex-col sm:flex-row items-center sm:items-start gap-6 text-center sm:text-left">
          <div className="p-3 rounded-2xl bg-amber-100 dark:bg-amber-900/30 shrink-0">
            <AlertTriangle className="w-6 h-6 text-amber-600 dark:text-amber-400" />
          </div>
          <div>
            <p className="text-base font-bold text-amber-900 dark:text-amber-200 mb-1 tracking-tight">Accuracy Disclaimer</p>
            <p className="text-sm text-amber-700 dark:text-amber-400 font-medium leading-relaxed">
              EduPilot AI is designed for academic assistance but can make mistakes. Always verify critical information against official course materials and your instructors' guidance.
            </p>
          </div>
        </div>

        {/* Bottom Bar */}
        <div className="pt-10 border-t border-slate-200/40 dark:border-white/5 flex flex-col md:flex-row justify-between items-center gap-6 text-[11px] font-bold uppercase tracking-widest text-slate-400">
          <p>
            &copy; {new Date().getFullYear()} EduPilot. All rights reserved.
          </p>
          <div className="flex items-center gap-2">
            <span className="w-1.5 h-1.5 rounded-full bg-emerald-500 shadow-[0_0_8px_rgba(16,185,129,0.5)]" />
            <span>Vietnam Academic Assistant</span>
          </div>
        </div>
      </div>
    </footer>
  );
}
