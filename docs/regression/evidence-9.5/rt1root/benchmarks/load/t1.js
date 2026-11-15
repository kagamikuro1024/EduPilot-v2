// k6 — tải T1 trên dữ liệu `scripts/seed-t1.mjs` (FEAT-nfr-eval US-NFR-04 (a) và (d); SYSTEM_DESIGN §1.2, §5). Chạy qua `node benchmarks/load/run-t1.mjs`.
//   SCENARIO=browse    (a) 300 người dùng đồng thời: 300 luồng SSE thông báo (`GET /events`, mỗi luồng một VU) + 300 người duyệt (đọc : ghi ≈ 20 : 1, nghĩ 5–15 s)
//                      + heartbeat 10 req/s. Ngưỡng: đọc p95 < 300 ms, ghi p95 < 500 ms, 5xx < 0,5 %, SSE: byte đầu < 300 ms.
//   SCENARIO=dashboard (d) giảng viên mở Hôm nay / danh sách SV (kèm rủi ro) / sổ điểm / hồ sơ 360 của từng lớp. Ngưỡng: p95 < 3 s.
// Tài khoản: t1sv0001… / t1gv1…3 @edupilot.local, mật khẩu SEED_DEFAULT_PASSWORD. `META` (JSON) do run-t1.mjs truyền để ghi vào tệp kết quả.
import http from 'k6/http';
import { check, sleep } from 'k6';
import { Counter, Trend } from 'k6/metrics';

const BASE = (__ENV.BASE || 'https://localhost').replace(/\/$/, '');
const API = `${BASE}/api/v1`;
const PASSWORD = __ENV.SEED_DEFAULT_PASSWORD || '';
const SCENARIO = __ENV.SCENARIO || 'browse';
const VUS = Number(__ENV.VUS || 300);
const STUDENTS = Number(__ENV.STUDENTS || 1000); // quy mô seed-t1 (`--scale` làm nhỏ thì đặt lại)
const DURATION = __ENV.DURATION || '5m';
const pad = (n, w = 4) => String(n).padStart(w, '0');

const sseFirst = new Trend('sse_first_ms', true);
// US-NFR-04 AC4: STUDENT gọi API chỉ giảng viên giữa lúc tải: mọi lần phải 403, không lần nào 200.
const forbiddenOk = new Counter('forbidden_403');
const forbiddenLeak = new Counter('forbidden_leak');
// Chỉ 2xx / 3xx là thành công; heartbeat trả 204 (đã nằm trong dải).
http.setResponseCallback(http.expectedStatuses({ min: 200, max: 399 }));

const BROWSE = {
  browse: { executor: 'constant-vus', exec: 'browse', vus: VUS, duration: DURATION, gracefulStop: '10s' },
  sse: { executor: 'constant-vus', exec: 'sse', vus: VUS, duration: DURATION, gracefulStop: '130s' },
  forbidden: { executor: 'constant-arrival-rate', exec: 'forbidden', rate: 1, timeUnit: '1s', duration: DURATION, preAllocatedVUs: 2, maxVUs: 4 },
  heartbeat: { executor: 'constant-arrival-rate', exec: 'heartbeat', rate: 10, timeUnit: '1s', duration: DURATION, preAllocatedVUs: 20, maxVUs: 60 },
};
const OPTIONS = {
  browse: {
    scenarios: BROWSE,
    thresholds: { 'http_req_duration{kind:read}': ['p(95)<300'], 'http_req_duration{kind:write}': ['p(95)<500'], sse_first_ms: ['p(95)<300', 'max>0'], http_req_failed: ['rate<0.005'], forbidden_leak: ['count==0'], forbidden_403: ['count>=100'] },
  },
  dashboard: {
    scenarios: { dashboard: { executor: 'constant-vus', exec: 'dashboard', vus: 5, duration: DURATION } },
    thresholds: { http_req_duration: ['p(95)<3000'], http_req_failed: ['rate<0.005'] },
  },
}[SCENARIO];
if (!OPTIONS) throw new Error(`SCENARIO phải là browse | dashboard (nhận: ${SCENARIO})`);
export const options = { insecureSkipTLSVerify: true, summaryTrendStats: ['avg', 'med', 'p(90)', 'p(95)', 'p(99)', 'max'], setupTimeout: '10m', ...OPTIONS };

const hdr = (t) => ({ 'Content-Type': 'application/json', Accept: 'application/json', Authorization: `Bearer ${t}` });
function login(email) {
  const r = http.post(`${API}/auth/login`, JSON.stringify({ email: `${email}@edupilot.local`, password: PASSWORD }), { headers: { 'Content-Type': 'application/json' }, tags: { name: 'login', kind: 'setup' } });
  if (r.status !== 200) throw new Error(`đăng nhập ${email} → ${r.status} (đã seed-t1 và đặt SEED_DEFAULT_PASSWORD chưa?)`);
  return r.json('access_token');
}
const courseOf = (token) => http.get(`${API}/me/courses`, { headers: hdr(token), tags: { kind: 'setup' } }).json('items.0.course.id');

