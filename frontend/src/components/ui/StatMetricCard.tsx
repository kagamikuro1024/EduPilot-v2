import type { ReactNode } from "react";

export interface StatMetricCardProps {
  label: string;
  value: string;
  helperText: string;
  /** Icon or small illustration for the metric */
  icon: ReactNode;
  /** Tailwind classes for the icon wrapper (background + text color) */
  iconAccentClassName: string;
}

/**
 * Reusable metric tile for dashboards (documents, analytics-style KPIs, etc.).
 */
export function StatMetricCard({ label, value, helperText, icon, iconAccentClassName }: StatMetricCardProps) {
  return (
    <article className="rounded-2xl glass-panel glass-panel-hover p-5 transition-all duration-300 hover:-translate-y-1 hover:shadow-2xl hover:shadow-indigo-500/10">
      <div className="flex items-start justify-between gap-3">
        <div>
          <p className="text-[10px] font-bold uppercase tracking-widest text-slate-500 dark:text-slate-400">{label}</p>
          <p className="mt-2 text-3xl font-extrabold tracking-tight text-slate-900 dark:text-white">{value}</p>
        </div>
        <div className={`rounded-xl p-2.5 shadow-sm ${iconAccentClassName} transition-transform group-hover:scale-110`}>
          {icon}
        </div>
      </div>
      <div className="mt-4 flex items-center gap-1.5">
        <div className="h-1.5 w-1.5 rounded-full bg-emerald-500 animate-pulse" />
        <p className="text-[11px] font-bold text-slate-400 dark:text-slate-500 tracking-tight">{helperText}</p>
      </div>
    </article>
  );
}

