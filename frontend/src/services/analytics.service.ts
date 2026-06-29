import javaClient from "./javaClient";
import type {
  MetricsWire,
  AnalyticsSummaryWire,
  AtRiskStudentWire,
  StudentActivityPointWire,
  TopicDifficultyWire,
} from "@/types/analytics";

export type AnalyticsRange = "7d" | "30d";

function toSinceIso(range: AnalyticsRange): string {
  const now = Date.now();
  const days = range === "30d" ? 30 : 7;
  return new Date(now - days * 24 * 60 * 60 * 1000).toISOString();
}

export const analyticsService = {
  async getSummary(): Promise<AnalyticsSummaryWire> {
    const response = await javaClient.get<AnalyticsSummaryWire>("/api/v1/admin/analytics/summary");
    return response.data;
  },

  async getTopics(): Promise<TopicDifficultyWire[]> {
    const response = await javaClient.get<TopicDifficultyWire[]>("/api/v1/admin/analytics/topics");
    return Array.isArray(response.data) ? response.data : [];
  },

  async getAtRiskStudents(): Promise<AtRiskStudentWire[]> {
    const response = await javaClient.get<AtRiskStudentWire[]>("/api/v1/admin/analytics/at-risk");
    return Array.isArray(response.data) ? response.data : [];
  },

  async getMetrics(range: AnalyticsRange): Promise<MetricsWire> {
    const response = await javaClient.get<MetricsWire>("/api/v1/admin/analytics/metrics", {
      params: { since: toSinceIso(range) },
    });
    return response.data;
  },

  async getStudentActivity(range: AnalyticsRange): Promise<StudentActivityPointWire[]> {
    const response = await javaClient.get<StudentActivityPointWire[]>(
      "/api/v1/admin/analytics/student-activity",
      {
        params: { since: toSinceIso(range) },
      },
    );
    return Array.isArray(response.data) ? response.data : [];
  },
};
