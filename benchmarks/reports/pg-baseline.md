# Số liệu nền của bản Go (sprint 2 — PG)

Nguồn: US-PG-07 AC15. Máy đo, ngày đo và 4 số dưới đây do lead điền ở cổng nghiệm thu PG;
số không đạt mục tiêu thì ghi số thật + lý do, **không** nới SLO (SRS 8.5).

- Máy đo: Apple Silicon (11 lõi), colima 4 CPU / 8 GiB (aarch64) · Ngày đo: 2026-10-02
- Stack: `docker compose --env-file .env.local -f docker-compose.local.yml -p edupilot up -d --scale gateway=2 --wait` (chế độ mặc định)

| Chỉ số | Mục tiêu | Đo được | Cách đo |
| --- | --- | --- | --- |
| RAM nghỉ gateway (MiB) | — | 3,98 (gateway-1), 3,96 (gateway-2) | `docker stats --no-stream --format '{{.Name}} {{.MemUsage}}'` sau 60 s rảnh |
| RAM nghỉ worker (MiB) | — | 5,23 | như trên |
| Thời gian khởi động gateway (ms) | — | 9 và 69 (hai bản; bản 69 ms khởi động cùng lúc `migrate` / Postgres) | `$C logs --no-log-prefix gateway \| jq -r 'select(.msg=="gateway ready").startup_ms' \| head -1` |
| Kích thước image edupilot-gateway (byte) | < 40000000 | 9424483 (nonroot) | `docker image inspect -f '{{.Size}}' edupilot-gateway` |
| Kích thước image edupilot-worker (byte) | < 40000000 | 8917258 (nonroot) | `docker image inspect -f '{{.Size}}' edupilot-worker` |
| p95 `/api/v1/healthz` (ms) | < 300 | 5,75 (qua Caddy, hai gateway) | `k6 run -e BASE=https://localhost -e TOKEN="$(tok STUDENT $U1)" benchmarks/load/smoke.js` |

Ngưỡng tham chiếu khác của k6 smoke: p95 `/api/v1/readyz` < 300 ms, p95 `/api/v1/jobs/{id}` (404, có truy vấn DB qua PgBouncer) < 300 ms,
p95 ghi `/api/v1/_test/items` < 500 ms (chỉ ở chế độ thử), `http_req_failed` < 0.005, `checks` > 0.99.

## Ghi chú khi số không đạt

Tất cả đạt mục tiêu; không nới SLO. Cùng lần chạy k6: p95 `readyz` 6,92 ms, p95 `jobs/{id}` (404, có truy vấn DB qua PgBouncer) 8,57 ms, `http_req_failed` 0,00 %, `checks` 100 %. Biến thể ghi (`-e TEST_ROUTES=1`, stack `docker-compose.test.yml`): p95 `POST /_test/items` 52,09 ms, `http_req_failed` 0,00 %. Ghi chú: kịch bản `jobs404` (10 req/s × 30 s = 300) nằm đúng giới hạn `RATE_LIMIT_IP_PER_MIN=300`, nên request thứ 301 trong cùng cửa sổ phút nhận 429 (đúng hành vi); `smoke.js` chấp nhận 404 hoặc 429 ở check này. Không chạy hai lần liên tiếp trong chưa đầy 60 s từ cùng IP.
