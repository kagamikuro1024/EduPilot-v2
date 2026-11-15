import type { Page, Route } from "@playwright/test";
import { BASE_URL } from "./env";

export const COURSE = { id: "00000000-0000-7000-8000-00000000c001", class_code: "761987", subject_code: "INT1006", name: "An ninh mạng", semester: "2026-2027-HK1", status: "ACTIVE" };
export const cors = { "Access-Control-Allow-Origin": BASE_URL, "Access-Control-Allow-Credentials": "true", Vary: "Origin" };
export const ok = (r: Route, body: unknown, status = 200) => r.fulfill({ status, contentType: "application/json", headers: cors, body: JSON.stringify(body) });
export const fail = (r: Route, status: number, code: string, message = "Lỗi") => ok(r, { code, message, trace_id: "t-1" }, status);

const topic = (rank: number, name: string, students: number, questions: number, extra: object = {}) => ({
  rank, name, why: `${students} sinh viên hỏi, ${questions} câu`, score: 1 - rank / 10,
  signals: { students, questions, low_confidence_rate: 0.4, not_helpful_rate: 0.1, practice_wrong_rate: 0 },
  recommendation: `Ôn lại ${name} bằng một ví dụ`, examples: students >= 3 ? [`${name} là gì?`, `Cho ví dụ về ${name}`, `Khác gì so với cách cũ`] : [],
  few_asked: students < 3 || undefined, named_by: "llm", chapter: "Chương 2", week: 3, keywords: ["tcp"], ...extra,
});

export const REPORT = {
  report: {
    id: "00000000-0000-7000-8000-0000000000a1", period_from: "2026-10-15", period_to: "2026-11-14", data_watermark: "2026-11-14T08:00:00Z", created_at: "2026-11-14T09:00:00Z",
    payload: {
      topics: [topic(1, "Bắt tay ba bước TCP", 12, 31), topic(2, "Truy vấn SQL nhiều bảng", 9, 20), topic(3, "Docker compose", 2, 3, { named_by: "fallback" })],
      uncovered: [{ text: "Cách cấu hình bộ định tuyến lượng tử", count: 10 }],
      counts: { questions: 54, students: 20, clusters: 3 }, period_days: 30,
    },
  },
  has_new_data: false,
  previous: { id: "00000000-0000-7000-8000-0000000000a0", created_at: "2026-10-22T09:00:00Z", new_topics: ["Docker compose"], moved: [{ name: "Truy vấn SQL nhiều bảng", from: 1, to: 2 }] },
};

export type InsightsMock = { latest?: unknown | null; calls?: { generate: number; paths: string[] } };

/** Gateway giả cho một lớp thật của giảng viên: `/me/courses`, thông báo, và ba đường báo cáo. `latest` null → 404. */
export async function insightsApi(page: Page, mock: InsightsMock = {}) {
  const calls = mock.calls ?? { generate: 0, paths: [] };
  await page.route("**/api/v1/me/courses**", (r) => ok(r, { items: [{ course: COURSE, role_in_course: "TEACHER", enrollment_status: "ACTIVE" }], next_cursor: null }));
  await page.route("**/api/v1/me/today**", (r) => ok(r, { count: 0, actions: [] }));
  await page.route("**/api/v1/notifications**", (r) => ok(r, { items: [], next_cursor: null, unread_count: 0 }));
  await page.route("**/api/v1/courses/*/insights/knowledge-gap/latest", (r) => {
    calls.paths.push("latest");
    return mock.latest == null ? fail(r, 404, "NOT_FOUND") : ok(r, mock.latest);
  });
  return calls;
}
