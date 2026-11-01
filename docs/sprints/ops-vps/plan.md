# OPS-VPS: triển khai EduPilot lên VPS + đánh giá LLM thật

**Chủ dự án yêu cầu 2026-11-01:** deploy cả hệ thống lên VPS. LLM thật lấy qua 9router có sẵn trên VPS, model mạnh nhất `ag/gemini-3.8-flash`. Đo thật những phần báo cáo / luận văn đang ghi "chưa test LLM thật", rồi báo cáo. Nhánh `ops/vps-deploy` tách từ `main` (`8808b1c9`).

## Máy
- netcup VPS 4000 G12, Vienna: 12 vCPU amd64, 32 GB RAM, 1 TB, Debian 13, Docker + compose plugin có sẵn.
- `ssh root@152.53.64.175` (chỉ khoá; khoá của máy Mac đã có). Hostname công khai `v2202609419695521098.happysrv.de` trỏ đúng IP, dùng cho HTTPS (Caddy tự cấp chứng chỉ).
- **Không đụng** container `9router`, thư mục `/root/9router`, network `9router_default`, cron `tunnel-keepalive.sh`.

## 9router (đã kiểm 2026-11-01)
- API tương thích OpenAI tại `http://127.0.0.1:20128/v1`, chỉ nghe ở loopback của host. Từ container: gắn service vào network ngoài `9router_default` rồi gọi `http://9router:20128/v1`.
- Khoá: `sqlite3 -readonly /root/9router/data/db/data.sqlite "select key from apiKeys where isActive=1 limit 1;"`. **Chỉ đọc trong shell VPS vào tệp env trên VPS; không in ra, không commit, không dán vào chat / tài liệu.**
- Model chat: `ag/gemini-3.8-flash` (mạnh nhất; có `-high/-medium/-low`), `ag/gemini-3.7-flash*`, `ag/gemini-3.1-pro-low`, `ag/claude-sonnet-4-6`, `ag/gpt-oss-120b-medium`…
- `stream:false` chạy được. **`response_format: json_schema` bị bỏ qua** (trả chữ tự do). Vì vậy dùng loại provider `openai_compatible` của `internal/llm`: `json_object` + schema trong lời nhắc, Go vẫn kiểm schema.
- **Không có embeddings** (lỗi `does not support embeddings`). Embedding giữ provider hiện có (giả / tất định); tìm lai vẫn có nhánh `tsv`. Ghi giới hạn, không thêm service mới (D46).
- Tài khoản phía sau là OAuth miễn phí, nên có hạn mức. Đặt RPM vừa phải; đo theo lô nhỏ; lỗi 429 thì lùi, không dội.

## Việc
1. **dev:** `prompts/dev-deploy.md`: deploy + smoke + handoff.
2. **research (Tech Lead):** `prompts/tl-eval.md`: chuẩn bị bộ đo song song, chạy khi dev báo stack lên; báo cáo `docs/research/2026-11-01-real-llm-eval.md`.
3. **qc:** smoke hộp đen trên URL công khai khi dev xong (`prompts/qc-smoke.md`).
4. PM tổng hợp, báo chủ dự án; lỗi thật → dev sửa trên `ops/vps-deploy`, merge `main` khi CI xanh.
