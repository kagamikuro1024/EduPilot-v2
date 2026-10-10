// k6 — chat riêng (US-P3-08 AC5; SRS FEAT-private-chat-pii 8, D47 mục 4). Chạy trên stack TEST đã seed (`node scripts/seed.mjs`: 30 sinh viên lớp 761987, tài liệu READY, mật khẩu SEED_DEFAULT_PASSWORD).
//
//   k6 run benchmarks/load/chat.js -e BASE=https://localhost:773 -e SCENARIO=first_event
//   k6 run benchmarks/load/chat.js -e BASE=https://localhost:773 -e SCENARIO=ttft
//
//   first_event : 50 luồng (mỗi luồng MỘT tài khoản sinh viên seed riêng — chat riêng chỉ cho một lượt sinh chữ mỗi người, CHAT_BUSY — nên số luồng ≤ số sinh viên có lớp trong seed: 51), 60 s,
//                 provider giả trễ 5–15 s (stack: FAKE_LLM_LATENCY=5000-15000, LLM_MAX_CONCURRENCY=100); mỗi luồng gửi tiếp khi lượt trước đã xong HẲN (stream đóng + 1 s cho bộ khoá nhả)
//                 → first_event_ms p95 < 300 = thời gian từ lúc gửi yêu cầu tới byte đầu của phản hồi SSE. Gateway ghi header + `event: status` NGAY khi nhận xong thân, trước khi gọi provider,
//                   nên `timings.waiting` của k6 chính là sự kiện đầu (kết nối đã mở sẵn bằng keep-alive). Provider trễ 5–15 s KHÔNG được làm sự kiện đầu chậm lại.
//   ttft        : 20 luồng, 60 s, provider giả trễ 300 ms (FAKE_LLM_LATENCY=300-300; SRS gọi là FAKE_LLM_TTFT_MS) → hai chỉ số riêng:
//                   ttft_cache_ms (câu đã có trong bộ nhớ đệm câu trả lời)  p95 < 1500
//                   ttft_rag_ms   (câu mới: truy xuất + sinh)               p95 < 4000
//                 TTFT = thời gian tới sự kiện `token` đầu. k6 không thấy từng khung giữa luồng, nên tính: first_event (đo ở client) + (mốc thời gian của khung `token` đầu − mốc của khung `status` đầu);
//                 hai mốc là phần mili-giây trong `id: <lượt>:<id Redis Stream>` của khung SSE (cùng đồng hồ máy chủ).
// Vượt ngưỡng — hoặc KHÔNG có mẫu nào (`*_samples` count>0: ngưỡng p95 của Trend rỗng là 0, đẹp giả) — → k6 thoát mã khác 0.
import http from 'k6/http';
import { check, fail, sleep } from 'k6';
import exec from 'k6/execution';
import { Counter, Trend } from 'k6/metrics';

const BASE = (__ENV.BASE || 'https://localhost:773').replace(/\/$/, '');
const API = `${BASE}/api/v1`;
const PASSWORD = __ENV.SEED_DEFAULT_PASSWORD || 'Edupilot#Seed-2026';
const SCENARIO = __ENV.SCENARIO || 'first_event';
const DOMAIN = '@edupilot.local';
// 30 sinh viên lớp 761987 + 20 sinh viên chỉ học lớp 761988 (sv31…sv50): mỗi người ghi danh ACTIVE nên chat được; đều là tài khoản seed.
const SEED_STUDENTS = ['sv.gioi', 'sv.kha', 'sv.nguyco', ...Array.from({ length: 47 }, (_, i) => `sv${String(i + 4).padStart(2, '0')}`)];

const firstEvent = new Trend('first_event_ms', true);
const ttftCache = new Trend('ttft_cache_ms', true);
const ttftRag = new Trend('ttft_rag_ms', true);
// Trend rỗng có p95 = 0 (qua ngưỡng giả) nên mỗi chỉ số kèm bộ đếm mẫu với ngưỡng `count>0`.
const nFirst = new Counter('first_event_samples');
const nCache = new Counter('ttft_cache_samples');
const nRag = new Counter('ttft_rag_samples');

// Chỉ 2xx / 3xx là thành công: 429 (hạn mức) hay 409 tính vào http_req_failed — không để lỗi trả nhanh làm số đo đẹp giả.
http.setResponseCallback(http.expectedStatuses({ min: 200, max: 399 }));

const WARM = [
  'Phishing là gì và làm sao nhận ra một email lừa đảo?',
  'Điều kiện để được dự thi cuối kỳ là gì?',
  'Ransomware hoạt động như thế nào?',
  'Thời hạn phúc khảo điểm thi là bao lâu?',
  'Tấn công DDoS khác tấn công DoS ở điểm nào?',
];
const FRESH = [
  'Man-in-the-middle xảy ra trong tình huống nào',
  'Sinh viên bị cảnh báo học vụ khi nào',
  'Botnet được dùng để làm gì',
  'Quy định về vắng mặt tối đa trong một học phần',
];

