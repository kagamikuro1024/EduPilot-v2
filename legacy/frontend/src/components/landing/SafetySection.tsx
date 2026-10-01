"use client";

import { motion } from "framer-motion";
import { ShieldCheck, Users, Zap, GitBranch } from "lucide-react";

export default function SafetySection() {
  const features = [
    {
      icon: <ShieldCheck className="w-6 h-6 text-indigo-600 dark:text-indigo-400" />,
      title: "Verified Knowledge",
      desc: "Every answer is directly generated from official course materials and checked for strict attribution.",
    },
    {
      icon: <GitBranch className="w-6 h-6 text-violet-600 dark:text-violet-400" />,
      title: "Smart Escalation",
      desc: "If system confidence falls below optimal levels, your query automatically routes to human staff.",
    },
    {
      icon: <Users className="w-6 h-6 text-blue-600 dark:text-blue-400" />,
      title: "TA Oversight",
      desc: "Local expert Teaching Assistants monitor quality to confirm accuracy for specific tricky edge cases.",
    },
    {
      icon: <Zap className="w-6 h-6 text-amber-600 dark:text-amber-400" />,
      title: "Continuous Learning",
      desc: "AI learning loop adapts safely to local curriculum variations ensuring future perfect answers.",
    },
  ];

  return (
    <section className="py-32 relative overflow-hidden bg-transparent">
      {/* Decorative background elements */}
      <div className="absolute top-1/2 left-1/2 -translate-x-1/2 -translate-y-1/2 w-full h-full -z-10 pointer-events-none">
        <div className="absolute top-0 right-0 w-[500px] h-[500px] bg-indigo-500/5 rounded-full blur-[100px]" />
        <div className="absolute bottom-0 left-0 w-[500px] h-[500px] bg-violet-500/5 rounded-full blur-[100px]" />
      </div>

      <div className="max-w-7xl mx-auto px-6 relative z-10">
        <div className="grid grid-cols-1 lg:grid-cols-2 gap-20 items-center">
          {/* Text Side */}
          <div>
            <motion.h2 
              initial={{ opacity: 0, x: -20 }}
              whileInView={{ opacity: 1, x: 0 }}
              viewport={{ once: true }}
              className="text-xs font-bold uppercase tracking-[0.2em] text-indigo-600 dark:text-indigo-400 mb-4"
            >
              Safe & Guarded
            </motion.h2>
            <motion.h3 
              initial={{ opacity: 0, y: 20 }}
              whileInView={{ opacity: 1, y: 0 }}
              viewport={{ once: true }}
              transition={{ delay: 0.1 }}
              className="text-4xl md:text-6xl font-bold text-slate-900 dark:text-white tracking-tight mb-8 leading-[1.1]"
            >
              Accuracy You <br /> <span className="text-gradient">Can Trust</span>
            </motion.h3>
            <motion.p 
              initial={{ opacity: 0, y: 20 }}
              whileInView={{ opacity: 1, y: 0 }}
              viewport={{ once: true }}
              transition={{ delay: 0.2 }}
              className="text-xl text-slate-600 dark:text-slate-400 leading-relaxed mb-12 max-w-xl font-medium"
            >
              We combine lightning-fast computation with expert oversight. If our intelligent engine encounters an ambiguous question, it hands the mic directly to local human experts.
            </motion.p>
            
            <motion.div 
              initial={{ opacity: 0, scale: 0.9 }}
              whileInView={{ opacity: 1, scale: 1 }}
              viewport={{ once: true }}
              transition={{ delay: 0.3 }}
              className="flex items-center gap-6 p-8 rounded-[2rem] glass-panel max-w-md border-indigo-100/50"
            >
              <div className="flex -space-x-3">
                {[1, 2, 3, 4].map((i) => (
                  <div key={i} className="w-12 h-12 rounded-2xl border-4 border-white dark:border-zinc-950 bg-slate-200 dark:bg-zinc-800 overflow-hidden shadow-lg transform hover:-translate-y-1 transition-transform">
                    <div className={`w-full h-full bg-gradient-to-br ${i % 2 === 0 ? 'from-indigo-400 to-violet-500' : 'from-blue-400 to-indigo-500'} opacity-90`} />
                  </div>
                ))}
              </div>
              <div>
                <p className="text-base font-bold text-slate-900 dark:text-white">Supported by verified TAs</p>
                <p className="text-sm text-slate-500 dark:text-slate-400 font-medium tracking-tight">Guarding accuracy 24/7</p>
              </div>
            </motion.div>
          </div>

          {/* Features Grid */}
          <div className="grid grid-cols-1 sm:grid-cols-2 gap-6">
            {features.map((feat, i) => (
              <motion.div
                key={i}
                initial={{ opacity: 0, y: 30 }}
                whileInView={{ opacity: 1, y: 0 }}
                viewport={{ once: true }}
                transition={{ delay: i * 0.1, duration: 0.6, ease: [0.16, 1, 0.3, 1] }}
                whileHover={{ y: -5 }}
                className="glass-panel glass-panel-hover p-8 rounded-[2rem] border-slate-100/50 dark:border-white/5"
              >
                <div className="w-14 h-14 rounded-2xl bg-slate-50 dark:bg-zinc-900 flex items-center justify-center mb-6 shadow-sm border border-slate-100 dark:border-white/5">
                  {feat.icon}
                </div>
                <h4 className="text-xl font-bold text-slate-900 dark:text-white mb-3 tracking-tight">
                  {feat.title}
                </h4>
                <p className="text-base text-slate-600 dark:text-slate-400 leading-relaxed font-medium">
                  {feat.desc}
                </p>
              </motion.div>
            ))}
          </div>
        </div>
      </div>
    </section>
  );
}
