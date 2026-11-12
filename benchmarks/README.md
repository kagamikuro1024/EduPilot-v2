# Đo trên Mac

Thiết bị: MacBook Apple Silicon 18 GB, Colima 6 CPU / 10 GB, k6 trong VM 1,5 CPU. Mỗi lúc một stack đo (`-p ep-<vai>`, cổng lệch); không ghi lên stack demo. Xong thì `docker compose -p ep-<vai> down -v`.
Mọi tệp `benchmarks/reports/**/*.json` có khối `meta` (`node benchmarks/check-meta.mjs` kiểm).

## Dựng stack đo T1 (1.000 SV)
```bash
export COMPOSE_PROJECT_NAME=ep-t1 EP_PORT_OFFSET=400 EP_PORT_MOCK_GRAPH=8490 EP_PORT_MOCK_IMAP=8491 BLOB_PUBLIC_ENDPOINT=localhost:9400
export POSTGRES_DB=edupilot_t1 DATABASE_URL='postgres://edupilot:edupilot-dev@postgres:5432/edupilot_t1?sslmode=disable' \
       PGBOUNCER_URL='postgres://edupilot:edupilot-dev@pgbouncer:6432/edupilot_t1?sslmode=disable'
pnpm dev                                    # SEED_ON_EMPTY_DB=false: chỉ cần admin
docker compose --env-file .env.local -f docker-compose.local.yml -p $COMPOSE_PROJECT_NAME exec -T gateway /gateway admin create   # nhập SEED_DEFAULT_PASSWORD
SEED_DEFAULT_PASSWORD=… node scripts/seed-t1.mjs            # ≈ 20 s; --reset để làm lại; --scale 0.5 để thu nhỏ (ghi quy mô thực)
```
Khi dựng bằng `-f docker-compose.local.yml -f benchmarks/load/k6-stack.yml`, `K6_LAT=5000-15000` đặt độ trễ provider giả (kịch bản chat).

## Tải
```bash
SEED_DEFAULT_PASSWORD=… node benchmarks/load/run-t1.mjs browse --duration 5m      # (a) 300 VU + 300 SSE + heartbeat 10 req/s
SEED_DEFAULT_PASSWORD=… node benchmarks/load/run-t1.mjs dashboard --duration 2m   # (d) Hôm nay / danh sách SV / sổ điểm / hồ sơ 360 (giảng viên)
```
k6 chạy trong VM (`grafana/k6`, `--cpus 1.5 --memory 1.5g`, chung mạng với `caddy` của stack đo). Chat (b): `k6 run benchmarks/load/chat.js -e ACCOUNTS=t1 -e SCENARIO=ttft …`.

## LLM thật (Gemini)
```bash
PHASE=probe node deploy/vps-llm.mjs                                  # đầu phiên: chọn MODEL / FALLBACK (≤ 9 lời gọi)
BASE=https://localhost:<cổng HTTPS> PHASE=seed node deploy/vps-llm.mjs    # trước khi seed nạp tài liệu: embedding thật, chat giả
BASE=… MODEL=… FALLBACK=… PHASE=live node deploy/vps-llm.mjs             # sau seed
```
Khoá `GEMINI_API_KEY` chỉ ở `.env.local` / biến môi trường. Không bật LLM thật ở stack demo.
