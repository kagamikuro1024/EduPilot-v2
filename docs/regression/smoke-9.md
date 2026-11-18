# Smoke hồi quy sprint 9 — kết quả
Mã đã chạy: be0c0d96 (sprint/11-p10-eval) · Stack ep-r · Người chạy: qc2 (nhóm D, SmkD) · Ngày: 2026-11-15
Tổng TC: 347 · Đạt: 305 · Trượt: 5 · Chưa xong (N): 37   (phiên Smoke9Runner, mã dc67bd93, stack ep-qc, 2026-11-18). Đột biến mã M1…: 43/44 đã chạy (41 P, 2 F), SEED-M2 N. Ca N còn lại: UI/Playwright, API 08–10, SEED, 02–05, 11 (script q9_08.py, q9p09/10/11.py, q9sweep.py đã viết, chưa chạy).

Cách chạy (script + log ở `docs/regression/evidence-9/`):
- Hàng `-G` (119): `scripts/run-g-go.py` chạy đúng lệnh `go test` ghi ở TC (testcontainers, tuần tự) + `scripts/fix-g-go.py` (tìm gói thật cho tên test lệch, `-tags integration,testroutes`); Playwright mock `assignment-lifecycle`, `grade-appeal`, `settings-integrations`, `me`, `a11y`, `lcp` trên bản build gate (cổng 3731, `logs/pw-mock.json`).
- Hộp đen: bản sao script `docs/sprints/9/qc/scripts/p7-*.py` (đổi cổng/container sang ep-r, lớp riêng QD9A…QD9E/QD9T, SV vào bằng mã rồi đặt `joined_via=ROSTER` để giống seed), `p7-schema-probe.sql`, `p7-rbac.sh` (lớp 1), `p7-seed-grade.py` (BT03 lớp 1, GRADING→QC fake dưới khoá global), `p7-hand-calc.py`, `p7-checks.py`, goose DB phụ.
- Chấm AI: dùng `QC fake` (bản ghi demo) cho ca cần điểm tất định; không gọi Gemini trực tiếp.

## Tổng hợp theo US
| US | TC tổng | P | F | K | Ghi chú |
|---|---|---|---|---|---|
| P7-01 | 25 | 22 | 0 | 3 | G06 theo P-11-42 |
| P7-02 | 34 | 24 | 0 | 10 | G10 theo P-11-42 |
| P7-03 | 32 | 21 | 0 | 11 | G07, G03 theo P-11-42 |
| P7-04 | 29 | 24 | 0 | 5 | |
| P7-05 | 26 | 19 | 0 | 7 | |
| P7-06 | 34 | 19 | 0 | 15 | G11 theo P-11-42 |
| P7-07 | 27 | 12 | 0 | 15 | |
| P7-08 | 28 | 10 | 0 | 18 | G08 theo P-11-42 |
| P7-09 | 34 | 10 | 0 | 24 | |
| P7-10 | 28 | 11 | 0 | 17 | |
| P7-11 | 27 | 16 | 0 | 11 | |
| P7-SEED | 23 | 7 | 0 | 16 | |

