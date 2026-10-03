// k6 nhẹ cho "Hôm nay" (FEAT-course-foundation US-P2-11 AC10): 20 yêu cầu/giây, p95 < 300 ms.
//   k6 run -e BASE=https://localhost -e TOKEN="$(tok TEACHER $U)" benchmarks/load/today.js
// TOKEN là access token của một giảng viên có 2 lớp (seed của US-P2-12). Hai nửa: /me/today (tất cả lớp) và /courses/{id}/today.
import http from 'k6/http';
import { check } from 'k6';

const BASE = __ENV.BASE || 'https://localhost';
const TOKEN = __ENV.TOKEN || '';
const COURSE = __ENV.COURSE || '';
const AUTH = { headers: { Authorization: `Bearer ${TOKEN}` } };

http.setResponseCallback(http.expectedStatuses({ min: 200, max: 399 }));

export const options = {
  scenarios: {
    today: { executor: 'constant-arrival-rate', rate: 20, timeUnit: '1s', duration: '30s', preAllocatedVUs: 20, maxVUs: 80 },
  },
  thresholds: { 'http_req_duration': ['p(95)<300'], http_req_failed: ['rate<0.005'] },
};

export default function () {
  const res = http.get(`${BASE}/api/v1/me/today`, AUTH);
  check(res, { 'me/today 200': (r) => r.status === 200 });
  if (COURSE) {
    const one = http.get(`${BASE}/api/v1/courses/${COURSE}/today`, AUTH);
    check(one, { 'course today 200': (r) => r.status === 200 });
  }
}
