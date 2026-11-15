// k6 — điểm danh bằng mã (FEAT-attendance-selfcheck US-AS-14 AC3, SYSTEM_DESIGN §5, ARCHITECTURE §11.8). Chạy trên stack T1 đã seed (`scripts/seed-t1.mjs`: 20 lớp T1-01…T1-20, 50 SV / lớp,
// t1sv0001… / t1gv1…3 @edupilot.local, mật khẩu SEED_DEFAULT_PASSWORD). 200 SV = 4 lớp đầu (T1-01…04; lớp n có SV 50(n−1)+1 … 50n; giảng viên t1gv((n−1)%3+1)).
//
//   k6 run benchmarks/load/checkin.js -e BASE=https://localhost:843 -e SEED_DEFAULT_PASSWORD=… -e SCENARIO=spread   200 SV dàn đều trong 30 s
//   k6 run benchmarks/load/checkin.js -e BASE=… -e SCENARIO=burst                                                  200 lượt dồn trong 5 s
//   (k6 trong VM: `docker run --rm --network <mạng của stack> grafana/k6 run - <benchmarks/load/checkin.js`, như run-t1.mjs; dựng stack kèm `-f benchmarks/load/k6-stack.yml` để nới DISPLAY_RATE_PER_MIN)
//
// Mỗi lần chạy dùng MỘT buổi riêng (spread: buổi 1; burst: buổi 2) nên mỗi lần đúng 200 hàng `attendance_checkins`. setup(): bật self_checkin (chu kỳ mã 60 s để một token sống trọn cả pha đo), mở cửa sổ ở CẢ 20 lớp,
// lấy token từ `…/display` của từng lớp dùng. Trong khi đó 20 màn chiếu poll `…/display` 1 req / 10 s (kịch bản `display`) và một luồng đọc nền (`GET …/sessions`) để kiểm «route khác không vượt SLO».
// Ngưỡng: `POST …/checkins` p95 ≤ 500 ms, lỗi 0 %, đúng 200 lượt ở teardown (đếm qua API Staff); display p95 < 300 ms; đọc nền p95 < 300 ms.
// Lượt gửi không có `device` (k6 không ký ECDSA): cờ NO_DEVICE_KEY, vẫn 201 — số đo là đường nóng không có bước xác minh chữ ký; ghi rõ ở báo cáo.
import http from 'k6/http';
import { check, fail, sleep } from 'k6';
import { Counter } from 'k6/metrics';
import exec from 'k6/execution';
import { SharedArray } from 'k6/data';

const BASE = (__ENV.BASE || 'https://localhost').replace(/\/$/, '');
const API = `${BASE}/api/v1`;
const PASSWORD = __ENV.SEED_DEFAULT_PASSWORD || '';
const SCENARIO = __ENV.SCENARIO || 'spread';
const USED = 4; // lớp có SV điểm danh (4 × 50 = 200 SV)
const ALL = 20; // lớp có màn chiếu mở
const PER = 50;
const DOMAIN = '@edupilot.local';
const pad = (n, w = 4) => String(n).padStart(w, '0');
const SESSION_NO = { spread: 1, burst: 2 }[SCENARIO];
if (!SESSION_NO) fail(`SCENARIO phải là spread | burst (nhận: ${SCENARIO})`);

const okCount = new Counter('checkin_ok');
const rowsMismatch = new Counter('checkin_rows_mismatch');
http.setResponseCallback(http.expectedStatuses({ min: 200, max: 299 }));

const students = new SharedArray('students', () => Array.from({ length: USED * PER }, (_, i) => i + 1));

export const options = {
  insecureSkipTLSVerify: true,
  summaryTrendStats: ['avg', 'med', 'p(90)', 'p(95)', 'p(99)', 'max'],
  setupTimeout: '5m',
  scenarios: {
    checkin: SCENARIO === 'spread'
      ? { executor: 'per-vu-iterations', exec: 'checkin', vus: 200, iterations: 1, maxDuration: '2m', startTime: '2s' } // mỗi VU ngủ ngẫu nhiên 0–30 s rồi gửi
      : { executor: 'per-vu-iterations', exec: 'checkin', vus: 200, iterations: 1, maxDuration: '2m', startTime: '2s' }, // burst: ngủ 0–5 s
    display: { executor: 'constant-arrival-rate', exec: 'display', rate: 2, timeUnit: '1s', duration: SCENARIO === 'spread' ? '40s' : '15s', preAllocatedVUs: 10, maxVUs: 30 }, // 20 lớp × 1 req / 10 s
    background: { executor: 'constant-arrival-rate', exec: 'background', rate: 5, timeUnit: '1s', duration: SCENARIO === 'spread' ? '40s' : '15s', preAllocatedVUs: 5, maxVUs: 20 },
  },
  thresholds: {
    'http_req_duration{name:checkin}': ['p(95)<500'],
    'http_req_failed{name:checkin}': ['rate==0'],
    checkin_ok: ['count==200'],
    checkin_rows_mismatch: ['count==0'],
    'http_req_duration{name:display}': ['p(95)<300'],
    'http_req_failed{name:display}': ['rate==0'],
    'http_req_duration{name:background}': ['p(95)<300'],
  },
};

