# Caddy 2 reverse_proxy cho SSE và nhiều bản gateway (Q11, US-PG-07 AC3–AC6)

**Câu hỏi.** Với `deploy/caddy/Caddyfile` hiện tại (`dynamic a gateway 8080`, `flush_interval -1`, `response_header_timeout 35s`), SSE có bị đệm / cắt không, và Caddy có loại được bản gateway hỏng khi `--scale gateway=2` không?

**Kết luận.**
- **SSE ổn, giữ cấu hình.** Caddy tự flush ngay mọi response `text/event-stream` (mã nguồn v2.11.6 `streaming.go:277`), nên `flush_interval -1` **thừa nhưng vô hại**: đo trễ 0 ms cả khi có lẫn khi bỏ. `response_header_timeout 35s` không cắt stream (stream 45 s nhận 44/45 sự kiện, chỉ dừng vì client hết giờ). Client ngắt → gateway thấy `context canceled` ngay (đo 2,0 s / 3,0 s đúng lúc client bỏ), kể cả khi có `flush_interval -1`.
- **`dynamic a` qua được AC6 (tắt một bản):** `docker stop` → 0/200 lỗi; `docker kill` → 0/200 lỗi, một request chậm 2,26 s (chờ `dial_timeout` rồi thử bản khác).
- **Nhưng `dynamic` không có health check chủ động** — tài liệu Caddy: "active health checks do not run for dynamic upstreams". Bản gateway **treo nhưng vẫn nhận TCP** (giả lập bằng `docker pause`) không bao giờ bị loại: **101/200 lỗi**, 40/60 lỗi ở đuôi. Upstream tĩnh + `health_uri`: **1/200 lỗi**. FR-60 / AC6 ghi "health check chủ động" là sai với cấu hình hiện tại.
- Đề xuất: sprint 2 giữ `dynamic a` (AC6 đạt), sửa chữ trong spec thành "health check bị động"; ghi Nợ P10/PR: chuyển upstream tĩnh + `health_uri /api/v1/healthz` khi số bản gateway đã cố định ở môi trường chạy thật. Ghim `caddy:2.11` thay vì `caddy:2`.
- Độ chắc chắn: **cao** cho SSE và AC6; **trung bình** cho mức độ hiếm của "gateway treo" ở tải T1 [SUY LUẬN].

## Phương án — khám phá upstream

| Tiêu chí | `dynamic a gateway 8080` (repo) | Upstream tĩnh `edupilot-gateway-1/2:8080` + `health_uri` |
| --- | --- | --- |
| Khớp SRS 8.2 / Q11 | Đúng mặc định Q11 | Đúng phương án lùi của Q11 |
| Đổi số bản (`--scale N`) | Không sửa Caddyfile | Phải liệt kê đủ N tên |
| Health check chủ động | **Không chạy** (tài liệu Caddy) | Có (`health_interval 1s`) |
| Gateway dừng êm (`docker stop`) | 0/200 lỗi | 0/200 lỗi |
| Gateway chết cứng (`docker kill`) | 0/200 lỗi, max 2,27 s | 0/200 lỗi, max 2,26 s |
| Gateway treo, vẫn nhận TCP (`docker pause`) | **101/200 lỗi**, không tự hồi | **1/200 lỗi** |
| Công sức | 0 | ~6 dòng Caddyfile |
| Rủi ro | Treo một bản = mất ~50 % request tới khi người can thiệp | Caddyfile gắn với tên container compose |

Vì sao `dynamic` không tự loại bản treo: request tới bản treo không bao giờ có header; client bỏ trước 35 s → Caddy ghi nhận "client huỷ" chứ không phải "upstream lỗi", nên health check bị động (`max_fails`) không bao giờ đếm. [SUY LUẬN từ `reverseproxy.go:705` — `context.Canceled` được xử lý riêng; khớp với số đo.]

## Bằng chứng

