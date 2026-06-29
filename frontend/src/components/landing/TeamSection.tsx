"use client";

import { motion } from "framer-motion";
import Image from "next/image";

const team = [
  {
    name: "Mạc Phạm Thiên Long",
    role: "AI Application",
    initials: "ML",
    avatar: "/team/long.jpg",
    desc: "Designs the student-facing experience that makes learning feel effortless.",
  },
  {
    name: "Lê Tuấn Đạt",
    role: "AI Infrastructure",
    initials: "LTĐ",
    avatar: "/team/dat.jpg",
    desc: "Builds the engine behind every fast, reliable answer.",
  },
  {
    name: "Lê Văn Quang Trung",
    role: "AI Business & Product",
    initials: "LVQT",
    avatar: "/team/trung.jpg",
    desc: "Connects student needs with product vision to deliver real impact.",
  },
];

export default function TeamSection() {
  return (
    <section className="py-32 relative overflow-hidden bg-transparent">
      {/* Background Decorative Elements */}
      <div className="absolute top-0 left-1/2 -translate-x-1/2 w-full h-full -z-10 pointer-events-none">
        <div className="absolute top-1/4 left-1/4 w-96 h-96 bg-indigo-500/5 rounded-full blur-[120px] animate-pulse" />
        <div className="absolute bottom-1/4 right-1/4 w-96 h-96 bg-violet-500/5 rounded-full blur-[120px] animate-pulse" style={{ animationDelay: '2s' }} />
      </div>

      <div className="max-w-7xl mx-auto px-6 relative z-10">
        <div className="text-center mb-20">
          <motion.h2 
            initial={{ opacity: 0, y: 10 }}
            whileInView={{ opacity: 1, y: 0 }}
            viewport={{ once: true }}
            className="text-xs font-bold uppercase tracking-[0.2em] text-indigo-600 dark:text-indigo-400 mb-4"
          >
            The Humans Behind It
          </motion.h2>
          <motion.h3 
            initial={{ opacity: 0, y: 20 }}
            whileInView={{ opacity: 1, y: 0 }}
            viewport={{ once: true }}
            transition={{ delay: 0.1 }}
            className="text-4xl md:text-6xl font-bold text-slate-900 dark:text-white tracking-tight max-w-3xl mx-auto leading-[1.1]"
          >
            A small team with a <span className="text-gradient">big belief</span>
          </motion.h3>
          <motion.p 
            initial={{ opacity: 0, y: 20 }}
            whileInView={{ opacity: 1, y: 0 }}
            viewport={{ once: true }}
            transition={{ delay: 0.2 }}
            className="mt-6 text-lg text-slate-600 dark:text-slate-400 max-w-xl mx-auto font-medium"
          >
            We believe every student deserves instant, trustworthy support.
          </motion.p>
        </div>

        <div className="grid grid-cols-1 md:grid-cols-3 gap-8">
          {team.map((member, idx) => (
            <motion.div
              key={member.name}
              initial={{ opacity: 0, y: 40 }}
              whileInView={{ opacity: 1, y: 0 }}
              viewport={{ once: true }}
              transition={{ delay: idx * 0.1, duration: 0.8, ease: [0.16, 1, 0.3, 1] as any }}
              whileHover={{ y: -8 }}
              className="glass-panel glass-panel-hover p-10 rounded-[2.5rem] text-center relative group overflow-hidden"
            >
              {/* Card Background Glow */}
              <div className="absolute -top-24 -right-24 w-48 h-48 bg-indigo-500/10 rounded-full blur-3xl opacity-0 group-hover:opacity-100 transition-opacity duration-500" />
              
              <div className="relative z-10">
                {/* Avatar Container */}
                <div className="mx-auto w-24 h-24 mb-8 relative">
                  <div className="w-full h-full rounded-3xl bg-gradient-to-br from-indigo-500 via-violet-500 to-fuchsia-500 p-[2px] shadow-2xl group-hover:rotate-6 transition-transform duration-500">
                    <div className="w-full h-full rounded-[1.4rem] bg-white dark:bg-zinc-950 overflow-hidden flex items-center justify-center">
                      {member.avatar ? (
                        <Image 
                          src={member.avatar} 
                          alt={member.name} 
                          fill 
                          className="object-cover transition-transform duration-700 group-hover:scale-110" 
                        />
                      ) : (
                        <span className="text-transparent bg-clip-text bg-gradient-to-br from-indigo-600 to-violet-600 font-bold text-2xl tracking-tight">
                          {member.initials}
                        </span>
                      )}
                    </div>
                  </div>
                  
                  {/* Decorative element behind avatar */}
                  <div className="absolute -z-10 top-1/2 left-1/2 -translate-x-1/2 -translate-y-1/2 w-32 h-32 bg-indigo-500/20 rounded-full blur-2xl scale-0 group-hover:scale-100 transition-transform duration-500" />
                </div>

                <h4 className="text-2xl font-bold text-slate-900 dark:text-white mb-2 tracking-tight">
                  {member.name}
                </h4>
                
                <div className="inline-flex px-4 py-1.5 rounded-full bg-indigo-50 dark:bg-indigo-900/30 text-[10px] font-extrabold text-indigo-600 dark:text-indigo-400 mb-6 uppercase tracking-widest border border-indigo-100/50 dark:border-indigo-800/50">
                  {member.role}
                </div>

                <p className="text-base text-slate-600 dark:text-slate-400 leading-relaxed font-medium">
                  "{member.desc}"
                </p>
              </div>
            </motion.div>
          ))}
        </div>
      </div>
    </section>
  );
}
