# Bộ triển khai VPS (`ssh edupilot-vps`)

Từ 2026-11-25 Mac không chạy Docker (lệnh chủ dự án); stack demo `edupilot` chạy trên VPS ở `/opt/edupilot`, cùng 9router (`127.0.0.1:20128`), dịch vụ test `edupilot-test-*` (`docs/team/CONTEXT.md` mục «Chạy test») và stack đánh giá `ep-eval` (`benchmarks/eval/ep-eval.compose.yml`). `/opt/edupilot` là bản `git archive` không có `.git`: sửa ở worktree Mac, commit + push, rồi chép sang; lỡ sửa trên VPS thì chép ngược về và commit ngay.

| Tệp | Việc |
| --- | --- |
| `deploy/vps.compose.yml` | override chồng lên `docker-compose.local.yml`: bỏ cổng công khai, Caddyfile riêng |
| `deploy/caddy/Caddyfile.vps` | site `$EP_HOST` (HTTPS tự cấp) + `/edupilot/*` → MinIO |
| `deploy/vps-env.sh` | sinh `.env.local` (bí mật ngẫu nhiên, `chmod 600`, không commit) |
| `deploy/dc` | bọc `docker compose` với hai tệp trên, project `edupilot`, profile `ingest` |
| `deploy/vps-llm.mjs` | cấu hình LLM qua API quản trị: `router-chat` (9router) + `embed-gia` (fake) (`PHASE=probe|seed|live`, `BASE`, `MODEL`, `FALLBACK`; khoá 9router từ `ROUTER_API_KEY` hoặc `~/.9router/db/data.sqlite`) |
| `deploy/vps-llm-setup.mjs` | cấu hình OpenCode Go qua API quản trị: nhà cung cấp «OpenCode Go» (`qwen3-embedding-8b` 1536 chiều + `deepseek-v4.1-flash`) và tuyến đã chọn (mặc định `EMBEDDING,INSIGHT`). Khoá chỉ đọc từ **tệp** `OPENCODE_KEY_FILE` (chmod 600, ngoài repo, vd `/opt/edupilot/secrets/opencode-go.key`), không bao giờ in; mật khẩu admin = `SEED_DEFAULT_PASSWORD` trong `ENV_FILE`. Chạy lại an toàn. Đặt `EMBEDDING` (+ `INSIGHT`) **trước** seed, `CHAT` và tuyến khác **sau** seed (seed cần bản giả cho kết quả tất định) |

## LLM trên VPS

- Đổi tuyến (UI hay API) có hiệu lực ở gateway sau ≤ ~40 s (đo trên VPS 25/11; cơ chế: `ep:llm:reload` pub/sub + thăm dò 60 s). Đợi rồi mới kiểm.
- Tuyến không nhận chuỗi rỗng (`422 ROUTE_INVALID`). Đưa một tác vụ về bản giả: xoá hàng của tác vụ đó trong `llm_task_routes` (`deploy/dc exec -T postgres psql -U edupilot -d edupilot -c "delete from llm_task_routes where task = '<TASK>'"`), rồi đợi lần thăm dò 60 s hoặc tạo lại `gateway` và `worker`. Chi tiết: `README.md` mục 5.3.
- `scripts/check-knowledge-seed.mjs` chỉ tất định khi chữ OCR của tài liệu seed khớp bản ghi (tạo trên Mac arm64); trên VPS amd64 chữ OCR có thể khác nên kết quả có thể lệch (dev báo qua PM 25/11).

Quy trình đo LLM thật: `benchmarks/README.md` mục "LLM thật" và `benchmarks/real-llm/README.md`.