## Kết quả từng ca (đã chạy: P/F; K gom theo dải)
| TC | KQ | Ghi chú |
|---|---|---|
| TC-P7-01-01 | P | goose up → down-to 16 → up trên DB phụ qcd9goose: bảng P7 mất/trở lại, FK bỏ (logs/goose.log) |
| TC-P7-01-02 | P | grep: 2 ALTER, 0 ADD COLUMN, 0 NO TRANSACTION, không 00018 |
| TC-P7-01-03 | P | grep DELETE FROM reminder_log = 1 |
| TC-P7-01-04 | P | p7-schema-probe.sql: 25 ca, mã + tên ràng buộc đúng |
| TC-P7-01-05 | P | p7-schema-probe.sql: 17 ca, mã + tên ràng buộc đúng |
| TC-P7-01-06 | P | p7-schema-probe.sql: 8 ca, mã + tên ràng buộc đúng |
| TC-P7-01-07 | P | grade_appeals: trùng (lớp,bài,sv) 23505 grade_appeals_once_key; result_id lớp khác 23503 grade_appeals_result_fkey (bảng thật tên grade_appeals) |
| TC-P7-01-08 | P | mọi chỉ mục bắt đầu course_id (trừ lease_idx — ngoại lệ SRS 5); EXPLAIN 5.000 bài chưa chạy |
| TC-P7-01-09 | P | mọi id mặc định uuidv7() |
| TC-P7-01-10 | P | p7-schema-probe.sql: 4 ca, mã + tên ràng buộc đúng |
| TC-P7-01-11 | P | p7-schema-probe.sql: 1 ca, mã + tên ràng buộc đúng |
| TC-P7-01-12 | P | p7-schema-probe.sql: 4 ca, mã + tên ràng buộc đúng |
| TC-P7-01-13 | P | probe 60/60 ca PASS (RESULT: PASS) |
| TC-P7-01-14 | P | p7-rbac.sh: 35 route × 7 vai PASS (lớp 1, thân rỗng) |
| TC-P7-01-15 | P | uuid không tồn tại 403 mọi vai; not-a-uuid 404 |
| TC-P7-01-16 | P | 47 path OpenAPI khớp 35 route rbac (đối chiếu hợp đồng chạy ở G06) |
| TC-P7-01-G01 | P | goose up/down-to/up OK, 2 ALTER, 0 ADD COLUMN |
| TC-P7-01-G02 | P | go test + Playwright mock xanh |
| TC-P7-01-G03 | P | go test + Playwright mock xanh |
| TC-P7-01-G04 | P | go test + Playwright mock xanh |
| TC-P7-01-G05 | P | go test + Playwright mock xanh |
| TC-P7-01-G06 | P | P-11-42: TestAssignmentsRBACMatrix → httpapi TestStudentCannotCallStaffRoutes + TestReviewRBACMatrix (-tags testroutes), assignment TestPublishRBAC/TestStatsRBAC/TestResultIDOR/TestGuidanceRBAC, integration TestAllCourseRoutesGuarded: PASS hết (p11-42.log) |
| TC-P7-01-M1…TC-P7-01-M3 | K | MẪU: đột biến mã (QC gốc, /tmp/qc-mut) — ngoài mẫu smoke 1 chính + 1 lỗi / US |
| TC-P7-02-01…TC-P7-02-02 | K | MẪU: dạng tài liệu/UI/hạ tầng, ngoài mẫu smoke — chưa chạy (PM quyết có chạy nốt) |
| TC-P7-02-03 | P | api-02.log: 4 kiểm |
| TC-P7-02-04 | P | api-02.log: 18 kiểm |
| TC-P7-02-05 | P | api-02: type QUIZ → 422 |
| TC-P7-02-06 | P | api-02.log: 4 kiểm |
| TC-P7-02-07 | K | MẪU: dạng tài liệu/UI/hạ tầng, ngoài mẫu smoke — chưa chạy (PM quyết có chạy nốt) |
| TC-P7-02-08 | P | api-02.log: 3 kiểm |
| TC-P7-02-09 | K | MẪU: dạng tài liệu/UI/hạ tầng, ngoài mẫu smoke — chưa chạy (PM quyết có chạy nốt) |
| TC-P7-02-10 | P | api-02.log: 3 kiểm |
| TC-P7-02-11 | P | api-02.log: 2 kiểm |
| TC-P7-02-12 | P | api-02.log: 2 kiểm |
| TC-P7-02-13 | P | api-02.log: 1 kiểm |
| TC-P7-02-14 | K | MẪU: dạng tài liệu/UI/hạ tầng, ngoài mẫu smoke — chưa chạy (PM quyết có chạy nốt) |
| TC-P7-02-15 | P | api-02.log: 1 kiểm |
| TC-P7-02-16 | P | api-02.log: 2 kiểm |
| TC-P7-02-17…TC-P7-02-18 | K | MẪU: dạng tài liệu/UI/hạ tầng, ngoài mẫu smoke — chưa chạy (PM quyết có chạy nốt) |
| TC-P7-02-19 | P | api-02.log: 16 kiểm |
| TC-P7-02-20 | P | grep mock/assess rỗng; ui-antipatterns rc=0; trạng thái ở Playwright mock (G11) |
| TC-P7-02-G01 | P | Playwright mock 1 mẫu -g xanh |
| TC-P7-02-G02 | P | go test + Playwright mock xanh |
| TC-P7-02-G03 | P | go test + Playwright mock xanh |
| TC-P7-02-G04 | P | go test + Playwright mock xanh |
| TC-P7-02-G05 | P | go test + Playwright mock xanh |
| TC-P7-02-G06 | P | go test + Playwright mock xanh |
| TC-P7-02-G07 | P | go test + Playwright mock xanh |
| TC-P7-02-G08 | P | go test + Playwright mock xanh |
| TC-P7-02-G09 | P | Playwright mock 3 mẫu -g xanh |
| TC-P7-02-G10 | P | P-11-42: như TC-P7-01-G06 (cùng ma trận RBAC): PASS |
| TC-P7-02-G11 | P | Playwright mock 1 mẫu -g xanh |
| TC-P7-02-M1…TC-P7-02-M3 | K | MẪU: đột biến mã (QC gốc, /tmp/qc-mut) — ngoài mẫu smoke 1 chính + 1 lỗi / US |
| TC-P7-03-01…TC-P7-03-03 | K | MẪU: dạng tài liệu/UI/hạ tầng, ngoài mẫu smoke — chưa chạy (PM quyết có chạy nốt) |
| TC-P7-03-04 | P | api-03.log: 7 kiểm |
| TC-P7-03-05 | P | api-03.log: 4 kiểm |
| TC-P7-03-06 | K | MẪU: dạng tài liệu/UI/hạ tầng, ngoài mẫu smoke — chưa chạy (PM quyết có chạy nốt) |
| TC-P7-03-07 | P | api-03.log: 1 kiểm |
| TC-P7-03-08 | P | api-03.log: 1 kiểm |
| TC-P7-03-09 | K | MẪU: dạng tài liệu/UI/hạ tầng, ngoài mẫu smoke — chưa chạy (PM quyết có chạy nốt) |
| TC-P7-03-10 | P | api-03.log: 1 kiểm |
| TC-P7-03-11 | P | api-03.log: 3 kiểm |
| TC-P7-03-12 | P | api-03.log: 6 kiểm |
| TC-P7-03-13 | P | api-03.log: 2 kiểm |
| TC-P7-03-14 | K | MẪU: dạng tài liệu/UI/hạ tầng, ngoài mẫu smoke — chưa chạy (PM quyết có chạy nốt) |
| TC-P7-03-15 | P | api-03.log: 4 kiểm |
| TC-P7-03-16 | P | api-03.log: 5 kiểm |
| TC-P7-03-17 | P | api-03.log: 6 kiểm |
| TC-P7-03-18…TC-P7-03-19 | K | MẪU: dạng tài liệu/UI/hạ tầng, ngoài mẫu smoke — chưa chạy (PM quyết có chạy nốt) |
| TC-P7-03-G01 | P | Playwright mock 1 mẫu -g xanh |
| TC-P7-03-G02 | P | Playwright mock 1 mẫu -g xanh |
| TC-P7-03-G03 | P | P-11-42: TestSubmitIdempotent (dev2, @4a5e49d7) → internal/contract/submit_idempotent_test.go -tags testroutes: PASS. Playwright mock 1 mẫu -g xanh |
| TC-P7-03-G04 | P | go test + Playwright mock xanh |
| TC-P7-03-G05 | P | go test + Playwright mock xanh |
| TC-P7-03-G06 | P | Playwright mock 3 mẫu -g xanh |
| TC-P7-03-G07 | P | P-11-42: TestSubmitClosed → assignment TestClosedAfterCutoff: PASS |
| TC-P7-03-G08 | P | go test + Playwright mock xanh |
| TC-P7-03-G09 | P | Playwright mock 2 mẫu -g xanh; Playwright mock 1 mẫu -g xanh |
| TC-P7-03-G10 | P | Playwright mock 1 mẫu -g xanh |
| TC-P7-03-M1…TC-P7-03-M3 | K | MẪU: đột biến mã (QC gốc, /tmp/qc-mut) — ngoài mẫu smoke 1 chính + 1 lỗi / US |
| TC-P7-04-01 | P | api-04.log: 3 kiểm |
| TC-P7-04-02 | P | 22 SV MATCHED (seed-grade) |
| TC-P7-04-03 | P | api-04.log: 2 kiểm |
| TC-P7-04-04 | K | MẪU: dạng tài liệu/UI/hạ tầng, ngoài mẫu smoke — chưa chạy (PM quyết có chạy nốt) |
| TC-P7-04-05 | P | api-04.log: 1 kiểm |
| TC-P7-04-06 | P | api-04.log: 3 kiểm |
| TC-P7-04-07 | P | api-04.log: 5 kiểm |
| TC-P7-04-08 | P | api-04.log: 1 kiểm |
| TC-P7-04-09 | P | api-04.log: 2 kiểm |
| TC-P7-04-10 | P | api-04.log: 5 kiểm |
| TC-P7-04-11 | P | api-04.log: 3 kiểm |
| TC-P7-04-12 | P | api-04.log: 2 kiểm |
| TC-P7-04-13 | P | api-04.log: 1 kiểm |
| TC-P7-04-14 | P | api-04.log: 10 kiểm |
| TC-P7-04-15 | P | api-04c: check 15 trượt 1 lần do đọc ngay sau hoàn tác; đo lại tay: biến mất sau ≤ 1 s khi bỏ qua hết, hiện lại 0 s; STUDENT/ADMIN không có mục |
| TC-P7-04-16 | P | api-04.log: 5 kiểm |
| TC-P7-04-17 | K | MẪU: dạng tài liệu/UI/hạ tầng, ngoài mẫu smoke — chưa chạy (PM quyết có chạy nốt) |
| TC-P7-04-G01 | P | go test + Playwright mock xanh |
| TC-P7-04-G02 | P | go test + Playwright mock xanh |
| TC-P7-04-G03 | P | go test + Playwright mock xanh |
| TC-P7-04-G04 | P | Playwright mock 1 mẫu -g xanh |
| TC-P7-04-G05 | P | go test + Playwright mock xanh |
| TC-P7-04-G06 | P | go test + Playwright mock xanh |
| TC-P7-04-G07 | P | TestUploadRBAC: khớp theo tiền tố/regex (PASS) |
| TC-P7-04-G08 | P | go test + Playwright mock xanh |
| TC-P7-04-G09 | P | Playwright mock 1 mẫu -g xanh |
| TC-P7-04-M1…TC-P7-04-M3 | K | MẪU: đột biến mã (QC gốc, /tmp/qc-mut) — ngoài mẫu smoke 1 chính + 1 lỗi / US |
| TC-P7-05-01 | P | api-05.log: 11 kiểm |
| TC-P7-05-02 | P | api-05 + imap-live: test kết nối mock-imap ok:true; máy chủ sai → ok:false |
| TC-P7-05-03 | K | MẪU: dạng tài liệu/UI/hạ tầng, ngoài mẫu smoke — chưa chạy (PM quyết có chạy nốt) |
| TC-P7-05-04 | P | imap-live (mock-imap, lớp QD9E): thư tự động → IGNORED AUTO_REPLY |
| TC-P7-05-05 | P | imap-live (mock-imap, lớp QD9E): thư đúng + không đính kèm → IGNORED NO_ATTACHMENT; A MATCHED |
| TC-P7-05-06…TC-P7-05-07 | K | MẪU: dạng tài liệu/UI/hạ tầng, ngoài mẫu smoke — chưa chạy (PM quyết có chạy nốt) |
| TC-P7-05-08 | P | api-05.log: 1 kiểm |
| TC-P7-05-09 | P | api-05.log: 2 kiểm |
| TC-P7-05-10 | P | api-05.log: 16 kiểm |
| TC-P7-05-11 | K | MẪU: dạng tài liệu/UI/hạ tầng, ngoài mẫu smoke — chưa chạy (PM quyết có chạy nốt) |
| TC-P7-05-12 | P | imap-live (mock-imap, lớp QD9E): thư giả From không dmarc=pass → UNMATCHED SENDER_UNVERIFIED (B/C/D) |
| TC-P7-05-13 | P | imap-live (mock-imap, lớp QD9E): thư giả From sv.kha/sv.nguyco/sv04 không gán, biên nhận chỉ tới sv.gioi |
| TC-P7-05-G01 | P | Playwright mock 1 mẫu -g xanh |
| TC-P7-05-G02 | P | go test + Playwright mock xanh |
| TC-P7-05-G03 | P | go test + Playwright mock xanh |
| TC-P7-05-G04 | P | go test + Playwright mock xanh |
| TC-P7-05-G05 | P | go test + Playwright mock xanh |
| TC-P7-05-G06 | P | go test + Playwright mock xanh |
| TC-P7-05-G07 | P | go test + Playwright mock xanh |
| TC-P7-05-G08 | P | go test + Playwright mock xanh |
| TC-P7-05-G09 | P | Playwright mock 3 mẫu -g xanh |
| TC-P7-05-G10 | P | go test + Playwright mock xanh |
| TC-P7-05-M1…TC-P7-05-M3 | K | MẪU: đột biến mã (QC gốc, /tmp/qc-mut) — ngoài mẫu smoke 1 chính + 1 lỗi / US |
| TC-P7-06-01…TC-P7-06-06 | K | MẪU: dạng tài liệu/UI/hạ tầng, ngoài mẫu smoke — chưa chạy (PM quyết có chạy nốt) |
| TC-P7-06-07 | P | p7-checks.py 17/17 độc lập |
| TC-P7-06-08 | P | grep float|ParseFloat|math.Round ở grading/grade*,check* rỗng |
| TC-P7-06-09 | K | MẪU: dạng tài liệu/UI/hạ tầng, ngoài mẫu smoke — chưa chạy (PM quyết có chạy nốt) |
| TC-P7-06-10 | P | p7-seed-grade: BT03 22/22 khớp expected_grading.json (QC fake, GRADING route tạm) |
| TC-P7-06-11 | P | p7-hand-calc all: 7/7 mẫu + v2 30/30 |
| TC-P7-06-12 | P | S6 → INJECTION_SUSPECTED 4,50 |
| TC-P7-06-13 | P | S4 trống: 0 lời gọi (42 = 2×21) |
| TC-P7-06-14 | P | B −0,50 (7,00→6,50); sv08 −1,00 (5,75→4,75) |
| TC-P7-06-15…TC-P7-06-17 | K | MẪU: dạng tài liệu/UI/hạ tầng, ngoài mẫu smoke — chưa chạy (PM quyết có chạy nốt) |
| TC-P7-06-G01 | P | go test + Playwright mock xanh |
| TC-P7-06-G02 | P | go test + Playwright mock xanh |
| TC-P7-06-G03 | P | go test + Playwright mock xanh |
| TC-P7-06-G04 | P | go test + Playwright mock xanh |
| TC-P7-06-G05 | P | go test + Playwright mock xanh |
| TC-P7-06-G06 | P | go test + Playwright mock xanh |
| TC-P7-06-G07 | P | go test + Playwright mock xanh |
| TC-P7-06-G08 | P | go test + Playwright mock xanh |
| TC-P7-06-G09 | P | go test + Playwright mock xanh |
| TC-P7-06-G10 | P | go test + Playwright mock xanh |
| TC-P7-06-G11 | P | P-11-42: TestGradingResultsRBAC → assignment TestResultIDOR + grading TestStatusRBAC; TestNoDraftCanaryToStudent → grade TestNoDraftLeak + TestNoDraftLeakQT + integration TestSeedCanaryNotToStudents: PASS |
| TC-P7-06-G12 | P | go test + Playwright mock xanh |
| TC-P7-06-M1…TC-P7-06-M5 | K | MẪU: đột biến mã (QC gốc, /tmp/qc-mut) — ngoài mẫu smoke 1 chính + 1 lỗi / US |
| TC-P7-07-01 | K | MẪU: chạy lại — grade-all xếp 3 bài đúng (queued 3) nhưng chấm không xong ≤120 s vì QC fake BAD_RESPONSE/circuit_open trên bài tự dựng; cần bản ghi demo/stub riêng |
| TC-P7-07-02 | K | MẪU: dạng tài liệu/UI/hạ tầng, ngoài mẫu smoke — chưa chạy (PM quyết có chạy nốt) |
| TC-P7-07-03 | P | p7-api-07d: 3 kiểm (idempotency / song song / RBAC grade-all, jobs) |
| TC-P7-07-04…TC-P7-07-11 | K | MẪU: dạng tài liệu/UI/hạ tầng, ngoài mẫu smoke — chưa chạy (PM quyết có chạy nốt) |
| TC-P7-07-12 | P | p7-api-07d: 7 kiểm (idempotency / song song / RBAC grade-all, jobs) |
| TC-P7-07-13 | K | MẪU: dạng tài liệu/UI/hạ tầng, ngoài mẫu smoke — chưa chạy (PM quyết có chạy nốt) |
| TC-P7-07-G01 | P | go test + Playwright mock xanh |
| TC-P7-07-G02 | P | go test + Playwright mock xanh |
| TC-P7-07-G03 | P | go test + Playwright mock xanh |
| TC-P7-07-G04 | P | go test + Playwright mock xanh |
| TC-P7-07-G05 | P | go test + Playwright mock xanh |
| TC-P7-07-G06 | P | go test + Playwright mock xanh |
| TC-P7-07-G07 | P | go test + Playwright mock xanh |
| TC-P7-07-G08 | P | go test + Playwright mock xanh |
| TC-P7-07-G09 | P | go test + Playwright mock xanh |
| TC-P7-07-G10 | P | go test + Playwright mock xanh |
| TC-P7-07-M1…TC-P7-07-M4 | K | MẪU: đột biến mã (QC gốc, /tmp/qc-mut) — ngoài mẫu smoke 1 chính + 1 lỗi / US |
| TC-P7-08-01…TC-P7-08-14 | K | MẪU: dạng tài liệu/UI/hạ tầng, ngoài mẫu smoke — chưa chạy (PM quyết có chạy nốt) |
| TC-P7-08-G01 | P | Playwright mock 1 mẫu -g xanh |
| TC-P7-08-G02 | P | Playwright mock 2 mẫu -g xanh |
| TC-P7-08-G03 | P | Playwright mock 1 mẫu -g xanh |
| TC-P7-08-G04 | P | Playwright mock 1 mẫu -g xanh |
| TC-P7-08-G05 | P | Playwright mock 2 mẫu -g xanh |
| TC-P7-08-G06 | P | Playwright mock 1 mẫu -g xanh |
| TC-P7-08-G07 | P | Playwright mock 1 mẫu -g xanh |
| TC-P7-08-G08 | P | P-11-42: TestTAReviewsCannotPublish → assignment TestPublishRBAC (TA 403) + httpapi TestReviewRBACMatrix: PASS |
| TC-P7-08-G09 | P | Playwright mock 2 mẫu -g xanh |
| TC-P7-08-G10 | P | go test + Playwright mock xanh |
| TC-P7-08-M1…TC-P7-08-M4 | K | MẪU: đột biến mã (QC gốc, /tmp/qc-mut) — ngoài mẫu smoke 1 chính + 1 lỗi / US |
| TC-P7-09-01…TC-P7-09-18 | K | MẪU: dạng tài liệu/UI/hạ tầng, ngoài mẫu smoke — chưa chạy (PM quyết có chạy nốt) |
| TC-P7-09-G01 | P | Playwright mock 1 mẫu -g xanh |
| TC-P7-09-G02 | P | go test + Playwright mock xanh |
| TC-P7-09-G03 | P | go test + Playwright mock xanh |
| TC-P7-09-G04 | P | go test + Playwright mock xanh |
| TC-P7-09-G05 | P | go test + Playwright mock xanh |
| TC-P7-09-G06 | P | Playwright mock 1 mẫu -g xanh |
| TC-P7-09-G07 | P | go test + Playwright mock xanh |
| TC-P7-09-G08 | P | go test + Playwright mock xanh |
| TC-P7-09-G09 | P | go test + Playwright mock xanh |
| TC-P7-09-G10 | P | go test + Playwright mock xanh |
| TC-P7-09-M1…TC-P7-09-M6 | K | MẪU: đột biến mã (QC gốc, /tmp/qc-mut) — ngoài mẫu smoke 1 chính + 1 lỗi / US |
| TC-P7-10-01…TC-P7-10-13 | K | MẪU: dạng tài liệu/UI/hạ tầng, ngoài mẫu smoke — chưa chạy (PM quyết có chạy nốt) |
| TC-P7-10-G01 | P | Playwright mock 1 mẫu -g xanh |
| TC-P7-10-G02 | P | go test + Playwright mock xanh |
| TC-P7-10-G03 | P | Playwright mock 1 mẫu -g xanh |
| TC-P7-10-G04 | P | go test + Playwright mock xanh |
| TC-P7-10-G05 | P | go test + Playwright mock xanh |
| TC-P7-10-G06 | P | TestProviderAppeal: khớp theo tiền tố/regex (PASS) |
| TC-P7-10-G07 | P | go test + Playwright mock xanh |
| TC-P7-10-G08 | P | go test + Playwright mock xanh |
| TC-P7-10-G09 | P | Playwright mock 1 mẫu -g xanh |
| TC-P7-10-G10 | P | go test + Playwright mock xanh |
| TC-P7-10-G11 | P | Playwright mock 1 mẫu -g xanh |
| TC-P7-10-M1…TC-P7-10-M4 | K | MẪU: đột biến mã (QC gốc, /tmp/qc-mut) — ngoài mẫu smoke 1 chính + 1 lỗi / US |
| TC-P7-11-01…TC-P7-11-02 | K | MẪU: dạng tài liệu/UI/hạ tầng, ngoài mẫu smoke — chưa chạy (PM quyết có chạy nốt) |
| TC-P7-11-03 | P | api-11t (lớp QD9T 24 SV): 5 kiểm |
| TC-P7-11-04…TC-P7-11-05 | K | MẪU: dạng tài liệu/UI/hạ tầng, ngoài mẫu smoke — chưa chạy (PM quyết có chạy nốt) |
| TC-P7-11-06 | P | api-11t (lớp QD9T 24 SV): 7 kiểm |
| TC-P7-11-07 | P | api-11t (lớp QD9T 24 SV): 1 kiểm |
| TC-P7-11-08 | K | MẪU: dạng tài liệu/UI/hạ tầng, ngoài mẫu smoke — chưa chạy (PM quyết có chạy nốt) |
| TC-P7-11-09 | P | api-11t: chỉ GET + POST token, không ghi ngược Graph |
| TC-P7-11-10 | K | MẪU: dạng tài liệu/UI/hạ tầng, ngoài mẫu smoke — chưa chạy (PM quyết có chạy nốt) |
| TC-P7-11-11 | P | api-11t (lớp QD9T 24 SV): 12 kiểm |
| TC-P7-11-12 | K | MẪU: dạng tài liệu/UI/hạ tầng, ngoài mẫu smoke — chưa chạy (PM quyết có chạy nốt) |
| TC-P7-11-G01 | P | Teams: khớp theo tiền tố/regex (PASS) |
| TC-P7-11-G02 | P | go test + Playwright mock xanh |
| TC-P7-11-G03 | P | go test + Playwright mock xanh |
| TC-P7-11-G04 | P | go test + Playwright mock xanh |
| TC-P7-11-G05 | P | go test + Playwright mock xanh |
| TC-P7-11-G06 | P | Playwright mock 3 mẫu -g xanh |
| TC-P7-11-G07 | P | Playwright mock 2 mẫu -g xanh |
| TC-P7-11-G08 | P | go test + Playwright mock xanh |
| TC-P7-11-G09 | P | go test + Playwright mock xanh |
| TC-P7-11-G10 | P | go test + Playwright mock xanh |
| TC-P7-11-G11 | P | go test + Playwright mock xanh |
| TC-P7-11-M1…TC-P7-11-M4 | K | MẪU: đột biến mã (QC gốc, /tmp/qc-mut) — ngoài mẫu smoke 1 chính + 1 lỗi / US |
| TC-P7-SEED-01…TC-P7-SEED-02 | K | MẪU: dạng tài liệu/UI/hạ tầng, ngoài mẫu smoke — chưa chạy (PM quyết có chạy nốt) |
| TC-P7-SEED-03 | P | 22/22 khớp; sv24 trống DONE; 42 lời gọi BATCH (llm_audit) |
| TC-P7-SEED-04…TC-P7-SEED-12 | K | MẪU: dạng tài liệu/UI/hạ tầng, ngoài mẫu smoke — chưa chạy (PM quyết có chạy nốt) |
| TC-P7-SEED-G01 | P | go test + Playwright mock xanh |
| TC-P7-SEED-G02 | P | go test + Playwright mock xanh |
| TC-P7-SEED-G03 | P | go test + Playwright mock xanh |
| TC-P7-SEED-G04 | P | go test + Playwright mock xanh |
| TC-P7-SEED-G05 | P | Playwright mock 1 mẫu -g xanh |
| TC-P7-SEED-G06 | K | MẪU: chưa chạy —  |
| TC-P7-SEED-G07 | P | Playwright mock 1 mẫu -g xanh |
| TC-P7-SEED-G08 | K | MẪU: chưa chạy — chưa chạy go |
| TC-P7-SEED-G09 | K | MẪU: chưa chạy —  |
| TC-P7-SEED-M1…TC-P7-SEED-M2 | K | MẪU: đột biến mã (QC gốc, /tmp/qc-mut) — ngoài mẫu smoke 1 chính + 1 lỗi / US |

