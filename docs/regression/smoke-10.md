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

## Lượt 2026-11-18 — luật chủ dự án: ca «chặn/hoãn» phải chạy tới kết luận thật (mã dc67bd93, stack ep-qc)
| TC | KQ | Ghi chú |
| --- | --- | --- |
| TC-02-18 | N | (b) «Lighthouse/k6 hoãn» không còn hợp lệ (lệnh chủ dự án 11-18): chạy thật, thu nhỏ tải nếu cần ghi rõ |
| TC-07-25 | N | như TC-02-18 |
| TC-12-17 | N | như TC-02-18 |
| TC-08-08 | K(c) | ngoài phạm vi theo spec (P-10-7 Q16) — giữ |
| TC-11-09 | N | «LT» cũ chỉ ghi L12 (lỗi đã sửa), chưa có P-xx/D-xx duyệt đổi hành vi → chạy lại; kết luận ghi ở dòng sau |
| TC-08-08 | N | không còn nhãn K(c): chạy tới kết luận (P-10-7 Q16 chỉ nêu phạm vi, chưa là quyết định bỏ ca) |

## Lượt 2026-11-18 — kết quả chạy thật (mã dc67bd93, stack ep-qc, lớp riêng QD10A/QD10B; không đụng 761987/761988)
| TC | KQ | Ghi chú |
| --- | --- | --- |
| TC-05-18 | P | SQL practice.sql: PracticePool + PracticeTopics có NOT EXISTS (exam_items ⋈ exams WHERE e.status <> 'PUBLISHED') ở truy vấn; start.go không lọc sau khi lấy; frontend/src/features/practice không có lọc used_in_exams/PUBLISHED |
| TC-12-12 | P | grep '@/mock/practice' + 'mock/questions' trong frontend/src = 0; frontend/src/mock/practice.ts và questions.ts không còn; bash scripts/ui-antipatterns.sh exit 0 |
| TC-12-16 | F | docs/PROGRESS.md @dc67bd93: F12 ☐ (kỳ vọng ☑ sau khi luồng đạt) và P9 «Dev xong, chờ QC» chưa cập nhật theo kết quả QC sprint 10 — PM cập nhật (Thấp, tài liệu) |
| TC-12-18 | P | ARCHITECTURE.md: dòng migration 00019 practice (l.132), route practice thật (l.179, 289); không còn practice_exams; DEMO_SCRIPT chỉ nhắc «các việc Luyện đề» ở bản ghi fake, chưa có bước Luyện đề riêng — TC cho phép «nếu PM đồng ý (ghi)»: ghi nhận |
| TC-12-22 | P | git diff 8ae5393f..HEAD giới hạn đường dẫn practice/quiz/questions/00019 (39 commit): 0 t.Skip/float64 điểm/fetch trần/secret mới; test.skip chỉ lọc theo project playwright (desktop/mobile); contract test chỉ THÊM (0 dòng assert bị xoá), expected_final_grades.csv không đổi; không thêm thư viện cho practice |
| TC-09-11 | P | go test qc9s_signals_test (bản sao tạm trong rig): P (3 SV) có students_affected=3; Q (2 SV) không có; khoá {topic,wrong_answers,students_affected,sample_question_ids}; JSON không chứa id SV; + TestWrongAnswerSignalsNoIdentity PASS |
| TC-09-12 | P | ngoài dữ liệu → rỗng; from>to → len=0 err=nil, không panic (go test qc9s_signals_test) |
| TC-09-13 | P | SV thứ 3 sai Q → Q students_affected=3, P vẫn 3 (go test qc9s_signals_test) |
| TC-10-02 | P | 64 lượt TOPIC/MOCK (8 SV × 8 vòng) chủ đề Y trên QD10A + 40 lượt lần trước: X1,X2,X4,X6 0 lần ở đề / phản hồi PUT answers (rò=0, phản hồi=0); explain không nhắc X1-X4,X6 (4 câu); X5 PUBLISHED xuất hiện 32 lần (được phép) |
| TC-05-07 | P | X1,X2,X3,X4,X6 không xuất hiện trong 40 đề (5 vòng × 8 SV) + đề trọn ngân hàng; X5 có (PUBLISHED) — đếm theo X1..X6 = [0, 0, 0, 0, 27, 0]; rò phản hồi [] |
| TC-05-09 | P | X3: vắng khi OPEN và CLOSED, hiện sau PUBLISHED (đề trọn ngân hàng) — {'OPEN': False, 'CLOSED': False, 'PUBLISHED': True} |
| TC-05-04 | F | ô X xin 6 chỉ có 3: bù từ ô khác, tổng đủ 10, notes mô tả — 201 items=10 F4=3 notes=[] |
| TC-05-05 | P | lượt A cách 10 ngày: 0 câu đúng cũ xuất hiện lại (3 lần dựng, chủ đề 30 câu) — trùng với 5 câu đúng: [0, 0, 0] |
| TC-05-06 | P | lượt A cách 40 ngày (ngoài cửa sổ 30 ngày): câu đúng cũ có thể xuất hiện (xin cả 30 câu → đủ 30, gồm 5 câu đã đúng) — 201 đề 30 câu; trong đó 5/5 câu đúng cũ |
| TC-05-15 | P | SV enrollment PENDING start → 403/404 — 403 |
| TC-06-02 | P | SV chưa có lượt: khuyến nghị = chủ đề có nhiều câu APPROVED nhất — khuyến nghị=QC-r10-A có 30/30 (max=['QC-r10-A']) |
| TC-07-03 | P | duration 9/181, count 4/51 → 422 VALIDATION_FAILED — [(20, 9, 422, 'VALIDATION_FAILED'), (20, 181, 422, 'VALIDATION_FAILED'), (4, 20, 422, 'VALIDATION_FAILED'), (51, 20, 422, 'VALIDATION_FAILED')] |
| TC-07-24 | P | SV khác lớp/cùng lớp: 404 mọi API lượt; staff PUT answers: 403 — sv.gioi [404, 404, 404]; sv lớp khác [403, 403, 403]; GV/TA PUT [403, 403] |
| TC-08-07 | P | 8 lượt MOCK ma trận mặc định: 0 câu ESSAY — ESSAY=0 |
| TC-08-13 | P | hồ sơ SV của GV (profile) không có bài viết / nhận xét luyện — 200 852 ký tự; khoá: ['student', 'risk', 'attendance', 'participation', 'study', 'activity'] |
| TC-10-16 | F | question_id lớp 2 trong PUT answers → 422 INVALID_QUESTION; đọc câu lớp 2 → 403/404 — PUT 422 VALIDATION_FAILED; GET câu 403; GET ds 403 |
| TC-10-17 | P | SV đang có lượt thi OPEN dựng luyện cùng chủ đề: không có câu của bài thi OPEN — exam_attempt=201 start=201 X3=0 |
| TC-09-07 | P | luyện chủ đề W cách 2 ngày: không có việc W; ép 4 ngày: có lại — 2 ngày: 0 việc; 4 ngày: 1 việc |
| TC-09-08 | P | submit lượt TOPIC của chủ đề xong: việc Ôn chủ đề biến mất ≤ 5 s — biến mất sau 1.0 s |
| TC-09-18 | P | SV rời lớp (enrollment REMOVED): history lớp 1 → 403 hoặc rỗng (hành vi thật ghi lại) — trước 1 lượt; sau khi REMOVED: 403 {"code": "FORBIDDEN", "message": "B\u1ea1n kh\u00f4ng c\u00f3 quy\u1ec1n th\u1ef1c hi\u1ec7n thao t\u00e1c n\u00e0y.", "trace_id": "77f447bda13e3491804bffb95bf4 |
| TC-06-07 | F | lượt TOPIC 10 câu: «Câu 1/10 · Đúng 0» + «Thoát về Luyện đề» + không đồng hồ + không tràn ngang ở 1440 và 375; NHƯNG ≥1024 px KHÔNG có danh sách số câu thu gọn (TopicRun không dựng «Danh sách câu»; chỉ MockRun có aside) — [{"w":1440,"hasIdx":true,"hasExit":true,"clock":false,"list":false,"noOverflow":true},{"w":375,"hasIdx":true,"hasExit":true,"clock":false,"list":false,"noOverflow":true}] |
| TC-06-10 | F | offline: «Chưa gửi được» false, phản hồi lộ false; online: phản hồi hiện false; practice_answers của lượt = 0 (kỳ vọng 1) |
| TC-06-21 | F | trước tải lại đã chọn (Kiểm tra bật=true); sau tải lại: «Câu 6/10» true; 5 câu trước đã chấm (DB 5); lựa chọn chưa gửi còn (Kiểm tra bật=false, aria-pressed=false) |
| TC-06-23 | P | chữ: «Luyện đề Lịch sử luyện tập  Giảng viên chưa mở câu hỏi»; nút trong main: [] |
| TC-06-24 | F | lượt 10 câu bằng thao tác chạm ở 375: tràn ngang []; điều khiển <44 px []; thanh hành động sticky/fixed false |
| TC-06-26 | P | quét chữ SV ở 375 (/practice, câu, phản hồi, kết quả, /practice/history): chữ cấm []; «Nguồn trong tài liệu» có false; nút là động từ: «Kiểm tra», «Câu tiếp theo», «Xem kết quả» (TC ghi «Trả lời» — UI dùng «Kiểm tra») |
| TC-07-04 | P | máy khách lệch +5 phút (Date ghi đè): 1440px: hiển thị 19:58 (1198s) vs máy chủ 1197.3s, chênh 0.7s; font-variant-numeric «tabular-nums» / 375px: hiển thị 19:58 (1198s) vs máy chủ 1197.4s, chênh 0.6s; font-variant-numeric «tabular-nums» |
| TC-07-05 | P | PUT answers 2 lần (ms kể từ lúc chọn: [2079,9317]): [{"v":0,"tab":"58efff9d-10bb-4848-82f4-eaec2771b4d4"},{"v":0,"tab":"58efff9d-10bb-4848-82f4-eaec2771b4d4"}]; trạng thái sau 1,2 s «Đang lưu…», sau 7 s «Đã lưu lúc 22:50:44», sau câu 2 «Đã lưu lúc 22:50:52» |
| TC-07-09 | F | offline, chọn 5 câu: thanh dưới «Chưa gửi được 5 câu» (kỳ vọng 5); IndexedDB mục = 0 {}; localStorage ["ep:ui:course=36B","ep:draft:01a1deef-db94-77c8-9b76-05f2df6145d3:practice:01a1ef5a-aa48-7c8c-ae4d-047e1b8c4294=635B"] |
| TC-07-10 | P | online: thanh dưới biến mất true; practice_answers = 5 (kỳ vọng 5); thứ tự gửi ["bb7d","ad0d","3283","6db0","c096"] vs chọn ["bb7d","ad0d","3283","6db0","c096"] |
| TC-07-12 | P | trước ngưỡng «Tự lưu/»; tới 5:00: cảnh báo true (màu lab(30.233 5.52832 2.9892)), dialog 0; tải lại khi còn ~4:00: hiện lại false |
| TC-07-26 | P | 375 MOCK 20 câu: tràn ngang []; điều khiển <44 px []; thanh dưới sticky/fixed true; vị trí/chiều cao đồng hồ khác nhau 1 giá trị |
| TC-07-15 | P | hộp xác nhận hiện 1 lần («Nộp bài thi thử? Bạn đã trả lời 5/5 câu. Nộp xong không sửa được.»), trước khi xác nhận POST submit = 0; xác nhận → POST submit (regex kiểm cũ chỉ chặt hơn chữ «không đảo ngược»; nội dung cảnh báo đủ ý) |
| TC-07-21 | N | lượt UI: sau khi có mạng bài tự nộp (1 POST submit, 1 Idempotency-Key, lượt SUBMITTED 1) nhưng KHÔNG thấy đúng chữ «Chưa nộp được. Đang thử lại.» khi offline và chỉ 1 lần POST (không thấy gửi lại cùng khoá) — cần chạy lại có ghi chữ thật ở thanh dưới |
| TC-07-22 | N | chạy lỗi: API khởi tạo lượt quá thời gian 30 s (DB ep-qc vừa phục hồi sau sự cố Postgres) — chạy lại |

**HANDOFF QC cặp 3 (2026-11-18, dừng theo lệnh chủ dự án):** sprint 10 còn 75 N (xem `python3 docs/regression/evidence-9s/progress9s.py`). Đã chạy xong: API/DB/Go/DIFF nhóm 05-xx,06-02,07-03,07-24,08-07/13,09-07..13/18,10-02/16/17,12-12/16/18/22 và UI SV 06-07/10/21/23/24/26, 07-04/05/09/10/12/15/26. Đang dở/N: 07-21 (cần chạy lại ghi chữ thật), 07-22 (API quá thời gian sau sự cố Postgres), UI GV `/questions` (04-xx) chưa viết spec, nhóm extract/generate/LLM (02-xx, 03-xx, 06-13/14/15/20, 08-05/10/11, 10-10/11/12), Forms (11-xx), seed (12-01..06), UI còn lại (09-02/05/19/20, 10-09, 12-09..15/19/20). Dữ liệu: lớp QD10A (193 câu: 107 bản sao APPROVED từ 761987 đọc-chỉ + chủ đề QC-r10-A/X/Y/W; 12 PENDING; tài liệu «QC S10 Đề mẫu/Đáp án mẫu» READY), QD10B (12 câu), QD10C (rỗng); SV tham gia sv.kha/gioi/nguyco/moi/sv31-37; script: `evidence-9s/s10/` (`mkclass10.py`, `s10a.py`, `s10b.py`, `s10_docs.py`, `rec.py`/`fin.py`, `specs/qc-real-9s-s10-sv.spec.ts`; chạy spec từ `~/Documents/TA_Agent_qc5/frontend`, QC_RES=…/s10/results_m.jsonl); ghi KQ vào smoke bằng `evidence-9s/apply.py smoke-10.md s10/results_m.jsonl`. Lưu ý: sửa DB bằng SQL phải xoá khoá Redis `ep:today:{uid}:{course}`. Không đụng 761987/761988.
