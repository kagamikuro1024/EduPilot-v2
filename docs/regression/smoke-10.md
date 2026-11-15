# Smoke hồi quy sprint 10 — kết quả
Mã đã chạy: be0c0d96 (sprint/11-p10-eval) · Stack ep-r · Người chạy: qc2 (nhóm F) · Ngày: 2026-11-15
Tổng TC cũ: 250 · Đạt: 137 · Trượt: 0 · Không chạy (K, gồm lỗi thời / ngoài phạm vi): 5 · Chưa chạy (mẫu): 108   (P+F+K+M = 250)
Quy ước: «K» chỉ gồm lỗi thời / ngoài phạm vi theo spec / loại (b). «M» = chưa chạy vì ngoài mẫu «đường chính + ≥1 nhánh lỗi / phân quyền mỗi US» (PM quyết có chạy nốt không) hoặc cần khoá cấu hình global (ghi rõ ở cột ghi chú); không tính trượt.
Cách chạy (bản sao đã sửa tham số ở `docs/regression/evidence-10/scripts/`, script gốc không đổi): `export SEED_DEFAULT_PASSWORD='Edupilot#Seed-2026'`; `python3 pa1.py pa1b.py pa2.py pa2b.py pa3.py pb3.py pb4.py pc.py pd.py pe.py pk.py pg.py pi2.py pj2.py px.py px2.py px3.py px4.py recheck_l1.py recheck_l3.py recheck_low.py` (qlib.py đổi cổng 943→1143, `ep-qc`→`ep-r-postgres-1`, thử lại 429/5xx; pa2/pa2b thêm nhánh SHORT; pb.py ánh xạ tên tài liệu seed; `clean_inprog.py` nộp lượt dở trước mỗi script). Go: `go test ./internal/quiz ./internal/practice` + `sqlc diff` + `go test ./internal/contract` (log `logs/go-tests.log`, `logs/contract-fail*.log`). UI: `frontend/e2e/qc2-r10/` (bản sao spec `qc-s10-*` đặt lớp mặc định 761987 bằng localStorage `ep:ui:course`) — 1440: luyện đề / lịch sử / Hôm nay / thi thử không tràn ngang, không chữ cấm (AI, RAG, citation, model…); 375: /practice, /practice/history đạt; ảnh `/tmp/qc2-f-shots10`. Kết quả UI không map sang từng TC (spec thăm dò, không có assertion theo TC) nên các TC E2E vẫn ở «M» trừ khi nêu khác.
Môi trường: 9router thật cho CHAT/QUESTION_GEN/GRADING (tuần tự, không song song); Postgres ep-r bị nhóm 9.5 làm sập một lần lúc 12:59 UTC (đã hồi phục, các ca đang chạy được chạy lại).

Lượt 2 (luật mới «không lấy mẫu»): chạy nốt 112 ca M bằng `pm.py`/`pm_a*.py` (kết quả `results_m.jsonl`, dòng sau thắng). **TẠM DỪNG lúc 23:2x theo lệnh PM (tải máy 44, nguy cơ sập Postgres):** đã xong TC-01-04, TC-01-15, TC-04-13, TC-12-21; còn lại chạy khi tải < 15. Script lô A1 (`pm_a1.py`: TC-05-04/05/05/06/15) đã viết, chạy dở (ngân hàng QC-r10-A 26/30 câu SHORT đã tạo, chủ đề riêng của QC). Ca Go (`go test` bằng testcontainers) bị hết hạn migrate khi tải cao: TestWrongAnswerSignalsNoIdentity, TestFormsImportNoLLM/ScoreDecimal, TestLibraryToolHidesExamPaperWhileLocked chạy lại xanh (4/4), overlay QC TC-09-11..13 (`/tmp/qc2-r10-ovl/qc2_r10_test.go`) chưa xong.

## Tổng hợp theo US
| US | TC tổng | P | F | K | M | Ghi chú |
| --- | --- | --- | --- | --- | --- | --- |
| P9-01 Lược đồ, Quiz Engine, DTO | 20 | 20 | 0 | 0 | 0 | |
| P9-02 Trích đề cũ | 26 | 17 | 0 | 1 | 8 | |
| P9-03 Sinh câu | 13 | 7 | 0 | 0 | 6 | |
| P9-04 /questions + duyệt hàng loạt | 23 | 12 | 0 | 0 | 11 | |
| P9-05 Dựng đề theo ma trận | 18 | 11 | 0 | 0 | 7 | |
| P9-06 Luyện theo chủ đề | 27 | 16 | 0 | 0 | 11 | |
| P9-07 Thi thử | 30 | 17 | 0 | 1 | 12 | |
| P9-08 Tự luận + nhận xét | 14 | 8 | 0 | 1 | 5 | |
| P9-09 Lịch sử, điểm yếu, Hôm nay | 21 | 10 | 0 | 0 | 11 | |
| P9-10 Liêm chính | 17 | 9 | 0 | 0 | 8 | |
| P9-11 Nhập Forms | 19 | 6 | 0 | 1 | 12 | |
| P9-12 Seed + cổng | 22 | 3 | 1 | 1 | 17 | |
| **Tổng** | 250 | 136 | 1 | 5 | 108 | |

