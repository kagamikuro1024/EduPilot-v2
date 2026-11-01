# Triển khai lên VPS (OPS-VPS)

Một máy (netcup, Debian, Docker + compose), cả stack như `pnpm dev` nhưng **chỉ Caddy mở 80/443**. `APP_ENV=dev` (seed demo, mock-imap, mock-graph chạy); Postgres / Redis / MinIO / mailpit / mock-* chỉ bind `127.0.0.1`.

| Tệp | Việc |
| --- | --- |
| `deploy/vps.compose.yml` | override chồng lên `docker-compose.local.yml`: bỏ cổng công khai, Caddyfile riêng, gắn gateway + worker vào network ngoài `9router_default` |
| `deploy/caddy/Caddyfile.vps` | site `$EP_HOST` (HTTPS tự cấp) + `/edupilot/*` → MinIO (URL ký sẵn path-style cho trình duyệt) |
| `deploy/vps-env.sh` | sinh `.env.local` (bí mật ngẫu nhiên, `chmod 600`, KHÔNG commit) |
| `deploy/vps-llm.mjs` | khai nhà cung cấp 9router + tuyến cho 7 tác vụ qua API quản trị |
| `deploy/dc` | `docker compose` với đủ tệp / profile |

## Lần đầu
```bash
git clone -b ops/vps-deploy https://github.com/kagamikuro1024/EduPilot-v2.git /opt/edupilot && cd /opt/edupilot
EP_HOST=<hostname công khai> bash deploy/vps-env.sh        # .env.local
deploy/dc up -d --build --wait                              # build + dựng (docling build lâu một lần)
# Node 24 chỉ để chạy seed / vps-llm (tải bản linux-x64 vào /opt/node, ngoài repo)
export PATH=/opt/node/bin:$PATH API_URL=https://<hostname>/api/v1 MAILPIT_URL=http://127.0.0.1:8025 \
  SEED_ADMIN_CMD="deploy/dc exec -T gateway /gateway admin create"
node scripts/seed.mjs                                       # seed bằng API thật (LLM giả + bản ghi demo: seed cần để trích quy chế)
node deploy/vps-llm.mjs                                     # SAU seed: chuyển sang LLM thật
```
Sau seed, bỏ bốn dòng `RATE_LIMIT_IP_PER_MIN`, `AUTH_*_IP_*` khỏi `.env.local` rồi `deploy/dc up -d gateway` (giới hạn theo IP trở lại mặc định).

## LLM
- 9router (`http://9router:20128/v1`, OpenAI-compatible) qua loại `openai_compatible`: `json_object` + schema trong lời nhắc (9router bỏ qua `json_schema`), Go vẫn kiểm schema. Model mọi tác vụ sinh chữ: `ag/gemini-3.8-flash`; RPM 20 / nhà cung cấp (tài khoản miễn phí phía sau 9router có hạn mức).
- **Không có embeddings** ở 9router → tác vụ `EMBEDDING` dùng nhà cung cấp `fake` (tất định 1536 chiều); tìm lai vẫn có nhánh `tsv`. Đây là giới hạn của bản triển khai này.
- Khoá 9router đọc thẳng từ sqlite của 9router bởi `vps-llm.mjs`, gửi cho gateway (lưu mã hoá trong DB); không nằm trong `.env.local`, không in ra. KHÔNG đụng container / thư mục / cron của 9router.
- Đổi model: `MODEL=ag/gemini-3.7-flash node deploy/vps-llm.mjs` chỉ đặt lại tuyến nếu nhà cung cấp đã có model đó; muốn model mới hãy thêm ở `/settings/llm`.

## Cập nhật / vận hành
```bash
cd /opt/edupilot && git pull && deploy/dc up -d --build --wait
deploy/dc ps ; deploy/dc logs -f gateway worker ; docker stats --no-stream
ssh -L 8025:127.0.0.1:8025 -L 9001:127.0.0.1:9001 root@<vps>   # mailpit, MinIO console
```
Sao lưu: volume `edupilot_postgres_data` (và `minio_data`); chưa có lịch sao lưu tự động (thuộc phase PR).
