# Prompt cho `ba` — Spec sprint 2 (PG Nền Go)

**Làm trong worktree `/Users/kuro/Documents/TA_Agent_v2-s2` (nhánh `sprint/2-pg`)**, không phải thư mục repo chính (repo chính đang ở nhánh 1.5). Mọi lệnh git chạy với `git -C /Users/kuro/Documents/TA_Agent_v2-s2`.

## Đọc
`docs/team/CONTEXT.md`, `docs/team/BA.md`, `docs/team/TEMPLATES.md`, `docs/sprints/2/plan.md`, `docs/phases/PG.md` (nguồn chính), `docs/ARCHITECTURE.md` mục 2, 3, 5, `docs/SYSTEM_DESIGN.md` mục 2, 3.2, 3.3, 5, `docs/DECISIONS.md` D22, D45–D48, D52, luật 10–15 trong `AGENTS.md`. Mã hiện có: `backend-go/`, `docker-compose.local.yml`, `.github/workflows/ci.yml`.

## Việc
Viết `docs/specs/FEAT-pg-foundation/{US.md, SRS.md, QUESTIONS.md}` theo `TEMPLATES.md`, 7 story US-PG-01..07 đúng bảng trong `plan.md`:
- Mỗi checkbox của `PG.md` thành ít nhất một AC **kiểm được bằng lệnh hoặc test** (ghi lệnh, mã HTTP, tên header, khoá Redis, tên bảng/cột, ngưỡng số).
- Nhánh lỗi bắt buộc: thiếu env → chết sớm kèm tên biến; Idempotency-Key gửi đôi → một bản ghi, response y hệt; cursor không sót/không lặp khi có bản ghi chen; sai `version` → 409 kèm giá trị hiện tại; quá 2 kết nối SSE → 429; Last-Event-ID nối lại không mất/không trùng; sự kiện phát từ gateway B tới client nối gateway A; tắt một gateway giữa stream; outbox retry 3 lần rồi dead-letter; JWT hết hạn/sai chữ ký → 401 đúng định dạng lỗi; sai role → 403.
- SRS: schema `00001` đầy đủ cột + kiểu + index (users dạng cuối theo `ARCHITECTURE.md`), bảng mã lỗi, định dạng cursor (mã hoá gì), khoá Redis (tiền tố, TTL), danh sách env (tên, bắt buộc?, mặc định dev), cấu hình Caddy/PgBouncer, SLO k6 smoke.
- Không có endpoint nghiệp vụ. Endpoint được phép: `/healthz`, `/readyz` (nếu cần), `/api/v1/jobs/{id}`, một endpoint SSE nền tảng (ví dụ `/api/v1/events`) và endpoint **chỉ bật trong test** để kiểm helper — ghi rõ.
- Câu hỏi mở ghi `QUESTIONS.md`; chủ dự án vắng mặt nên mỗi câu kèm **phương án đề xuất mặc định** để PM chốt, không chặn.

Chỉ sửa `docs/specs/FEAT-pg-foundation/**`. Commit `sprint 2: spec FEAT-pg-foundation` đúng đường dẫn đó trên nhánh `sprint/2-pg`. Trả lời tóm tắt số AC từng story + câu hỏi mở, rồi dừng chờ PM duyệt.
