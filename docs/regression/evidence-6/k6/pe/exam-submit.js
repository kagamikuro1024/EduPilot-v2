// k6 — nộp bài thi và tự lưu (US-PE-09 AC6; SYSTEM_DESIGN §5). Chạy qua Caddy trên stack TEST (gateway dựng bằng docker-compose.test.yml, judge thật `JUDGE_PARALLELISM=2`)
// với dữ liệu mẫu (`node scripts/seed.mjs`: 30 sinh viên lớp 761987, mật khẩu SEED_DEFAULT_PASSWORD). Mỗi lần chạy `setup()` TỰ dựng một bài thi tải riêng (1 câu code 10 test + 1 câu trắc nghiệm,
// mở ngay) bằng API thật rồi cho 30 sinh viên bắt đầu lượt — không để lại dữ liệu nào ngoài một bài thi tên "Tải k6 …".
//
//   k6 run benchmarks/load/exam-submit.js -e BASE=https://localhost:773 -e SCENARIO=judge_burst
//   SCENARIO ∈ judge_burst | autosave | chat | mixed        (xem bảng ngưỡng ở dưới)
//   Chạy cặp chat / mixed 3 lần với -e RUN=1|2|3 rồi `node scripts/ttft-ratio.mjs` (hoặc scripts/gate-pe.sh với GATE_K6=1).
//
//   judge_burst : 60 lần nộp code (<bits/stdc++.h>, 10 test) rải 5 phút từ 30 sinh viên      → judge_done_ms p95 < 60 000, judge_ie == 0, lỗi HTTP < 0,5 %
//   autosave    : 300 phiên đồng thời lưu câu trả lời mỗi 2 s trong 3 phút (10 phiên / lượt)  → autosave_ms p95 < 150, lỗi < 0,5 %
//   chat        : 50 luồng chat giả `POST /_test/llm/chat` (INTERACTIVE, KHÔNG stream) 5 phút → đo chat_ttft_ms = thời gian tới byte đầu của câu trả lời (đường cơ sở cho mixed)
//   mixed       : judge_burst + chat + `Chạy thử` rải đều                                      → chat_ttft_ms p95 < 1500 (SLO TTFT), run_ms p95 ≤ 5 000; TỈ LỆ so với `chat` riêng do scripts/gate-pe.sh tính (#18)
// Góp ý #18: TTFT phải đo với provider `fake` có trễ THẬT. Stack test-seed đặt FAKE_LLM_LATENCY=300-300 (trễ trước token đầu = TTFT) và LLM_MAX_CONCURRENCY=100 (50 luồng không xếp hàng).
// Vì sao không stream: route `_test` ghi header SSE ngay, trước token đầu → `waiting` của k6 chỉ đo header (13 ms dù trễ 300 ms). Không stream thì thân trả sau đúng độ trễ của provider + hàng đợi `INTERACTIVE`.
// Mỗi kịch bản chạy 3 cặp (`-e RUN=1|2|3`, báo cáo `...-<kịch bản>-<RUN>.json`); cổng lấy TRUNG VỊ của 3 tỉ lệ p95(mixed)/p95(chat) ≤ 1,2.
import http from 'k6/http';
import { check, sleep, fail } from 'k6';
import exec from 'k6/execution';
import { Counter, Trend } from 'k6/metrics';

const BASE = (__ENV.BASE || 'https://localhost:773').replace(/\/$/, '');
const API = `${BASE}/api/v1`;
const PASSWORD = __ENV.SEED_DEFAULT_PASSWORD || 'Edupilot#Seed-2026';
const SCENARIO = __ENV.SCENARIO || 'judge_burst';
const RUN = __ENV.RUN ? `-${__ENV.RUN}` : '';
const DOMAIN = '@edupilot.local';
const STUDENTS = ['sv.gioi', 'sv.kha', 'sv.nguyco', ...Array.from({ length: 27 }, (_, i) => `sv${String(i + 4).padStart(2, '0')}`)];

const judgeDone = new Trend('judge_done_ms', true);
const judgeIE = new Counter('judge_ie');
const autosave = new Trend('autosave_ms', true);
const runMs = new Trend('run_ms', true);
const chatTTFT = new Trend('chat_ttft_ms', true);

// Chỉ 2xx / 3xx là thành công: 429 (hạn mức) hay 4xx khác tính vào http_req_failed. (Bản đầu cho 4xx qua → 429 trả nhanh làm số đo đẹp giả.)
http.setResponseCallback(http.expectedStatuses({ min: 200, max: 399 }));

