"use client";

import { useEffect, useState } from "react";
import Link from "next/link";
import { WorkspaceLayout } from "@/components/layout/WorkspaceLayout";
import {
  Area,
  CartesianGrid,
  ComposedChart,
  Legend,
  Line,
  ResponsiveContainer,
  Tooltip,
  XAxis,
  YAxis,
} from "recharts";
import {
  AlertTriangle,
  BookOpen,
  ChevronRight,
  Clock,
  Download,
  Filter,
  MessageSquareWarning,
  Sparkles,
  Target,
  TrendingUp,
} from "lucide-react";
import { useAnalytics } from "@/hooks/useAnalytics";
import { useAuthStore } from "@/store/authStore";

// -----------------------------------------------------------------------------
// TypeScript models — mirror future API payloads where possible
// -----------------------------------------------------------------------------

export type DateRangePreset = "7d" | "30d";

export type DifficultyLevel = "high" | "medium" | "low";

export type EngagementBand = "critical" | "warning" | "at_risk";

export type TrendVariant = "positive" | "negative";

export interface StatCardData {
  id: string;
  label: string;
  value: string;
  trendLabel: string;
  trendVariant: TrendVariant;
}

/** One day bucket for the Student Activity chart (two series). */
export interface StudentActivityDay {
  day: string;
  totalThreads: number;
  aiResolved: number;
}

export interface KnowledgeGapTopic {
  id: string;
  topic: string;
  queryCount: number;
  difficulty: DifficultyLevel;
  /** 0–100 width for the severity / volume bar */
  gapIntensity: number;
}

export interface AtRiskStudent {
  id: string;
  studentId: string;
  displayName: string;
  avatarInitials: string;
  unansweredThreads: number;
  lastActiveLabel: string;
  engagement: EngagementBand;
}

export type KeyObservationIcon = "spike" | "latency" | "accuracy";

export interface KeyObservation {
  id: string;
  title: string;
  description: string;
  icon: KeyObservationIcon;
}

export interface RecommendedAction {
  id: string;
  label: string;
}

export interface TaTipCopy {
  headline: string;
  ctaLabel: string;
}

export interface AnalyticsInsightsBundle {
  observations: KeyObservation[];
  recommendedActions: RecommendedAction[];
  taTip: TaTipCopy;
  lastAnalysisLabel: string;
}

// -----------------------------------------------------------------------------
// Small presentational pieces (co-located for a single-file deliverable)
// -----------------------------------------------------------------------------

function difficultyLabel(level: DifficultyLevel): string {
  switch (level) {
    case "high":
      return "High";
    case "medium":
      return "Medium";
    case "low":
      return "Low";
  }
}

function difficultyStyles(level: DifficultyLevel): string {
  switch (level) {
    case "high":
      return "text-rose-600 bg-rose-50 ring-rose-100";
    case "medium":
      return "text-amber-700 bg-amber-50 ring-amber-100";
    case "low":
      return "text-emerald-700 bg-emerald-50 ring-emerald-100";
  }
}

function engagementPill(band: EngagementBand): { label: string; className: string } {
  switch (band) {
    case "critical":
      return { label: "Critical", className: "bg-red-100 text-red-700 ring-red-200" };
    case "warning":
      return { label: "Warning", className: "bg-amber-100 text-amber-800 ring-amber-200" };
    case "at_risk":
      return {
        label: "At Risk",
        className: "bg-slate-100 text-slate-600 ring-slate-200",
      };
  }
}

function StatCard({ card }: { card: StatCardData }) {
  const trendColor =
    card.trendVariant === "positive" ? "text-emerald-600" : "text-rose-600";

  return (
    <div className="rounded-xl border border-slate-200 bg-white p-5 shadow-sm">
      <p className="text-sm font-medium text-slate-500">{card.label}</p>
      <p className="mt-2 text-3xl font-semibold tracking-tight text-slate-900">{card.value}</p>
      <p className={`mt-2 text-sm font-semibold ${trendColor}`}>{card.trendLabel}</p>
    </div>
  );
}

