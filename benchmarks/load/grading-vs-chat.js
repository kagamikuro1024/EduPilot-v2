// k6 — chấm bài hàng loạt (làn BATCH) KHÔNG làm chậm chat riêng của sinh viên (US-P7-07 AC5; SYSTEM_DESIGN §5). Chạy trên stack TEST đã seed
// (`node scripts/seed.mjs`; US-S9-SEED: lớp 761987 có bài tập BT-TAI với ≥ 200 bài nộp MATCHED chưa chấm; provider giả).
//
//   Lượt 1 — đo chat khi KHÔNG chấm:
//     k6 run benchmarks/load/grading-vs-chat.js -e BASE=https://localhost:773 -e COURSE_ID=<lớp> -e ASSIGNMENT_ID=<bài> -e PHASE=baseline
//   Lượt 2 — kích `Chấm tất cả` rồi đo chat song song (đưa p95 TTFT của lượt 1 vào BASELINE_TTFT_MS):
//     k6 run benchmarks/load/grading-vs-chat.js -e BASE=… -e COURSE_ID=… -e ASSIGNMENT_ID=… -e PHASE=grading -e BASELINE_TTFT_MS=<p95 lượt 1>
//
//   baseline : 20 luồng chat riêng (mỗi luồng MỘT sinh viên seed), 40 s, provider giả trễ 300 ms (FAKE_LLM_LATENCY=300-300).
//   grading  : như trên, nhưng ở giây 5 TEACHER gọi `POST …/assignments/{aid}/grade-all` (hàng chục–hàng trăm bài, làn BATCH) và chat vẫn chạy 40 s.
//              first_event_ms (chờ hàng + mở luồng) p95 ≤ 500; ttft_ms p95 ≤ 1,2 × BASELINE_TTFT_MS (không chậm hơn +20 %).
// Vượt ngưỡng — hoặc KHÔNG có mẫu nào hay `grade-all` không xếp việc nào (grading_queued count>0) — → k6 thoát mã khác 0.
import http from 'k6/http';
import { check, fail, sleep } from 'k6';
import exec from 'k6/execution';
import { Counter, Trend } from 'k6/metrics';

const BASE = (__ENV.BASE || 'https://localhost:773').replace(/\/$/, '');
const API = `${BASE}/api/v1`;
const PASSWORD = __ENV.SEED_DEFAULT_PASSWORD || 'Edupilot#Seed-2026';
const PHASE = __ENV.PHASE || 'baseline';
const DOMAIN = '@edupilot.local';
const COURSE_ID = __ENV.COURSE_ID;
const ASSIGNMENT_ID = __ENV.ASSIGNMENT_ID;
const BASELINE = Number(__ENV.BASELINE_TTFT_MS || 0);
const VUS = 20;
const SEED_STUDENTS = ['sv.gioi', 'sv.kha', 'sv.nguyco', ...Array.from({ length: 17 }, (_, i) => `sv${String(i + 4).padStart(2, '0')}`)];
const QUESTIONS = [
  'Phishing là gì và làm sao nhận ra một email lừa đảo?',
  'Ransomware hoạt động như thế nào?',
  'Tấn công DDoS khác tấn công DoS ở điểm nào?',
  'Điều kiện để được dự thi cuối kỳ là gì?',
];

const firstEvent = new Trend('first_event_ms', true);
const ttft = new Trend('ttft_ms', true);
const nFirst = new Counter('first_event_samples');
const nTtft = new Counter('ttft_samples');
const nQueued = new Counter('grading_queued'); // số bài `grade-all` đã xếp (Trend rỗng có p95 = 0: kèm bộ đếm mẫu)

http.setResponseCallback(http.expectedStatuses({ min: 200, max: 399 }));

if (!['baseline', 'grading'].includes(PHASE)) fail(`PHASE phải là baseline | grading (nhận: ${PHASE})`);
if (PHASE === 'grading' && (!COURSE_ID || !ASSIGNMENT_ID || !BASELINE)) fail('PHASE=grading cần COURSE_ID, ASSIGNMENT_ID và BASELINE_TTFT_MS (p95 TTFT của lượt baseline)');