export function setup() {
  if (!PASSWORD) throw new Error('thiếu -e SEED_DEFAULT_PASSWORD');
  if (SCENARIO === 'dashboard') {
    return { teachers: [1, 2, 3].map((g) => { const t = login(`t1gv${g}`); return { token: t, courses: http.get(`${API}/me/courses`, { headers: hdr(t), tags: { kind: 'setup' } }).json('items').map((x) => x.course.id) }; }) };
  }
  // VUS sinh viên rải đều khắp 1.000 SV (nên khắp 20 lớp), mỗi người một lớp.
  const users = [];
  for (let i = 0; i < VUS; i++) {
    const token = login(`t1sv${pad(Math.floor((i * STUDENTS) / VUS) + 1)}`);
    users.push({ token, course: courseOf(token) });
  }
  return { users };
}

const pick = (a) => a[Math.floor(Math.random() * a.length)];
const AREAS = ['CHAT', 'THREADS', 'PRACTICE', 'LIBRARY', 'CALENDAR'];
const READS = [
  (u) => ['me_today', `${API}/me/today`],
  (u) => ['me_grades', `${API}/courses/${u.course}/me/grades`],
  (u) => ['me_attendance', `${API}/courses/${u.course}/me/attendance?limit=100`],
  (u) => ['threads', `${API}/courses/${u.course}/threads?limit=30`],
  (u) => ['library', `${API}/courses/${u.course}/library`],
  (u) => ['notifications', `${API}/notifications?limit=20`],
];

export function browse(d) {
  const u = d.users[__VU % d.users.length];
  if (Math.random() < 1 / 21) { // 20 đọc : 1 ghi
    const r = http.post(`${API}/activity/heartbeat`, JSON.stringify({ course_id: u.course, area: pick(AREAS), tab_id: '00000000-0000-4000-8000-000000000001' }), { headers: hdr(u.token), tags: { name: 'heartbeat_w', kind: 'write' } });
    check(r, { 'ghi 204': (x) => x.status === 204 });
  } else {
    const [name, url] = pick(READS)(u);
    const r = http.get(url, { headers: hdr(u.token), tags: { name, kind: 'read' } });
    check(r, { 'đọc 200': (x) => x.status === 200 });
  }
  sleep(5 + Math.random() * 10);
}

/** Một VU giữ một luồng SSE tới khi máy chủ đóng (`event: reconnect` sau 120 s) rồi nối lại. `waiting` = byte đầu (gateway ghi header ngay khi nhận). */
export function sse(d) {
  const u = d.users[__VU % d.users.length];
  const r = http.get(`${API}/events`, { headers: { ...hdr(u.token), Accept: 'text/event-stream' }, timeout: '130s', tags: { name: 'sse', kind: 'sse' } });
  if (check(r, { 'sse 200': (x) => x.status === 200 })) sseFirst.add(r.timings.waiting);
}

export function heartbeat(d) {
  const u = pick(d.users);
  const r = http.post(`${API}/activity/heartbeat`, JSON.stringify({ course_id: u.course, area: pick(AREAS), tab_id: '00000000-0000-4000-8000-000000000001' }), { headers: hdr(u.token), tags: { name: 'heartbeat', kind: 'write' } });
  check(r, { 'heartbeat 204': (x) => x.status === 204 });
}

export function forbidden(d) {
  const u = pick(d.users);
  const r = http.get(`${API}/courses/${u.course}/gradebook`, { headers: hdr(u.token), tags: { name: 'student_gradebook', kind: 'forbidden' }, responseCallback: http.expectedStatuses(403) });
  if (r.status === 403) forbiddenOk.add(1);
  else forbiddenLeak.add(1);
}

export function dashboard(d) {
  const t = pick(d.teachers), course = pick(t.courses), h = { headers: hdr(t.token) };
  http.get(`${API}/me/today`, { ...h, tags: { name: 'teacher_me_today' } });
  http.get(`${API}/courses/${course}/today`, { ...h, tags: { name: 'course_today' } });
  const list = http.get(`${API}/courses/${course}/students?limit=100`, { ...h, tags: { name: 'students_list' } });
  http.get(`${API}/courses/${course}/gradebook`, { ...h, tags: { name: 'gradebook' } });
  const first = list.status === 200 ? pick(list.json('items') || []) : null;
  if (first) http.get(`${API}/courses/${course}/students/${first.student.id}/profile`, { ...h, tags: { name: 'profile_360' } });
  sleep(1);
}

export function handleSummary(data) {
  const out = __ENV.OUT || `load-${SCENARIO}.json`;
  const m = __ENV.META ? JSON.parse(__ENV.META) : null; // thiếu META → check-meta.mjs báo lỗi, đúng ý: không có số đo hợp lệ
  return { [out]: JSON.stringify({ meta: m, scenario: SCENARIO, metrics: data.metrics, thresholds: Object.fromEntries(Object.entries(data.metrics).filter(([, v]) => v.thresholds).map(([k, v]) => [k, v.thresholds])) }, null, 2), stdout: `\n${SCENARIO}: xem ${out}\n` };
}