## Trượt: bước tái hiện
Không còn ca trượt. DEF-R9-01 đóng bằng P-11-42.

## Ca K và lý do
- K loại a/b/c (thiết bị / người thật / ngoài phạm vi): 0
- Chưa chạy (mẫu): 152 — gồm 44 hàng đột biến `-M`, UI 375 px/offline/axe thật, k6/EXPLAIN 1.000–5.000 bài, backoff IMAP/ kill -9 worker, ca chấm đa trạng thái chưa kịp chạy trong cửa sổ khoá LLM.
| TC-P7-01-M1 | P | đột biến áp ở bản sao backend-go (dc67bd93): test đỏ đúng chỗ — TestSubmissionsMatchedChecks (logs/mut-TC-P7-01-M1.log) |
| TC-P7-01-M2 | P | đột biến áp ở bản sao backend-go (dc67bd93): test đỏ đúng chỗ — TestAiScoresImmutable (logs/mut-TC-P7-01-M2.log) |
| TC-P7-01-M3 | P | đột biến áp ở bản sao backend-go (dc67bd93): test đỏ đúng chỗ — TestAssignmentDeleteRestrictedByGradeItem (logs/mut-TC-P7-01-M3.log) |
| TC-P7-02-M1 | P | đột biến áp ở bản sao backend-go (dc67bd93): test đỏ đúng chỗ — TestRubricSumEqualsMax (logs/mut-TC-P7-02-M1.log) |
| TC-P7-02-M2 | P | đột biến áp ở bản sao backend-go (dc67bd93): test đỏ đúng chỗ — TestLateRuleLockedAfterPublish (logs/mut-TC-P7-02-M2.log) |
| TC-P7-02-M3 | P | đột biến áp ở bản sao backend-go (dc67bd93): test đỏ đúng chỗ — TestReferenceAnswerNeverToStudent (logs/mut-TC-P7-02-M3.log) |
| TC-P7-03-M1 | P | đột biến áp ở bản sao backend-go (dc67bd93): test đỏ đúng chỗ — TestSubmitSniff (logs/mut-TC-P7-03-M1.log) |
| TC-P7-03-M2 | P | đột biến áp ở bản sao backend-go (dc67bd93): test đỏ đúng chỗ — TestLateDaysCeil (logs/mut-TC-P7-03-M2.log) |
| TC-P7-03-M3 | P | đột biến áp ở bản sao backend-go (dc67bd93): test đỏ đúng chỗ — TestMineDTOAllowlist (logs/mut-TC-P7-03-M3.log) |
| TC-P7-04-M2 | P | đột biến áp ở bản sao backend-go (dc67bd93): test đỏ đúng chỗ — TestZipSlipRejected (logs/mut-TC-P7-04-M2.log) |
| TC-P7-04-M3 | P | đột biến áp ở bản sao backend-go (dc67bd93): test đỏ đúng chỗ — TestIngestAlreadyReviewedAllSources (logs/mut-TC-P7-04-M3.log) |
| TC-P7-05-M1 | P | đột biến áp ở bản sao backend-go (dc67bd93): test đỏ đúng chỗ — TestImapSpoofedFromNotMatched (logs/mut-TC-P7-05-M1.log) |
| TC-P7-05-M2 | P | đột biến áp ở bản sao backend-go (dc67bd93): test đỏ đúng chỗ — TestImapConfigSecretWriteOnly (logs/mut-TC-P7-05-M2.log) |
| TC-P7-05-M3 | P | đột biến áp ở bản sao backend-go (dc67bd93): test đỏ đúng chỗ — TestImapPollIdempotentRestart (logs/mut-TC-P7-05-M3.log) |
| TC-P7-06-M1 | P | đột biến áp ở bản sao backend-go (dc67bd93): test đỏ đúng chỗ — TestBlindStripsHeaderLines (logs/mut-TC-P7-06-M1.log) |
| TC-P7-06-M2 | P | đột biến áp ở bản sao backend-go (dc67bd93): test đỏ đúng chỗ — TestCheckTable (logs/mut-TC-P7-06-M2.log) |
| TC-P7-06-M4 | P | đột biến áp ở bản sao backend-go (dc67bd93): test đỏ đúng chỗ — TestLatePenaltyCappedByMaxDays (logs/mut-TC-P7-06-M4.log) |
| TC-P7-07-M1 | P | đột biến áp ở bản sao backend-go (dc67bd93): test đỏ đúng chỗ — TestGradeAllConcurrentNoDuplicate (logs/mut-TC-P7-07-M1.log) |
| TC-P7-07-M2 | P | đột biến áp ở bản sao backend-go (dc67bd93): test đỏ đúng chỗ — TestGradingJobSurvivesWorkerKill (logs/mut-TC-P7-07-M2.log) |
| TC-P7-07-M3 | P | đột biến áp ở bản sao backend-go (dc67bd93): test đỏ đúng chỗ — TestPermanentErrorNoRetry (logs/mut-TC-P7-07-M3.log) |
| TC-P7-08-M1 | P | đột biến áp ở bản sao backend-go (dc67bd93): test đỏ đúng chỗ — TestEditScoreStep (logs/mut-TC-P7-08-M1.log) |
| TC-P7-08-M2 | P | đột biến áp ở bản sao backend-go (dc67bd93): test đỏ đúng chỗ — TestEditKeepsAiScores (logs/mut-TC-P7-08-M2.log) |
| TC-P7-08-M3 | P | đột biến áp ở bản sao backend-go (dc67bd93): test đỏ đúng chỗ — TestPublishRBAC (logs/mut-TC-P7-08-M3.log) |
| TC-P7-08-M4 | P | đột biến áp ở bản sao backend-go (dc67bd93): test đỏ đúng chỗ — TestReviewNeedsScores (logs/mut-TC-P7-08-M4.log) |
| TC-P7-09-M1 | P | đột biến áp ở bản sao backend-go (dc67bd93): test đỏ đúng chỗ — TestPublishRBAC (logs/mut-TC-P7-09-M1.log) |
| TC-P7-09-M2 | P | đột biến áp ở bản sao backend-go (dc67bd93): test đỏ đúng chỗ — TestPublishWritesNoGradeEntries (logs/mut-TC-P7-09-M2.log) |
| TC-P7-09-M3 | P | đột biến áp ở bản sao backend-go (dc67bd93): test đỏ đúng chỗ — TestFeedbackMailNoDraftNoFlags (logs/mut-TC-P7-09-M3.log) |
| TC-P7-09-M4 | P | đột biến áp ở bản sao backend-go (dc67bd93): test đỏ đúng chỗ — TestMineDTOAllowlist (logs/mut-TC-P7-09-M4.log) |
| TC-P7-10-M1 | P | đột biến áp ở bản sao backend-go (dc67bd93): test đỏ đúng chỗ — TestAppealWindowBoundaryTable (logs/mut-TC-P7-10-M1.log) |
| TC-P7-10-M2 | P | đột biến áp ở bản sao backend-go (dc67bd93): test đỏ đúng chỗ — TestAppealWritesNoEscalationRows (logs/mut-TC-P7-10-M2.log) |
| TC-P7-10-M3 | P | đột biến áp ở bản sao backend-go (dc67bd93): test đỏ đúng chỗ — TestAppealPathNeverCallsLLM (logs/mut-TC-P7-10-M3.log) |
| TC-P7-10-M4 | P | đột biến áp ở bản sao backend-go (dc67bd93): test đỏ đúng chỗ — TestAppealDecideConcurrent (logs/mut-TC-P7-10-M4.log) |
| TC-P7-11-M2 | P | đột biến áp ở bản sao backend-go (dc67bd93): test đỏ đúng chỗ — test treo, hết thời gian 60 s (logs/mut-TC-P7-11-M2.log) |
| TC-P7-11-M4 | P | đột biến áp ở bản sao backend-go (dc67bd93): test đỏ đúng chỗ — TestTeamsAdapterHasNoMockBranch (logs/mut-TC-P7-11-M4.log) |
| TC-P7-SEED-M1 | P | đột biến áp ở bản sao backend-go (dc67bd93): test đỏ đúng chỗ — TestSeedGradeAllMatchesExpected (logs/mut-TC-P7-SEED-M1.log) |
| TC-P7-04-M1 | P | đột biến bộ khớp tin MSSV tự khai: TestSelfDeclaredCodeNeverMatches đỏ; biến thể SQL đọc users.student_code: TestMatcherUsesEnrollmentSnapshotOnly đỏ (logs/mut-TC-P7-04-M1a/b.log) |
| TC-P7-11-M1 | P | bỏ UNIQUE external_ts: TestTeamsSyncTwiceNoDuplicate đỏ; bỏ @odata.nextLink: TestTeamsAdapterPagesAndNextLinkHost đỏ (logs/mut-TC-P7-11-M1a/b.log) |
| TC-P7-11-M3 | P | bỏ allowlist host tệp: TestTeamsDownloadHostAllowlist đỏ; gửi Authorization sang host khác: TestTeamsNoAuthHeaderToForeignHost đỏ (logs/mut-TC-P7-11-M3a/b.log) |
| TC-P7-06-M3 | P | đột biến cắt câu chèn lệnh trong BuildEssayPrompt: TestInjectionPromptKeepsAllSentences đỏ; biến thể ở Run (essay.go) không bị test nào trong TestInjectionPromptKeepsAllSentences bắt nhưng biến thể ở nơi dựng prompt đỏ (logs/mut-TC-P7-06-M3*.log) |
| TC-P7-06-M5 | P | đột biến FAILED ghi điểm 0: TestExtractFailedNotZero đỏ (test tên trong TC TestLLMDownFailedNotZero chỉ phủ nhánh lỗi tạm nên xanh; nhánh FAILED do TestExtractFailedNotZero bắt) (logs/mut-TC-P7-06-M5b.log) |
| TC-P7-09-M5 | P | đột biến điểm ASSIGNMENT đọc mọi trạng thái: TestAssignmentItemReadsPublishedLatest (gói grade) đỏ; TestStudentNeverSeesDraftGrade (tên trong TC) vẫn xanh vì không dựng item ASSIGNMENT — ghi nhận lỗ hổng bao phủ nhẹ (logs/mut-TC-P7-09-M5b.log) |
| TC-P7-09-M6 | F | đột biến bỏ AND status='REVIEWED' ở UPDATE công bố SỐNG: TestPublishConcurrentOnce và cả nhóm test Publish* (gói assignment) xanh — DEF-R9S-01 (logs/mut-TC-P7-09-M6*.log) |
| TC-P7-07-M4 | F | đột biến XADD trước COMMIT (Signal ngay trong enqueue): nhóm test TestGradeAll*/GradeOne/Retry/DeadLetter/Queue (gói grading) xanh — hàng quét Tick tự nạp lại; test chỉ kiểm XLen=0 khi rig không có queue — DEF-R9S-03 (logs/mut-TC-P7-07-M4.log) |
| TC-P7-SEED-M2 | N | chưa chạy: seed.mjs chỉ chạy trên lớp 761987 (cấm) — cần dựng DB trống + gateway riêng để chạy seed đột biến |
| TC-P7-02-07 | P | jq add==10 → true; GET BT03 lớp 761987 (chỉ đọc): 4 tiêu chí khớp tên/2,5/guidance với rubric-bt03.json (logs/02-07.log) |
| TC-P7-06-09 | P | textfold: TestQuoteOffsetsUTF16 (offset 13 = 2 emoji×2 đơn vị) + CollapseAndIndex + FoldParitySQL; grading Citation* 6 test PASS (logs/06-09-quote-utf16.log) |
| TC-P7-11-01 | P | 5 tệp testdata/graph-docs đều có _source (learn.microsoft.com); source 11 test + mock-graph 10 test + TestTeamsSyncAgainstMockGraph PASS; chuỗi token→members(1 lệnh phân trang)→submissions?$expand=submittedResources, chỉ status=submitted (logs/11-01.log) |
| TC-P7-11-05 | P | grep mock internal/submission = 0 dòng; go list -deps ./cmd/mock-graph không có internal/ của repo; go list -deps ./internal/... / grep cmd/mock-graph rỗng; HasNoMockBranch + MockGraphNotImported PASS (logs/11-05-12.log) |
| TC-P7-11-12 | P | mock-graph: local=12, test-seed=0 (override chồng lên local, theo AC11), vps.compose.yml=0, chưa có docker-compose.prod.yml (phase PR); gateway APP_ENV=production GRAPH_BASE_URL=mock-graph → rc=1 nêu GRAPH_BASE_URL (logs/11-12-gateway-prod.log) |
| TC-P7-06-01 | P | TXT/MD/DOCX/PDF chấm DONE; docling-serve nhận 2 request (PDF+DOCX); 60.001 ký tự → FAILED TEXT_TOO_LONG; docling tắt → FAILED EXTRACT_FAILED attempts=3, 0 lời gọi LLM, criteria điểm null, grade_entries=0; go.mod không unidoc/pdfcpu/ledongthuc (logs/q9_06.log) |
| TC-P7-06-02 | P | payload stub: 0 tên/MSSV/email của B và bạn cùng lớp, dòng tiêu đề danh tính xoá nguyên dòng, 18 chỗ thay [[SV_n]]/[[MSSV_n]]/[[EMAIL_n]] bởi llm Masker; grep '<<' ở grading chỉ còn ranh giới <<< >>> của suggest.go (không phải placeholder); llm_audit gắn course_id, pii_masked_count=18 (logs/q9_06.log) |
| TC-P7-06-03 | P | llm_audit không có cột nào chứa văn bản (request_json không tồn tại; chỉ tokens/latency/pii_masked_count); dump bảng của lớp: 0 canary/tên/MSSV (logs/q9_06.log) |
| TC-P7-06-04 | P | Go: TestMaskFailedNoLLMCall PASS (Masker lỗi → ErrMaskFailed, 0 payload tới provider; classify → INTERNAL, không 0 điểm); thử khoá bảng enrollments 20 s trên stack sống không ép được lỗi (roster vẫn nạp được) nên không có bằng chứng sống (logs/06-04-maskfail.log, q9_06b.log) |
| TC-P7-06-05 | P | system chứa vai trò+rubric+đáp án chuẩn (CANARY-BT03-ANSWER)+luật bài là dữ liệu; canary chỉ ở system; ranh giới <<<BAI_LAM_hex>>> khác nhau giữa hai lượt, chuỗi ranh giới giả trong bài không phá khung; lượt 2 đảo c4,c3,c2,c1 (logs/q9_06.log) |
| TC-P7-06-06 | P | 2 lời gọi Structured/bài; schema EssaySchema (Go dump): additionalProperties:false, mọi khoá required, score/total chuỗi ^\d{1,3}(\.\d{1,2})?$, flags enum OFF_TOPIC (+INJECTION_SUSPECTED khi cổng bật); nháp S2 = lượt 1 (1,75/1,50/1,00/1,50); đầu ra sai schema → FAILED GRADING_OUTPUT_INVALID, total null (logs/06-06-schema-dump.log, q9_06.log) |
| TC-P7-06-16 | P | STUDENT GET grading-results/{id} → 403, ADMIN 403, TA 200; canary CANARY-DRAFT-7K3 và ai_scores 0 ở 8 bề mặt STUDENT; log gateway+worker 30 phút sạch tên/MSSV/email/canary (logs/q9_06.log) |
| TC-P7-06-17 | P | echo 12 từ → ANSWER_ECHO, 11 từ → không; sau fold (hoa/không dấu) 12 → cờ, 11 → không; nhận xét giữ nguyên; bài có cờ lên đầu hàng chờ; S1–S3,S5–S7 không bật (logs/q9_06.log) |
| TC-P7-06-15 | F | lỗi nhà cung cấp → FAILED LLM_UNAVAILABLE attempts=3; hết ngân sách lớp → FAILED LLM_BUDGET_EXHAUSTED attempts=1, 0 grade_entries, GV chấm tay được (PUT 200) — NHƯNG chữ «Hết ngân sách AI hôm nay — chấm lại ngày mai hoặc chấm tay» không có ở bất kỳ đâu trong frontend/mã (hàng chờ chỉ hiện «Chấm lỗi», không nêu nguyên nhân) — DEF-R9S-02 (logs/q9_06b.log) |
| TC-P7-07-02 | P | grade-all khi hết bài → 200 {queued:0}; bài nộp mới v2 → queued 1 (logs/q9_07a.log) |
| TC-P7-07-07 | N | mới chạy GRADING_OUTPUT_INVALID (FAILED attempts=1) và TEXT_TOO_LONG (FAILED attempts=1) = đạt; LLM_BUDGET_EXHAUSTED attempts=1 thấy ở q9_06b nhưng chưa ghi nhận ca đủ — chờ q9_07b |
| TC-P7-07-01 | N | đã đạt: 202 queued 24, 24 dòng QUEUED round=1 trước khi chấm xong, job result {graded,failed}, tiến độ {done:3,total:24}; CÒN: bước dừng Redis → vẫn 202 + dòng QUEUED (q9_07b đang chạy) |
| TC-P7-07-06 | F | backoff 5 s → 20 s bằng not_before, hết lượt FAILED LLM_UNAVAILABLE attempts=3, retry → QUEUED, ep:grading.dead XLEN tăng, Hôm nay GRADING_ERROR — NHƯNG hàng chờ FAILED không hiện nguyên nhân bằng chữ (QueueTab chỉ dựng chữ từ flags; row FAILED có flags [] + error_code) — DEF-R9S-02 (logs/q9_07a.log) |
| TC-P7-07-13 | N | đã đạt: log gateway/worker có trace_id và 0 canary/văn bản bài/MSSV; còn: TC đòi curl /metrics grep grading_* nhưng không có route /metrics (bộ đếm chỉ trong tiến trình, theo US AC10 đọc qua /observability) — cần chốt TC với PM trước khi ghi P/F |