const scenarios = { chat: { executor: 'constant-vus', exec: 'chatVU', vus: VUS, duration: '40s', gracefulStop: '30s' } };
const thresholds = { ttft_samples: ['count>0'], first_event_samples: ['count>0'], http_req_failed: ['rate<0.01'] };
if (PHASE === 'grading') {
  scenarios.grade = { executor: 'per-vu-iterations', exec: 'gradeAll', vus: 1, iterations: 1, startTime: '5s', maxDuration: '30s' };
  thresholds.first_event_ms = ['p(95)<500'];
  thresholds.ttft_ms = [`p(95)<${Math.round(BASELINE * 1.2)}`];
  thresholds.grading_queued = ['count>0'];
}
export const options = { insecureSkipTLSVerify: true, summaryTrendStats: ['avg', 'med', 'p(90)', 'p(95)', 'max'], scenarios, thresholds };

const json = (token) => ({ 'Content-Type': 'application/json', Accept: 'application/json', ...(token ? { Authorization: `Bearer ${token}` } : {}) });

function login(key) {
  const r = http.post(`${API}/auth/login`, JSON.stringify({ email: key + DOMAIN, password: PASSWORD }), { headers: json(), tags: { name: 'login' } });
  if (r.status !== 200) fail(`đăng nhập ${key} → ${r.status} ${r.body}`);
  return r.json('access_token');
}

/** setup: mỗi luồng một sinh viên seed + một phiên chat riêng; TEACHER (chỉ ở lượt `grading`). */
export function setup() {
  const sessions = [];
  for (let i = 0; i < VUS; i++) {
    const token = login(SEED_STUDENTS[i]);
    const course = http.get(`${API}/me/courses`, { headers: json(token) }).json('items')[0]?.course.id;
    if (!course) fail(`${SEED_STUDENTS[i]} chưa có lớp (chạy node scripts/seed.mjs)`);
    const r = http.post(`${API}/chat/sessions`, JSON.stringify({ course_id: course }), { headers: { ...json(token), 'Idempotency-Key': `k6-gvc-${PHASE}-${Date.now()}-${i}` } });
    if (r.status !== 201 && r.status !== 200) fail(`tạo phiên chat → ${r.status} ${r.body}`);
    sessions.push({ token, sid: r.json('id') });
  }
  return { sessions, teacher: PHASE === 'grading' ? login('teacher') : null };
}

const msOf = (body, event) => {
  const m = new RegExp(`id: \\d+:(\\d+)-\\d+\\r?\\nevent: ${event}\\b`).exec(body);
  return m ? Number(m[1]) : null;
};

export function chatVU(d) {
  const s = d.sessions[exec.vu.idInTest - 1];
  const q = `${QUESTIONS[exec.vu.idInTest % QUESTIONS.length]} (${crypto.randomUUID().slice(0, 8)})`;
  const r = http.post(`${API}/chat/sessions/${s.sid}/messages`, JSON.stringify({ content: q }), {
    headers: { ...json(s.token), Accept: 'text/event-stream', 'Idempotency-Key': crypto.randomUUID() }, tags: { name: 'chat_send' }, timeout: '90s',
  });
  if (check(r, { 'chat 200': (x) => x.status === 200 })) {
    firstEvent.add(r.timings.waiting);
    nFirst.add(1);
    const status = msOf(r.body, 'status'), token = msOf(r.body, 'token');
    if (status !== null && token !== null) { // TTFT = sự kiện `token` đầu (như chat.js)
      ttft.add(r.timings.waiting + (token - status));
      nTtft.add(1);
    }
  } else {
    console.error(`chat → ${r.status} ${String(r.body).slice(0, 120)}`); // lỗi thật KHÔNG bị che
  }
  sleep(3.5); // dưới CHAT_RATE_PER_MIN mặc định: đo dưới hạn mức THẬT
}

export function gradeAll(d) {
  const r = http.post(`${API}/courses/${COURSE_ID}/assignments/${ASSIGNMENT_ID}/grade-all`, JSON.stringify({}), {
    headers: { ...json(d.teacher), 'Idempotency-Key': `k6-grade-${Date.now()}` }, tags: { name: 'grade_all' },
  });
  if (check(r, { 'grade-all 202': (x) => x.status === 202 })) nQueued.add(r.json('queued'));
  else console.error(`grade-all → ${r.status} ${String(r.body).slice(0, 160)}`);
}
