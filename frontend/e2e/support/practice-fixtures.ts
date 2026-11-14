import type { Page, Route } from "@playwright/test";
import { BASE_URL } from "./env";

// Luyện đề THẬT (US-P9-05…09) với gateway giả có trạng thái, theo HỢP ĐỒNG THẬT (backend-go/api/openapi.yaml `/courses/{id}/practice…`).
export const COURSE = "00000000-0000-7000-8000-00000000c001";
const CORS = { "Access-Control-Allow-Origin": BASE_URL, "Access-Control-Allow-Credentials": "true", Vary: "Origin", "Access-Control-Allow-Headers": "*", "Access-Control-Allow-Methods": "*" };
export const json = (body: unknown, status = 200) => ({ status, contentType: "application/json", headers: CORS, body: JSON.stringify(body) });
export const err = (status: number, code: string, details?: unknown) => json({ code, message: "m", trace_id: "t".repeat(32), details }, status);
const frame = (event: string, data: unknown) => `event: ${event}\ndata: ${JSON.stringify(data)}\n\n`;

type Opt = { id: string; body: string };
type Item = { id: string; position: number; type: string; stem: string; topic: string; options: Opt[]; correct: string; explanation: string; answer: { answer: Record<string, unknown>; version: number; locked: boolean } | null };
type Attempt = { id: string; mode: "TOPIC" | "MOCK"; topic: string | null; status: "IN_PROGRESS" | "SUBMITTED" | "EXPIRED"; started_at: string; deadline_at: string | null; items: Item[]; current_index: number; version: number; expired: boolean };
export type Call = { method: string; path: string; body: string; headers: Record<string, string> };
type Handler = (route: Route, next: () => Promise<void>) => Promise<void> | void;

export const TOPICS = [{ topic: "Mật mã khối", available: 8 }, { topic: "Hàm băm", available: 6 }, { topic: "Ít câu", available: 3 }];

function makeItems(n: number, topic: string): Item[] {
  return Array.from({ length: n }, (_, i) => {
    const o = (k: string): Opt => ({ id: `q${i}-${k}`, body: `Phương án ${k.toUpperCase()} của câu ${i + 1}` });
    return { id: `item-${i}`, position: i, type: "MCQ_SINGLE", stem: `Câu ${i + 1} về ${topic}?`, topic, options: [o("a"), o("b"), o("c")], correct: `q${i}-b`, explanation: "Vì B đúng.", answer: null };
  });
}

