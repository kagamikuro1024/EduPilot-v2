// US-NFR-07 AC3: p50 / p95 của ≥ 12 endpoint đại diện, mỗi endpoint 200 lần TUẦN TỰ (một VU, không tải nền) trên T1.
// Đọc: Hôm nay (SV, GV), danh sách thread, điểm của tôi, điểm danh của tôi, thư viện, sổ điểm, danh sách SV, hồ sơ 360, Hôm nay của lớp. Ghi: heartbeat, tạo phiên chat, đăng thread (không gọi LLM sinh chữ trong request).
// Chạy bằng benchmarks/load/run-t1.mjs baseline. Ngưỡng: đọc p95 < 300 ms, ghi p95 < 500 ms (SYSTEM_DESIGN §5).
import http from 'k6/http';
import { fail } from 'k6';

const BASE = (__ENV.BASE || 'https://localhost').replace(/\/$/, '');
const API = `${BASE}/api/v1`;
const N = Number(__ENV.ITERATIONS || 200);
http.setResponseCallback(http.expectedStatuses({ min: 200, max: 399 }));

const READS = ['sv_me_today', 'sv_threads', 'sv_me_grades', 'sv_me_attendance', 'sv_library', 'gv_me_today', 'gv_course_today', 'gv_students', 'gv_gradebook', 'gv_profile_360'];
const WRITES = ['w_heartbeat', 'w_chat_session', 'w_thread'];
const thresholds = {};
for (const n of READS) thresholds[`http_req_duration{name:${n}}`] = ['p(95)<300'];
for (const n of WRITES) thresholds[`http_req_duration{name:${n}}`] = ['p(95)<500'];
thresholds.http_req_failed = ['rate<0.005'];
export const options = { insecureSkipTLSVerify: true, summaryTrendStats: ['avg', 'med', 'p(90)', 'p(95)', 'p(99)', 'max'], scenarios: { seq: { executor: 'per-vu-iterations', vus: 1, iterations: N, maxDuration: '20m' } }, thresholds };

const H = (t, extra) => ({ 'Content-Type': 'application/json', Authorization: `Bearer ${t}`, ...extra });
const login = (u) => { const r = http.post(`${API}/auth/login`, JSON.stringify({ email: `${u}@edupilot.local`, password: __ENV.SEED_DEFAULT_PASSWORD }), { headers: { 'Content-Type': 'application/json' } }); if (r.status !== 200) fail(`login ${u} → ${r.status}`); return r.json('access_token'); };

export function setup() {
  const sv = login('t1sv0001'), gv = login('t1gv1');
  const course = http.get(`${API}/me/courses`, { headers: H(sv) }).json('items.0.course.id');
  const list = http.get(`${API}/courses/${course}/students?limit=1`, { headers: H(gv) }).json('items.0.student.id');
  return { sv, gv, course, student: list };
}

export default function (d) {
  const sv = H(d.sv), gv = H(d.gv), c = d.course, T = (name) => ({ tags: { name } });
  http.get(`${API}/me/today`, { headers: sv, ...T('sv_me_today') });
  http.get(`${API}/courses/${c}/threads?limit=30`, { headers: sv, ...T('sv_threads') });
  http.get(`${API}/courses/${c}/me/grades`, { headers: sv, ...T('sv_me_grades') });
  http.get(`${API}/courses/${c}/me/attendance?limit=100`, { headers: sv, ...T('sv_me_attendance') });
  http.get(`${API}/courses/${c}/library`, { headers: sv, ...T('sv_library') });
  http.get(`${API}/me/today`, { headers: gv, ...T('gv_me_today') });
  http.get(`${API}/courses/${c}/today`, { headers: gv, ...T('gv_course_today') });
  http.get(`${API}/courses/${c}/students?limit=100`, { headers: gv, ...T('gv_students') });
  http.get(`${API}/courses/${c}/gradebook`, { headers: gv, ...T('gv_gradebook') });
  http.get(`${API}/courses/${c}/students/${d.student}/profile`, { headers: gv, ...T('gv_profile_360') });
  http.post(`${API}/activity/heartbeat`, JSON.stringify({ course_id: c, area: 'CHAT', tab_id: '00000000-0000-4000-8000-000000000001' }), { headers: sv, ...T('w_heartbeat') });
  http.post(`${API}/chat/sessions`, JSON.stringify({ course_id: c }), { headers: H(d.sv, { 'Idempotency-Key': crypto.randomUUID() }), ...T('w_chat_session') });
  http.post(`${API}/courses/${c}/threads`, JSON.stringify({ title: `Câu hỏi đo p95 ${crypto.randomUUID().slice(0, 8)}`, body: 'Nội dung thử để đo độ trễ ghi.' }), { headers: H(d.sv, { 'Idempotency-Key': crypto.randomUUID() }), ...T('w_thread') });
}

export function handleSummary(data) {
  const out = __ENV.OUT;
  if (!out) return {};
  const m = __ENV.META ? JSON.parse(__ENV.META) : null;
  const rows = {};
  for (const [k, v] of Object.entries(data.metrics)) { const x = /^http_req_duration\{name:(.+)\}$/.exec(k); if (x) rows[x[1]] = { p50: v.values.med, p95: v.values['p(95)'], p99: v.values['p(99)'], max: v.values.max, ok: v.thresholds ? Object.values(v.thresholds).every((t) => t.ok) : null }; }
  return { [out]: JSON.stringify({ meta: m, scenario: 'baseline', iterations: N, endpoints: rows, metrics: data.metrics, thresholds: Object.fromEntries(Object.entries(data.metrics).filter(([, v]) => v.thresholds).map(([k, v]) => [k, v.thresholds])) }, null, 2), stdout: `\n${out}\n` };
}