const hdr = (t, extra = {}) => ({ 'Content-Type': 'application/json', Accept: 'application/json', ...(t ? { Authorization: `Bearer ${t}` } : {}), ...extra });
function login(local) {
  const r = http.post(`${API}/auth/login`, JSON.stringify({ email: local + DOMAIN, password: PASSWORD }), { headers: hdr(), tags: { name: 'login' } });
  if (r.status !== 200) fail(`đăng nhập ${local} → ${r.status} (đã seed-t1 và đặt SEED_DEFAULT_PASSWORD chưa?)`);
  return r.json('access_token');
}
const idem = () => `k6-${Math.random().toString(36).slice(2)}${Date.now()}`;
const teacherOf = (n) => `t1gv${((n - 1) % 3) + 1}`;

export function setup() {
  if (!PASSWORD) fail('Thiếu SEED_DEFAULT_PASSWORD');
  const tTok = {};
  for (const g of ['t1gv1', 't1gv2', 't1gv3']) tTok[g] = login(g);
  const out = { courses: [], studentTokens: {} };
  // id lớp T1-01…20 qua giảng viên
  const idByCode = {};
  for (const g of Object.keys(tTok)) {
    const r = http.get(`${API}/me/courses?limit=100`, { headers: hdr(tTok[g]) });
    for (const it of r.json('items')) idByCode[it.course.class_code] = it.course.id;
  }
  for (let n = 1; n <= ALL; n++) {
    const code = `T1-${pad(n, 2)}`;
    const id = idByCode[code];
    if (!id) fail(`không thấy lớp ${code}`);
    const token = tTok[teacherOf(n)];
    const c = http.get(`${API}/courses/${id}`, { headers: hdr(token) });
    const ver = c.json('settings.version');
    const put = http.put(`${API}/courses/${id}/settings`, JSON.stringify({ attendance: { self_checkin: true, rotate_seconds: 60, late_after_min: 30, require: 'WEB', campus_cidrs: [] }, version: ver }), { headers: hdr(token, { 'Idempotency-Key': idem() }) });
    if (put.status !== 200) fail(`bật self_checkin ${code} → ${put.status} ${put.body}`);
    const ss = http.get(`${API}/courses/${id}/sessions?limit=100`, { headers: hdr(token) }).json('items').sort((a, b) => a.session_no - b.session_no);
    const session = ss.find((s) => s.session_no === SESSION_NO);
    if (!session) fail(`${code} chưa có buổi ${SESSION_NO}`);
    const w = http.post(`${API}/courses/${id}/sessions/${session.id}/checkin-windows`, JSON.stringify({ minutes: 60 }), { headers: hdr(token, { 'Idempotency-Key': idem() }) });
    if (w.status !== 201 && w.status !== 409) fail(`mở cửa sổ ${code} → ${w.status} ${w.body}`);
    const wid = w.status === 201 ? w.json('id') : w.json('details.window_id');
    out.courses.push({ n, id, session: session.id, wid, teacher: teacherOf(n) });
  }
  // token lấy MỘT lần mỗi lớp có SV (chu kỳ 60 s: sống trọn pha đo); các lớp còn lại chỉ để màn chiếu poll
  for (const c of out.courses.slice(0, USED)) {
    const d = http.get(`${API}/courses/${c.id}/sessions/${c.session}/checkin-windows/${c.wid}/display`, { headers: hdr(tTok[c.teacher]) });
    if (d.status !== 200) fail(`display ${c.n} → ${d.status}`);
    c.token = d.json('qr_url').split('#')[1];
  }
  out.teachers = tTok;
  // đăng nhập 200 SV ở setup (không tính vào số đo)
  for (const i of students) out.studentTokens[i] = login(`t1sv${pad(i)}`);
  return out;
}

export function checkin(data) {
  const i = exec.scenario.iterationInTest + 1; // 1…200 (idInTest đếm cả VU của display / background nên lệch)
  const n = Math.ceil(i / PER);
  const c = data.courses[n - 1];
  sleep(Math.random() * (SCENARIO === 'spread' ? 30 : 5));
  const r = http.post(`${API}/courses/${c.id}/checkins`, JSON.stringify({ token: c.token }), { headers: hdr(data.studentTokens[i], { 'Idempotency-Key': idem() }), tags: { name: 'checkin' } });
  if (check(r, { '201': (x) => x.status === 201 })) okCount.add(1);
}

export function display(data) {
  const c = data.courses[exec.scenario.iterationInTest % ALL];
  http.get(`${API}/courses/${c.id}/sessions/${c.session}/checkin-windows/${c.wid}/display`, { headers: hdr(data.teachers[c.teacher]), tags: { name: 'display' } });
}

export function background(data) {
  const c = data.courses[exec.scenario.iterationInTest % ALL];
  http.get(`${API}/courses/${c.id}/sessions?limit=20`, { headers: hdr(data.teachers[c.teacher]), tags: { name: 'background' } });
}

export function teardown(data) {
  // đúng 200 lượt ở 4 lớp (đếm qua API Staff; mỗi lớp ≤ 100 nên một trang)
  let total = 0;
  for (const c of data.courses.slice(0, USED)) {
    const r = http.get(`${API}/courses/${c.id}/sessions/${c.session}/checkins?limit=100`, { headers: hdr(data.teachers[c.teacher]) });
    total += (r.json('items') || []).length;
  }
  if (total !== USED * PER) rowsMismatch.add(1);
  console.log(`lượt điểm danh đếm được: ${total} (cần ${USED * PER})`);
}
