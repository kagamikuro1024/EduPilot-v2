# Dev · lát 1 sprint 9.5 (phần không đụng QUESTIONS): meta, seed-t1, k6 T1, vps-llm

Nhánh `sprint/9.5-nfr`. Phục vụ US-NFR-01 (AC3, AC4), US-NFR-03, US-NFR-04 (a)(d), US-NFR-08 (cấu hình LLM, P-95-4/6).

| Tệp | Việc |
| --- | --- |
| `benchmarks/lib/meta.mjs`, `benchmarks/check-meta.mjs` | khối `meta` (+ `k6_cpus`, `demo_stack` cho `load-*.json`); `refuseDemo(BASE)` từ chối cổng 80/443/3000; `check-meta` bỏ qua 9 tệp đo cũ trước 9.5 (`e1`, `pe-exam-submit-*`) |
| `scripts/seed-t1.mjs` | SQL trực tiếp, tất định theo `SEED_RNG`; DB phải tên `edupilot_t1`, project phải `ep-*`; `--reset`, `--scale`, `--no-vectors`; ghi `benchmarks/reports/seed-t1.json` |
| `benchmarks/load/t1.js`, `run-t1.mjs` | (a) `browse`: 300 SSE + 300 người duyệt (đọc : ghi 20 : 1) + heartbeat 10 req/s; (d) `dashboard`; k6 trong VM `--cpus 1.5 --memory 1.5g`, chung mạng với `caddy` |
| `benchmarks/load/chat.js` | `-e ACCOUNTS=t1` dùng 50 SV T1 (kịch bản (b)) |
| `deploy/vps-llm.mjs` | `PHASE=probe|seed|live`; `BASE`, `ENV_FILE`, `ADMIN_EMAIL`, `MODEL`, `FALLBACK`, `CHAT_RPM`; `gemini-chat` + `gemini-embed` (`type: gemini`); bỏ 9router / sqlite / `EP_HOST` |
| `benchmarks/README.md` | quy trình Mac: dựng stack T1, seed, k6, LLM thật |

## Đã chạy
- `seed-t1` trên Postgres 18 + pgvector dùng một lần (migration 00001–00017 bằng `gateway migrate up`, admin giả): quy mô đủ chạy 20 s, 403 MB. Số dòng: `chat_messages` 90.000, `study_time_daily` 525.000, `attendance_records` 15.000, `submissions` 4.000, `content_chunks` 20.000 (bảng §1.2 ±10 %). `--reset` rồi chạy lại hai lần ở `--scale 0.1`: số dòng bằng nhau từng bảng. Chạy lại không `--reset` bị từ chối; `COMPOSE_PROJECT_NAME=edupilot` bị từ chối.
- `k6 inspect` cả ba kịch bản + `chat.js ACCOUNTS=t1`; `node --check` mọi `.mjs`; `vps-llm.mjs` từ chối BASE demo và thiếu MODEL.

## Chưa chạy (cần stack, nhường QC)
- Gateway thật trên `edupilot_t1`: đăng nhập `t1sv…`, `/me/today`, sổ điểm, hồ sơ 360 có ra 200 không; 20 cặp SV khác lớp 403/404 (AC4). Seed ghi thẳng SQL nên đây là rủi ro lớn nhất: trường thiếu (vd `user_settings`) có thể làm một API 500.
- Chạy `t1.js` thật (số đo p95), `PHASE=probe` (cần khoá Gemini; 9 lời gọi), `PHASE=seed|live` của `vps-llm.mjs`.
- Mật khẩu: mọi tài khoản T1 dùng lại `password_hash` của admin (cùng bcrypt, cùng `SEED_DEFAULT_PASSWORD`).

## Cần PM biết
- `deploy/vps-seed.sh`, `deploy/README.md`, `deploy/vps.compose.yml` còn nói VPS / 9router; `vps-seed.sh` gọi `vps-llm.mjs` không có `BASE` nên hỏng. Sẽ gỡ / thay trong lát US-NFR-01 (README Mac, `real-llm/README.md`).
- Dựng stack ở offset ≠ 0 cần đặt thêm `EP_PORT_MOCK_GRAPH`, `EP_PORT_MOCK_IMAP` (không dời theo offset) và `BLOB_PUBLIC_ENDPOINT=localhost:<9000+offset>` (mặc định trỏ vào MinIO của stack demo, tệp seed có thể rơi vào đó). Đã ghi vào `benchmarks/README.md`.
- Seed T0 trên Mac cần `--profile ingest` (docling) mới qua bước 11 và 15; không có thì dừng ở bước 11.