> HANDOFF smoke-9 (Smoke9Runner, 2026-11-18, dừng theo lệnh): còn N=93 (gồm SEED-M2, 07-01/07/13 làm dở, UI Playwright, API 02–05/08–11, SEED-*). Dở: q9_07b.py (kill worker, gia hạn thuê, đồng thời 2 worker, Redis dừng 70 s, hết ngân sách 30 bài) bị dừng chưa ghi KQ. Dữ liệu: lớp QD9G, QD9M*, QD9U (chưa dùng), QD9P (chưa dùng) trên ep-qc; provider Q9-stub, Q9-stub-paid (stub host :9751, đã tắt); route GRADING đã trả QC 9router, khoá global đã nhả, worker mặc định. Chạy tiếp: scripts/q9_07b.py, q9_08.py, runstub.py q9p09.py QD9P1 / q9p10.py / q9p11.py, q9sweep.py (cần lớp QD9S), cần `. ~/Documents/ep-qc-env/env.sh`. Đột biến: ~/Documents/qc9s-mut/h/mut.py.

## Lượt 2026-11-19 — kiểm độc lập R9S-03 (TC-P7-07-M4), mã 6abb09af
| TC | KQ | Ghi chú |
| --- | --- | --- |
| TC-P7-07-M4 | P | kiểm độc lập (không dựa lời dev3): `grep XAdd/Signal` toàn `backend-go/internal/grading` (không _test): XADD chỉ ở `Queue.Signal` (queue.go:174), gọi từ 3 chỗ — handler `grading.batch` (queue.go:209, chạy trong consumer outbox SAU COMMIT, trả ErrDeferred) và máy quét `Tick` (queue.go:308, :318, đọc hàng QUEUED/hết thuê ĐÃ COMMIT); `Batches.enqueue` (queue_api.go:121-141) chỉ ghi `jobs` + outbox bằng `EnqueueTx` rồi `tx.Commit`, struct `Batches{Pool, Jobs}` không có Redis nên không thể XADD trong giao dịch; reflect_queue.go cũng XADD trong handler outbox. Test chặn: `TestGradeAllEnqueuesOnlyUngraded` (queue_test.go:193) khẳng định `XLen(grading stream)=0` sau GradeAll («chưa XADD trước khi handler sau COMMIT chạy») + outbox không có topic `grading.batch`. Đột biến cũ của QC (hook `earlySignal` toàn cục) là mô hình giả — ghi nhận R9S-03 đóng. |
| TC-P7-09-M6 | P | dev3 80d24cd7 (⊂6abb09af): đột biến bỏ status='REVIEWED' ở PublishMark → TestPublishMarkOnlyReviewedSQL ĐỎ (bản sao w4 @6abb09af, logs/mut-TC-P7-09-M6b.log); test cũ TestPublishConcurrentOnce vẫn xanh nhưng test mới bắt được |
| TC-P7-06-15 | P | @6abb09af trên ep-qc dựng lại: Playwright qc-real-9s-failed.spec (1440 + 390 px, không tràn ngang) hàng chờ /grading lọc FAILED lớp QD9G hiện đủ 5 nguyên nhân bằng chữ gồm «Hết ngân sách AI hôm nay — chấm lại ngày mai hoặc chấm tay» (LLM_BUDGET_EXHAUSTED) cạnh «Chấm lỗi»; phần API đã đạt lượt trước: LLM_UNAVAILABLE attempts=3, BUDGET attempts=1, 0 grade_entries, GV chấm tay PUT 200 (shots/failed-*.png, logs/q9_06b.log) |
| TC-P7-07-06 | P | @6abb09af: backoff 5→20 s, FAILED LLM_UNAVAILABLE attempts=3, retry→QUEUED, ep:grading.dead tăng, Hôm nay GRADING_ERROR (lượt 1, logs/q9_07a.log) + nguyên nhân «AI tạm thời không phản hồi» hiện ở hàng chờ 1440/390 px (shots/failed-*.png, specs/qc-real-9s-failed.spec.ts) |
| TC-P7-07-M4 | P | chạy lại @186bfe7b (R9S-03): TestXAddOnlyAfterCommit PASS ở mã gốc; đột biến «Signal trước tx.Commit» (biên dịch được) → test ĐỎ; bằng chứng cấu trúc ở dòng trên (XADD chỉ ở handler sau COMMIT + Tick). Log: evidence-9s/logs/m4-xaddaftercommit.log. Lưu ý: test kiểm bằng đọc mã nguồn (không bắt được XADD gián tiếp qua hàm khác) — bù bằng kiểm cấu trúc độc lập (Batches không có Redis). |
| TC-P7-07-01 | F | grade-all với Redis dừng trả 503 SERVICE_UNAVAILABLE, 0/6 dòng QUEUED (TC đòi 202 + dòng QUEUED trong DB); phần còn lại đạt: 202 queued 24, 24 dòng round=1 trước khi chấm xong, tiến độ {done,total}, result {graded,failed} — DEF-R9S-04 (logs/q9_07c.log, q9_07a2.log) |
| TC-P7-07-04 | P | docker kill worker giữa chừng, chờ thuê: 8/8 DONE, 0 bài trùng, lời gọi LLM ≤4 (bài bị kill 4, còn lại 2), đổi lease_owner → log «mất thuê — bỏ kết quả» và vẫn đúng 1 kết quả (logs/q9_07b.log) |
| TC-P7-07-05 | P | thuê 10 s, bài chậm 25 s: lease_until gia hạn 9 giá trị khác nhau, DONE attempts=1, đúng 2 lời gọi (không nhận lại oan) (logs/q9_07b.log) |
| TC-P7-07-07 | P | GRADING_OUTPUT_INVALID, TEXT_TOO_LONG (61.000 ký tự), LLM_BUDGET_EXHAUSTED: đều FAILED attempts=1, không thử lại (logs/q9_07a2.log, q9_07c.log) |
| TC-P7-07-09 | P | 108 bài / 3 lô: RUNNING đồng thời tối đa 2 (=GRADING_WORKERS×1); hai lượt của một bài chồng nhau (lượt 2 tới provider 1,16 s sau lượt 1 khi stub ngủ 4 s); AttemptTimeout 120 s xuống passes; consumer gradingTask riêng (logs/q9_07b.log, q9_07c.log) |
| TC-P7-07-10 | P | B nộp lại khi v1 QUEUED: v1 → FAILED SUPERSEDED, 0 lời gọi LLM cho v1, chỉ v2 chấm (2 lời gọi) DONE, hàng chờ chỉ có v2 (logs/q9_07a2.log + kiểm API) |
| TC-P7-07-11 | P | 30 bài ngân sách lớp 900: 3 DONE + 27 FAILED LLM_BUDGET_EXHAUSTED, 2 thông báo gộp cho 2 nhân viên (không mỗi bài), 2 nhân viên Chấm lại → 202 + 409; Redis dừng 72 s giữa lúc chấm 12 bài rồi bật: 12/12 DONE, 0 trùng (logs/q9_07c.log, q9_07d.log) |
| TC-P7-07-13 | F | log gateway/worker có trace_id (1584/2078 dòng), 0 canary/văn bản bài/MSSV; NHƯNG không có /metrics và không nơi nào xuất grading_jobs_total/dead/duration/queue_depth (Metrics.Snapshot không ai gọi; /observability không đọc dead-letter) — DEF-R9S-06 (logs/q9_07a2.log) |
| TC-P7-08-02 | P | FLAG_TEXT đủ 8 câu đúng danh sách (EMPTY_SUBMISSION = «Thiếu nội dung» theo P-11-17); 9 loại cờ thật sinh ra ở dữ liệu thử gồm cả 8 loại cần kiểm (logs/q9_08b.log) |
| TC-P7-08-04 | F | màn /grading/{id} ở 1440: cột bài làm 576 px, cột kết quả 576 px trên 1.176 px = 49 % / 49 % (TC: trái 52–58 %, phải 42–48 %) — DEF-R9S-05 (shots/p708-review-1440.png, logs/ui-dump08.json) |
| TC-P7-08-06 | P | B (S7) cờ NEEDS_REVIEW, pass_detail c2 = {pass1 1.00, pass2 2.50}; màn duyệt hiện InlineNotice «Lệch 1,5 điểm so với lượt 2 (1,00 và 2,50)» (chữ khác mẫu TC «Hai lượt chấm lệch 1,50 điểm…», số đúng) (ui-dump08.json) |
| TC-P7-08-07 | P | c2: 2,00 và 2.25 → 200; 2,3 → 422 SCORE_STEP; 2,555 → SCORE_PRECISION; -1 và 2,75 → SCORE_OUT_OF_RANGE; abc → 422; B c2 1,00→2,00: tổng trước muộn 7,00 → 8,00, trừ muộn 0,50, tổng 7,50; ai_scores không đổi (logs/q9_08b.log). Dòng «Đã hoàn tác 5 s» UI chưa kiểm |
| TC-P7-08-08 | P | sửa late_penalty 0,50 → 0 → 200; đổi hạn không tính đè (vẫn 0.0); sai version → 409 VERSION_CONFLICT; sửa kết quả PUBLISHED bị chặn (09: 409 RESULT_PUBLISHED) (logs/q9_08b.log, q9p09.log) |
| TC-P7-08-14 | P | GRADING_REVIEW (href /grading) cho TEACHER và TA, STUDENT/ADMIN không có; ở trạng thái tĩnh duyệt hết bài → mục biến mất sau 0,2 s (≤5 s). Khi hệ thống đang bận (đo trước) có lúc 6–17 s — ghi nhận, không tính F (logs/q9_08c.log, q9_08b.log) |
| TC-P7-09-02 | P | cùng Idempotency-Key → cùng kết quả, không thông báo thêm; công bố lại bài PUBLISHED → 409 ALREADY_PUBLISHED (logs/q9p09.log) |
| TC-P7-09-08 | P | B nhận đúng 1 chuông GRADE_PUBLISHED + 1 thư tới email đã xác minh; A không nhận gì về bài của B; thư B: điểm 4/10, 0 điểm nháp/cờ/canary/khoá nội bộ (logs/q9p09.log) |
| TC-P7-09-13 | P | quét 6 trạng thái (không có, QUEUED, RUNNING, DRAFT, REVIEWED, FAILED) × 12 bề mặt STUDENT + /jobs/{id} 404 + SSE + chat: 0 dấu vết nháp/cờ/tiến độ (logs/q9sweep.log; bước cuối của script dùng BT03 lớp 761987 không có ở lớp QD9S nên bỏ, phần B đã công bố ở 09-11/q9p09) |
| TC-P7-10-05 | P | UPHELD giữ 8,00; ADJUSTED không đổi → 422 NO_CHANGE; ADJUSTED hợp lệ cập nhật tại chỗ 4,00 → 4,50 (version+1, giữ published_at, ai_scores, score_after 4.50); phản hồi 9 ký tự 422; quyết lần hai 409; TA 403 (logs/q9p10.log) |
| TC-P7-10-12 | P | appeal: chỉ STUDENT (TEACHER 403); GET appeals TA/TEACHER 200, STUDENT/ADMIN 403; decide TA/STUDENT/ADMIN 403; A không thấy phúc khảo của B (logs/q9p10.log) |
| TC-P7-11-04 | P | go test cmd/mock-graph/server: TestMockGraphMatchesDocs, TestMockTokenExpires60s, TestMockDataFromPlan PASS (khoá khớp tài liệu Microsoft + 30 bản ghi từ teams-plan.csv) (logs/11-01.log) |

