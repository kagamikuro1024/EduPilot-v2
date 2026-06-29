import { useCallback, useEffect, useState } from "react";
import { extractErrorMessage } from "@/lib/utils";
import { analyticsService } from "@/services/analytics.service";
import type {
  AnalyticsSummaryWire,
  AtRiskStudentWire,
  MetricsWire,
  StudentActivityPointWire,
  TopicDifficultyWire,
} from "@/types/analytics";

export type AnalyticsRangePreset = "7d" | "30d";
export type DifficultyLevel = "high" | "medium" | "low";
export type EngagementBand = "critical" | "warning" | "at_risk";

export interface KnowledgeGapVm {
  id: string;
  topic: string;
  queryCount: number;
  difficulty: DifficultyLevel;
  gapIntensity: number;
}

export interface AtRiskStudentVm {
  id: string;
  studentId: string;
  displayName: string;
  avatarInitials: string;
  unansweredThreads: number;
  lastActiveLabel: string;
  engagement: EngagementBand;
}

export interface StudentActivityVm {
  day: string;
  totalThreads: number;
  aiResolved: number;
}

function riskLevelToBand(riskLevel: string): EngagementBand {
  const normalized = riskLevel.toUpperCase();
  if (normalized === "CRITICAL") return "critical";
  if (normalized === "WARNING") return "warning";
  return "at_risk";
}

function toInitials(name: string): string {
  const words = name.trim().split(/\s+/).filter(Boolean);
  const initials = words.slice(0, 2).map((w) => w[0]?.toUpperCase() ?? "").join("");
  return initials || "NA";
}

function scoreToDifficulty(score: number): DifficultyLevel {
  if (score > 0.7) return "high";
  if (score > 0.4) return "medium";
  return "low";
}

function topicsToKnowledgeGaps(topics: TopicDifficultyWire[]): KnowledgeGapVm[] {
  return topics.map((topic, index) => ({
    id: `${index}-${topic.topicName}`,
    topic: topic.topicName,
    queryCount: topic.queryCount,
    difficulty: scoreToDifficulty(topic.difficultyScore),
    gapIntensity: Math.min(Math.round(topic.difficultyScore * 100), 100),
  }));
}

function atRiskToRows(students: AtRiskStudentWire[]): AtRiskStudentVm[] {
  return students.map((student, index) => ({
    id: `${index}-${student.studentEmail}`,
    studentId: student.studentEmail,
    displayName: student.studentName,
    avatarInitials: toInitials(student.studentName),
    unansweredThreads: 0,
    lastActiveLabel: student.reason || "No additional details",
    engagement: riskLevelToBand(student.riskLevel),
  }));
}

function toDayLabel(isoDay: string): string {
  const parsed = new Date(isoDay);
  if (Number.isNaN(parsed.getTime())) return isoDay;
  return parsed.toLocaleDateString("en-US", { weekday: "short" });
}

function toStudentActivity(points: StudentActivityPointWire[]): StudentActivityVm[] {
  return points.map((point) => ({
    day: toDayLabel(point.day),
    totalThreads: point.totalThreads,
    aiResolved: point.aiResolved,
  }));
}

const EMPTY_SUMMARY: AnalyticsSummaryWire = {
  total_questions_this_week: 0,
  ai_resolution_rate: 0,
  top_tags: [],
  at_risk_students: [],
};

const EMPTY_METRICS: MetricsWire = {
  ta_verification_accuracy: null,
  correction_rate: null,
  rejection_rate: null,
  helpful_feedback_rate: null,
  needs_ta_count: 0,
  pending_verification_count: 0,
  verified_count: 0,
  corrected_count: 0,
  rejected_count: 0,
  sensitive_detections: 0,
  public_leak_prevention_count: 0,
  channel_switch_rate: 0,
};

function toPercentLabel(value: number | null): string {
  return value == null ? "N/A" : `${(value * 100).toFixed(1)}%`;
}

function toCountLabel(value: number | null | undefined): string {
  if (value == null) return "0";
  return String(Math.max(0, Math.round(value)));
}

export function useAnalytics(range: AnalyticsRangePreset) {
  const [summary, setSummary] = useState<AnalyticsSummaryWire>(EMPTY_SUMMARY);
  const [metrics, setMetrics] = useState<MetricsWire>(EMPTY_METRICS);
  const [studentActivity, setStudentActivity] = useState<StudentActivityVm[]>([]);
  const [knowledgeGaps, setKnowledgeGaps] = useState<KnowledgeGapVm[]>([]);
  const [atRiskStudents, setAtRiskStudents] = useState<AtRiskStudentVm[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  const load = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      const [summaryData, topicsData, atRiskData, metricsData, studentActivityData] = await Promise.all([
        analyticsService.getSummary(),
        analyticsService.getTopics(),
        analyticsService.getAtRiskStudents(),
        analyticsService.getMetrics(range),
        analyticsService.getStudentActivity(range),
      ]);

      setSummary(summaryData);
      setKnowledgeGaps(topicsToKnowledgeGaps(topicsData));
      setAtRiskStudents(atRiskToRows(atRiskData));
      setMetrics(metricsData);
      setStudentActivity(toStudentActivity(studentActivityData));
    } catch (err) {
      setSummary(EMPTY_SUMMARY);
      setMetrics(EMPTY_METRICS);
      setStudentActivity([]);
      setKnowledgeGaps([]);
      setAtRiskStudents([]);
      setError(extractErrorMessage(err, "Could not load analytics data."));
    } finally {
      setLoading(false);
    }
  }, [range]);

  useEffect(() => {
    void load();
  }, [load, range]);

  return {
    summary,
    metrics,
    studentActivity,
    taVerificationAccuracyLabel: toPercentLabel(metrics.ta_verification_accuracy),
    correctionRateLabel: toPercentLabel(metrics.correction_rate ?? null),
    rejectionRateLabel: toPercentLabel(metrics.rejection_rate ?? null),
    helpfulFeedbackRateLabel: toPercentLabel(metrics.helpful_feedback_rate),
    needsTaCountLabel: toCountLabel(metrics.needs_ta_count),
    pendingVerificationCountLabel: toCountLabel(metrics.pending_verification_count),
    verifiedCountLabel: toCountLabel(metrics.verified_count),
    correctedCountLabel: toCountLabel(metrics.corrected_count),
    rejectedCountLabel: toCountLabel(metrics.rejected_count),
    sensitiveDetectionsLabel: toCountLabel(metrics.sensitive_detections),
    publicLeakPreventionLabel: toCountLabel(metrics.public_leak_prevention_count),
    channelSwitchRateLabel: toPercentLabel(metrics.channel_switch_rate ?? null),
    knowledgeGaps,
    atRiskStudents,
    loading,
    error,
    reload: load,
  };
}