const OPTIONS = {
  first_event: {
    scenarios: { first_event: { executor: 'constant-vus', exec: 'firstEventVU', vus: 50, duration: '60s', gracefulStop: '40s' } },
    thresholds: { first_event_ms: ['p(95)<300'], first_event_samples: ['count>0'], http_req_failed: ['rate<0.01'] },
  },
  ttft: {
    scenarios: { ttft: { executor: 'constant-vus', exec: 'ttftVU', vus: 20, duration: '60s', gracefulStop: '20s' } },
    thresholds: { ttft_cache_ms: ['p(95)<1500'], ttft_rag_ms: ['p(95)<4000'], ttft_cache_samples: ['count>0'], ttft_rag_samples: ['count>0'], http_req_failed: ['rate<0.01'] },
  },
}[SCENARIO];
if (!OPTIONS) fail(`SCENARIO phải là first_event | ttft (nhận: ${SCENARIO})`);
export const options = { insecureSkipTLSVerify: true, summaryTrendStats: ['avg', 'med', 'p(90)', 'p(95)', 'max'], ...OPTIONS };

const json = (token) => ({ 'Content-Type': 'application/json', Accept: 'application/json', ...(token ? { Authorization: `Bearer ${token}` } : {}) });

function login(key) {
  const r = http.post(`${API}/auth/login`, JSON.stringify({ email: key + DOMAIN, password: PASSWORD }), { headers: json(), tags: { name: 'login' } });
  if (r.status !== 200) fail(`đăng nhập ${key} → ${r.status} ${r.body}`);
  return r.json('access_token');
}

/** setup: đăng nhập từng sinh viên seed (mỗi luồng một người, lớp đầu tiên của người đó) và tạo một phiên chat riêng; `ttft` còn làm nóng bộ nhớ đệm bằng các câu WARM. */
export function setup() {
  const vus = SCENARIO === 'first_event' ? 50 : 20;
  const sessions = [];
  for (let i = 0; i < vus; i++) {
    const token = login(SEED_STUDENTS[i]);
    const course = http.get(`${API}/me/courses`, { headers: json(token) }).json('items')[0]?.course.id;
    if (!course) fail(`${SEED_STUDENTS[i]} chưa có lớp (chạy node scripts/seed.mjs)`);
    const r = http.post(`${API}/chat/sessions`, JSON.stringify({ course_id: course }), { headers: { ...json(token), 'Idempotency-Key': `k6-chat-${SCENARIO}-${Date.now()}-${i}` } });
    if (r.status !== 201 && r.status !== 200) fail(`tạo phiên chat → ${r.status} ${r.body}`);
    sessions.push({ token, sid: r.json('id') });
  }
  if (SCENARIO === 'ttft') for (const q of WARM) send(sessions[0].token, sessions[0].sid, q); // đưa câu trả lời vào bộ nhớ đệm
  return { sessions };
}

function send(token, sid, content) {
  return http.post(`${API}/chat/sessions/${sid}/messages`, JSON.stringify({ content }), {
    headers: { ...json(token), Accept: 'text/event-stream', 'Idempotency-Key': crypto.randomUUID() }, tags: { name: 'chat_send' }, timeout: '90s',
  });
}

const msOf = (body, event) => {
  const m = new RegExp(`id: \\d+:(\\d+)-\\d+\\r?\\nevent: ${event}\\b`).exec(body);
  return m ? Number(m[1]) : null;
};

export function firstEventVU(d) {
  const s = d.sessions[exec.vu.idInTest - 1];
  const r = send(s.token, s.sid, WARM[Math.floor(Math.random() * WARM.length)] + ` (${exec.scenario.iterationInInstance})`);
  if (check(r, { 'chat 200': (x) => x.status === 200 && String(x.headers['Content-Type']).startsWith('text/event-stream') })) { firstEvent.add(r.timings.waiting); nFirst.add(1); }
  else console.error(`chat → ${r.status} ${String(r.body).slice(0, 120)}`); // lỗi thật (429 hạn mức, 409 bận…) KHÔNG bị che: vẫn tính vào http_req_failed
  sleep(1); // lượt này đã xong hẳn; chờ bộ khoá CHAT_BUSY nhả rồi mới gửi tiếp
}

export function ttftVU(d) {
  const s = d.sessions[exec.vu.idInTest - 1];
  const cached = exec.scenario.iterationInInstance % 2 === 0;
  const q = cached ? WARM[exec.vu.idInTest % WARM.length] : `${FRESH[exec.vu.idInTest % FRESH.length]} ${crypto.randomUUID().slice(0, 8)}?`;
  const r = send(s.token, s.sid, q);
  if (!check(r, { 'chat 200': (x) => x.status === 200 })) return;
  const status = msOf(r.body, 'status'), token = msOf(r.body, 'token');
  if (status === null || token === null) return; // câu trả lời mẫu (không sinh chữ): không có khung token
  (cached ? ttftCache : ttftRag).add(r.timings.waiting + (token - status));
  (cached ? nCache : nRag).add(1);
}
