# Handoff — OPS-VPS deploy (dev)

Thu hẹp theo PM / chủ dự án (2026-11-02): chỉ cần **seed chạy qua + chat dùng được**; các phép đo (judge, docker stats có hệ thống, k6, Lighthouse) để sprint riêng "seed + yêu cầu phi chức năng". Dưới đây là kết quả THẬT đã chạy.

## Dựng / vận hành
Xem `deploy/README.md` (mã ở `/opt/edupilot`, nhánh `ops/vps-deploy`; build / test của dev ở `/srv/work/dev`). Lệnh: `deploy/dc up -d --build --wait`, `deploy/vps-seed.sh`, `deploy/vps-llm.mjs`. Bí mật chỉ ở `/opt/edupilot/.env.local` (600), sinh ngẫu nhiên bởi `deploy/vps-env.sh`; khoá 9router đọc thẳng từ sqlite của 9router vào DB (mã hoá), không in; `GEMINI_API_KEY` do chủ dự án cấp, ở `.env.local`. 9router không bị đụng.

## Đã đổi (repo)
- `deploy/vps.compose.yml`, `deploy/caddy/Caddyfile.vps`, `deploy/vps-env.sh`, `deploy/vps-seed.sh`, `deploy/vps-llm.mjs`, `deploy/dc`, `deploy/README.md`.
- Mã: `BLOB_PUBLIC_USE_SSL` (URL ký sẵn MinIO https sau Caddy khi endpoint nội bộ http; test `TestPresign_PublicSchemeIndependent`); `openai_compatible`: gửi `stream:false` (9router trả SSE nếu thiếu → BAD_RESPONSE) và gộp chỉ dẫn schema vào tin system đầu (Gemini sau 9router chỉ theo MỘT tin system, bỏ qua schema ở tin thứ hai → đầu ra sai schema); test `provider_test.go` (stream:false, một tin system, `TestCompatSchemaMergedIntoCallerSystem`). `go vet`, `go test ./internal/llm/... ./internal/grading/... ./internal/platform/{blob,config}` xanh trên VPS (`/srv/work/dev`, golang:1.27 + testcontainers).

## Kết quả
| Việc | Kết quả thật |
| --- | --- |
| HTTPS công khai | `curl -I https://v2202609419695521098.happysrv.de` → 200, chứng chỉ ZeroSSL hợp lệ. Let's Encrypt báo 429 (50 chứng chỉ / tuần cho miền dùng chung `happysrv.de`, hết hạn 19:17 UTC 01/11) → Caddy tự dùng ZeroSSL khi có `email` (đã đặt). Cổng 80/443 do PM mở ở `DOCKER-USER` (`deploy/README.md`) |
| Seed | `deploy/vps-seed.sh` rc 0 trên DB trống (~12 phút gồm docling). Lần đầu gãy vì chạy qua nửa đêm giờ Việt Nam (bước 16 đòi buổi 10 = HÔM NAY) — chạy lại từ DB trống thì qua. Giới hạn IP nới chỉ lúc seed (đăng ký 5 / giờ / IP) |
| Chat riêng (bước 3) | Trả lời bằng Gemini thật qua 9router, **có trích dẫn tài liệu** (`citations` + nhãn `[2]`), tên / MSSV được che rồi khôi phục (`llm_audit.pii_masked_count` 1–3; CHAT `9router / ag/gemini-3.8-flash`). Hỏi về "Sâu Morris" (bài giảng seed) ra đúng nội dung bài giảng; câu ngoài tài liệu được từ chối lịch sự |
| Embedding | **Gemini THẬT**: nhà cung cấp loại `gemini`, `gemini-embedding-001`, `dimensions=1536` (khớp `vector(1536)`, không đổi migration), chuẩn hoá L2. Seed chạy với EMBEDDING = Gemini từ đầu nên tài liệu + thread seed đều được nhúng thật (không phải nạp lại). Lý do cần: embedding giả (sha256 ngẫu nhiên) → cosine ≈ 0 < `RagSimFloor` 0,25 → chat luôn "chưa tìm thấy nội dung" |
| Trích công thức (bước 4) | Quy chế lớp 2 → Gemini thật: job SUCCEEDED trong 16 s, bản nháp 3 thành phần (25 / 25 / 50), luật BONUS / PENALTY…, 3 mục chưa rõ (ROUNDING_MISSING…). Trước khi sửa schema-gộp: BAD_RESPONSE 3 lần |
| Upload qua URL ký sẵn | PUT PDF tới `https://<host>/edupilot/…` 200; docling đọc xong, tài liệu READY |
| Chấm BT03 (bước 5, ngoài phạm vi thu hẹp nhưng đã chạy một lần) | `grade-all` 22 bài với Gemini thật: job SUCCEEDED 218 s, 22 DRAFT, 0 lỗi, 4 bài gắn cờ (NEEDS_REVIEW, INJECTION_SUSPECTED, OFF_TOPIC, EMPTY_SUBMISSION). **Đo chất lượng so `expected_grading.json` chưa làm** (việc của Tech Lead). Hệ quả: BT03 trên stack demo đã có 22 nháp, không còn "chưa chấm" như seed — muốn demo lại `Chấm tất cả` thì seed lại |
| Judge (bước 6), docker stats có hệ thống (bước 7) | KHÔNG làm (thu hẹp). Có hai lần `docker stats`: rảnh — worker 17 MiB, gateway 19 MiB, postgres 125 MiB, docling 1,05 GiB, minio 96 MiB; lúc chấm — tương tự (worker 4,4 % CPU). Judge chạy `healthy` trên amd64 không `-no-seccomp` nhưng chưa chấm bài C++ nào |

## Giới hạn / nợ
- 9router là tài khoản OAuth miễn phí: RPM 20 / nhà cung cấp; latency mỗi lời gọi sinh chữ 4–15 s (model suy luận); `response_format: json_schema` bị bỏ qua nên cấu trúc dựa vào schema trong lời nhắc + Go kiểm lại (lỗi → BAD_RESPONSE → thử lại / dự phòng).
- Embedding Gemini cần khoá của chủ dự án; đổi embedding giữa chừng phải lập chỉ mục lại tài liệu (`documents/reindex`) và thread không có đường nạp lại (→ seed lại).
- Seed ĐÒI chạy trong cùng một ngày (giờ Việt Nam) — nợ của seed sprint 7/8 (bước 16), không sửa ở đây.
- Chưa có lịch sao lưu (phase PR); `docker-compose.prod.yml` vẫn chưa có.
- Chưa đo: judge amd64 seccomp, k6, Lighthouse, 1.000 SV, chất lượng AI so tay — thuộc sprint "seed + phi chức năng".
