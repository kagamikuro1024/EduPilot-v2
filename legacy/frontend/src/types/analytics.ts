export interface TopicDifficultyWire {
  topicName: string;
  difficultyScore: number;
  queryCount: number;
}

export interface AtRiskStudentWire {
  studentName: string;
  studentEmail: string;
  riskLevel: "CRITICAL" | "WARNING" | string;
  reason: string;
}

export interface AnalyticsSummaryTopTagWire {
  name: string;
  query_count: number;
  difficulty_score: number;
}

export interface AnalyticsSummaryAtRiskWire {
  student_id: string;
  risk_level: string;
  reason: string;
}

export interface AnalyticsSummaryWire {
  total_questions_this_week: number;
  ai_resolution_rate: number;
  top_tags: AnalyticsSummaryTopTagWire[];
  at_risk_students: AnalyticsSummaryAtRiskWire[];
}

export interface MetricsWire {
  ta_verification_accuracy: number | null;
  correction_rate?: number | null;
  rejection_rate?: number | null;
  helpful_feedback_rate: number | null;
  needs_ta_count?: number | null;
  pending_verification_count?: number | null;
  verified_count?: number | null;
  corrected_count?: number | null;
  rejected_count?: number | null;
  sensitive_detections?: number | null;
  public_leak_prevention_count?: number | null;
  channel_switch_rate?: number | null;
}

export interface StudentActivityPointWire {
  day: string;
  totalThreads: number;
  aiResolved: number;
}
