// TC-02-18: chat riêng 1 QPS, đo TTFT (tới sự kiện `token` đầu) = first_event (client) + (mốc khung token − mốc khung status, cùng đồng hồ máy chủ trong `id: <lượt>:<ms>-<seq>`).
import http from 'k6/http';
import { check } from 'k6';
import { Trend, Counter } from 'k6/metrics';
import { uuidv4 } from 'https://jslib.k6.io/k6-utils/1.4.0/index.js';
const BASE = __ENV.BASE || 'https://localhost:943', COURSE = __ENV.COURSE, TOKENS = (__ENV.TOKENS || '').split(',').filter(Boolean), DUR = __ENV.DUR || '60s', TAG = __ENV.TAG || 'run';
const ttft = new Trend('ttft_ms', true), first = new Trend('first_event_ms', true), n = new Counter('ttft_samples');
http.setResponseCallback(http.expectedStatuses({ min: 200, max: 399 }));
export const options = { insecureSkipTLSVerify: true, scenarios: { chat: { executor: 'constant-arrival-rate', rate: 1, timeUnit: '1s', duration: DUR, preAllocatedVUs: 10, maxVUs: 20 } }, thresholds: { ttft_samples: ['count>0'] } };
export default function () {
  const i = __ITER, token = TOKENS[i % TOKENS.length];
  const h = { headers: { Authorization: `Bearer ${token}`, 'Content-Type': 'application/json', 'Idempotency-Key': uuidv4() } };
  const s = http.post(`${BASE}/api/v1/chat/sessions`, JSON.stringify({ course_id: COURSE }), h); if (s.status !== 201) { console.log('session', s.status, (s.body || '').slice(0, 120)); return; }
  const h2 = { headers: Object.assign({}, h.headers, { 'Idempotency-Key': uuidv4() }) };
  const r = http.post(`${BASE}/api/v1/chat/sessions/${s.json().id}/messages`, JSON.stringify({ content: `Câu hỏi ${TAG} số ${i}: lịch học tuần này của lớp thế nào?` }), h2);
  if (r.status !== 200) console.log('msg', r.status, (r.body || '').slice(120, 420));
  check(r, { 'chat 200': (x) => x.status === 200 });
  const st = /id: \d+:(\d+)-\d+\nevent: status/.exec(r.body || ''), tk = /id: \d+:(\d+)-\d+\nevent: token/.exec(r.body || '');
  if (st && tk) { const d = Number(tk[1]) - Number(st[1]); first.add(r.timings.waiting); ttft.add(r.timings.waiting + (d > 0 ? d : 0)); n.add(1); }
}