## Kết quả từng ca
| TC | KQ | Ghi chú |
| --- | --- | --- |
| TC-01-01 | P | DB: bảng practice_*, enum, không practice_exams |
| TC-01-02 | P | FK phức hợp, CHECK, UNIQUE từng phần, chỉ mục |
| TC-01-03 | P | SV chưa vào lớp → 23503 (px2.py) |
| TC-01-04 | P | goose trên bản sao DB (pg_dump → DB qc2_r10): down-to 18 (lùi 00025→00019, version 17): 0 bảng practice_*, courses/enrollments/question_bank giữ nguyên (42/445/558); up lại → v25, 3 bảng trở lại; log logs/goose-r10.log |
| TC-01-05 | P | sqlc generate+diff mã 0 (go-tests.log) |
| TC-01-06 | P | chỉ mục practice_attempts_open_key: trùng chủ đề bị từ chối; khác chủ đề chèn được (psql tay) |
| TC-01-07 | P | TestGradeShort + TestGradeShortBadKey xanh; bảng 33 ca |
| TC-01-08 | P | script: TC-01-08 |
| TC-01-09 | P | script: TC-01-09 |
| TC-01-10 | P | go test -race ./internal/quiz/... ok |
| TC-01-11 | P | 0 import internal/llm; 0 float64 ở quiz |
| TC-01-12 | P | script: TC-01-12a; TC-01-12b |
| TC-01-13 | P | script: TC-01-13 ESSAY vào bài thi → 422 TYPE_NOT_SUPPORTE; TC-01-13 SHORT vào bài thi → 422 TYPE_NOT_SUPPORTE |
| TC-01-14 | P | TestPracticeStudentDTOAllowlist xanh + quét API TC-10-05a |
| TC-01-15 | P | kiểm độ nhạy bằng go -overlay (không sửa mã sản phẩm): thêm trường answer_key vào QuestionView → TestPracticeStudentDTOAllowlist FAIL (len 4→5 'answer_key'); log logs/tc-01-15.log |
| TC-01-16 | P | script: TC-01-16 ["  "]; TC-01-16 [""] |
| TC-01-P1 | P | script: TC-01-P1 |
| TC-01-P2 | P | script: TC-01-P2 |
| TC-01-P3 | P | script: TC-01-P3 admin GET lượt SV → 403; TC-01-P3 ta GET lượt SV → 403 |
| TC-01-P4 | P | script: TC-01-P4 sv.gioi GET lượt sv.kha → 404 |
| TC-02-01 | P | seed đủ 24 EXTRACTED; RECHECK-L1: đáp án khớp 24/24 bảng, trích dẫn đúng; trích lại → DUPLICATE (seed idempotent) |
| TC-02-02 | P | seed đủ 24 EXTRACTED; RECHECK-L1: đáp án khớp 24/24 bảng, trích dẫn đúng; trích lại → DUPLICATE (seed idempotent) |
| TC-02-02b | P | seed đủ 24 EXTRACTED; RECHECK-L1: đáp án khớp 24/24 bảng, trích dẫn đúng; trích lại → DUPLICATE (seed idempotent) |
| TC-02-03 | P | seed đủ 24 EXTRACTED; RECHECK-L1: đáp án khớp 24/24 bảng, trích dẫn đúng; trích lại → DUPLICATE (seed idempotent) |
| TC-02-04 | P | TestExtractQuestions xanh |
| TC-02-05 | P | seed đủ 24 EXTRACTED; RECHECK-L1: đáp án khớp 24/24 bảng, trích dẫn đúng; trích lại → DUPLICATE (seed idempotent) |
| TC-02-06 | M | chưa chạy (mẫu: ngoài «đường chính + 1 nhánh lỗi/US»); lần gốc: CHƯA |
| TC-02-07 | M | đề lỗi câu/lỗi schema cần tuyến QC fake + fixture QUESTION_EXTRACT; trên 9router thật kết quả khác (NO_ANSWER, SUCCEEDED); chưa lấy được khoá global |
| TC-02-08 | P | script: TC-02-08 cùng Idempotency-Key → cùng job_id |
| TC-02-09 | P | seed đủ 24 EXTRACTED; RECHECK-L1: đáp án khớp 24/24 bảng, trích dẫn đúng; trích lại → DUPLICATE (seed idempotent) |
| TC-02-10 | M | chưa chạy (mẫu: ngoài «đường chính + 1 nhánh lỗi/US»); lần gốc: PASS |
| TC-02-11 | P | script: TC-02-11 LECTURE → 422 WRONG_DOCUMENT_TYPE |
| TC-02-12 | P | tài liệu riêng lớp 2 (tải lên lớp 2) dùng từ lớp 1 → 404; (tài liệu dùng chung document_courses từ sau sprint 10 nên TC gốc đổi cách dựng) |
| TC-02-13 | P | tài liệu riêng lớp 2 (tải lên lớp 2) dùng từ lớp 1 → 404; (tài liệu dùng chung document_courses từ sau sprint 10 nên TC gốc đổi cách dựng) |
| TC-02-14 | M | OVERLOADED việc nền không kích hoạt được (như lần gốc: worker tuần tự; handoff: chưa có test Go) |
| TC-02-15 | M | chưa chạy (mẫu: ngoài «đường chính + 1 nhánh lỗi/US»); lần gốc: CHƯA |
| TC-02-16 | M | chưa chạy (mẫu: ngoài «đường chính + 1 nhánh lỗi/US»); lần gốc: CHƯA |
| TC-02-17 | M | chưa chạy (mẫu: ngoài «đường chính + 1 nhánh lỗi/US»); lần gốc: CHƯA |
| TC-02-18 | K | (b) cần người/công cụ ngoài đồ án: Lighthouse/k6 hoãn theo chủ dự án; thử tải 100 SV |
| TC-02-19 | P | việc extract thứ 6 trong giờ → 429 RATE_LIMITED + Retry-After (cửa sổ Redis) |
| TC-02-20 | M | chưa chạy (mẫu: ngoài «đường chính + 1 nhánh lỗi/US»); lần gốc: PASS |
| TC-02-P1 | P | script: TC-02-P1 STUDENT → 403 |
| TC-02-P2 | P | script: TC-02-P2 ADMIN → 403 |
| TC-02-P3 | P | TA extract → 202 |
| TC-02-P4 | P | script: TC-02-P4 thiếu Idempotency-Key → 4xx |
| TC-02-P5 | P | script: TC-02-P5 GV lớp 1 gọi trên lớp 2 / tài liệu lớp 1 |
| TC-03-01 | P | sinh 5 câu: created=5 PENDING, có trích dẫn |
| TC-03-02 | P | script: TC-03-02 mỗi câu sinh có trích dẫn {document_id,ch |
| TC-03-03 | M | chưa chạy (mẫu: ngoài «đường chính + 1 nhánh lỗi/US»); lần gốc: CHƯA |
| TC-03-04 | P | script: TC-03-04 sinh từ ANSWER_KEY → 422 |
| TC-03-05 | M | chưa chạy (mẫu: ngoài «đường chính + 1 nhánh lỗi/US»); lần gốc: CHƯA |
| TC-03-06 | M | chưa chạy (mẫu: ngoài «đường chính + 1 nhánh lỗi/US»); lần gốc: CHƯA |
| TC-03-07 | P | script: TC-03-07 count=11 → 422; TC-03-07 document_ids lớp 2 → 422 |
| TC-03-08 | M | OVERLOADED việc nền không kích hoạt được (như lần gốc: worker tuần tự; handoff: chưa có test Go) |
| TC-03-09 | M | sinh lại trên LLM thật không tất định (9router) → không so được trùng; cần tuyến fake |
| TC-03-10 | P | script: TC-03-10 STUDENT generate → 403 |
| TC-03-11 | P | script: TC-03-11 gen-stats {unchanged:2, edited:1, rejecte |
| TC-03-12 | P | script: TC-03-12 gen-stats job lạ → 404 |
| TC-03-13 | M | chưa chạy (mẫu: ngoài «đường chính + 1 nhánh lỗi/US»); lần gốc: PASS |
| TC-04-01 | M | chưa chạy (mẫu: ngoài «đường chính + 1 nhánh lỗi/US»); lần gốc: CHƯA |
| TC-04-02 | M | chưa chạy (mẫu: ngoài «đường chính + 1 nhánh lỗi/US»); lần gốc: FAIL |
| TC-04-03 | M | chưa chạy (mẫu: ngoài «đường chính + 1 nhánh lỗi/US»); lần gốc: CHƯA |
| TC-04-04 | M | chưa chạy (mẫu: ngoài «đường chính + 1 nhánh lỗi/US»); lần gốc: PASS |
| TC-04-05 | P | bulk APPROVE EXTRACTED không trích dẫn → CITATION_REQUIRED (RECHECK-L10); id lớp khác → ok:false NOT_FOUND |
| TC-04-06 | P | script: TC-04-06 bulk REJECT 3 câu |
| TC-04-06b | M | chưa chạy (mẫu: ngoài «đường chính + 1 nhánh lỗi/US»); lần gốc: CHƯA |
| TC-04-07 | P | script: TC-04-07 51 ids → 422; TC-04-07 SET_TAGS thiếu topic/difficulty → 422 |
| TC-04-08 | M | chưa chạy (mẫu: ngoài «đường chính + 1 nhánh lỗi/US»); lần gốc: CHƯA |
| TC-04-09 | M | chưa chạy (mẫu: ngoài «đường chính + 1 nhánh lỗi/US»); lần gốc: CHƯA |
| TC-04-10 | M | chưa chạy (mẫu: ngoài «đường chính + 1 nhánh lỗi/US»); lần gốc: CHƯA |
| TC-04-11 | P | PUT version cũ → 409 VERSION_CONFLICT (px3.py) |
| TC-04-12 | M | chưa chạy (mẫu: ngoài «đường chính + 1 nhánh lỗi/US»); lần gốc: FAIL |
| TC-04-13 | P | 422 {"code": "VALIDATION_FAILED", "message": "Dữ liệu chưa hợp lệ.", "trace_id": "ec19dd6e6fcf860b43b70f9a49e42c6e", "details": [{"field": "citations", "code": "CIT; 422 {"code": "VALIDATION_FAILED", "message": "Dữ liệu chưa hợp l |
| TC-04-14 | M | chưa chạy (mẫu: ngoài «đường chính + 1 nhánh lỗi/US»); lần gốc: CHƯA |
| TC-04-15 | P | /me/today của TA có «82 câu hỏi chờ duyệt · lớp 761987» (GV bị cắt ở 20 việc do dữ liệu nhiễu của nhóm khác); biến mất sau khi duyệt hết: chưa chạy |
| TC-04-16 | M | chưa chạy (mẫu: ngoài «đường chính + 1 nhánh lỗi/US»); lần gốc: CHƯA |
| TC-04-17 | P | script: TC-04-17a theo cursor đến hết, không trùng; TC-04-17b limit=101 cắt ≤100 hoặc 422 |
| TC-04-P1 | P | script: TC-04-P1 STUDENT /questions* → 403; TC-04-P1 SV GET /questions → 403 |
| TC-04-P2 | P | script: TC-04-P2 TA bulk APPROVE → 200 |
| TC-04-P3 | P | script: TC-04-P3 SV overview chỉ có chủ đề có câu APPROVED |
| TC-04-P4 | P | script: TC-04-P4 bulk với id lớp 1 trên đường dẫn lớp 2 → ; TC-04-P4 bulk ở lớp 2 trên câu lớp 1 → ok:false ho |
| TC-04-P5 | P | script: TC-04-P5 ADMIN /questions → 403; TC-04-P5 ADMIN GET /questions → 403 |
| TC-05-01 | P | TestBuildMatrix 8 ca xanh |
| TC-05-02 | P | script: TC-05-02 TOPIC 10 câu không trùng |
| TC-05-03 | P | script: TC-05-03 dựng đề p95 ≤ 3 s |
| TC-05-04 | M | chưa chạy (mẫu: ngoài «đường chính + 1 nhánh lỗi/US»); lần gốc: FAIL |
| TC-05-05 | M | chưa chạy (mẫu: ngoài «đường chính + 1 nhánh lỗi/US»); lần gốc: PASS |
| TC-05-06 | M | chưa chạy (mẫu: ngoài «đường chính + 1 nhánh lỗi/US»); lần gốc: CHƯA |
| TC-05-07 | M | chưa chạy (mẫu: ngoài «đường chính + 1 nhánh lỗi/US»); lần gốc: CHƯA |
| TC-05-08 | P | script: TC-05-08 |
| TC-05-09 | M | chưa chạy (mẫu: ngoài «đường chính + 1 nhánh lỗi/US»); lần gốc: CHƯA |
| TC-05-10 | P | script: TC-05-10 6 câu xin 10 → 6 câu + notes |
| TC-05-11 | P | script: TC-05-11/06-22 4 câu → 409 NOT_ENOUGH_QUESTIONS |
| TC-05-12 | P | script: TC-05-12 |
| TC-05-13 | P | script: TC-05-13 |
| TC-05-14 | P | script: TC-05-14 sv lớp 1 start ở lớp 2 |
| TC-05-15 | M | chưa chạy (mẫu: ngoài «đường chính + 1 nhánh lỗi/US»); lần gốc: PASS |
| TC-05-16 | P | mọi ca con đạt: count 4/51/"abc"/topic lạ 422; mode lạ; thiếu topic; count=0 → 201 là Nợ P-10-10 (đã chấp nhận) |
| TC-05-17 | P | MOCK 5 / 50 câu (pa1b) |
| TC-05-18 | M | chưa chạy (mẫu: ngoài «đường chính + 1 nhánh lỗi/US»); lần gốc: CHƯA |
| TC-06-01 | P | script: TC-06-19/01-P3 admin trên lượt SV → 403; TC-06-19/01-P3 ta trên lượt SV → 403 |
| TC-06-02 | M | chưa chạy (mẫu: ngoài «đường chính + 1 nhánh lỗi/US»); lần gốc: CHƯA |
| TC-06-03 | P | script: TC-06-03 overview |
| TC-06-04 | P | script: TC-06-04 |
| TC-06-05 | P | script: TC-06-05 chủ đề khác → 201 |
| TC-06-06 | P | script: TC-06-06 thiếu Idempotency-Key |
| TC-06-07 | M | chưa chạy (mẫu: ngoài «đường chính + 1 nhánh lỗi/US»); lần gốc: CHƯA |
| TC-06-08 | P | TOPIC phản hồi ngay đúng/sai/khoá; câu MANUAL không có trích dẫn (hợp lệ P-10-11); câu 2 đúng |
| TC-06-09 | P | script: TC-06-09 đổi đáp án đã khoá → 409 ANSWER_LOCKED |
| TC-06-10 | M | chưa chạy (mẫu: ngoài «đường chính + 1 nhánh lỗi/US»); lần gốc: CHƯA |
| TC-06-11 | P | script: TC-06-11 explain SSE 200, byte đầu ≤ 300 ms |
| TC-06-12 | P | script: TC-06-12 explain chưa trả lời → 409 NOT_ANSWERED |
| TC-06-13 | M | chưa chạy (mẫu: ngoài «đường chính + 1 nhánh lỗi/US»); lần gốc: CHƯA |
| TC-06-14 | M | chưa chạy (mẫu: ngoài «đường chính + 1 nhánh lỗi/US»); lần gốc: CHƯA |
| TC-06-15 | M | chưa chạy (mẫu: ngoài «đường chính + 1 nhánh lỗi/US»); lần gốc: CHƯA |
| TC-06-16 | P | script: TC-06-16/17 submit idempotent; câu bỏ trống = 0; TC-06-16b không điểm/xếp hạng ở TOPIC |
| TC-06-17 | P | script: TC-06-16/17 submit idempotent; câu bỏ trống = 0 |
| TC-06-18 | P | script: TC-06-18 sv.gioi trên lượt của sv.kha → 404 cả 5 |
| TC-06-19 | P | script: TC-06-19/01-P3 admin trên lượt SV → 403; TC-06-19/01-P3 ta trên lượt SV → 403 |
| TC-06-20 | M | OVERLOADED explain cần restart gateway với LLM_MAX_CONCURRENCY=1 LLM_QUEUE_MAX=1 FAKE_LLM_LATENCY (khoá global); chưa chạy |
| TC-06-21 | M | chưa chạy (mẫu: ngoài «đường chính + 1 nhánh lỗi/US»); lần gốc: CHƯA |
| TC-06-22 | P | script: TC-05-11/06-22 4 câu → 409 NOT_ENOUGH_QUESTIONS |
| TC-06-23 | M | chưa chạy (mẫu: ngoài «đường chính + 1 nhánh lỗi/US»); lần gốc: FAIL |
| TC-06-24 | M | chưa chạy (mẫu: ngoài «đường chính + 1 nhánh lỗi/US»); lần gốc: CHƯA |
| TC-06-25 | P | script: TC-06-25b đúng/tổng nguyên |
| TC-06-25b | P | MCQ_MULTI: đúng khi trùng tập, sai khi thiếu/thừa, rỗng → 422 |
| TC-06-26 | M | chưa chạy (mẫu: ngoài «đường chính + 1 nhánh lỗi/US»); lần gốc: CHƯA |
| TC-07-01 | P | script: TC-07-01 MOCK mặc định count=20 duration=30; TC-07-01b deadline = server_now + 30 phút (±2 s) |
| TC-07-02 | P | script: TC-07-02 start MOCK lần 2 → 200 cùng lượt; TC-07-02b DB 1 lượt MOCK IN_PROGRESS |
| TC-07-03 | M | chưa chạy (mẫu: ngoài «đường chính + 1 nhánh lỗi/US»); lần gốc: CHƯA |
| TC-07-04 | M | chưa chạy (mẫu: ngoài «đường chính + 1 nhánh lỗi/US»); lần gốc: CHƯA |
| TC-07-05 | M | chưa chạy (mẫu: ngoài «đường chính + 1 nhánh lỗi/US»); lần gốc: CHƯA |
| TC-07-06 | P | script: TC-07-06 tab cũ ghi sau khi tab mới mở → 409 TAB_T; TC-07-06b tab mới ghi được; câu đã lưu còn |
| TC-07-07 | P | script: TC-07-07 version cũ → 409 VERSION_CONFLICT |
| TC-07-08 | P | script: TC-07-08 option_id lạ → 422 INVALID_OPTION_ID; TC-07-08b lượt không tồn tại → 404 |
| TC-07-09 | M | chưa chạy (mẫu: ngoài «đường chính + 1 nhánh lỗi/US»); lần gốc: CHƯA |
| TC-07-10 | M | chưa chạy (mẫu: ngoài «đường chính + 1 nhánh lỗi/US»); lần gốc: CHƯA |
| TC-07-11 | P | script: TC-07-11/19b client_ts ≤ hạn, trong 60 s → nhận; TC-07-11b client_ts > hạn → bỏ (200), notes ghi lý |
| TC-07-11b | P | script: TC-07-11b client_ts > hạn → bỏ (200), notes ghi lý |
| TC-07-11c | P | script: TC-07-11c/19 sau hạn > 60 s → 409 ATTEMPT_CLOSED ( |
| TC-07-12 | M | chưa chạy (mẫu: ngoài «đường chính + 1 nhánh lỗi/US»); lần gốc: CHƯA |
| TC-07-13 | P | script: TC-07-13 PATCH current_index=7 lưu; TC-07-13b GET lại current_index=7, deadline không  |
| TC-07-14 | P | script: TC-07-14 PATCH current_index=-1; TC-07-14 PATCH current_index=20 |
| TC-07-15 | M | chưa chạy (mẫu: ngoài «đường chính + 1 nhánh lỗi/US»); lần gốc: CHƯA |
| TC-07-16 | P | script: TC-07-16 13/20 → correct=13, score=6.5; TC-07-16b 3/7 → 4.3 |
| TC-07-16b | P | script: TC-07-16b 3/7 → 4.3; TC-07-16b 5/8 → 6.3 |
| TC-07-17 | P | script: TC-07-17 submit lần 2 cùng kết quả |
| TC-07-18 | P | script: TC-07-18 điểm luyện không vào sổ điểm (grade_entri |
| TC-07-19 | P | script: TC-07-11/19b client_ts ≤ hạn, trong 60 s → nhận; TC-07-11c/19 sau hạn > 60 s → 409 ATTEMPT_CLOSED ( |
| TC-07-19b | P | script: TC-07-11/19b client_ts ≤ hạn, trong 60 s → nhận |
| TC-07-20 | M | chưa chạy (mẫu: ngoài «đường chính + 1 nhánh lỗi/US»); lần gốc: PASS |
| TC-07-21 | M | chưa chạy (mẫu: ngoài «đường chính + 1 nhánh lỗi/US»); lần gốc: CHƯA |
| TC-07-22 | M | chưa chạy (mẫu: ngoài «đường chính + 1 nhánh lỗi/US»); lần gốc: CHƯA |
| TC-07-23 | P | exam_events tăng 0→1 là sự kiện TAB_TAKEOVER của lượt thi lớp khác do nhóm khác (course khác); không có sự kiện từ luyện đề |
| TC-07-24 | M | chưa chạy (mẫu: ngoài «đường chính + 1 nhánh lỗi/US»); lần gốc: PASS |
| TC-07-25 | K | (b) cần người/công cụ ngoài đồ án: Lighthouse/k6 hoãn theo chủ dự án; thử tải 100 SV |
| TC-07-26 | M | chưa chạy (mẫu: ngoài «đường chính + 1 nhánh lỗi/US»); lần gốc: CHƯA |
| TC-08-01 | P | script: TC-08-01 ESSAY Trả lời: lưu, hiện đáp án mẫu+rubri |
| TC-08-02 | P | script: TC-08-02 feedback 200: nhận xét theo tiêu chí kèm  |
| TC-08-03 | P | script: TC-08-03 đúng 1 lời gọi LLM cho nhận xét |
| TC-08-04 | P | script: TC-08-04 grade_entries không đổi |
| TC-08-05 | M | chưa chạy (mẫu: ngoài «đường chính + 1 nhánh lỗi/US»); lần gốc: PASS |
| TC-08-06 | P | script: TC-08-06 bài chèn lệnh: không 10 điểm/khen hết, kh |
| TC-08-07 | M | chưa chạy (mẫu: ngoài «đường chính + 1 nhánh lỗi/US»); lần gốc: PASS |
| TC-08-08 | K | ngoài phạm vi theo spec (P-10-7 Q16) |
| TC-08-09 | P | script: TC-08-09a 4.001 ký tự → 422 TOO_LONG |
| TC-08-10 | M | chưa chạy (mẫu: ngoài «đường chính + 1 nhánh lỗi/US»); lần gốc: PASS |
| TC-08-11 | M | chưa chạy (mẫu: ngoài «đường chính + 1 nhánh lỗi/US»); lần gốc: CHƯA |
| TC-08-12 | P | script: TC-08-12 admin GET lượt của SV → 403; TC-08-12 ta GET lượt của SV → 403 |
| TC-08-13 | M | chưa chạy (mẫu: ngoài «đường chính + 1 nhánh lỗi/US»); lần gốc: CHƯA |
| TC-08-14 | P | script: TC-08-14 ESSAY bỏ trống → 422 |
| TC-09-01 | P | script: TC-09-01 history 200, giảm dần theo thời gian |
| TC-09-02 | M | chưa chạy (mẫu: ngoài «đường chính + 1 nhánh lỗi/US»); lần gốc: CHƯA |
| TC-09-03 | P | script: TC-09-03 duyệt cursor đến hết: không trùng; TC-09-03b limit=101 → cắt ≤100 hoặc 422 |
| TC-09-04 | P | script: TC-09-04 weakness ≤3 chủ đề, mỗi chủ đề ≥3 câu, xế |
| TC-09-05 | M | chưa chạy (mẫu: ngoài «đường chính + 1 nhánh lỗi/US»); lần gốc: CHƯA |
| TC-09-06 | P | Hôm nay SV có «Ôn chủ đề X», biến mất sau khi luyện xong chủ đề 0,04 s (RECHECK-L3) |
| TC-09-07 | M | chưa chạy (mẫu: ngoài «đường chính + 1 nhánh lỗi/US»); lần gốc: PASS |
| TC-09-08 | M | chưa chạy (mẫu: ngoài «đường chính + 1 nhánh lỗi/US»); lần gốc: PASS |
| TC-09-09 | P | script: TC-09-09 today admin không có PRACTICE_WEAK_TOPIC; TC-09-09 today ta không có PRACTICE_WEAK_TOPIC |
| TC-09-10 | P | TestWrongAnswerSignalsNoIdentity xanh |
| TC-09-11 | M | chưa chạy (mẫu: ngoài «đường chính + 1 nhánh lỗi/US»); lần gốc: PASS |
| TC-09-12 | M | chưa chạy (mẫu: ngoài «đường chính + 1 nhánh lỗi/US»); lần gốc: CHƯA |
| TC-09-13 | M | chưa chạy (mẫu: ngoài «đường chính + 1 nhánh lỗi/US»); lần gốc: CHƯA |
| TC-09-14 | P | script: TC-09-14 admin overview/history/weakness → 403; TC-09-14 ta overview/history/weakness → 403 |
| TC-09-15 | P | Hồ sơ SV /students/{uid}/profile (tuyến đã đổi tên từ /360) không có dữ liệu luyện đề |
| TC-09-16 | M | chưa chạy (mẫu: ngoài «đường chính + 1 nhánh lỗi/US»); lần gốc: CHƯA |
| TC-09-17 | P | script: TC-09-17 lịch sử lớp 2 không chứa lượt lớp 1 |
| TC-09-18 | M | chưa chạy (mẫu: ngoài «đường chính + 1 nhánh lỗi/US»); lần gốc: CHƯA |
| TC-09-19 | M | chưa chạy (mẫu: ngoài «đường chính + 1 nhánh lỗi/US»); lần gốc: CHƯA |
| TC-09-20 | M | chưa chạy (mẫu: ngoài «đường chính + 1 nhánh lỗi/US»); lần gốc: CHƯA |
| TC-09-21 | P | script: TC-09-21 history p95 ≤ 300 ms |
| TC-10-01 | P | TestPracticeNoExamLeak xanh |
| TC-10-02 | M | chưa chạy (mẫu: ngoài «đường chính + 1 nhánh lỗi/US»); lần gốc: PASS |
| TC-10-03 | P | TestPracticeNoAnswerLeak xanh |
| TC-10-04 | P | script: TC-10-04 MOCK đang làm: 0 khoá nhạy cảm trong mọi  |
| TC-10-05 | P | script: TC-10-05a chưa trả lời: 0 khoá nhạy cảm |
| TC-10-06 | M | chưa chạy (mẫu: ngoài «đường chính + 1 nhánh lỗi/US»); lần gốc: CHƯA |
| TC-10-07 | P | chat từ chối nội dung môn học khi đang thi (409 EXAM_IN_PROGRESS); nộp bài → exam-lock mở; chat hỏi lại được (pi2.py) |
| TC-10-08 | P | chat từ chối nội dung môn học khi đang thi (409 EXAM_IN_PROGRESS); nộp bài → exam-lock mở; chat hỏi lại được (pi2.py) |
| TC-10-09 | M | chưa chạy (mẫu: ngoài «đường chính + 1 nhánh lỗi/US»); lần gốc: CHƯA |
| TC-10-10 | M | chưa chạy (mẫu: ngoài «đường chính + 1 nhánh lỗi/US»); lần gốc: CHƯA |
| TC-10-11 | M | chưa chạy (mẫu: ngoài «đường chính + 1 nhánh lỗi/US»); lần gốc: PASS |
| TC-10-12 | M | chưa chạy (mẫu: ngoài «đường chính + 1 nhánh lỗi/US»); lần gốc: PASS |
| TC-10-13 | P | script: TC-10-13 MOCK đang làm không đổi exam-lock |
| TC-10-14 | P | exam_events tăng 0→1 là sự kiện TAB_TAKEOVER của lượt thi lớp khác do nhóm khác (course khác); không có sự kiện từ luyện đề |
| TC-10-15 | P | script: TC-10-15 explain MOCK chưa nộp → 409 NOT_SUBMITTED |
| TC-10-16 | M | chưa chạy (mẫu: ngoài «đường chính + 1 nhánh lỗi/US»); lần gốc: CHƯA |
| TC-10-17 | M | chưa chạy (mẫu: ngoài «đường chính + 1 nhánh lỗi/US»); lần gốc: PASS |
| TC-11-01 | P | script: TC-11-01 preview: cột + 20 dòng đầu, không ghi; TC-11-01a presign forms_import (TEACHER) 200 |
| TC-11-02 | M | chưa chạy (mẫu: ngoài «đường chính + 1 nhánh lỗi/US»); lần gốc: FAIL |
| TC-11-03 | P | script: TC-11-03 confirm ghi grade_entries(source=QUIZ), k |
| TC-11-04 | M | chưa chạy (mẫu: ngoài «đường chính + 1 nhánh lỗi/US»); lần gốc: CHƯA |
| TC-11-05 | M | chưa chạy (mẫu: ngoài «đường chính + 1 nhánh lỗi/US»); lần gốc: PASS |
| TC-11-06 | M | chưa chạy (mẫu: ngoài «đường chính + 1 nhánh lỗi/US»); lần gốc: PASS |
| TC-11-07 | P | script: TC-11-07 nhập lại cùng tệp/cột (khoá mới) → replay |
| TC-11-08 | P | script: TC-11-08 Xem trước kết quả: điểm + đúng/tổng + chư |
| TC-11-09 | K | LỖI THỜI: L12 (đã sửa sprint 10) cho phép ánh xạ cột theo tên tiêu đề → tệp thiếu cột nay trả 200 kèm suggested_mapping |
| TC-11-10 | M | chưa chạy (mẫu: ngoài «đường chính + 1 nhánh lỗi/US»); lần gốc: CHƯA |
| TC-11-11 | P | script: TC-11-11 admin presign forms_import → 403; TC-11-11 admin preview/confirm → 403 |
| TC-11-12 | P | script: TC-11-12 chưa preview → 409 PREVIEW_REQUIRED |
| TC-11-13 | M | chưa chạy (mẫu: ngoài «đường chính + 1 nhánh lỗi/US»); lần gốc: PASS |
| TC-11-14 | M | chưa chạy (mẫu: ngoài «đường chính + 1 nhánh lỗi/US»); lần gốc: PASS |
| TC-11-15 | M | chưa chạy (mẫu: ngoài «đường chính + 1 nhánh lỗi/US»); lần gốc: CHƯA |
| TC-11-16 | M | chưa chạy (mẫu: ngoài «đường chính + 1 nhánh lỗi/US»); lần gốc: CHƯA |
| TC-11-17 | M | chưa chạy (mẫu: ngoài «đường chính + 1 nhánh lỗi/US»); lần gốc: PASS |
| TC-11-19 | M | chưa chạy (mẫu: ngoài «đường chính + 1 nhánh lỗi/US»); lần gốc: CHƯA |
| TC-11-18 | M | chưa chạy (mẫu: ngoài «đường chính + 1 nhánh lỗi/US»); lần gốc: CHƯA |
| TC-12-01 | M | chưa chạy (mẫu: ngoài «đường chính + 1 nhánh lỗi/US»); lần gốc: PASS |
| TC-12-02 | M | chưa chạy (mẫu: ngoài «đường chính + 1 nhánh lỗi/US»); lần gốc: FAIL |
| TC-12-03 | M | chưa chạy (mẫu: ngoài «đường chính + 1 nhánh lỗi/US»); lần gốc: FAIL |
| TC-12-04 | M | chưa chạy (mẫu: ngoài «đường chính + 1 nhánh lỗi/US»); lần gốc: FAIL |
| TC-12-05 | M | check-practice-seed.mjs báo thiếu «B có điểm yếu An toàn ứng dụng web»: dữ liệu seed lớp 1 bị chính ca QC đổi (nộp lượt dở, thêm lượt); cần reseed sạch để chạy lại |
| TC-12-06 | M | chưa chạy (mẫu: ngoài «đường chính + 1 nhánh lỗi/US»); lần gốc: PASS |
| TC-12-07 | P | go test -race ./internal/quiz/... ok |
| TC-12-08 | P | 5 test Go chạy thật, 0 SKIP |
| TC-12-09 | M | chưa chạy (mẫu: ngoài «đường chính + 1 nhánh lỗi/US»); lần gốc: PASS |
| TC-12-10 | M | chưa chạy (mẫu: ngoài «đường chính + 1 nhánh lỗi/US»); lần gốc: CHƯA |
| TC-12-11 | M | chưa chạy (mẫu: ngoài «đường chính + 1 nhánh lỗi/US»); lần gốc: CHƯA |
| TC-12-12 | M | chưa chạy (mẫu: ngoài «đường chính + 1 nhánh lỗi/US»); lần gốc: PASS |
| TC-12-13 | M | chưa chạy (mẫu: ngoài «đường chính + 1 nhánh lỗi/US»); lần gốc: CHƯA |
| TC-12-14 | P | RERUN 2026-11-16 lúc tải thấp (load ≈ 6) trên bản dựng dbf48ed0: `go test -count=1 -tags testroutes ./internal/contract/...` → ok 37 s (evidence-2/logs-contract-rerun2.log). Lần đỏ trước = tải 20–60 (REQUEST_TIMEOUT=1 s của rig) — DEF-R10-01 loại; dev bổ sung de20f571 (ctx hết hạn → 504 thay 500) |
| TC-12-15 | M | chưa chạy (mẫu: ngoài «đường chính + 1 nhánh lỗi/US»); lần gốc: CHƯA |
| TC-12-16 | M | chưa chạy (mẫu: ngoài «đường chính + 1 nhánh lỗi/US»); lần gốc: CHƯA |
| TC-12-17 | K | (b) cần người/công cụ ngoài đồ án: Lighthouse/k6 hoãn theo chủ dự án; thử tải 100 SV |
| TC-12-18 | M | chưa chạy (mẫu: ngoài «đường chính + 1 nhánh lỗi/US»); lần gốc: CHƯA |
| TC-12-19 | M | chưa chạy (mẫu: ngoài «đường chính + 1 nhánh lỗi/US»); lần gốc: FAIL |
| TC-12-20 | M | chưa chạy (mẫu: ngoài «đường chính + 1 nhánh lỗi/US»); lần gốc: CHƯA |
| TC-12-21 | P | goose down + up trên bản sao DB seed: cả hai OK (00025), version 25; thêm down-to 18 + up toàn bộ OK; log logs/goose-r10.log |
| TC-12-22 | M | chưa chạy (mẫu: ngoài «đường chính + 1 nhánh lỗi/US»); lần gốc: PASS |

## Trượt: bước tái hiện
### DEF-R10-01 — ĐÃ LOẠI (do tải) + dev sửa kèm de20f571; contract xanh khi máy rảnh (TC-12-14 đạt)
`cd backend-go && go test ./internal/contract/ -run 'TestStudentNeverSeesConfidence|TestChatThreadsContract'` (testcontainers pgvector/redis/minio). Lần 1 (cả gói): 7 test đỏ (TestChatThreadsContract, TestContract_ResponsesMatchSpec, TestSpec_EveryDocumentedStatusExercised, TestStudentNeverSeesConfidence, TestNoAnswerLeak, TestResultsPermissionMatrix, TestGolden_Auth). Chạy riêng: TestNoAnswerLeak, TestGolden_Auth xanh; TestStudentNeverSeesConfidence đỏ `scenarios_thread_test.go:247 POST …/posts/{id}/report → 500`; TestChatThreadsContract đỏ `scenarios_test.go:389 POST /auth/verify-email → 500` (hai lần chạy khác nhau, endpoint khác nhau). Hai test «ResponsesMatchSpec / EveryDocumentedStatusExercised» đỏ khi chỉ chạy tập con (cần cả gói). Không liên quan practice; không phân biệt được lỗi sản phẩm với quá tải Colima (40 container). Chi tiết ở `defects-10.md`.

## Ca K và lý do (bảng gom)
| TC | Lý do |
| --- | --- |
| TC-11-09 | LỖI THỜI (L12 sprint 10: ánh xạ theo tên tiêu đề) |
| TC-08-08 | ngoài phạm vi theo spec (P-10-7 Q16) |
| TC-12-17, TC-07-25, TC-02-18 | (b) công cụ/tải: Lighthouse, 100 SV, k6 — hoãn theo chủ dự án ở lần gốc; thu nhỏ tải chưa làm trong nhóm này |

## Ca M (chưa chạy) — nhóm nguyên nhân
- Cần khoá global: TC-02-07/07a/07b, TC-03-09 (route QUESTION_GEN → QC fake + fixture), TC-06-20 (restart gateway LLM_MAX_CONCURRENCY=1), TC-02-14/TC-03-08 (OVERLOADED việc nền: không có đường kích hoạt, như lần gốc).
- TC-12-05: seed lớp 1 bị chính ca QC đổi → cần reseed sạch rồi chạy `node scripts/check-practice-seed.mjs` (API_URL, SEED_DEFAULT_PASSWORD).
- Còn lại: dạng tài liệu / E2E chưa có assertion theo TC, ngoài mẫu smoke.
