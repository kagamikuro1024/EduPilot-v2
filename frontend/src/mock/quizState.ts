/** Lát trạng thái bài QUIZ01, tách theo sinh viên (FR-X18): chat chỉ trả lời thủ tục khi người đó đang làm. Tách khỏi `practice.ts` (ngân hàng câu hỏi ~20 KB) để màn chat / lịch không kéo cả ngân hàng. */
export const quizKey = (studentId?: string) => `practice.quiz01.${studentId ?? "khach"}`;
export type QuizState = { status: "idle" | "doing" | "submitted"; answers: Record<string, number>; score?: number };
export const QUIZ_SEED: QuizState = { status: "idle", answers: {} };