const SRC = `#include <bits/stdc++.h>
using namespace std;
int main() {
  long long a, b;
  cin >> a >> b;
  cout << a + b << "\\n";
  return 0;
}
`;

const burst = { executor: 'constant-arrival-rate', exec: 'judgeBurst', rate: 12, timeUnit: '1m', duration: '5m', preAllocatedVUs: 30, maxVUs: 60 };
const scenarios = {
  judge_burst: { judge_burst: burst },
  autosave: { autosave: { executor: 'constant-vus', exec: 'autosaveVU', vus: 300, duration: '3m' } },
  chat: { chat: { executor: 'constant-vus', exec: 'chatVU', vus: 50, duration: '5m' } },
  mixed: {
    judge_burst: burst,
    chat: { executor: 'constant-vus', exec: 'chatVU', vus: 50, duration: '5m' },
    run_probe: { executor: 'constant-arrival-rate', exec: 'runProbe', rate: 6, timeUnit: '1m', duration: '5m', preAllocatedVUs: 6, maxVUs: 12 },
  },
}[SCENARIO];
if (!scenarios) throw new Error(`SCENARIO không hợp lệ: ${SCENARIO}`);

const thresholds = { http_req_failed: ['rate<0.005'] };
if (SCENARIO === 'judge_burst' || SCENARIO === 'mixed') {
  thresholds.judge_done_ms = ['p(95)<60000'];
  thresholds.judge_ie = ['count==0'];
}
if (SCENARIO === 'autosave') thresholds.autosave_ms = ['p(95)<150'];
if (SCENARIO === 'mixed') {
  thresholds.run_ms = ['p(95)<5000'];
  thresholds.chat_ttft_ms = ['p(95)<1500'];
}

export const options = { insecureSkipTLSVerify: true, scenarios, thresholds, setupTimeout: '10m', summaryTrendStats: ['avg', 'p(50)', 'p(95)', 'max'] };

// ---- tiện ích ---------------------------------------------------------------------------------------------------------
const uuid = () => 'xxxxxxxx-xxxx-4xxx-yxxx-xxxxxxxxxxxx'.replace(/[xy]/g, (c) => { const r = (Math.random() * 16) | 0; return (c === 'x' ? r : (r & 3) | 8).toString(16); });
const key = () => `k6-${Math.random().toString(36).slice(2)}-${Date.now()}`;
const auth = (t) => ({ Authorization: `Bearer ${t}`, 'Content-Type': 'application/json', Origin: BASE });
function call(method, path, body, token, extra = {}) {
  const headers = { ...auth(token), ...(extra.headers || {}) };
  if (method === 'POST') headers['Idempotency-Key'] = headers['Idempotency-Key'] || key();
  return http.request(method, API + path, body === undefined ? null : JSON.stringify(body), { headers, tags: extra.tags, timeout: extra.timeout || '60s' });
}
const must = (res, ok, what) => {
  if (!ok.includes(res.status)) fail(`${what}: HTTP ${res.status} ${String(res.body).slice(0, 200)}`);
  return res.json();
};
function login(who) {
  for (let i = 0; i < 6; i++) { // hạn mức đăng nhập theo IP: chờ `retry_after` rồi thử lại
    const r = http.post(`${API}/auth/login`, JSON.stringify({ email: who + DOMAIN, password: PASSWORD }), { headers: { 'Content-Type': 'application/json', Origin: BASE } });
    if (r.status === 429) {
      sleep(Math.min(30, Number(r.json('retry_after') || 5)) + 0.5);
      continue;
    }
    return must(r, [200], `login ${who}`).access_token;
  }
  return fail(`login ${who}: vẫn 429 sau 6 lần`);
}
function listAll(path, token) {
  const out = [];
  let cursor = '';
  for (let i = 0; i < 20; i++) {
    const r = call('GET', `${path}${path.includes('?') ? '&' : '?'}limit=100${cursor ? `&cursor=${encodeURIComponent(cursor)}` : ''}`, undefined, token);
    const b = must(r, [200], `GET ${path}`);
    out.push(...b.items);
    if (!b.next_cursor) break;
    cursor = b.next_cursor;
  }
  return out;
}
function waitJob(id, token) {
  for (let i = 0; i < 180; i++) {
    const b = must(call('GET', `/jobs/${id}`, undefined, token), [200], 'job');
    if (b.status === 'SUCCEEDED') return b;
    if (b.status === 'FAILED') fail(`việc ${id} lỗi: ${JSON.stringify(b.error)}`);
    sleep(1);
  }
  return fail(`việc ${id} quá hạn`);
}

