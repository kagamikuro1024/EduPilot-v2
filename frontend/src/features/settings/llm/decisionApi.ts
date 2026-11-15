"use client";

import { useQuery } from "@tanstack/react-query";
import { apiClient } from "@/shared/data";

// Kiểu và truy vấn của khối «Quyết định nhanh» — khớp `backend-go/api/openapi.yaml` (SRS FEAT-decision-model 6).
export type GateId = "THREADS_PRECHECK" | "INJECTION_FLAG" | "CHAT_ROUTING";
export type DecisionProvider = { id: string; model: string; status: "fake" | "connected" | "no_key" | "paused" | "not_configured"; fake: boolean };
export type DecisionGate = { gate: GateId; enabled: boolean; threshold: string; timeout_ms: number; version: number; courses_enabled: number; courses_total: number };
export type DecisionOverview = { provider: DecisionProvider | null; gates: DecisionGate[]; real_allowed: boolean };
export type DecisionCourse = { course_id: string; name: string; class_code: string; enabled: boolean; version: number };

export const DECISION_KEY = ["llm", "decision"] as const;
export const useDecision = () => useQuery({ queryKey: DECISION_KEY, queryFn: async ({ signal }) => (await apiClient.get<DecisionOverview>("/admin/llm/decision", { signal })).data });

export const GATES: Array<{ gate: GateId; label: string; range: string; unit: string }> = [
  { gate: "THREADS_PRECHECK", label: "Kiểm duyệt Threads", range: "0,01–0,20", unit: "ε" },
  { gate: "INJECTION_FLAG", label: "Cờ chèn lệnh khi chấm", range: "0,30–0,90", unit: "τ" },
  { gate: "CHAT_ROUTING", label: "Định tuyến câu hỏi chat", range: "0,70–0,99", unit: "τ" },
];

export const MODELS = ["decision-fake", "gemini-decision-1"] as const;

export const PROVIDER_STATUS: Record<DecisionProvider["status"] | "none", { text: string; tone: "green" | "amber" | "neutral" | "red" }> = {
  fake: { text: "Giả", tone: "neutral" },
  connected: { text: "Đã kết nối", tone: "green" },
  no_key: { text: "Chưa có khoá", tone: "amber" },
  paused: { text: "Tạm dừng", tone: "amber" },
  not_configured: { text: "Chưa cấu hình", tone: "neutral" },
  none: { text: "Chưa cấu hình", tone: "neutral" },
};
