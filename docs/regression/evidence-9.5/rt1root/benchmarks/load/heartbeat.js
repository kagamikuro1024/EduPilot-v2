// k6 — heartbeat thời gian học (US-P5-03 AC2; SRS FEAT-crm-360 4.6.1, SYSTEM_DESIGN 1.2 ≈ 10 req/s): 10 yêu cầu/giây trong 60 s, p95 < 20 ms, 0 lỗi 5xx.
//   k6 run benchmarks/load/heartbeat.js -e BASE=https://localhost:773 -e COURSE=<uuid lớp> -e TOKENS="tok1,tok2,..."
// TOKENS: access token của các sinh viên ACTIVE ở lớp COURSE đã bấm "Đã hiểu" (mỗi người một nhịp mỗi khung 30 s nên cần ≥ 10 người để 10 req/s đều được tính; nhịp bị bỏ vẫn là 204).
// Nhịp hợp lệ không bao giờ ghi Postgres; ngưỡng đo ở đây là phản hồi của gateway (Redis + Lua).
import http from 'k6/http';
import { check } from 'k6';

const BASE = (__ENV.BASE || 'https://localhost:773').replace(/\/$/, '');
const COURSE = __ENV.COURSE || '';
const TOKENS = (__ENV.TOKENS || '').split(',').filter(Boolean);
const AREAS = ['CHAT', 'THREADS', 'PRACTICE', 'LIBRARY', 'CALENDAR'];

// 204 là thành công; 5xx hoặc bất kỳ 4xx nào đều tính vào http_req_failed.
http.setResponseCallback(http.expectedStatuses(204));

export const options = {
  scenarios: {
    heartbeat: { executor: 'constant-arrival-rate', rate: 10, timeUnit: '1s', duration: '60s', preAllocatedVUs: 20, maxVUs: 60 },
  },
  thresholds: { http_req_duration: ['p(95)<20'], http_req_failed: ['rate<0.001'], checks: ['rate>0.999'] },
};

export function setup() {
  if (!COURSE || TOKENS.length === 0) throw new Error('thiếu -e COURSE và -e TOKENS');
}

export default function () {
  const tok = TOKENS[Math.floor(Math.random() * TOKENS.length)];
  const body = JSON.stringify({ course_id: COURSE, area: AREAS[Math.floor(Math.random() * AREAS.length)], tab_id: '00000000-0000-4000-8000-000000000001' });
  const res = http.post(`${BASE}/api/v1/activity/heartbeat`, body, { headers: { Authorization: `Bearer ${tok}`, 'Content-Type': 'application/json' } });
  check(res, { 'heartbeat 204': (r) => r.status === 204 });
}
