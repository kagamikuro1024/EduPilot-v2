// k6 smoke của nền PG (SRS 8.5). Chạy qua Caddy, gateway ×2:
//   k6 run -e BASE=https://localhost -e TOKEN="$(tok STUDENT $U1)" benchmarks/load/smoke.js
// Thêm -e TEST_ROUTES=1 để chạy kịch bản ghi (chỉ có ở stack dựng bằng docker-compose.test.yml).
import http from 'k6/http';
import { check, fail } from 'k6';

const BASE = __ENV.BASE || 'https://localhost';
const TOKEN = __ENV.TOKEN || '';
const TEST_ROUTES = __ENV.TEST_ROUTES === '1';
const UNKNOWN_JOB = '00000000-0000-7000-8000-000000000009';
const AUTH = { headers: { Authorization: `Bearer ${TOKEN}` } };

// SLO của SRS 8.5 đo theo 5xx. 404 (job lạ) và 429 (giới hạn tần suất) là phản hồi đúng của gateway,
// không tính vào http_req_failed; mọi 5xx và lỗi kết nối thì tính.
http.setResponseCallback(http.expectedStatuses({ min: 200, max: 499 }));

const arrival = (exec, rate, startTime) => ({
  executor: 'constant-arrival-rate',
  exec,
  rate,
  timeUnit: '1s',
  duration: '30s',
  startTime,
  preAllocatedVUs: rate,
  maxVUs: rate * 4,
});

// `jobs404` và `write` chạy lệch nhau ≥ 60 s để mỗi kịch bản nằm gọn trong một cửa sổ phút của
// rate limit theo IP (RATE_LIMIT_IP_PER_MIN = 300; SRS 5.6). healthz/readyz được miễn rate limit.
const scenarios = {
  healthz: arrival('healthz', 20, '0s'),
  readyz: arrival('readyz', 10, '0s'),
  jobs404: arrival('jobs404', 10, '30s'),
};

const thresholds = {
  'http_req_duration{scenario:healthz}': ['p(95)<300'],
  'http_req_duration{scenario:readyz}': ['p(95)<300'],
  'http_req_duration{scenario:jobs404}': ['p(95)<300'],
  http_req_failed: ['rate<0.005'],
  checks: ['rate>0.99'],
};

if (TEST_ROUTES) {
  scenarios.write = arrival('write', 5, '120s');
  thresholds['http_req_duration{scenario:write}'] = ['p(95)<500'];
}

export const options = {
  insecureSkipTLSVerify: true,
  noConnectionReuse: false,
  scenarios,
  thresholds,
};

export function setup() {
  if (!TEST_ROUTES) return;
  const r = http.get(`${BASE}/api/v1/_test/whoami`, AUTH);
  if (r.status !== 200) {
    fail('TEST_ROUTES=1 cần stack dựng bằng docker-compose.test.yml');
  }
}

export function healthz() {
  const r = http.get(`${BASE}/api/v1/healthz`);
  check(r, { 'healthz 200 ok': (x) => x.status === 200 && x.json('status') === 'ok' });
}

export function readyz() {
  const r = http.get(`${BASE}/api/v1/readyz`);
  check(r, { 'readyz 200': (x) => x.status === 200 });
}

export function jobs404() {
  const r = http.get(`${BASE}/api/v1/jobs/${UNKNOWN_JOB}`, AUTH);
  check(r, { 'jobs lạ 404': (x) => x.status === 404 });
}

export function write() {
  const params = {
    headers: {
      Authorization: `Bearer ${TOKEN}`,
      'Content-Type': 'application/json',
      'Idempotency-Key': `k6-${__VU}-${__ITER}-${Date.now()}`,
    },
  };
  const r = http.post(`${BASE}/api/v1/_test/items`, JSON.stringify({ name: `k6-${__VU}-${__ITER}` }), params);
  check(r, { 'tạo item 2xx': (x) => x.status === 200 || x.status === 201 });
}