- Caddy [v2.11.6](https://github.com/caddyserver/caddy/releases/tag/v2.11.6) (2026-10-01) là bản mới nhất; image `caddy:2` trên Docker Hub ngày 2026-10-02 còn là **v2.11.4** (`docker run --rm caddy:2 caddy version`).
- Tài liệu [`reverse_proxy`](https://caddyserver.com/docs/caddyfile/directives/reverse_proxy) (nguồn `caddyserver/website` `src/docs/markdown/caddyfile/directives/reverse_proxy.md`):
  - "Dynamic upstreams … active health checks do not run for dynamic upstreams".
  - `flush_interval`: "This option is ignored and responses are flushed immediately to the client if … `Content-Type: text/event-stream` … `Content-Length` is unknown". Cùng đoạn ghi `-1` "does not cancel the request to the backend even if the client disconnects early" — **không khớp** với v2.11.4 đo được (backend vẫn bị huỷ ngay), nên đừng dựa vào câu này.
  - `lb_retry_match`: "If the connection to the upstream failed, a retry is always allowed. By default, only `GET` requests are retried" (sau khi đã gửi được request).
  - `response_header_timeout` / `read_timeout`: mặc định không giới hạn; `stream_timeout` mặc định không giới hạn.
- Mã nguồn Caddy v2.11.6 (`git clone --branch v2.11.6`): `modules/caddyhttp/reverseproxy/streaming.go:271–291` — `flushInterval()` trả `-1` cho `text/event-stream` và cho `ContentLength == -1`; không có chỗ nào tách context khỏi request khi `FlushInterval < 0`.

PoC (`/tmp/research-caddy/`, Docker trên colima, `caddy:2` = v2.11.4): gateway giả `server/main.go` (Go, `/api/sse` phát 1 sự kiện/s kèm `ms` của server, `/api/slow`, `/api/healthz` trả `X-Instance-Id`, `Shutdown` êm khi SIGTERM) chạy 2 bản cùng alias mạng `gateway`. Caddy A = Caddyfile của repo (bỏ header/`frontend`), B = A bỏ `flush_interval -1`, C = upstream tĩnh `rpoc-gw1:8080 rpoc-gw2:8080` + `health_uri /api/healthz`, `health_interval 1s`, `health_timeout 1s`, `health_fails 1`. Kết quả thật:

```
# phân phối (A, 20 request)          10 4168ee994172 / 10 e5e47880aa53
# trễ SSE (lag = giờ nhận − giờ phát; ±2 ms là lệch đồng hồ VM)
port 18443 (A, có flush -1):  lag ∈ {-2,-1,0} ms (5 sự kiện)
port 18444 (B, không flush):  lag ∈ {-2,0} ms   (5 sự kiện)
# client ngắt (curl --max-time) → log gateway
sse closed after 3.0s: context canceled   (A và B)
slow cancelled after 2.0s                 (A và B)
# stream dài qua A: curl --max-time 45 … | grep -c '^data:'  → 44   (curl rc=28, tự hết giờ)
# failover.sh: 200 GET /api/healthz, 10/s, giây 5 làm hỏng rpoc-gw1
== 18443 stop:   200×200; đuôi 60: 0 lỗi; chậm nhất 0.26s
== 18443 kill:   200×200; đuôi 60: 0 lỗi; chậm nhất 2.27s
== 18443 pause:  101×000 99×200; đuôi 60: 40 lỗi   (curl --max-time 3)
== 18445 pause:  1×000 199×200; đuôi 60: 0 lỗi     (C, health check chủ động)
== 18445 stop:   200×200; đuôi 60: 0 lỗi
== 18445 kill:   200×200; đuôi 60: 0 lỗi; chậm nhất 2.26s
```

Thêm: client → Caddy đi HTTP/2 (`curl -w '%{http_version}'` → `2`), nên SSE không chiếm giới hạn 6 kết nối HTTP/1.1 mỗi origin của trình duyệt.

## Ảnh hưởng

- `deploy/caddy/Caddyfile`: không bắt buộc đổi cho sprint 2. `flush_interval -1` giữ cũng được (SRS FR-60 nêu tên); bỏ cũng không đổi hành vi SSE.
- `docker-compose.local.yml`: đổi `image: caddy:2` → `caddy:2.11` (hoặc thẻ cụ thể) — hiện không ghim, trái tinh thần "ghim phiên bản" của Q12.
- Spec (việc của BA qua PM): FR-60 và 07-AC6 ghi "health check chủ động" → với `dynamic a` chỉ có health check **bị động** (`fail_duration`, `max_fails`) + `lb_try_duration`. AC6 vẫn là phép thử quyết định và **đạt**.
- Nếu chuyển sang upstream tĩnh: dùng `health_uri /api/v1/healthz`, **không** dùng `/api/v1/readyz` — `readyz` trả 503 khi Redis chết (07-AC14) nên cả hai bản bị loại cùng lúc và toàn bộ API trả 503, trong khi đọc vẫn chạy được. Gateway đang drain tự đóng listener nên vẫn bị loại nhanh (PoC `stop` 0 lỗi).
- Nếu sau này Caddy được reload cấu hình khi đang có stream: thêm `stream_close_delay` để tránh mọi client SSE nối lại cùng lúc (tài liệu Caddy, mục Streaming). Repo đặt `admin off` nên hiện không reload.
- Không đụng `ARCHITECTURE.md` / `DECISIONS.md`.

## Đề xuất cho PM

`Q11: giữ dynamic a gateway 8080 cho sprint 2 — PoC docker stop/kill 0/200 lỗi, SSE flush ngay, 35 s không cắt stream (docs/research/2026-10-02-caddy-sse-multi-gateway.md); sửa FR-60/07-AC6 "health check chủ động" → "bị động" vì Caddy không chạy health check chủ động cho dynamic upstream (bản treo: 101/200 lỗi); ghim caddy:2.11; Nợ P10/PR: upstream tĩnh + health_uri /api/v1/healthz (bản treo: 1/200 lỗi).`