> HANDOFF 2 (2026-11-19, build 186bfe7b): N=73. Còn: SEED-* (15), 02 UI/API (01,02,09,14,17,18), 03 (8 ca UI/MinIO), 04-04/17, 05-03/06/07/11, 07-08 (k6), 08-01/03/05/09/11/12/13, 09-01/03..07/09..12/14..18, 10-01..04/06..11/13, 11-02/08/10. Script xong: q9_06*, q9_07a/b/c/d, q9_08/08c, q9p09/10/11, q9sweep (evidence-9s/scripts). Dữ liệu QD9G/QD9U/QD9P/QD9P1/QD9S/QD9T/QD9M*. Spec Playwright: specs/qc-real-9s-failed.spec.ts, qc-real-9s-review.spec.ts (exploration 08-01/04). Đã kiểm tra: khoá global nhả, route GRADING = QC 9router, worker mặc định.
| TC-P7-09-10 | P | B nháp 7,00, sửa c2 1,00→2,00, duyệt, công bố: deviation=1.00, teacher_changed_criteria={c2}; stats TA/TEACHER có đủ 7 khoá (published 3, mean_abs_deviation 0.33, teacher_changed_rate 0.3333…), STUDENT 403, mẫu số 0 → null (không NaN/0) (logs/q9_09x.log) |
| TC-P7-09-14 | P | sv04 (DRAFT) và sv05 (REVIEWED): GET grading-results/{id của mình} và submissions/{id} → 403; ?include=draft, ?debug=1, header X-Include-Draft bị bỏ qua (mine không lộ ai_scores/cờ/độ tin cậy); nộp lại ở cả hai trạng thái trả cùng 409 SUBMISSION_CLOSED «Đã hết hạn nộp bài.» (thân giống hệt, không nhắc duyệt/nháp) (logs/q9_09x.log) |
| TC-P7-09-16 | P | 10 vòng × 3 yêu cầu công bố song song cùng bài: mỗi vòng đúng 1 lần 200, 10 PUBLISHED, 10 sự kiện outbox assignment.result_published (không trùng) (logs/q9_09x.log). Phần «sửa dở → VERSION_CONFLICT» và replay outbox dựa test CI TestPublishVersionConflict / TestPublishOutboxReplayNoDuplicate |
| TC-P7-09-17 | P | publish: TA/STUDENT/ADMIN/GV lớp khác → 403; stats TEACHER/TA 200, STUDENT/ADMIN/GV khác 403; A đọc kết quả của B → 403, id không tồn tại → 403 cùng thân (trừ trace_id) (logs/q9_09x.log) |
| TC-P7-09-18 | P | TEACHER có PUBLISH_PENDING «3 bài đã duyệt chờ công bố · Bài tập 03…» href /grading, TA không; PUBLISH_PENDING biến mất 0,0 s sau khi công bố hết; B có ASSIGNMENT_GRADED «Điểm Bài tập 03… đã có — Xem» href /assignments/{id}, sv04 (chưa công bố) không; mục GRADED hết 7 ngày biến mất ≤0,2 s sau khi cache nguội; bậc 33 ≠ 32 bởi Go test TestPublishPendingNotSameTierAsExamHold/Tier33 PASS (logs/q9_09x.log, q9_09y.log, 10-08-today-go.log) |
| TC-P7-10-02 | P | lý do 9 và 1.001 ký tự → 422 REASON_LENGTH; 10 và 1.000 ký tự → 201; không tiêu chí → 422; tiêu chí lạ → 422 CRITERION_UNKNOWN (logs/q9_09x.log) |
| TC-P7-10-03 | P | bài chưa công bố (đã nộp) → 404 ASSIGNMENT_NOT_FOUND; appeal_days=0 → 409 APPEAL_DISABLED; lần hai (đã trả lời) → 409 APPEAL_ALREADY_FILED; −7 ngày −1 phút → 409 APPEAL_WINDOW_CLOSED dù gửi header Date lùi (giờ máy chủ); đúng hạn −7 ngày +1 phút → 201 (q9p10); sửa appeal_days 7→30 sau công bố: PUT 200 và sv04 (published −7 ngày −1 phút) gửi được 201 (logs/q9_09x.log, q9p10.log) |
| TC-P7-10-08 | P | TEACHER Hôm nay có đúng 1 mục APPEAL «2 yêu cầu xem lại điểm · Bài tập 03…» href /grading/{submissionId}, TA không; overdue >48 giờ (q9p10); chuông TEACHER gộp 1 dòng APPEAL_NEW «2 yêu cầu xem lại mới»; chuông + thư B (q9p10); mục ASSIGNMENT_APPEAL_REPLY bậc 48 chỉ-sinh-viên: Go TestProviderAppealReplyStudentOnly PASS (trên stack sống B luôn thấy mục bậc cao hơn nên không quan sát được) (logs/q9_09x.log, 10-08-today-go.log) |
| TC-P7-10-10 | P | bài có 3 công bố, 2 phúc khảo, 1 ADJUSTED: appealed 2, appeal_rate 0.6667, adjusted 1, appeal_success_rate 0.5000 (chuỗi decimal); mẫu số 0 → null (09-10) (logs/q9_09x.log) |
| TC-P7-02-14 | P | nộp +8 giờ so với hạn: late_days 1, trừ 0,50; PUT due_at +2 ngày → DB late_days 0, trừ 0,00 sau ≈0,5 s (tính lại bất đồng bộ); đưa hạn về cũ → 1 / 0,50 lại (logs/q9_02x.log) |
| TC-P7-03-06 | P | PUT MinIO với x-amz-checksum-sha256 sai → 403; thân ngắn hơn size đã ký (Content-Length lệch) → 499 (từ chối); URL quá 10 phút (620 s) → 403; mỗi lần submit sau đó → 422 UPLOAD_INCOMPLETE, 0 bài nộp mới (logs/q9_03x.log) |
| TC-P7-03-09 | P | thân submit kèm student_id/user_id của A: máy chủ từ chối 422 «Trường không được hỗ trợ» (chặt hơn «bỏ qua» của TC); thân chuẩn → 201, bài thuộc B, A có 0 bài (logs/q9_03y.log) |
| TC-P7-05-03 | P | Go integration TestImapPollNewOnly, TestImapPollIdempotentRestart, TestImapUidValidityChange PASS (restart không nhập đôi, UIDVALIDITY đổi không nhập lại); chạy trên mock-imap trong testcontainers vì mock-imap ep-qc dùng chung với lớp seed 761987 (cấm ghi) (logs/05-11-go.log) |
| TC-P7-05-06 | P | Go TestIngestAlreadyReviewedAllSources PASS: bản mới nhất REVIEWED, thư/Teams/upload mới → UNMATCHED ALREADY_REVIEWED (logs/05-11-go.log) |
| TC-P7-05-07 | P | Go TestImapToDraftGrade PASS: thư IMAP đúng mẫu + s2-kha → MATCHED source IMAP → chấm nháp S2 (5,75, EVIDENCE_NOT_FOUND); sinh viên không thấy nháp: q9sweep 6 trạng thái sạch (logs/05-11-go.log, q9sweep.log) |
| TC-P7-11-08 | P | happy: SUCCEEDED 25 bài (24 MATCHED + 1 UNMATCHED), tiến độ 0→100; throttle: 2 lần 429 rồi vẫn đủ 25 sau 2,3 s (logs/q9p11b.log) |
| TC-P7-11-10 | P | tắt mock-graph giữa chừng → job FAILED UNREACHABLE «Các bài đã lấy vẫn giữ nguyên», khoá ep:lock:teams* rỗng; bật lại → chạy lại SUCCEEDED đủ 25, không đôi; JSON hỏng/treo: Go TestTeamsInvalidJSON (TEAMS_RESPONSE_INVALID) + TestTeamsSyncTimeout (TIMEOUT) PASS (logs/q9p11b.log, 05-11-go.log) |
| TC-P7-11-02 | P | live: paged (30 bản ghi / 3 trang, 32 yêu cầu tới mock-graph) → 25 bài (24 MATCHED + 1 UNMATCHED); đồng bộ lần hai ngay: created 0, deduped 25 (UNIQUE external_ts); nộp lại → phiên bản mới + đã REVIEWED → ALREADY_REVIEWED: Go TestTeamsResubmitNewVersion, TestIngestAlreadyReviewedAllSources PASS (lượt đầu job kẹt QUEUED vì worker bận — chạy lại đạt) (logs/q9p11c.log, 05-11-go.log) |
| TC-P7-08-01 | P | TEACHER /grading (1440): cột Sinh viên · Bài tập · Nguồn · Điểm · Trạng thái duyệt · Thời gian nộp; lý do cờ bằng chữ ngay dưới tên sinh viên («Hai lượt chấm lệch hơn 1 điểm», «Trích dẫn không có trong bài», «Tổng điểm chưa khớp»…; không có cột riêng «Lý do cờ»); bộ lọc mặc định chip «Cần xem kỹ», 17 hàng đầu đều có cờ; nhãn nguồn «Trong EduPilot» (SOURCE_LABEL: UPLOAD = «Tải lên», không phải «ZIP» như TC); không từ trace/model/flags/run_state (shots/p708-queue-1440.png, logs/ui1-queue-review.json) |
| TC-P7-08-10 | P | bấm Duyệt bài / phím A (không đang gõ): sv04 → REVIEWED, reviewed_by có, audit_log grading_result.edit + .review; thanh «Đã duyệt 0/22» → «Đã duyệt 1/22», chuyển sang bài kế (URL đổi), không toast «thành công»; 422 SCORES_REQUIRED, duyệt bài còn cờ, duyệt lại idempotent: q9_08.py (đã ghi ở 08-02/08-06) (logs/ui2-review-actions.json) |
| TC-P7-08-12 | P | TA: /grading và /grading/{id} mở được, sửa + duyệt được; nút «Công bố» ở thanh đầu hàng chờ không có với TA (TEACHER có, bị khoá khi chưa chọn bài); trong DOM của TA chỉ còn nút «Công bố» nằm trong hộp thoại xác nhận <dialog> đóng (0×0, không hiển thị, không có nút mở); API publish TA 403 (09-17) (logs/ui6-publish-btn.json, ui7-ta-publish-dialog.json) |
| TC-P7-08-13 | F | grep mock/assess frontend/src = 0; ui-antipatterns.sh rc=0; axe AA 0 lỗi; focus ring thấy (box-shadow 2 px); lỗi mạng «Không kết nối được tới máy chủ…» + Thử lại; không toast; NHƯNG trạng thái rỗng lớp không có bài nộp hiện «Không có bài chưa duyệt», không phải «Chưa có bài nộp» + «Đồng bộ từ Teams» / «Tải bài lên» như TC → DEF-R9S-07 (logs/ui3-empty-error.json, 08-13-antipatterns.log) |
| TC-P7-02-01 | P | TEACHER và TA /grading: Tabs «Hàng chờ» · «Bài tập» · «Chưa khớp»; tab Bài tập: bảng Tiêu đề · Hạn · Trạng thái · Đã nộp / Tổng · Đã chấm / Đã công bố (2 bài: 22/40 · 22/0 và 2/40 · 2/2); «Tạo bài tập» chỉ TEACHER (1), TA 0 (shots/p702-tabs-*.png, logs/ui5-sv30-tabs.json) |
| TC-P7-02-02 | F | lớp còn việc chờ → mặc định «Hàng chờ» đạt; lớp mới chưa có bài tập và chưa có bài nộp (QD9W) mặc định vẫn «Hàng chờ», TC yêu cầu «Bài tập» → DEF-R9S-08 (logs/ui3-empty-error.json, ui6) |
| TC-P7-03-01 | P | sv30 (chưa nộp) 1440 và 375: thứ tự tên + hạn «đã quá hạn 2 ngày» → đề → «Tiêu chí chấm» (tên + «2,5 điểm», không guidance/đáp án) → vùng nộp («Chọn tệp», «Kéo thả…») → «Nộp bài» → «Chưa nộp»; sidebar không có mục cho trang; không từ kỹ thuật; không tràn ngang (shots/p703-sv30-*.png, logs/ui5-sv30-tabs.json) |
| TC-P7-03-03 | P | UI: .exe → «Chỉ nhận PDF, DOCX, TXT, MD»; tệp rỗng → «Tệp rỗng»; 20 MB + 1 → «Tệp lớn hơn 20 MB»; .PDF (hoa) được nhận; 0 yêu cầu presign trước khi chọn tệp hợp lệ, 1 sau khi chọn .md và «Nộp bài»; input chỉ nhận một tệp (không multiple) (logs/ui4-C-submit.json) |
| TC-P7-03-19 | P | B: GET assignments chỉ bài đã công bố (bài nháp không hiện, chi tiết 404), my:{state:NOT_SUBMITTED,…}, không rubric/guidance/reference_answer/ai_scores; lịch /calendar có mục assignment DEADLINE; Hôm nay có ASSIGNMENT_DUE «Còn 12 giờ — Nộp bài», mất sau 0,5 s khi nộp, C (chưa nộp) vẫn còn; nhắc hạn chỉ cho người chưa nộp, một lần: Go TestReminderOnlyNotSubmitted/Dedupe/OncePerEvent PASS (logs/q9_03z.log, 03-19-go.log) |
| TC-P7-09-11 | P | B mở bài đã công bố 1440 + 375: «7,50 / 10,00», từng tiêu chí «2,00 / 2,50» + nhận xét + đoạn trích của chính bài mình, «Trừ nộp muộn: −0,50 điểm», «Nhận xét giảng viên…», nhãn «Nhận xét · đã được giảng viên duyệt», nút «Yêu cầu xem lại»; không ai_scores/độ lệch/cờ/confidence/lượt 2/guidance/đáp án/tên người duyệt/tên SV khác; 375 không tràn ngang, mục chạm ≥44 px, axe AA 0 lỗi (shots/p709-B-published-*.png, logs/ui4-B-published.json) |
| TC-P7-09-03 | P | Go integration (DB thật trong testcontainers): TestAssignmentItemReadsPublishedLatest/Direct (đọc thẳng grading_results PUBLISHED bản mới nhất, không ghi grade_entries), EqualsStudentView (xem trước = /me/grades), NotEditable, NotPublishable, LinkValidation PASS (logs/10-06-07-09-go.log) |
| TC-P7-09-04 | P | Go TestAssignmentItemBatchNoNPlusOne PASS: số truy vấn không đổi theo số sinh viên, dùng bản mới nhất (logs/10-06-07-09-go.log) |
| TC-P7-09-05 | P | Go TestAssignmentItemLateLinkAfterPublish PASS: gắn bài sau công bố, điểm hiện ngay trong xem trước (logs/10-06-07-09-go.log) |
| TC-P7-09-07 | P | Go PASS: TestPublishWhenLockedKeepsSnapshot (công bố khi sổ LOCKED thành công, snapshot không đổi), TestPublishShowsDriftWhenLocked («Khác bản chốt»), TestPublishLockedRejected, TestPublishSerializesWithFinalize (FOR SHARE gradebooks), TestAdjustedAfterUnlockThenRefinalizeNotifies (mở khoá + chốt lại → GRADE_CHANGED) (logs/10-06-07-09-go.log) |
| TC-P7-10-06 | P | Go PASS: TestAppealDecideWhenGradebookLocked (quyết ADJUSTED khi LOCKED không bị chặn), TestAdjustedShowsDriftWhenLocked (drift, snapshot không đổi), TestAdjustedAfterUnlockThenRefinalizeNotifies (GRADE_CHANGED sau chốt lại), TestPublishSerializesWithFinalize (logs/10-06-07-09-go.log) |
| TC-P7-10-07 | P | Go PASS: TestAppealPathNeverCallsLLM (0 lời gọi llm.* qua gửi/xem/quyết), TestAppealNotInAgentTools (không tool agent nào đọc grade_appeals), TestAppealReasonNotIndexed (lý do không vào chunks/RAG) (logs/10-06-07-09-go.log) |
| TC-P7-10-09 | P | Go PASS: TestAppealWritesNoEscalationRows, TestAppealCreatesNoTicket, TestAppealNotInInboxOrMeTickets, TestAppealNotCountedInTicketCap (gửi được khi đủ 5 ticket), TestAdminStalledIgnoresAppeal, TestAppealNotInKnowledge (logs/10-06-07-09-go.log) |
| TC-P7-02-09 | P | Go TestSuggestRubricLargestRemainder PASS: weight_pct [33,33,34] × max 10 → tổng đúng 10, mọi giá trị bội 0,25 (logs/10-06-07-09-go.log) |

> HANDOFF 3 (2026-11-19): N=37 (P=305, F=5: DEF-R9S-04/05/06/07/08, mọi mức Thấp). Còn: SEED-* (15, cần ep-sd sạch), 02-17/18, 03-02/14/18, 04-04/17, 05-11, 07-08, 08-03/05/09/11, 09-01/06/09/12/15, 10-01/04/11/13, 11-01…; chi tiết từng ca thiếu bước nào xem hàng N/ghi chú ở trên. Specs UI: evidence-9s/specs/qc-real-9s-ui*.spec.ts; dữ liệu UI: q9_ui_seed.py (lớp QD9V).
