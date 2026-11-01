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
# GEMINI_API_KEY (khoá embeddings Gemini, do chủ dự án cấp) thêm vào .env.local TRƯỚC bước dưới; không commit
deploy/vps-seed.sh                                          # seed bằng API thật + cấu hình LLM (xem "LLM")
```
`vps-seed.sh` nới giới hạn theo IP chỉ trong lúc seed (seed là MỘT IP gửi hàng nghìn yêu cầu) rồi gỡ và `up -d gateway`.

## Tường lửa của host (quan trọng)
Host có hardening cũ trong `/etc/iptables/rules.v4` (+ `rules.v6`): chuỗi `DOCKER-USER` có `-i eth0 -m conntrack --ctstate NEW -j DROP` chặn mọi kết nối mới tới cổng container công bố. Đã viết lại tối giản: `DOCKER-USER` chỉ cho `-i eth0 -m conntrack --ctorigdstport 80/443` (đúng hai cổng Caddy), còn lại vẫn DROP; SSH (INPUT) không đổi. Bản cũ: `/root/rules.v4.before-80-443`, `/root/rules.v6.before-80-443`. Muốn mở thêm cổng ra Internet thì thêm luật `--ctorigdstport` tương ứng; không publish cổng ra `0.0.0.0` ở compose khi chưa cần.
**KHÔNG chạy `netfilter-persistent reload` (hay `iptables-restore` bản cũ) khi Docker đang chạy** — nó flush bảng `filter` và xoá các chuỗi `DOCKER*` do Docker dựng: container mất mạng ra / vào cho tới khi `systemctl restart docker`. Sửa luật bằng `iptables -I/-D DOCKER-USER …` rồi `netfilter-persistent save`.

## LLM
Cấu hình bằng API quản trị (`deploy/vps-llm.mjs`, ghi vào DB, khoá lưu mã hoá AES-GCM), hai pha:
- **Sinh chữ (CHAT, CLASSIFY, UTILITY, GRADING, QUESTION_GEN, INSIGHT):** 9router (`http://9router:20128/v1`) qua loại `openai_compatible`, model `ag/gemini-3.8-flash`, RPM 20. 9router bỏ qua `json_schema` nên gateway dùng `json_object` + schema trong lời nhắc (Go vẫn kiểm schema). **9router trả SSE nếu thân không nêu `stream`** → `openai_compatible` gửi `stream:false` tường minh (đã sửa ở `internal/llm/provider`). Khoá 9router đọc thẳng từ sqlite của 9router bởi `vps-llm.mjs`, gửi cho gateway; không nằm trong `.env.local`, không in ra. KHÔNG đụng container / thư mục / cron của 9router.
- **EMBEDDING:** Gemini THẬT — nhà cung cấp loại `gemini` (`https://generativelanguage.googleapis.com/v1beta/openai/`, xác thực `Authorization: Bearer`), model `gemini-embedding-001`, gateway ép `dimensions=1536` (khớp cột `vector(1536)`, không đổi migration) và chuẩn hoá L2. Khoá `GEMINI_API_KEY` ở `.env.local` (600). 9router không có embeddings; embedding giả (hash ngẫu nhiên) làm mọi truy vấn RAG có cosine ≈ 0 < `RagSimFloor` nên chat luôn "chưa tìm thấy nội dung" — không dùng cho triển khai thật.
- **Pha `seed`:** seed trích quy chế bằng bản ghi demo của provider `fake` nên lúc seed tuyến sinh chữ trỏ `chat-gia` (fake) còn EMBEDDING đã là Gemini (tài liệu, thread seed được nhúng thật ngay từ đầu, không phải nạp lại); xong seed chuyển pha `live`. Đổi embedding giữa chừng thì phải lập chỉ mục lại (`POST /courses/{id}/documents/reindex`; thread không có đường nạp lại → seed lại).
- Đổi model chat: `MODEL=ag/gemini-3.7-flash PHASE=live node deploy/vps-llm.mjs` (tạo nhà cung cấp nếu chưa có model đó; nếu `9router` đã có thì thêm model ở `/settings/llm`).

## Cập nhật / vận hành
```bash
cd /opt/edupilot && git pull && deploy/dc up -d --build --wait
deploy/dc ps ; deploy/dc logs -f gateway worker ; docker stats --no-stream
ssh -L 8025:127.0.0.1:8025 -L 9001:127.0.0.1:9001 root@<vps>   # mailpit, MinIO console
```
Sao lưu: volume `edupilot_postgres_data` (và `minio_data`); chưa có lịch sao lưu tự động (thuộc phase PR).