function ObservationIcon({ kind }: { kind: KeyObservationIcon }) {
  const iconClass = "h-5 w-5 text-blue-600";
  switch (kind) {
    case "spike":
      return <TrendingUp className={iconClass} aria-hidden />;
    case "latency":
      return <Clock className={iconClass} aria-hidden />;
    case "accuracy":
      return <Target className={iconClass} aria-hidden />;
  }
}

// -----------------------------------------------------------------------------
// Page
// -----------------------------------------------------------------------------

export default function AnalyticsPage() {
  const [range, setRange] = useState<DateRangePreset>("7d");
  const [chartReady, setChartReady] = useState(false);
  const role = useAuthStore((s) => s.role);
  const isStaff = role === "TA" || role === "ADMIN";
  const {
    summary,
    taVerificationAccuracyLabel,
    correctionRateLabel,
    rejectionRateLabel,
    helpfulFeedbackRateLabel,
    needsTaCountLabel,
    pendingVerificationCountLabel,
    verifiedCountLabel,
    correctedCountLabel,
    rejectedCountLabel,
    sensitiveDetectionsLabel,
    publicLeakPreventionLabel,
    channelSwitchRateLabel,
    studentActivity,
    knowledgeGaps,
    atRiskStudents,
    loading,
    error,
  } = useAnalytics(range);

  useEffect(() => {
    const id = requestAnimationFrame(() => {
      setChartReady(true);
    });
    return () => cancelAnimationFrame(id);
  }, []);

  const statCards: StatCardData[] = [
    {
      id: "total-questions",
      label: "Total Questions",
      value: summary.total_questions_this_week.toLocaleString(),
      trendLabel: "Backend live",
      trendVariant: "positive",
    },
    {
      id: "ai-accuracy",
      label: "AI Participation Rate",
      value: `${(summary.ai_resolution_rate * 100).toFixed(1)}%`,
      trendLabel: "Backend live",
      trendVariant: "positive",
    },
    {
      id: "ta-verification-accuracy",
      label: "TA Verification Accuracy",
      value: taVerificationAccuracyLabel,
      trendLabel: "Backend live",
      trendVariant: "positive",
    },
    {
      id: "helpful-feedback-rate",
      label: "Helpful Feedback Rate",
      value: helpfulFeedbackRateLabel,
      trendLabel: "Live metric",
      trendVariant: "positive",
    },
    {
      id: "needs-ta-count",
      label: "Needs TA Count",
      value: needsTaCountLabel,
      trendLabel: "Escalation feedback",
      trendVariant: "negative",
    },
    {
      id: "pending-verification-count",
      label: "Pending AI Verification",
      value: pendingVerificationCountLabel,
      trendLabel: "TA review queue",
      trendVariant: "negative",
    },
    {
      id: "corrected-rate",
      label: "Correction Rate",
      value: correctionRateLabel,
      trendLabel: `Verified ${verifiedCountLabel} | Corrected ${correctedCountLabel}`,
      trendVariant: "positive",
    },
    {
      id: "rejected-rate",
      label: "Rejection Rate",
      value: rejectionRateLabel,
      trendLabel: `Rejected ${rejectedCountLabel}`,
      trendVariant: "negative",
    },
    {
      id: "sensitive-detections",
      label: "Sensitive Detections",
      value: sensitiveDetectionsLabel,
      trendLabel: "Privacy firewall signal",
      trendVariant: "positive",
    },
    {
      id: "public-leak-prevention",
      label: "Public Leak Prevented",
      value: publicLeakPreventionLabel,
      trendLabel: "Backend enforced blocks",
      trendVariant: "positive",
    },
    {
      id: "channel-switch-rate",
      label: "Channel Switch Suggestion Rate",
      value: channelSwitchRateLabel,
      trendLabel: "Classify-intent live",
      trendVariant: "positive",
    },
  ];

  const insights: AnalyticsInsightsBundle = {
    observations: [
      {
        id: "obs-sensitive",
        title: "Sensitive Content Detections",
        description: `Detected ${sensitiveDetectionsLabel} potentially sensitive inputs in selected window.`,
        icon: "spike",
      },
      {
        id: "obs-prevented",
        title: "Public Leak Prevention",
        description: `Blocked ${publicLeakPreventionLabel} public-channel privacy risk attempts before output.`,
        icon: "accuracy",
      },
      {
        id: "obs-switch-rate",
        title: "Channel Switching Pressure",
        description: `Suggested channel switch rate is ${channelSwitchRateLabel} based on live classify-intent traffic.`,
        icon: "latency",
      },
    ],
    recommendedActions: [
      { id: "act-privacy-policy", label: "Review privacy policy prompts" },
      { id: "act-threads", label: "Audit top blocked public thread attempts" },
      { id: "act-education", label: "Notify students to use private channel for PII" },
    ],
    taTip: {
      headline: `There are ${atRiskStudents.length} at-risk students in current dataset. Prioritize critical cases first.`,
      ctaLabel: "Open At-Risk List",
    },
    lastAnalysisLabel: `LAST ANALYSIS: ${new Date().toLocaleTimeString()}`,
  };

  if (!isStaff) {
    return (
      <WorkspaceLayout footerLine2="operational">
        <div className="mx-auto max-w-3xl px-6 py-10">
          <div className="rounded-xl border border-amber-200 bg-amber-50 p-6">
            <h1 className="text-xl font-semibold text-amber-900">Access restricted</h1>
            <p className="mt-2 text-sm text-amber-800">
              Analytics dashboard chỉ dành cho tài khoản TA/ADMIN.
            </p>
          </div>
        </div>
      </WorkspaceLayout>
    );
  }

  return (
    <WorkspaceLayout
      footerLine2="operational"
      sidePanel={
        <div className="space-y-6 p-5">
          <div>
            <p className="text-xs font-semibold uppercase tracking-wide text-slate-400">AI Quick Insights</p>
            <h2 className="mt-1 text-lg font-semibold text-slate-900">Key Observations</h2>
          </div>
          <div className="space-y-3">
            {insights.observations.map((obs) => (
              <div
                key={obs.id}
                className="rounded-xl border border-slate-200 bg-slate-50/80 p-4 shadow-sm"
              >
                <div className="flex items-start gap-3">
                  <div className="rounded-lg bg-white p-2 shadow-sm ring-1 ring-slate-100">
                    <ObservationIcon kind={obs.icon} />
                  </div>
                  <div>
                    <p className="text-sm font-semibold text-slate-900">{obs.title}</p>
                    <p className="mt-1 text-sm leading-relaxed text-slate-600">{obs.description}</p>
                  </div>
                </div>
              </div>
            ))}
          </div>

          <div>
            <h3 className="text-sm font-semibold text-slate-900">Recommended Actions</h3>
            <div className="mt-3 space-y-2">
              {insights.recommendedActions.map((action) => (
                <button
                  key={action.id}
                  type="button"
                  className="flex w-full items-center justify-between rounded-lg border border-slate-200 bg-white px-3 py-2 text-left text-sm font-semibold text-slate-800 shadow-sm transition hover:border-blue-200 hover:bg-blue-50/40"
                >
                  <span className="flex items-center gap-2">
                    <Sparkles className="h-4 w-4 text-blue-500" />
                    {action.label}
                  </span>
                  <ChevronRight className="h-4 w-4 text-slate-400" />
                </button>
              ))}
            </div>
          </div>

          <div className="rounded-xl bg-blue-600 p-4 text-white shadow-md">
            <div className="flex items-start gap-2">
              <AlertTriangle className="mt-0.5 h-5 w-5 text-white" aria-hidden />
              <div>
                <p className="text-sm font-semibold leading-snug">{insights.taTip.headline}</p>
                <button
                  type="button"
                  className="mt-3 inline-flex items-center justify-center rounded-lg bg-white px-3 py-2 text-sm font-semibold text-blue-700 shadow-sm transition hover:bg-slate-50"
                >
                  {insights.taTip.ctaLabel}
                </button>
              </div>
            </div>
          </div>

          <div className="space-y-2 border-t border-slate-100 pt-4 text-xs text-slate-500">
            <p className="font-semibold tracking-wide text-slate-600">{insights.lastAnalysisLabel}</p>
            <button type="button" className="text-sm font-semibold text-blue-600 hover:text-blue-700">
              View Full History
            </button>
          </div>
        </div>
      }
    >
      <div className="mx-auto max-w-6xl space-y-6 px-6 py-6">
              {error ? (
                <div className="rounded-lg border border-red-200 bg-red-50 px-4 py-3 text-sm text-red-700">
                  {error}
                </div>
              ) : null}
              {/* Page header */}
              <div className="flex flex-col gap-4 lg:flex-row lg:items-start lg:justify-between">
                <div>
                  <h1 className="text-2xl font-semibold tracking-tight text-slate-900">
                    Analytics Dashboard
                  </h1>
                  <p className="mt-1 max-w-2xl text-sm text-slate-600">
                    Monitor system health, student engagement, and knowledge gaps.
                  </p>
                </div>
                <div className="flex flex-wrap items-center gap-2">
                  <div className="inline-flex rounded-lg border border-slate-200 bg-white p-1 shadow-sm">
                    <button
                      type="button"
                      onClick={() => setRange("7d")}
                      className={`rounded-md px-3 py-1.5 text-sm font-medium transition ${
                        range === "7d"
                          ? "bg-blue-50 text-blue-700 ring-1 ring-blue-100"
                          : "text-slate-600 hover:text-slate-900"
                      }`}
                    >
                      7 Days
                    </button>
                    <button
                      type="button"
                      onClick={() => setRange("30d")}
                      className={`rounded-md px-3 py-1.5 text-sm font-medium transition ${
                        range === "30d"
                          ? "bg-blue-50 text-blue-700 ring-1 ring-blue-100"
                          : "text-slate-600 hover:text-slate-900"
                      }`}
                    >
                      30 Days
                    </button>
                  </div>
                  <button
                    type="button"
                    className="inline-flex items-center gap-2 rounded-lg border border-slate-200 bg-white px-3 py-2 text-sm font-medium text-slate-700 shadow-sm transition hover:bg-slate-50"
                  >
                    <Filter className="h-4 w-4 text-slate-500" />
                    Filters
                  </button>
                  <button
                    type="button"
                    className="inline-flex items-center gap-2 rounded-lg bg-blue-600 px-3 py-2 text-sm font-semibold text-white shadow-sm transition hover:bg-blue-700"
                  >
                    <Download className="h-4 w-4" />
                    Export Report
                  </button>
                </div>
              </div>

              {/* KPI row */}
              <div className="grid grid-cols-1 gap-4 md:grid-cols-2 xl:grid-cols-4">
                {statCards.map((card) => (
                  <StatCard key={card.id} card={card} />
                ))}
              </div>

              {/* Middle split */}
              <div className="grid grid-cols-1 gap-4 xl:grid-cols-5">
                <section className="rounded-xl border border-slate-200 bg-white p-5 shadow-sm xl:col-span-3">
                  <div className="flex items-start justify-between gap-3">
                    <div>
                      <h2 className="text-lg font-semibold text-slate-900">Student Activity</h2>
                      <p className="mt-1 text-sm text-slate-500">
                        Daily volume of new threads vs AI resolutions
                      </p>
                    </div>
                    <Sparkles className="h-5 w-5 text-blue-500" aria-hidden />
                  </div>
                  <div className="mt-4 h-[300px] w-full min-w-0">
                    {chartReady ? (
                      studentActivity.length > 0 ? (
                        <ResponsiveContainer width="100%" height="100%" minHeight={280} minWidth={0}>
                          <ComposedChart data={studentActivity} margin={{ top: 8, right: 8, left: 0, bottom: 0 }}>
                            <defs>
                              <linearGradient id="aiResolvedFill" x1="0" y1="0" x2="0" y2="1">
                                <stop offset="0%" stopColor="#3b82f6" stopOpacity={0.35} />
                                <stop offset="100%" stopColor="#3b82f6" stopOpacity={0} />
                              </linearGradient>
                            </defs>
                            <CartesianGrid strokeDasharray="3 3" stroke="#e2e8f0" vertical={false} />
                            <XAxis dataKey="day" tickLine={false} axisLine={false} tick={{ fill: "#64748b", fontSize: 12 }} />
                            <YAxis tickLine={false} axisLine={false} tick={{ fill: "#64748b", fontSize: 12 }} width={36} />
                            <Tooltip
                              contentStyle={{
                                borderRadius: 10,
                                borderColor: "#e2e8f0",
                                boxShadow: "0 4px 12px rgba(15,23,42,0.08)",
                              }}
                            />
                            <Legend />
                            <Area
                              type="monotone"
                              dataKey="aiResolved"
                              name="AI Resolved"
                              stroke="#60a5fa"
                              strokeWidth={2}
                              fill="url(#aiResolvedFill)"
                              fillOpacity={1}
                            />
                            <Line
                              type="monotone"
                              dataKey="totalThreads"
                              name="Total Threads"
                              stroke="#1d4ed8"
                              strokeWidth={2}
                              dot={{ r: 3, strokeWidth: 0, fill: "#1d4ed8" }}
                              activeDot={{ r: 4 }}
                            />
                          </ComposedChart>
                        </ResponsiveContainer>
                      ) : (
                        <div className="flex h-full items-center justify-center rounded-lg border border-dashed border-slate-200 bg-slate-50 text-sm text-slate-500">
                          No activity data for selected range.
                        </div>
                      )
                    ) : (
                      <div className="flex h-full items-center justify-center rounded-lg border border-dashed border-slate-200 bg-slate-50 text-sm text-slate-500">
                        Loading chart…
                      </div>
                    )}
                  </div>
                </section>

                <section className="rounded-xl border border-slate-200 bg-white p-5 shadow-sm xl:col-span-2">
                  <div className="flex items-start justify-between gap-3">
                    <div>
                      <h2 className="text-lg font-semibold text-slate-900">Knowledge Gaps</h2>
                      <p className="mt-1 text-sm text-slate-500">Topics with elevated student friction</p>
                    </div>
                    <BookOpen className="h-5 w-5 text-slate-400" aria-hidden />
                  </div>
                  <div className="mt-4 space-y-4">
                    {loading ? (
                      <div className="space-y-2">
                        <div className="h-12 animate-pulse rounded-lg bg-slate-100" />
                        <div className="h-12 animate-pulse rounded-lg bg-slate-100" />
                        <div className="h-12 animate-pulse rounded-lg bg-slate-100" />
                      </div>
                    ) : (
                      knowledgeGaps.map((gap) => (
                        <div key={gap.id} className="space-y-2">
                          <div className="flex items-center justify-between gap-2">
                            <div>
                              <p className="text-sm font-semibold text-slate-900">{gap.topic}</p>
                              <p className="text-xs text-slate-500">{gap.queryCount} queries</p>
                            </div>
                            <span
                              className={`rounded-full px-2 py-0.5 text-xs font-semibold ring-1 ${difficultyStyles(
                                gap.difficulty,
                              )}`}
                            >
                              {difficultyLabel(gap.difficulty)}
                            </span>
                          </div>
                          <div className="h-2 w-full overflow-hidden rounded-full bg-slate-100">
                            <div
                              className="h-full rounded-full bg-blue-500 transition-all"
                              style={{ width: `${gap.gapIntensity}%` }}
                            />
                          </div>
                        </div>
                      ))
                    )}
                  </div>
                  <button
                    type="button"
                    className="mt-5 w-full rounded-lg border border-blue-200 bg-blue-50 px-3 py-2 text-sm font-semibold text-blue-700 transition hover:bg-blue-100"
                  >
                    Analyze Curriculum Content
                  </button>
                </section>
              </div>

              {/* At-risk table */}
              <section className="rounded-xl border border-slate-200 bg-white shadow-sm">
                <div className="flex flex-col gap-3 border-b border-slate-100 px-5 py-4 sm:flex-row sm:items-center sm:justify-between">
                  <div>
                    <h2 className="text-lg font-semibold text-slate-900">At-Risk Students</h2>
                    <p className="text-sm text-slate-500">Students who may need a nudge from teaching staff</p>
                  </div>
                  <Link
                    href="/at-risk"
                    className="inline-flex items-center gap-2 rounded-lg border border-slate-200 bg-white px-3 py-2 text-sm font-semibold text-slate-700 shadow-sm transition hover:bg-slate-50"
                  >
                    View All Students
                    <ChevronRight className="h-4 w-4" />
                  </Link>
                </div>
                <div className="overflow-x-auto">
                  <table className="min-w-full divide-y divide-slate-200 text-left text-sm">
                    <thead className="bg-slate-50 text-xs font-semibold uppercase tracking-wide text-slate-500">
                      <tr>
                        <th className="px-5 py-3">Student</th>
                        <th className="px-5 py-3">Unanswered Threads</th>
                        <th className="px-5 py-3">Last Active</th>
                        <th className="px-5 py-3">Engagement Score</th>
                        <th className="px-5 py-3">Action</th>
                      </tr>
                    </thead>
                    <tbody className="divide-y divide-slate-100 bg-white">
                      {loading ? (
                        <tr>
                          <td className="px-5 py-4" colSpan={5}>
                            <div className="h-10 animate-pulse rounded-lg bg-slate-100" />
                          </td>
                        </tr>
                      ) : atRiskStudents.map((student) => {
                        const pill = engagementPill(student.engagement);
                        return (
                          <tr key={student.id} className="hover:bg-slate-50/80">
                            <td className="px-5 py-4">
                              <div className="flex items-center gap-3">
                                <div className="flex h-10 w-10 items-center justify-center rounded-full bg-gradient-to-br from-slate-200 to-slate-300 text-xs font-semibold text-slate-700">
                                  {student.avatarInitials}
                                </div>
                                <div>
                                  <p className="font-semibold text-slate-900">{student.displayName}</p>
                                  <p className="text-xs text-slate-500">{student.studentId}</p>
                                </div>
                              </div>
                            </td>
                            <td className="px-5 py-4">
                              <div className="inline-flex items-center gap-1.5 rounded-full bg-amber-50 px-2 py-1 text-sm font-semibold text-amber-800 ring-1 ring-amber-100">
                                <MessageSquareWarning className="h-4 w-4 text-amber-600" />
                                {student.unansweredThreads}
                              </div>
                            </td>
                            <td className="px-5 py-4 text-slate-600">{student.lastActiveLabel}</td>
                            <td className="px-5 py-4">
                              <span
                                className={`inline-flex items-center rounded-full px-2.5 py-1 text-xs font-semibold ring-1 ${pill.className}`}
                              >
                                {pill.label}
                              </span>
                            </td>
                            <td className="px-5 py-4">
                              <button
                                type="button"
                                className="rounded-lg border border-dashed border-slate-200 px-3 py-1.5 text-xs font-semibold text-slate-400 transition hover:border-slate-300 hover:text-slate-600"
                                aria-label={`Actions for ${student.displayName}`}
                              >
                                …
                              </button>
                            </td>
                          </tr>
                        );
                      })}
                    </tbody>
                  </table>
                </div>
              </section>
      </div>
    </WorkspaceLayout>
  );
}
