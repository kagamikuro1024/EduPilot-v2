# Số liệu nền của bản Go (sprint 2 — PG)

Nguồn: US-PG-07 AC15. Máy đo, ngày đo và 4 số dưới đây do lead điền ở cổng nghiệm thu PG;
số không đạt mục tiêu thì ghi số thật + lý do, **không** nới SLO (SRS 8.5).

- Máy đo: colima 4 CPU / 8 GiB (điền lại nếu khác) · Ngày đo: (điền)
- Stack: `docker compose --env-file .env.local -f docker-compose.local.yml -p edupilot up -d --scale gateway=2 --wait` (chế độ mặc định)

| Chỉ số | Mục tiêu | Đo được | Cách đo |
| --- | --- | --- | --- |
| RAM nghỉ gateway (MiB) | — | (điền) | `docker stats --no-stream --format '{{.Name}} {{.MemUsage}}'` sau 60 s rảnh |
| RAM nghỉ worker (MiB) | — | (điền) | như trên |
| Thời gian khởi động gateway (ms) | — | (điền) | `$C logs --no-log-prefix gateway \| jq -r 'select(.msg=="gateway ready").startup_ms' \| head -1` |
| Kích thước image edupilot-gateway (byte) | < 40000000 | (điền) | `docker image inspect -f '{{.Size}}' edupilot-gateway` |
| Kích thước image edupilot-worker (byte) | < 40000000 | (điền) | `docker image inspect -f '{{.Size}}' edupilot-worker` |
| p95 `/api/v1/healthz` (ms) | < 300 | (điền) | `k6 run -e BASE=https://localhost -e TOKEN="$(tok STUDENT $U1)" benchmarks/load/smoke.js` |

Ngưỡng tham chiếu khác của k6 smoke: p95 `/api/v1/readyz` < 300 ms, p95 `/api/v1/jobs/{id}` (404, có truy vấn DB qua PgBouncer) < 300 ms,
p95 ghi `/api/v1/_test/items` < 500 ms (chỉ ở chế độ thử), `http_req_failed` < 0.005, `checks` > 0.99.

## Ghi chú khi số không đạt

(điền: số thật, nguyên nhân, việc cần làm ở phase sau)