// ---- dựng dữ liệu -----------------------------------------------------------------------------------------------------
export function setup() {
  const teacher = login('teacher');
  const courses = must(call('GET', '/me/courses?limit=50', undefined, teacher), [200], 'me/courses').items;
  const course = courses.find((x) => x.course.class_code === '761987')?.course.id;
  if (!course) fail('Chưa có lớp 761987: chạy node scripts/seed.mjs trước.');
  const qb = `/courses/${course}/questions`;
  const have = listAll(qb, teacher);
  // câu code tải: 10 test (2 mẫu + 8 ẩn), dùng lại nếu đã có
  const title = 'Tải k6 — tổng hai số (10 test)';
  let q = have.find((x) => x.title === title);
  if (!q || q.review_status !== 'APPROVED') {
    if (!q) q = must(call('POST', qb, { type: 'CODE', title, topic: 'Tải', stem: 'Đọc hai số nguyên a b và in a + b.' }, teacher), [201], 'tạo câu code');
    const cur = must(call('GET', `${qb}/${q.id}`, undefined, teacher), [200], 'đọc câu');
    must(call('PUT', `${qb}/${q.id}/code`, { languages: ['cpp17'], time_limit_ms: 2000, memory_limit_mb: 128, checker: 'EXACT', starter_code: { cpp17: '#include <bits/stdc++.h>\nint main(){}\n' }, reference: { language: 'cpp17', source: SRC }, version: cur.version }, teacher), [200], 'cấu hình code');
    for (let i = 0; i < 10; i++) {
      must(call('POST', `${qb}/${q.id}/testcases`, { name: `t${i + 1}`, input: `${i + 1} ${i * 3}\n`, expected: `${i + 1 + i * 3}\n`, is_sample: i < 2, weight: 1 }, teacher), [201], 'thêm test');
    }
    const ids = must(call('GET', `${qb}/${q.id}/testcases?limit=100`, undefined, teacher), [200], 'đọc test').items.map((t) => t.id);
    must(call('POST', `${qb}/${q.id}/testcases/approve`, { ids }, teacher), [200], 'duyệt test');
    waitJob(must(call('POST', `${qb}/${q.id}/reference/verify`, {}, teacher), [202], 'xác minh').job_id, teacher);
    const fin = must(call('GET', `${qb}/${q.id}`, undefined, teacher), [200], 'đọc câu');
    must(call('PUT', `${qb}/${q.id}/review`, { decision: 'APPROVE', version: fin.version }, teacher), [200], 'duyệt câu');
  }
  const mcq = have.find((x) => x.type === 'MCQ_SINGLE' && x.review_status === 'APPROVED');
  if (!mcq) fail('Chưa có câu trắc nghiệm APPROVED: chạy node scripts/seed.mjs trước.');
  // bài thi tải: mở sau 8 s, dài 25 phút, làm 20 phút
  const now = Date.now();
  const eb = `/courses/${course}/exams`;
  const e = must(call('POST', eb, { title: `Tải k6 ${SCENARIO} ${new Date(now).toISOString()}`, duration_minutes: 20, multi_scoring: 'PARTIAL', opens_at: new Date(now + 8000).toISOString(), closes_at: new Date(now + 25 * 60000).toISOString() }, teacher), [201], 'tạo bài');
  must(call('PUT', `${eb}/${e.id}/items`, { items: [{ question_id: q.id, points: '5' }, { question_id: mcq.id, points: '5' }], version: e.version }, teacher), [200], 'chọn câu');
  must(call('POST', `${eb}/${e.id}/schedule`, {}, teacher), [200], 'lên lịch');
  sleep(Math.max(0, (now + 9500 - Date.now()) / 1000));
  const students = STUDENTS.map((who) => {
    const token = login(who);
    const tab = uuid();
    const r = call('POST', `${eb}/${e.id}/attempts`, undefined, token, { headers: { 'X-Exam-Tab': tab } });
    const b = must(r, [200, 201], `bắt đầu ${who}`);
    const codeItem = b.items.find((i) => i.type === 'CODE');
    const mcqItem = b.items.find((i) => i.type !== 'CODE');
    return { token, tab, attempt: b.attempt.id, codeItem: codeItem.item_id, mcqItem: mcqItem.item_id, option: mcqItem.options[0].id };
  });
  const admin = SCENARIO === 'chat' || SCENARIO === 'mixed' ? login('admin') : '';
  return { course, exam: e.id, students, admin };
}