/** `h` ghi đè theo khoá "METHOD đuôi" (vd. "PUT answers", "POST submit"); `next()` chạy xử lý mặc định. `now` đặt đồng hồ máy chủ giả (mặc định: giờ thật). */
export async function practiceApi(page: Page, h: Record<string, Handler> = {}, opts: { mockCount?: number; minutes?: number } = {}) {
  const calls: Call[] = [];
  const attempts = new Map<string, Attempt>();
  let seq = 0;

  const view = (a: Attempt) => {
    const closed = a.status !== "IN_PROGRESS";
    const showFb = (it: Item) => it.answer && (a.mode === "TOPIC" || closed);
    const correct = a.items.filter((it) => it.answer && (it.answer.answer.option_ids as string[] | undefined)?.[0] === it.correct).length;
    return {
      id: a.id, mode: a.mode, topic: a.topic, status: a.status, started_at: a.started_at, deadline_at: a.deadline_at, server_now: new Date().toISOString(), item_count: a.items.length,
      current_index: a.current_index, notes: [], version: a.version,
      items: a.items.map((it) => ({
        id: it.id, position: it.position, question: { type: it.type, topic: it.topic, stem: it.stem, options: it.options },
        answer: it.answer,
        feedback: showFb(it) ? { correct: (it.answer!.answer.option_ids as string[])[0] === it.correct, key: { option_ids: [it.correct] }, explanation: it.explanation, citations: [] } : null,
      })),
      result: closed ? { correct, total: a.items.length, score: ((correct * 10) / a.items.length).toFixed(1), topics: [], weakest_topic: null, expired: a.expired } : null,
    };
  };

  const handle = async (route: Route, skip = false): Promise<void> => {
    const req = route.request();
    if (req.method() === "OPTIONS") return route.fulfill({ status: 204, headers: CORS });
    const url = new URL(req.url());
    const path = url.pathname.replace(/^.*\/practice/, "");
    const parts = path.split("/").filter(Boolean);
    const body = req.postData() ?? "";
    if (!skip) calls.push({ method: req.method(), path, body, headers: req.headers() });
    const tail = parts[0] === "attempts" ? (parts[2] === "answers" ? "answers" : parts[2] === "items" ? parts[4] : parts[2]) : parts[0];
    const custom = h[`${req.method()} ${tail ?? ""}`];
    if (custom && !skip) return custom(route, () => handle(route, true));

    const a = parts[0] === "attempts" ? attempts.get(parts[1]) : undefined;
    if (req.method() === "GET" && parts[0] === "overview") {
      const open = [...attempts.values()].filter((x) => x.status === "IN_PROGRESS").map((x) => ({ id: x.id, mode: x.mode, topic: x.topic, deadline_at: x.deadline_at, item_count: x.items.length, current_index: x.current_index }));
      return route.fulfill(json({ recommended: { topic: TOPICS[0].topic, count: 8 }, topics: TOPICS, in_progress: open }));
    }
    if (req.method() === "GET" && parts[0] === "history") return route.fulfill(json({ items: [], next_cursor: null }));
    if (req.method() === "GET" && parts[0] === "weakness") return route.fulfill(json({ topics: [] }));
    if (req.method() === "POST" && parts[0] === "start") {
      const b = JSON.parse(body) as { mode: "TOPIC" | "MOCK"; topic?: string; count?: number; duration_minutes?: number };
      const id = `att-${++seq}`;
      const n = b.mode === "MOCK" ? (opts.mockCount ?? b.count ?? 20) : Math.min(5, b.count ?? 5);
      attempts.set(id, { id, mode: b.mode, topic: b.topic ?? null, status: "IN_PROGRESS", started_at: new Date().toISOString(), deadline_at: b.mode === "MOCK" ? new Date(Date.now() + (opts.minutes ?? b.duration_minutes ?? 30) * 60_000).toISOString() : null, items: makeItems(n, b.topic ?? "Thi thử"), current_index: 0, version: 1, expired: false });
      return route.fulfill(json(view(attempts.get(id)!), 201));
    }
    if (!a) return route.fulfill(err(404, "NOT_FOUND"));
    if (req.method() === "GET" && parts.length === 2) return route.fulfill(json(view(a)));
    if (req.method() === "PATCH") {
      a.current_index = (JSON.parse(body) as { current_index: number }).current_index;
      return route.fulfill(json({ current_index: a.current_index, version: ++a.version }));
    }
    if (req.method() === "PUT" && parts[2] === "answers") {
      if (a.status !== "IN_PROGRESS") return route.fulfill(err(409, "ATTEMPT_CLOSED"));
      const it = a.items.find((x) => x.id === parts[3])!;
      if (a.mode === "TOPIC" && it.answer) return route.fulfill(err(409, "ANSWER_LOCKED"));
      const b = JSON.parse(body) as { answer: Record<string, unknown> };
      it.answer = { answer: b.answer, version: (it.answer?.version ?? 0) + 1, locked: a.mode === "TOPIC" };
      const ok = (b.answer.option_ids as string[])[0] === it.correct;
      return route.fulfill(json({ version: it.answer.version, locked: it.answer.locked, feedback: a.mode === "TOPIC" ? { correct: ok, key: { option_ids: [it.correct] }, explanation: it.explanation, citations: [] } : null }));
    }
    if (req.method() === "POST" && parts[2] === "submit") {
      a.status = "SUBMITTED";
      return route.fulfill(json(view(a)));
    }
    if (req.method() === "POST" && parts[4] === "explain") {
      return route.fulfill({ status: 200, headers: { ...CORS, "Content-Type": "text/event-stream", "Cache-Control": "no-cache" }, body: frame("token", { text: "Giải thích thêm: " }) + frame("token", { text: "B đúng vì khoá đối xứng." }) + frame("done", {}) });
    }
    return route.fulfill(err(404, "NOT_FOUND"));
  };
  await page.route(/\/api\/v1\/courses\/[^/]+\/practice/, (r) => handle(r));
  await page.route("**/api/v1/me/exam-lock", (r) => r.fulfill(json({ locked: false })));
  return { calls, attempts, closeOnServer: (id: string, expired = true) => { const a = attempts.get(id)!; a.status = expired ? "EXPIRED" : "SUBMITTED"; a.expired = expired; } };
}
