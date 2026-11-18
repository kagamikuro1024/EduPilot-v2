# Smoke hồi quy sprint 9 — kết quả
Mã đã chạy: be0c0d96 (sprint/11-p10-eval) · Stack ep-r · Người chạy: qc2 (nhóm D, SmkD) · Ngày: 2026-11-15
Tổng TC: 347 · Đạt: 250 · Trượt: 4 · Chưa xong (N): 93   (phiên Smoke9Runner, mã dc67bd93, stack ep-qc, 2026-11-18). Đột biến mã M1…: 43/44 đã chạy (41 P, 2 F), SEED-M2 N. Ca N còn lại: UI/Playwright, API 08–10, SEED, 02–05, 11 (script q9_08.py, q9p09/10/11.py, q9sweep.py đã viết, chưa chạy).

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