const hdr = (d, s, idem) => ({ 'X-Exam-Tab': s.tab, ...(idem ? { 'Idempotency-Key': key() } : {}) });
const base = (d, s) => `/courses/${d.course}/exams/${d.exam}/attempts/${s.attempt}`;

// ---- kịch bản ---------------------------------------------------------------------------------------------------------
export function judgeBurst(d) {
  const s = d.students[exec.scenario.iterationInTest % d.students.length];
  const t0 = Date.now();
  const r = call('POST', `${base(d, s)}/code/${s.codeItem}/submit`, { language: 'cpp17', source: SRC }, s.token, { headers: hdr(d, s, true), tags: { name: 'submit' } });
  if (!check(r, { 'nộp code 202': (x) => x.status === 202 })) return;
  const id = r.json('submission_id');
  for (let i = 0; i < 120; i++) {
    const g = call('GET', `${base(d, s)}/submissions/${id}`, undefined, s.token, { tags: { name: 'poll' } });
    const st = g.status === 200 ? g.json('status') : '';
    if (st === 'DONE' || st === 'ERROR') {
      judgeDone.add(Date.now() - t0);
      if (st === 'ERROR' || g.json('verdict') === 'IE') judgeIE.add(1);
      return;
    }
    sleep(1);
  }
  judgeDone.add(120000);
  judgeIE.add(1);
}

export function autosaveVU(d) {
  const s = d.students[(exec.vu.idInTest - 1) % d.students.length];
  const r = call('PUT', `${base(d, s)}/answers`, { items: [{ item_id: s.mcqItem, answer: { option_ids: [s.option] } }] }, s.token, { headers: hdr(d, s, false), tags: { name: 'autosave' } });
  if (check(r, { 'lưu 200': (x) => x.status === 200 })) autosave.add(r.timings.duration);
  sleep(2);
}

export function chatVU(d) {
  const r = http.post(`${API}/_test/llm/chat`, JSON.stringify({ task: 'CHAT', prompt: 'Giải thích ngắn gọn về hàm băm.', lane: 'INTERACTIVE', stream: false }), { headers: auth(d.admin), tags: { name: 'chat' }, timeout: '60s' });
  if (check(r, { 'chat 200': (x) => x.status === 200 })) chatTTFT.add(r.timings.waiting);
  else sleep(1); // lỗi: không quay vòng nóng
}

export function runProbe(d) {
  const s = d.students[exec.scenario.iterationInTest % d.students.length];
  const t0 = Date.now();
  const r = call('POST', `${base(d, s)}/code/${s.codeItem}/run`, { language: 'cpp17', source: SRC }, s.token, { headers: hdr(d, s, true), tags: { name: 'run' } });
  if (!check(r, { 'chạy thử 202': (x) => x.status === 202 })) return;
  const id = r.json('run_id');
  for (let i = 0; i < 60; i++) {
    const g = call('GET', `${base(d, s)}/runs/${id}`, undefined, s.token, { tags: { name: 'poll_run' } });
    if (g.status === 200 && (g.json('status') === 'DONE' || g.json('status') === 'ERROR')) {
      runMs.add(Date.now() - t0);
      return;
    }
    sleep(0.5);
  }
  runMs.add(30000);
}

export function handleSummary(data) {
  const day = new Date().toISOString().slice(0, 10);
  const slim = { scenario: SCENARIO, date: day, base: BASE, metrics: {}, thresholds: {} };
  for (const [name, m] of Object.entries(data.metrics)) {
    slim.metrics[name] = m.values;
    if (m.thresholds) slim.thresholds[name] = Object.fromEntries(Object.entries(m.thresholds).map(([k, v]) => [k, v.ok]));
  }
  return { [`benchmarks/reports/pe-exam-submit-${day}-${SCENARIO}${RUN}.json`]: JSON.stringify(slim, null, 2), stdout: textSummary(slim) };
}

function textSummary(slim) {
  const lines = [`\n=== pe-exam-submit · ${slim.scenario} ===`];
  for (const k of ['judge_done_ms', 'judge_ie', 'autosave_ms', 'run_ms', 'chat_ttft_ms', 'http_req_failed']) {
    const v = slim.metrics[k];
    if (v) lines.push(`${k}: ${Object.entries(v).map(([a, b]) => `${a}=${typeof b === 'number' ? Math.round(b * 1000) / 1000 : b}`).join(' ')}`);
  }
  for (const [m, t] of Object.entries(slim.thresholds)) lines.push(`ngưỡng ${m}: ${Object.entries(t).map(([k, ok]) => `${k} ${ok ? 'PASS' : 'FAIL'}`).join(', ')}`);
  return lines.join('\n') + '\n';
}
