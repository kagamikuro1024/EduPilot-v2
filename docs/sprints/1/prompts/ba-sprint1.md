# Prompt cho `ba` — sprint 1

1. Đọc và nhận vai theo `docs/team/BA.md`. Mẫu bắt buộc: `docs/team/TEMPLATES.md`.
2. Kế hoạch sprint đã duyệt: `docs/sprints/1/plan.md` (đọc kỹ mục "Quyết định PM tự chốt" và "Trả lời của chủ dự án"). Nhánh: `sprint/1-p0-prep` — làm trên nhánh này, không đổi nhánh.
3. Backlog nguồn: `docs/phases/P0.md` lát L0, L1, L2.

## Việc

| Feature | Story | Sản phẩm |
| --- | --- | --- |
| FEAT-demo-script | US-P0-01 | `docs/specs/FEAT-demo-script/{US,QUESTIONS}.md` **và** `docs/DEMO_SCRIPT.md` (ngoại lệ vùng file được chủ dự án duyệt cho sprint này) |
| FEAT-ci-mailhog | US-P0-02 | `docs/specs/FEAT-ci-mailhog/{US,SRS,QUESTIONS}.md` |
| FEAT-api-contract | US-P0-03 | `docs/specs/FEAT-api-contract/{US,SRS,QUESTIONS}.md` |

- US-P0-02 và US-P0-03 là hạ tầng, không có màn hình: SRS rút gọn gồm mục 1, 4, 6, 8, 9, 11 (các mục khác ghi "Không áp dụng — story hạ tầng"). AC "phân quyền" của BA.md: với US-P0-03 là AC mỗi endpoint có bảo vệ phải ghi 401/403 và vai trò được phép; với US-P0-02 ghi "không áp dụng" kèm lý do.
- AC phải kiểm được bằng lệnh. Ghi sẵn lệnh kiểm (ví dụ `docker compose ... ps`, `curl -s localhost:8025`, validator OpenAPI chạy bằng `npx`, `gh run list`).
- `DEMO_SCRIPT.md`: theo P0 L0 — 15 phút, F2 → F3 → F5 → F7 → F9 → F10 → F17, 2 lớp seed; mỗi bước: vai trò, route (bám `docs/design/DESIGN.md` §14), thao tác, kết quả phải thấy, phase làm cho bước đó chạy thật, mốc thời gian; mục dự phòng `DEMO_MODE=true` + provider `fake` + video quay sẵn. Lời thoại phía sinh viên không dùng từ kỹ thuật AI (DESIGN §13). Không bịa màn hình không có trong DESIGN/PRD; thiếu thì ghi QUESTIONS.

## Sự thật về repo (PM đã kiểm, dùng thay cho chữ trong P0.md/CLAUDE.md)
- Gateway Java: `backend-java/aitrogiang/`, build bằng Gradle wrapper (`./gradlew test`), không có Maven.
- Frontend khoá phụ thuộc bằng npm (`frontend/package-lock.json`); CI dùng `npm ci`, `npm run lint`, `npm run build`.
- Controller Java: 13 file không phải test (Admin, Assignment, Auth, Chat, ChatFeedback, ChatStream, ChatStreaming, Health, Intent, Internal, Message, Thread, User). P0.md ghi 16 — spec dùng con số thật, yêu cầu dev xác nhận.
- Frontend gọi gateway qua `frontend/src/services/*.ts` (javaClient, auth, chat, threads, documents, analytics, intent, aiClient).
- Compose local: `docker-compose.local.yml`, service hiện có: postgres-db, redis-cache, jaeger, python-ai, java-backend, frontend.

## Khi xong
Commit `sprint 1: spec FEAT-…` (chỉ file trong vùng của bạn), rồi tóm tắt ≤ 10 dòng: feature nào xong, số US, số câu hỏi mở. Dừng và chờ.
