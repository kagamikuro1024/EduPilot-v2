// k6 thi thử (FEAT-practice US-P9-05 AC, SRS 5): N sinh viên cùng bắt đầu một lượt thi thử 20 câu, lưu từng câu rồi nộp. Mốc: dựng đề p95 ≤ 3 s, lưu câu p95 < 300 ms, 0 lỗi 5xx.
//   k6 run -e BASE=https://localhost -e COURSE=<uuid lớp> -e TOKENS="$(paste -sd, tokens.txt)" -e VUS=100 benchmarks/load/practice-mock.js
// TOKENS: access token của các sinh viên của lớp (seed-t1); mỗi VU dùng một token (VU i → token i mod N). Provider LLM không liên quan (thi thử không gọi LLM).
import http from 'k6/http';
import { check, group } from 'k6';
import { Trend } from 'k6/metrics';
import { uuidv4 } from 'https://jslib.k6.io/k6-utils/1.4.0/index.js';

const BASE = __ENV.BASE || 'https://localhost';
const COURSE = __ENV.COURSE || '';
const TOKENS = (__ENV.TOKENS || '').split(',').filter(Boolean);
const VUS = Number(__ENV.VUS || 100);
const P = `${BASE}/api/v1/courses/${COURSE}/practice`;

http.setResponseCallback(http.expectedStatuses({ min: 200, max: 399 }));
const startMs = new Trend('practice_start_ms', true);
const saveMs = new Trend('practice_save_ms', true);

export const options = {
  scenarios: { mock: { executor: 'per-vu-iterations', vus: VUS, iterations: 1, maxDuration: '5m' } },
  thresholds: { practice_start_ms: ['p(95)<3000'], practice_save_ms: ['p(95)<300'], http_req_failed: ['rate<0.005'] },
};

export default function () {
  const token = TOKENS[(__VU - 1) % Math.max(TOKENS.length, 1)];
  const tab = uuidv4();
  const h = (extra) => ({ headers: Object.assign({ Authorization: `Bearer ${token}`, 'Content-Type': 'application/json', 'X-Practice-Tab': tab }, extra || {}) });
  let attempt;
  group('start', () => {
    const r = http.post(`${P}/start`, JSON.stringify({ mode: 'MOCK', count: 20, duration_minutes: 30 }), h({ 'Idempotency-Key': uuidv4() }));
    startMs.add(r.timings.duration);
    check(r, { 'start 201/200': (x) => x.status === 201 || x.status === 200 });
    attempt = r.json();
  });
  if (!attempt || !attempt.items) return;
  group('save', () => {
    for (const it of attempt.items) {
      const opt = it.question.options[0];
      const body = opt ? { answer: { option_ids: [opt.id] }, version: 0 } : { answer: { text: 'x' }, version: 0 };
      const r = http.put(`${P}/attempts/${attempt.id}/answers/${it.id}`, JSON.stringify(body), h());
      saveMs.add(r.timings.duration);
      check(r, { 'save 200': (x) => x.status === 200 });
    }
  });
  group('submit', () => {
    const r = http.post(`${P}/attempts/${attempt.id}/submit`, null, h({ 'Idempotency-Key': uuidv4() }));
    check(r, { 'submit 200': (x) => x.status === 200 });
  });
}
