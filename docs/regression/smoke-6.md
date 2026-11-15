# Smoke hồi quy sprint 6 — kết quả
Mã đã chạy: be0c0d96 (sprint/11-p10-eval) · Stack ep-r · Người chạy: qc2 (nhóm E) · Ngày: 2026-11-15
Tổng TC cũ: 536 · Đạt: 296 · Trượt: 3 · Không chạy K (a/b/c): 1 · Lỗi thời (T, tính vào cột K của brief): 6 · Chưa chạy (mẫu) N: 230   (P+F+K+T+N = 536 = 536)

Quy ước: P = đạt; F = trượt (chạy lại 1 lần vẫn trượt); K = không chạy vì (a) cần thiết bị không có, (b) cần người thật, (c) ngoài phạm vi đồ án; T = TC lỗi thời do spec đổi hợp lệ (ghi proposal); N = chưa chạy, ngoài mẫu smoke «đường chính + ≥ 1 nhánh lỗi/phân quyền cho mỗi US» (PM quyết có chạy nốt hay không).

Cách chạy:
- Go: `go test [-tags integration] <gói> -run …` đúng lệnh trong tc-*.md cho 49 ca G (store, privacy, llm, agent, chat, thread, integration, contract `-tags testroutes`) — chạy trong `backend-go` (testcontainers), log `/tmp/qc2e/gorun6.log`.
- API (python/urllib, kèm lib ở `/tmp/qc2e`): `s6_p301*.py` (ràng buộc DB trong giao dịch ROLLBACK), `s6_p302.py` (precheck PII), `s6_chat*.py` (chat riêng, SSE, idempotency, phân quyền), `s6_thr*.py` (Threads + AI trả lời + xác nhận/sửa/loại), `s6_p8*.py` (presign/PUT/complete DOCX, ingest docling, thư viện, chunk, phân quyền), `s6_cal.py` (lịch/ICS), `s6_misc.py`, `s6_today.py`, `s6_reg.py` (P-106-11, P-11-16).
- Script cũ: `p602-attacks.py` (--set mssv,email,phone,cccd / negatives / redact, gw=https://localhost:1143), `benchmarks/eval_pii.py` (E1: recall 0,960, false_block 0,000), `scan-pii-leak.sh` (COMPOSE shim cho ep-r; 5 bảng ERR do lỗi tách `::` của script → quét lại toàn bảng bằng Python: PII chỉ nằm ở bảng được phép).
- Dữ liệu: lớp riêng E6C1/E6C2 (GV `teacher`, TA `ta`, SV sv07–sv09, sv20, sv.gioi, sv.kha) — không sửa seed 761987/761988 (chỉ thêm/xoá mềm vài thread `[qc-e6]` ở 761987 đã dọn). Mẹo RAG với embedding giả: nhúng tài liệu DOCX có đoạn trùng đúng `tiêu đề\nnội dung` thread/tin chat → truy xuất cosine = 1, AI trả lời thật qua 9router (tuần tự).
- UI: `docs/regression/evidence-6/scripts/ui-smoke.mjs` (Playwright, https://localhost:1143) — 1440 px: /chat, /threads, /library, /calendar (SV), /documents, /calendar, /threads (GV) tải được, 0 tràn ngang, 0 lỗi 5xx; 375 px chạy lại sau khi xoá `ep:rl:auth*`: cả 7 trang đạt (0 tràn ngang).
- Môi trường: Postgres ep-r bị crash-recovery ~5 phút giữa phiên (gateway 500 `conn closed`), đã chờ phục hồi, dữ liệu nguyên vẹn; không tính vào kết quả.
- Bổ sung: D60 — `GET /admin/llm/decision`: cả 3 cổng (CHAT_ROUTING, INJECTION_FLAG, THREADS_PRECHECK) `enabled=false`, `courses_enabled=0/27`; sau hàng trăm tin chat/precheck ở lớp E6C1, `llm_audit` task='DECIDE' của lớp E6* = 0 ⇒ mặc định TẮT đúng. (`/admin/llm/decision/courses` trả 404 — route thật là `/api/v1/courses/{id}/decision-gates`.)
- Postgres ep-r crash-recovery: 12:59:54 UTC «terminating any other active server processes» → 13:00:10 «not properly shut down; automatic recovery» → 13:02:01 ready (~7 phút gateway 500). Container không OOMKilled, RestartCount 0; nghi một tiến trình Postgres bị giết do áp lực RAM trên VM Colima 9,69 GiB dùng chung 8 nhóm (load average 17–23). Log: `evidence-7/postgres-crash-recovery.log`.

## Tổng hợp theo US
| US | TC tổng | P | F | K | T | N | Ghi chú |
|---|---|---|---|---|---|---|---|
| US-P3-01 | 51 | 46 | 1 | 0 | 1 | 3 |  |
| US-P3-02 | 61 | 34 | 1 | 0 | 0 | 26 |  |
| US-P3-03 | 36 | 18 | 0 | 0 | 1 | 17 |  |
| US-P3-04 | 55 | 27 | 0 | 0 | 1 | 27 |  |
| US-P3-05 | 59 | 32 | 0 | 0 | 0 | 27 |  |
| US-P3-06 | 49 | 37 | 1 | 0 | 0 | 11 |  |
| US-P3-07 | 25 | 16 | 0 | 0 | 0 | 9 |  |
| US-P3-08 | 29 | 12 | 0 | 0 | 1 | 16 |  |
| US-P8-01 | 62 | 19 | 0 | 0 | 0 | 43 |  |
| US-P8-02 | 49 | 16 | 0 | 0 | 0 | 33 |  |
| US-P8-03 | 60 | 39 | 0 | 1 | 2 | 18 |  |

## Kết quả từng ca
(Mỗi ca P/F/K/T một dòng. Ca N liền nhau cùng lý do được gom một dòng.)
| TC | KQ | Ghi chú |
|---|---|---|
| TC-P3-01-01 | P | goose up trên DB trống tới version 25 (gồm 00007, 00008), không Pending |
| TC-P3-01-02 | P | goose up→down(về 0)→up trên DB tạm qc2e_tmp (thư viện goose v3.28.0 qua chương trình ngoài repo; CLI goose không build được với go.sum), version 25, pg_dump --schema-only trước/sau giống hệt (diff rỗn |
| TC-P3-01-03 | P | db/migrations/00007_chat_threads.sql:0 db/migrations/00008_privacy.sql:0 |
| TC-P3-01-04 | P | git diff origin/main -- migrations/0000[1-6]* rỗng; 6/6 cặp sha256 khớp |
| TC-P3-01-05 | T | PROGRESS.md mục "Ánh xạ migration" có 00007 chat_threads + 00008 privacy; dòng 00009 calendar đã đổi thành "00011 calendar (số 00009 bỏ trống, proposals #13)" — spec đổi hợp lệ, proposals #13 |
| TC-P3-01-06 | P | p601-schema --tables chat_sessions,chat_messages: 35 cột, 0 lệch; confidence numeric(4,3), attempt smallint, citations/blocks jsonb default [] (đối chiếu DB qcb_mig1 migrate sạch) |
| TC-P3-01-07 | P | partial_content + stream_status đều nằm trong CREATE TABLE chat_messages (grep -c = 5 ≥ 2) |
| TC-P3-01-08 | P | {STREAMING,DONE,FAILED,CANCELLED} {USER,ASSISTANT} {PRIVATE,PUBLIC} |
| TC-P3-01-09 | P | SQLSTATE 23505 |
| TC-P3-01-10 | P | SQLSTATE 23514 |
| TC-P3-01-11 | P | SQLSTATE 23514 |
| TC-P3-01-12 | P | SQLSTATE 23514 |
| TC-P3-01-13 | P | SQLSTATE 23514 |
| TC-P3-01-14 | P | updated_at tăng sau update (trigger) |
| TC-P3-01-15 | P | SQLSTATE 23514 |
| TC-P3-01-16 | P | go test ok (1 test/ca con, ~8s) |
| TC-P3-01-17 | P | forum_threads có đủ cột SRS 5.4 (embedding vector(1536), ai_skip_reason, similar_of, pinned_at, version…; hidden_at/hidden_reason có thêm ở sprint sau); verification_state nằm ở forum_posts theo SRS 5 |
| TC-P3-01-18 | P | SQLSTATE 23505 |
| TC-P3-01-19 | P | SQLSTATE 23514 |
| TC-P3-01-20 | P | SQLSTATE 23514 |
| TC-P3-01-21 | P | SQLSTATE 23514 |
| TC-P3-01-22 | P | 6 vi phạm đều 23514 |
| TC-P3-01-23 | P | go test ok (1 test/ca con, ~7s) |
| TC-P3-01-24 | P | SQLSTATE 23503 |
| TC-P3-01-25 | P | SQLSTATE 23503 |
| TC-P3-01-26 | P | SQLSTATE 23503 |
| TC-P3-01-27 | P | insert chat_messages lớp A có reply_to=tin lớp B → 23503 (chat_messages_reply_fkey (course_id, reply_to)) |
| TC-P3-01-28 | P | delete ok, document_id→NULL |
| TC-P3-01-29 | P | SQLSTATE 23502 |
| TC-P3-01-30 | P | updated_at tăng |
| TC-P3-01-31 | P | go test ok (5 test/ca con, ~15s) |
| TC-P3-01-32 | P | id,course_id,session_id,user_id,channel,pii_type,count,action,created_at |
| TC-P3-01-33 | P |  |
| TC-P3-01-34 | P | ERROR:  42501: audit_log is append-only CONTEXT:  PL/pgSQL function audit_log_bl |
| TC-P3-01-35 | P |  |
| TC-P3-01-36 | P | {MSSV,EMAIL,PHONE,CCCD,NAME,PERSONAL_QUESTION,OTHER_PERSON} {BLOCKED,REDACTED,SWITCHED,MASKED} |
| TC-P3-01-37 | P | go test ok (2 test/ca con, ~11s) |
| TC-P3-01-38 | P | 18 chỉ mục không-pkey (≥11) |
| TC-P3-01-39 | P | DB tạm qcb_mig1: 10.000 dòng/bảng (2 lớp, 30 người) + ANALYZE; 4 kế hoạch đều Index Scan 0 Seq Scan: chat_sessions_user_idx 0,15 ms; chat_messages_session_idx 8,3 ms; forum_threads_course_idx 16,4 ms; |
| TC-P3-01-40 | P | go test ok (1 test/ca con, ~13s) |
| TC-P3-01-41 | P | sqlc diff rc=0; git status --porcelain internal/store rỗng (không chạy sqlc generate để khỏi ghi vào worktree) |
| TC-P3-01-42 | P | go test ok (34 test/ca con, ~30s) |
| TC-P3-01-43 | P | up lần hai: rc=0, "no migrations to run. current version: 25"; schema-only trước/sau giống nhau (diff 0 dòng) |
| TC-P3-01-44 | P | goose -dir mig-broken up: rc=1, thông điệp có 00008_privacy.sql + SQLSTATE 42P01; 0 bảng pii_events, 0 enum pii_*; status: 00008 Pending (00007 Applied) |
| TC-P3-01-45 | N | chưa chạy (mẫu): ngoài mẫu smoke 1 chính + 1 lỗi / US |
| TC-P3-01-46 | P | go test ok (2 test/ca con, ~8s) |
| TC-P3-01-47 | P | canary xuất hiện ngoài chat_messages: [] |
| TC-P3-01-48 | P | go test ok (1 test/ca con, ~13s) |
| TC-P3-01-49 | F | Vai DB ứng dụng edupilot rolsuper=t (như BUG-1 gốc); DEF-R6-01 |
| TC-P3-01-50 | N | chưa chạy (mẫu): ngoài mẫu smoke 1 chính + 1 lỗi / US |
| TC-P3-02-01 | P | 2 mẫu đúng |
| TC-P3-02-02 | P | p602 --set mssv: mã chữ-số bị chặn (14 mẫu mssv/email/phone/cccd đạt) |
| TC-P3-02-03 | P | 3 mẫu đúng |
| TC-P3-02-04 | P | 3 mẫu đúng |
| TC-P3-02-05 | P | 4 mẫu đúng |
| TC-P3-02-06 | P | 2 mẫu đúng |
| TC-P3-02-07 | P | p602-attacks.py --set negatives-number: 15 mẫu cho qua |
| TC-P3-02-08 | N | chưa chạy (mẫu): ngoài mẫu smoke 1 chính + 1 lỗi / US |
| TC-P3-02-09 | P | 1 mẫu đúng |
| TC-P3-02-10 | P | go test ok (2 test/ca con, ~3s) |
| TC-P3-02-11 | P | 1 mẫu đúng |
| TC-P3-02-12 | P | 1 mẫu đúng |
| TC-P3-02-13 | P | 1 mẫu đúng |
| TC-P3-02-14 | P | 2 mẫu đúng |
| TC-P3-02-15 | P | 1 mẫu đúng |
| TC-P3-02-16 | P | p602-attacks.py --set negatives-name: cho qua đúng |
| TC-P3-02-17 | N | chưa chạy (mẫu): ngoài mẫu smoke 1 chính + 1 lỗi / US |
| TC-P3-02-18 | P | go test ok (3 test/ca con, ~2s) |
| TC-P3-02-19 | P | 2 mẫu đúng |
| TC-P3-02-20 | P | exists=1 ttl=3597 type=string |
| TC-P3-02-21 | F | vẫn F trên ảnh ep-r hiện chạy (tạo 2026-11-15 19:38, trước fix afbd433e): vào mã/duyệt không xoá ep:roster; retest sau fix chờ Main dựng lại ảnh (script /tmp/qc2e/s6_roster2.py) — DEF-R6-02 |
| TC-P3-02-22 | P | mời ra → cache xoá 0,5 s, tên không còn bị chặn |
| TC-P3-02-23 | N | chưa chạy (mẫu): ngoài mẫu smoke 1 chính + 1 lỗi / US |
| TC-P3-02-24 | P | 2 mẫu đúng |
| TC-P3-02-25 | P | go test ok (3 test/ca con, ~37s) |
| TC-P3-02-26 | P | [đã ẩn] [đã ẩn] [đã ẩn] [đã ẩn] |
| TC-P3-02-27 | P | redact idempotent |
| TC-P3-02-28 | P | p602-attacks.py --set redact: idempotent + văn bản sạch giữ nguyên |
| TC-P3-02-29 | P | go test ok (2 test/ca con, ~5s) |
| TC-P3-02-30 … TC-P3-02-39 (10 ca) | N | chưa chạy (mẫu): ngoài mẫu smoke 1 chính + 1 lỗi / US |
| TC-P3-02-40 | P | go test ok (3 test/ca con, ~3s) |
| TC-P3-02-41 … TC-P3-02-47 (7 ca) | N | chưa chạy (mẫu): ngoài mẫu smoke 1 chính + 1 lỗi / US |
| TC-P3-02-48 | P | go test ok (4 test/ca con, ~2s) |
| TC-P3-02-49 … TC-P3-02-50 (2 ca) | N | chưa chạy (mẫu): ngoài mẫu smoke 1 chính + 1 lỗi / US |
| TC-P3-02-51 | P | go test ok (2 test/ca con, ~2s) |
| TC-P3-02-52 … TC-P3-02-53 (2 ca) | N | chưa chạy (mẫu): ngoài mẫu smoke 1 chính + 1 lỗi / US |
| TC-P3-02-54 | P | go test ok (1 test/ca con, ~2s) |
| TC-P3-02-55 | P | 30 lượt 4.000 ký tự: p95=176 ms |
| TC-P3-02-56 | P | precheck 8000 ký tự a1: 0.02s (200) |
| TC-P3-02-57 | N | chưa chạy (mẫu): ngoài mẫu smoke 1 chính + 1 lỗi / US |
| TC-P3-02-58 | P | SV ngoài lớp precheck → 403 |
| TC-P3-02-59 | P | ADMIN precheck → 403 |
| TC-P3-02-60 | N | chưa chạy (mẫu): ngoài mẫu smoke 1 chính + 1 lỗi / US |
| TC-P3-02-61 | P | TA 403, TCH 403 |
| TC-P3-03-01 | P | go test ok (1 test/ca con, ~6s) |
| TC-P3-03-02 | P | grep: lời gọi Mask/Unmask/NewStreamUnmasker chỉ ở internal/llm (mask.go), internal/privacy, dây nối internal/llm/llmrt; hit ở internal/agent/types.go và internal/grading/essay_blind.go chỉ là CHÚ THÍC |
| TC-P3-03-03 … TC-P3-03-04 (2 ca) | N | chưa chạy (mẫu): ngoài mẫu smoke 1 chính + 1 lỗi / US |
| TC-P3-03-05 | P | tin có tên+MSSV+email+SĐT: llm_request_payloads.masked_prompt chỉ còn [[SV_1]] [[MSSV_1]] [[EMAIL_1]] [[SDT_1]], 0 PII thô, pii_masked_count=8 |
| TC-P3-03-06 … TC-P3-03-09 (4 ca) | N | chưa chạy (mẫu): ngoài mẫu smoke 1 chính + 1 lỗi / US |
| TC-P3-03-10 | P | COURSE_QA có ngữ cảnh (đoạn tài liệu chứa PII) → payload che sạch; câu trả lời gốc được khôi phục cho chủ phiên |
| TC-P3-03-11 | P | go test ok (2 test/ca con, ~3s) |
| TC-P3-03-12 | N | chưa chạy (mẫu): ngoài mẫu smoke 1 chính + 1 lỗi / US |
| TC-P3-03-13 | P | llm_audit.pii_masked_count=8 khớp số thực thể |
| TC-P3-03-14 | P | go test ok (1 test/ca con, ~4s) |
| TC-P3-03-15 | P | SSE token/done không còn dạng [[SV_…]] (khôi phục đủ) |
| TC-P3-03-16 | N | chưa chạy (mẫu): ngoài mẫu smoke 1 chính + 1 lỗi / US |
| TC-P3-03-17 | P | go test ok (2 test/ca con, ~3s) |
| TC-P3-03-18 | N | chưa chạy (mẫu): ngoài mẫu smoke 1 chính + 1 lỗi / US |
| TC-P3-03-19 | P | llm_audit chứa PII/placeholder: 0 |
| TC-P3-03-20 | P | go test ok (3 test/ca con, ~19s) |
| TC-P3-03-21 | P | go test ok (1 test/ca con, ~17s) |
| TC-P3-03-22 … TC-P3-03-24 (3 ca) | N | chưa chạy (mẫu): ngoài mẫu smoke 1 chính + 1 lỗi / US |
| TC-P3-03-25 | P | go test ok (2 test/ca con, ~4s) |
| TC-P3-03-26 | P | NoMask: định nghĩa trong internal/llm/gateway.go+mask.go, llmtest/decision.go (helper test), WithNoMask chỉ ở build tag testroutes; không điểm dùng nào ở đường có nội dung người dùng (lưu ý: không có  |
| TC-P3-03-27 … TC-P3-03-28 (2 ca) | N | chưa chạy (mẫu): ngoài mẫu smoke 1 chính + 1 lỗi / US |
| TC-P3-03-29 | P | go test ok (1 test/ca con, ~2s) |
| TC-P3-03-30 | P | admin: [200, 200], rò: [] |
| TC-P3-03-31 | T | LỖI THỜI: `GET /admin/llm/usage` nay cho ADMIN **và TEACHER** (openapi «ADMIN, TEACHER», đổi từ sprint 9/10); TA/SV vẫn 403 |
| TC-P3-03-32 | P | go test ok (1 test/ca con, ~15s) |
| TC-P3-03-33 … TC-P3-03-35 (3 ca) | N | chưa chạy (mẫu): ngoài mẫu smoke 1 chính + 1 lỗi / US |
| TC-P3-03-36 | P | ADMIN vào route chat của SV → [403, 403, 403] |
| TC-P3-04-01 | N | chưa chạy (mẫu): ngoài mẫu smoke 1 chính + 1 lỗi / US |
| TC-P3-04-02 | P | go test ok (1 test/ca con, ~6s) |
| TC-P3-04-03 | P | intent=PERSONAL_GRADE embed+=0 |
| TC-P3-04-04 … TC-P3-04-06 (3 ca) | N | chưa chạy (mẫu): ngoài mẫu smoke 1 chính + 1 lỗi / US |
| TC-P3-04-07 | P | go test ok (3 test/ca con, ~3s) |
| TC-P3-04-08 … TC-P3-04-10 (3 ca) | N | chưa chạy (mẫu): ngoài mẫu smoke 1 chính + 1 lỗi / US |
| TC-P3-04-11 | P | go test ok (1 test/ca con, ~2s) |
| TC-P3-04-12 … TC-P3-04-13 (2 ca) | N | chưa chạy (mẫu): ngoài mẫu smoke 1 chính + 1 lỗi / US |
| TC-P3-04-14 | P | go test ok (1 test/ca con, ~3s) |
| TC-P3-04-15 | P | go test ok (1 test/ca con, ~2s) |
| TC-P3-04-16 … TC-P3-04-17 (2 ca) | N | chưa chạy (mẫu): ngoài mẫu smoke 1 chính + 1 lỗi / US |
| TC-P3-04-18 | P | POST thêm user_id → 422 |
| TC-P3-04-19 | P | go test ok (2 test/ca con, ~10s) |
| TC-P3-04-20 | P | go test ok (2 test/ca con, ~4s) |
| TC-P3-04-21 … TC-P3-04-23 (3 ca) | N | chưa chạy (mẫu): ngoài mẫu smoke 1 chính + 1 lỗi / US |
| TC-P3-04-24 | P | intents=['OTHER_PERSON', 'OTHER_PERSON', 'OTHER_PERSON', 'OTHER_PERSON'] llm+=0 |
| TC-P3-04-25 | P | pii_events +4 |
| TC-P3-04-26 | N | chưa chạy (mẫu): ngoài mẫu smoke 1 chính + 1 lỗi / US |
| TC-P3-04-27 | P | intent=PERSONAL_ATTENDANCE Hệ thống chưa có dữ liệu điểm danh của bạn. |
| TC-P3-04-28 | N | chưa chạy (mẫu): ngoài mẫu smoke 1 chính + 1 lỗi / US |
| TC-P3-04-29 | P | go test ok (3 test/ca con, ~2s) |
| TC-P3-04-30 | N | chưa chạy (mẫu): ngoài mẫu smoke 1 chính + 1 lỗi / US |
| TC-P3-04-31 | P | MSSV của SVB → OTHER_PERSON, câu từ chối, không lộ dữ liệu (kỳ vọng AC9) |
| TC-P3-04-32 | P | go test ok (1 test/ca con, ~3s) |
| TC-P3-04-33 | P | ['PERSONAL_ATTENDANCE', 'PERSONAL_PARTICIPATION', 'PERSONAL_GRADE', 'GRADE_FORMULA'] |
| TC-P3-04-34 | N | chưa chạy (mẫu): ngoài mẫu smoke 1 chính + 1 lỗi / US |
| TC-P3-04-35 | P | go test ok (2 test/ca con, ~2s) |
| TC-P3-04-36 … TC-P3-04-37 (2 ca) | N | chưa chạy (mẫu): ngoài mẫu smoke 1 chính + 1 lỗi / US |
| TC-P3-04-38 | P | go test ok (1 test/ca con, ~3s) |
| TC-P3-04-39 | P | ep:ans 4→4 |
| TC-P3-04-40 … TC-P3-04-43 (4 ca) | N | chưa chạy (mẫu): ngoài mẫu smoke 1 chính + 1 lỗi / US |
| TC-P3-04-44 | P | go test ok (3 test/ca con, ~3s) |
| TC-P3-04-45 | T | LỖI THỜI (US-P4-07 2026-10-14: khủng hoảng hai tầng, câu mẫu đổi, "tu tu" không dấu không còn là khoá); kiểm lại ở sprint 7 |
| TC-P3-04-46 | P | lần đầu lệch do nhóm khác ghi chung (đếm toàn cục); chạy lại theo lớp E6C1: tickets/notifications/pii_events không đổi, 0 lời gọi LLM, intent=CRISIS lưu (xem TC-P4-07-09) |
| TC-P3-04-47 | P | go test ok (2 test/ca con, ~2s) |
| TC-P3-04-48 … TC-P3-04-49 (2 ca) | N | chưa chạy (mẫu): ngoài mẫu smoke 1 chính + 1 lỗi / US |
| TC-P3-04-50 | P | Mình chưa tìm thấy nội dung này trong tài liệu của lớp. |
| TC-P3-04-51 | P | go test ok (3 test/ca con, ~3s) |
| TC-P3-04-52 | P | TA/TCH/ADM tạo phiên → 403/403/403 |
| TC-P3-04-53 | N | chưa chạy (mẫu): ngoài mẫu smoke 1 chính + 1 lỗi / US |
| TC-P3-04-54 | P | SVB gửi vào phiên SVA → 404 |
| TC-P3-04-55 | P | SV ngoài lớp tạo phiên → 403 |
| TC-P3-05-01 | P | go test ok (2 test/ca con, ~22s) |
| TC-P3-05-02 | N | chưa chạy (mẫu): ngoài mẫu smoke 1 chính + 1 lỗi / US |
| TC-P3-05-03 | P | status=200, khung đầu 0.054s, các khung=['done', 'status', 'token'] |
| TC-P3-05-04 | P | chuỗi khung: ['status', 'status', 'token']…done, 4 khung |
| TC-P3-05-05 | N | chưa chạy (mẫu): ngoài mẫu smoke 1 chính + 1 lỗi / US |
| TC-P3-05-06 | P | go test ok (1 test/ca con, ~13s) |
| TC-P3-05-07 | N | chưa chạy (mẫu): ngoài mẫu smoke 1 chính + 1 lỗi / US |
| TC-P3-05-08 | P | go test ok (3 test/ca con, ~14s) |
| TC-P3-05-09 | N | chưa chạy (mẫu): ngoài mẫu smoke 1 chính + 1 lỗi / US |
| TC-P3-05-10 | P | GET stream sau done → 200, 4 khung |
| TC-P3-05-11 | N | chưa chạy (mẫu): ngoài mẫu smoke 1 chính + 1 lỗi / US |
| TC-P3-05-12 | P | go test ok (1 test/ca con, ~12s) |
| TC-P3-05-13 … TC-P3-05-14 (2 ca) | N | chưa chạy (mẫu): ngoài mẫu smoke 1 chính + 1 lỗi / US |
| TC-P3-05-15 | P | cancel 2 lần → 204,204 |
| TC-P3-05-16 | P | SV khác cancel → 404 |
| TC-P3-05-17 … TC-P3-05-21 (5 ca) | N | chưa chạy (mẫu): ngoài mẫu smoke 1 chính + 1 lỗi / US |
| TC-P3-05-22 | P | 409 EXAM_IN_PROGRESS details={'until': '2026-11-15T17:08:41Z'}; chat_messages mới=0; fake_calls 0→0; llm_audit 0→0; exam_events CHAT_BLOCKED/{} |
| TC-P3-05-23 | P | 5 lần gửi → [409, 409, 409, 409, 409]; CHAT_BLOCKED tổng=1 (≤1/phút); TTL ep:chat_blocked=55; exam_events loại khác 0→0 (không trừ hạn mức sự kiện máy khách) |
| TC-P3-05-24 | P | retry → 409 EXAM_IN_PROGRESS; tạo phiên có document_id → 201; gửi trong phiên đó → 409 EXAM_IN_PROGRESS |
| TC-P3-05-25 | P | nộp bài → gửi lại được sau 0.3s (≤10 s); SSE 200, 3 khung |
| TC-P3-05-26 | N | chưa chạy (mẫu): ngoài mẫu smoke 1 chính + 1 lỗi / US |
| TC-P3-05-27 | P | /chat?course=… khi khoá: "Chat tạm khóa trong lúc bạn làm bài thi. Dùng lại được sau 00:08." (không nêu tên bài thi/lớp), ô soạn disabled; GET /me/exam-lock 1 lần lúc mở + 1 lần/30 s; nháp "nháp của t |
| TC-P3-05-28 | P | 2 lần cùng key → 200/200, hàng USER=1 |
| TC-P3-05-29 | P | key khác → hàng USER=2 |
| TC-P3-05-30 | P | 10 yêu cầu song song cùng key → hàng USER=1, mã=[200] |
| TC-P3-05-31 | P | rỗng/khoảng trắng → 422/422 |
| TC-P3-05-32 | P | 4100 ký tự → 422 {'code': 'MESSAGE_TOO_LONG', 'message': 'Tin nhắn quá dài.', |
| TC-P3-05-33 | P | tin thứ 21 → 429 (kỳ vọng 21); thân=[('http', '{"code":"RATE_LIMITED","message":"Bạn thao tác quá nhanh. Hãy thử lại sau giây lát.","trace_id":"9e757362f9933a1b16cf889b16c053bc","retry_a; keys ep:rl:c |
| TC-P3-05-34 | P | (a) phiên đã xoá→404 (b) khác chủ→404 (d) không auth→401 |
| TC-P3-05-35 … TC-P3-05-38 (4 ca) | N | chưa chạy (mẫu): ngoài mẫu smoke 1 chính + 1 lỗi / US |
| TC-P3-05-39 | P | scan-pii-leak.sh + TestNoPayloadLeak ok; quét lại toàn bảng: PII chỉ ở chat_messages/users/enrollments/hộp thư mời (được phép) |
| TC-P3-05-40 … TC-P3-05-42 (3 ca) | N | chưa chạy (mẫu): ngoài mẫu smoke 1 chính + 1 lỗi / US |
| TC-P3-05-43 | P | chủ phiên 204, SV khác 404 |
| TC-P3-05-44 | N | chưa chạy (mẫu): ngoài mẫu smoke 1 chính + 1 lỗi / US |
| TC-P3-05-45 | P | limit=2→200 2 mục, cursor=True; limit 0/101→422/422 |
| TC-P3-05-46 | P | delete lặp→404, restore→200, đọc lại→200 |
| TC-P3-05-47 | P | danh sách TA/TCH/ADM [403, 403, 403], tin [403, 403, 403] |
| TC-P3-05-48 | P | SV A trên phiên SV B → [404, 404, 404, 404, 404] |
| TC-P3-05-49 | P | SV ngoài lớp: list→403, from-draft→422 |
| TC-P3-05-50 … TC-P3-05-51 (2 ca) | N | chưa chạy (mẫu): ngoài mẫu smoke 1 chính + 1 lỗi / US |
| TC-P3-05-52 | P | UI 1440: /chat «Chat riêng» tải, 0 tràn, 0 5xx |
| TC-P3-05-53 | P | UI 375: /chat «Chat riêng» tải, scrollWidth-innerWidth = 0, 0 lỗi 5xx (chạy lại sau xoá ep:rl:auth*) |
| TC-P3-05-54 | P | 4 câu mẫu: khuôn khung [('status', 1, 'done'), ('status', 1, 'done'), ('status', 1, 'done'), ('status', 1, 'done')] (status→1 token→done); intent lưu ['OTHER_PERSON', 'CRISIS', 'PERSONAL_ATTENDANCE',  |
| TC-P3-05-55 | P | trả lời='Mình chỉ xem được dữ liệu của chính bạn.'; intent=OTHER_PERSON; pii_events OTHER_PERSON/BLOCKED; MSSV người khác trong SSE: False; (kiểm payload provider: 0 lời gọi provider — intent chặn trư |
| TC-P3-05-56 … TC-P3-05-57 (2 ca) | N | chưa chạy (mẫu): ngoài mẫu smoke 1 chính + 1 lỗi / US |
| TC-P3-07-07 … TC-P3-07-05 (2 ca) | N | chưa chạy (mẫu): ngoài mẫu smoke 1 chính + 1 lỗi / US |
| TC-P3-06-01 | P | 6 trang, 16 thread duy nhất |
| TC-P3-06-02 | P | các bộ lọc → [200, 200, 200, 200, 200] |
| TC-P3-06-03 | P | SV 16 / GV 16 |
| TC-P3-06-04 | P | allowed=True, số thread không đổi |
| TC-P3-06-05 | P | [{'type': 'EMAIL', 'count': 1}, {'type': 'MSSV', 'count': 1}, {'type': 'PHONE', 'count': 1}] |
| TC-P3-06-06 | P | 61 yêu cầu liên tiếp, 429 xuất hiện ở lần 61; 200 trước đó=60 (PRECHECK_RATE_PER_MIN=60); go test TestPrecheck* ok (3 pass) |
| TC-P3-06-07 … TC-P3-06-09 (3 ca) | N | chưa chạy (mẫu): ngoài mẫu smoke 1 chính + 1 lỗi / US |
| TC-P3-06-10 | P | pii_events: không cột văn bản (xem TC-P3-01-33) |
| TC-P3-06-11 … TC-P3-06-12 (2 ca) | N | chưa chạy (mẫu): ngoài mẫu smoke 1 chính + 1 lỗi / US |
| TC-P3-06-13 | P | lần 1 422; gửi lại redact:true với khoá cũ → 422 IDEMPOTENCY_KEY_REUSED; khoá MỚI → 201; thread lưu: 'Hỏi điểm QC-C 2 / Em [đã ẩn] [đã ẩn] được mấy điểm? Mail [đã ẩn] SĐT [đã ẩn] CCCD [đã ẩn]'; pii_ev |
| TC-P3-06-14 | N | chưa chạy (mẫu): ngoài mẫu smoke 1 chính + 1 lỗi / US |
| TC-P3-06-15 | P | 422 PII_DETECTED |
| TC-P3-06-16 | P | PII ở tiêu đề→422, ở bình luận→422 |
| TC-P3-06-17 | P | 422 PII_DETECTED |
| TC-P3-06-18 | P | không dấu → 422 PII_DETECTED |
| TC-P3-06-19 | P | ai_state=ANSWERED/; số bài AI=1 |
| TC-P3-06-20 | P | 1 bài AI mỗi thread (UNIQUE) — giao lại outbox không thực hiện (không sửa outbox) |
| TC-P3-06-21 | P | thread ngoài ngữ cảnh → SKIPPED/LOW_SCORE |
| TC-P3-06-22 | N | chưa chạy (mẫu): ngoài mẫu smoke 1 chính + 1 lỗi / US |
| TC-P3-06-23 | P | SV có confidence=False; GV có=True |
| TC-P3-06-24 | P | verify 200, lặp lại 200 |
| TC-P3-06-25 | P | correct → 200 |
| TC-P3-06-26 | P | reject → 200; bài AI còn thấy với SV=False |
| TC-P3-06-27 | P | verify 2 lần → đúng 1 THREAD_VERIFIED (dedupe decided:…:VERIFIED:2) |
| TC-P3-06-28 | P | correct sai version → 409 |
| TC-P3-06-29 | P | có THREAD_ANSWERED, THREAD_VERIFIED, THREAD_REPLY cho tác giả; không báo người hành động |
| TC-P3-06-30 | P | similar → 200 2 mục |
| TC-P3-06-31 | P | SV verify → 403; TA verify → 200 |
| TC-P3-06-32 | P | SV ngoài lớp list→403, chi tiết→403 |
| TC-P3-06-33 | P | {'full_name': 'Vũ Hoàng Giang', 'role': 'STUDENT', 'is_me': False} |
| TC-P3-06-34 | N | chưa chạy (mẫu): ngoài mẫu smoke 1 chính + 1 lỗi / US |
| TC-P3-06-35 | P | 10 yêu cầu song song cùng key → 1 thread, mã [201] |
| TC-P3-06-36 | P | UI 1440: /threads «Hỏi đáp công khai» tải (SV & GV), 0 tràn, 0 5xx |
| TC-P3-06-37 | N | chưa chạy (mẫu): ngoài mẫu smoke 1 chính + 1 lỗi / US |
| TC-P3-06-38 | P | UI 375: /threads tải (SV & GV), 0 tràn ngang |
| TC-P3-06-39 | P | go test ok (1 test/ca con, ~3s) |
| TC-P3-06-40 | P | 409 EXAM_IN_PROGRESS until có; forum_threads mới=0; exam_events CHAT_BLOCKED/{}; bài chứa PII cũng 409 (khoá chạy trước tường lửa): 409 |
| TC-P3-06-41 | P | danh sách 200, chi tiết 200, precheck 200, bình luận 201, from-draft 201 |
| TC-P3-06-42 | F | /threads khi khoá: ô tiêu đề/nội dung vô hiệu, dòng báo là "Chat tạm khóa trong lúc bạn làm bài thi. Dùng lại được sau 00:08." thay vì "Đăng bài tạm khóa trong lúc bạn làm bài thi…" (SRS 472). Nộp bài |
| TC-P3-06-43 | N | chưa chạy (mẫu): ngoài mẫu smoke 1 chính + 1 lỗi / US |
| TC-P3-06-44 | P | bỏ sót []; personal_question=True reasons=[{'type': 'PERSONAL_QUESTION', 'count': 1}] |
| TC-P3-06-45 | P | 5 mẫu âm, chặn nhầm: [] |
| TC-P3-06-46 | N | chưa chạy (mẫu): ngoài mẫu smoke 1 chính + 1 lỗi / US |
| TC-P3-06-47 | P | như TC-P3-05-39 (thread/forum_posts/pii_events/outbox/audit_log sạch) |
| TC-P3-06-48 | P | ai_state=SKIPPED/LOW_SCORE; bài AI chứa canary=0 |
| TC-P3-06-49 | P | duyệt con trỏ 6 trang không trùng |
| TC-P3-07-01 | P | go test ok (3 test/ca con, ~3s) |
| TC-P3-07-02 … TC-P3-07-03 (2 ca) | N | chưa chạy (mẫu): ngoài mẫu smoke 1 chính + 1 lỗi / US |
| TC-P3-07-04 | P | tin "Tuần này em có lịch gì?" → intent UPCOMING_EVENTS, chat_messages.confidence=1.000, low_confidence=false; khối upcoming_events có 6 mục |
| TC-P3-07-06 … TC-P3-07-09 (3 ca) | N | chưa chạy (mẫu): ngoài mẫu smoke 1 chính + 1 lỗi / US |
| TC-P3-07-10 | P | GET tin chat riêng: GV/TA/ADMIN → [403, 403, 403]; go test TestStaffConfidenceThreadsOnly/TestStaffCannotReadPrivateConfidence ok (2 pass) |
| TC-P3-07-11 | P | trả lời='Mình chưa tìm thấy nội dung này trong tài liệu của lớp.'; no_context/low_confidence/intent=t/t/COURSE_QA; llm_audit 6→6; go test TestNoContextCannedReply ok |
| TC-P3-07-12 | P | eval_pii.py --validate: 200 mẫu, strata đúng |
| TC-P3-07-13 | P | số mẫu theo strata khớp EXPECTED_COUNTS (validate) |
| TC-P3-07-14 | P | 0 id trùng, 0 mẫu thiếu trường; split dev60/test140 rải đều mọi nhóm; dương→PRIVATE, âm→PUBLIC; pii_types ⊂ 6 loại cho phép |
| TC-P3-07-15 | P | soát 25 dương + 25 âm (shuf seed 7): 0 bất đồng với expect_block; toàn dữ liệu mô phỏng (ô điền roster seed, SĐT/CCCD bịa); LABELING.md không có dòng bất đồng cần BA phán |
| TC-P3-07-16 | P | E1 trên ep-r: recall 0,960 / false_block 0,000 (4 FN / 0 FP) đạt ngưỡng; ghi `e1out/e1.json` |
| TC-P3-07-17 | N | chưa chạy (mẫu): ngoài mẫu smoke 1 chính + 1 lỗi / US |
| TC-P3-07-18 | P | mã thoát 0 với ngưỡng 0,95 / 0,05 |
| TC-P3-07-19 | N | chưa chạy (mẫu): ngoài mẫu smoke 1 chính + 1 lỗi / US |
| TC-P3-07-20 | P | API không tới → mã thoát 2 |
| TC-P3-07-21 | P | token sai vai/hết hạn → mã thoát 2, từ chối |
| TC-P3-07-22 | P | dup.jsonl: "trùng id: S1-01", badlabel: "expect_block phải là bool", missingslot: "ô điền {{SV99.mssv}} không có trong roster seed" — cả ba rc=2, không PASS |
| TC-P3-07-23 | P | --validate --strict-synthetic → OK rc=0; token TEACHER → "chỉ chạy bằng token Sinh viên (vai TEACHER); từ chối", rc=2 |
| TC-P3-07-24 | N | chưa chạy (mẫu): ngoài mẫu smoke 1 chính + 1 lỗi / US |
| TC-P3-07-25 | P | chèn "Trương Quốc Bảo Anh" (ngoài roster/outside) → --strict-synthetic rc=2, nêu đúng cụm vi phạm; không strict → OK |
| TC-P3-07-26 | N | chưa chạy (mẫu): ngoài mẫu smoke 1 chính + 1 lỗi / US |
| TC-P3-07-27 | P | gate-p3.sh có bước `run "E1: eval_pii.py --min-recall 0.95 --max-false-block 0.05"` sau contract, trước Playwright; run() dừng khi FAILED=1 (đã chạy gate-lib: bước hỏng → các bước sau không chạy, GATE |
| TC-P3-08-01 … TC-P3-08-06 (6 ca) | N | chưa chạy (mẫu): ngoài mẫu smoke 1 chính + 1 lỗi / US |
| TC-P3-08-07 | P | me/today SV có continue: 3 mục |
| TC-P3-08-08 | P | SV chưa vào lớp: continue=[] |
| TC-P3-08-09 | P | continue của sv07 ⊂ phiên của sv07: 3/3 |
| TC-P3-08-10 | N | chưa chạy (mẫu): ngoài mẫu smoke 1 chính + 1 lỗi / US |
| TC-P3-08-11 | P | /courses/{id}/today: GV & TA có AI_CONFIRM (/me/today gom bị cắt 12 mục, bị GRADE_SCHEME_UNCONFIRMED lấn — không phải lỗi) |
| TC-P3-08-12 | N | chưa chạy (mẫu): ngoài mẫu smoke 1 chính + 1 lỗi / US |
| TC-P3-08-13 | P | SV/ADMIN không có AI_CONFIRM |
| TC-P3-08-14 … TC-P3-08-17 (4 ca) | N | chưa chạy (mẫu): ngoài mẫu smoke 1 chính + 1 lỗi / US |
| TC-P3-08-18 | P | mô phỏng bằng gate-lib (không có biến môi trường tắt che): bước hỏng → bước sau không chạy; "GATE P3: FAIL" rc=1 |
| TC-P3-08-19 … TC-P3-08-21 (3 ca) | N | chưa chạy (mẫu): ngoài mẫu smoke 1 chính + 1 lỗi / US |
| TC-P3-08-22 | P | `go test -tags testroutes ./internal/contract -run TestChatThreadsContract` ok (lệnh gốc thiếu `-tags testroutes` nên lần đầu 404 `_test/llm/fake`) |
| TC-P3-08-23 | P | contract/testdata/golden chỉ có auth/ và llm/ — 0 khoá cấm (confidence/retrieval_score/groundedness/ai_body/hidden_reason/verified_by); chat_threads_contract_test.go cưỡng chế phản hồi sinh viên không |
| TC-P3-08-24 | P | gate-p3.sh chạy TestChatMatrix + TestThreadsMatrix; router_test.go dựng STUDENT + TA + TEACHER + ADMIN (+PENDING, ngoài lớp) — đủ bốn vai |
| TC-P3-08-25 | P | PATH không có k6: "SKIP k6 chat.js: chưa cài", dòng cuối "GATE P3: PASS (có SKIP)", rc=0 (chạy khối k6 của gate-lib, không chạy toàn gate) |
| TC-P3-08-26 | T | DOCKER_HOST sai: strict_test → "FAIL TestNoPayloadLeak: thiếu stack (Docker không tới được)", GATE P3: FAIL rc=1. Lệnh TC cũ (SKIP) đã bị thay: chỉ k6/docling được SKIP — ACCEPTED sprint 6 proposals # |
| TC-P3-08-27 | N | chưa chạy (mẫu): ngoài mẫu smoke 1 chính + 1 lỗi / US |
| TC-P3-08-28 | P | fetch( ngoài shared/: 0; OFFSET trong store/queries: 1 dòng chỉ là chú thích «không OFFSET» |
| TC-P3-08-29 | P | ui-antipatterns.sh rc=0 |
| TC-P8-01-01 | P | presign hợp lệ → 200 |
| TC-P8-01-02 | P | 422 FILE_TYPE_NOT_ALLOWED |
| TC-P8-01-03 | P | size 52428801/0/-1 → [422, 422, 422] |
| TC-P8-01-04 | P | sha ngắn/có G, tên 256 ký tự, ../ → [422, 422, 422, 422] |
| TC-P8-01-05 | N | chạy lại: presign 11 lần liên tiếp gặp 502/503 do stack quá tải/đang recreate (không phải kết quả); cần chạy lại khi ep-r ổn |
| TC-P8-01-06 | N | chưa chạy (mẫu): ngoài mẫu smoke 1 chính + 1 lỗi / US |
| TC-P8-01-07 | P | PUT sai kích thước → 403, đúng → 200 (mốc hết hạn 10 phút: không chờ) |
| TC-P8-01-08 | P | complete → 202 |
| TC-P8-01-09 | P | 3 lần cùng key → [202, 202, 202], thân giống, số tài liệu=1 |
| TC-P8-01-10 … TC-P8-01-14 (5 ca) | N | chưa chạy (mẫu): ngoài mẫu smoke 1 chính + 1 lỗi / US |
| TC-P8-01-15 | P | READY/1 sau 69s |
| TC-P8-01-16 … TC-P8-01-20 (5 ca) | N | chưa chạy (mẫu): ngoài mẫu smoke 1 chính + 1 lỗi / US |
| TC-P8-01-21 | P | vector_dims=1536 |
| TC-P8-01-22 | P | chunks/emb/min_len=1/1/119 |
| TC-P8-01-23 | N | chưa chạy (mẫu): ngoài mẫu smoke 1 chính + 1 lỗi / US |
| TC-P8-01-24 | P | PATCH ẩn → 200; audience ALL→STAFF |
| TC-P8-01-25 … TC-P8-01-28 (4 ca) | N | chưa chạy (mẫu): ngoài mẫu smoke 1 chính + 1 lỗi / US |
| TC-P8-01-29 | P | EXPLAIN (ANALYZE,VERBOSE) trên lớp 761987: cả nhánh vec và kw có Filter d.use_for_rag/visible_to_students/type<>ANSWER_KEY/status=READY + c.course_ids @>/audience=ANY/embedding IS NOT NULL nằm dưới Li |
| TC-P8-01-30 … TC-P8-01-44 (15 ca) | N | chưa chạy (mẫu): ngoài mẫu smoke 1 chính + 1 lỗi / US |
| TC-P8-01-45 | P | goose up→down(về 0)→up trên DB tạm qc2e_tmp (thư viện goose v3.28.0 qua chương trình ngoài repo; CLI goose không build được với go.sum), version 25, pg_dump --schema-only trước/sau giống hệt (diff rỗn |
| TC-P8-01-46 … TC-P8-01-47 (2 ca) | N | chưa chạy (mẫu): ngoài mẫu smoke 1 chính + 1 lỗi / US |
| TC-P8-01-48 | P | 422 {'code': 'FILE_TYPE_MISMATCH', 'message': 'Tệp không đọc được. Dùng PDF, DOCX hoặc PPTX.', 'trace_id': 'e88fb71abcc9b0763b7b7deaae98f677'}; docs=0; blob trước/sau=3/3; jobs 655→655 |
| TC-P8-01-49 | P | (a) 422 UPLOAD_INCOMPLETE; (b) PUT=403 422 UPLOAD_INCOMPLETE; blob trước PUT/sau PUT/sau complete=7/7/7; không tạo documents/jobs |
| TC-P8-01-50 | P | 404/404/404 UPLOAD_NOT_FOUND; cùng thân=True; TTL mô phỏng bằng xoá khoá ['ep:upload:5837e536-039a-470f-aa2f-a3c4ce284652']; không tạo documents |
| TC-P8-01-51 … TC-P8-01-57 (7 ca) | N | chưa chạy (mẫu): ngoài mẫu smoke 1 chính + 1 lỗi / US |
| TC-P8-01-58 | P | {'sv.gioi': [403, 403, 403, 403], 'ta': [422, 422, 409, 202], 'teacher': [422, 422, 409, 202], 'admin': [403, 403, 403, 403]} |
| TC-P8-01-59 | P | reindex cả lớp: TA 403, TEACHER 202 |
| TC-P8-01-60 | P | staff lớp khác/SV ngoài → [403, 403, 403]; xem job người khác → 404 |
| TC-P8-01-61 … TC-P8-01-62 (2 ca) | N | chưa chạy (mẫu): ngoài mẫu smoke 1 chính + 1 lỗi / US |
| TC-P8-02-01 | P | UI: GV /documents «Tài liệu» tải ở 1440 và 375, 0 tràn |
| TC-P8-02-02 | N | chưa chạy (mẫu): ngoài mẫu smoke 1 chính + 1 lỗi / US |
| TC-P8-02-03 | P | limit=1 → 1 mục; limit 0/101 → 422/422 |
| TC-P8-02-04 … TC-P8-02-11 (8 ca) | N | chưa chạy (mẫu): ngoài mẫu smoke 1 chính + 1 lỗi / US |
| TC-P8-02-12 | P | PATCH lần 1 → 200; version cũ → 409 VERSION_CONFLICT |
| TC-P8-02-13 | P | thư viện SV: hiện 1 → ẩn 0 (ngay yêu cầu kế tiếp) |
| TC-P8-02-14 | P | document.patch,document.patch,document.patch |
| TC-P8-02-15 | N | chưa chạy (mẫu): ngoài mẫu smoke 1 chính + 1 lỗi / US |
| TC-P8-02-16 | P | đọc chunks 200, sửa 200, DB đã đổi=True |
| TC-P8-02-17 | P | text rỗng/4001 → 422/422 |
| TC-P8-02-18 | P | {'total': 1, 'by_type': {'LECTURE': 1}, 'by_status': {'READY': 1}, 'chunks': 1, 'embedded_chunks': 1 |
| TC-P8-02-19 … TC-P8-02-20 (2 ca) | N | chưa chạy (mẫu): ngoài mẫu smoke 1 chính + 1 lỗi / US |
| TC-P8-02-21 | P | TA xoá tài liệu → 403 |
| TC-P8-02-22 | N | chưa chạy (mẫu): ngoài mẫu smoke 1 chính + 1 lỗi / US |
| TC-P8-02-23 | P | library q → [200, 200, 200] |
| TC-P8-02-24 … TC-P8-02-25 (2 ca) | N | chưa chạy (mẫu): ngoài mẫu smoke 1 chính + 1 lỗi / US |
| TC-P8-02-26 | P | limit=1 ok, 101 → 422 |
| TC-P8-02-27 | P | chi tiết → 200, download → 200; keys=['id', 'title', 'type', 'file_kind', 'category', 'week_no', 'updated_at', 'snippet'] |
| TC-P8-02-28 … TC-P8-02-36 (9 ca) | N | chưa chạy (mẫu): ngoài mẫu smoke 1 chính + 1 lỗi / US |
| TC-P8-02-37 | P | UI 375: /library «Thư viện» tải, 0 tràn ngang (lần đầu 5xx thoáng qua do tải, lần chạy lại ổn) |
| TC-P8-02-38 … TC-P8-02-44 (7 ca) | N | chưa chạy (mẫu): ngoài mẫu smoke 1 chính + 1 lỗi / US |
| TC-P8-02-45 | P | {'sv.gioi': [403, 403, 403, 403], 'ta': [200, 200, 200, 200], 'teacher': [200, 200, 200, 200], 'admin': [403, 403, 403, 403]} |
| TC-P8-02-46 | N | chưa chạy (mẫu): ngoài mẫu smoke 1 chính + 1 lỗi / US |
| TC-P8-02-47 | P | {'ta': 403, 'teacher': 403, 'admin': 403, 'sv.gioi': 200} |
| TC-P8-02-48 | P | SV ngoài lớp: library 403, download 403 |
| TC-P8-02-49 | N | chưa chạy (mẫu): ngoài mẫu smoke 1 chính + 1 lỗi / US |
| TC-P8-03-01 | P | goose up→down(về 0)→up trên DB tạm qc2e_tmp (thư viện goose v3.28.0 qua chương trình ngoài repo; CLI goose không build được với go.sum), version 25, pg_dump --schema-only trước/sau giống hệt (diff rỗn |
| TC-P8-03-02 | T | reminder_log đúng cột/FK/UNIQUE 5 cột/idx(starts_at); dòng trùng → 23505; kind≠T24H → 23514. reminder_source nay có 4 giá trị (thêm ASSIGNMENT, sprint 9 proposals #3) thay vì 3 |
| TC-P8-03-03 | T | ics_token thô/64×G → 23514 users_ics_token_hash_chk; 64 hex & NULL → UPDATE 1. Phần "PROGRESS.md ghi 00009" lỗi thời do số migration đổi 00011 (P-6-13) |
| TC-P8-03-04 | P | 200 0 mục |
| TC-P8-03-05 | P | DRAFT vắng; SCHEDULED starts/ends khớp opens/closes: 2026-11-17T16:25:47Z → 2026-11-17T17:25:47Z |
| TC-P8-03-06 | P | thiếu to/khoảng >: [422, 422, 422] |
| TC-P8-03-07 | P | limit=101 → 422; limit=1 → 200 0 mục |
| TC-P8-03-08 | P | không có mục lớp 2 (2 mục); lớp 2 → 403 |
| TC-P8-03-09 | P | tạo → 201 |
| TC-P8-03-10 | P | GV lớp khác tạo ở lớp này → 403 |
| TC-P8-03-11 | N | chưa chạy (mẫu): ngoài mẫu smoke 1 chính + 1 lỗi / US |
| TC-P8-03-12 | P | ETag=W/"4PgGAwvHxDQ8RGzv" → If-None-Match → 304 |
| TC-P8-03-13 | P | sau sửa sự kiện, ETag cũ → 200 |
| TC-P8-03-14 | P | trước mốc ETag→304(304); qua opens_at → 200; sau bắt đầu lượt thi ETag cũ → 200, personal_state=IN_PROGRESS |
| TC-P8-03-15 | P | sv31=('SUBMITTED', '/exams/01a1e007-9f2a-7463-9e7b-00fcdc08f26d/take') sv33=('NOT_STARTED', '/exams/01a1e007-9f2a-7463-9e7b-00fcdc08f26d/take') teacher=(None, '/exams/01a1e007-9f2a-7463-9e7b-00fcdc08f |
| TC-P8-03-16 | P | từ khoá cấm trong thân lịch SV: [] |
| TC-P8-03-17 | N | chưa chạy (mẫu): ngoài mẫu smoke 1 chính + 1 lỗi / US |
| TC-P8-03-18 | P | tạo token 201; url có token=True |
| TC-P8-03-19 | P | token cũ sau xoay → 404; token mới → 200 |
| TC-P8-03-20 | P | thu hồi → 204; token sau thu hồi → 404 |
| TC-P8-03-21 | P | ics_token DB: 1/1 |
| TC-P8-03-22 | N | chưa chạy (mẫu): ngoài mẫu smoke 1 chính + 1 lỗi / US |
| TC-P8-03-23 | P | 200 VEVENT=50 ctype=text/calendar; charset=utf-8 |
| TC-P8-03-24 | N | chưa chạy (mẫu): ngoài mẫu smoke 1 chính + 1 lỗi / US |
| TC-P8-03-25 | P | dòng dài nhất 75 byte (≤75) |
| TC-P8-03-26 | P | DESCRIPTION: 0 |
| TC-P8-03-27 | P | ETag=W/"2BfbuFxVMOB0cmQU", cache=private, max-age=300, If-None-Match→304 |
| TC-P8-03-28 | K | (b) cần người thật: dán URL feed vào Google Calendar ("Bạn tự kiểm"); phần HTTP/ICS đã chạy (TC-18…29) |
| TC-P8-03-29 | P | 62 yêu cầu liên tiếp → 429 xuất hiện ở lần 54 |
| TC-P8-03-30 … TC-P8-03-33 (4 ca) | N | chưa chạy (mẫu): ngoài mẫu smoke 1 chính + 1 lỗi / US |
| TC-P8-03-34 | P | reminder_log=5 (5 SV ACTIVE, kind T24H, starts_at=2026-11-16 16:04:24); notifications=5/1//calendar/remind:CALENDAR_EVENT:01a1e00f-5d8d-70e3-9e3d-a8af13b175ff:1794845064 (dedupe `remind:CALENDAR_EVENT |
| TC-P8-03-35 | P | sau thêm ≥3 nhịp cron: reminder_log 5→5, chuông 5→5, thư 1; trùng (user,nguồn,starts_at) = ''. 20 nhịp song song không chạy được bằng CLI (worker không có lệnh reminder-tick) → dùng go test TestRemind |
| TC-P8-03-36 | P | đổi giờ → thêm đúng 1 bộ dòng mới: 2026-11-16 14:34:24/5 / 2026-11-16 16:04:24/5; chuông 10 (2/SV); không nhắc lần 3 sau ≥2 nhịp |
| TC-P8-03-37 | P | EV_A (xoá) + EX_B (bỏ lịch → DRAFT): reminder_log=0, chuông=0, thư=0 |
| TC-P8-03-38 | P | người nhận ['sv31@edupilot.local', 'sv32@edupilot.local', 'sv33@edupilot.local', 'sv34@edupilot.local', 'sv35@edupilot.local']; không TA/GV, không sv36 PENDING, sv37 REMOVED, sv38 DISABLED |
| TC-P8-03-39 | P | GET /me/settings.reminders mặc định={'exam': True, 'class_session': False, 'other': True} (API thật là `reminders` cấp cao, không bọc `preferences` — QC gốc cũng PASS); EXAM lần đầu → 5 người, OTHER c |
| TC-P8-03-40 | P | thư: chỉ sv31 (sv32/33 tắt, sv34 chưa xác minh, sv35 tắt); chuông vẫn có cho cả 5: {'sv31': 4, 'sv32': 3, 'sv33': 3, 'sv34': 3, 'sv35': 2} |
| TC-P8-03-41 | P | lớp ARCHIVED (admin archive → 200) có sự kiện trong 24h: sau ≥3 nhịp reminder_log=0; người DISABLED nhận 0 dòng; SV ACTIVE lớp 1 vẫn nhận bình thường (TC-34/38) |
| TC-P8-03-42 | N | chưa chạy (mẫu): ngoài mẫu smoke 1 chính + 1 lỗi / US |
| TC-P8-03-43 | P | thư có tên sự kiện + giờ + đúng 1 liên kết ['https://localhost/calendar']; không điểm/MSSV/tên người; mail_outbox.template='reminder' (không phải `calendar_reminder`: tên template do dev đặt `reminder |
| TC-P8-03-44 … TC-P8-03-48 (5 ca) | N | chưa chạy (mẫu): ngoài mẫu smoke 1 chính + 1 lỗi / US |
| TC-P8-03-49 | P | UI 375: /calendar (SV & GV) tải, 0 tràn ngang |
| TC-P8-03-50 … TC-P8-03-51 (2 ca) | N | chưa chạy (mẫu): ngoài mẫu smoke 1 chính + 1 lỗi / US |
| TC-P8-03-52 | P | tiêu đề rỗng/kết thúc<bắt đầu/type sai → [422, 422, 422] |
| TC-P8-03-53 | P | PUT 200; version cũ 409 |
| TC-P8-03-54 | P | token rỗng/sai/không tồn tại → [404, 404, 404] |
| TC-P8-03-55 | N | chưa chạy (mẫu): ngoài mẫu smoke 1 chính + 1 lỗi / US |
| TC-P8-03-56 | P | {'sv08': 200, 'ta': 200, 'teacher': 200, 'admin': 403, 'sv.moi': 403, 'r5t': 403} |
| TC-P8-03-57 | P | SV tạo → 403, ADMIN → 403 |
| TC-P8-03-58 | P | ?user_id bị bỏ qua: token vẫn của chính mình (201) |
| TC-P8-03-59 | N | chưa chạy (mẫu): ngoài mẫu smoke 1 chính + 1 lỗi / US |
| TC-P8-03-60 | P | A(sv33) thấy ['NOT_STARTED'], B(sv32) thấy ['IN_PROGRESS']; lịch lớp 2 → 403; /me/settings 200: null |
| TC-id | N | chưa chạy (mẫu): ngoài mẫu smoke 1 chính + 1 lỗi / US |

## Trượt: bước tái hiện
### DEF-R6-01 — Vai DB của ứng dụng là siêu quyền (TC-P3-01-49): `rolsuper=t` ở ep-r (cùng BUG-1 gốc của sprint 6; trigger append-only vẫn chặn 42501). Chi tiết `defects-6.md`.
### DEF-R6-02 — Bộ nhớ đệm roster PII không vô hiệu khi SV vào lớp (TC-P3-02-21): duyệt / vào bằng mã không phát `course.member_changed`, tên SV mới không bị che tới 1 giờ. Chi tiết `defects-6.md`.
### DEF-R6-C-01 — Dòng khoá ở /threads khi đang thi dùng chữ của chat (TC-P3-06-42). Chi tiết `defects-6.md`.

## Ca K / T / N và lý do
- T: TC-P3-03-31 (usage mở cho TEACHER) và TC-P3-04-45 (khủng hoảng hai tầng, câu mẫu đổi — US-P4-07 2026-10-14; kiểm lại hành vi mới ở sprint 7: 20/20 dương, 17/17 âm).
- K(b): TC-P8-03-28 (Google Calendar).
- N: còn lại sau 3 lượt — xem tổng hợp theo US; state các slice ở `docs/regression/evidence-7/state-{c,e,g}.md`.
